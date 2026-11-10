package tester

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

// BuildErrors is how one ecosystem's compiler says, in the test
// command's output, that a build broke, and when that is only a name the
// code lacks: a test of code not written yet, red for the right reason
// (ADR-0055, the proof).
type BuildErrors struct {
	// Error matches one error line; its groups `file` and `message`. A
	// file absolute, or a file:// URI, under the folder the tests ran in
	// is read from the repository's root (#1177).
	Error string `json:"error"`
	// MissingName matches a message naming what the code lacks; the
	// name itself in its group `name`, else the first quoted word of the
	// message (#1111).
	MissingName []string `json:"missing-name"`
	// NameLine finds, on a line after a missing-name message that gives
	// no name, the name in its group `name`: javac's `symbol:` line
	// (#1183). Read up to the next error line.
	NameLine string `json:"name-line"`
	// MessageLine finds, on a line after the error line, its message in
	// the group `message`, when the error line gives none: pytest's `E`
	// line under the file it could not collect (#1283). Read up to the
	// next error line; the error is quoted with it.
	MessageLine string `json:"message-line"`
	// TestName finds the tests a test file declares, the first of its
	// groups that matched the name: their ids while the file does not
	// build. Empty, a row naming no tests (a build file's errors).
	TestName string `json:"test-name"`
}

type buildRules struct {
	error    *regexp.Regexp
	missing  []*regexp.Regexp
	nameLine *regexp.Regexp
	msgLine  *regexp.Regexp
	testName *regexp.Regexp
}

// compileBuildErrors reads the build-errors setting; a pattern that does
// not compile, or lacks its groups, is said.
func compileBuildErrors(list []BuildErrors) ([]buildRules, error) {
	var out []buildRules
	for i, b := range list {
		var r buildRules
		var err error
		if r.error, err = regexp.Compile(b.Error); err != nil {
			return nil, fmt.Errorf("build-errors[%d].error: %v", i, err)
		}
		if r.error.SubexpIndex("file") < 0 || r.error.SubexpIndex("message") < 0 {
			return nil, fmt.Errorf("build-errors[%d].error: the groups (?P<file>…) and (?P<message>…) are needed", i)
		}
		for j, m := range b.MissingName {
			re, err := regexp.Compile(m)
			if err != nil {
				return nil, fmt.Errorf("build-errors[%d].missing-name[%d]: %v", i, j, err)
			}
			r.missing = append(r.missing, re)
		}
		if b.NameLine != "" {
			if r.nameLine, err = regexp.Compile(b.NameLine); err != nil || r.nameLine.SubexpIndex("name") < 0 {
				if err == nil {
					err = fmt.Errorf("the group (?P<name>…) is needed")
				}
				return nil, fmt.Errorf("build-errors[%d].name-line: %v", i, err)
			}
		}
		if b.MessageLine != "" {
			if r.msgLine, err = regexp.Compile(b.MessageLine); err != nil || r.msgLine.SubexpIndex("message") < 0 {
				if err == nil {
					err = fmt.Errorf("the group (?P<message>…) is needed")
				}
				return nil, fmt.Errorf("build-errors[%d].message-line: %v", i, err)
			}
		}
		if b.TestName == "" {
			out = append(out, r)
			continue
		}
		if r.testName, err = regexp.Compile(b.TestName); err != nil || r.testName.NumSubexp() < 1 {
			if err == nil {
				err = fmt.Errorf("a group holding the name is needed")
			}
			return nil, fmt.Errorf("build-errors[%d].test-name: %v", i, err)
		}
		out = append(out, r)
	}
	return out, nil
}

// buildError is one error line of a broken build.
type buildError struct {
	file, line string
	msg        string // its message, as the row's group `message` read it
	inTest     bool   // in a test file of the change
	missing    bool   // in a test file of the change, naming what the code lacks
	name       string // the name it lacks, when the message gives it
	rules      int    // the ecosystem it was read by
}

// buildErrorsIn reads the error lines of output by each ecosystem's
// pattern. Each is a missing name only when it lies in one of tests (the
// change's test files) and its message names what the code lacks. A
// file under one of roots, the folder the command ran in, is read from
// there, and its line is kept with those roots taken off (#1274).
func buildErrorsIn(output string, tests []string, rules []buildRules, roots ...string) []buildError {
	inTests := map[string]bool{}
	for _, t := range tests {
		inTests[t] = true
	}
	var errs []buildError
	unnamed := -1 // the last error, a missing name whose name a later line gives
	untold := -1  // the last error, whose message a later line gives
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		read := false
		for i, r := range rules {
			m := r.error.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			file := fromRoot(m[r.error.SubexpIndex("file")], roots)
			msg := m[r.error.SubexpIndex("message")]
			e := buildError{file: file, line: unrooted(line, roots), msg: msg, inTest: inTests[file], rules: i}
			errs = append(errs, e)
			unnamed, untold = -1, -1
			if msg == "" && r.msgLine != nil {
				untold = len(errs) - 1
			} else if missingIn(&errs[len(errs)-1], r) {
				unnamed = len(errs) - 1
			}
			read = true
			break
		}
		switch {
		case read:
		case untold >= 0:
			re := rules[errs[untold].rules].msgLine
			if m := re.FindStringSubmatch(line); m != nil {
				e := &errs[untold]
				e.msg = m[re.SubexpIndex("message")]
				e.line = strings.TrimSpace(e.line) + " " + strings.TrimSpace(unrooted(line, roots))
				if missingIn(e, rules[e.rules]) {
					unnamed = untold
				}
				untold = -1
			}
		case unnamed >= 0:
			re := rules[errs[unnamed].rules].nameLine
			if m := re.FindStringSubmatch(line); m != nil {
				errs[unnamed].name = m[re.SubexpIndex("name")]
				unnamed = -1
			}
		}
	}
	return errs
}

