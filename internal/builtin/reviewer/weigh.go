package reviewer

// The tester's "none needed" weighed (#1303): a finding of a lens whose
// judge reads tests, sent to the tester, may come back answered that no
// test is needed, its reason on the merge request (tester.toReview). The
// reviewer has a judge weigh that reason, once: one that holds settles
// the finding — a block goes, a conversation on its line is resolved —
// one that does not leaves it, the judge's why said. A test the tester
// adds instead is read as any commit (readAgain).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// TesterAnswerKey starts the sticky key of the tester's answer to a
// test finding, on the pull request whose review found it, its thread's
// id after it (#1265).
const TesterAnswerKey = "tester-answer-"

// TesterAnswer is the tester's answer to a test finding of this review,
// found on found and titled title, its thread at url: what it did, text —
// a test added, or **None needed** and why. The title names the finding
// the answer is for, its line moved or another finding on the same line.
func TesterAnswer(found, title, url, text string) string {
	on := ""
	if found != "" {
		on = " on `" + found + "`"
		if t := strings.ReplaceAll(FindingTitle(title), "”", "\""); t != "" {
			on += ", “" + t + "”"
		}
	}
	return fmt.Sprintf("**The tester** answers a test finding of this review%s, sent to its Draft ([the thread](%s)): %s", on, url, text)
}

// FindingTitle is a finding's title as a row, its line or the record
// says it: its severity, its why, what it was grouped with and where it
// was found left out.
func FindingTitle(s string) string {
	s = strings.TrimPrefix(strings.TrimSpace(s), "Important:")
	for _, sep := range []string{" — ", "; also ", " (found "} {
		s, _, _ = strings.Cut(s, sep)
	}
	return strings.TrimSpace(s)
}

// noneNeededWords read a "none needed" answer as TesterAnswer says it:
// the place the review found the finding on, its title, then the reason.
var noneNeededWords = regexp.MustCompile("(?s)\\*\\*The tester\\*\\* answers a test finding of this review on `([^`]+)`(?:, “([^”]*)”)?, sent to its Draft \\(\\[the thread\\]\\([^)]*\\)\\): \\*\\*None needed\\*\\*: (.*)")

// Answer is the tester's answer that no test is needed for a finding:
// the place the review found it on, its title ("" from a tester before
// titles), and the reason.
type Answer struct {
	Where  string `json:"where"`
	Title  string `json:"title,omitempty"`
	Reason string `json:"reason"`
}

// NoneNeeded reads a comment of the tester's on a reviewed pull request
// (TesterAnswer): the finding it answers that no test is needed for, and
// its reason, the markers hidden after it left out. ok is false for any
// other comment.
func NoneNeeded(body string) (Answer, bool) {
	m := noneNeededWords.FindStringSubmatch(body)
	if m == nil {
		return Answer{}, false
	}
	reason, _, _ := strings.Cut(m[3], "<!-- workline:")
	return Answer{Where: m[1], Title: m[2], Reason: strings.TrimSpace(reason)}, true
}

// fileOf is the file of a place, path:line.
func fileOf(where string) string {
	if path, _, ok := fileLine(where); ok {
		return path
	}
	return where
}

// answerTo is the tester's reason, among answers, for the finding titled
// title at where: one naming it by its title, on its file, its line moved
// or not; one with no title, on its line. The last one said wins.
func answerTo(answers []Answer, where, title string) (string, bool) {
	reason, found := "", false
	for _, a := range answers {
		if a.Title != "" && (a.Title != FindingTitle(title) || fileOf(a.Where) != fileOf(where)) || a.Title == "" && a.Where != where {
			continue
		}
		reason, found = a.Reason, true
	}
	return reason, found
}

// Weighed is a tester's answer that no test is needed, as a judge
// weighed it: it holds, its finding settled, or it does not, why said.
// Answer keys the reason: a reason changed is weighed again. Title,
// with the file, names the finding: another on the same line is not
// settled by it, the same one, its line moved, is.
type Weighed struct {
	Lens   string `json:"lens"`
	Where  string `json:"where"`
	Title  string `json:"title,omitempty"`
	Answer string `json:"answer"`
	Holds  bool   `json:"holds"`
	Why    string `json:"why,omitempty"`
	Judge  string `json:"judge,omitempty"`
}

