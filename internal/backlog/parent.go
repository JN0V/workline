package backlog

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/testdef"
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
	text := strings.TrimSpace(work.ByMeaning(body)[work.Verification].Text)
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
	Body       string
	Parts      int
	Closed     int      // the parts closed, or gone from the forge
	Undone     []int    // the parts closed without delivering: not planned, a duplicate, gone
	Unproved   []string // the Verification items no part delivered quotes, or whose test is not in the code
	NoTest     []string // the tests a quoted item names that the code does not hold
	TestUnread []string // the tests a quoted item names that could not be looked for, with why
	Unread     []Part   // the parts whose closer the forge refused to say, with why
	AllClosed  bool
}

// ReadEvidence writes a parent's report from its parts: what each became
// and what closed it; each item of its Verification proved by a part
// delivered that quotes it — in its own Verification, in the scenario of
// its Validation the split gave the item (#1279), or in the text of the
// pull request or commit that closed it — and, when it names a test,
// that test found in the code by tests (nil: not looked for); or said not
// proved.
func ReadEvidence(parent forge.Issue, parts []Part, role string, tests TestFinder, def testdef.Def) Evidence {
	ev := Evidence{Parts: len(parts)}
	var b strings.Builder
	fmt.Fprintf(&b, "**The parts of this need**, as they close: kept up to date by the %s role, which closes this issue only on a person's yes — accepting the need is a person's.\n\n", strings.ReplaceAll(role, "-", " "))
	b.WriteString("| Part | State | Closed by |\n|---|---|---|\n")
	for _, p := range parts {
		title, state, by := p.Issue.Title, "open", "—"
		switch {
		case p.Gone:
			title, state = "", "gone from the forge (deleted or moved): its part not delivered here"
		case !p.Issue.Closed && Readied(p.Issue):
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
	proofs := work.Name(work.Verification)
	if len(items) == 0 {
		fmt.Fprintf(&b, "\nThis issue has no %s of its own: nothing to prove its parts against; a person judges from the parts alone.\n", proofs)
	} else {
		fmt.Fprintf(&b, "\n**Its %[1]s**, each item against the parts delivered — proved when a part's %[1]s, the scenario of its %[2]s that delivers it, or what closed it, quotes it, and a test it names, or that part's %[1]s declares for that scenario, is in the code:\n\n", proofs, work.Name(work.Validation))
		for _, item := range items {
			q := proof(item, parts)
			where := q.where
			if q.short != "" {
				// Quoted where the split gave it, but no test declared for
				// it: a draft's words are no proof (#1279).
				ev.Unproved = append(ev.Unproved, item)
				fmt.Fprintf(&b, "- %s: \"%s\" — %s, but %s.\n", notYet, item, where, q.short)
				continue
			}
			if where != "" && tests != nil {
				var in, missing, unread []string
				// The tests the item names, then those the part's
				// Verification declares for the scenario quoting it.
				type look struct {
					name string
					find func() (string, error)
				}
				var looks []look
				for _, name := range TestNames(item, def) {
					looks = append(looks, look{name, func() (string, error) { return tests(name) }})
				}
				for _, id := range q.tests {
					if slices.Contains(TestNames(item, def), id) {
						continue
					}
					looks = append(looks, look{id, func() (string, error) { return declaredTest(tests, id) }})
				}
				for _, l := range looks {
					name := l.name
					path, err := l.find()
					switch {
					case err != nil:
						unread = append(unread, "`"+name+"`")
						if why := fmt.Sprintf("`%s` (%v)", name, err); !slices.Contains(ev.TestUnread, why) {
							ev.TestUnread = append(ev.TestUnread, why)
						}
					case path == "":
						missing = append(missing, "`"+name+"`")
					case path == name || strings.HasPrefix(name, path+"::"):
						in = append(in, "`"+name+"`") // a file, or a test in it, named by its path
					default:
						in = append(in, fmt.Sprintf("`%s` in %s", name, path))
					}
				}
				switch {
				case len(missing) > 0:
					ev.Unproved = append(ev.Unproved, item)
					ev.NoTest = append(ev.NoTest, missing...)
					fmt.Fprintf(&b, "- %s: \"%s\" — %s, but the code holds no test %s: a test named is no proof until it is there.\n", notYet, item, where, strings.Join(missing, ", "))
					continue
				case len(unread) > 0:
					// Never said missing when git could not say: not looked for.
					ev.Unproved = append(ev.Unproved, item)
					fmt.Fprintf(&b, "- %s: \"%s\" — %s, but the test %s could not be looked for in the code: not proved until it is read.\n", notYet, item, where, strings.Join(unread, ", "))
					continue
				}
				if len(in) > 0 {
					where += "; the test " + strings.Join(in, ", ") + ", read from the code"
				}
				// A name alone may be a test's: looked up in the test files,
				// said either way, never a missing test (ADR-0065, Rule A).
				var seen, none []string
				for _, name := range BareNames(item, def) {
					if path, err := tests(name); err == nil && path != "" {
						seen = append(seen, fmt.Sprintf("`%s` in %s", name, path))
					} else {
						none = append(none, "`"+name+"`")
					}
				}
				if len(seen) > 0 {
					where += "; " + strings.Join(seen, ", ") + ", a test file's word, read from the code"
				}
				if len(none) > 0 {
					where += "; " + strings.Join(none, ", ") + " not found in a test file"
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
		fmt.Fprintf(&b, "\n**All %d parts are closed: for a person to accept.** Close this issue as completed, or label it `%s`, to accept the need; or keep it open — reopen a part, or open one for what is missing.", ev.Parts, LabelAccepted)
		var missing []string
		if len(ev.Undone) > 0 {
			missing = append(missing, "the part of "+issueList(ev.Undone)+" not delivered")
		}
		if n := len(ev.Unproved); n > 0 {
			missing = append(missing, fmt.Sprintf("%d item(s) of its %s not proved", n, work.Name(work.Verification)))
		}
		if len(missing) > 0 {
			b.WriteString(" Before accepting: " + strings.Join(missing, "; ") + ".")
		}
		b.WriteString("\n")
	}
	ev.Body = strings.TrimRight(b.String(), "\n")
	return ev
}

// quoted is where a part delivered quotes a parent's Verification item.
type quoted struct {
	where string   // where it is quoted; "" when no part delivered quotes it
	tests []string // the tests the part's Verification declares for the scenario quoting it, to find in the code
	short string   // why a quote in a part's Validation proves nothing: no test declared for it
}

// proof says which part delivered quotes a Verification item, and where:
// in its Verification, in the scenario of its Validation the split gave
// it — proved only by the tests the part's Verification declares for that
// scenario (#1279) —, or in what closed it. A quote that proves wins over
// one that does not.
func proof(item string, parts []Part) quoted {
	want := plain(item)
	if want == "" {
		return quoted{}
	}
	var weak quoted
	for _, p := range parts {
		if !p.Delivered() {
			continue
		}
		closed := ""
		if len(p.Closers) > 0 {
			closed = ", closed by " + closerName(p.Closers[0])
		}
		sections := work.ByMeaning(p.Issue.Body)
		if s := sections[work.Verification]; strings.Contains(plain(s.Text), want) {
			return quoted{where: fmt.Sprintf("quoted in #%d's %s%s", p.ID, s.Heading, closed)}
		}
		if s := sections[work.Validation]; strings.Contains(plain(s.Text), want) {
			q := scenarioProof(want, s.Text, sections[work.Verification])
			q.where = fmt.Sprintf("quoted in #%d's %s%s%s", p.ID, s.Heading, q.where, closed)
			if q.short == "" {
				return q
			}
			if weak.where == "" {
				weak = q
			}
			continue
		}
		for _, c := range p.Closers {
			if strings.Contains(plain(c.Text), want) {
				return quoted{where: fmt.Sprintf("quoted by %s, which closed #%d", closerName(c), p.ID)}
			}
		}
	}
	return weak
}

// noTestCan opens the tester's line for a scenario no test can check:
// "- **Empty day:** no test can check it: <why>".
const noTestCan = "no test can check it"

// scenarioProof is, for an item a part's Validation quotes, the scenario
// that quotes it and the tests the part's Verification declares for that
// scenario, each in a code span: where reads ", its scenario "Three
// sales", proved by `tests.test_x` in its Verification". Short says why
// none proves it: the quote in no scenario, no line for it, no test named,
// or no test can check it.
func scenarioProof(want, validation string, verification work.Section) quoted {
	name := ""
	for _, sc := range work.Scenarios(validation) {
		if strings.Contains(plain(sc.Name+" "+sc.Text), want) {
			name = sc.Name
			break
		}
	}
	if name == "" {
		return quoted{short: "in no scenario of it: no test is declared for it"}
	}
	q := quoted{where: fmt.Sprintf(", its scenario \"%s\"", name)}
	heading := verification.Heading
	if heading == "" {
		heading = work.Name(work.Verification)
	}
	for _, line := range declared(verification.Text) {
		label, text, _ := strings.Cut(line, "\x00")
		if plain(label) != plain(name) {
			continue
		}
		if why, ok := strings.CutPrefix(strings.TrimSpace(text), noTestCan); ok {
			q.short = fmt.Sprintf("its %s says no test can check it", heading)
			if why = strings.TrimRight(strings.TrimLeft(why, " :"), " ."); why != "" {
				q.short += " (" + why + ")"
			}
			q.short += ": a person judges"
			return q
		}
		for _, m := range codeSpan.FindAllStringSubmatch(text, -1) {
			if id := strings.TrimSpace(m[1]); !strings.ContainsAny(id, " \t") && !slices.Contains(q.tests, id) {
				q.tests = append(q.tests, id)
			}
		}
		break
	}
	if len(q.tests) == 0 {
		q.short = fmt.Sprintf("its %s names no test for that scenario", heading)
		return q
	}
	q.where += fmt.Sprintf(", proved by %s in its %s", "`"+strings.Join(q.tests, "`, `")+"`", heading)
	return q
}

// declaredLine is a scenario's line in a Verification, as the tester
// writes it: its name bold at the head of a list item, then its tests or
// why none: "- **Three sales:** `tests.test_x`.".
var declaredLine = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+\*\*([^*\n]+?):?\*\*:?\s*(.*)$`)

// declared are a Verification's scenario lines, each its name, a NUL, and
// its text: the lines indented under it with it, up to a blank line or
// the next item — never the prose after the list.
func declared(verification string) []string {
	var out []string
	open := false
	for _, l := range strings.Split(verification, "\n") {
		l = strings.TrimRight(l, "\r")
		switch m := declaredLine.FindStringSubmatch(l); {
		case m != nil:
			out = append(out, strings.TrimSpace(m[1])+"\x00"+strings.TrimSpace(m[2]))
			open = true
		case open && strings.TrimSpace(l) != "" && strings.TrimLeft(l, " \t") != l && !listMark.MatchString(l):
			out[len(out)-1] += " " + strings.TrimSpace(l)
		default:
			open = false
		}
	}
	return out
}

// listMark opens a list item.
var listMark = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+`)

// declaredTest finds a test a part's Verification declares, by its id as
// the tester ran it; when the code holds no such name, by the test's own
// name, its id's last word ("tests.test_x.test_y" is test_y).
func declaredTest(tests TestFinder, id string) (string, error) {
	path, err := tests(id)
	if err != nil || path != "" {
		return path, err
	}
	if last := id[strings.LastIndexAny(id, "./:")+1:]; last != "" && last != id && bareName.MatchString(last) {
		return tests(last)
	}
	return "", nil
}

// TestFinder says where the code holds a test a Verification item names:
// the file, or "" when it holds none; an error when it could not look,
// never read as none (ADR-0029).
type TestFinder func(name string) (string, error)

// codeSpan is a Markdown code span: where an item names a test.
var codeSpan = regexp.MustCompile("`([^`\n]+)`")

// testBefore are the words that say the next code span names a test:
// "test", "test case", "conformance case" — never "case" alone, prose's
// "in that case".
var testBefore = regexp.MustCompile(`(?i)\b(tests?|(test|conformance)\s+cases?)\s*$`)

// TestNames are the tests a Verification item names, each in a code span:
// a test file by the project's definition (ADR-0065; a path, a `/` or a
// `.` in it, with ::name after it for one test in it; a word alone is a
// BareName), or any name right after the word test,
// test case or conformance case. The rest of the item is prose, and names
// none.
func TestNames(item string, def testdef.Def) []string {
	var out []string
	for _, m := range codeSpan.FindAllStringSubmatchIndex(item, -1) {
		name := strings.TrimSpace(item[m[2]:m[3]])
		path, _, _ := strings.Cut(name, "::")
		switch {
		case strings.ContainsAny(name, " \t"):
		case strings.ContainsAny(path, "/.") && def.Test(path), testBefore.MatchString(item[:m[0]]):
			if !slices.Contains(out, name) {
				out = append(out, name)
			}
		}
	}
	return out
}

// bareName is a name alone in a code span: a word, no path.
var bareName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// BareNames are the names alone, in code spans, of a Verification item
// that TestNames does not take: each may be a test's own name, by no
// language's rule; looked up in the test files, never required there.
func BareNames(item string, def testdef.Def) []string {
	named := TestNames(item, def)
	var out []string
	for _, m := range codeSpan.FindAllStringSubmatch(item, -1) {
		name := strings.TrimSpace(m[1])
		if bareName.MatchString(name) && !slices.Contains(named, name) && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

// CodeTests finds a test in the repository at HEAD: a test file named by
// its path, one test in it by path::name, or a name found as a word in a
// test file (TestNames). The run's commit, never the working tree.
func CodeTests(repo string, def testdef.Def) TestFinder {
	// grep lists the files at HEAD holding a word, under path when given:
	// none is git's exit 1, any other failure an error.
	grep := func(word, path string) ([]string, error) {
		args := []string{"-C", repo, "grep", "-l", "-w", "-F", "-e", word, "HEAD"}
		if path != "" {
			args = append(args, "--", ":(literal)"+path)
		}
		out, err := exec.Command("git", args...).Output()
		var exit *exec.ExitError
		switch {
		case errors.As(err, &exit) && exit.ExitCode() == 1:
			return nil, nil
		case err != nil:
			return nil, fmt.Errorf("git grep: %v", err)
		}
		var files []string
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			files = append(files, strings.TrimPrefix(l, "HEAD:"))
		}
		return files, nil
	}
	return func(name string) (string, error) {
		path, fn, two := strings.Cut(name, "::")
		if def.Test(path) && (two || strings.Contains(path, "/") || strings.Contains(path, ".")) {
			// A file, a blob — never a folder —, its path taken literally.
			out, err := exec.Command("git", "-C", repo, "ls-tree", "HEAD", "--", ":(literal)"+path).Output()
			info, file, _ := strings.Cut(strings.TrimRight(string(out), "\n"), "\t")
			switch {
			case err != nil:
				return "", fmt.Errorf("git ls-tree: %v", err)
			case file != path || !strings.Contains(info, " blob "):
				return "", nil
			case !two:
				return path, nil
			}
			files, err := grep(fn, path)
			if err != nil || len(files) == 0 {
				return "", err
			}
			return path, nil
		}
		files, err := grep(name, "")
		for _, p := range files {
			if def.Test(p) {
				return p, nil
			}
		}
		return "", err
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

// sortedIDs are the numbers of the issues given, lowest first.
func sortedIDs(issues map[int]forge.Issue) []int {
	ids := make([]int, 0, len(issues))
	for id := range issues {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
