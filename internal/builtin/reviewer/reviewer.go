// Package reviewer holds the deterministic steps of the reviewer role
// (roles/reviewer, ADR-0020): the rules no judgement is needed for, the
// lenses put to the agent as parts of one question, each finding's quotes
// found again, whether it lies in the change or outside it, the judge's
// answers read, and the verdict. The role's scripts call them through
// `workline builtin reviewer pre|post`.
package reviewer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/release"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

const (
	exitBlock    = 1
	exitExternal = 3
	exitNothing  = 10
)

// Settings are the role's settings, as merged by the engine.
type Settings struct {
	Base            string   `json:"base"`                 // where a review on a machine starts from, when no range is given
	Ignore          []string `json:"ignore"`               // files that are not code: a change touching only these asks nobody
	Lenses          []string `json:"lenses"`               // the lenses, in the order a merge request takes them in turn
	LensesPerPush   int      `json:"lenses-per-push"`      // on a merge request: lenses a push gets
	CommitsPerPush  int      `json:"commits-per-push"`     // on a merge request, after ready: new commits the lenses that warn read a push, the oldest first; 0: every one (ADR-0050)
	FindingsMax     int      `json:"findings-max"`         // findings on the change reported a run; the rest counted
	IssuesMax       int      `json:"issues-max"`           // issues opened a run for what lies outside the change
	DiffLinesMax    int      `json:"diff-lines-max"`       // lines of the change's hunks one call is given; past them, the change is read in parts (#446)
	PartsMax        int      `json:"parts-max"`            // parts a change too large for one call may be read in; past them, no lens asked (#446)
	CodeLinesMax    int      `json:"code-lines-max"`       // lines of the files changed given to a lens, all together
	CommentBlockMax int      `json:"comment-block-max"`    // lines of one comment added
	StoryWords      []string `json:"story-words"`          // what tells a bug's story in a comment
	AIFindings      Gate     `json:"ai-findings"`          // warn, until measured (#90); block: a verified important finding blocks; or one of the two by lens
	JudgeAtLeast    string   `json:"judge-at-least"`       // the independence a verification needs: context, model, provider
	ForgeWrites     bool     `json:"forge-writes"`         // on a merge request: the comments on lines, the summary comment and the issues
	FinderFloor     bool     `json:"finder-floor"`         // ask each lens to look for a number of candidates before it stops
	Tests           []string `json:"tests"`                // the test files: what a judge reading tests is shown of them
	TestsLinesMax   int      `json:"tests-lines-max"`      // lines of those tests a judge is shown, all together
	JudgeLinesMax   int      `json:"judge-lines-max"`      // lines of code a judge is shown: the cause's function, then those it reaches; 0: 31 lines around the cause
	JudgeReadRounds int      `json:"judge-read-rounds"`    // rounds of reads a judge of the code may ask before it answers (ADR-0045); 0: it reads none
	JudgeReadMax    int      `json:"judge-read-lines-max"` // lines read for one judge, all its rounds together
	LensesTogether  bool     `json:"lenses-together"`      // the lenses of a run asked in one call, the change given once
	AIMaxTokens     int      `json:"ai-max-tokens"`        // what a run may spend, all calls together; the engine stops asking
	IssueLinesMax   int      `json:"issue-lines-max"`      // lines of the issues the change closes the lenses are given, all together
	TestimonyLines  int      `json:"testimony-lines-max"`  // lines of what the author says (commit messages, the merge request) the lenses are given
	QuestionsMax    int      `json:"questions-max"`        // decisions put to a person a run; the rest counted
	DiffAlone       bool     `json:"diff-alone"`           // ask the diff-alone facet too, apart, given the change only
	SpecLenses      []string `json:"spec-lenses"`          // the lenses a spec is read through (#128)
	SpecLinesMax    int      `json:"spec-lines-max"`       // lines of a spec the lenses are given; the rest cut, said
	SpecRounds      int      `json:"spec-rounds"`          // on the forge: reviews in a row with a finding open, then a person (#128)
	SpecsPerRun     int      `json:"specs-per-run"`        // on the forge: the specs a run reads; the rest wait for the next (#433)
}

// alone is the facet given the change only, apart from the lenses (#126):
// what the change says by itself, which a reader holding the files and
// the author's words explains away. Its text is lenses/diff-alone.md; its
// findings are a lens's, quoted, judged and routed alike.
const alone = "diff-alone"

// Finding is one defect a lens reported, once its quotes were found again.
type Finding struct {
	Lens     string    `json:"lens"`
	Severity string    `json:"severity"` // important or nit
	Title    string    `json:"title"`
	Why      string    `json:"why"`
	Fix      string    `json:"fix,omitempty"`
	Seen     string    `json:"seen,omitempty"`     // what a person sees when it fails, in plain words: an issue opens with it
	Claim    string    `json:"claim,omitempty"`    // the author's words it contradicts, found again in what they said (#126)
	Decision string    `json:"decision,omitempty"` // not a defect: the question put to a person, never judged nor blocking (#126)
	Cause    Quote     `json:"cause"`
	Symptom  *Quote    `json:"symptom,omitempty"`
	Where    string    `json:"where"`          // the cause's file and line
	On       int       `json:"on,omitempty"`   // the change's: the line of its quote the change touched, where its comment goes on the forge (#277)
	Related  bool      `json:"related"`        // the cause lies in the change: the author fixes it
	Side     string    `json:"side,omitempty"` // the change's: where its cause was found, onHead or onRemoved (#570)
	Line     string    `json:"line,omitempty"` // outside the change: the line the cause starts at, as it reads, which keys its issue
	Also     []Finding `json:"also,omitempty"` // other findings on the same line, grouped under this one, each judged by its own lens
	Verified string    `json:"verified,omitempty"`
	Level    string    `json:"independence,omitempty"`
}

// Quote is a place a finding quotes.
type Quote struct {
	Path  string `json:"path" yaml:"path"`
	Quote string `json:"quote" yaml:"quote"`
}

// state is what pre found on its first pass, for the passes after the
// lenses and the judge answered.
type state struct {
	Head       string             `json:"head"`
	From       string             `json:"from"`             // where the commits not reviewed yet start
	Before     string             `json:"before,omitempty"` // where the diff a lens reads starts, when not From: From with the base merged in (#134)
	Base       string             `json:"base"`
	Change     Change             `json:"change"` // the whole range's
	Mechanical []verdict.Finding  `json:"mechanical"`
	Commits    []string           `json:"commits"`
	Patches    []string           `json:"patches,omitempty"` // the patch ids of the commits' changes, recorded with them (#286)
	Lenses     []string           `json:"lenses"`
	Together   bool               `json:"together,omitempty"` // the lenses asked in one call (lenses-together)
	Parts      int                `json:"parts,omitempty"`    // the parts the change is read in, too large for one call (#446); 0: whole
	Alone      bool               `json:"alone,omitempty"`    // the diff-alone facet asked too, apart
	Record     Record             `json:"record"`
	Every      bool               `json:"every,omitempty"` // every lens asked on the whole change, the run on ready (#129, #359)
	Advisories []verdict.Finding  `json:"advisories,omitempty"`
	Issues     []ClosedIssue      `json:"issues,omitempty"`    // the issues the change says it closes, as read (#126)
	Testimony  string             `json:"testimony,omitempty"` // what the author says of it, as the lenses were given it
	Skipped    []string           `json:"skipped,omitempty"`   // lenses not asked, each with why
	Spec       *Spec              `json:"spec,omitempty"`      // the spec reviewed, when the subject is a spec (#128)
	Part       string             `json:"part,omitempty"`      // one spec of several a run reads: what its parts' names start with (#433)
	Specs      []state            `json:"specs,omitempty"`     // on the forge: the specs the run reads, each reviewed as if alone (#433)
	Stops      []intent.Intention `json:"stops,omitempty"`     // on the forge: the comments putting to a person the specs whose rounds are spent (#128)
	Stopped    []string           `json:"stopped,omitempty"`   // what the run's summary says of each of those
	Waits      []string           `json:"waits,omitempty"`     // on the forge: the specs left for the next run, past specs-per-run (#433)
	Standing   []Block            `json:"standing,omitempty"`  // the last review's findings that block, still standing (#326)
	Read       string             `json:"read,omitempty"`      // the record as read: the apply merges the comment it finds with it (#375)
	Warn       string             `json:"warn,omitempty"`      // apart: the verdict's run, the lenses that block; only: the run apart, those that warn (ADR-0050)
	Upto       string             `json:"upto,omitempty"`      // the newest commit the lenses read, past commits-per-push new commits: the rest left (ADR-0050, step 6); "": the head
	Fresh      int                `json:"fresh,omitempty"`     // with Upto: the new commits of the push
	Left       []string           `json:"left,omitempty"`      // with Upto: the new commits left for the next run, oldest first
}

// candidates is what the lenses answered, its quotes checked, before the judge.
type candidates struct {
	Related, Outside []Finding
	Questions        []Finding         // decisions for a person: never judged, never an issue (#126)
	Logged           []verdict.Finding // dropped, each said
	Failed           []string          // lenses with no answer that reads
	Asked            []string          // the questions put to the judge, by finding
	// By JudgeKey: what each important finding is judged on, its patch and
	// its key (Answered); and, judged before on the same, the verdict kept,
	// its judge not asked again (ADR-0050, step 6).
	Same   map[string]Answered `json:"same,omitempty"`
	Reused map[string]Answered `json:"reused,omitempty"`
}

// Review is what the run found, as post reads it and out/review.json gives
// it to the author's agent.
type Review struct {
	Subject   string            `json:"subject,omitempty"` // code, or spec (#128)
	Spec      string            `json:"spec,omitempty"`    // the spec reviewed: its file, or its issue (`#4`)
	Status    string            `json:"status"`
	Summary   string            `json:"summary"`
	Findings  []verdict.Finding `json:"findings"`
	Change    []Finding         `json:"change"`              // on the change: the author fixes them
	Outside   []Finding         `json:"outside"`             // outside it: an issue each
	Questions []Finding         `json:"questions,omitempty"` // decisions put to a person: not defects, not judged (#126)
	Lenses    []string          `json:"lenses,omitempty"`
	Parts     int               `json:"parts,omitempty"` // the parts the change was read in, too large for one call (#446)
	Complete  bool              `json:"complete"`        // every lens answered: the commits are recorded as reviewed
	Record    Record            `json:"record"`
	Local     bool              `json:"local"`            // the record is kept on this machine
	Calls     []Spent           `json:"calls,omitempty"`  // what each call to an agent used
	Reads     []JudgeRead       `json:"reads,omitempty"`  // what each judge read, a round each (ADR-0045)
	Warn      string            `json:"warn,omitempty"`   // apart: the verdict's run; only: the run apart, the lenses that warn (ADR-0050)
	Reused    int               `json:"reused,omitempty"` // findings judged before on the same patch, their verdicts kept (ADR-0050, step 6)
}

// JudgeRead is one round of reads a judge asked (ADR-0045): what it asked,
// how many lines it was given, what was refused, with why. Its call is in
// Calls, by the same `for`.
type JudgeRead struct {
	For     string   `json:"for"` // the judge's question (judge/03)
	Where   string   `json:"where"`
	Lens    string   `json:"lens"`
	Round   int      `json:"round"`
	Asked   []string `json:"asked"`
	Lines   int      `json:"lines"`
	Refused []string `json:"refused,omitempty"`
}

// Spent is what one call of the review used: a lens's (or the lenses',
// asked together), or a judge's on one finding.
type Spent struct {
	For       string `json:"for"`             // the part (1-lenses, 2-edge-cases) or the judge's question (judge/03)
	Lenses    string `json:"lenses"`          // the lenses it was for
	Where     string `json:"where,omitempty"` // a judge's: the finding's place
	Model     string `json:"model,omitempty"`
	TokensIn  int    `json:"tokens-in"`
	TokensOut int    `json:"tokens-out"`
}

// Pre prepares the review, in up to three passes: the rules and the lenses'
// questions; the lenses' answers read and the judge's questions written;
// the judge's answers read and what follows proposed. Before the last, a
// pass a round serves the reads judges asked, and asks them again
// (ADR-0045).
func Pre(runDir, repo string) int {
	s, err := settings(runDir)
	if err != nil {
		return fail(err)
	}
	switch {
	case os.Getenv("WORKLINE_JUDGED") == "answered":
		// A judge that asked to read is served, and asked again, before
		// anything is settled (ADR-0045).
		if again, err := readsAsked(runDir, repo, s); err != nil || again {
			if err != nil {
				return fail(err)
			}
			return 0
		}
		return settle(runDir, repo, s)
	case os.Getenv("WORKLINE_PARTS") == "answered":
		return readLenses(runDir, repo, s)
	case os.Getenv("WORKLINE_EVENT") == specEvent, os.Getenv("WORKLINE_EVENT") == gardening:
		return prepareSpec(runDir, repo, s)
	}
	return prepare(runDir, repo, s)
}

