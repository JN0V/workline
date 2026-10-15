package documentalist

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// The agent reads the pages the reader checks flagged (#256, ADR-0047):
// per page, what a reader would miss or skip, each passage quoted, and a
// patch making it easy to read. Last in a gardening run, at most
// `reader.agent-pages` pages a run within one task; a page is not read
// again until it changes, its header recording the body it was read as
// (`reader:`). What the agent answers never blocks: a claim quoting what
// the page does not hold is dropped, and a patch the engine cannot check
// against the claims is withheld, each said for a person.

// readerRules are the reader checks whose page the agent reads. A page
// only out of reach is fixed on another page: it is not read for that.
var readerRules = []string{"reference-unlinked", "paragraph-too-long", "cell-too-long", "flow-without-diagram"}

// readerShown are the findings a page's task shows the agent.
var readerShown = append(slices.Clone(readerRules), "page-unreachable")

// readerTask is what pre records of the reader task, for post to judge.
type readerTask struct {
	Pages []string `yaml:"pages"`
	Key   string   `yaml:"key"`
}

// readerRead is the fallback telling post the agent answered the reader
// task: the engine writes a fallback marked if-answered only with an
// answer that reads, and keeps one marked own beside the agent's notes.
// Post takes it out: it is never applied.
var readerRead = intent.Intention{Kind: "note", Value: map[string]any{"own": true, "if-answered": true, "reader-read": true}}

// readerAnswered says whether the intentions hold readerRead.
func readerAnswered(intents []intent.Intention) bool {
	for _, in := range intents {
		if m, ok := in.Value.(map[string]any); ok && in.Kind == "note" && m["reader-read"] == true {
			return true
		}
	}
	return false
}

// readerHash is what a page's record names: its body, the header aside.
func readerHash(content string) string {
	sum := sha256.Sum256([]byte(body(content)))
	return hex.EncodeToString(sum[:6])
}

var readerLine = regexp.MustCompile(`^reader:\s*(\S+)\s*$`)

// readerRecord is the hash a page's header records it was read as, or "".
func readerRecord(content string) string {
	meta, n := header(content)
	if n == 0 {
		return ""
	}
	for _, l := range strings.Split(meta, "\n") {
		if m := readerLine.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			return strings.Trim(m[1], `"'`)
		}
	}
	return ""
}

// withRecord writes in a page's header the hash of the body it was read
// as; a page with no header gets one, in a comment no rendering shows.
func withRecord(content, hash string) string {
	line := "reader: " + hash
	_, n := header(content)
	if n == 0 {
		return "<!-- workline\n" + line + "\n-->\n" + content
	}
	lines := strings.Split(content, "\n")
	for i := 1; i < n-1; i++ {
		if readerLine.MatchString(strings.TrimSpace(lines[i])) {
			lines[i] = line
			return strings.Join(lines, "\n")
		}
	}
	return strings.Join(slices.Insert(lines, n-1, line), "\n")
}

const readerTaskHeader = `Kind: reader

These pages are for people, and the reader checks flagged each one: what
they found is under its name. Read each page as its reader would — someone
looking for one thing, reading quickly — and say what they would miss or
skip.

For each passage a reader would miss or skip, one ` + "`claim`" + `, quoting the
page's words exactly as the lines shown here read:

    - claim:
        doc: docs/guide.md
        lines: "12-14"
        quote: "the passage's words, as they read"
        why: "what a reader misses or skips here, and why"

Then one ` + "`patch`" + ` making those passages easy to read: a wall of prose broken
into bullets, a decision, doc or issue named as a link, one short sentence a
table cell, a diagram opening a page that explains a flow. A unified diff
with context lines, whose hunks cite the lines shown here.

- Change only the lines your claims quote.
- Keep every link, code span, number and heading the page has.
- Reword, never add a fact: what the page says stays true.
- Leave the header as it is.

A patch the engine cannot check against your claims is withheld, for a
person. A page that serves its reader gets no claim.

`

