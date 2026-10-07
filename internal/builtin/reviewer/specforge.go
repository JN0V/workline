package reviewer

// A spec on the forge (ADR-0020, #128): in the gardening line, after the
// product owner, the reviewer reads one issue it refined, a run: four
// sections written, not ready, its body not read as it is. What it finds
// goes to one comment on the issue, which hides a record the product owner
// reads back: ready is held while an important finding is open; the
// product owner answers at its next refine; past `spec-rounds` reviews in
// a row with findings open, the next is a question to a person, and the
// reviewer stops reading it.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// gardening is the event the reviewer reads the refined specs on: the
// gardening line's, after the product owner.
const gardening = "schedule"

// keeper is the role whose refined issues the reviewer reads on the forge:
// it keeps their state (docs/spec/backlog-acts.md).
const keeper = "product-owner"

// waiting is the issue a run reads on the forge, and what the reviewer
// recorded on it before.
type waiting struct {
	issue  forge.Issue
	prior  *backlog.SpecReview
	before string // the reviewer's comment as it reads, its findings carried to a person
}

// nextSpec finds the first issue in the backlog's order whose spec waits on
// a review: kept by the product owner, not ready, its four sections
// written (drafts too), its body not read as it is, its rounds not stopped.
// ok is false when none waits.
func nextSpec(f forge.Backlog) (w waiting, ok bool, err error) {
	open, err := f.Issues()
	if err != nil {
		return w, false, err
	}
	backlog.Order(open)
	for _, is := range open {
		if slices.Contains(is.Labels, backlog.LabelReady) || len(backlog.NotReady(is.Body, true)) > 0 {
			continue
		}
		comments, err := f.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return w, false, err
		}
		if _, kept, _ := backlog.ReadState(comments, keeper); !kept {
			continue
		}
		prior := backlog.ReadSpecReview(comments)
		if prior != nil && (prior.Stopped || prior.Body == backlog.BodyDigest(is.Body)) {
			continue
		}
		w = waiting{issue: is, prior: prior}
		for i := len(comments) - 1; i >= 0 && prior != nil; i-- {
			if strings.Contains(comments[i], backlog.SpecMarker) {
				w.before = comments[i]
				break
			}
		}
		return w, true, nil
	}
	return w, false, nil
}

// round is the review this one is: one more after a review that left an
// important finding open, else the first.
func round(prior *backlog.SpecReview) int {
	if prior != nil && prior.Open > 0 {
		return prior.Round + 1
	}
	return 1
}

// specOnForge is a spec run's first step on the forge: the issue that
// waits, read as the spec; none, nothing asked; its rounds spent, a
// question to a person and no agent.
func specOnForge(runDir, repo string, s Settings) (*Spec, *backlog.SpecReview, int, bool) {
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return nil, nil, fail(fmt.Errorf("the issues to read: the forge: %v", err)), false
	}
	b, ok := f.(forge.Backlog)
	if f == nil || !ok {
		return nil, nil, final(runDir, Review{Subject: "spec", Status: verdict.Pass, Summary: "no forge: no refined issue to read",
			Findings: []verdict.Finding{{Rule: "no-forge", Level: "warn", Message: "on " + gardening + ", the reviewer reads the issues the product owner refined; " + forge.Missing}}}), false
	}
	w, found, err := nextSpec(b)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, exitExternal, false
	}
	if err != nil {
		return nil, nil, fail(err), false
	}
	if !found {
		return nil, nil, final(runDir, Review{Subject: "spec", Status: verdict.Pass, Summary: "no refined issue waits on a spec review"}), false
	}
	n := round(w.prior)
	where := fmt.Sprintf("#%d", w.issue.ID)
	if n > s.SpecRounds {
		// The rounds spent: a person decides, the reviewer stops (#128).
		rec := *w.prior
		rec.Stopped = true
		v := Review{Subject: "spec", Spec: where, Status: verdict.Pass,
			Summary: fmt.Sprintf("%s: %d important findings still open after %d rounds: put to a person, not read again", where, rec.Open, rec.Round),
			Findings: []verdict.Finding{{Rule: "spec-rounds-spent", Where: where, Level: "warn",
				Message: fmt.Sprintf("its spec answered %d times, %d important findings still open: a question for a person, on the issue; ready stays held until they decide", rec.Round, rec.Open)}}}
		if s.ForgeWrites {
			fallback := []intent.Intention{{Kind: "comment", Value: map[string]any{"issue": w.issue.ID, "sticky": backlog.SpecKey, "body": stopComment(w, rec)}}}
			if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
				return nil, nil, fail(err), false
			}
		}
		if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
			return nil, nil, fail(err), false
		}
		return nil, nil, 0, false
	}
	sp := Spec{Path: where, Title: strings.TrimSpace(w.issue.Title), Text: w.issue.Body, Issue: w.issue.ID,
		Digest: backlog.BodyDigest(w.issue.Body), Round: n}
	return &sp, w.prior, 0, true
}