// weigh is an answer a run asks a judge to weigh, by Key, the judge's
// question.
type weigh struct {
	Lens   string `json:"lens"`
	Where  string `json:"where"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	Block  bool   `json:"block,omitempty"`
	Key    string `json:"key,omitempty"`
}

// weighQuestion is what the judge is asked of a tester's answer: yes,
// the finding still holds, as for any finding; no, the reason settles it.
const weighQuestion = "The reviewer's tests lens found a test missing at the place below; the tester, who writes the tests, answers that none is needed, for the reason quoted. " +
	"Does the finding still hold, the reason not settling it? Answer yes, saying why the reason does not hold; no, when it holds: a test would prove nothing the tests do not, or the behaviour is not the change's."

// answerKey keys a reason, its spaces aside.
func answerKey(reason string) string {
	sum := sha256.Sum256([]byte(strings.Join(strings.Fields(reason), " ")))
	return hex.EncodeToString(sum[:6])
}

// of says whether w is a verdict on lens's finding titled title at
// where: by its title on the same file, or, with no title, by its line.
func (w Weighed) of(lens, where, title string) bool {
	if w.Lens != lens {
		return false
	}
	if w.Title != "" {
		return w.Title == FindingTitle(title) && fileOf(w.Where) == fileOf(where)
	}
	return w.Where == where
}

// weighedOf is the verdict ws holds on lens's finding titled title at
// where, for the reason keyed answer.
func weighedOf(ws []Weighed, lens, where, title, answer string) (Weighed, bool) {
	for _, w := range ws {
		if w.of(lens, where, title) && w.Answer == answer {
			return w, true
		}
	}
	return Weighed{}, false
}

// settles says whether the tester's answer on the merge request, as a
// judge weighed it, settles lens's finding titled title at where: the
// answer said still, its reason the one that held.
func (st state) settles(lens, where, title string) bool {
	reason, ok := answerTo(st.Answers, where, title)
	if !ok {
		return false
	}
	w, ok := weighedOf(st.Record.Weighed, lens, where, title, answerKey(reason))
	return ok && w.Holds
}

// refusedAt is the last verdict ws holds on lens's finding titled title
// at where that the tester's answer does not settle.
func refusedAt(ws []Weighed, lens, where, title string) (Weighed, bool) {
	for i := len(ws) - 1; i >= 0; i-- {
		if w := ws[i]; w.of(lens, where, title) && !w.Holds {
			return w, true
		}
	}
	return Weighed{}, false
}

// unionWeighed is a's verdicts, then b's on an answer a does not hold.
func unionWeighed(a, b []Weighed) []Weighed {
	out := slices.Clone(a)
	for _, w := range b {
		if !slices.ContainsFunc(out, func(o Weighed) bool {
			return o.Lens == w.Lens && o.Where == w.Where && o.Title == w.Title && o.Answer == w.Answer
		}) {
			out = append(out, w)
		}
	}
	return out
}

// weighing reads the tester's answers that no test is needed on the
// merge request, and what the record says of them: a block one settled
// goes; an answer not weighed yet, on a block or on an open conversation
// of the reviewer's own, is put in Weigh, for a judge. With no agent,
// none is weighed, said; the findings stand.
func (st *state) weighing(repo string, s Settings) {
	t := mergeRequest()
	if t == nil || !s.ForgeWrites || st.Spec != nil {
		return
	}
	unread := func(err error) {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "tester-answers-unread", Level: "warn",
			Message: "the tester's answers on the merge request could not be read, none weighed: " + forge.Said(err)})
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil || f == nil {
		if err != nil {
			unread(err)
		}
		return
	}
	kept, err := forge.ReadKept(f, *t)
	if err != nil {
		unread(err)
		return
	}
	for _, c := range kept.Each("<!-- workline:sticky=tester/" + TesterAnswerKey) {
		if a, ok := NoneNeeded(c); ok && a.Reason != "" {
			st.Answers = append(st.Answers, a)
		}
	}
	if len(st.Answers) == 0 {
		return
	}
	lensOf := lensReader(repo)
	readsTests := func(lens string) bool {
		l, err := lensOf(lens)
		return err == nil && l.Judge.Reads == "tests"
	}
	asked := func(lens, where, title string) bool {
		return slices.ContainsFunc(st.Weigh, func(w weigh) bool {
			return w.Lens == lens && fileOf(w.Where) == fileOf(where) && FindingTitle(w.Title) == FindingTitle(title)
		})
	}
	var stands []Block
	for _, b := range st.Standing {
		reason, ok := answerTo(st.Answers, b.Where, b.Title)
		if !ok || !readsTests(b.Lens) {
			stands = append(stands, b)
			continue
		}
		if w, ok := weighedOf(st.Record.Weighed, b.Lens, b.Where, b.Title, answerKey(reason)); ok {
			if !w.Holds {
				stands = append(stands, b)
			}
			continue // settled before: it goes
		}
		stands = append(stands, b)
		if !asked(b.Lens, b.Where, b.Title) {
			st.Weigh = append(st.Weigh, weigh{Lens: b.Lens, Where: b.Where, Title: b.Title, Reason: reason, Block: true})
		}
	}
	st.Standing = stands
	if th, ok := f.(forge.Threaded); ok {
		threads, err := th.Threads(t.ID)
		if err != nil {
			unread(err)
			threads = nil
		}
		for _, t := range threads {
			if t.Resolved || len(t.Comments) == 0 || !kept.Wrote(t.Comments[0]) ||
				slices.ContainsFunc(t.Comments[1:], func(n forge.Note) bool { return !kept.Wrote(n) && !n.Bot }) {
				continue
			}
			keys := forge.LineKeys(t.Comments[0].Body)
			if len(keys) != 1 {
				continue
			}
			by, _, _ := strings.Cut(keys[0], "/")
			where := fmt.Sprintf("%s:%d", t.Path, t.Line)
			title, _ := lineTitle(t.Comments[0].Body)
			reason, ok := answerTo(st.Answers, where, title)
			if !ok || !readsTests(by) || !ofThisSide(s, *st, by) || asked(by, where, title) {
				continue
			}
			if _, ok := weighedOf(st.Record.Weighed, by, where, title, answerKey(reason)); ok {
				continue // its thread resolved by its verdict, or left (answered)
			}
			st.Weigh = append(st.Weigh, weigh{Lens: by, Where: where, Title: title, Reason: reason})
		}
	}
	if ai := os.Getenv("WORKLINE_AI"); len(st.Weigh) > 0 && (ai == "" || ai == "none") {
		n := len(st.Weigh)
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "tester-answers-unweighed", Level: "warn",
			Message: fmt.Sprintf("no agent: %d %s of the tester's that no test is needed not weighed, %s standing; a run with one (`--ai`) weighs %s, or a person does",
				n, plural(n, "answer", "answers"), plural(n, "its finding", "their findings"), plural(n, "it", "them"))})
		st.Weigh = nil
	}
}

// lineTitle is the title of a finding's comment on its line (lineBody).
func lineTitle(body string) (string, bool) {
	_, rest, ok := strings.Cut(body, "**Important: ")
	if !ok {
		return "", false
	}
	title, _, _ := strings.Cut(rest, "**")
	return "Important: " + title, true
}

// askWeighs writes the judge's question on each answer to weigh, beside
// the findings' (in/judge/weigh-NN).
func (st *state) askWeighs(runDir, repo string, s Settings) error {
	for i := range st.Weigh {
		w := &st.Weigh[i]
		w.Key = fmt.Sprintf("weigh-%02d", i+1)
		dir := filepath.Join(runDir, "in", "judge", w.Key)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		data, _ := json.Marshal(map[string]string{"question": weighQuestion, "material": weighMaterial(repo, *st, s, *w)})
		if err := os.WriteFile(filepath.Join(dir, "question.yaml"), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// weighMaterial is what the judge weighs an answer on: the finding, the
// tester's reason, the code around its line at the head, and the tests
// touching its file, as the tests lens's judge is shown them.
func weighMaterial(repo string, st state, s Settings, w weigh) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The finding\n\nThe %s lens, at `%s`: %s\n\n## The tester's answer: no test is needed\n\n%s\n", w.Lens, w.Where, w.Title, w.Reason)
	path, n, ok := fileLine(w.Where)
	if !ok {
		return b.String()
	}
	if text, ok := fileAt(repo, st.Head, path); ok {
		lines := strings.Split(text, "\n")
		from, to := max(1, n-15), min(len(lines), n+15)
		fmt.Fprintf(&b, "\n## `%s`, lines %d to %d\n\n```\n", path, from, to)
		for k := from; k <= to; k++ {
			fmt.Fprintf(&b, "%d  %s\n", k, lines[k-1])
		}
		b.WriteString("```\n")
	}
	b.WriteString(testsTouching(repo, st.Head, path, s.Tests, s.TestsLinesMax))
	return b.String()
}

