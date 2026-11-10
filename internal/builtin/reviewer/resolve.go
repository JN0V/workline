package reviewer

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// lensesRead is what a run's lenses read: those that answered, on a review
// read whole (no lens failed, no finding left unjudged), and the files of
// the commits they read. A run asking no lens reads nothing.
type lensesRead struct {
	lenses []string
	files  []string
}

// answered are the reviewer's own open threads on this run's side that
// the code now answers, as a block goes (judge.md): its keyed line gone
// from the head, or its lens read the file again (a test file touching it,
// for the tests lens) and found it no more; or the tester's answer that
// no test is needed for it, weighed, holds (#1303). A thread a person opened or
// spoke in is left; one unread is said, never an error.
func answered(repo string, s Settings, st state, lines []forge.LineComment, r lensesRead) ([]string, *verdict.Finding) {
	t := mergeRequest()
	if t == nil || !s.ForgeWrites {
		return nil, nil
	}
	unread := func(err error) ([]string, *verdict.Finding) {
		return nil, &verdict.Finding{Rule: "threads-unread", Level: "warn",
			Message: "the merge request's conversations on lines could not be read, none of the reviewer's resolved: " + forge.Said(err)}
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return unread(err)
	}
	th, ok := f.(forge.Threaded)
	if _, resolves := f.(forge.Resolver); !ok || !resolves {
		return nil, nil
	}
	kept, err := forge.ReadKept(f, *t)
	if err != nil {
		return unread(err)
	}
	threads, err := th.Threads(t.ID)
	if err != nil {
		return unread(err)
	}
	stands := map[string]bool{}
	for _, c := range lines {
		stands[c.Key] = true
	}
	files := map[string][]string{}
	linesOf := func(path string) []string {
		if l, ok := files[path]; ok {
			return l
		}
		if text, ok := fileAt(repo, st.Head, path); ok {
			files[path] = strings.Split(text, "\n")
		} else {
			files[path] = nil
		}
		return files[path]
	}
	for _, b := range st.Standing {
		if path, n, ok := fileLine(b.Where); ok {
			stands[lineKeyOf(linesOf(path), path, n, b.Lens)] = true
		}
	}
	lensOf := lensReader(repo)
	fixed := fixedRows(kept)
	var out []string
	for _, t := range threads {
		if t.Resolved || len(t.Comments) == 0 || !kept.Wrote(t.Comments[0]) {
			continue
		}
		if slices.ContainsFunc(t.Comments[1:], func(n forge.Note) bool { return !kept.Wrote(n) && !n.Bot }) {
			continue // a person spoke in it: theirs to resolve
		}
		keys := forge.LineKeys(t.Comments[0].Body)
		if len(keys) != 1 || stands[keys[0]] {
			continue
		}
		by, _, _ := strings.Cut(keys[0], "/")
		if !ofThisSide(s, st, by) {
			continue
		}
		// A lens asked this run that did not answer whole could not raise
		// it again: its warning waits for a run that reads cleanly.
		failed := slices.Contains(facets(st), by) && !slices.Contains(r.lenses, by)
		warnFixed := !s.AIFindings.Blocks(by) && !failed && fixedWarn(fixed, by, t)
		// A finding the tester's answer settled, its reason weighed (#1303).
		title, _ := lineTitle(t.Comments[0].Body)
		weighed := st.settles(by, fmt.Sprintf("%s:%d", t.Path, t.Line), title)
		if gone(linesOf(t.Path), t.Path, by, keys[0]) || readAgain(repo, s, st, r, lensOf, by, t.Path) || warnFixed || weighed {
			out = append(out, t.ID)
		}
	}
	return out, nil
}

// ofThisSide says whether a key's finder is this run's to settle: a lens
// of its side, or a rule on the verdict's run.
func ofThisSide(s Settings, st state, by string) bool {
	if slices.Contains(s.Lenses, by) {
		return s.ofSide(st.Warn, by)
	}
	return st.Warn != warnOnly
}

// gone says whether no line of the file at the head holds key: the line
// it was found on changed, or was removed, or its file is gone.
func gone(lines []string, path, by, key string) bool {
	for n := 1; n <= len(lines); n++ {
		if lineKeyOf(lines, path, n, by) == key {
			return false
		}
	}
	return true
}

// readAgain says whether the lens by read the file again and did not find
// its finding: the file among those the commits it read change, or, for
// a lens whose judge reads tests, a test file touching it.
func readAgain(repo string, s Settings, st state, r lensesRead, lensOf func(string) (Lens, error), by, path string) bool {
	if !slices.Contains(r.lenses, by) {
		return false
	}
	if slices.Contains(r.files, path) {
		return true
	}
	l, err := lensOf(by)
	if err != nil || l.Judge.Reads != "tests" {
		return false
	}
	return slices.ContainsFunc(testFiles(repo, st.Head, path, s.Tests), func(f string) bool { return slices.Contains(r.files, f) })
}

// resolvedLine is what the summary comment says of the conversations it
// resolves.
func resolvedLine(n int) string {
	return fmt.Sprintf("**Resolved**: %d %s on %s of a review before, the code answering %s now.",
		n, plural(n, "conversation", "conversations"), plural(n, "a line", "lines"), plural(n, "it", "them"))
}

// fixedRows are the reviewer's rows the developer's rounds asked and
// answered with a fix pushed, its tests green, as its record on the merge
// request keeps them (sticky=developer/code): what a warning's thread
// needs to be settled (#1050). No record, or one that does not read: none.
func fixedRows(kept forge.Kept) []string {
	c, ok := kept.Last(forge.Marker("sticky=developer/code"))
	if !ok {
		return nil
	}
	const opening = "<!-- workline:developer-act "
	i := strings.Index(c, opening)
	if i < 0 {
		return nil
	}
	body, _, _ := strings.Cut(c[i+len(opening):], " -->")
	var act struct {
		Rounds []struct {
			Asked  []string `json:"asked"`
			Result string   `json:"result"`
		} `json:"rounds"`
	}
	if json.Unmarshal([]byte(body), &act) != nil {
		return nil
	}
	var out []string
	for _, r := range act.Rounds {
		if r.Result == "green" {
			out = append(out, r.Asked...)
		}
	}
	return out
}

// fixedWarn says whether a round of the developer answered a warning's
// thread, a fix pushed: its row names the thread's lens, file and title
// (`warns <lens> `+"`path:line`"+`: Important: <title>`). A warning asks nothing
// (ADR-0051, cut 4), so the run after that fix, not finding it again,
// settles it, its lens reading the file or not (#1050).
func fixedWarn(rows []string, lens string, t forge.Thread) bool {
	_, rest, ok := strings.Cut(t.Comments[0].Body, "**Important: ")
	if !ok {
		return false
	}
	title, _, _ := strings.Cut(rest, "**")
	for _, row := range rows {
		head, text, ok := strings.Cut(row, ": Important: ")
		if !ok || text != title || !strings.HasPrefix(head, "warns "+lens+" `"+t.Path+":") {
			continue
		}
		return true
	}
	return false
}
