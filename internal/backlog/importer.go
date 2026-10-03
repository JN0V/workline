package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// Entry is one entry of a roadmap file: a heading carrying an id, and the
// text under it down to the next heading of its level or above.
type Entry struct {
	ID, Title string
	Line      int // the heading's line, from 1
	Body      string
	Done      bool // its heading says it is done
	Sources   []string
}

var entryHead = regexp.MustCompile(`^(#{2,4}) ([A-Z][A-Z0-9]*(?:-[A-Z][A-Z0-9]*)*-\d+)\s*[—:-]+\s*(.+)$`)

// DoneWords are the words a heading says an entry is finished with.
var DoneWords = regexp.MustCompile(`(?i)\b(done|closed|withdrawn|refuted|superseded|merged|fixed|wontfix)\b`)

// ParseRoadmap reads the entries of a roadmap: each heading of level 2 to
// 4 starting with an id (BUG-4, SEC-12, …). done tells an entry finished
// from its heading.
func ParseRoadmap(text string, done *regexp.Regexp) []Entry {
	lines := strings.Split(text, "\n")
	var out []Entry
	var cur *Entry
	level := 0
	var body []string
	flush := func() {
		if cur != nil {
			cur.Body = strings.TrimSpace(strings.Join(body, "\n"))
			out = append(out, *cur)
			cur, body = nil, nil
		}
	}
	for i, l := range lines {
		if strings.HasPrefix(l, "#") {
			n := len(l) - len(strings.TrimLeft(l, "#"))
			if cur != nil && n <= level {
				flush()
			}
			if m := entryHead.FindStringSubmatch(l); m != nil {
				flush()
				level = len(m[1])
				title := regexp.MustCompile(`\s+—\s+\*\*.*$`).ReplaceAllString(m[3], "")
				cur = &Entry{ID: m[2], Title: strings.TrimSpace(title), Line: i + 1, Done: done.MatchString(m[3])}
				continue
			}
		}
		if cur != nil {
			body = append(body, l)
		}
	}
	flush()
	return out
}

var (
	pathLike  = regexp.MustCompile(`[\w.-]+(?:/[\w.-]+)*\.[A-Za-z]{1,5}\b`)
	backticks = regexp.MustCompile("`([A-Za-z_][\\w:]*\\w)(?:\\(\\))?`")
	docFile   = regexp.MustCompile(`\.(md|txt|rst|adoc)$`)
	testFile  = regexp.MustCompile(`(^|/)(tests?|spec)/|_test\.|\btest_`)
)

// ResolveSources names the code an entry is about, at most max files: the
// files it names, by path or by a name only one file has, then the code
// files its backticked names are in, the most named first, tests last. What is not
// found is left out: a person, or the product owner, names it later.
func ResolveSources(repo string, e Entry, max int) []string {
	out, _ := exec.Command("git", "-C", repo, "ls-files").Output()
	tracked := strings.Split(strings.TrimSpace(string(out)), "\n")
	var code []string
	for _, t := range tracked {
		if !docFile.MatchString(t) {
			code = append(code, t)
		}
	}
	var found []string
	add := func(p string) {
		if len(found) < max && !slices.Contains(found, p) {
			found = append(found, p)
		}
	}
	text := e.Title + "\n" + e.Body
	for _, tok := range pathLike.FindAllString(text, -1) {
		tok = strings.Trim(tok, "./")
		var cands []string
		for _, t := range code {
			if t == tok || strings.HasSuffix(t, "/"+tok) {
				cands = append(cands, t)
			}
		}
		if len(cands) == 1 {
			add(cands[0])
		}
	}
	score := map[string]int{}
	seen := map[string]bool{}
	for _, m := range backticks.FindAllStringSubmatch(text, -1) {
		name := m[1][strings.LastIndex(m[1], ":")+1:]
		if len(name) < 5 || seen[name] {
			continue
		}
		seen[name] = true
		hits, err := exec.Command("git", append([]string{"-C", repo, "grep", "-lw", name, "--"}, code...)...).Output()
		files := strings.Fields(string(hits))
		if err != nil || len(files) == 0 || len(files) > 4 {
			continue // a name everywhere says nothing of where the entry is
		}
		for _, f := range files {
			score[f]++
		}
	}
	var ranked []string
	for f := range score {
		ranked = append(ranked, f)
	}
	slices.SortFunc(ranked, func(a, b string) int {
		if ta, tb := testFile.MatchString(a), testFile.MatchString(b); ta != tb {
			if ta {
				return 1 // the code before its tests
			}
			return -1
		}
		if score[a] != score[b] {
			return score[b] - score[a]
		}
		return strings.Compare(a, b)
	})
	for _, f := range ranked {
		add(f)
	}
	return found
}

// IssueTitle is the title an entry's issue takes: its id kept, so the
// entry is found again by it.
func (e Entry) IssueTitle() string { return e.ID + " — " + e.Title }

// IssueBody is the entry's text, and where it came from.
func (e Entry) IssueBody() string {
	return fmt.Sprintf("%s\n\nImported from line %d of the project's roadmap, where it is `%s`.", e.Body, e.Line, e.ID)
}

// Import opens an issue for each entry not done whose id no open issue
// carries, with its state: its sources, confirmed at commit. It returns the
// issues it opened, and the ids it found already open.
func Import(f forge.Forge, role, commit string, entries []Entry) (opened []int, already []string, err error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, nil, fmt.Errorf("this forge cannot list issues")
	}
	open, err := b.Issues()
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if e.Done {
			continue
		}
		if slices.ContainsFunc(open, func(is forge.Issue) bool { return strings.HasPrefix(is.Title, e.ID+" — ") }) {
			already = append(already, e.ID)
			continue
		}
		id, err := f.OpenIssue(e.IssueTitle(), e.IssueBody(), forge.Marker("import="+e.ID))
		if err != nil {
			return opened, already, err
		}
		st := State{Sources: e.Sources, Confirmed: commit}
		if err := f.Sticky(forge.Target{Kind: "issue", ID: id}, FormatState(st), StateMarker(role), true); err != nil {
			return opened, already, err
		}
		opened = append(opened, id)
	}
	return opened, already, nil
}
