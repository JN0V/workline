package documentalist

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// A doc whose sources, as they are now, do not fit a task is judged in parts
// (ADR-0009): each part holds the doc whole, then a share of its source
// files, and answers with claims about the passages its share speaks to.
// The engine asks the parts; pre, run again, checks their quotes, puts them
// together without AI, and asks one last call to fix what they found wrong.
// `checked` does not move: nobody read the doc whole against its sources.

// Defaults, when the project sets none.
const (
	partsMaxDefault       = 8  // parts a doc may take; past it, its sources are too wide
	partsMaxPerRunDefault = 16 // parts asked in one run; the other docs wait for the next
)

// partFile is the share of a source file one part holds: lines from to to.
type partFile struct {
	Path  string `yaml:"path"`
	From  int    `yaml:"from"`
	To    int    `yaml:"to"`
	Lines int    `yaml:"lines"` // the whole file's
}

// partPlan is what pre records of a part beside its task, to read its answer.
type partPlan struct {
	Doc   string     `yaml:"doc"`
	Files []partFile `yaml:"files"`
}

// partsTaskHeader is what every part asks, before the doc and its share.
const partsTaskHeader = `Kind: part

This doc is too large to be judged whole against its sources, so it is judged
in parts: each part holds the doc whole, then a share of its sources. This
part holds only the share below; other parts hold the rest.

For each passage of the doc — a range of its lines — that this share of the
sources says something about, return one ` + "`claim`" + `:

- contradicted: the source says otherwise. Quote the doc's words, and the
  source's, with its path and lines; say why they disagree.
- partial: the source bears on it but does not settle it; say in ` + "`why`" + ` what
  this share leaves out.
- supported: the source says the same; quote both.

Say nothing of a passage this share does not speak to: it is not covered by
this share, and another part may hold what does. Quote exactly, as the lines
read, citing the line numbers shown here: a quote not found at the lines it
cites is dropped. Claims only: no patch, no note. If this share says nothing
of the doc, answer ` + "`[]`" + `.

Write each claim in block style, one field a line, as below — not on one line
between braces, where one brace too many spoils the answer:

    - claim:
        lines: "10-12"
        status: contradicted
        quote: "the doc's words, as they read"
        source:
          path: src/example.go
          lines: "7"
          quote: "the source's words, as they read"
        why: "why they disagree"

`

// fixTaskHeader asks the one call that fixes what the parts found.
const fixTaskHeader = `Kind: fix

These docs were too large to be judged whole against their sources, and were
judged in parts: each part read the doc with a share of its sources. Below,
for each doc, the passages the parts found wrong, those they disagree on, and
those to be read together, each with the source lines the parts cited.

Patch what is wrong, and nothing else: a unified diff with context lines,
whose hunks cite the doc's lines by the numbers shown here. Leave the header
as it is — ` + "`checked`" + ` does not move, since nobody read the doc whole. A passage
the excerpts do not let you settle stays as it is; say so in a note.

`