func prepare(runDir, repo string, s Settings) int {
	// The verdict's run asks the lenses that block, the run apart those that
	// warn (ADR-0050): each keeps its own record, so that neither waits for
	// the other nor loses what the other did not read.
	side := input(runDir, "warn")
	if side != "" && side != warnApart && side != warnOnly {
		return fail(fmt.Errorf("input warn %q: %s, the verdict's run, or %s, the run apart", side, warnApart, warnOnly))
	}
	if ev := os.Getenv("WORKLINE_EVENT"); side != "" && ev != "merge-request" {
		// Its role.yaml's `apart`: elsewhere nothing reads what it would leave.
		return fail(fmt.Errorf("input warn %q on %s: the lenses that warn read apart on a merge request only", side, ev))
	}
	if side == warnOnly && !s.warns() {
		return final(runDir, Review{Status: verdict.Pass, Summary: "every lens blocks (ai-findings): nothing reads apart from the verdict", Warn: side})
	}
	rng := input(runDir, "range")
	if rng == "" {
		rng = s.Base + "..HEAD"
	}
	head, err := git(repo, "rev-parse", "--verify", gitrange.Head(rng)+"^{commit}")
	if err != nil {
		return fail(fmt.Errorf("the range %s: %v", rng, err))
	}
	head = strings.TrimSpace(head)
	base := head
	if b := gitrange.Base(rng); b != "" {
		mb, err := git(repo, "merge-base", b, head)
		if err != nil {
			return fail(fmt.Errorf("the range %s: no common commit with %s: %v", rng, b, err))
		}
		base = strings.TrimSpace(mb)
	}
	files, err := codeFiles(repo, s, base, head)
	if err != nil {
		return fail(err)
	}
	if len(files) == 0 {
		return final(runDir, Review{Status: verdict.Pass, Summary: "no code changed (" + notCode() + "): nobody asked"})
	}
	change, err := readChange(repo, base, head, files)
	if err != nil {
		return fail(err)
	}
	codes, err := committerSettings(repo)
	if err != nil {
		return fail(err)
	}
	mech, err := mechanical(change, s, codes)
	if err != nil {
		return fail(err)
	}
	// The rules at the levels the project set them to (`enforce`), before
	// they decide anything: one set to warn is shown, never holds the
	// lenses; one set to off is not said (#550).
	enforce, err := enforcement(repo)
	if err != nil {
		return fail(err)
	}
	mech = verdict.Levels(mech, enforce)
	// The rules block on the verdict's run, which reports them; the run
	// apart waits for them as it does, saying nothing of them again.
	rulesHold := status(mech) == verdict.Block
	if side == warnOnly {
		mech = nil
	}
	rec, where, err := loadRecord(repo, s, side)
	recordRead := err == nil
	var advisories []verdict.Finding
	if err != nil {
		// Never read as none in silence (#302): the range is reviewed whole,
		// once, and a whole review writes the record anew.
		advisories = append(advisories, verdict.Finding{Rule: "record-unread", Level: "warn",
			Message: "the record of what was reviewed before, " + where + ", could not be read (" + err.Error() + "): every commit of the range is reviewed again, and a whole review writes it anew"})
		rec = Record{Judged: rec.Judged, Gone: rec.Gone} // the findings judged read apart: kept (ADR-0046)
	}
	read := rec.Read() // before a draft's run drops its marks (runOnReady)
	sp, err := rec.since(repo, base, head)
	if err != nil {
		return fail(err)
	}
	if sp.Unread != nil {
		advisories = append(advisories, verdict.Finding{Rule: "patch-ids-unread", Level: "warn",
			Message: "the patch ids of the commits' changes could not be read (" + sp.Unread.Error() + "): a commit rebased is reviewed again"})
	}
	start := sp.From
	marked := rec.ReadAll || rec.Retried || len(rec.Failed) > 0 // what a draft's run forgets (runOnReady)
	// The lenses owing commits are this run's: the other run's owe them to
	// its own record (ADR-0050).
	rec.Failed = slices.DeleteFunc(rec.Failed, func(l string) bool { return !s.ofSide(side, l) })
	ready := runOnReady(runDir, &rec, recordRead)
	if ready {
		start = base
	}
	st := state{Head: head, From: start, Base: base, Change: change, Mechanical: mech, Commits: sp.Commits, Patches: sp.Patches, Record: rec, Every: ready, Advisories: advisories, Read: read, Warn: side}
	// What blocked the last review stands until a commit answers it (#326):
	// a run with nothing new to read never turns it green.
	st.Standing = standing(repo, s, st, rec.Blocks)
	if start == head {
		return held(runDir, repo, s, st, review(st, nil, fmt.Sprintf("the %d %s reviewed already: only the rules ran", len(sp.Commits), plural(len(sp.Commits), "commit was", "commits were"))))
	}
	// The rules first: what they block on is fixed before any agent is
	// asked, and asked again after each push while it is not.
	if rulesHold {
		v := review(st, nil, "the rules found what the author must fix first; the review follows once they pass")
		v.Record.Owed = v.Record.Owed || ready
		return held(runDir, repo, s, st, v)
	}
	// A lens reads what the commits not reviewed yet change, not every file
	// of the merge request, nor the base's code they merged in; when they
	// change no code, nobody is asked.
	if start != base {
		if st.Before, err = mergedIn(repo, base, start); err != nil {
			st.Advisories = append(st.Advisories, verdict.Finding{Rule: "base-not-set-apart", Level: "warn",
				Message: "the base's code the new commits merged in could not be set apart (" + err.Error() + "): the lens is given it too"})
		}
		if files, err = codeFiles(repo, s, st.lensFrom(), head); err != nil {
			return fail(err)
		}
		if len(files) == 0 {
			return recordOnly(runDir, s, st, sp.New)
		}
	}
	if os.Getenv("WORKLINE_AI_WAITS") == "ready" {
		return waitsForReady(runDir, s, st, marked)
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "not-reviewed", Level: "warn",
			Message: "no agent: only the rules that need no judgement ran; the change waits for a person's review"})
		return final(runDir, review(st, nil, "no agent: only the rules ran"))
	}
	// What the change is for, and what its author says of it (#126): the
	// merge request's text and the commit messages name the issues it
	// closes; those are read from the forge.
	mr, err := mergeRequestText(repo)
	if err != nil {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "merge-request-unread", Level: "warn",
			Message: "the merge request's title and body could not be read (" + err.Error() + "): the lenses are given the commit messages only"})
	}
	msgs, err := git(repo, "log", "--format=%B", base+".."+head)
	if err != nil {
		return fail(err)
	}
	var ids []int
	for _, name := range s.Lenses {
		if l, err := readLens(repo, name); err == nil && l.Needs == "issue" { // one failing to read stops the run below
			ids = closes(mr, msgs)
			break
		}
	}
	var unread []verdict.Finding
	st.Issues, unread = readIssues(repo, ids, s.IssueLinesMax)
	st.Advisories = append(st.Advisories, unread...)
	skip, err := unasked(repo, s.Lenses, st.Issues, ids)
	if err != nil {
		return fail(err)
	}
	for _, l := range s.Lenses {
		if why, ok := skip[l]; ok {
			st.Skipped = append(st.Skipped, fmt.Sprintf("%s (%s)", l, why))
		}
	}
	st.Lenses, st.Alone, err = lenses(runDir, s, rec, skip, ready)
	if err != nil {
		return fail(err)
	}
	if len(st.Lenses) == 0 && !st.Alone {
		return final(runDir, review(st, nil, "no lens asked can review this change: not asked: "+strings.Join(st.Skipped, ", ")))
	}
	// Of the lenses this push takes, in turn or every one on ready, each
	// run asks those of its side (ADR-0050); the other's are its to ask.
	// The commits a lens of this side left past commits-per-push are its
	// to read, on the other's turn too (#591).
	if other := st.side(s); len(st.Lenses) == 0 && !st.Alone {
		if st.Lenses, st.Alone = rec.leftTo(s, side, skip); len(st.Lenses) == 0 && !st.Alone {
			return otherSide(runDir, repo, s, st, other, sp.New)
		}
	}
	// After ready, the lenses that warn read at most commits-per-push new
	// commits a push, the oldest first; the rest wait, said (ADR-0050).
	if err := capCommits(repo, s, &st, sp); err != nil {
		return fail(err)
	}
	if st.Upto != "" {
		if files, err = codeFiles(repo, s, st.lensFrom(), st.Upto); err != nil {
			return fail(err)
		}
		if len(files) == 0 {
			return recordOnly(runDir, s, st, st.Fresh-len(st.Left))
		}
	}
	if st.Testimony, err = testimony(repo, st, mr, s.TestimonyLines); err != nil {
		return fail(err)
	}
	st.Together = s.LensesTogether && len(st.Lenses) > 1
	why, err := ask(runDir, repo, s, &st, files)
	if err != nil {
		return fail(err)
	}
	if why != "" {
		return tooLarge(runDir, repo, s, st, why)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review-state.json"), st); err != nil {
		return fail(err)
	}
	return 0
}

// codeFiles is the code the commits from..head change: the files its
// `ignore` does not leave, by their path at head. A file moved whole has
// no line to read, and is left too (#358); on a release tool's merge
// request, a file whose every line changed is a version the tool bumps,
// on a line it marks (#384).
func codeFiles(repo string, s Settings, from, head string) ([]File, error) {
	all, err := changedFiles(repo, from, head)
	if err != nil {
		return nil, err
	}
	var files []File
	for _, f := range all {
		if f.Same || pathglob.Any(s.Ignore, f.Path) {
			continue
		}
		bump, err := releaseBump(repo, from, head, f)
		if err != nil {
			return nil, err
		}
		if !bump {
			files = append(files, f)
		}
	}
	return files, nil
}

// releaseBump says whether, on a release tool's merge request (the
// engine's WORKLINE_RELEASE_BRANCH, ADR-0017), from..head only bumps
// versions in f as the tool does, by its own marks (release.OnlyBumps).
func releaseBump(repo, from, head string, f File) (bool, error) {
	if os.Getenv("WORKLINE_RELEASE_BRANCH") == "" || f.Was != "" {
		return false, nil
	}
	diff, err := git(repo, "diff", "-U0", "--no-color", "--no-ext-diff", from, head, "--", f.Path)
	if err != nil {
		return false, err
	}
	return release.OnlyBumps(f.Path, diff, func() (string, error) { return git(repo, "show", head+":"+f.Path) }), nil
}

// notCode is what a change that asks nobody holds.
func notCode() string {
	if os.Getenv("WORKLINE_RELEASE_BRANCH") != "" {
		return "only files moved unchanged, files the reviewer leaves by its `ignore`, or the versions a release tool bumps"
	}
	return "only files moved unchanged, or files the reviewer leaves by its `ignore`"
}

// mergedIn is where a lens's diff starts when the commits after from, the
// newest reviewed, merged the base in (#134): from with the base merged in
// by git, in memory (`git merge-tree --write-tree`), a conflict's markers
// left in it. Against the head it shows the branch's own commits and how
// it settled a conflict, both sides, and none of the base's code; a tree,
// read as a commit is, whatever commits came with the merge (ADR-0013).
// "" when the base is in from already: from is where.
func mergedIn(repo, base, from string) (string, error) {
	if _, err := git(repo, "merge-base", "--is-ancestor", base, from); err == nil {
		return "", nil
	}
	cmd := exec.Command("git", "-C", repo, "merge-tree", "--write-tree", "--no-messages", from, base)
	var errOut strings.Builder
	cmd.Stderr = &errOut
	out, err := cmd.Output()
	var exit *exec.ExitError
	if err != nil && (!errors.As(err, &exit) || exit.ExitCode() != 1) { // 1: a conflict, its markers in the tree
		return "", fmt.Errorf("git merge-tree: %v: %s", err, strings.TrimSpace(errOut.String()))
	}
	tree, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(tree), nil
}

// The runs a merge request's review is split in (input `warn`, ADR-0050):
// the verdict's, asking the lenses that block, and the run apart, asking
// those that warn, its summary in a comment of its own (WarnKey).
const (
	warnApart = "apart"
	warnOnly  = "only"
)

// side keeps of the lenses the run takes those of its side: the lenses
// that block (`ai-findings`) on the verdict's run, those that warn on the
// run apart, the diff-alone facet by its own; and returns the others, the
// other run's. A run not split keeps them all.
func (st *state) side(s Settings) []string {
	if st.Warn == "" {
		return nil
	}
	mine := func(lens string) bool { return s.ofSide(st.Warn, lens) }
	var kept, other []string
	for _, l := range st.Lenses {
		if mine(l) {
			kept = append(kept, l)
		} else {
			other = append(other, l)
		}
	}
	st.Lenses = kept
	if st.Alone && !mine(alone) {
		st.Alone, other = false, append(other, alone)
	}
	return other
}