// readerFor picks the pages the agent reads this run, and writes its task:
// the pages the reader checks flagged, in order, a page recorded as read
// as it is now or held by a waiting merge request left out, at most
// agent-pages within one task. What it leaves out is said on the page's
// findings. It returns nothing when no page is read: no agent, the
// setting at 0, nothing flagged, or an earlier round of this run read some.
func readerFor(findings []verdict.Finding, tree Tree, s Settings, w waiting, repo string) (*readerTask, string, []verdict.Finding, error) {
	if s.Reader.AgentPages <= 0 || os.Getenv("WORKLINE_AI") == "none" {
		return nil, "", findings, nil
	}
	shown := map[string][]verdict.Finding{}
	var cands []string
	for _, f := range findings {
		if !slices.Contains(readerShown, f.Rule) || tree.Docs[f.Where] == "" {
			continue
		}
		shown[f.Where] = append(shown[f.Where], f)
		if slices.Contains(readerRules, f.Rule) && !slices.Contains(cands, f.Where) {
			cands = append(cands, f.Where)
		}
	}
	sort.Strings(cands)
	note := func(p, text string) {
		for i := range findings {
			if findings[i].Where == p && slices.Contains(readerRules, findings[i].Rule) {
				findings[i].Message += "\n" + text
			}
		}
	}
	earlier, err := readThisRun(repo, tree, s.Reader, w)
	if err != nil {
		return nil, "", findings, err
	}
	if earlier {
		for _, p := range cands {
			if readerRecord(tree.Docs[p]) != readerHash(tree.Docs[p]) {
				note(p, "(read by the agent in a later run: this run read other pages already)")
			}
		}
		return nil, "", findings, nil
	}
	var picked []string
	var b strings.Builder
	b.WriteString(readerTaskHeader)
	for _, p := range cands {
		content := tree.Docs[p]
		if readerRecord(content) == readerHash(content) {
			note(p, "(read by the agent as it is now: not read again until it changes)")
			continue
		}
		if t := readerHeld(w, p); t != "" {
			note(p, "(workline/documentalist/"+t+" holds it and waits for review: not read again until it is merged or closed)")
			continue
		}
		block := readerBlock(p, content, shown[p])
		switch {
		case len(readerTaskHeader)+len(block) > taskChars(s):
			findings = append(findings, tooLargeToRewrite("read", p, len(readerTaskHeader)+len(block), taskChars(s)))
		case len(picked) >= s.Reader.AgentPages || b.Len()+len(block) > taskChars(s):
			note(p, fmt.Sprintf("(read by the agent in a later run: this run's task holds %d pages, at most %d (`reader.agent-pages`) within %d characters)", len(picked), s.Reader.AgentPages, taskChars(s)))
		default:
			picked = append(picked, p)
			b.WriteString(block)
		}
	}
	if len(picked) == 0 {
		return nil, "", findings, nil
	}
	return &readerTask{Pages: picked, Key: readerKey(picked, w)}, b.String(), findings, nil
}

// readerBlock is one page of the task: what the reader checks found, then
// the page with its line numbers.
func readerBlock(p, content string, found []verdict.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\nWhat the reader checks found:\n\n", p)
	for _, f := range found {
		fmt.Fprintf(&b, "- %s: %s\n", f.Rule, strings.SplitN(f.Message, "\n", 2)[0])
	}
	b.WriteString("\nThe page as it is now, with its line numbers:\n\n```\n")
	for i, l := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		fmt.Fprintf(&b, "%4d | %s\n", i+1, l)
	}
	b.WriteString("```\n\n")
	return b.String()
}

// readerHeld names the waiting merge request holding a page: one changing
// it, or a reader task whose changes could not be read.
func readerHeld(w waiting, p string) string {
	var tasks []string
	for t := range w.open {
		tasks = append(tasks, t)
	}
	for t := range w.proposed {
		if !w.open[t] {
			tasks = append(tasks, t)
		}
	}
	sort.Strings(tasks)
	for _, t := range tasks {
		if slices.Contains(w.files[t], p) || strings.HasPrefix(t, "reader") && !w.read(t) {
			return t
		}
	}
	return ""
}

// readThisRun says whether an earlier round of this run read pages: one
// proposed a reader task, or applied a page's record not committed yet.
// A run reads at most agent-pages pages; the rest wait for the next.
func readThisRun(repo string, tree Tree, r Reader, w waiting) (bool, error) {
	for t := range w.proposed {
		if strings.HasPrefix(t, "reader") {
			return true, nil
		}
	}
	pages := userPages(tree, r)
	if len(pages) == 0 {
		return false, nil
	}
	out, err := git(repo, append([]string{"diff", "HEAD", "--unified=0", "--no-color", "--"}, pages...)...)
	if err != nil {
		return false, err
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "+") && readerLine.MatchString(strings.TrimSpace(l[1:])) {
			return true, nil
		}
	}
	return false, nil
}

