package sample

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/work"
)

// The tester's acts in the weekly sample (ADR-0046, the tester's row), on
// each merge request of its merged in the window, read where it did them:
// its tests, from its commit and the record on its Draft; an objection
// answered kept or rewritten, from its reply's hidden line, and the five
// whys a third answer posted with it, drawn apart; a removal of
// its curator's, from its commit and its pull request's proof. Not drawn:
// an objection to the need itself, sent to the product owner by the
// builder's own word, and a flaky verdict, three runs counted: mechanical
// acts, which conformance proves.

const testRole = "tester"

// testerKinds are the tester's kinds of act drawn, in the order it does them.
var testerKinds = []string{"tests", "kept", "rewritten", "whys", "removal"}

// testerAct is what a tester's act is re-checked against, read with it.
type testerAct struct {
	mr       int
	record   tester.Record
	recorded bool
	answered tester.Answered
	thread   []forge.Note // an answer's thread, up to it
	reply    string       // an answer's words
	body     string       // a removal's pull request, its proof
}

// testerActs are the tester's acts on its merge requests merged in the
// window, and what could not be read of them.
func testerActs(repo string, f forge.Forge, mrs []forge.MergeRequest) ([]Act, []string, error) {
	var out []Act
	var unread []string
	for _, m := range mrs {
		if !strings.HasPrefix(m.Branch, "workline/tester/") {
			continue
		}
		ref := forge.MergeRef(f, m.ID)
		list, found, err := mergedCommits(repo, m)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			unread = append(unread, fmt.Sprintf("the tester's acts on %s: its head is not in this clone", ref))
			continue
		}
		var commit, subject, by, day string
		for _, c := range strings.Fields(list) { // newest first: the first of its own kept
			info, err := git(repo, "log", "-1", "--format=%(trailers:key=Workline-Role,valueonly,separator=%x2C)%x00%(trailers:key=Workline-Model,valueonly,separator=%x2C%x20)%x00%aI%x00%s", c)
			if err != nil {
				return nil, nil, err
			}
			p := strings.Split(info, "\x00")
			if len(p) != 4 || !hasValue(p[0], testRole) ||
				!strings.HasPrefix(p[3], tester.TestsSubject) && !strings.HasPrefix(p[3], tester.CurateSubject) {
				continue
			}
			commit, by, subject, day = c, p[1], p[3], ""
			if t, err := time.Parse(time.RFC3339, p[2]); err == nil {
				day = t.UTC().Format(time.DateOnly)
			}
			break
		}
		switch {
		case strings.HasPrefix(m.Branch, tester.CurateBranch):
			if commit == "" {
				unread = append(unread, fmt.Sprintf("the tester's removal on %s: no commit of its on the head", ref))
				continue
			}
			full, err := f.MergeRequest(m.ID)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, Act{Role: testRole, Kind: "removal", Ref: ref, Commit: commit, Day: day, By: by, Line: subject, key: commit,
				tester: &testerAct{mr: m.ID, body: full.Body}})
		case strings.HasPrefix(m.Branch, tester.TestsBranch):
			r, ok, err := tester.ReadRecord(f, m.ID)
			if err != nil {
				return nil, nil, err
			}
			if commit != "" {
				out = append(out, Act{Role: testRole, Kind: "tests", Ref: ref, Commit: commit, Day: day, By: by, Line: subject, key: commit,
					tester: &testerAct{mr: m.ID, record: r, recorded: ok}})
			}
			answers, why, err := answeredOn(f, m.ID, ref, r, ok)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, answers...)
			if why != "" {
				unread = append(unread, why)
			}
		}
	}
	return out, unread, nil
}