// partsFor judges in parts the docs too large to be judged whole, when the
// project turned it on. Asked first (no WORKLINE_PARTS), it writes each
// part's question under in/parts; run again with the parts answered, it puts
// their claims together, and returns the fix task, the docs it judges, the
// header each doc gains, and the docs judged in parts, by the commit they
// were judged at. Each doc's finding says what became of it (sd.note).
func partsFor(runDir, repo string, cands []*suspectDoc, s Settings, total *int, findings *[]verdict.Finding) (task string, judged map[string]map[string]string, fallback []intent.Intention, inParts map[string]string, err error) {
	answered := os.Getenv("WORKLINE_PARTS") == "answered"
	partsMax, perRun := orInt(s.PartsMax, partsMaxDefault), orInt(s.PartsMaxPerRun, partsMaxPerRunDefault)
	head, err := git(repo, "rev-parse", "HEAD")
	if err != nil {
		return "", nil, nil, nil, err
	}
	var fixes strings.Builder
	judged, inParts = map[string]map[string]string{}, map[string]string{}
	for i, sd := range cands {
		plans, why, err := planParts(sd.doc, repo, s.PartChars)
		if err != nil {
			return "", nil, nil, nil, err
		}
		switch {
		case why != "":
			sd.note = why
			continue
		case len(plans) > partsMax:
			*findings = append(*findings, verdict.Finding{Rule: "sources-too-wide", Where: sd.doc.Path,
				Message: fmt.Sprintf("judging it in parts would take %d parts, past parts-max (%d): its sources name more than a doc can be judged against; narrow them to the files it describes, or split the doc", len(plans), partsMax)})
			sd.note = "(too large to be judged whole, and its sources too wide to be judged in parts: a person judges it)"
			continue
		case *total+len(plans) > perRun:
			sd.deferred = true
			sd.note = fmt.Sprintf("(to be judged in parts, %d parts, in a later round: past parts-max-per-run, %d)", len(plans), perRun)
			continue
		}
		*total += len(plans)
		names := make([]string, len(plans))
		for k := range plans {
			names[k] = fmt.Sprintf("%02d-%s-%d", i+1, partSlug(sd.doc.Path), k+1)
		}
		if !answered {
			for k, p := range plans {
				if err := writePart(runDir, repo, names[k], p); err != nil {
					return "", nil, nil, nil, err
				}
			}
			sd.note = fmt.Sprintf("(judged in parts: %d parts asked)", len(plans))
			continue
		}
		verdictOf, err := readParts(runDir, repo, names, plans)
		if err != nil {
			return "", nil, nil, nil, err
		}
		if verdictOf.unanswered == len(plans) && os.Getenv("WORKLINE_AI") == "none" {
			sd.note = "(too large to be judged whole, and judged in parts only with an agent: a person judges it)"
			continue
		}
		if verdictOf.unanswered > 0 {
			sd.note = fmt.Sprintf("(judged in parts, but %d of its %d parts got no answer that can be read: nothing they found is kept, and it is judged again on a later run; until then, a person judges it)", verdictOf.unanswered, len(plans))
			continue
		}
		content, err := os.ReadFile(filepath.Join(repo, sd.doc.Path))
		if err != nil {
			return "", nil, nil, nil, err
		}
		if u := uncovered(string(content), verdictOf.kept); u != "" {
			*findings = append(*findings, verdict.Finding{Rule: "uncovered", Where: sd.doc.Path,
				Message: u + ", which no part of its sources says anything of: renamed or gone from the code, or a source missing from its header; read those lines against the code"})
		}
		if verdictOf.dropped > 0 {
			*findings = append(*findings, verdict.Finding{Rule: "claims-dropped", Where: sd.doc.Path, Level: "warn",
				Message: fmt.Sprintf("%d claims of its parts quoted what is not at the lines they cite, or cited what their part did not hold: dropped, never read as an answer", verdictOf.dropped)})
		}
		at := head[:7]
		patch, err := judgedInPartsPatch(sd.doc.Path, string(content), at)
		if err != nil {
			return "", nil, nil, nil, err
		}
		entry := verdictOf.fixEntry(sd.doc.Path, string(content), repo)
		summary := fmt.Sprintf("judged in parts at %s: %d parts, %d claims kept", at, len(plans), len(verdictOf.kept))
		switch {
		case entry == "":
			sd.note = "(" + summary + ", nothing found wrong. `checked` stays: nobody read it whole against its sources. It is not put before an agent again until one of them changes; a person reads it whole, then moves `checked`)"
		case fixes.Len()+len(entry)+len(fixTaskHeader)+len(citeTask) > taskMaxChars:
			sd.deferred = true
			sd.note = "(" + summary + "; what they found wrong does not fit this round's fix: judged in parts again in a later round)"
			continue
		default:
			fixes.WriteString(entry)
			judged[sd.doc.Path] = map[string]string{"": head}
			sd.note = "(" + summary + "; what they found wrong is fixed in this run, and `checked` stays: nobody read it whole against its sources. It is not put before an agent again until one of them changes; a person reads it whole, then moves `checked`)"
		}
		// A claim dropped may have been the one saying what is wrong: it is
		// said, for a person. The doc is recorded all the same: asked again
		// each night, its parts cost as much to the same end (ADR-0014, step
		// 4: 0.23M to 0.32M tokens a night); a source changing asks again.
		if verdictOf.dropped > 0 {
			sd.note = strings.TrimSuffix(sd.note, ")") + "; but some claims were dropped: a person reads it whole)"
		}
		fallback = append(fallback, intent.Intention{Kind: "patch", Value: patch})
		inParts[sd.doc.Path] = at
	}
	if fixes.Len() > 0 {
		task = fixTaskHeader + citeTask + fixes.String()
	}
	return task, judged, fallback, inParts, nil
}