// ofSide says whether a lens, or the diff-alone facet, is a run's to ask:
// one that blocks on the verdict's run, one that warns on the run apart,
// any on a run not split.
func (s Settings) ofSide(warn, lens string) bool {
	return warn == "" || s.AIFindings.Blocks(lens) == (warn == warnApart)
}

// warns says whether a lens of the project only warns, or the diff-alone
// facet, when on: what the run apart has to read at all.
func (s Settings) warns() bool {
	for _, l := range s.Lenses {
		if !s.AIFindings.Blocks(l) {
			return true
		}
	}
	return s.DiffAlone && !s.AIFindings.Blocks(alone)
}

// otherSide settles a run whose push takes only the other run's lenses
// (ADR-0050): nobody asked here, the commits recorded and the turn moved
// on, as the other run moves its own; on ready, every lens of this side
// read, none being asked. The summary comment says where they read. A
// lens of this side owing them is asked instead, never passed over
// (lenses, #588): a commit recorded here is one none of them had to read.
func otherSide(runDir, repo string, s Settings, st state, other []string, fresh int) int {
	why := fmt.Sprintf("no lens that blocks on this push: %s, which only %s, %s apart from the verdict, in %s own comment", strings.Join(other, ", "),
		plural(len(other), "warns", "warn"), plural(len(other), "reads", "read"), plural(len(other), "its", "their"))
	if st.Warn == warnOnly {
		why = fmt.Sprintf("no lens that warns on this push: %s, which %s, %s it on the verdict's run", strings.Join(other, ", "),
			plural(len(other), "blocks", "block"), plural(len(other), "reads", "read"))
	}
	v := review(st, nil, fmt.Sprintf("%s; the %d new %s recorded", why, fresh, plural(fresh, "commit", "commits")))
	v.Complete = true
	v.Record = st.Record.add(st.Commits, st.Patches)
	v.Record.Blocks, v.Record.Owed = st.Standing, false
	everyLens(st, &v, true)
	var fallback []intent.Intention
	if mergeRequest() != nil && s.ForgeWrites {
		fallback = append(fallback, summaryIntention(repo, s, st, v, nil, 0))
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// lensFrom is where the diff a lens reads starts.
func (st state) lensFrom() string {
	if st.Before != "" {
		return st.Before
	}
	return st.From
}

// lensTo is where it ends: the head, or the newest commit within
// commits-per-push (ADR-0050). The quotes are still found at the head,
// where the comments go.
func (st state) lensTo() string {
	if st.Upto != "" {
		return st.Upto
	}
	return st.Head
}

// recordOnly settles a run whose new commits change no code: nobody asked,
// the commits recorded as reviewed, and the turn of the lenses left where
// it was, for the next push that changes code.
func recordOnly(runDir string, s Settings, st state, fresh int) int {
	why := notCode()
	if st.Before != "" {
		why = "only the base's code, merged in and reviewed there, files moved unchanged, or files the reviewer leaves"
	}
	v := review(st, nil, fmt.Sprintf("no code changed since the last review (%s): nobody asked, the %d new commits recorded%s", why, fresh, st.leftSaid()))
	v.Complete = true
	v.Record = st.recorded()
	v.Record.Runs, v.Record.Blocks = st.Record.Runs, st.Standing
	var fallback []intent.Intention
	if mergeRequest() != nil && s.ForgeWrites {
		fallback = append(fallback, intent.Intention{Kind: "comment", Value: summaryValue(st, summaryComment(v, 0, nil, nil))})
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// waitsForReady settles a draft's run, its agent set aside by the engine
// (WORKLINE_AI_WAITS, ADR-0050): the rules ran, no lens nor judge is
// asked, and the review says it waits for ready. No review is posted, the
// last one standing as it is; but where its record said every lens
// read the change (marked), the record alone is written anew, without
// it, so that ready again reads every lens once more (#359), even after a
// cancelled run. The comment is the one the record was read from; a run
// apart whose record a summary comment gave has none of its own to write.
func waitsForReady(runDir string, s Settings, st state, marked bool) int {
	why := "a draft: no lens asked; every lens reads the whole change once it is marked ready"
	if os.Getenv("WORKLINE_FORGE") == "gitlab" {
		why += "; GitLab starts no pipeline then: push, or run its pipeline"
	}
	st.Advisories = append(st.Advisories, verdict.Finding{Rule: verdict.WaitsForReady, Level: "warn", Message: why})
	v := review(st, nil, "a draft: only the rules ran; every lens reads the whole change once it is marked ready")
	body := ""
	if marked && mergeRequest() != nil && s.ForgeWrites && st.Record.comment != "" {
		if body = keptReview(st.Record.comment, st.Warn, v.Record); body == "" {
			// Never kept in silence (principle 12): ready again would read
			// the lens in turn, not every lens.
			v.Findings = append(v.Findings, verdict.Finding{Rule: "record-not-written", Level: "warn",
				Message: "the record still says every lens read the change, and could not be written anew without it: once it is ready, ask every lens (`--input lenses=all`)"})
		}
	}
	if body == "" {
		return final(runDir, v)
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), []intent.Intention{{Kind: "comment", Value: summaryValue(st, body)}}); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// runOnReady says whether this run is the one on ready, every lens once on
// the whole change (#129): told `lenses=all`; or, told no lenses, the
// record owing them, or the merge request no draft (`draft=false`) and its
// record, read, not saying every lens read it (#359: a run on ready a push
// cancelled wrote nothing). A draft's run drops those marks (roles/reviewer/
// docs/ready.md).
func runOnReady(runDir string, rec *Record, recordRead bool) bool {
	if os.Getenv("WORKLINE_EVENT") != "merge-request" {
		return false
	}
	draft := input(runDir, "draft")
	if draft == "true" {
		rec.ReadAll, rec.Retried, rec.Failed, rec.LeftBy = false, false, nil, nil
	}
	switch asked := input(runDir, "lenses"); asked {
	case "all":
		return true
	case "":
		return rec.Owed || rec.Retried || draft == "false" && recordRead && !rec.ReadAll
	}
	return false
}

// everyLens marks the record of the run on ready (#359): every lens read
// the change once each answered; one that did not leaves every lens owed
// to the next push, once. Failing again, said, only the lenses that did
// not answer are asked again, on the whole change, at each run until they
// do (lenses): the record never says every lens read it before (#588).
func everyLens(st state, v *Review, answered bool) {
	switch {
	case !st.Every:
		return
	case answered:
		v.Record.ReadAll, v.Record.Retried = true, false
		return
	case !st.Record.Retried && len(st.Record.Failed) == 0:
		v.Record.Retried = true
		v.Findings = append(v.Findings, verdict.Finding{Rule: "every-lens-again", Level: "warn",
			Message: "not every lens answered on the run on ready: the next push asks every lens again, on the whole change, once"})
		return
	}
	v.Record.Retried = false
	n := len(v.Record.Failed)
	v.Findings = append(v.Findings, verdict.Finding{Rule: "every-lens-not-whole", Level: "warn",
		Message: fmt.Sprintf("not every lens answered, the run on ready asked twice: the change was not reviewed whole by every lens; %s that did not answer, %s, %s again on the whole change at the next push, or when the run runs again",
			plural(n, "the lens", "the lenses"), strings.Join(v.Record.Failed, ", "), plural(n, "is asked", "are asked"))})
}

// lenses are the lenses this run asks: every one on a machine, or on the
// run on ready (every, runOnReady); on a merge request, the next in turn,
// or those the run is told (`lenses=`); before either, those owing the
// commits after the record (#588). A lens
// in skip, which has nothing to read in this change, is never asked. With
// them, the diff-alone facet, when its setting is on; a run told which
// lenses to ask asks it only when it names it, whatever its setting.
func lenses(runDir string, s Settings, rec Record, skip map[string]string, every bool) ([]string, bool, error) {
	if len(s.Lenses) == 0 {
		return nil, false, fmt.Errorf("settings: no lenses")
	}
	var usable []string
	for _, l := range s.Lenses {
		if _, ok := skip[l]; !ok {
			usable = append(usable, l)
		}
	}
	asked := input(runDir, "lenses")
	switch {
	case asked == "all":
		return usable, s.DiffAlone, nil
	case asked != "":
		var out []string
		named := false
		for _, l := range strings.Split(asked, ",") {
			l = strings.TrimSpace(l)
			if l == alone {
				named = true
				continue
			}
			if !slices.Contains(s.Lenses, l) {
				return nil, false, fmt.Errorf("lens %q: not one of %s, %s", l, strings.Join(s.Lenses, ", "), alone)
			}
			if slices.Contains(usable, l) && !slices.Contains(out, l) {
				out = append(out, l)
			}
		}
		return out, named, nil
	case os.Getenv("WORKLINE_EVENT") != "merge-request":
		return usable, s.DiffAlone, nil
	}
	// The lenses that did not answer read their commits before the turn
	// moves on, whosever turn it is (#588); on the run on ready past its
	// retry, they alone: the others read the whole change already.
	var owed []string
	for _, l := range rec.Failed {
		if l == alone || slices.Contains(usable, l) {
			owed = append(owed, l)
		}
	}
	if len(owed) > 0 && !rec.Retried && !rec.Owed {
		facet := slices.Contains(owed, alone) || !every && s.DiffAlone
		return slices.DeleteFunc(owed, func(l string) bool { return l == alone }), facet, nil
	}
	switch {
	case every: // the run on ready (#129, #359)
		return usable, s.DiffAlone, nil
	case len(usable) == 0:
		return nil, s.DiffAlone, nil
	}
	// In turn over every lens, one with nothing to read passed over: the
	// same run count names the same place in the turn whatever this
	// change closes.
	n := max(1, min(s.LensesPerPush, len(usable)))
	var out []string
	for i := 0; len(out) < n && i < len(s.Lenses); i++ { // a lens named twice counts once
		if l := s.Lenses[(rec.Runs+i)%len(s.Lenses)]; slices.Contains(usable, l) && !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	return out, s.DiffAlone, nil
}

// unasked are the lenses this change gives nothing to read, each with why:
// one needing the issue the change closes, when it closes none it could
// read (#126).
func unasked(repo string, names []string, issues []ClosedIssue, ids []int) (map[string]string, error) {
	skip := map[string]string{}
	for _, name := range names {
		l, err := readLens(repo, name)
		if err != nil {
			return nil, err
		}
		if l.Subject == "spec" {
			return nil, fmt.Errorf("lens %q reads a spec: it belongs under spec-lenses, not lenses", name)
		}
		if l.Needs == "issue" && len(issues) == 0 {
			skip[name] = "the change closes no issue"
			if len(ids) > 0 {
				skip[name] = "no issue it closes could be read"
			}
		}
	}
	return skip, nil
}

// ask writes the lenses' parts: the lens, the commits not reviewed yet,
// and the files they change. A change past one call's context budget or
// diff-lines-max is read in parts, a share of its files each, st.Parts
// saying how many (#446); one that would take more than parts-max writes
// none, and ask says why.
func ask(runDir, repo string, s Settings, st *state, files []File) (string, error) {
	// Each file once (#147): whole, the change marked in it, while the
	// files fit in code-lines-max; the others by the change's hunks. A file
	// given whole and again in the diff was a third of a lens's call.
	var units []unit
	var hunked []File
	left := s.CodeLinesMax
	for _, f := range files {
		body, ok := marked(repo, st.lensFrom(), st.lensTo(), f)
		n := strings.Count(body, "\n")
		if !ok || n > left {
			hunked = append(hunked, f)
			continue
		}
		left -= n
		units = append(units, unit{path: f.Path, whole: true, label: f.Path, text: body})
	}
	if len(hunked) > 0 {
		diff, err := git(repo, append([]string{"diff", "-U5", "--no-color", "--no-ext-diff", "--find-renames", st.lensFrom(), st.lensTo(), "--"}, pathspec(hunked)...)...)
		if err != nil {
			return "", err
		}
		units = append(units, fileDiffs(diff)...)
	}
	floor := func(size int) string {
		if !s.FinderFloor { // a floor on candidates, never on what is shown; from the size of what is read
			return ""
		}
		each := ""
		if st.Together {
			each = " for each lens"
		}
		return fmt.Sprintf("\nLook for at least %d candidates%s before you stop; then give only those whose cause you can quote, important or nit as each deserves.\n", candidatesFor(size), each)
	}
	read := map[string]Lens{}
	for _, lens := range st.Lenses {
		l, err := readLens(repo, lens)
		if err != nil {
			return "", err
		}
		read[lens] = l
	}
	// The issues closed only when a lens asked reads them: on a merge
	// request, a push asking another lens does not pay for them.
	purpose := ""
	if slices.ContainsFunc(st.Lenses, func(l string) bool { return read[l].Needs == "issue" }) {
		purpose = whatFor(st.Issues)
	}
	write := func(name, task string) error {
		dir := filepath.Join(runDir, "in", "parts", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "task.md"), []byte(task), 0o644)
	}
	// The diff-alone facet: its own call, the change only, once the
	// lenses' parts are known to be asked.
	facet := func() error {
		if !st.Alone {
			return nil
		}
		task, err := aloneTask(repo, s, *st, files)
		if err != nil {
			return err
		}
		return write(partName(*st, alone), task)
	}
	// What each call asks before the change: one for the lenses together,
	// the change given once, their questions differing, what they read not
	// (#147); else one a lens.
	type call struct{ lens, head string }
	var calls []call
	if st.Together {
		var b strings.Builder
		fmt.Fprintf(&b, "# Lenses: %s\n\nEach lens below is a question of its own: ask each in turn, of the whole change, and name in each finding the lens that found it: `lens:` one of %s.\n",
			strings.Join(st.Lenses, ", "), strings.Join(st.Lenses, ", "))
		for _, lens := range st.Lenses {
			fmt.Fprintf(&b, "\n## Lens: %s\n\n%s\n", lens, strings.TrimSpace(read[lens].Text))
		}
		calls = []call{{st.Lenses[0], b.String()}}
	} else {
		for _, lens := range st.Lenses {
			calls = append(calls, call{lens, fmt.Sprintf("# Lens: %s\n\n%s\n", lens, strings.TrimSpace(read[lens].Text))})
		}
	}
	longest := 0
	for _, c := range calls {
		longest = max(longest, len(c.head))
	}
	put := func(share []unit, k, n int) error {
		text, size := lensMaterial(st.Testimony, purpose, share)
		note := ""
		if n > 1 {
			note = partNote(k, n, files)
		}
		for _, c := range calls {
			if err := write(sharePart(*st, c.lens, k), c.head+floor(size)+"\n"+note+text); err != nil {
				return err
			}
		}
		return nil
	}
	// The whole change in one call, while it fits the role's context
	// budget and diff-lines-max.
	room, err := taskRoom(runDir, repo)
	if err != nil {
		return "", err
	}
	text, size := lensMaterial(st.Testimony, purpose, units)
	diffLines := 0
	for _, u := range units {
		diffLines += u.lines()
	}
	if (room == 0 || longest+len(floor(size))+1+len(text) <= room) && (s.DiffLinesMax <= 0 || diffLines <= s.DiffLinesMax) {
		if err := put(units, 1, 1); err != nil {
			return "", err
		}
		return "", facet()
	}
	// Else in parts (ADR-0009's way): what every part holds first, then a
	// share of the files, in the order of their paths, a directory kept
	// together where it fits.
	sort.SliceStable(units, func(i, j int) bool { return units[i].path < units[j].path })
	skeleton, _ := lensMaterial(st.Testimony, purpose, nil)
	share := 0 // no limit but diff-lines-max: the role's budget is not known
	if room > 0 {
		share = room - longest - len(floor(math.MaxInt)) - 1 - len(partNote(99, 99, files)) - len(skeleton)
		if share < unitsRoomMin {
			return fmt.Sprintf("what every part holds before its share of the change — the lenses, what the author says, the issue it closes, every file it touches by name — leaves less than %d characters within the role's context budget: no lens asked; split the change into smaller ones, each reviewed apart", unitsRoomMin), nil
		}
	}
	parts := pack(units, share, s.DiffLinesMax)
	if len(parts) > s.PartsMax {
		return fmt.Sprintf("the change would take %d parts, each within the role's context budget and diff-lines-max, past parts-max (%d): no lens asked; split it into smaller changes, each reviewed apart", len(parts), s.PartsMax), nil
	}
	st.Parts = len(parts)
	for k, p := range parts {
		if err := put(p, k+1, len(parts)); err != nil {
			return "", err
		}
	}
	return "", facet()
}

// tooLarge settles a run whose change would take more parts than
// parts-max (#446): no lens asked, no token spent, the change not
// reviewed whole, said; its commits not recorded, the summary comment
// still written on a merge request, for its author.
func tooLarge(runDir, repo string, s Settings, st state, why string) int {
	v := review(st, []verdict.Finding{{Rule: "review-not-whole", Level: "warn", Message: why}}, "")
	v.Lenses = facets(st)
	v.Summary = "0 findings on the change; lenses: " + strings.Join(v.Lenses, ", ") + "; no lens asked, the change too large: not reviewed whole" + st.still()
	if len(st.Skipped) > 0 {
		v.Summary += "; not asked: " + strings.Join(st.Skipped, ", ")
	}
	var fallback []intent.Intention
	if mergeRequest() != nil && s.ForgeWrites {
		fallback = append(fallback, summaryIntention(repo, s, st, v, nil, 0))
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// aloneTask is the diff-alone facet's question (#126): its text and the
// change, as a diff with three lines around each hunk, up to
// diff-lines-max; nothing else, not the files it changes whole, not what
// the author says, not the issue it closes. Its own call: the lenses'
// shared material is what it must not hold.
func aloneTask(repo string, s Settings, st state, files []File) (string, error) {
	l, err := readLens(repo, alone)
	if err != nil {
		return "", err
	}
	diff, err := git(repo, append([]string{"diff", "-U3", "--no-color", "--no-ext-diff", "--find-renames", st.lensFrom(), st.lensTo(), "--"}, pathspec(files)...)...)
	if err != nil {
		return "", err
	}
	diff = capLines(diff, s.DiffLinesMax, "the change is cut here: review what is above")
	floor := ""
	if s.FinderFloor {
		floor = fmt.Sprintf("\nLook for at least %d candidates before you stop; then give only those whose cause you can quote, important or nit as each deserves.\n", candidatesFor(len(diff)))
	}
	return fmt.Sprintf("# Lens: %s\n\n%s\n%s\n## The change\n\n```diff\n%s```\n", alone, strings.TrimSpace(l.Text), floor, diff), nil
}

// candidatesFor is the finder floor for what a lens reads, from its size in
// bytes: one, and one more as the square root of its kilobytes grows, up
// to ten.
func candidatesFor(size int) int {
	return min(int(math.Floor(math.Sqrt(float64(size)/1024)+1)), 10)
}

// lensText is what a lens asks: the project's own (.workline/roles/reviewer/
// lenses/<lens>.md), else the role's.
func lensText(repo, lens string) (string, error) {
	for _, dir := range []string{filepath.Join(repo, ".workline", "roles", roleName()), filepath.Join(os.Getenv("WORKLINE_ROLES_DIR"), roleName())} {
		if data, err := os.ReadFile(filepath.Join(dir, "lenses", lens+".md")); err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("lens %q: no lenses/%s.md in the role", lens, lens)
}

// judgeQuestion is what the judge of an important finding is asked when its
// lens names no question of its own: whether the code fails.
const judgeQuestion = "Is this finding about the code true: does the code quoted, as it reads, fail the way the finding says? Answer no if it is not a defect, if the code shown handles it, or if the finding only guesses."

// Lens is a lens's file: what it asks, and, in its front matter, what the
// judge of its important findings is asked and shown besides the finding,
// the code around it and the change (#223).
type Lens struct {
	Text    string `yaml:"-"`
	Subject string `yaml:"subject"` // spec: a lens of a spec, under spec-lenses (#128); empty, of code
	Needs   string `yaml:"needs"`   // issue: asked only of a change closing an issue, given it (#126); code: of a spec naming code, given it (#128)
	Cites   string `yaml:"cites"`   // claim: each finding quotes a claim of the author, found again in what they said, or is dropped
	Judge   struct {
		Question string `yaml:"question"` // empty: judgeQuestion
		Reads    string `yaml:"reads"`    // tests: the tests that touch the cause's file; issue: the issues the change closes; code: the code a spec's finding stands on
	} `yaml:"judge"`
}

// readLens reads a lens's file, its front matter apart from its text.
func readLens(repo, name string) (Lens, error) {
	var l Lens
	text, err := lensText(repo, name)
	if err != nil {
		return l, err
	}
	text = strings.ReplaceAll(text, "\r\n", "\n") // a checkout with CRLF endings
	l.Text = text
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		head, body, closed := strings.Cut("\n"+rest+"\n", "\n---\n") // empty, or closed on the last line
		if !closed {
			return l, fmt.Errorf("lens %q: its front matter is not closed by a line `---`", name)
		}
		if err := yaml.Unmarshal([]byte(head), &l); err != nil {
			return l, fmt.Errorf("lens %q: its front matter: %w", name, err)
		}
		l.Text = body
	}
	if r := l.Judge.Reads; r != "" && r != "tests" && r != "issue" && r != "code" {
		return l, fmt.Errorf("lens %q: judge.reads %q: only `tests`, `issue` and `code` are known", name, r)
	}
	if l.Needs != "" && l.Needs != "issue" && l.Needs != "code" {
		return l, fmt.Errorf("lens %q: needs %q: only `issue` and `code` are known", name, l.Needs)
	}
	if l.Subject != "" && l.Subject != "spec" {
		return l, fmt.Errorf("lens %q: subject %q: only `spec` is known", name, l.Subject)
	}
	if l.Cites != "" && l.Cites != "claim" {
		return l, fmt.Errorf("lens %q: cites %q: only `claim` is known", name, l.Cites)
	}
	return l, nil
}

// readLenses reads what each lens answered, finds each finding's quotes
// again, tells the findings on the change from those outside it, caps them,
// and puts each important one to the judge. A run reading several specs
// does so for each, its judges numbered after the spec's before (#433).
func readLenses(runDir, repo string, s Settings) int {
	var st state
	if err := readJSON(filepath.Join(runDir, "in", "review-state.json"), &st); err != nil {
		return fail(err)
	}
	subs := []state{st}
	if len(st.Specs) > 0 {
		subs = st.Specs
	}
	cs := make([]candidates, len(subs))
	unavailable, asked := 0, 0
	for i, sub := range subs {
		c, un, err := lensAnswers(runDir, repo, s, sub)
		if err != nil {
			return fail(err)
		}
		cs[i], unavailable, asked = c, unavailable+un, asked+len(facets(sub))
	}
	if unavailable > 0 && unavailable == asked {
		v := review(st, nil, "the agent could not be reached: only the rules ran")
		if len(st.Specs) > 0 {
			v.Subject = "spec"
			waitingQuestions(&v)
		}
		v.Status = verdict.BlockedExternal
		if code := final(runDir, v); code != exitNothing {
			return code
		}
		return exitExternal
	}
	n := 0
	for i, sub := range subs {
		if err := askJudges(runDir, repo, s, sub, &cs[i], &n); err != nil {
			return fail(err)
		}
		if err := writeJSON(filepath.Join(runDir, "in", candidatesFile(sub)), cs[i]); err != nil {
			return fail(err)
		}
	}
	if n > 0 {
		return 0 // the judge first
	}
	return settle(runDir, repo, s)
}

// candidatesFile is where what the lenses answered is kept for the passes
// after the judge: one file a spec when a run reads several (#433).
func candidatesFile(st state) string {
	if st.Part != "" {
		return "candidates-" + st.Part + ".json"
	}
	return "candidates.json"
}

// lensReader reads a lens once, then from what it read.
func lensReader(repo string) func(string) (Lens, error) {
	read := map[string]Lens{}
	return func(name string) (Lens, error) {
		l, ok := read[name]
		if !ok {
			var err error
			if l, err = readLens(repo, name); err != nil {
				return l, err
			}
			read[name] = l
		}
		return l, nil
	}
}

// lensAnswers reads what each lens of a review answered, finds each
// finding's quotes again, tells the findings on the change from those
// outside it, and caps them; unavailable counts the lenses the agent could
// not be reached for.
func lensAnswers(runDir, repo string, s Settings, st state) (candidates, int, error) {
	var c candidates
	lensOf := lensReader(repo)
	unavailable := 0
	var asked [][]string // the lenses of each part: one a part, or all in one
	if st.Together {
		asked = append(asked, st.Lenses)
	} else {
		for _, lens := range st.Lenses {
			asked = append(asked, []string{lens})
		}
	}
	if st.Alone {
		asked = append(asked, []string{alone})
	}
	for _, ls := range asked {
		// A change read in parts (#446): each part of the lenses read, the
		// findings of every part put together; one part failing leaves
		// the lenses not answered, the others' findings kept.
		shares, gone := 1, 0
		if ls[0] != alone && st.Parts > 1 {
			shares = st.Parts
		}
		for k := 1; k <= shares; k++ {
			dir := filepath.Join(runDir, "in", "parts", sharePart(st, ls[0], k))
			which := fmt.Sprintf("the %s lens", ls[0])
			if len(ls) > 1 {
				which = fmt.Sprintf("the lenses %s, asked together,", strings.Join(ls, ", "))
			}
			what := "what it looks for was not reviewed"
			if shares > 1 {
				if len(ls) == 1 {
					which += ","
				}
				which += fmt.Sprintf(" part %d of %d,", k, shares)
				what += " in that part's files"
			}
			failed := func(message string) {
				for _, l := range ls {
					if !slices.Contains(c.Failed, l) {
						c.Failed = append(c.Failed, l)
					}
				}
				c.Logged = append(c.Logged, verdict.Finding{Rule: "lens-failed", Level: "warn", Message: message})
			}
			if why, err := os.ReadFile(filepath.Join(dir, "unanswered")); err == nil {
				if strings.HasPrefix(string(why), "unavailable:") {
					gone++
				}
				failed(fmt.Sprintf("%s got no answer that reads (%s): %s", which, strings.TrimSpace(string(why)), what))
				continue
			}
			if !exists(filepath.Join(dir, "answer.yaml")) {
				failed(fmt.Sprintf("%s was not asked: %s", which, what))
				continue
			}
			answers, err := intent.Read(filepath.Join(dir, "answer.yaml"))
			if err != nil {
				failed(fmt.Sprintf("%s: its answer does not read (%v): %s", which, err, what))
				continue
			}
			if err := lensFindings(repo, s, st, ls, answers, lensOf, &c); err != nil {
				return c, 0, err
			}
		}
		if gone == shares {
			unavailable += len(ls)
		}
	}
	important := func(l []Finding) {
		sort.SliceStable(l, func(i, j int) bool { return l[i].Severity == "important" && l[j].Severity != "important" })
	}
	important(c.Related)
	important(c.Outside)
	return capped(s, c), unavailable, nil
}

// lensFindings reads one part's answers: each finding's quotes found
// again, the change's told from the rest, questions for a person apart,
// findings on one line grouped.
func lensFindings(repo string, s Settings, st state, ls []string, answers []intent.Intention, lensOf func(string) (Lens, error), c *candidates) error {
	for _, a := range answers {
		lens := ls[0]
		if len(ls) > 1 {
			// Asked together, a finding names its lens. One naming none
			// is read as the first lens's, said: dropping it lost a whole
			// call's findings (tried.md, #147). One naming a lens not
			// asked is dropped, said.
			named, title := lensNamed(a.Value)
			switch {
			case named == "":
				c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-lens-unnamed", Level: "warn",
					Message: fmt.Sprintf("%q names no lens: read as the %s lens's, and judged by its question", title, ls[0])})
			case !slices.Contains(ls, named):
				c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-unfounded", Level: "warn",
					Message: fmt.Sprintf("dropped: %q names the lens %q, not one asked (%s)", title, named, strings.Join(ls, ", "))})
				continue
			default:
				lens = named
			}
		}
		l, err := lensOf(lens)
		if err != nil {
			return err
		}
		f, why := found(repo, st, lens, l.Cites == "claim", a.Value, "")
		if why != "" {
			c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-unfounded", Level: "warn",
				Message: fmt.Sprintf("dropped, from the %s lens: %q — %s", lens, f.Title, why)})
			continue
		}
		// A decision is a question for a person, never a defect: not
		// grouped with the findings, nor judged (#126). One a place: a
		// second on it asks the same choice again, and would take a
		// place of questions-max. In an issue, which has no lines, the
		// place is the words quoted.
		if f.Decision != "" {
			if !slices.ContainsFunc(c.Questions, func(q Finding) bool { return questionAt(q) == questionAt(f) }) {
				c.Questions = append(c.Questions, f)
			}
			continue
		}
		list := &c.Outside
		if f.Related {
			list = &c.Related
		}
		// Two findings on one line are grouped, never one dropped: each
		// is judged by its own lens, the group only shows them together
		// (#229); the one ahead leads, the others beside it.
		if dup := slices.IndexFunc(*list, func(o Finding) bool { return o.Where == f.Where }); dup >= 0 {
			(*list)[dup] = group(s, append(members((*list)[dup]), f))
			continue
		}
		*list = append(*list, f)
	}
	return nil
}

