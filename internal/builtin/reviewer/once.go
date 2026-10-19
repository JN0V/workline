package reviewer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/verdict"
)

// After the run on ready, AI reads only what is new (ADR-0050, step 6):
// the record keeps the commits read, by patch id (#286); here, the lenses
// that warn read at most commits-per-push new commits a push, and a
// finding judged once keeps its verdict while its patch stays the same.

// capCommits caps the new commits a run whose lenses all warn reads a push
// on a merge request, once every lens read it on ready (read=all): past
// commits-per-push of them changing code, counted the oldest first, merges
// aside, the lenses read up to the last one within it (st.Upto); those are
// recorded, the rest left for the next run, said. The run on ready, a run
// asking a lens that blocks, and a machine's review read every one: what
// can hold the verdict is never left unread.
func capCommits(repo string, s Settings, st *state, sp span) error {
	if s.CommitsPerPush <= 0 || st.Every || !st.Record.ReadAll || mergeRequest() == nil || len(sp.Fresh) <= s.CommitsPerPush {
		return nil
	}
	for _, l := range facets(*st) {
		if s.AIFindings.Blocks(l) {
			return nil
		}
	}
	upto, n := "", 0
	for i, c := range sp.Fresh {
		if sp.merges[c] {
			continue
		}
		// A commit git cannot diff with its parent is counted: never one
		// left out of the count in silence.
		if code, err := codeFiles(repo, s, c+"^", c); err == nil && len(code) == 0 {
			continue
		}
		if n++; n == s.CommitsPerPush+1 {
			upto = sp.Fresh[i-1]
		}
	}
	if upto == "" {
		return nil
	}
	out, err := git(repo, "rev-list", st.Base+".."+upto)
	if err != nil {
		return err
	}
	held := strings.Fields(out)
	var commits, patches, left []string
	for _, c := range st.Commits {
		if slices.Contains(held, c) {
			commits = append(commits, c)
			if p, ok := sp.patchOf[c]; ok {
				patches = append(patches, p)
			}
		}
	}
	for _, c := range sp.Fresh {
		if !slices.Contains(held, c) {
			left = append(left, c)
		}
	}
	// The diff a lens reads starts from the base that commit holds: a merge
	// of the base left past the cap is not read as the base's code undone.
	base := st.Base
	if _, err := git(repo, "merge-base", "--is-ancestor", st.Base, upto); err != nil {
		mb, err := git(repo, "merge-base", st.Base, upto)
		if err != nil {
			return err
		}
		base = strings.TrimSpace(mb)
	}
	st.Before = ""
	switch {
	case st.From == st.Base && base != st.Base:
		st.Before = base
	case st.From != st.Base:
		if st.Before, err = mergedIn(repo, base, st.From); err != nil {
			st.Advisories = append(st.Advisories, verdict.Finding{Rule: "base-not-set-apart", Level: "warn",
				Message: "the base's code the new commits merged in could not be set apart (" + err.Error() + "): the lens is given it too"})
		}
	}
	st.Upto, st.Fresh, st.Commits, st.Patches, st.Left = upto, len(sp.Fresh), commits, patches, left
	names := make([]string, len(left))
	for i, c := range left {
		names[i] = short(c)
	}
	st.Advisories = append(st.Advisories, verdict.Finding{Rule: "commits-left", Level: "warn",
		Message: fmt.Sprintf("%d new commits change code, past the %d the lenses that warn read a push (commits-per-push): the oldest %d read; the %d left, %s, are read first at the next push, or when this run runs again on the same head",
			n, s.CommitsPerPush, s.CommitsPerPush, len(left), strings.Join(names, ", "))})
	return nil
}

// leftSaid is what a summary says of the commits left past commits-per-push.
func (st state) leftSaid() string {
	if st.Upto == "" {
		return ""
	}
	return fmt.Sprintf("; %d of the %d new commits read, %d left for the next push", st.Fresh-len(st.Left), st.Fresh, len(st.Left))
}

// sameAs is what a finding's judge stands on, as its line keeps it
// (Answered): Patch, the patch id of the merge request's change to the
// cause's file, base..head, the same whatever commit it sits on, so a
// rebase or a merge of the base keeps it; the file's blob at the head when
// the change leaves it. Finding keys the finding: its lens, its cause and
// claim as quoted, spaces aside, its symptom, and what else its judge
// reads that the patch does not hold: the symptom's file, the tests a
// tests judge reads, the issues an intent judge reads. A cause quoted from
// an issue: the whole change's patch id, the issue's text in the key.
// Nothing when git cannot tell: the finding is judged.
func sameAs(repo string, st state, s Settings, f Finding, l Lens) Answered {
	h := sha256.New()
	put := func(parts ...string) {
		for _, p := range parts {
			h.Write([]byte(p))
			h.Write([]byte{0})
		}
	}
	put(f.Lens, f.Cause.Path, norm(f.Cause.Quote), norm(f.Claim))
	var patch string
	if is := issueOf(st, f.Cause.Path); is != nil {
		patch = patchOf(repo, st.Base, st.Head, "")
		put(is.Text)
	} else {
		patch = patchOf(repo, st.Base, st.Head, f.Cause.Path)
	}
	if patch == "" {
		return Answered{}
	}
	if f.Symptom != nil {
		put(f.Symptom.Path, norm(f.Symptom.Quote))
		if f.Symptom.Path != f.Cause.Path {
			sym := patchOf(repo, st.Base, st.Head, f.Symptom.Path)
			if sym == "" {
				return Answered{}
			}
			put(sym)
		}
	}
	switch l.Judge.Reads {
	case "tests":
		if files := testFiles(repo, st.Head, f.Cause.Path, s.Tests); len(files) > 0 {
			blobs, err := git(repo, append([]string{"ls-tree", st.Head, "--"}, files...)...)
			if err != nil {
				return Answered{}
			}
			put(blobs)
		}
	case "issue":
		put(whatFor(st.Issues))
	}
	return Answered{Patch: patch, Finding: hex.EncodeToString(h.Sum(nil)[:6])}
}

// patchOf is the patch id of the change base..head to a file, or to every
// file when none is named, read from `git diff-tree` as the record's are
// (#286); the file's blob at head when the change leaves it; "" when git
// cannot tell.
func patchOf(repo, base, head, file string) string {
	args := []string{"diff-tree", "-p", "--no-color", "--no-renames", base, head}
	if file != "" {
		args = append(args, "--", file)
	}
	diff, err := git(repo, args...)
	if err != nil {
		return ""
	}
	if strings.TrimSpace(diff) == "" {
		if file == "" {
			return ""
		}
		blob, err := git(repo, "rev-parse", "--verify", "-q", head+":"+file)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(blob)
	}
	out, err := gitIn(repo, diff, "patch-id", "--verbatim")
	if err != nil {
		return ""
	}
	id, _, _ := strings.Cut(strings.TrimSpace(out), " ")
	return id
}

// judgedBefore is the last verdict a record holds on the same finding on
// the same patch, by its lens: a no, or a yes at an independence
// judge-at-least still takes. A line written before the keys holds none:
// never the same.
func judgedBefore(rec Record, lens string, same Answered, floor string) (Answered, bool) {
	if same.Patch == "" || same.Finding == "" {
		return Answered{}, false
	}
	if floor == "" {
		floor = "context"
	}
	for _, j := range slices.Backward(rec.Judged) {
		if j.Lens != lens || j.Patch != same.Patch || j.Finding != same.Finding {
			continue
		}
		if j.Yes && judge.Levels[j.Level] < judge.Levels[floor] {
			return Answered{}, false
		}
		return j, true
	}
	return Answered{}, false
}