// specRecord is what the comment on the issue records of this review: the
// body read, its round, the important findings open and the sections they
// lie in. A review not whole leaves the body unread, its round uncounted:
// the spec is read again at the next run.
func specRecord(st state, v Review) backlog.SpecReview {
	sp := st.Spec
	r := backlog.SpecReview{Body: sp.Digest, Round: sp.Round}
	for _, g := range v.Change {
		ms := members(g)
		if !slices.ContainsFunc(ms, func(f Finding) bool { return f.Severity == "important" }) {
			continue
		}
		r.Open++
		_, at, _ := strings.Cut(g.Where, ":")
		line, _ := strconv.Atoi(at)
		if name := backlog.SectionAt(sp.Text, line); name != "" && !slices.Contains(r.In, name) {
			r.In = append(r.In, name)
		}
	}
	if !v.Complete {
		r.Body, r.Round = "", sp.Round-1
	}
	return r
}

// specComment is the one comment the issue gets, edited at each review:
// what holds it from ready, the findings, the questions, the record.
func specComment(v Review, r backlog.SpecReview, rounds int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**workline reviewer**, its spec — %s.\n\n", v.Summary)
	switch {
	case r.Body == "":
		b.WriteString("Not read whole: the reviewer reads it again at the next run; ready waits.\n\n")
	case r.Open > 0:
		fmt.Fprintf(&b, "**Holds it from ready**: %d important %s open, round %d of %d. The product owner answers at its next refine, rewriting what is its own in %s; the reviewer then reads the spec again. A person may accept the issue as it reads with `%s`.\n\n",
			r.Open, map[bool]string{true: "finding", false: "findings"}[r.Open == 1], r.Round, rounds, strings.Join(r.In, ", "), backlog.LabelAccepted)
	default:
		b.WriteString("No important finding open: the product owner may move it to ready.\n\n")
	}
	b.WriteString(findingsTable(v, "No finding on the spec."))
	if len(v.Questions) > 0 {
		b.WriteString("**Questions for a person**\n\nNot defects, not judged: choices the spec leaves to a person; a reply here is enough, they hold nothing.\n\n")
		for _, q := range v.Questions {
			fmt.Fprintf(&b, "- `%s` (%s): %s On: “%s”\n", q.Where, q.Lens, oneLine(q.Decision, 300), oneLine(q.Cause.Quote, 200))
		}
		b.WriteString("\n")
	}
	b.WriteString("The reviewer never edits the issue nor moves it: the product owner answers, a person decides.\n\n")
	b.WriteString(r.String())
	return b.String()
}

// stopComment is the comment once the rounds are spent: a question to a
// person, the last review's findings kept under it.
func stopComment(w waiting, r backlog.SpecReview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**workline reviewer**, its spec — %d important %s still open after %d rounds between the product owner and the reviewer.\n\n",
		r.Open, map[bool]string{true: "finding", false: "findings"}[r.Open == 1], r.Round)
	fmt.Fprintf(&b, "**Question for a person**: is the spec good enough to build? Accept it as it reads with `%s`; or settle the findings below yourself, editing the spec, then set `%s`; or delete this comment to give the product owner and the reviewer %s more. Until then the reviewer does not read it again, and it is not moved to ready.\n\n",
		backlog.LabelAccepted, backlog.LabelReady, "the same rounds")
	if prev := lastFindings(w.before); prev != "" {
		fmt.Fprintf(&b, "The last review (round %d):\n\n%s\n\n", r.Round, prev)
	}
	b.WriteString(r.String())
	return b.String()
}

// lastFindings is a review comment's findings and questions, its lead, its
// closing line and its record left out.
func lastFindings(comment string) string {
	var kept []string
	for _, l := range strings.Split(comment, "\n") {
		switch {
		case strings.HasPrefix(l, "**workline reviewer**"), strings.HasPrefix(l, "**Holds it from ready**"),
			strings.HasPrefix(l, "No important finding open"), strings.HasPrefix(l, "Not read whole"),
			strings.HasPrefix(l, "The reviewer never"), strings.Contains(l, "<!-- workline:"):
			continue
		}
		kept = append(kept, l)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}