// capped keeps of the lenses' findings what a run shows: findings-max on
// the change, questions-max, the important ones outside it up to
// issues-max; the rest counted.
func capped(s Settings, c candidates) candidates {
	if n := len(c.Related) - s.FindingsMax; s.FindingsMax > 0 && n > 0 {
		c.Related = c.Related[:s.FindingsMax]
		c.Logged = append(c.Logged, verdict.Finding{Rule: "findings-capped", Level: "warn",
			Message: fmt.Sprintf("%d more findings on the change were not checked nor shown (findings-max %d): fix these, and review again", n, s.FindingsMax)})
	}
	if n := len(c.Questions) - s.QuestionsMax; s.QuestionsMax > 0 && n > 0 {
		c.Questions = c.Questions[:s.QuestionsMax]
		c.Logged = append(c.Logged, verdict.Finding{Rule: "questions-capped", Level: "warn",
			Message: fmt.Sprintf("%d more questions for a person were not shown (questions-max %d)", n, s.QuestionsMax)})
	}
	nits := 0
	var outside []Finding
	for _, f := range c.Outside {
		if f.Severity != "important" {
			nits++
			continue
		}
		outside = append(outside, f)
	}
	if nits > 0 {
		c.Logged = append(c.Logged, verdict.Finding{Rule: "nits-outside", Level: "warn",
			Message: fmt.Sprintf("%d nits outside the change left: only what matters there becomes an issue", nits)})
	}
	if n := len(outside) - s.IssuesMax; n > 0 {
		outside = outside[:max(s.IssuesMax, 0)]
		c.Logged = append(c.Logged, verdict.Finding{Rule: "issues-capped", Level: "warn",
			Message: fmt.Sprintf("%d more findings outside the change were not checked nor opened (issues-max %d)", n, s.IssuesMax)})
	}
	c.Outside = outside
	return c
}

