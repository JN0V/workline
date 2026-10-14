package release

import (
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// The versions a release tool bumps on its own pull request, by its own
// marks: release-please's generic updater and its manifest, and
// releaser-pleaser's marker (#384).

var (
	semver = regexp.MustCompile(`v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?`) // a version, as the tools write it
	number = regexp.MustCompile(`\d+`)                                  // one part of it: a major, a minor, a patch
)

// Markers are what the known tools' generic updaters look for on a line
// whose version they bump, and what they bump there: a version, or one
// part of it.
var Markers = []struct {
	Text  string
	Bumps *regexp.Regexp
}{
	{"x-release-please-version", semver}, {"x-releaser-pleaser-version", semver},
	{"x-release-please-major", number}, {"x-release-please-minor", number}, {"x-release-please-patch", number},
}

// BlockStart and BlockEnd are release-please's markers around lines it
// bumps, each line in between: `x-release-please-start-version` (or
// -major, -minor, -patch), then `x-release-please-end`.
const BlockStart, BlockEnd = "x-release-please-start-", "x-release-please-end"

// Manifests are the files a tool keeps its versions in, every line a
// version it bumps: release-please's.
var Manifests = []string{".release-please-manifest.json"}

var hunk = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// marked is what a tool bumps on a line, by the marker it holds; nil for a
// line no marker names.
func marked(line string) *regexp.Regexp {
	for _, m := range Markers {
		if strings.Contains(line, m.Text) {
			return m.Bumps
		}
	}
	return nil
}

// blocks are, for each line of a file between release-please's markers,
// what is bumped on it.
func blocks(text string) map[int]*regexp.Regexp {
	in := map[int]*regexp.Regexp{}
	var bumps *regexp.Regexp
	for i, l := range strings.Split(text, "\n") {
		switch {
		case strings.Contains(l, BlockStart):
			part, _, _ := strings.Cut(l[strings.Index(l, BlockStart)+len(BlockStart):], " ")
			bumps = marked("x-release-please-" + strings.TrimRight(part, "-*/#>;"))
		case strings.Contains(l, BlockEnd):
			bumps = nil
		case bumps != nil:
			in[i+1] = bumps
		}
	}
	return in
}

// OnlyBumps says whether a file's change, its `git diff -U0`, only bumps
// versions as a release tool does: each hunk puts back as many lines as it
// takes, each the same line with one version (or one part of one, for a
// major, minor or patch marker) in place of another, on a line the tool
// marks — its marker on the line, the line between a block's markers in
// the file now, or a line of its manifest. now reads the file at the
// diff's end, only when a block is to be looked for. A file added,
// removed, moved or changed in its mode is no bump.
func OnlyBumps(path, diff string, now func() (string, error)) bool {
	manifest := slices.Contains(Manifests, filepath.Base(path))
	var inBlock map[int]*regexp.Regexp
	bumpsAt := func(line string, at int) *regexp.Regexp {
		if r := marked(line); r != nil {
			return r
		}
		if manifest {
			return semver
		}
		if inBlock == nil {
			text, err := now()
			if err != nil {
				return nil
			}
			inBlock = blocks(text)
		}
		return inBlock[at]
	}
	var removed, added []string
	at, hunks, header := 0, 0, true
	settled := func() bool {
		if len(removed) == 0 || len(removed) != len(added) {
			return false
		}
		for i := range added {
			bumps := bumpsAt(added[i], at+i)
			if bumps == nil {
				return false
			}
			was, is := bumps.ReplaceAllString(removed[i], "\x00"), bumps.ReplaceAllString(added[i], "\x00")
			if was != is || !strings.Contains(is, "\x00") {
				return false
			}
		}
		return true
	}
	for _, l := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "@@"):
			m := hunk.FindStringSubmatch(l)
			if m == nil || (!header && !settled()) {
				return false
			}
			at, _ = strconv.Atoi(m[1])
			header, removed, added, hunks = false, nil, nil, hunks+1
		case header && (strings.HasPrefix(l, "diff --git ") || strings.HasPrefix(l, "index ") ||
			strings.HasPrefix(l, "--- ") || strings.HasPrefix(l, "+++ ")):
		case header:
			return false // a mode, a move, a file added or removed, a binary
		case strings.HasPrefix(l, "-"):
			removed = append(removed, l[1:])
		case strings.HasPrefix(l, "+"):
			added = append(added, l[1:])
		case strings.HasPrefix(l, `\`): // no newline at the end
		default:
			return false // another file's diff, or a line of context
		}
	}
	return hunks > 0 && settled()
}
