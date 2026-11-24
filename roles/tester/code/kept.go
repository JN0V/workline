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

// guardedSaid holds each reason of a scenario kept as today, or naming a
// test as code in its own words (#2165), to the existing test it names (#1359): kept when one name it writes as code is
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
		kept := strings.HasPrefix(strings.ToLower(strings.TrimSpace(reason)), keptPrefix)
		var names []string
		for _, m := range codeNames.FindAllStringSubmatch(reason, -1) {
			// in its own words, only a name that reads as a test's counts
			if n := strings.TrimSpace(m[1]); kept || testLike(n, s) {
				names = append(names, n)
			}
		}
		if !kept && len(names) == 0 {
			continue // in its own words, naming no test: nothing to check
		}
		found := ""
		for _, n := range names {
			if guardIn(repo, base, n, tests()) {
				found = n
				break
			}
		}
		switch {
		case found != "":
			if !kept { // a test of the base named: read as kept as today (#2165)
				said[sc] = keptPrefix + ": guarded by `" + found + "`"
			}
		case len(names) > 0:
			said[sc] = unguarded + " (`" + strings.Join(names, "`, `") + "` named, in no test file of the base)"
		default:
			said[sc] = unguarded
		}
	}
}

// testNamed reads a test's name: a segment starting or ending in test(s),
// (`Test` followed by a capital or digit, case-sensitive, so `testing` and
// `testdata` are not) as `TestAdd`, `test_add`, `add_test.go`, `tests.test_add`.
var testNamed = regexp.MustCompile(`(?i:(^|[._:#/\-])tests?([._:#/\-]|$))|(^|[._:#/\-])Tests?[A-Z0-9]|(?i:(^|[._:#/\-])[A-Za-z0-9]*(_|\.)(tests?|spec)(\.|$))`)

// testLike says whether a name written as code names a test, not any
// identifier: a test file of the base's kind, or a Test-style name.
func testLike(name string, s Settings) bool {
	return s.Tests().Test(name) || testNamed.MatchString(name)
}

// guardIn says whether name is a test of the base: one of its test files
// (its path, its name, or its name without the extension), or a whole
// word one of them holds. The word is the name's last part, its
// parameters aside (`test_late` of `tests.test_pricing.test_late`); a
// part that is the test files' extension is passed over (`java` of
// `NoSuchTest.java`); a Go subtest or a suite's case is read by each side
// of its `/` too (`TestAdd` of `TestAdd/sub_case`). A part under 4
// characters is too common to read.
func guardIn(repo, base, name string, files []string) bool {
	if len(files) == 0 {
		return false
	}
	exts := map[string]bool{}
	for _, f := range files {
		if f == name || path.Base(f) == name || strings.HasSuffix(f, "/"+name) {
			return true
		}
		if e := path.Ext(f); e != "" {
			exts[strings.ToLower(e[1:])] = true
		}
	}
	id := name
	if i := strings.IndexAny(id, "(["); i > 0 {
		id = id[:i]
	}
	segs := strings.FieldsFunc(id, func(r rune) bool { return strings.ContainsRune(".:#", r) })
	for len(segs) > 1 && exts[strings.ToLower(strings.TrimSpace(segs[len(segs)-1]))] {
		segs = segs[:len(segs)-1] // a file's extension, not a name
	}
	if len(segs) == 0 || exts[strings.ToLower(strings.TrimSpace(segs[len(segs)-1]))] {
		return false
	}
	part := strings.TrimSpace(segs[len(segs)-1])
	for _, f := range files {
		if strings.TrimSuffix(path.Base(f), path.Ext(f)) == part {
			return true
		}
	}
	parts := []string{part}
	if i := strings.Index(part, "/"); i > 0 {
		parts = append(parts, part[:i], part[strings.LastIndex(part, "/")+1:])
	}
	for _, p := range parts {
		if len(p) >= 4 && grepped(repo, base, p, files) {
			return true
		}
	}
	return false
}

// grepped says whether one of the base's test files holds word whole.
func grepped(repo, base, word string, files []string) bool {
	out, err := git(repo, "grep", "-l", "-F", "-w", "-e", word, base)
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

// unprovenScenarios are the issue's scenarios, in its order, that no test
// proves: said rather than tested, but for one kept as today that an
// existing test of the base guards (guardedSaid kept its reason).
func unprovenScenarios(t task, said map[string]string) []string {
	var out []string
	for _, sc := range t.Scenarios {
		n, ok := said[sc.Name]
		if !ok {
			continue
		}
		if r := strings.ToLower(strings.TrimSpace(n)); strings.HasPrefix(r, keptPrefix) && !strings.HasPrefix(n, unguarded) {
			continue // guarded by a test the base holds
		}
		out = append(out, sc.Name)
	}
	return out
}
