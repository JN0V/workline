package reviewer

// The repository finder: once a merge request is otherwise
// mergeable, one finder reads the repository at the head — callers,
// callees, the workflows that run the tests — blind to what the lenses
// found and what their judges said. It is a question that may read, as
// a judge of the code is: the engine serves its reads from
// git. Its yes names a defect and quotes its cause; found again, the
// finding is judged apart from it, then holds the merge as a lens's does
// (ai-findings, by the name `repository`).

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/kit/reviewmark"
	"github.com/JN0V/workline/internal/kit/verdict"
)

const (
	finderLens = "repository" // its findings' lens, its prompt lenses/repository.md
	finderKey  = "finder"     // its question: in/judge/finder
	// finderFilesMax is how many of the repository's file names its
	// question lists; the rest counted, read by a grep.
	finderFilesMax = 400
)

// finderAnswer is how the finder answers, in its question alone: the
// judge's facets ask one note.
const finderAnswer = "\nYour answer: yes, when you found a defect, naming it in one sentence, with the line that causes it quoted beside the note, as it reads at the head, with its path — a line of the change where the change causes it; no, when you found none; or unsure. A yes whose quote is not found again is dropped:\n\n" +
	"- note: \"yes: Mean now calls sum, which skips the first value, so every mean comes out low.\"\n- quote: {path: \"calc/calc.go\", text: \"return sum(values) / len(values)\"}\n\n" +
	"- note: \"no: the one caller of Parse, in cmd/main.go, already handles its new error.\"\n"

// finderDue says whether this run may ask the finder: on a merge request,
// on its side of a split review, never asked on it yet, or its own block
// standing at a head it did not read.
func finderDue(s Settings, st state) bool {
	if !s.RepoFinder || st.Spec != nil || mergeRequest() == nil || !s.ofSide(st.Warn, finderLens) {
		return false
	}
	if st.Record.Finder == "" {
		return true
	}
	return st.Record.Finder != st.Head && slices.ContainsFunc(st.Standing, func(b Block) bool { return b.Lens == finderLens })
}

// mergeable says whether a review holds nothing but the finder's own
// block: every lens answered, no finding blocking.
func mergeable(v Review) bool {
	return v.Complete && !slices.ContainsFunc(v.Findings, func(f verdict.Finding) bool { return f.Blocks() && f.Rule != finderLens })
}

// finder asks the repository finder when the review is otherwise
// mergeable, then reads its answer: a finding found again is added to
// the candidates and its judge asked. true: a question was written, the
// engine asks it before pre settles.
func finder(runDir, repo string, s Settings, st *state, c *candidates) (bool, error) {
	dir := filepath.Join(runDir, "in", "judge", finderKey)
	if !exists(filepath.Join(dir, "question.yaml")) {
		if !finderDue(s, *st) {
			return false, nil
		}
		dry := s
		dry.ForgeWrites = false // what it would settle, nothing proposed
		v, _, err := settled(runDir, repo, dry, *st, *c)
		if err != nil {
			return false, err
		}
		if !mergeable(v) {
			c.Finder = "waits until nothing else holds the merge"
			return false, nil
		}
		l, err := readLens(repo, finderLens)
		if err != nil {
			return false, err
		}
		m, cause := finderMaterial(repo, s, *st)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false, err
		}
		r := reading{Question: strings.Join(strings.Fields(l.Text), " "), Material: m, Cause: cause, Find: true}
		return true, r.write(dir, s, st.Head)
	}
	if c.Finder != "" { // read on a pass before: its finding's judge answered since
		return false, nil
	}
	a, err := answerIn(dir)
	rd, _ := readingOf(dir)
	read := rd.said()
	switch {
	case err != nil || a.Error != "":
		why := a.Error
		if why == "" {
			why = "the finder was not asked"
		}
		c.Finder = "did not run (" + why + "): it is asked again at the next run"
		return false, nil
	case len(a.Reads) > 0:
		// Unsure: not a read, so its block stands and it is asked again.
		c.Finder = "could not tell: it asked to read again, no read left (judge-read-rounds, judge-read-lines-max)" + read + "; it is asked again at the next run"
		return false, nil
	case a.Unsure:
		c.Finder = "could not tell: " + a.Why + read + "; it is asked again at the next run"
		return false, nil
	case a.Yes == nil:
		c.Finder = "did not run (no answer): it is asked again at the next run"
		return false, nil
	case !*a.Yes:
		c.FinderRead, c.Finder = true, "read the repository and found nothing"+read
		return false, nil
	}
	c.FinderRead = true
	cause := map[string]string{}
	if a.Quote != nil {
		cause = map[string]string{"path": a.Quote.Path, "quote": a.Quote.Text}
	}
	f, why := found(repo, *st, finderLens, false, map[string]any{"severity": "important", "title": oneLine(a.Why, 120), "why": a.Why, "cause": cause}, "")
	if why != "" {
		c.Finder = fmt.Sprintf("found %q, dropped: %s%s", oneLine(a.Why, 120), why, read)
		return false, nil
	}
	c.Finder = "read the repository and found one defect" + read
	if f.Related {
		c.Related = append(c.Related, f)
	} else {
		c.Outside = append(c.Outside, f)
	}
	// Its judge, apart from the finder as from the lenses.
	one := candidates{}
	if f.Related {
		one.Related = []Finding{f}
	} else {
		one.Outside = []Finding{f}
	}
	n := judged(runDir)
	if err := askJudges(runDir, repo, s, *st, &one, &n); err != nil {
		return false, err
	}
	c.Asked = append(c.Asked, one.Asked...)
	for k, v := range one.Same {
		if c.Same == nil {
			c.Same = map[string]reviewmark.Answered{}
		}
		c.Same[k] = v
	}
	for k, v := range one.Reused {
		if c.Reused == nil {
			c.Reused = map[string]reviewmark.Answered{}
		}
		c.Reused[k] = v
	}
	if len(one.Asked) == 0 { // judged before on the same patch: settled now
		return false, nil
	}
	for _, a := range one.Asked {
		if err := authored(filepath.Join(runDir, "in", "judge", a[strings.LastIndex(a, "\x00")+1:]), dir, s, st.Head); err != nil {
			return false, err
		}
	}
	return true, writeJSON(filepath.Join(runDir, "in", candidatesFile(*st)), *c)
}