// readerKey is the reader task's merge request: named after its first
// page, a short hash added when one of that name already waits.
func readerKey(pages []string, w waiting) string {
	key := "reader " + pages[0]
	if _, waits, _ := w.holds(key); waits {
		sum := sha256.Sum256([]byte(strings.Join(pages, " ")))
		key += " " + hex.EncodeToString(sum[:3])
	}
	return key
}

// readerJudgement is what post makes of the agent's answer to a reader
// task: the patches the engine applies — each page as the kept fix leaves
// it, with its record —, what was dropped or withheld, said for a person,
// and what became of each page, for the merge request.
type readerJudgement struct {
	intents  []intent.Intention
	withheld []verdict.Finding
	outcome  map[string]string
	claims   map[string][]claim
}

// readerClaim is a claim of the reader task: a passage of a page a reader
// would miss or skip.
type readerClaim struct {
	Doc   string    `yaml:"doc"`
	Lines lineRange `yaml:"lines"`
	Quote string    `yaml:"quote"`
	Why   string    `yaml:"why"`
}

// judgeReader reads the agent's answer to a reader task. It never refuses:
// a claim not found at its lines is dropped; a page's patch changing lines
// no kept claim quotes, its header, or losing a link, a code span, a number
// or a heading, is withheld whole; each is said. Every page of the task is
// recorded as read, as the kept patch leaves it.
func judgeReader(repo string, r *readerTask, intents, fallback []intent.Intention) (readerJudgement, error) {
	j := readerJudgement{outcome: map[string]string{}, claims: map[string][]claim{}}
	old := map[string]string{}
	for _, p := range r.Pages {
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p)))
		if err != nil {
			return j, err
		}
		old[p] = string(data)
	}
	say := func(rule, where, msg string) {
		j.withheld = append(j.withheld, verdict.Finding{Rule: rule, Where: where, Message: msg})
	}
	for _, in := range intents {
		if in.Kind != "claim" {
			continue
		}
		data, _ := yaml.Marshal(in.Value)
		var c readerClaim
		if err := yaml.Unmarshal(data, &c); err != nil || c.Lines.from == 0 {
			say("reader-claim-dropped", "", "a claim of the reader task could not be read (doc, lines, quote, why): dropped")
			continue
		}
		if c.Doc == "" && len(r.Pages) == 1 {
			c.Doc = r.Pages[0]
		}
		switch {
		case !slices.Contains(r.Pages, c.Doc):
			say("reader-claim-dropped", c.Doc, fmt.Sprintf("a claim for %s %s names no page of the task: dropped", c.Doc, c.Lines))
		case !quotedAt(old[c.Doc], c.Lines, c.Quote):
			say("reader-claim-dropped", c.Doc, fmt.Sprintf("the claim for %s quotes %q, which the page does not hold there: dropped, never read as an answer", c.Lines, c.Quote))
		default:
			j.claims[c.Doc] = append(j.claims[c.Doc], claim{Lines: c.Lines, Quote: c.Quote, Why: c.Why})
		}
	}
	now := map[string]string{}
	for p, c := range old {
		now[p] = c
	}
	withheld := map[string]bool{}
	places := map[string]int{}
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue
		}
		diff, ok := in.Value.(string)
		if !ok {
			say("reader-patch-withheld", "", "the patch is not a unified diff: withheld")
			continue
		}
		files, err := parseDiff(diff)
		if err != nil {
			say("reader-patch-withheld", "", "the patch could not be read ("+err.Error()+"): withheld")
			continue
		}
		for _, f := range files {
			p := f.path
			if !slices.Contains(r.Pages, p) || f.deleted {
				say("reader-patch-withheld", p, "the reader task changes the pages it gives, and deletes none: this part of the patch is withheld")
				continue
			}
			if withheld[p] {
				continue
			}
			next, bad, why := applyDoc(now[p], f, diff)
			if bad == "" {
				bad, why = readerChecks(now[p], next, placed(now[p], f), j.claims[p])
			}
			if bad != "" {
				withheld[p] = true
				say("reader-patch-withheld", p, fmt.Sprintf("the proposal for %s is withheld, the page left as it was (%s): %s", p, bad, why))
				continue
			}
			places[p] += len(bodyBlocks(now[p], placed(now[p], f)))
			now[p] = next
		}
	}
	for _, p := range r.Pages {
		after := now[p]
		if withheld[p] {
			after = old[p]
		}
		final := withRecord(after, readerHash(after))
		switch {
		case withheld[p]:
			j.outcome[p] = "read; its proposal withheld, for a person (below)"
		case after != old[p]:
			j.outcome[p] = "rewritten in " + nPlaces(places[p]) + " for its reader"
		default:
			j.outcome[p] = "read; nothing proposed"
		}
		if final == old[p] {
			continue
		}
		d, err := unifiedDiff(p, old[p], final, 3)
		if err != nil {
			return j, err
		}
		j.intents = append(j.intents, intent.Intention{Kind: "patch", Value: d})
	}
	// The rest of the run's proposals stay — the engine's own, the agent's
	// notes —; the agent's patches are the pages above, its claims read.
	var kept []intent.Intention
	for _, in := range intents {
		fb := isFallback(in, fallback)
		if fb && !readerAnswered([]intent.Intention{in}) || !fb && in.Kind == "note" {
			kept = append(kept, in)
		}
	}
	j.intents = append(kept, j.intents...)
	return j, nil
}

