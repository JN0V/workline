package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// A parent and its parts (ADR-0029): a need split into children stays
// open, its own; as its parts close, the engine writes on it, with no
// agent, what each delivered and which items of its Verification a part
// delivered quotes. A person accepts it by closing it: the role never does.

// EvidenceKey is the sticky comment's key on a parent:
// <!-- workline:sticky=<role>/parts -->.
const EvidenceKey = "parts"

// taskRef is a task list's item naming one issue: "- [ ] #13".
var taskRef = regexp.MustCompile(`(?m)^\s*[-*+] \[[ xX]\] #(\d+)\b`)

// Parts are an issue's children as the forge shows them, one rule for the
// report, the order and the comment: its relation (GitHub's sub-issues,
// GitLab's tasks) and the task list under "## Sub-issues" in its body —
// where a split links its children, or lists them. A part a person
// unlinked is no longer one.
func Parts(is forge.Issue) []int {
	return SplitInto(is, nil)
}

// SplitInto are its parts and the children the role's split recorded in
// its state (st may be nil): never closed by the role, nor split again.
func SplitInto(is forge.Issue, st *State) []int {
	var out []int
	add := func(n int) {
		if n != is.ID && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	for _, n := range is.Children {
		add(n)
	}
	for _, m := range taskRef.FindAllStringSubmatch(work.Sections(is.Body)["Sub-issues"], -1) {
		if n, err := strconv.Atoi(m[1]); err == nil {
			add(n)
		}
	}
	if st != nil {
		for _, n := range st.Split {
			add(n)
		}
	}
	slices.Sort(out)
	return out
}

// verificationItem is a Markdown list item: its text after the bullet or number,
// and a task box.
var verificationItem = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?(.+?)\s*$`)

// VerificationItems are the items of a body's Verification section: each
// list item, or, with no list, the section whole.
func VerificationItems(body string) []string {
	text := strings.TrimSpace(work.Sections(body)["Verification"])
	if text == "" {
		return nil
	}
	var items []string
	for _, l := range strings.Split(text, "\n") {
		if m := verificationItem.FindStringSubmatch(l); m != nil {
			items = append(items, m[1])
		}
	}
	if len(items) == 0 {
		return []string{squeeze(text)}
	}
	return items
}

// plain is a text as it is compared: case, spaces, Markdown's emphasis and
// code marks, and the final punctuation aside.
func plain(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '`' || r == '*' || r == '_' {
			return -1
		}
		return r
	}, strings.ToLower(s))
	return strings.TrimRight(squeeze(s), " .;:,!")
}

// Part is one child as the parent's report reads it: the issue, open or
// closed, and what closed it; Gone when the forge no longer has it;
// Unread when the forge refused to say what closed it.
type Part struct {
	ID      int
	Issue   forge.Issue
	Closers []forge.Closer
	Unread  error
	Gone    bool
}

// Delivered says whether a part's work is done: closed, and not as not
// planned nor as a duplicate. GitLab and the local forge keep no reason: a
// closing there is taken as done.
func (p Part) Delivered() bool {
	return !p.Gone && p.Issue.Closed && p.Issue.Reason != "not_planned" && p.Issue.Reason != "duplicate"
}

// Evidence is a parent's report, as the engine writes it.
type Evidence struct {
	Body      string
	Parts     int
	Closed    int      // the parts closed, or gone from the forge
	Undone    []int    // the parts closed without delivering: not planned, a duplicate, gone
	Unproved  []string // the Verification items no part delivered quotes, or whose test is not in the code
	NoTest    []string // the tests a quoted item names that the code does not hold
	Unread    []Part   // the parts whose closer the forge refused to say, with why
	AllClosed bool
}