// judged is the highest number a judge's question of this run holds: the
// finder's finding's is the next.
func judged(runDir string) int {
	dirs, _ := filepath.Glob(filepath.Join(runDir, "in", "judge", "*"))
	n := 0
	for _, d := range dirs {
		if k, err := strconv.Atoi(filepath.Base(d)); err == nil {
			n = max(n, k)
		}
	}
	return n
}

// authored names the finder's model in its finding's question: the
// judge is picked apart from it.
func authored(dir, finderDir string, s Settings, head string) error {
	a, err := answerIn(finderDir)
	if err != nil || a.Model == "" {
		return nil
	}
	if r, ok := readingOf(dir); ok {
		r.Author = a.Model
		return r.write(dir, s, head)
	}
	var q map[string]any
	if err := readJSON(filepath.Join(dir, "question.yaml"), &q); err != nil {
		return err
	}
	q["author"] = a.Model
	return writeJSON(filepath.Join(dir, "question.yaml"), q)
}

// finderMaterial is what the finder is given: the whole change, base to
// head, and the repository's files by name; nothing a lens or a judge
// said. cause is the first file the change touches: a name is looked
// for there first.
func finderMaterial(repo string, s Settings, st state) (string, string) {
	var b strings.Builder
	diff, _ := git(repo, "diff", "-U3", "--no-color", "--no-ext-diff", "--find-renames", st.Base, st.Head, "--")
	fmt.Fprintf(&b, "## The change\n\nEvery commit of the merge request, as git shows it, %s..%s:\n\n```diff\n%s```\n", short(st.Base), short(st.Head), capLines(diff, s.DiffLinesMax, "the change is cut here, past diff-lines-max"))
	out, _ := git(repo, "ls-tree", "-r", "--name-only", st.Head)
	names := strings.Fields(out)
	shown := names[:min(len(names), finderFilesMax)]
	fmt.Fprintf(&b, "\n## The repository's files at %s\n\n```\n%s\n```\n", short(st.Head), strings.Join(shown, "\n"))
	if n := len(names) - len(shown); n > 0 {
		fmt.Fprintf(&b, "\n%d more %s not listed: find them with a grep.\n", n, plural(n, "file", "files"))
	}
	cause := ""
	if files, err := changedFiles(repo, st.Base, st.Head); err == nil && len(files) > 0 {
		cause = files[0].Path
	}
	return b.String(), cause
}

// finderReads are the rounds of reads the finder asked, as a judge's are
// said in out/review.json.
func finderReads(runDir string) []JudgeRead {
	r, ok := readingOf(filepath.Join(runDir, "in", "judge", finderKey))
	if !ok {
		return nil
	}
	var out []JudgeRead
	for i, round := range r.Rounds {
		out = append(out, JudgeRead{For: "judge/" + finderKey, Lens: finderLens, Round: i + 1, Asked: readsSaid(round.Asked), Lines: round.Lines, Refused: round.Refused})
	}
	return out
}
