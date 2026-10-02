package documentalist

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// What the weekly sample of the docs vouched for (ADR-0014, step 4;
// internal/sample) reads with the role's own rules: which files are docs,
// how much of their sources may be read whole, and a quote found outside a
// comment of a source, at the commit a doc's `checked` names.

// SampleSettings is the `sample` setting: the judge reading the sample, and
// the least independence from the model that vouched it may stand at.
type SampleSettings struct {
	Judge   string `json:"judge"`    // an --ai value: claude:opus, cmd:…
	AtLeast string `json:"at-least"` // provider, model or context (ADR-0005)
	// After: a commit (a tag) before which nothing is sampled — the
	// release whose engine earns `checked` (ADR-0014, step 0).
	After string `json:"after"`
}

// SettingsFrom reads the role's settings as the engine merged them.
func SettingsFrom(merged map[string]any) (Settings, error) {
	var s Settings
	data, err := json.Marshal(merged)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

// IsDoc says whether a path is one of the docs the settings name.
func (s Settings) IsDoc(p string) bool { return matchAny(s.Docs, p) }

// WholeCap is the cap on the characters a doc's sources may take to be
// judged whole.
func (s Settings) WholeCap() int { return wholeChars(s) }

// Header is a doc's header, as written, and how many lines it takes.
func Header(content string) (meta string, lines int) { return header(content) }

// at reads each repository a doc's sources are in at the commit its
// `checked` names there.
func at(repo string, d *Doc) (*places, error) {
	var cfg projectRepos
	if data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}
	gitDir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, err
	}
	pl := &places{repo: repo, gitDir: gitDir, cfg: cfg, known: map[string]place{}}
	for name, commit := range d.Checked {
		where, err := pl.get(name)
		if err != nil {
			return nil, err
		}
		if _, err := git(where.dir, "cat-file", "-e", commit+"^{commit}"); err != nil {
			return nil, fmt.Errorf("`checked` names %s, a commit this clone does not have", commit)
		}
		pl.known[name] = place{where.dir, commit}
	}
	return pl, nil
}

// SourcesAt is each source of a doc, whole, at the commit its `checked`
// names, as a task shows it. fits is false past limit characters: then the
// doc's `checked` could not have been earned by reading them (ADR-0014). An
// error is a source that cannot be read at all.
func SourcesAt(repo string, d *Doc, limit int) (evidence []string, fits bool, err error) {
	pl, err := at(repo, d)
	if err != nil {
		return nil, false, err
	}
	if evidence, ok := sourcesAs(d, pl, limit, "at the commit `checked` names"); ok {
		return evidence, true, nil
	}
	// Too large, or unreadable: unbounded, it says which.
	if _, ok := sourcesAs(d, pl, math.MaxInt, ""); !ok {
		return nil, false, fmt.Errorf("a source of %s cannot be read at the commit `checked` names", d.Path)
	}
	return nil, false, nil
}

// SourceFileAt is a file under one of a doc's sources at the commit its
// `checked` names (`name:file` for another repository's), and that commit.
func SourceFileAt(repo string, d *Doc, p string) (content, commit string, ok bool) {
	repoName, file := "", strings.TrimPrefix(path.Clean(strings.TrimPrefix(p, "./")), "b/")
	if n, f, cut := strings.Cut(file, ":"); cut && !strings.Contains(n, "/") {
		repoName, file = n, f
	}
	under := false
	for _, src := range d.Sources {
		name, sp, _ := splitSource(src)
		sp = strings.TrimSuffix(path.Clean(sp), "/")
		if name == repoName && (file == sp || sp == "." || strings.HasPrefix(file, sp+"/")) {
			under = true
		}
	}
	pl, err := at(repo, d)
	if !under || err != nil {
		return "", "", false
	}
	where, err := pl.get(repoName)
	if err != nil {
		return "", "", false
	}
	content, err = git(where.dir, "show", where.rev+":"+file)
	return content, where.rev, err == nil
}

// QuoteIn finds a quote in a file, white space aside: found anywhere, found
// with a character of it outside a comment, and the line it starts on. A
// quote found only in a comment is not evidence (ADR-0014, step 2).
func QuoteIn(p, content, quote string) (found, code bool, line int) {
	return quoteIn(p, content, quote)
}

// InCodeAt says whether a name is a whole word of the code (not Markdown)
// at rev.
func InCodeAt(repo, rev, name string) (bool, error) {
	found, err := inCode(repo, rev, map[string]bool{name: true})
	return found[name], err
}