// weighed reads the judges' answers on the tester's answers, each kept
// in the record: one that holds settles its finding, a block taken out
// of Standing; one that does not leaves it. What each says is returned,
// for the summary comment; an answer not weighed — no answer, the
// judge failing — is said, and weighed again next run.
func (st *state) weighed(runDir string) (said []string, settled int, unweighed []verdict.Finding) {
	for _, w := range st.Weigh {
		a, err := answerIn(filepath.Join(runDir, "in", "judge", w.Key))
		if err != nil || a.Yes == nil || a.Unsure {
			why := a.Error
			if why == "" {
				why = "the judge gave no answer"
			}
			unweighed = append(unweighed, verdict.Finding{Rule: "tester-answer-unweighed", Level: "warn", Where: w.Where,
				Message: fmt.Sprintf("the tester's answer that no test is needed was not weighed (%s): the %s finding stands, weighed next run", why, w.Lens)})
			continue
		}
		v := Weighed{Lens: w.Lens, Where: w.Where, Title: FindingTitle(w.Title), Answer: answerKey(w.Reason), Holds: !*a.Yes, Why: oneLine(a.Why, 200), Judge: a.Model}
		st.Record.Weighed = unionWeighed(st.Record.Weighed, []Weighed{v})
		what, verdict := "stands", "the reason does not hold"
		if v.Holds {
			what, verdict = "settled", "the reason holds"
			settled++
			st.Standing = slices.DeleteFunc(st.Standing, func(b Block) bool { return v.of(b.Lens, b.Where, b.Title) })
		}
		said = append(said, fmt.Sprintf("`%s` (%s, %s): %s, %s. The tester: “%s”. The judge (%s): %s.",
			w.Where, w.Lens, strings.TrimPrefix(w.Title, "Important: "), what, verdict, strings.TrimRight(oneLine(w.Reason, 400), "."), a.Model, strings.TrimRight(oneLine(a.Why, 300), ".")))
	}
	return said, settled, unweighed
}

// weighedSaid is the summary comment's section on the tester's answers
// a run weighed.
func weighedSaid(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("**The tester's answers weighed**: a finding the tester answered needs no test, its reason weighed by a judge.\n\n")
	for _, l := range lines {
		fmt.Fprintf(&b, "- %s\n", l)
	}
	return b.String() + "\n"
}

// settleHeld settles a run with nothing new to read whose judge weighed
// the tester's answers: the review held as it was, but for the findings
// they settle; on the merge request, its summary comment written again,
// the conversations they settle resolved.
func settleHeld(runDir, repo string, s Settings, st state) int {
	said, settled, unweighed := st.weighed(runDir)
	v := review(st, unweighed, st.Held)
	v.Weighed = said
	v.Summary += weighedSummary(len(said), settled)
	if mergeRequest() == nil || !s.ForgeWrites {
		return final(runDir, v)
	}
	fallback := []intent.Intention{summaryIntention(repo, s, st, &v, nil, 0, &lensesRead{})}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// weighedSummary is what a run's summary says of the n answers it
// weighed, settled of them settling their finding.
func weighedSummary(n, settled int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("; %d %s of the tester's that no test is needed weighed: %d settled", n, plural(n, "answer", "answers"), settled)
}