// mergedCommits are the commits of a merge request merged, newest first,
// read on its head: the ref the sample fetched, else its branch; past the
// base it was merged on. found is false when its head is not in the clone.
func mergedCommits(repo string, m forge.MergeRequest) (list string, found bool, err error) {
	head := ""
	for _, h := range []string{fmt.Sprintf("refs/workline/sample/%d", m.ID), m.Branch} {
		if c, err := git(repo, "rev-parse", "-q", "--verify", h+"^{commit}"); err == nil && c != "" {
			head = c
			break
		}
	}
	if head == "" {
		return "", false, nil
	}
	args := []string{"rev-list", "--no-merges", head}
	if m.MergedAs != "" {
		if p, err := git(repo, "rev-parse", "-q", "--verify", m.MergedAs+"^1"); err == nil && p != "" {
			args = append(args, "^"+p)
		}
	}
	list, err = git(repo, args...)
	return list, true, err
}

// answeredOn are the tester's answers kept or rewritten in the threads of
// its merge request id, each from its reply's hidden line.
func answeredOn(f forge.Forge, id int, ref string, r tester.Record, recorded bool) ([]Act, string, error) {
	th, ok := f.(forge.Threaded)
	if !ok {
		return nil, "the tester's answers to objections: this forge lists no conversation on a line", nil
	}
	threads, err := th.Threads(id)
	if err != nil {
		return nil, "", err
	}
	own, err := forge.OwnAccount(f)
	if err != nil {
		return nil, "", err
	}
	var out []Act
	for _, t := range threads {
		for i, c := range t.Comments {
			if !own.Wrote(c) || !tester.IsAnswered(c.Body) {
				continue
			}
			ans, ok := tester.ReadAnswered(c.Body)
			if !ok {
				continue // an answer before its line was kept: not drawn
			}
			words := answerWords(c.Body)
			day := c.Created
			if len(day) > len(time.DateOnly) {
				day = day[:len(time.DateOnly)]
			}
			out = append(out, Act{Role: testRole, Kind: ans.Act, Ref: ref, Where: ans.Test, Commit: ans.At, Day: day, By: ans.Model,
				Line: words, key: fmt.Sprintf("%s/%s/%d", ref, t.ID, i),
				tester: &testerAct{mr: id, record: r, recorded: recorded, answered: ans, thread: t.Comments[:i], reply: words}})
			if ans.Whys { // the analysis, its own act: its root cause read, not the answer's
				w := out[len(out)-1]
				w.Kind, w.key = "whys", w.key+"/whys"
				out = append(out, w)
			}
		}
	}
	return out, "", nil
}

var hiddenLine = regexp.MustCompile(`<!--[^>]*-->`)

// answerWords are a reply's words, its hidden lines left out.
func answerWords(body string) string {
	return strings.TrimSpace(hiddenLine.ReplaceAllString(body, ""))
}