// ReadEvidence writes a parent's report from its parts: what each became
// and what closed it; each item of its Verification proved by a part
// delivered that quotes it — in its own Verification, or in the text of
// the pull request or commit that closed it — and, when it names a test,
// that test found in the code by tests (nil: not looked for); or said not
// proved.
func ReadEvidence(parent forge.Issue, parts []Part, role string, tests TestFinder) Evidence {
	ev := Evidence{Parts: len(parts)}
	var b strings.Builder
	fmt.Fprintf(&b, "**The parts of this need**, as they close: kept up to date by the %s role, which never closes this issue — accepting the need is a person's.\n\n", strings.ReplaceAll(role, "-", " "))
	b.WriteString("| Part | State | Closed by |\n|---|---|---|\n")
	for _, p := range parts {
		title, state, by := p.Issue.Title, "open", "—"
		switch {
		case p.Gone:
			title, state = "", "gone from the forge (deleted or moved): its part not delivered here"
		case !p.Issue.Closed && slices.Contains(p.Issue.Labels, LabelReady):
			state = "open, ready"
		case !p.Issue.Closed:
		case p.Issue.Reason == "not_planned":
			state = "closed as not planned: its part of the need not delivered"
		case p.Issue.Reason == "duplicate":
			state = "closed as a duplicate: its part not delivered here"
		case p.Issue.Reason == "completed":
			state = "closed as completed"
		default:
			state = "closed"
		}
		if p.Issue.Closed || p.Gone {
			ev.Closed++
			if !p.Delivered() {
				ev.Undone = append(ev.Undone, p.ID)
			}
		}
		if p.Delivered() {
			by = "by hand: no pull request nor commit linked"
			if p.Unread != nil {
				by = "not read: the forge did not say"
				ev.Unread = append(ev.Unread, p)
			}
			if len(p.Closers) > 0 {
				var refs []string
				for _, c := range p.Closers {
					refs = append(refs, closerName(c))
				}
				by = strings.Join(refs, ", ")
			}
		}
		fmt.Fprintf(&b, "| #%d %s | %s | %s |\n", p.ID, cell(title), state, by)
	}
	ev.AllClosed = ev.Parts > 0 && ev.Closed == ev.Parts
	notYet := map[bool]string{true: "**Not proved**", false: "**Not proved yet**"}[ev.AllClosed]
	items := VerificationItems(parent.Body)
	if len(items) == 0 {
		b.WriteString("\nThis issue has no Verification of its own: nothing to prove its parts against; a person judges from the parts alone.\n")
	} else {
		b.WriteString("\n**Its Verification**, each item against the parts delivered — proved when a part's Verification, or what closed it, quotes it, and a test it names is in the code:\n\n")
		for _, item := range items {
			where := proof(item, parts)
			if where != "" && tests != nil {
				var in, missing []string
				for _, name := range TestNames(item) {
					if path := tests(name); path == name || strings.HasPrefix(name, path+"::") {
						in = append(in, "`"+name+"`") // a file, or a test in it, named by its path
					} else if path != "" {
						in = append(in, fmt.Sprintf("`%s` in %s", name, path))
					} else {
						missing = append(missing, "`"+name+"`")
					}
				}
				if len(missing) > 0 {
					ev.Unproved = append(ev.Unproved, item)
					ev.NoTest = append(ev.NoTest, missing...)
					fmt.Fprintf(&b, "- %s: \"%s\" — %s, but the code holds no test %s: a test named is no proof until it is there.\n", notYet, item, where, strings.Join(missing, ", "))
					continue
				}
				if len(in) > 0 {
					where += "; the test " + strings.Join(in, ", ") + ", read from the code"
				}
			}
			if where != "" {
				fmt.Fprintf(&b, "- Proved: \"%s\" — %s.\n", item, where)
				continue
			}
			ev.Unproved = append(ev.Unproved, item)
			fmt.Fprintf(&b, "- %s: \"%s\" — no part delivered quotes it.\n", notYet, item)
		}
	}
	if !ev.AllClosed {
		fmt.Fprintf(&b, "\n**%d of %d parts closed.** Once all are, a person accepts this need by closing this issue.\n", ev.Closed, ev.Parts)
	} else {
		fmt.Fprintf(&b, "\n**All %d parts are closed: for a person to accept.** Close this issue as completed to accept the need; or keep it open — reopen a part, or open one for what is missing.", ev.Parts)
		var missing []string
		if len(ev.Undone) > 0 {
			missing = append(missing, "the part of "+issueList(ev.Undone)+" not delivered")
		}
		if n := len(ev.Unproved); n > 0 {
			missing = append(missing, fmt.Sprintf("%d item(s) of its Verification not proved", n))
		}
		if len(missing) > 0 {
			b.WriteString(" Before accepting: " + strings.Join(missing, "; ") + ".")
		}
		b.WriteString("\n")
	}
	ev.Body = strings.TrimRight(b.String(), "\n")
	return ev
}

// proof says which part delivered quotes a Verification item, and where;
// "" when none does.
func proof(item string, parts []Part) string {
	want := plain(item)
	if want == "" {
		return ""
	}
	for _, p := range parts {
		if !p.Delivered() {
			continue
		}
		closed := ""
		if len(p.Closers) > 0 {
			closed = ", closed by " + closerName(p.Closers[0])
		}
		if strings.Contains(plain(work.Sections(p.Issue.Body)["Verification"]), want) {
			return fmt.Sprintf("quoted in #%d's Verification%s", p.ID, closed)
		}
		for _, c := range p.Closers {
			if strings.Contains(plain(c.Text), want) {
				return fmt.Sprintf("quoted by %s, which closed #%d", closerName(c), p.ID)
			}
		}
	}
	return ""
}

