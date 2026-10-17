package reviewer

// A spec on the forge (ADR-0020, #128): in the gardening line, after the
// product owner, the reviewer reads the issues it refined, up to
// `specs-per-run` a run (#433): four sections written, not ready, the body
// not read as it is. What it finds on each goes to one comment on that
// issue, which hides a record the product owner reads back: ready is held
// while an important finding is open; the product owner answers at its
// next refine; past `spec-rounds` reviews in a row with findings open, the
// next is a question to a person, and the reviewer stops reading it.

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

// waiting is an issue whose spec waits on a review, and what the reviewer
// recorded on it before.
type waiting struct {
	issue  forge.Issue
	prior  *backlog.SpecReview
	before string // the reviewer's comment as it reads, its findings carried to a person
}

// waitingSpecs finds, in the backlog's order, the issues whose spec waits
// on a review: kept by the product owner, not ready, their four sections
// written (drafts too), their body not read as it is, their rounds not
// stopped. One whose record does not read is passed over and said (#302):
// read again as round 1, it cost an agent call a night, and the issues
// after it waited.
func waitingSpecs(f forge.Backlog) (ws []waiting, unread []verdict.Finding, err error) {
	open, err := f.Issues()
	if err != nil {
		return nil, nil, err
	}
	backlog.Order(open)
	for _, is := range open {
		if slices.Contains(is.Labels, backlog.LabelReady) || len(backlog.NotReady(is.Body, nil, true)) > 0 {
			continue
		}
		comments, err := f.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil, unread, err
		}
		if _, kept, _ := backlog.ReadState(comments, keeper); !kept {
			continue
		}
		prior, err := backlog.ReadSpecReview(comments)
		if err != nil {
			if !backlog.Accepted(is) { // accepted: a person's yes, the record read by nobody
				unread = append(unread, verdict.Finding{Rule: backlog.SpecRecordUnread, Where: fmt.Sprintf("#%d", is.ID), Level: "warn", Message: backlog.SpecUnread(err)})
			}
			continue
		}
		if prior != nil && (prior.Stopped || prior.Body == backlog.BodyDigest(is.Body)) {
			continue
		}
		w := waiting{issue: is, prior: prior}
		for i := len(comments) - 1; i >= 0 && prior != nil; i-- {
			if strings.Contains(comments[i], backlog.SpecMarker) {
				w.before = comments[i]
				break
			}
		}
		ws = append(ws, w)
	}
	return ws, unread, nil
}

// round is the review this one is: one more after a review that left an
// important finding open, else the first.
func round(prior *backlog.SpecReview) int {
	if prior != nil && prior.Open > 0 {
		return prior.Round + 1
	}
	return 1
}

// queue is what a spec run finds on the forge: the specs it reads, at most
// specs-per-run, in the backlog's order; the comments putting to a person
// those whose rounds are spent, which ask no agent and take no place; and
// what the run says: the issues passed over for a record that does not
// read, the rounds spent, the specs left for the next run.
type queue struct {
	read    []Spec
	stops   []intent.Intention
	stopped []string // what the summary says of each issue put to a person
	rest    []string // the specs left for the next run
	said    []verdict.Finding
}