// testerMaterial gathers what a tester's act was done on.
func (rd *reader) testerMaterial(a *Act, m *material) string {
	t := a.tester
	if t == nil {
		return "its record was not read with it"
	}
	m.author, m.acted = actedBy(a.By)
	repo := rd.o.Repo
	switch a.Kind {
	case "tests":
		if !t.recorded {
			return fmt.Sprintf("the tester's record of its tests on %s cannot be read", a.Ref)
		}
		scen, about := t.record.Scenarios, "The scenarios as the tester read them"
		is, err := rd.o.Forge.Issue(t.record.Issue)
		if err != nil {
			return fmt.Sprintf("#%d cannot be read: %v", t.record.Issue, err)
		}
		if len(scen) == 0 {
			scen, about = tester.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text), "The scenarios as the issue reads now (its record, older, keeps none)"
		}
		m.items = append(m.items, scenariosItem(scen))
		files := slices.Sorted(maps.Keys(t.record.Files))
		for _, file := range files {
			content, err := git(repo, "show", a.Commit+":"+file)
			if err != nil {
				return fmt.Sprintf("%s is not at %.7s, the tester's commit", file, a.Commit)
			}
			m.items = append(m.items, item{name: file, text: content, full: content, code: true})
		}
		var paths []string
		listed, _ := git(repo, "ls-tree", "-r", "--name-only", t.record.Base)
		for _, w := range work.ScopeWords(work.ByMeaning(is.Body)[work.Scope].Text) {
			if f := work.InRepo(w, strings.Fields(listed)); f != "" {
				w = f // `parts.go:943`: the file at the base
			}
			if !slices.Contains(paths, w) {
				paths = append(paths, w)
			}
		}
		code := codeAt(repo, t.record.Base, paths, codeLines)
		for i := range code {
			code[i].name += " (base)"
		}
		m.items = append(m.items, code...)
		m.act = fmt.Sprintf("The tests the tester wrote for #%d's scenarios, in commit %.7s, proven red on the base %.7s before any code was written.", t.record.Issue, a.Commit, t.record.Base)
		m.about = about + "; each test file at the tester's commit; the code the issue's " + work.Name(work.Scope) + " names, at the base (`(base)`)."
	case "kept", "rewritten", "whys":
		ans := t.answered
		m.items = append(m.items, scenariosItem([]tester.Scenario{ans.Scenario}), item{name: "objection", text: threadText(t.thread), full: threadText(t.thread)})
		before, err := git(repo, "show", ans.At+":"+ans.Test)
		if err != nil {
			return fmt.Sprintf("%s is not at %.7s, where the answer read it", ans.Test, ans.At)
		}
		if a.Kind != "rewritten" {
			m.items = append(m.items, item{name: ans.Test, text: before, full: before, code: true})
			m.act = fmt.Sprintf("The tester kept its test `%s` against the builder's objection.", ans.Test)
			if a.Kind == "whys" {
				m.act = fmt.Sprintf("Two answers did not settle the builder's objection to `%s`: the tester's third gives the five whys, the cause and one action.", ans.Test)
			}
		} else {
			after, err := git(repo, "cat-file", "-p", ans.Blob)
			if err != nil {
				return fmt.Sprintf("the rewrite of %s (blob %.7s) is not in this clone", ans.Test, ans.Blob)
			}
			m.items = append(m.items, item{name: "before", text: before, full: before, code: true}, item{name: "after", text: after, full: after, code: true})
			m.act = fmt.Sprintf("The tester rewrote its test `%s` on the builder's objection: `before`, as it stood; `after`, as rewritten.", ans.Test)
		}
		m.items = append(m.items, item{name: "answer", text: t.reply, full: t.reply})
		m.about = "The scenario as the tester read it, the objection's thread up to the answer, the test, and the answer."
	case "removal":
		diff, err := git(repo, "show", "--format=", "--no-color", "--patch", a.Commit)
		if err != nil {
			return fmt.Sprintf("the diff of %.7s cannot be read", a.Commit)
		}
		diff = cutDiff(diff)
		files, _ := git(repo, "diff-tree", "--no-commit-id", "--name-only", "-r", a.Commit)
		m.items = append(m.items, item{name: "pull request", text: t.body, full: t.body}, item{name: "diff", text: diff, full: diff})
		m.items = append(m.items, codeAt(repo, a.Commit, strings.Fields(files), codeLines)...)
		m.act = fmt.Sprintf("The tester's curator removed or merged tests in commit %.7s, opened as %s.", a.Commit, a.Ref)
		m.about = "Its pull request (the proof's numbers and why each test goes), the diff, and the test files it touched as they read after it."
	default:
		return fmt.Sprintf("no question for the tester's `%s`", a.Kind)
	}
	return ""
}


// scenariosItem names each scenario with its Given / When / Then.
func scenariosItem(scen []tester.Scenario) item {
	var b strings.Builder
	for _, s := range scen {
		fmt.Fprintf(&b, "- **%s:** %s\n", s.Name, s.Text)
	}
	return item{name: "scenarios", text: b.String(), full: b.String()}
}

// threadText is an objection's thread up to the answer, each comment's
// hidden lines left out.
func threadText(notes []forge.Note) string {
	var b strings.Builder
	for i, n := range notes {
		who := n.Author
		if who == "" {
			who = "workline"
		}
		fmt.Fprintf(&b, "%d. %s: %s\n\n", i+1, who, answerWords(n.Body))
	}
	return strings.TrimSpace(b.String())
}