// askJudges puts each important finding of a review to the judge, a
// question each, numbered from n on; one judged before, the same finding
// on the same patch, keeps its verdict, its judge not asked again
// (ADR-0050, step 6).
func askJudges(runDir, repo string, s Settings, st state, c *candidates, n *int) error {
	lensOf := lensReader(repo)
	var judged []Finding
	for _, g := range append(slices.Clone(c.Related), c.Outside...) {
		judged = append(judged, members(g)...)
	}
	for _, f := range judged {
		if f.Severity != "important" {
			continue
		}
		l, err := lensOf(f.Lens)
		if err != nil {
			return err
		}
		if st.Spec == nil {
			same := sameAs(repo, st, s, f, l)
			if c.Same == nil {
				c.Same = map[string]Answered{}
			}
			c.Same[JudgeKey(f)] = same
			if before, ok := judgedBefore(st.Record, f.Lens, same, s.JudgeAtLeast); ok {
				if c.Reused == nil {
					c.Reused = map[string]Answered{}
				}
				c.Reused[JudgeKey(f)] = before
				continue
			}
		}
		question := l.Judge.Question
		if question == "" {
			question = judgeQuestion
		}
		var m string
		switch {
		case st.Spec != nil:
			m = specMaterial(repo, st, s, f, l.Judge.Reads == "code")
		case l.Judge.Reads == "tests":
			m = material(repo, st, s, f)
			m += testsTouching(repo, st.Head, f.Cause.Path, s.Tests, s.TestsLinesMax)
		case l.Judge.Reads == "issue" && issueOf(st, f.Cause.Path) == nil: // one quoting an issue is shown it already
			m = material(repo, st, s, f) + "\n" + whatFor(st.Issues)
		default:
			m = material(repo, st, s, f)
		}
		*n++
		key := fmt.Sprintf("%02d", *n)
		dir := filepath.Join(runDir, "in", "judge", key)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		question = strings.Join(strings.Fields(question), " ")
		if s.JudgeReadRounds > 0 && st.Spec == nil { // the code's judge may read first (ADR-0045)
			if err := (reading{Question: question, Material: m, Cause: f.Cause.Path}).write(dir, s, st.Head); err != nil {
				return err
			}
		} else {
			data, _ := json.Marshal(map[string]string{"question": question, "material": m}) // JSON, which YAML reads: code quoted may start a line with a tab
			if err := os.WriteFile(filepath.Join(dir, "question.yaml"), data, 0o644); err != nil {
				return err
			}
		}
		c.Asked = append(c.Asked, JudgeKey(f)+"\x00"+key)
	}
	return nil
}

// The sides of the change a cause is found on (#570): a line the head
// holds — added, kept beside a removal, or using what a removed line
// tested — or a line the change removed.
const (
	onHead    = "head"
	onRemoved = "removed"
)

// found reads one finding a lens answered, and finds its quotes again: its
// cause in the file at the head of the range, or among the lines the change
// removed; its symptom, when it gives one, in its file. Related: its cause
// lies on a line the change added or removed, on a kept line beside a
// removal, or on one using what a removed line of its function tested or
// assigned, the side it lies on said. on, when set, looks on that side
// alone: a block found before is the change's only where it was found
// (#570). why says why it is dropped.
func found(repo string, st state, lens string, cites bool, v any, on string) (Finding, string) {
	var raw struct {
		Severity string `json:"severity"`
		Title    string `json:"title"`
		Why      string `json:"why"`
		Fix      string `json:"fix"`
		Seen     string `json:"seen"`
		Claim    string `json:"claim"`
		Decision string `json:"decision"`
		Cause    *Quote `json:"cause"`
		Symptom  *Quote `json:"symptom"`
	}
	// Through JSON: YAML would not read back a quote starting with a tab.
	data, _ := json.Marshal(v)
	if err := json.Unmarshal(data, &raw); err != nil {
		return Finding{Title: fmt.Sprint(v)}, "it does not read: " + err.Error()
	}
	f := Finding{Lens: lens, Severity: strings.ToLower(strings.TrimSpace(raw.Severity)), Title: strings.TrimSpace(raw.Title), Why: strings.TrimSpace(raw.Why), Fix: strings.TrimSpace(raw.Fix), Seen: strings.TrimSpace(raw.Seen), Symptom: raw.Symptom}
	if f.Severity != "important" {
		f.Severity = "nit"
	}
	// A decision is never important: a person answers it, no judge (#126).
	if f.Decision = strings.TrimSpace(raw.Decision); f.Decision != "" {
		f.Severity = "question"
	}
	if raw.Cause == nil || strings.TrimSpace(raw.Cause.Quote) == "" || strings.TrimSpace(raw.Cause.Path) == "" {
		return f, "it quotes no cause"
	}
	// A lens checking the author's claims quotes the claim: found again in
	// what the author said, or the finding is dropped (#126).
	if cites {
		f.Claim = strings.TrimSpace(raw.Claim)
		if f.Claim == "" {
			return f, "it quotes no claim of the author"
		}
		if len(locate(saidByAuthor(st.Testimony), f.Claim)) == 0 {
			return f, "its claim is not found in what the author said"
		}
	}
	f.Cause = Quote{Path: clean(raw.Cause.Path), Quote: raw.Cause.Quote}
	if st.Spec != nil {
		return inSpec(repo, st, f)
	}
	// A cause quoted from the issue the change closes: what it asks and the
	// change does not do, the author's to do (#126).
	if strings.HasPrefix(f.Cause.Path, "#") {
		is := issueOf(st, f.Cause.Path)
		if is == nil {
			return f, fmt.Sprintf("its cause names %s, not an issue the change closes", f.Cause.Path)
		}
		if len(locate(strings.Split(is.Text, "\n"), f.Cause.Quote)) == 0 {
			return f, fmt.Sprintf("its cause is not found in %s, as the lenses were given it", f.Cause.Path)
		}
		f.Where, f.Related, f.Symptom = f.Cause.Path, true, nil
		return f, ""
	}
	// A file the change moved is found on its new path, quoted on its old
	// one (#358): its lines, kept or removed, are there.
	if to, ok := st.Change.Moved[f.Cause.Path]; ok {
		f.Cause.Path = to
	}
	text, ok := fileAt(repo, st.Head, f.Cause.Path)
	var places []Place
	if ok {
		places = locate(strings.Split(text, "\n"), f.Cause.Quote)
	}
	head, gone := on != onRemoved, on != onHead
	var atHead []func(string, int) bool
	if head {
		atHead = []func(string, int) bool{st.Change.addedAt, st.Change.exposedAt}
	}
	// The change's: a line it added, then a kept line beside a removal, then
	// a line it removed, even one also kept elsewhere (#224).
	for _, in := range atHead {
		for _, p := range places {
			for n := p.From; n <= p.To && f.Where == ""; n++ {
				if in(f.Cause.Path, n) {
					f.Where, f.Related, f.On, f.Side = fmt.Sprintf("%s:%d", f.Cause.Path, p.From), true, n, onHead
				}
			}
		}
	}
	if f.Where == "" && gone {
		var removed []string
		for _, l := range st.Change.Removed[f.Cause.Path] {
			removed = append(removed, l.Text)
		}
		if at := locate(removed, f.Cause.Quote); len(at) > 0 {
			f.On = st.Change.Removed[f.Cause.Path][at[0].From-1].At
			f.Where, f.Related, f.Side = fmt.Sprintf("%s:%d", f.Cause.Path, f.On), true, onRemoved
		}
	}
	// Then a kept line further down using a name a removed line tested or
	// assigned, in the same function (#357): its comment goes where the
	// removal shows.
	for _, p := range places {
		if f.Where != "" || !head {
			break
		}
		if at := st.Change.reaches(repo, st.Head, f.Cause.Path, p); at > 0 {
			f.Where, f.Related, f.On, f.Side = fmt.Sprintf("%s:%d", f.Cause.Path, p.From), true, at, onHead
		}
	}
	if f.Where == "" && len(places) > 0 {
		f.Where = fmt.Sprintf("%s:%d", f.Cause.Path, places[0].From)
		f.Line = norm(strings.Split(text, "\n")[places[0].From-1])
	}
	if f.Where == "" {
		return f, fmt.Sprintf("its cause is not found in %s, as it reads at %s nor among the lines the change removed", f.Cause.Path, short(st.Head))
	}
	// A question is the change's to raise, or nobody's: never an issue.
	if f.Decision != "" && !f.Related {
		return f, "a question on what the change does not touch, never an issue"
	}
	if f.Symptom != nil {
		f.Symptom.Path = clean(f.Symptom.Path)
		if to, ok := st.Change.Moved[f.Symptom.Path]; ok {
			f.Symptom.Path = to
		}
		text, ok := fileAt(repo, st.Head, f.Symptom.Path)
		if !ok || len(locate(strings.Split(text, "\n"), f.Symptom.Quote)) == 0 {
			return f, fmt.Sprintf("its symptom is not found in %s as it reads", f.Symptom.Path)
		}
	}
	return f, ""
}