// planParts cuts a doc's sources into parts: the doc whole in each, then
// source files packed to fill the task; a file larger than a part is cut
// into consecutive pieces, each read, so no line goes unread. Sources that
// are docs are left out: a doc following another is propagation, not
// evidence. why says why the doc cannot be judged in parts.
func planParts(d *Doc, repo string, share int) (plans []partPlan, why string, err error) {
	content, err := os.ReadFile(filepath.Join(repo, d.Path))
	if err != nil {
		return nil, "", err
	}
	room := taskMaxChars - len(partsTaskHeader) - len(numberedDoc(d.Path, string(content)))
	if room < taskMaxChars/4 {
		return nil, "(too large to be judged whole, and too long itself to be judged in parts: it leaves less than a quarter of a task for its sources; condense or split it first — a person judges it until then)", nil
	}
	if share > 0 && share < room { // measuring: smaller parts than a task holds
		room = share
	}
	var files []string
	for _, src := range d.Sources {
		name, p, _ := splitSource(src)
		if name != "" {
			return nil, "(too large to be judged whole; its sources in another repository are not judged in parts yet: a person judges it)", nil
		}
		if strings.HasSuffix(p, ".md") {
			continue
		}
		out, err := git(repo, "ls-tree", "-z", "-r", "--name-only", "HEAD", "--", p)
		if err != nil {
			return nil, "", err
		}
		for _, f := range pathList(out) {
			if f != "" && !strings.HasSuffix(f, ".md") {
				files = append(files, f)
			}
		}
	}
	cur := partPlan{Doc: d.Path}
	used := 0
	flush := func() {
		if len(cur.Files) > 0 {
			plans = append(plans, cur)
		}
		cur, used = partPlan{Doc: d.Path}, 0
	}
	for _, f := range files {
		text, err := showFile(repo, f)
		if err != nil {
			return nil, "", err
		}
		if strings.ContainsRune(text, 0) {
			continue // not text
		}
		lines := strings.Split(text, "\n")
		// size(from, to) is what numberedPiece takes, without building it.
		sum := make([]int, len(lines)+1)
		for i, l := range lines {
			sum[i+1] = sum[i] + len(fmt.Sprintf("%5d | ", i+1)) + len(l) + 1
		}
		size := func(from, to int) int {
			return len(fmt.Sprintf("## %s, lines %d-%d of %d\n\n```\n```\n\n", f, from, to, len(lines))) + sum[to] - sum[from-1]
		}
		// A file that fits a part of its own is not cut to fill this one.
		if used > 0 && used+size(1, len(lines)) > room && size(1, len(lines)) <= room {
			flush()
		}
		for from := 1; from <= len(lines); {
			to := from - 1
			for to < len(lines) && used+size(from, to+1) <= room {
				to++
			}
			if to < from { // not even a line fits what is left of this part
				if used > 0 {
					flush()
					continue
				}
				to = from // a line longer than a part is read alone
			}
			cur.Files = append(cur.Files, partFile{Path: f, From: from, To: to, Lines: len(lines)})
			used += size(from, to)
			if to < len(lines) {
				flush()
			}
			from = to + 1
		}
	}
	flush()
	if len(plans) == 0 {
		return nil, "(too large to be judged whole, and none of its sources is a file of code to judge it against in parts: a person judges it)", nil
	}
	return plans, "", nil
}

// numberedDoc is a doc as a part shows it, each line with its number.
func numberedDoc(p, content string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s, the doc judged, with its line numbers\n\n```\n", p)
	for i, l := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		fmt.Fprintf(&b, "%4d | %s\n", i+1, l)
	}
	b.WriteString("```\n\n")
	return b.String()
}

// numberedPiece is lines from to to of a source file, numbered as in the file.
func numberedPiece(p string, lines []string, from, to int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s, lines %d-%d of %d\n\n```\n", p, from, to, len(lines))
	for n := from; n <= to; n++ {
		fmt.Fprintf(&b, "%5d | %s\n", n, lines[n-1])
	}
	b.WriteString("```\n\n")
	return b.String()
}