// specOnForge is a spec run's first step on the forge: the issues that
// wait, read as specs; none, nothing asked; only issues whose rounds are
// spent, a question to a person on each and no agent. The issues passed
// over for a record that does not read are said, whatever the run does
// next.
func specOnForge(runDir, repo string, s Settings) (queue, int, bool) {
	var q queue
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return q, fail(fmt.Errorf("the issues to read: the forge: %v", err)), false
	}
	b, ok := f.(forge.Backlog)
	if f == nil || !ok {
		return q, final(runDir, Review{Subject: "spec", Status: verdict.Pass, Summary: "no forge: no refined issue to read",
			Findings: []verdict.Finding{{Rule: "no-forge", Level: "warn", Message: "on " + gardening + ", the reviewer reads the issues the product owner refined; " + forge.Missing}}}), false
	}
	ws, unread, err := waitingSpecs(b)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return q, exitExternal, false
	}
	if err != nil {
		return q, fail(err), false
	}
	if len(ws) == 0 {
		return q, final(runDir, Review{Subject: "spec", Status: verdict.Pass, Summary: "no refined issue waits on a spec review", Findings: unread}), false
	}
	q.said = unread
	var stoppedAt []string
	for _, w := range ws {
		n := round(w.prior)
		where := fmt.Sprintf("#%d", w.issue.ID)
		switch {
		case n > s.SpecRounds:
			// The rounds spent: a person decides, the reviewer stops (#128).
			rec := *w.prior
			rec.Stopped = true
			stoppedAt = append(stoppedAt, where)
			q.stopped = append(q.stopped, fmt.Sprintf("%s: %d important %s still open after %d rounds: put to a person, not read again", where, rec.Open, plural(rec.Open, "finding", "findings"), rec.Round))
			q.said = append(q.said, verdict.Finding{Rule: "spec-rounds-spent", Where: where, Level: "warn",
				Message: fmt.Sprintf("its spec answered %d times, %d important %s still open: a question for a person, on the issue; ready stays held until they decide", rec.Round, rec.Open, plural(rec.Open, "finding", "findings"))})
			if s.ForgeWrites {
				q.stops = append(q.stops, intent.Intention{Kind: "comment", Value: map[string]any{"issue": w.issue.ID, "sticky": backlog.SpecKey, "body": stopComment(w, rec, s.SpecRounds)}})
			}
		case len(q.read) < s.SpecsPerRun:
			q.read = append(q.read, Spec{Path: where, Title: strings.TrimSpace(w.issue.Title), Text: w.issue.Body, Issue: w.issue.ID,
				Digest: backlog.BodyDigest(w.issue.Body), Round: n, Prior: w.prior})
		default:
			q.rest = append(q.rest, where)
		}
	}
	if n := len(q.rest); n > 0 {
		// Left as they are: the next run takes the queue again in the
		// backlog's order, so these come first unless one is put before.
		q.said = append(q.said, verdict.Finding{Rule: "spec-waits", Level: "warn",
			Message: fmt.Sprintf("%d more %s on a review, read first at the next run (specs-per-run %d): %s", n, plural(n, "spec waits", "specs wait"), s.SpecsPerRun, strings.Join(q.rest, ", "))})
	}
	if len(q.read) > 0 {
		return q, 0, true
	}
	// Only questions to a person: no agent asked.
	if len(q.stops) > 0 {
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), q.stops); err != nil {
			return q, fail(err), false
		}
	}
	v := Review{Subject: "spec", Spec: strings.Join(stoppedAt, ", "), Status: verdict.Pass, Summary: strings.Join(q.stopped, "; "), Findings: q.said}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return q, fail(err), false
	}
	return q, 0, false
}

