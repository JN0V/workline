package reviewer

import (
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

// answered are the conversations on the merge request's lines the
// reviewer opened that the code now answers, by the rule a block goes by
// (#326, #1039): each one's thread id, for the engine to resolve. Only
// the threads of this run's side (ADR-0050), not resolved, whose first
// comment workline's own account wrote and carries a key; one a person
// spoke in is theirs to resolve, and a person's thread is never one. A
// thread is answered when its key is no longer found on its file at the
// head (the line changed, or was removed), or when its lens read the
// file again — or, for a lens whose judge reads tests, a test file that
// touches it — and found it no more. A finding found again, or a block
// still standing, keeps its thread open. A forge that lists no thread
// answers none; one that cannot be read is said, never an error.
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
		if gone(linesOf(t.Path), t.Path, by, keys[0]) || readAgain(repo, s, st, r, lensOf, by, t.Path) {
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