// writePart writes one part's question, and what it holds, to read its answer.
func writePart(runDir, repo, name string, p partPlan) error {
	dir := filepath.Join(runDir, "in", "parts", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Join(repo, p.Doc))
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(partsTaskHeader)
	b.WriteString(numberedDoc(p.Doc, string(content)))
	for _, f := range p.Files {
		text, err := showFile(repo, f.Path)
		if err != nil {
			return err
		}
		b.WriteString(numberedPiece(f.Path, strings.Split(text, "\n"), f.From, f.To))
	}
	if err := os.WriteFile(filepath.Join(dir, "task.md"), []byte(b.String()), 0o644); err != nil {
		return err
	}
	return writeYAML(filepath.Join(dir, "part.yaml"), p)
}

// partSlug names a doc in a part's folder name.
func partSlug(p string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(p) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// lineRange is a range of lines, written "12", "12-14", or as a number.
type lineRange struct{ from, to int }

func (r *lineRange) UnmarshalYAML(n *yaml.Node) error {
	v := strings.ReplaceAll(strings.TrimSpace(n.Value), "–", "-")
	a, b, found := strings.Cut(v, "-")
	from, err1 := strconv.Atoi(strings.TrimSpace(a))
	to, err2 := from, error(nil)
	if found {
		to, err2 = strconv.Atoi(strings.TrimSpace(b))
	}
	if err1 != nil || err2 != nil || from < 1 || to < from {
		return fmt.Errorf("%q is not a range of lines like 12-14", n.Value)
	}
	r.from, r.to = from, to
	return nil
}

func (r lineRange) String() string {
	if r.from == r.to {
		return fmt.Sprintf("line %d", r.from)
	}
	return fmt.Sprintf("lines %d-%d", r.from, r.to)
}

func (r lineRange) overlaps(o lineRange) bool { return r.from <= o.to && o.from <= r.to }

func (r lineRange) covers(o lineRange) bool { return r.from <= o.from && o.to <= r.to }

// claim is a part's answer about one passage of the doc.
type claim struct {
	Lines  lineRange `yaml:"lines"`
	Status string    `yaml:"status"`
	Quote  string    `yaml:"quote"`
	Source *struct {
		Path  string    `yaml:"path"`
		Lines lineRange `yaml:"lines"`
		Quote string    `yaml:"quote"`
	} `yaml:"source"`
	Why  string `yaml:"why"`
	part int    // which part said it
}

// partsVerdict is what the parts of one doc said, once their quotes are checked.
type partsVerdict struct {
	kept       []claim
	dropped    int // claims whose quotes are not at the lines they cite
	unanswered int // parts with no answer
}

// readParts reads each part's answer and keeps the claims whose quotes are
// found, whitespace aside, at the lines they cite: the doc's in the doc, the
// source's in a file of that part's share, within the lines it holds.
func readParts(runDir, repo string, names []string, plans []partPlan) (*partsVerdict, error) {
	v := &partsVerdict{}
	doc, err := os.ReadFile(filepath.Join(repo, plans[0].Doc))
	if err != nil {
		return nil, err
	}
	sources := map[string]string{}
	for k, name := range names {
		if _, err := os.Stat(filepath.Join(runDir, "in", "parts", name, "answer.yaml")); err != nil {
			v.unanswered++
			continue
		}
		answers, err := intent.Read(filepath.Join(runDir, "in", "parts", name, "answer.yaml"))
		if err != nil {
			return nil, err
		}
		for _, a := range answers {
			data, _ := yaml.Marshal(a.Value)
			var c claim
			if a.Kind != "claim" || yaml.Unmarshal(data, &c) != nil || !c.valid() || !quotedAt(string(doc), c.Lines, c.Quote) {
				v.dropped++
				continue
			}
			if c.Source != nil {
				src := path.Clean(strings.TrimPrefix(c.Source.Path, "./"))
				held := false
				for _, f := range plans[k].Files {
					held = held || f.Path == src && f.From <= c.Source.Lines.from && c.Source.Lines.to <= f.To
				}
				if _, ok := sources[src]; !ok && held {
					sources[src], _ = showFile(repo, src)
				}
				if !held || !quotedAt(sources[src], c.Source.Lines, c.Source.Quote) {
					v.dropped++
					continue
				}
				c.Source.Path = src
			}
			c.part = k
			v.kept = append(v.kept, c)
		}
	}
	return v, nil
}

// valid says whether a claim has what its status needs: a quote of the doc,
// and of the source for a contradiction or a support.
func (c claim) valid() bool {
	switch c.Status {
	case "contradicted", "supported":
		return c.Quote != "" && c.Source != nil && c.Source.Quote != ""
	case "partial":
		return c.Quote != "" && (c.Source == nil || c.Source.Quote != "")
	}
	return false
}

// quotedAt says whether quote is found in the lines r cites, whitespace aside.
func quotedAt(content string, r lineRange, quote string) bool {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	q := strings.Join(strings.Fields(quote), " ")
	if q == "" || r.from < 1 || r.to > len(lines) {
		return false
	}
	return strings.Contains(strings.Join(strings.Fields(strings.Join(lines[r.from-1:r.to], " ")), " "), q)
}

// fixEntry puts the parts' claims together, without AI, by the doc's lines:
// contradicted and supported by no other part is wrong; contradicted in one
// part and supported in another is a conflict; partial is to be read
// together, unless another part supports the lines it speaks of: that part
// held the source settling them, and the fix would act on a share that
// cannot (workline #29). It returns the doc's part of the fix task, with only the source
// lines the parts cited; nothing when there is nothing to fix.
func (v *partsVerdict) fixEntry(p, content, repo string) string {
	var wrong, conflict, together []claim
	for _, c := range v.kept {
		switch c.Status {
		case "contradicted":
			disputed := false
			for _, o := range v.kept {
				disputed = disputed || o.Status == "supported" && o.part != c.part && o.Lines.overlaps(c.Lines)
			}
			if disputed {
				conflict = append(conflict, c)
			} else {
				wrong = append(wrong, c)
			}
		case "partial":
			settled := false
			for _, o := range v.kept {
				settled = settled || o.Status == "supported" && o.part != c.part && o.Lines.covers(c.Lines)
			}
			if !settled {
				together = append(together, c)
			}
		}
	}
	var b strings.Builder
	if len(wrong)+len(conflict)+len(together) == 0 {
		return ""
	}
	fmt.Fprintf(&b, "## %s\n\n", p)
	sources := map[string]string{}
	show := func(title string, list []claim) {
		if len(list) == 0 {
			return
		}
		fmt.Fprintf(&b, "### %s\n\n", title)
		sort.SliceStable(list, func(i, j int) bool { return list[i].Lines.from < list[j].Lines.from })
		for _, c := range list {
			fmt.Fprintf(&b, "- The doc, %s: %q\n", c.Lines, c.Quote)
			if c.Source != nil {
				fmt.Fprintf(&b, "  %s, %s: %q\n", c.Source.Path, c.Source.Lines, c.Source.Quote)
			}
			if c.Why != "" {
				fmt.Fprintf(&b, "  Why: %s\n", strings.ReplaceAll(strings.TrimSpace(c.Why), "\n", " "))
			}
			if title == "Found wrong by one part, and supported by another" {
				for _, o := range v.kept {
					if o.Status == "supported" && o.part != c.part && o.Lines.overlaps(c.Lines) {
						fmt.Fprintf(&b, "  Supported by %s, %s: %q\n", o.Source.Path, o.Source.Lines, o.Source.Quote)
					}
				}
			}
		}
		b.WriteString("\n")
	}
	show("Found wrong", wrong)
	show("Found wrong by one part, and supported by another", conflict)
	show("To be read together: each part holds only a share of what they say", together)
	// Only the source lines the parts cited, with a few lines around them.
	cited := map[string][]lineRange{}
	for _, list := range [][]claim{wrong, conflict, together} {
		for _, c := range list {
			if c.Source != nil {
				cited[c.Source.Path] = append(cited[c.Source.Path], c.Source.Lines)
			}
		}
	}
	for _, c := range conflict {
		for _, o := range v.kept {
			if o.Status == "supported" && o.part != c.part && o.Lines.overlaps(c.Lines) {
				cited[o.Source.Path] = append(cited[o.Source.Path], o.Source.Lines)
			}
		}
	}
	files := make([]string, 0, len(cited))
	for f := range cited {
		files = append(files, f)
	}
	sort.Strings(files)
	if len(files) > 0 {
		b.WriteString("The source lines the parts cited, with a few lines around them:\n\n")
	}
	for _, f := range files {
		if _, ok := sources[f]; !ok {
			sources[f], _ = showFile(repo, f)
		}
		lines := strings.Split(sources[f], "\n")
		ranges := cited[f]
		sort.Slice(ranges, func(i, j int) bool { return ranges[i].from < ranges[j].from })
		for _, r := range ranges {
			from, to := max(1, r.from-3), min(len(lines), r.to+3)
			b.WriteString("#" + numberedPiece(f, lines, from, to)) // under the doc's heading
		}
	}
	b.WriteString("The doc as it is now, with its line numbers:\n\n```\n")
	for i, l := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		fmt.Fprintf(&b, "%4d | %s\n", i+1, l)
	}
	b.WriteString("```\n\n")
	return b.String()
}