// missingIn marks e a missing name when it lies in a test of the change
// and its message names what the code lacks, by r's patterns; true when
// a later line is to give the name it lacks (r's name-line).
func missingIn(e *buildError, r buildRules) bool {
	if !e.inTest {
		return false
	}
	for _, re := range r.missing {
		if m := re.FindStringSubmatch(e.msg); m != nil {
			e.missing, e.name = true, nameIn(re, m, e.msg)
			break
		}
	}
	return e.missing && e.name == "" && r.nameLine != nil
}

// fromRoot is a file an error line names, from the repository's root: a
// file:// URI as its path, one under a root as the path below it. Any
// other is kept, cleaned.
func fromRoot(file string, roots []string) string {
	return fromRootOn(file, roots, runtime.GOOS == "windows")
}

// drive is a Windows path's drive, `C:/`, after a URI's `/` or not.
var drive = regexp.MustCompile(`^/?([A-Za-z]:/)`)

// fromRootOn is fromRoot by Windows' rules or not (#1183): there, a
// path's `\` is `/`, it starts at its drive (file:///C:/… is C:/…), and
// case does not count.
func fromRootOn(file string, roots []string, windows bool) string {
	if rest, ok := strings.CutPrefix(file, "file://"); ok {
		if p, err := url.PathUnescape(rest); err == nil {
			rest = p
		}
		file = rest
	}
	slash := func(p string) string {
		if windows {
			p = drive.ReplaceAllString(strings.ReplaceAll(p, `\`, "/"), "$1")
		}
		return p
	}
	file = slash(file)
	if path.IsAbs(file) || (windows && drive.MatchString(file)) {
		for _, r := range roots {
			r = strings.TrimSuffix(slash(filepath.ToSlash(r)), "/")
			if r == "" || len(file) <= len(r) || file[len(r)] != '/' {
				continue
			}
			if head := file[:len(r)]; head == r || (windows && strings.EqualFold(head, r)) {
				return path.Clean(file[len(r)+1:])
			}
		}
	}
	return path.Clean(strings.TrimPrefix(file, "./"))
}

// rootsOf is the roots of a command run in dir: dir, and dir with its
// links resolved, as a compiler may print it (macOS's /var is
// /private/var).
func rootsOf(dir string) []string {
	roots := []string{dir}
	if r, err := filepath.EvalSymlinks(dir); err == nil && r != dir {
		roots = append(roots, r)
	}
	return roots
}

// unrooted is output with the folder the command ran in taken off its
// paths, as a path or a file:// URI (Gradle's): the same build broken the
// same way reads the same from one checkout to the next, as its judge is
// asked it (#1274).
func unrooted(output string, roots []string) string {
	var prefixes []string
	for _, r := range roots {
		if r != "" {
			r += string(filepath.Separator)
			prefixes = append(prefixes, "file://"+filepath.ToSlash(r), r)
		}
	}
	// The longest first: a URI holds its path, /private/var/x holds /var/x
	// (macOS).
	slices.SortStableFunc(prefixes, func(a, b string) int { return len(b) - len(a) })
	for _, p := range prefixes {
		output = strings.ReplaceAll(output, p, "")
	}
	return output
}

// testNames is the tests content declares, by the rules' test-name: the
// first group of each match that matched, as a Kotlin name in
// backquotes or not.
func testNames(content string, r buildRules) []string {
	if r.testName == nil {
		return nil
	}
	var names []string
	for _, m := range r.testName.FindAllStringSubmatch(content, -1) {
		for _, g := range m[1:] {
			if g != "" {
				names = append(names, g)
				break
			}
		}
	}
	return names
}

// quotedName is a word between quotes, as compilers quote the name they
// lack: 'x', ‘x’, "x", `x`.
var quotedName = regexp.MustCompile("['‘\"`]([^\\s'‘’\"`]+)['’\"`]")

// nameIn is the name a missing-name message gives: its group `name`,
// else its first quoted word; "" when it gives none.
func nameIn(re *regexp.Regexp, m []string, msg string) string {
	if i := re.SubexpIndex("name"); i >= 0 && m[i] != "" {
		return m[i]
	}
	if q := quotedName.FindStringSubmatch(msg); q != nil {
		return q[1]
	}
	return ""
}

// lastPart is a name's last word, past its qualifiers: `lostCount` of
// `MQTTStatistics::lostCount`, `Mul` of `calc.Mul`.
var lastPart = regexp.MustCompile(`[\p{L}\p{N}_]+$`)

// namedIn says whether text names name as a whole word, or its last
// part: a missing name the issue names is what the change adds; one it
// never names, a guess of the test's (#1111).
func namedIn(name, text string) bool {
	last := lastPart.FindString(name)
	if last == "" {
		return false
	}
	return regexp.MustCompile(`(?:^|[^\p{L}\p{N}_])` + regexp.QuoteMeta(last) + `(?:$|[^\p{L}\p{N}_])`).MatchString(text)
}
