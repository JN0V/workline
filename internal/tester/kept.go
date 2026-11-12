package tester

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// keptPrefix opens the reason of a scenario kept as it is today: its test
// would pass on the base, and is refused (#794).
const keptPrefix = "kept as today"

// unguarded is what a person reads of a scenario kept as today that no
// existing test was found to check (#1359): never a guard the tester
// did not name, or that is not there.
const unguarded = "kept as today: no existing test was found that checks it, so nothing guards it; a test of it would pass on the base, and is not written"

// codeNames are the names a reason writes as code, `like_this`.
var codeNames = regexp.MustCompile("`([^`\n]+)`")

// guardedSaid holds each reason of a scenario kept as today to the
// existing test it names (#1359): kept when one name it writes as code is
// a test file of the base, or a name one of them holds, its file's name
// included; else said plainly that nothing guards it, a name not found
// quoted. A person reads a guard only where a test is there.
func guardedSaid(repo, base string, s Settings, said map[string]string) {
	var files []string // the base's test files, read once, when asked
	tests := func() []string {
		if files == nil {
			files = []string{}
			out, err := git(repo, "ls-tree", "-r", "--name-only", base)
			if err == nil {
				d := s.Tests()
				for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
					if f != "" && d.Test(f) {
						files = append(files, f)
					}
				}
			}
		}
		return files
	}
	for sc, reason := range said {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(reason)), keptPrefix) {
			continue
		}
		var names []string
		for _, m := range codeNames.FindAllStringSubmatch(reason, -1) {
			names = append(names, strings.TrimSpace(m[1]))
		}
		found := false
		for _, n := range names {
			if guardIn(repo, base, n, tests()) {
				found = true
				break
			}
		}
		switch {
		case found:
		case len(names) > 0:
			said[sc] = unguarded + " (`" + strings.Join(names, "`, `") + "` named, in no test file of the base)"
		default:
			said[sc] = unguarded
		}
	}
}

// guardIn says whether name is a test of the base: one of its test files,
// a file's name without its extension, or a whole word one of them holds,
// the name's last part (`test_late` of `tests.test_pricing.test_late`),
// its parameters aside. A part under 4 characters is too common to read.
func guardIn(repo, base, name string, files []string) bool {
	if len(files) == 0 {
		return false
	}
	part := name
	if i := strings.IndexAny(part, "(["); i > 0 {
		part = part[:i]
	}
	part = part[strings.LastIndexAny(part, "./:#")+1:]
	part = strings.TrimSpace(part)
	for _, f := range files {
		if f == name || strings.TrimSuffix(path.Base(f), path.Ext(f)) == part && part != "" {
			return true
		}
	}
	if len(part) < 4 {
		return false
	}
	out, err := git(repo, "grep", "-l", "-F", "-w", "-e", part, base)
	if err != nil {
		return false // none holds it (exit 1), or unread: no guard said
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if f, ok := strings.CutPrefix(l, base+":"); ok && slices.Contains(files, f) {
			return true
		}
	}
	return false
}
