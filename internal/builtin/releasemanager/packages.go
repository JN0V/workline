package releasemanager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
)

// PackageSettings describe a repository released as one, made of packages
// with their own versions (roles/release-manager/README.md).
type PackageSettings struct {
	Glob            string   `json:"glob"`             // folders at the root that are packages
	VersionFile     string   `json:"version-file"`     // JSON file with a "version" key, in each package
	VersionPatterns []string `json:"version-patterns"` // other places the version is written, with {version}
	Counts          []string `json:"counts"`           // paths, relative to a package, whose changes move it
}

// RootSettings describe the version of the whole.
type RootSettings struct {
	VersionFile string `json:"version-file"`
}

// commit is a conventional commit and the files it touched.
type commit struct {
	Commit
	files []string
}

// Package is one package's move in a release.
type Package struct {
	Name, From, To string
}

var rank = map[string]int{"": 0, "patch": 1, "minor": 2, "major": 3}

// PackageBumps returns how much each package moves, from the commits that
// touched the paths that count in it.
func PackageBumps(pkgs []string, commits []commit, counts []string) map[string]string {
	out := map[string]string{}
	for _, p := range pkgs {
		var mine []Commit
		for _, c := range commits {
			for _, f := range c.files {
				rel, ok := strings.CutPrefix(f, p+"/")
				if ok && (len(counts) == 0 || pathglob.Any(counts, rel)) {
					mine = append(mine, c.Commit)
					break
				}
			}
		}
		out[p] = Bump(mine)
	}
	return out
}

var versionKey = regexp.MustCompile(`("version"\s*:\s*")([^"]*)(")`)

// readVersion returns the top-level "version" of a JSON file.
func readVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &v); err != nil || v.Version == "" {
		return "", fmt.Errorf("%s has no top-level \"version\"", path)
	}
	return v.Version, nil
}

// withVersion rewrites the first "version" key, the top-level one in these
// files; dependency versions further down are left alone.
func withVersion(content, version string) string {
	done := false
	return versionKey.ReplaceAllStringFunc(content, func(m string) string {
		if done {
			return m
		}
		done = true
		return versionKey.ReplaceAllString(m, "${1}"+version+"${3}")
	})
}

// prePackages is Pre for a repository made of packages.
func prePackages(runDir, repo string, s Settings, last string, commits []commit, date time.Time) int {
	p := s.Packages
	dirs, _ := filepath.Glob(filepath.Join(repo, p.Glob))
	var pkgs []string
	for _, d := range dirs {
		if _, err := os.Stat(filepath.Join(d, p.VersionFile)); err == nil {
			pkgs = append(pkgs, filepath.Base(d))
		}
	}
	sort.Strings(pkgs)
	if len(pkgs) == 0 {
		return fail(fmt.Errorf("no folder matching %q holds %s", p.Glob, p.VersionFile))
	}
	bumps := PackageBumps(pkgs, commits, p.Counts)

	// The whole moves by the most any package moves, unless a person says
	// otherwise — with a reason. The version is never the AI's decision.
	rootBump := ""
	for _, b := range bumps {
		if rank[b] > rank[rootBump] {
			rootBump = b
		}
	}
	if rootBump == "" {
		return 10 // no package moved
	}
	override, reason := input(runDir, "root-bump"), input(runDir, "root-reason")
	if override != "" {
		if _, ok := rank[override]; !ok || override == "" {
			return fail(fmt.Errorf("root-bump must be patch, minor or major, not %q", override))
		}
		if reason == "" {
			return fail(fmt.Errorf("a root-bump chosen by hand needs a root-reason, which goes into the notes"))
		}
		rootBump = override
	}

	var fallback []intent.Intention
	var moved []Package
	for _, name := range pkgs {
		if bumps[name] == "" {
			continue
		}
		file := filepath.Join(name, p.VersionFile)
		from, err := readVersion(filepath.Join(repo, file))
		if err != nil {
			return fail(err)
		}
		to, err := NextSemver(from, bumps[name])
		if err != nil {
			return fail(fmt.Errorf("%s: %w", file, err))
		}
		moved = append(moved, Package{Name: name, From: from, To: to})
		patches, err := versionPatches(repo, name, file, from, to, p.VersionPatterns)
		if err != nil {
			return fail(err)
		}
		fallback = append(fallback, patches...)
	}
	rootFile := s.Root.VersionFile
	if rootFile == "" {
		rootFile = p.VersionFile
	}
	rootFrom, err := readVersion(filepath.Join(repo, rootFile))
	if err != nil {
		return fail(err)
	}
	if want := strings.TrimPrefix(last, s.TagPrefix); last != "" && want != rootFrom {
		return fail(fmt.Errorf("%s says %s but the last tag is %s; fix that first", rootFile, rootFrom, last))
	}
	rootTo, err := NextSemver(rootFrom, rootBump)
	if err != nil {
		return fail(err)
	}
	content, err := os.ReadFile(filepath.Join(repo, rootFile))
	if err != nil {
		return fail(err)
	}
	fallback = append(fallback, intent.Intention{Kind: "patch", Value: map[string]any{"file": rootFile, "content": withVersion(string(content), rootTo)}})

	version := s.TagPrefix + rootTo
	var plain []Commit
	for _, c := range commits {
		plain = append(plain, c.Commit)
	}
	section := Section(heading(s, version, rootTo, date), plain)
	var extra strings.Builder
	if reason != "" {
		fmt.Fprintf(&extra, "\n> This release is a %s, though its packages would make it a %s: %s\n", override, maxBump(bumps), reason)
	}
	extra.WriteString("\n### Packages\n\n")
	for _, m := range moved {
		fmt.Fprintf(&extra, "- %s %s → %s\n", m.Name, m.From, m.To)
	}
	head, rest, _ := strings.Cut(section, "\n")
	section = head + "\n" + extra.String() + rest
	return finish(runDir, repo, s, version, section, fallback)
}

func maxBump(b map[string]string) string {
	out := ""
	for _, x := range b {
		if rank[x] > rank[out] {
			out = x
		}
	}
	return out
}

// versionPatches rewrites a package's version file and every place its
// version patterns appear, under the package folder.
func versionPatches(repo, name, file, from, to string, patterns []string) ([]intent.Intention, error) {
	content, err := os.ReadFile(filepath.Join(repo, file))
	if err != nil {
		return nil, err
	}
	out := []intent.Intention{{Kind: "patch", Value: map[string]any{"file": file, "content": withVersion(string(content), to)}}}
	if len(patterns) == 0 {
		return out, nil
	}
	files, err := git(repo, "ls-files", "--", name)
	if err != nil {
		return nil, err
	}
	for _, f := range strings.Split(files, "\n") {
		if f == "" || f == file {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repo, f))
		if err != nil {
			continue
		}
		text, changed := string(data), false
		for _, pat := range patterns {
			old := strings.ReplaceAll(pat, "{version}", from)
			if strings.Contains(text, old) {
				text, changed = strings.ReplaceAll(text, old, strings.ReplaceAll(pat, "{version}", to)), true
			}
		}
		if changed {
			out = append(out, intent.Intention{Kind: "patch", Value: map[string]any{"file": f, "content": text}})
		}
	}
	return out, nil
}

func input(runDir, name string) string {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
	return strings.TrimSpace(string(data))
}