// settleSpecs settles each spec a run read on the forge apart, as if alone:
// its verdict, its comment and its record (#433). A spec none of whose
// lenses got an answer, the agent gone or the run's tokens spent before
// it, is not read: nothing written on it, its record as it was, said, and
// read first at the next run. The run's verdict holds every spec's
// findings, each where it lies.
func settleSpecs(runDir, repo string, s Settings, run state) int {
	fallback := slices.Clone(run.Stops)
	v := Review{Subject: "spec", Findings: slices.Clone(run.Advisories), Complete: true}
	var vs []Review
	var paths, notRead []string
	for _, sub := range run.Specs {
		var c candidates
		if err := readJSON(filepath.Join(runDir, "in", candidatesFile(sub)), &c); err != nil {
			return fail(err)
		}
		if why := noAnswer(runDir, sub); why != "" {
			v.Complete = false
			notRead = append(notRead, sub.Spec.Path)
			v.Findings = append(v.Findings, verdict.Finding{Rule: "spec-not-read", Where: sub.Spec.Path, Level: "warn",
				Message: "not read: " + why + "; nothing written on it, read first at the next run"})
			continue
		}
		sv, fb, err := settled(runDir, repo, s, sub, c)
		if err != nil {
			return fail(err)
		}
		vs, paths, fallback = append(vs, sv), append(paths, sv.Spec), append(fallback, fb...)
		for _, f := range sv.Findings {
			if f.Where == "" {
				f.Where = sv.Spec // a lens failed, a finding dropped: on this spec
			}
			v.Findings = append(v.Findings, f)
		}
		v.Change, v.Questions = append(v.Change, sv.Change...), append(v.Questions, sv.Questions...)
		v.Calls, v.Reads = append(v.Calls, sv.Calls...), append(v.Reads, sv.Reads...)
		for _, l := range sv.Lenses {
			if !slices.Contains(v.Lenses, l) {
				v.Lenses = append(v.Lenses, l)
			}
		}
		v.Complete = v.Complete && sv.Complete
	}
	v.Spec = strings.Join(paths, ", ")
	v.Status = status(v.Findings)
	v.Summary = specsSummary(run, vs, notRead, v, s)
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// waitingQuestions says, of a run the agent could not be reached for, that
// its questions to a person wait: an outside failure applies nothing of
// the run, though they need no agent. Their record asks them again at the
// next run.
func waitingQuestions(v *Review) {
	var at []string
	for i, f := range v.Findings {
		if f.Rule == "spec-rounds-spent" {
			at = append(at, f.Where)
			v.Findings[i].Message = "its question to a person waits for the next run: the agent could not be reached, and nothing of this run is written; ready stays held"
		}
	}
	if len(at) > 0 {
		v.Summary += fmt.Sprintf("; the %s to a person %s for the next run (%s)", plural(len(at), "question", "questions"), plural(len(at), "waits", "wait"), strings.Join(at, ", "))
	}
}

// noAnswer says why none of a spec's lenses got an answer for want of the
// agent: it could not be reached, or the run spent its tokens before them;
// "" when one answered, or failed otherwise (its answer did not read).
func noAnswer(runDir string, st state) string {
	why := ""
	for _, l := range facets(st) {
		data, err := os.ReadFile(filepath.Join(lensDir(runDir, st, l), "unanswered"))
		if err != nil {
			return ""
		}
		kind, said, _ := strings.Cut(strings.TrimSpace(string(data)), ": ")
		if kind != "spent" && kind != "unavailable" {
			return ""
		}
		why = strings.TrimPrefix(said, "not asked: ")
	}
	return why
}

// specsSummary is what a run reading specs on the forge says: one spec
// read, as when it is alone; several, each in a few words, then the lenses
// and the tokens of them all; then the specs not read, those put to a
// person, those left for the next run.
func specsSummary(run state, vs []Review, notRead []string, v Review, s Settings) string {
	var b strings.Builder
	switch {
	case len(run.Specs) == 1 && len(vs) == 1:
		b.WriteString(vs[0].Summary)
	case len(vs) == 0:
		b.WriteString("no spec read")
	default:
		var each []string
		for _, sv := range vs {
			line := "no finding"
			if n := len(sv.Change); n > 0 {
				line = fmt.Sprintf("%d %s", n, plural(n, "finding", "findings"))
			}
			if n := len(sv.Questions); n > 0 {
				line += fmt.Sprintf(", %d %s for a person", n, plural(n, "question", "questions"))
			}
			if !sv.Complete {
				line += ", not read whole"
			}
			each = append(each, sv.Spec+": "+line)
		}
		fmt.Fprintf(&b, "%d %s read (%s); lenses: %s%s", len(vs), plural(len(vs), "spec", "specs"), strings.Join(each, "; "), strings.Join(v.Lenses, ", "), tokensLine(v.Calls, s.AIMaxTokens))
	}
	if n := len(notRead); n > 0 {
		fmt.Fprintf(&b, "; %d not read, the agent asking nothing more: %s", n, strings.Join(notRead, ", "))
	}
	return b.String() + afterSpecs(run)
}

// afterSpecs is what a summary says after the specs read: those put to a
// person, those left for the next run.
func afterSpecs(run state) string {
	var b strings.Builder
	for _, l := range run.Stopped {
		b.WriteString("; " + l)
	}
	if n := len(run.Waits); n > 0 {
		fmt.Fprintf(&b, "; %d more %s for the next run", n, plural(n, "waits", "wait"))
	}
	return b.String()
}

// specRecord is what the comment on the issue records of this review: the
// body read, its round, the important findings open and the sections they
// lie in. A review not whole leaves the record as it was, its body unread:
// the spec is read again at the next run, the rounds counted as before.
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
		r = backlog.SpecReview{}
		if sp.Prior != nil {
			r = *sp.Prior
		}
		r.Body = ""
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
			r.Open, plural(r.Open, "finding", "findings"), r.Round, rounds, strings.Join(r.In, ", "), backlog.LabelAccepted)
	default:
		b.WriteString("No important finding open: the product owner may move it to ready.\n\n")
	}
	// The issues passed over for a record that does not read are the run's
	// to say, never this issue's.
	v.Findings = slices.DeleteFunc(slices.Clone(v.Findings), func(f verdict.Finding) bool { return f.Rule == backlog.SpecRecordUnread })
	b.WriteString(findingsTable(v, "No finding on the spec.", nil))
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
func stopComment(w waiting, r backlog.SpecReview, rounds int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**workline reviewer**, its spec — %d important %s still open after %d rounds between the product owner and the reviewer.\n\n",
		r.Open, plural(r.Open, "finding", "findings"), r.Round)
	fmt.Fprintf(&b, "**Question for a person**: is the spec good enough to build? Accept it as it reads with `%s`; or settle the findings below yourself, editing the spec, then set `%s`; or delete this comment to give the product owner and the reviewer %d rounds more. Until then the reviewer does not read it again, and it is not moved to ready.\n\n",
		backlog.LabelAccepted, backlog.LabelReady, rounds)
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