// material is what the judge reads of a finding: the finding, the code it
// stands on (reach), and what the change did near its cause.
func material(repo string, st state, s Settings, f Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The finding\n\n%s: %s\n\nWhy: %s\n", f.Severity, f.Title, f.Why)
	_, line, _ := strings.Cut(f.Where, ":")
	at := 1
	fmt.Sscan(line, &at)
	fmt.Fprintf(&b, "\nIts cause, quoted:\n\n```\n%s\n```\n", strings.TrimSpace(f.Cause.Quote))
	if f.Claim != "" {
		fmt.Fprintf(&b, "\nThe author's claim it contradicts, quoted from what they said:\n\n> %s\n", strings.ReplaceAll(f.Claim, "\n", "\n> "))
	}
	// A cause quoted from an issue: the issue, and the change it is
	// compared with: the whole range, base..head, commits reviewed before
	// included, so a part one of them did is not missing; up to code-lines-max.
	if is := issueOf(st, f.Cause.Path); is != nil {
		b.WriteString("\n" + whatFor([]ClosedIssue{*is}))
		if diff, err := git(repo, "diff", "-U1", "--no-color", "--no-ext-diff", "--find-renames", st.Base, st.Head, "--"); err == nil {
			fmt.Fprintf(&b, "\n## The whole change: every commit of the merge request, those reviewed before too\n\n```diff\n%s```\n", capLines(diff, s.CodeLinesMax, "the change is cut here, past code-lines-max"))
		}
		return b.String()
	}
	symptomAt := 0
	if f.Symptom != nil {
		text, _ := fileAt(repo, st.Head, f.Symptom.Path)
		if p := locate(strings.Split(text, "\n"), f.Symptom.Quote); len(p) > 0 {
			symptomAt = p[0].From
		}
	}
	b.WriteString(reach(repo, st.Head, s.Tests, f, at, symptomAt, s.JudgeLinesMax))
	if diff, err := git(repo, append([]string{"diff", "-U3", "--no-color", "--no-ext-diff", "--find-renames", st.Base, st.Head, "--"}, pathspec([]File{st.Change.file(f.Cause.Path)})...)...); err == nil && diff != "" {
		near, far := hunksNear(diff, at, nearCause)
		fmt.Fprintf(&b, "\n## What the change did to %s, near the cause\n\n", f.Cause.Path)
		if near != "" {
			fmt.Fprintf(&b, "```diff\n%s```\n", capLines(near, 200, "cut"))
		} else {
			fmt.Fprintf(&b, "Nothing within %d lines of it.\n", nearCause)
		}
		if far > 0 {
			fmt.Fprintf(&b, "\n(%d %s further from the cause not shown.)\n", far, plural(far, "hunk", "hunks"))
		}
	}
	return b.String()
}

// nearCause is how far from a finding's cause, in lines, a hunk of the
// change is shown to its judge: what the change did there, not the whole
// file's diff at every call (#147).
const nearCause = 40

// hunksNear keeps of one file's diff its header and the hunks whose lines,
// at the head, come within `within` lines of line at; far counts the others.
func hunksNear(diff string, at, within int) (near string, far int) {
	var head strings.Builder
	var hunks []string
	for _, l := range strings.SplitAfter(diff, "\n") {
		switch {
		case strings.HasPrefix(l, "@@"):
			hunks = append(hunks, l)
		case len(hunks) == 0:
			head.WriteString(l)
		default:
			hunks[len(hunks)-1] += l
		}
	}
	var kept strings.Builder
	for _, h := range hunks {
		from, n := 0, 1 // "@@ -a,b +from,n @@"
		if _, plus, ok := strings.Cut(h, " +"); ok {
			r, _, _ := strings.Cut(plus, " ")
			first, count, has := strings.Cut(r, ",")
			fmt.Sscan(first, &from)
			if has {
				fmt.Sscan(count, &n)
			}
		}
		if at+within < from || at-within > from+max(n, 1)-1 {
			far++
			continue
		}
		kept.WriteString(h)
	}
	if kept.Len() == 0 {
		return "", far
	}
	return head.String() + kept.String(), far
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// testFiles are the test files at the head (the `tests` setting) that
// touch a file: in its folder, then those naming it as a word.
func testFiles(repo, head, file string, patterns []string) []string {
	dir, stem := path.Dir(file), strings.TrimSuffix(path.Base(file), path.Ext(file))
	var near, far []string
	if out, err := git(repo, "ls-tree", "-r", "--name-only", head, "--", dir+"/"); err == nil {
		for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
			if f != "" && path.Dir(f) == dir && pathglob.Any(patterns, f) {
				near = append(near, f)
			}
		}
	}
	if out, err := git(repo, "grep", "-l", "-w", "-F", "-e", stem, head, "--"); err == nil { // none: git grep fails
		for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
			f = strings.TrimPrefix(f, head+":")
			if f != "" && path.Dir(f) != dir && pathglob.Any(patterns, f) {
				far = append(far, f)
			}
		}
	}
	return append(near, far...)
}

// testsTouching is what a judge reading tests is shown besides: the test
// files at the head (the `tests` setting) that touch the cause's file — in
// its folder, or naming it as a word — whole, its folder's first, up to limit
// lines all together; those left out are named.
func testsTouching(repo, head, file string, patterns []string, limit int) string {
	var b strings.Builder
	dir, stem := path.Dir(file), strings.TrimSuffix(path.Base(file), path.Ext(file))
	fmt.Fprintf(&b, "\n## The tests that touch %s\n\nThe test files in %s/ and those naming `%s`, as they read at %s.\n", file, dir, stem, short(head))
	if len(patterns) == 0 {
		b.WriteString("\nNone shown: no `tests` setting says which files are tests.\n")
		return b.String()
	}
	files := testFiles(repo, head, file, patterns)
	if len(files) == 0 {
		b.WriteString("\nNo test file touches it.\n")
		return b.String()
	}
	left := limit
	var cut []string
	for _, f := range files {
		text, ok := fileAt(repo, head, f)
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		if !ok || len(lines) > left {
			cut = append(cut, f)
			continue
		}
		left -= len(lines)
		fmt.Fprintf(&b, "\n### %s\n\n```\n%s\n```\n", f, strings.Join(lines, "\n"))
	}
	if len(cut) > 0 {
		fmt.Fprintf(&b, "\nNot shown, past %d lines: %s. A test in them may exercise it.\n", limit, strings.Join(cut, ", "))
	}
	return b.String()
}