// uncovered names the lines of the doc naming a name from the code that no
// claim says anything of, with those names; empty when there are none.
func uncovered(content string, kept []claim) string {
	var out []string
	for _, l := range scan(content) {
		if l.code {
			continue
		}
		var ids []string
		for _, span := range codeSpan.FindAllString(l.text, -1) {
			if name, ok := codeName(span); ok {
				ids = append(ids, "`"+name+"`")
			}
		}
		if len(ids) == 0 {
			continue
		}
		said := false
		for _, c := range kept {
			said = said || c.Lines.overlaps(lineRange{l.n, l.n})
		}
		if !said {
			out = append(out, fmt.Sprintf("line %d names %s", l.n, strings.Join(ids, ", ")))
		}
	}
	return strings.Join(out, "; ")
}

// judgedInPartsPatch records in a doc's header the commit it was judged in
// parts at: a diff replacing the line that closes the header (or the one it
// replaces), so it applies after the fix, wherever the fix left that line.
func judgedInPartsPatch(p, content, at string) (string, error) {
	_, n := header(content)
	if n == 0 {
		return "", fmt.Errorf("%s: a doc judged in parts has a header", p)
	}
	lines := strings.Split(content, "\n")
	for i := 1; i < n-1; i++ {
		if strings.HasPrefix(lines[i], "judged-in-parts:") {
			return fmt.Sprintf("--- a/%s\n+++ b/%s\n@@ -%d,1 +%d,1 @@\n-%s\n+judged-in-parts: %s\n", p, p, i+1, i+1, lines[i], at), nil
		}
	}
	return fmt.Sprintf("--- a/%s\n+++ b/%s\n@@ -%d,1 +%d,2 @@\n-%s\n+judged-in-parts: %s\n+%s\n", p, p, n, n, lines[n-1], at, lines[n-1]), nil
}