// TestFinder says where the code holds a test a Verification item names:
// the file, or "" when it holds none (ADR-0029).
type TestFinder func(name string) string

// codeSpan is a Markdown code span: where an item names a test.
var codeSpan = regexp.MustCompile("`([^`\n]+)`")

// testFile is a path a test lives in, by the conventions of the common
// languages: a tests or spec folder, a _test, .test, .spec or _spec file,
// a test_ file.
var testFile = regexp.MustCompile(`(?i)(^|/)(tests?|specs?|__tests__)/|(_test|\.test|\.spec|_spec)\.[a-z0-9]+$|(^|/)test_[^/]+$`)

// testFunc is a test's own name: Go's and JUnit's TestX, Python's test_x,
// JavaScript's and Java's testX.
var testFunc = regexp.MustCompile(`^(Test[A-Z0-9_]\w*|test_\w+|test[A-Z]\w*)$`)

// testBefore is the word that says the next code span names a test.
var testBefore = regexp.MustCompile(`(?i)\b(tests?|cases?|specs?)\s*$`)

// TestNames are the tests a Verification item names, each in a code span:
// a test file (a path, with ::name after it for one test in it), a test's
// own name, or any name right after the word test or case. The rest of
// the item is prose, and names none.
func TestNames(item string) []string {
	var out []string
	for _, m := range codeSpan.FindAllStringSubmatchIndex(item, -1) {
		name := strings.TrimSpace(item[m[2]:m[3]])
		path, _, _ := strings.Cut(name, "::")
		switch {
		case strings.ContainsAny(name, " \t"):
		case testFile.MatchString(path), testFunc.MatchString(name), testBefore.MatchString(item[:m[0]]):
			if !slices.Contains(out, name) {
				out = append(out, name)
			}
		}
	}
	return out
}

// CodeTests finds a test in the repository at HEAD: a test file named by
// its path, one test in it by path::name, or a name found as a word in a
// test file (TestNames). The run's commit, never the working tree.
func CodeTests(repo string) TestFinder {
	return func(name string) string {
		path, fn, two := strings.Cut(name, "::")
		if testFile.MatchString(path) && (two || strings.Contains(path, "/") || strings.Contains(path, ".")) {
			if exec.Command("git", "-C", repo, "cat-file", "-e", "HEAD:"+path).Run() != nil {
				return ""
			}
			if !two {
				return path
			}
			if exec.Command("git", "-C", repo, "grep", "-q", "-w", "-F", "-e", fn, "HEAD", "--", path).Run() != nil {
				return ""
			}
			return path
		}
		out, err := exec.Command("git", "-C", repo, "grep", "-l", "-w", "-F", "-e", name, "HEAD").Output()
		if err != nil {
			return ""
		}
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if p := strings.TrimPrefix(l, "HEAD:"); testFile.MatchString(p) {
				return p
			}
		}
		return ""
	}
}

// closerName says what closed an issue as the forge names it.
func closerName(c forge.Closer) string {
	switch {
	case c.Kind == "commit":
		return "commit " + c.Ref
	case strings.HasPrefix(c.Ref, "!"):
		return "merge request " + c.Ref
	}
	return "pull request " + c.Ref
}

// cell is a text fit for a table's cell: one line, no column mark.
func cell(s string) string {
	return strings.ReplaceAll(squeeze(s), "|", `\|`)
}

// readToAccept finds the open parents whose parts are all closed, and
// records them: the report is rewritten when they change, not otherwise.
func (p *Plan) readToAccept() {
	var ids []int
	for _, id := range sortedIDs(p.issues) {
		parts := Parts(p.issues[id])
		if id != p.Report && len(parts) > 0 && !slices.ContainsFunc(parts, func(n int) bool { return p.open[n] }) {
			ids = append(ids, id)
		}
	}
	if !slices.Equal(ids, p.Record.ToAccept) {
		p.Record.ToAccept, p.Changed = ids, true
	}
}

// toAccept is the report's part on the parents whose parts are all closed:
// for a person to accept, by closing them (ADR-0029).
func (p *Plan) toAccept() string {
	var lines []string
	for _, id := range p.Record.ToAccept {
		is := p.issues[id]
		lines = append(lines, fmt.Sprintf("- #%d %s: its %d parts are closed; what each delivered, and what is not proved, is on the issue. Close it to accept the need, or reopen a part.", id, is.Title, len(Parts(is))))
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n## To accept\n\nNeeds split into parts, every part closed: a person accepts each by closing it; the role never does.\n\n" + strings.Join(lines, "\n") + "\n"
}

// sortedIDs are the numbers of the issues given, lowest first.
func sortedIDs(issues map[int]forge.Issue) []int {
	ids := make([]int, 0, len(issues))
	for id := range issues {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