// settle reads the judge's answers and proposes what follows: the findings
// on the change in the verdict, an issue for each verified one outside it,
// and the summary on the merge request. Several specs: each settled apart,
// the run's verdict holding them all (#433).
func settle(runDir, repo string, s Settings) int {
	var st state
	var c candidates
	if err := readJSON(filepath.Join(runDir, "in", "review-state.json"), &st); err != nil {
		return fail(err)
	}
	if len(st.Specs) > 0 {
		return settleSpecs(runDir, repo, s, st)
	}
	if err := readJSON(filepath.Join(runDir, "in", "candidates.json"), &c); err != nil {
		return fail(err)
	}
	v, fallback, err := settled(runDir, repo, s, st, c)
	if err != nil {
		return fail(err)
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// settled is one review settled from the judge's answers: its verdict, and
// what it proposes.
func settled(runDir, repo string, s Settings, st state, c candidates) (Review, []intent.Intention, error) {
	keys := map[string]string{}
	for _, a := range c.Asked {
		i := strings.LastIndex(a, "\x00")
		keys[a[:i]] = a[i+1:]
	}
	floor := s.JudgeAtLeast
	if floor == "" {
		floor = "context"
	}
	unjudged := 0 // findings the judge was not asked of, the run's tokens spent
	reused := 0   // findings judged before on the same patch, their verdicts kept (ADR-0050, step 6)
	var readsDone []JudgeRead
	var judged []Answered // the findings the judge kept or dropped (ADR-0046)
	// Each keyed by what it was judged on, so the same finding on the same
	// patch is not judged again; a yes keeps its independence, a no its
	// reason (ADR-0050, step 6).
	answered := func(f Finding, a readAnswer, yes bool, why string) {
		same := c.Same[JudgeKey(f)]
		j := Answered{Lens: f.Lens, Where: f.Where, At: st.Head, Patch: same.Patch, Finding: same.Finding, Yes: yes, Finder: a.Author, Judge: a.Model, Title: f.Title}
		if yes {
			j.Level = a.Level
		} else {
			j.Why = oneLine(why, 200)
		}
		judged = append(judged, j)
	}
	verify := func(l []Finding) []Finding {
		var kept []Finding
		for _, f := range l {
			if before, ok := c.Reused[JudgeKey(f)]; ok {
				reused++
				again := fmt.Sprintf("judged reviewing %s: the same finding on the same patch, not judged again", short(before.At))
				if before.Yes {
					f.Verified, f.Level = fmt.Sprintf("verified, independence: %s (%s → %s), %s", before.Level, before.Finder, before.Judge, again), before.Level
					kept = append(kept, f)
					continue
				}
				c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-judged-no", Level: "warn",
					Message: fmt.Sprintf("dropped, from the %s lens: %q at %s — the judge said no, %s: %s", f.Lens, f.Title, f.Where, again, before.Why)})
				continue
			}
			key, ok := keys[JudgeKey(f)]
			if !ok {
				kept = append(kept, f) // a nit: reported as found
				continue
			}
			dir := filepath.Join(runDir, "in", "judge", key)
			a, err := answerIn(dir)
			// A judge that may read (ADR-0045): what it read, said beside
			// its answer; a no stands on its quote, found again.
			rd, reads := readingOf(dir)
			read := rd.said()
			for i, round := range rd.Rounds {
				readsDone = append(readsDone, JudgeRead{For: "judge/" + key, Where: f.Where, Lens: f.Lens, Round: i + 1,
					Asked: readsSaid(round.Asked), Lines: round.Lines, Refused: round.Refused})
			}
			who := fmt.Sprintf("independence: %s (%s → %s)", a.Level, a.Author, a.Model)
			stopped := strings.Contains(a.Error, tokensSpent)
			if stopped && len(rd.Rounds) == 0 {
				unjudged++
			}
			unsure := "not verified: the judge cannot tell (judge-unsure): "
			switch {
			case reads && stopped && len(rd.Rounds) > 0: // it answered once: its reads, cut short
				f.Verified = unsure + "its reads were stopped, the run having spent its ai-max-tokens" + read
			case reads && err == nil && len(a.Reads) > 0:
				f.Verified = unsure + "it asked to read again, no read left (judge-read-rounds, judge-read-lines-max)" + read
			case reads && err == nil && a.Unsure:
				f.Verified = unsure + a.Why + read
			case err != nil || a.Yes == nil:
				why := a.Error
				if why == "" {
					why = "the judge was not asked"
				}
				f.Verified = "not verified: " + why
			case !*a.Yes:
				reason := a.Why
				if reads {
					where, notFound := rd.quoted(repo, st, a.Quote)
					if notFound != "" {
						f.Verified = fmt.Sprintf("not verified: the judge said no (judge-no-unquoted): %s — %s%s", notFound, a.Why, read)
						kept = append(kept, f)
						continue
					}
					reason = strings.TrimRight(a.Why, ". ") + ", quoting " + where + read
				}
				c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-judged-no", Level: "warn",
					Message: fmt.Sprintf("dropped, from the %s lens: %q at %s — the judge: %s (%s)", f.Lens, f.Title, f.Where, reason, who)})
				answered(f, a, false, a.Why)
				continue
			case judge.Levels[a.Level] < judge.Levels[floor]:
				f.Verified, f.Level = fmt.Sprintf("not verified: the judge stood at %s, below %s (%s)", a.Level, floor, a.Why), a.Level
			default:
				f.Verified, f.Level = "verified, "+who+read, a.Level
				answered(f, a, true, "")
			}
			kept = append(kept, f)
		}
		return kept
	}
	// Each finding of a group is judged apart; what its judge keeps is
	// grouped again, the one ahead leading (#229).
	regroup := func(l []Finding) []Finding {
		var kept []Finding
		for _, g := range l {
			if ms := verify(members(g)); len(ms) > 0 {
				kept = append(kept, group(s, ms))
			}
		}
		return kept
	}
	related, outside := regroup(c.Related), regroup(c.Outside)
	var err error
	if st.Standing, err = reread(repo, s, st, c, related, len(c.Failed) == 0 && unjudged == 0); err != nil {
		return Review{}, nil, err
	}
	v := review(st, nil, "")
	v.Lenses, v.Change, v.Complete, v.Parts = facets(st), related, len(c.Failed) == 0, st.Parts
	var fallback []intent.Intention
	onForge := mergeRequest() != nil && s.ForgeWrites
	blocks := slices.Clone(st.Standing)
	for _, f := range related {
		level := "warn"
		if s.blocking(f) {
			level = ""
			blocks = append(blocks, Block{Lens: f.Lens, At: st.Head, Where: f.Where, Title: shortMessage(f), Cause: f.Cause, Side: f.Side})
		}
		v.Findings = append(v.Findings, verdict.Finding{Rule: f.Lens, Where: f.Where, Level: level, Message: message(f)})
	}
	v.Questions = c.Questions
	for _, f := range c.Questions {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "decision", Where: f.Where, Level: "question",
			Message: fmt.Sprintf("Question for a person: %s — %s (the %s lens; not a defect, not judged)", f.Decision, f.Why, f.Lens)})
	}
	for _, f := range outside {
		if !strings.HasPrefix(f.Verified, "verified") {
			c.Logged = append(c.Logged, verdict.Finding{Rule: "outside-unverified", Level: "warn",
				Message: fmt.Sprintf("outside the change, at %s, not opened as an issue (%s): %s", f.Where, f.Verified, f.Title)})
			continue
		}
		v.Outside = append(v.Outside, f)
		if s.ForgeWrites && os.Getenv("WORKLINE_FORGE") != "" {
			fallback = append(fallback, intent.Intention{Kind: "issue", Value: map[string]any{
				"title": f.Title, "at": map[string]any{"path": f.Cause.Path, "text": f.Line},
				"body": issueBody(f), "found": fmt.Sprintf("while reviewing %s, outside that change", short(st.Head))}})
		} else {
			c.Logged = append(c.Logged, verdict.Finding{Rule: "outside-the-change", Level: "warn",
				Message: fmt.Sprintf("at %s, outside the change, for an issue (none opened: no forge to write to): %s — %s", f.Where, f.Title, f.Why)})
		}
	}
	// A finding left unjudged for want of tokens leaves the review not
	// whole: its commits are reviewed again, the judge asked then (#147).
	if unjudged > 0 {
		v.Complete = false
		c.Logged = append(c.Logged, verdict.Finding{Rule: "review-not-whole", Level: "warn",
			Message: fmt.Sprintf("%d %s not judged, the run having spent its ai-max-tokens (%d): the commits are not recorded as reviewed; review again, with fewer commits or a larger ai-max-tokens", unjudged, plural(unjudged, "finding", "findings"), s.AIMaxTokens)})
	}
	v.Findings = append(v.Findings, c.Logged...)
	if st.Spec == nil { // a spec is read whole each run: no commits to record, nothing carried
		if v.Complete {
			v.Record = st.recorded()
		} else {
			// Unread by the lenses that did not answer, the commits are
			// theirs: no run records them before they read them (#588).
			v.Record.Failed = st.Record.owing(facets(st), c.Failed)
		}
		v.Record.Blocks = blocks
		v.Record.Judged = slices.Clone(v.Record.Judged)
		for _, j := range judged { // one answer a finding at a head: the same asked again is not a second act
			if !slices.Contains(v.Record.Judged, j) {
				v.Record.Judged = append(v.Record.Judged, j)
			}
		}
	}
	v.Record.Owed = false // the lenses were asked: a rule holds them no more
	everyLens(st, &v, len(c.Failed) == 0)
	v.Calls = spent(runDir, st, c)
	v.Reads = readsDone
	v.Reused = reused
	v.Status = status(v.Findings)
	v.Summary = summary(v, len(fallback), len(c.Failed), unjudged) + st.leftSaid() + st.still()
	if len(st.Skipped) > 0 {
		v.Summary += "; not asked: " + strings.Join(st.Skipped, ", ")
	}
	v.Summary += tokensLine(v.Calls, s.AIMaxTokens)
	if onForge {
		fallback = append(fallback, summaryIntention(repo, s, st, v, related, len(fallback)))
	}
	if sp := st.Spec; sp != nil && sp.Issue > 0 && s.ForgeWrites {
		// On the forge (#128): one comment on the issue, its record read
		// back by the product owner's ready.
		fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{"issue": sp.Issue, "sticky": backlog.SpecKey,
			"body": specComment(v, specRecord(st, v), s.SpecRounds)}})
	}
	return v, fallback, nil
}

// Post writes the verdict from what pre settled, and keeps the record on
// this machine once every lens answered.
func Post(runDir, repo string) int {
	var v Review
	if err := readJSON(filepath.Join(runDir, "in", "review.json"), &v); err != nil {
		return fail(err)
	}
	if v.Complete && v.Subject != "spec" {
		if err := saveLocal(repo, v.Warn, v.Record); err != nil {
			return fail(err)
		}
	}
	if err := writeJSON(filepath.Join(runDir, "out", "review.json"), v); err != nil {
		return fail(err)
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &verdict.Verdict{Status: v.Status, Summary: v.Summary, Findings: v.Findings, Final: true}); err != nil {
		return fail(err)
	}
	if v.Status == verdict.Block {
		return exitBlock
	}
	return 0
}

// review starts a review from the rules' findings and the last review's
// that still block (#326), the record keeping those.
func review(st state, extra []verdict.Finding, summary string) Review {
	v := Review{Subject: "code", Findings: append(append(slices.Clone(st.Mechanical), extra...), st.standingFindings()...), Record: st.Record, Warn: st.Warn}
	v.Findings = append(v.Findings, st.Advisories...)
	if st.Spec != nil {
		v.Subject, v.Spec = "spec", st.Spec.Path
	} else {
		v.Record.Blocks = st.Standing
	}
	v.Status = status(v.Findings)
	v.Summary = summary
	if summary != "" {
		v.Summary += st.still()
	}
	return v
}

// standingFindings are the last review's findings that still block, each
// saying where it was found: no lens read them again (#326).
func (st state) standingFindings() []verdict.Finding {
	var out []verdict.Finding
	for _, b := range st.Standing {
		at := "by a review before"
		if b.At != "" {
			at = "reviewing " + short(b.At)
		}
		out = append(out, verdict.Finding{Rule: b.Lens, Where: b.Where,
			Message: fmt.Sprintf("%s (found %s; no commit since answers it)", b.Title, at)})
	}
	return out
}

// still is what a summary says of the findings standing.
func (st state) still() string {
	n := len(st.Standing)
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("; %d %s of a review before still %s, until a commit answers %s", n,
		plural(n, "finding", "findings"), plural(n, "blocks", "block"), plural(n, "it", "them"))
}

// blocking tells whether a lens's finding blocks: important, verified, its
// lens set to block (ai-findings).
func (s Settings) blocking(f Finding) bool {
	return s.AIFindings.Blocks(f.Lens) && f.Severity == "important" && strings.HasPrefix(f.Verified, "verified")
}

// status blocks on a finding that blocks: a rule's, or a verified finding
// once ai-findings says block for its lens.
func status(fs []verdict.Finding) string {
	for _, f := range fs {
		if f.Level == "" || f.Level == "block" {
			return verdict.Block
		}
	}
	return verdict.Pass
}

func message(f Finding) string {
	sev := "Important"
	if f.Severity != "important" {
		sev = "Nit"
	}
	m := fmt.Sprintf("%s: %s — %s", sev, f.Title, f.Why)
	if f.Fix != "" {
		m += " Fix: " + f.Fix
	}
	if f.Symptom != nil {
		m += fmt.Sprintf(" (shows at %s)", f.Symptom.Path)
	}
	if f.Claim != "" {
		m += fmt.Sprintf(" (the author: %q)", f.Claim)
	}
	if len(f.Also) > 0 {
		var also []string
		for _, o := range f.Also {
			a := o.Lens + ": " + o.Title
			if o.Claim != "" {
				a += fmt.Sprintf(" (the author: %q)", o.Claim)
			}
			if o.Verified != "" {
				a += " (" + o.Verified + ")"
			}
			also = append(also, a)
		}
		m += " (also found on this line — " + strings.Join(also, "; ") + ")"
	}
	if f.Verified != "" {
		m += " (" + f.Verified + ")"
	}
	return m
}

// members are the findings a group holds: the one leading, then the others.
func members(g Finding) []Finding {
	lead := g
	lead.Also = nil
	return append([]Finding{lead}, g.Also...)
}

// group puts findings on one line together: the one ahead leads, the others
// beside it in the order they came.
func group(s Settings, ms []Finding) Finding {
	lead := 0
	for i := range ms {
		if ahead(s, ms[i], ms[lead]) {
			lead = i
		}
	}
	g := ms[lead]
	g.Also = slices.Delete(slices.Clone(ms), lead, lead+1)
	return g
}

// ahead tells whether a finding leads another on their line: a verified
// one first, then an important one from a blocking lens (ai-findings,
// #222), then an important one; of two alike, the first found.
func ahead(s Settings, a, b Finding) bool {
	rank := func(f Finding) int {
		r := 0
		if f.Severity == "important" {
			r++
			if s.AIFindings.Blocks(f.Lens) {
				r++
			}
		}
		if strings.HasPrefix(f.Verified, "verified") {
			r += 4
		}
		return r
	}
	return rank(a) > rank(b)
}

// JudgeKey keys a finding's question to the judge: its line, lens,
// severity, title and cause; two findings alike in all read one answer.
func JudgeKey(f Finding) string {
	return strings.Join([]string{f.Where, f.Lens, f.Severity, f.Title, f.Cause.Quote}, "\x00")
}

// questionAt is where a question stands: its line, or, quoted from an
// issue, the issue and the words quoted.
func questionAt(f Finding) string {
	if strings.HasPrefix(f.Where, "#") {
		return f.Where + "\x00" + norm(f.Cause.Quote)
	}
	return f.Where
}

// oneLine is a text on one line, cut past max characters.
func oneLine(t string, max int) string {
	t = strings.Join(strings.Fields(t), " ")
	if r := []rune(t); len(r) > max {
		t = string(r[:max]) + "…"
	}
	return t
}

// issueBody is the issue a finding outside the change becomes, written as
// the product owner writes one (roles/product-owner/docs/writing.md): what
// a person sees first, then the cause in the code, folded; the lens and
// the judge hidden. A finding that says nothing of what a person sees
// opens with its cause.
func issueBody(f Finding) string {
	top := strings.TrimSpace(f.Seen)
	quote := strings.TrimSpace(f.Cause.Quote)
	fence := backlog.FenceFor(quote) // a fence in the quote does not end the block
	cause := fmt.Sprintf("At `%s`:\n\n%s\n%s\n%s", f.Where, fence, quote, fence)
	if top == "" {
		top = f.Why
	} else {
		cause = f.Why + "\n\n" + cause
	}
	if fix := fixLine(f); fix != "" {
		cause += "\n\n" + fix
	}
	how := fmt.Sprintf("found=the %s lens; %s", f.Lens, f.Verified)
	for strings.Contains(how, "--") { // a run of dashes, however long, would end the marker
		how = strings.ReplaceAll(how, "--", "-")
	}
	return top + "\n\n<details><summary>The cause in the code</summary>\n\n" + cause + "\n\n</details>\n\n" + forge.Marker(how)
}

func fixLine(f Finding) string {
	if f.Fix == "" {
		return ""
	}
	return "What would fix it: " + f.Fix
}