// holdJudgedInParts keeps from the agent the docs judged in parts, or
// judged whole without being vouched for, whose sources did not change
// since, saying so.
func holdJudgedInParts(docs map[string]*suspectDoc, repo string) {
	for _, sd := range docs {
		switch {
		case sd.note != "":
		case held(sd.doc, sd.doc.JudgedInParts, repo):
			sd.note = fmt.Sprintf("(judged in parts at %s, its sources unchanged since: not put before an agent again until one changes; a person reads it whole against them, then moves `checked`)", sd.doc.JudgedInParts)
		case held(sd.doc, sd.doc.Judged, repo):
			sd.note = fmt.Sprintf("(judged at %s, but the agent could not vouch for every sentence — its note said what; its sources unchanged since: not put before an agent again until one changes; a person reads it against them, then moves `checked`)", sd.doc.Judged)
		}
	}
}

// held says whether a doc judged at a commit waits: none of its sources
// changed since, so it is not put before an agent again. A commit a squash
// or a rebase left out of HEAD stands for the one that brought it there.
func held(d *Doc, at, repo string) bool {
	if at == "" {
		return false
	}
	at, ok := onMain(repo, "HEAD", d.Path, at)
	if !ok {
		return false
	}
	for _, src := range d.Sources {
		name, p, anchor := splitSource(src)
		if name != "" {
			return false
		}
		if commits, err := changed(repo, at, "HEAD", p, anchor); err != nil || commits != "" {
			return false
		}
	}
	return true
}

// showFile is a file as HEAD holds it, its leading lines kept: its lines
// are cited by number.
func showFile(repo, p string) (string, error) {
	out, err := exec.Command("git", "-C", repo, "show", "HEAD:"+p).Output()
	if err != nil {
		return "", fmt.Errorf("git show HEAD:%s: %v", p, err)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

func orInt(n, d int) int {
	if n <= 0 {
		return d
	}
	return n
}