// readerChecks checks a page's patch against the reader task's claims: no
// change to its header, every run of changed lines within lines a kept
// claim quotes, and every link, code span, number and heading the page had
// still there. It returns why not, or "".
func readerChecks(was, now string, f fileDiff, claims []claim) (rule, why string) {
	if before, _ := header(was); before != headerOf(now) {
		return "header", "the header changed; leave it as it is"
	}
	for _, b := range bodyBlocks(was, f) {
		at := lineRange{from: b.from, to: max(b.from, b.to)}
		if b.to < b.from { // lines only added, after b.from-1
			at = lineRange{from: max(1, b.from-1), to: b.from}
		}
		claimed := false
		for _, c := range claims {
			claimed = claimed || c.Lines.overlaps(at)
		}
		if !claimed {
			return "unclaimed", fmt.Sprintf("it changes %s, which no claim kept quotes", at)
		}
	}
	for _, kind := range []struct {
		name string
		re   *regexp.Regexp
	}{{"link", readerLink}, {"code span", readerCode}, {"number", readerNumber}, {"heading", readerHeading}} {
		left := map[string]int{}
		for _, m := range kind.re.FindAllStringSubmatch(body(now), -1) {
			left[group(m)]++
		}
		for _, m := range kind.re.FindAllStringSubmatch(body(was), -1) {
			if t := group(m); left[t] > 0 {
				left[t]--
			} else {
				return "lost", fmt.Sprintf("it loses the %s %q; keep every link, code span, number and heading the page has", kind.name, t)
			}
		}
	}
	return "", ""
}

var (
	readerLink    = regexp.MustCompile(`\]\(([^)\s]+)[^)]*\)|<(https?://[^>]+)>`)
	readerCode    = regexp.MustCompile("(`+[^`\n]+`+)")
	readerNumber  = regexp.MustCompile(`(?:^|[^\w.])(\d+(?:[.,]\d+)*)\b`)
	readerHeading = regexp.MustCompile(`(?m)^(#{1,6} .+?)\s*$`)
)

// headerOf is a doc's header, "" when it has none.
func headerOf(content string) string {
	h, _ := header(content)
	return h
}

// group is the first group a match took.
func group(m []string) string {
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return m[0]
}

// readerList is the merge request's account of what the agent found a
// reader would miss or skip, page by page.
func readerList(claims map[string][]claim) string {
	var b strings.Builder
	for _, p := range claimPages(claims) {
		for _, c := range claims[p] {
			fmt.Fprintf(&b, "- `%s`, %s: %s\n", p, c.Lines, strings.ReplaceAll(strings.TrimSpace(c.Why), "\n", " "))
		}
	}
	return b.String()
}

func claimPages(m map[string][]claim) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