func summary(v Review, issues, failed, unjudged int) string {
	if v.Status == verdict.BlockedExternal {
		return v.Summary
	}
	var parts []string
	on := "the change"
	if v.Subject == "spec" {
		on = "the spec"
	}
	parts = append(parts, fmt.Sprintf("%d findings on %s", len(v.Change), on))
	if n := len(v.Questions); n > 0 {
		parts = append(parts, fmt.Sprintf("%d %s for a person", n, plural(n, "question", "questions")))
	}
	if issues > 0 {
		parts = append(parts, fmt.Sprintf("%d issues proposed for what lies outside it", issues))
	}
	s := strings.Join(parts, ", ") + "; lenses: " + strings.Join(v.Lenses, ", ")
	if v.Parts > 1 {
		s += fmt.Sprintf("; read in %d parts, the change too large for one call", v.Parts)
	}
	if n := v.Reused; n > 0 {
		s += fmt.Sprintf("; %d %s judged before on the same patch: %s kept, not judged again", n, plural(n, "finding", "findings"), plural(n, "its verdict", "their verdicts"))
	}
	switch {
	case v.Complete:
	case failed > 0:
		s += "; not every lens answered: not reviewed whole"
	case unjudged > 0:
		s += "; not every finding judged, the tokens spent: not reviewed whole"
	}
	if v.Status == verdict.Block {
		s += "; the author fixes what blocks"
	}
	return s
}

// summaryIntention is the summary comment, carrying the findings on lines
// as its review (#277): each on its line, its row here kept to its title.
func summaryIntention(repo string, s Settings, st state, v Review, related []Finding, issues int) intent.Intention {
	lines := lineComments(repo, s, st, related)
	short := map[string]string{}
	for _, g := range related {
		if _, _, ok := fileLine(g.Where); ok {
			short[g.Lens+"\x00"+g.Where] = shortMessage(g)
		}
	}
	value := summaryValue(st, summaryComment(v, issues, short, lines))
	if len(lines) > 0 {
		value["review"] = reviewOf(st, lines)
	}
	return intent.Intention{Kind: "comment", Value: value}
}

// summaryValue is the summary comment's proposal: its body, and the
// record the run read, which the apply merges with the one the comment
// holds by then, a run applied since having written it (#375).
func summaryValue(st state, body string) map[string]any {
	return map[string]any{"sticky": keyOf(st.Warn), "body": body, "read": st.Read}
}

// summaryComment is the one summary the merge request gets, edited on each run,
// the record hidden in it. With findings on lines, the engine says where
// the review stands in its region (forge.FillReview), and a row on its
// line is short; an engine posting no comment on a line leaves what the
// region holds: each finding in full.
func summaryComment(v Review, issues int, short map[string]string, lines []forge.LineComment) string {
	var b strings.Builder
	switch v.Warn {
	case warnOnly: // the run apart (ADR-0050)
		fmt.Fprintf(&b, "**workline reviewer, the lenses that warn** — %s.\n\n", v.Summary)
		b.WriteString("They read apart from the verdict, which never waits for them: what they find warns, posted here once they answered, maybe after the verdict was given.\n\n")
	case warnApart:
		fmt.Fprintf(&b, "**workline reviewer** — %s.\n\n", v.Summary)
		b.WriteString("The lenses that warn read apart, and post in their own comment: they never hold the verdict.\n" + splitMark + "\n\n")
	default:
		fmt.Fprintf(&b, "**workline reviewer** — %s.\n\n", v.Summary)
	}
	if len(lines) > 0 {
		fmt.Fprintf(&b, "%s\n%s\n%s\n\n", forge.ReviewHere,
			forge.Unplaced(lines, "the engine that applied this run posts no comment on a line").Section(), forge.ReviewEnd)
	}
	b.WriteString(findingsTable(v, "No finding on the commits reviewed.", short))
	if len(v.Questions) > 0 {
		b.WriteString("**Questions for a person**\n\nNot defects, not judged: choices the change leaves to a person; a reply here is enough, the reviewer does not wait for it.\n\n")
		for _, q := range v.Questions {
			fmt.Fprintf(&b, "- `%s` (%s): %s On: “%s”\n", q.Where, q.Lens, oneLine(q.Decision, 300), oneLine(q.Cause.Quote, 200))
		}
		b.WriteString("\n")
	}
	if issues > 0 {
		fmt.Fprintf(&b, "What lies outside this change goes to %d issues, labelled `needs-triage`: not the author's to fix here.\n\n", issues)
	}
	b.WriteString("The reviewer never approves and never changes the code: the author fixes, a person merges.\n\n")
	b.WriteString(v.Record.String())
	if v.Subject != "spec" { // what every run's judge answered, within the comment's size (ADR-0046)
		b.WriteString(v.Record.judgedLines(SummaryMax - b.Len()))
	}
	return b.String()
}

// findingsTable is the findings as a table, what blocks first (#226), the
// decisions left out: they are asked apart (#126); none, empty. A finding
// on its line on the forge reads short, by its rule and place (#277).
func findingsTable(v Review, empty string, short map[string]string) string {
	var b strings.Builder
	rows := slices.DeleteFunc(slices.Clone(v.Findings), func(f verdict.Finding) bool { return f.Rule == "decision" })
	if len(rows) == 0 {
		return empty + "\n\n"
	}
	blocks := func(f verdict.Finding) bool { return status([]verdict.Finding{f}) == verdict.Block }
	sort.SliceStable(rows, func(i, j int) bool { return blocks(rows[i]) && !blocks(rows[j]) })
	b.WriteString("| | | Where | Finding |\n|---|---|---|---|\n")
	for _, f := range rows {
		where := f.Where
		if where != "" {
			where = "`" + where + "`"
		}
		holds := "warns"
		if blocks(f) {
			holds = "**blocks**"
		}
		message := f.Message
		if m, ok := short[f.Rule+"\x00"+f.Where]; ok {
			message = m
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", holds, f.Rule, where, strings.ReplaceAll(message, "|", "\\|"))
	}
	b.WriteString("\n")
	return b.String()
}

// held settles a run the rules alone decide: blocked on a merge request it
// may write to, the summary comment still says why, the record left as it
// was, no lens having run (#226); post then blocks. Otherwise, final.
func held(runDir, repo string, s Settings, st state, v Review) int {
	if v.Status != verdict.Block || mergeRequest() == nil || !s.ForgeWrites {
		return final(runDir, v)
	}
	fallback := []intent.Intention{summaryIntention(repo, s, st, v, nil, 0)}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// final writes a verdict pre settles alone, with nothing to ask: no agent,
// nothing new, no code.
func final(runDir string, v Review) int {
	if v.Status == "" {
		v.Status = verdict.Pass
	}
	if err := writeJSON(filepath.Join(runDir, "out", "review.json"), v); err != nil {
		return fail(err)
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &verdict.Verdict{Status: v.Status, Summary: v.Summary, Findings: v.Findings}); err != nil {
		return fail(err)
	}
	return exitNothing
}

func lensDir(runDir string, st state, lens string) string {
	return filepath.Join(runDir, "in", "parts", partName(st, lens))
}

// partName is the part a lens is asked in: its own, or, the lenses asked
// together, the one part holding them all; the diff-alone facet's after
// the lenses'. One spec of several a run reads: its number first (#433).
func partName(st state, lens string) string {
	n := len(st.Lenses)
	if st.Together {
		n = 1
	}
	name := fmt.Sprintf("%d-%s", slices.Index(st.Lenses, lens)+1, lens)
	switch {
	case lens == alone:
		name = fmt.Sprintf("%d-%s", n+1, alone)
	case st.Together:
		name = "1-lenses"
	}
	if st.Part != "" {
		return st.Part + "-" + name
	}
	return name
}

// sharePart is the part a lens is asked in for the k-th share of a change
// read in parts (#446): the first keeps the lens's own name, the others
// the number of the lens's part and their own, `1.2-lenses`, asked after
// it.
func sharePart(st state, lens string, k int) string {
	name := partName(st, lens)
	i := strings.Index(name, "-")
	if k <= 1 || i < 0 {
		return name
	}
	return fmt.Sprintf("%s.%d%s", name[:i], k, name[i:])
}

// facets are what a run asked: its lenses, then the diff-alone facet.
func facets(st state) []string {
	if st.Alone {
		return append(slices.Clone(st.Lenses), alone)
	}
	return st.Lenses
}

// lensNamed is the lens a finding names, written as the lenses are named,
// and its title, to say which was dropped.
func lensNamed(v any) (lens, title string) {
	m, _ := v.(map[string]any)
	lens, _ = m["lens"].(string)
	title, _ = m["title"].(string)
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(lens)), " ", "-"), title
}

// committerSettings are the committer's, as the project sets them: a code
// it refuses in a subject is refused in a comment, and one it allows,
// allowed.
func committerSettings(repo string) (committer.Settings, error) {
	var s committer.Settings
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), "committer")
	if err != nil {
		return s, fmt.Errorf("the committer's internal codes: %w", err)
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return s, err
	}
	data, _ := json.Marshal(r.MergedSettings(cfg))
	return s, json.Unmarshal(data, &s)
}

// enforcement is how hard each rule bites in this project, as the engine
// reads it on the verdict after: `roles.<name>.enforce`.
func enforcement(repo string) (map[string]string, error) {
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return nil, err
	}
	return cfg.Roles[roleName()].Enforce, nil
}

func settings(runDir string) (Settings, error) {
	s := Settings{Base: "main", FindingsMax: 10, IssuesMax: 3, DiffLinesMax: 1500, PartsMax: 2, CodeLinesMax: 600, TestsLinesMax: 300, JudgeLinesMax: 200, LensesPerPush: 1, CommitsPerPush: 5, IssueLinesMax: 80, TestimonyLines: 80, QuestionsMax: 3, SpecLinesMax: 300, SpecRounds: 5, SpecsPerRun: 5}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("settings: %w", err)
	}
	if s.SpecRounds < 1 || s.SpecRounds > 20 {
		return s, fmt.Errorf("settings: spec-rounds: %d is not a number from 1 to 20", s.SpecRounds)
	}
	if s.SpecsPerRun < 1 || s.SpecsPerRun > 20 {
		return s, fmt.Errorf("settings: specs-per-run: %d is not a number from 1 to 20", s.SpecsPerRun)
	}
	if s.PartsMax < 1 || s.PartsMax > 8 {
		return s, fmt.Errorf("settings: parts-max: %d is not a number from 1 to 8", s.PartsMax)
	}
	if s.CommitsPerPush < 0 {
		return s, fmt.Errorf("settings: commits-per-push: %d is below 0 (0: every new commit)", s.CommitsPerPush)
	}
	if slices.Contains(s.Lenses, alone) {
		return s, fmt.Errorf("settings: lenses names %q, a facet asked apart by its own setting, diff-alone", alone)
	}
	for lens := range s.AIFindings.ByLens {
		if !slices.Contains(s.Lenses, lens) && !slices.Contains(s.SpecLenses, lens) && lens != alone {
			return s, fmt.Errorf("settings: ai-findings names %q, not one of the lenses %v, the spec lenses %v, nor %s", lens, s.Lenses, s.SpecLenses, alone)
		}
	}
	return s, nil
}

// Gate is what a verified important finding does: warn or block, one value
// for every lens, or a map by lens where a lens not named warns (#222).
type Gate struct {
	All    string            // the one value, when no map is given
	ByLens map[string]string // by lens, when a map is given
}

// Blocks says whether a verified important finding of the lens blocks.
func (g Gate) Blocks(lens string) bool {
	if g.ByLens != nil {
		return g.ByLens[lens] == "block"
	}
	return g.All == "block"
}

// UnmarshalJSON reads warn, block, or a map of the two by lens; anything
// else is refused, never read as warn (principle 12).
func (g *Gate) UnmarshalJSON(data []byte) error {
	check := func(v string) error {
		if v != "warn" && v != "block" {
			return fmt.Errorf("ai-findings: %q is neither warn nor block", v)
		}
		return nil
	}
	if string(data) == "null" {
		*g = Gate{}
		return nil
	}
	var one string
	if json.Unmarshal(data, &one) == nil {
		*g = Gate{All: one}
		return check(one)
	}
	var by map[string]string
	if err := json.Unmarshal(data, &by); err != nil {
		return fmt.Errorf("ai-findings: warn, block, or a map of the two by lens: %w", err)
	}
	for lens, v := range by {
		if err := check(v); err != nil {
			return fmt.Errorf("%w, for the %s lens", err, lens)
		}
	}
	if by == nil {
		by = map[string]string{}
	}
	*g = Gate{ByLens: by}
	return nil
}

func roleName() string {
	if r := os.Getenv("WORKLINE_ROLE"); r != "" {
		return r
	}
	return "reviewer"
}

func input(runDir, name string) string {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
	return strings.TrimSpace(string(data))
}

func capLines(s string, max int, why string) string {
	lines := strings.SplitAfter(s, "\n")
	if max <= 0 || len(lines) <= max {
		return s
	}
	return strings.Join(lines[:max], "") + "\n(" + why + ")\n"
}

func clean(p string) string {
	p = strings.TrimSpace(p)
	for _, pre := range []string{"./", "a/", "b/"} {
		p = strings.TrimPrefix(p, pre)
	}
	return filepath.ToSlash(filepath.Clean(p))
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "reviewer:", err)
	return 99
}
