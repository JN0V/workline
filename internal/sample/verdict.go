package sample

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/builtin/documentalist"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Each act drawn is re-checked by one auditor call, asked its kind's
// question (kinds/<kind>.md) on what the act was done on, a finding blind,
// never on a model that acted. An answer against the act is false only
// with a quote the engine finds again; the calls stop at the sample's
// ai-max-tokens (ADR-0046, docs/commands.md#workline-sample).

// reader reads the week's sample: the judge it asks, and what it spent.
type reader struct {
	o         Options
	settings  documentalist.Settings
	auditor   *role.Role
	judgeRole *role.Role // the panel's judges
	spec      string
	res       *Result
	cap       int
	spent     int
}

// over says why no more call is made: the week's tokens spent.
func (rd *reader) over() string {
	if rd.spent < rd.cap {
		return ""
	}
	return fmt.Sprintf("the sample spent %d tokens, past its ai-max-tokens (%d): not read this week; for a person", rd.spent, rd.cap)
}

// called counts a call the agent answered, or failed.
func (rd *reader) called(c agent.Call) {
	if c.Agent == "" {
		return
	}
	rd.res.AgentCalls++
	rd.res.Calls = append(rd.res.Calls, c)
	rd.spent += c.TokensIn + c.TokensOut
}

// noJudge is why nothing is read without an agent.
const noJudge = "no judge is set (--judge, WORKLINE_JUDGE, or the `sample.judge` setting): for a person to read"

// item is a piece of what an act was done on: a file of the code at its
// commit, the issue, a comment, the message, the diff.
type item struct {
	name string // as the task names it: a claim's `source.path`
	text string // what the task shows of it
	full string // what a quote is looked for in: all of it
	code bool   // a file of the code: a quote is evidence only outside a comment
}

// material is what one act is re-checked against, and how its answer reads.
type material struct {
	facet  string   // the kind's question
	act    string   // the act, as the task gives it
	about  string   // what the task says of the material
	items  []item   //
	acted  []string // the models that acted, as their records say
	author string   // the agent that acted, when its record says: claude…
	holds  string   // the note's first word saying the act stands: true, or false for a finding dropped
}

// kindOf is the facet an act is asked with; "" when ADR-0046 sets its kind
// no question.
func kindOf(a Act) string {
	switch a.Role {
	case reviewRole:
		return "finding"
	case commitRole:
		return "rewrite"
	case testRole:
		switch a.Kind {
		case "tests", "kept", "rewritten", "whys", "removal":
			return a.Kind
		}
	case productOwner:
		switch {
		case a.Kind == "ready", a.Kind == "refine", a.Kind == "sources", a.Kind == "ask", a.Kind == "split":
			return a.Kind
		case a.Kind == "order", a.Kind == "milestone", a.Kind == "estimate":
			return "order"
		case strings.HasPrefix(a.Kind, "close-"):
			return "closing"
		}
	}
	return ""
}

// facet is a kind's question: the project's own (.workline/roles/auditor/
// kinds/<kind>.md), else the role's.
func (rd *reader) facet(kind string) (string, error) {
	for _, dir := range []string{filepath.Join(rd.o.Repo, ".workline", "roles", "auditor"), rd.auditor.Dir} {
		if data, err := os.ReadFile(filepath.Join(dir, "kinds", kind+".md")); err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}
	return "", fmt.Errorf("the auditor has no kinds/%s.md", kind)
}

// readAct has one act re-checked, and says its verdict.
func (rd *reader) readAct(a *Act) {
	a.Verdict = "not-read"
	kind := kindOf(*a)
	if kind == "" {
		a.Why = fmt.Sprintf("ADR-0046 asks no question of a `%s`: for a person", a.Kind)
		return
	}
	if rd.spec == "" || rd.spec == "none" {
		a.Why = noJudge
		return
	}
	facet, err := rd.facet(kind)
	if err != nil {
		a.Why = err.Error() + "; for a person"
		return
	}
	m, why := rd.material(a, kind)
	if why != "" {
		a.Why = why + "; for a person"
		return
	}
	m.facet = facet + m.facet
	spec, moved := apart(rd.spec, m.acted)
	if why := rd.over(); why != "" {
		a.Why = why
		return
	}
	a.Judge = spec
	answer, call, err := ask(spec, rd.auditor, m.task())
	rd.called(call)
	a.Model = call.Model
	a.Independence = "unknown (its record does not say the model that acted)"
	level := ""
	if m.author != "" && len(m.acted) > 0 {
		level = judge.Independence(spec, call.Model, m.author, m.acted)
		a.Independence = fmt.Sprintf("%s (%s → %s)", level, strings.Join(m.acted, ", "), orUnknown(call.Agent, call.Model))
	}
	switch {
	case errors.Is(err, agent.ErrUnavailable):
		a.Why = "the judge could not answer: " + err.Error() + "; for a person"
		return
	case err != nil:
		a.Why = "the judge's answer could not be read: " + err.Error() + "; for a person"
		return
	case level == "context":
		a.Why = fmt.Sprintf("the judge answered with a model that acted (%s): no independent read; for a person", call.Model)
		return
	}
	if floor := rd.settings.Sample.AtLeast; floor != "" {
		if _, ok := judge.Levels[floor]; !ok {
			a.Why = fmt.Sprintf("sample.at-least %q is not provider, model or context", floor)
			return
		}
		if judge.Levels[level] < judge.Levels[floor] {
			a.Why = fmt.Sprintf("the judge stands below sample.at-least (%s): %s; for a person", floor, a.Independence)
			return
		}
	}
	a.Why = moved
	settle(a, m, answer)
	if a.Verdict != "false" {
		return
	}
	a.Panel = rd.settleByPanel(a.key, kind, m.holds == "true", a.Proofs, m.panelText(), &a.Verdict, &a.Why)
	// A true finding its judge dropped: for an issue, its defect found
	// again on the branch when it is written (Apply).
	if a.Role == reviewRole && a.Kind == "dropped" && a.Verdict == "false" && (a.Panel == nil || a.Panel.For == "auditor") {
		path, _, _ := cutLine(a.Where)
		for _, p := range a.Proofs {
			if p.Path == path {
				a.Defect = &Defect{Path: path, Quote: p.Quote, Lens: a.Lens, Title: a.Title}
				break
			}
		}
	}
}

// Defect is a true finding the reviewer's judge dropped: opened as an
// issue while its cause is still on the branch (ADR-0046).
type Defect struct {
	Path  string `json:"path"`
	Quote string `json:"quote"` // the code that fails, as the auditor quoted it
	Lens  string `json:"lens"`
	Title string `json:"title"`
	Issue int    `json:"issue,omitempty"` // the issue it is opened as, or held by
	Said  string `json:"said,omitempty"`  // what became of it
}

// apart is the judge to ask: of Claude, a model none of those that acted
// is, and why it moved; else the one set.
func apart(spec string, acted []string) (string, string) {
	rest, ok := strings.CutPrefix(spec, "claude:")
	if !ok {
		return spec, ""
	}
	alias, effort, _ := strings.Cut(rest, "@")
	used := func(a string) bool {
		for _, m := range acted {
			if a != "" && strings.Contains(m, a) {
				return true
			}
		}
		return false
	}
	if !used(alias) {
		return spec, ""
	}
	for _, other := range []string{"opus", "sonnet", "haiku"} {
		if !used(other) {
			s := "claude:" + other
			if effort != "" {
				s += "@" + effort
			}
			return s, fmt.Sprintf("the judge set, %s, is a model that acted (%s): read by %s instead", spec, strings.Join(acted, ", "), s)
		}
	}
	return spec, ""
}

// settle reads the judge's answer: the act stands when its note says so;
// said wrong, it is false with a claim whose quote is found again in the
// item it names, else unproven, for a person.
func settle(a *Act, m *material, answer []intent.Intention) {
	note := ""
	var claims []claimed
	for _, in := range answer {
		switch in.Kind {
		case "note":
			if note == "" {
				note = strings.TrimSpace(fmt.Sprint(in.Value))
			}
		case "claim":
			data, _ := yaml.Marshal(in.Value)
			var c claimed
			yaml.Unmarshal(data, &c)
			claims = append(claims, c)
		}
	}
	word, _, _ := strings.Cut(note, ":")
	word = strings.ToLower(strings.Trim(strings.TrimSpace(word), "*`\"'"))
	switch {
	case note == "":
		a.Verdict, a.Why = "not-read", "the judge answered no note; for a person"
		return
	case word != "true" && word != "false":
		a.Verdict, a.Why = "not-read", fmt.Sprintf("the judge's note says neither true nor false (%q); for a person", oneLine(note, 120))
		return
	case word == m.holds:
		a.Verdict = "true"
		return
	}
	for _, c := range claims {
		p := Proof{Passage: strings.Join(strings.Fields(c.Quote), " "), Why: c.Why}
		why := ""
		switch {
		case c.Source == nil || strings.TrimSpace(c.Source.Path) == "" || strings.TrimSpace(c.Source.Quote) == "":
			why = "a claim gives the `source` it quotes: its `path` and `quote`"
		default:
			p.Path, p.Quote = strings.Trim(strings.TrimSpace(c.Source.Path), "`"), strings.Join(strings.Fields(c.Source.Quote), " ")
			it := m.find(p.Path)
			switch {
			case it == nil:
				why = fmt.Sprintf("%s is none of what the act was done on (%s)", p.Path, m.names())
			case it.code:
				found, code, line := documentalist.QuoteIn(it.name, it.full, c.Source.Quote)
				p.Line = line
				switch {
				case !found:
					why = fmt.Sprintf("%q is not in %s", p.Quote, it.name)
				case !code:
					why = fmt.Sprintf("%q is a comment (%s:%d), not the code: a comment is not evidence", p.Quote, it.name, line)
				}
			case !strings.Contains(strings.Join(strings.Fields(it.full), " "), p.Quote):
				why = fmt.Sprintf("%q is not in %s", p.Quote, it.name)
			}
		}
		if why != "" {
			p.Why = why
			a.Unproven = append(a.Unproven, p)
		} else {
			a.Proofs = append(a.Proofs, p)
		}
	}
	if len(a.Proofs) > 0 {
		a.Verdict = "false"
		return
	}
	a.Verdict = "unproven"
	if a.Why == "" {
		a.Why = "said wrong, " + oneLine(note, 200) + ", with no quote found again: for a person"
	}
}

// find is the item of the material a claim names.
func (m *material) find(name string) *item {
	name = strings.Join(strings.Fields(name), " ")
	for i := range m.items {
		if strings.EqualFold(m.items[i].name, name) {
			return &m.items[i]
		}
	}
	return nil
}

func (m *material) names() string {
	var n []string
	for _, it := range m.items {
		n = append(n, it.name)
	}
	return strings.Join(n, ", ")
}

// panelText is what a dispute's judges are given: the act and what it was
// done on, the auditor's question left out.
func (m *material) panelText() string {
	t := m.task()
	if i := strings.Index(t, "## The act"); i >= 0 {
		return t[i:]
	}
	return t
}

// task is what the judge is given: the question, the act, and what it was
// done on.
func (m *material) task() string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The question\n\n%s\n\n## The act\n\n%s\n\n## What it was done on\n\n", strings.TrimSpace(m.facet), m.act)
	if m.about != "" {
		b.WriteString(m.about + "\n\n")
	}
	for _, it := range m.items {
		fence := backlog.FenceFor(it.text)
		fmt.Fprintf(&b, "### %s\n\n%s\n%s\n%s\n\n", it.name, fence, strings.TrimRight(it.text, "\n"), fence)
	}
	return b.String()
}

// material gathers what an act was done on, as its records let it be read
// again; why, when it cannot be.
func (rd *reader) material(a *Act, kind string) (*material, string) {
	m := &material{holds: "true"}
	switch a.Role {
	case reviewRole:
		return m, rd.finding(a, m)
	case commitRole:
		return m, rd.rewrite(a, m)
	case testRole:
		return m, rd.testerMaterial(a, m)
	}
	return m, rd.issueAct(a, m, kind)
}

// finding is a finding's material: the code of its file at the head its
// review read, the finding blind — its lens, place and title, never
// whether it was kept, nor its judge's reason.
func (rd *reader) finding(a *Act, m *material) string {
	path, line, ok := cutLine(a.Where)
	if !ok {
		return fmt.Sprintf("the finding is at %q, no line of the code", a.Where)
	}
	if a.Commit == "" {
		return "the head its review read is not in this clone"
	}
	content, err := git(rd.o.Repo, "show", a.Commit+":"+path)
	if err != nil {
		return fmt.Sprintf("%s is not at %.7s, the head its review read", path, a.Commit)
	}
	m.items = append(m.items, item{name: path, text: excerpt(content, line), full: content, code: true})
	if q := lensQuestion(rd.o.Repo, rd.o.RolesDir, a.Lens); q != "" {
		m.facet = fmt.Sprintf("\n\nThe %s lens asks of a finding: %s", a.Lens, q)
	}
	m.act = fmt.Sprintf("A finding of the reviewer's `%s` lens, at `%s`: %s", a.Lens, a.Where, a.Title)
	m.about = fmt.Sprintf("The code of `%s` at %.7s, the head its review read.", path, a.Commit)
	for _, model := range strings.Split(a.By, " → ") {
		if model = strings.TrimSpace(model); model != "" {
			m.acted = append(m.acted, model)
		}
	}
	if len(m.acted) > 0 && strings.HasPrefix(m.acted[0], "claude") {
		m.author = "claude"
	}
	if a.Kind == "dropped" {
		m.holds = "false"
	}
	return ""
}

// rewrite is a rewritten commit's material: its message, its trailers
// left out, and its diff, at most diffChars characters (ADR-0046).
func (rd *reader) rewrite(a *Act, m *material) string {
	msg, err := git(rd.o.Repo, "log", "-1", "--format=%B", a.Commit)
	if err != nil {
		return fmt.Sprintf("commit %s is not in this clone", a.Ref)
	}
	var keep []string
	for _, l := range strings.Split(msg, "\n") {
		if !strings.HasPrefix(l, "Workline-") {
			keep = append(keep, l)
		}
	}
	diff, err := git(rd.o.Repo, "show", "--format=", "--no-color", "--patch", a.Commit)
	if err != nil {
		return fmt.Sprintf("the diff of %s cannot be read", a.Ref)
	}
	diff = cutDiff(diff)
	text := strings.TrimSpace(strings.Join(keep, "\n"))
	m.items = append(m.items, item{name: "message", text: text, full: text}, item{name: "diff", text: diff, full: diff})
	m.act = fmt.Sprintf("The message of commit %s, which the committer rewrote.", a.Ref)
	m.author, m.acted = actedBy(a.By)
	return ""
}

// cutDiff is a diff as its judge reads it: at most diffChars characters.
func cutDiff(diff string) string {
	if len(diff) <= diffChars {
		return diff
	}
	cut := diffChars
	for cut > 0 && !utf8.RuneStart(diff[cut]) {
		cut-- // never inside a character
	}
	return diff[:cut] + fmt.Sprintf("\n… (cut: the diff takes more than %d characters)", diffChars)
}

// actedBy reads the agents and models a record names, <agent>:<model>, a
// comma between two: the agent, and each model.
func actedBy(list string) (string, []string) {
	author := ""
	var models []string
	for _, b := range strings.Split(list, ",") {
		provider, model, _ := strings.Cut(strings.TrimSpace(b), ":")
		if model != "" {
			author, models = provider, append(models, model)
		}
	}
	return author, models
}

// diffChars is the most of a rewritten commit's diff its judge reads.
const diffChars = 8000

// codeLines is the most of the code an issue names its judge reads.
const codeLines = 300

var issueRef = regexp.MustCompile(`#(\d+)`)

// issueAct is a product owner's act's material: the issue's text it read,
// found by its record's digest — the body now, else the version the
// forge's edit history keeps —, its comments up to the act's day, its
// record's own comments left out; the code it names at the commit it read
// at, for a sources or a closing; the parts of a split; the other issue
// of a duplicate. A record an engine before ADR-0046 kept says none of
// these: the issue as it reads now, the code at the last commit of the
// act's day, said (older).
func (rd *reader) issueAct(a *Act, m *material, kind string) string {
	id, err := strconv.Atoi(strings.TrimPrefix(a.Ref, "#"))
	if err != nil {
		return fmt.Sprintf("%s is no issue", a.Ref)
	}
	b, ok := rd.o.Forge.(forge.Backlog)
	if !ok {
		return "no forge to read the issue from"
	}
	is, err := rd.o.Forge.Issue(id)
	if err != nil {
		return fmt.Sprintf("%s could not be read: %v", a.Ref, err)
	}
	notes, err := b.Notes(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return fmt.Sprintf("the comments of %s could not be read: %v", a.Ref, err)
	}
	m.about = "The issue as it reads now, and its comments up to that day: its record, from an engine before ADR-0046, does not say the text the product owner read."
	a.older = a.Commit == ""
	var now *forge.Issue // the issue as it reads now, beside the text a refine read: what it wrote
	if a.Body != "" {
		var versions func() ([]string, error)
		if e, ok := rd.o.Forge.(forge.Edited); ok {
			versions = func() ([]string, error) { return e.Versions(id) }
		}
		body, edited, why := textRead(is.Body, a.Body, versions)
		if why != "" {
			return fmt.Sprintf("the text the product owner read (body %s) is %s", a.Body, strings.ReplaceAll(why, "the issue's", a.Ref+"'s"))
		}
		m.about = "The issue as the product owner read it, its body unchanged since, and its comments up to that day."
		if edited {
			m.about = "The issue as the product owner read it, its body as the forge's edit history keeps it, and its comments up to that day."
			if kind == "refine" {
				m.about += fmt.Sprintf(" %s now is the issue as it reads today: the sections the refine wrote, and any edit since.", a.Ref)
				n := *is
				now = &n
			}
		}
		is.Body = body
	}
	m.author, m.acted = actedBy(a.Acted)
	m.items = append(m.items, issueItem(is))
	if now != nil {
		it := issueItem(now)
		it.name += " now"
		m.items = append(m.items, it)
	}
	k := 0
	for _, n := range notes {
		if strings.Contains(n.Body, "<!-- workline:sticky="+productOwner+"/") {
			continue // the role's record, its reasoning: never given
		}
		if len(n.Created) >= 10 && n.Created[:10] > a.Day {
			continue
		}
		k++
		who, when := n.Author, n.Created
		if who == "" {
			who = "someone"
		}
		if when == "" {
			when = "undated"
		}
		text := fmt.Sprintf("%s, %s:\n\n%s", who, when, n.Body)
		m.items = append(m.items, item{name: fmt.Sprintf("#%d comment %d", id, k), text: text, full: text})
	}
	st, _, _ := backlog.ReadState(forge.Bodies(notes), productOwner)
	m.act = fmt.Sprintf("The product owner's `%s` on #%d, on %s, done alone at %s: %s", a.Kind, id, a.Day, a.By, a.Line)
	switch {
	case kind == "split" && st != nil:
		for _, child := range st.Split {
			if c, err := rd.o.Forge.Issue(child); err == nil {
				m.items = append(m.items, issueItem(c))
			}
		}
	case a.Kind == "close-duplicate":
		for _, r := range issueRef.FindAllStringSubmatch(a.Line, -1) {
			if other, _ := strconv.Atoi(r[1]); other != id {
				if c, err := rd.o.Forge.Issue(other); err == nil {
					m.items = append(m.items, issueItem(c))
					break
				}
			}
		}
	case (kind == "sources" || kind == "closing") && st != nil && len(st.Sources) > 0:
		at, which := a.Commit, "the commit it read at"
		if at == "" {
			if at, err = dayCommit(rd.o.Repo, a.Day); err != nil {
				return err.Error()
			}
			which = "the last commit on the branch that day"
		} else if _, err := git(rd.o.Repo, "rev-parse", "-q", "--verify", at+"^{commit}"); err != nil {
			return fmt.Sprintf("%s, the commit it read at, is not in this clone", at)
		}
		m.items = append(m.items, codeAt(rd.o.Repo, at, st.Sources, codeLines)...)
		m.about += fmt.Sprintf(" The code it names at %.7s, %s, %d lines at most.", at, which, codeLines)
	}
	return ""
}

// textRead is the issue's text the product owner read, found by its
// digest (backlog.BodyDigest): its body now when it matches, else the
// version its edit history keeps, edited then; why, when neither holds
// it — versions nil, a forge keeping no history.
func textRead(now, digest string, versions func() ([]string, error)) (text string, edited bool, why string) {
	if backlog.BodyDigest(now) == digest {
		return now, false, ""
	}
	if versions == nil {
		return "", false, "not the issue's body now, and this forge keeps no edit history"
	}
	all, err := versions()
	if err != nil {
		return "", false, "not the issue's body now, and its edit history could not be read: " + err.Error()
	}
	for _, v := range all {
		if backlog.BodyDigest(v) == digest {
			return v, true, ""
		}
	}
	return "", false, "neither the issue's body now nor a version its edit history keeps"
}

func issueItem(is *forge.Issue) item {
	text := fmt.Sprintf("Title: %s\n\n%s", is.Title, is.Body)
	return item{name: fmt.Sprintf("#%d", is.ID), text: text, full: text}
}

// dayCommit is the last commit on the branch on a day, UTC.
func dayCommit(repo, day string) (string, error) {
	d, err := time.Parse(time.DateOnly, day)
	if err != nil {
		return "", fmt.Errorf("the act's day %q is no day", day)
	}
	end := d.AddDate(0, 0, 1).Unix()
	out, err := git(repo, "log", "--first-parent", "--format=%H %ct", "HEAD")
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(out, "\n") {
		sha, ts, ok := strings.Cut(l, " ")
		var sec int64
		fmt.Sscan(ts, &sec)
		if ok && sec < end {
			return sha, nil
		}
	}
	return "", fmt.Errorf("no commit on the branch by %s", day)
}

// codeAt is the files the paths name at a commit — a folder's, each —
// at most max lines together; a quote is looked for in each file whole.
func codeAt(repo, commit string, paths []string, max int) []item {
	var out []item
	left := max
	for _, p := range paths {
		if strings.Contains(p, ":") || left <= 0 {
			continue // another repository's, or the lines spent
		}
		files := []string{p}
		if t, _ := git(repo, "cat-file", "-t", commit+":"+strings.TrimSuffix(p, "/")); t == "tree" {
			list, _ := git(repo, "ls-tree", "-r", "--name-only", commit, "--", strings.TrimSuffix(p, "/"))
			files = strings.Fields(list)
		}
		for _, f := range files {
			if left <= 0 {
				break
			}
			content, err := git(repo, "show", commit+":"+f)
			if err != nil {
				continue
			}
			lines := strings.Split(content, "\n")
			shown := content
			if len(lines) > left {
				shown = strings.Join(lines[:left], "\n") + "\n… (cut)"
			}
			left -= min(len(lines), left)
			out = append(out, item{name: f, text: shown, full: content, code: true})
		}
	}
	return out
}

// cutLine reads a place, path:line.
func cutLine(where string) (string, int, bool) {
	i := strings.LastIndex(where, ":")
	if i <= 0 || strings.HasPrefix(where, "#") {
		return "", 0, false
	}
	n, err := strconv.Atoi(where[i+1:])
	return where[:i], n, err == nil
}

// excerpt is a file as its finding's judge reads it: whole up to 400
// lines, else the 80 lines either side of the finding's.
func excerpt(content string, line int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= 400 {
		return content
	}
	from, to := max(0, line-81), min(len(lines), line+80)
	return fmt.Sprintf("… (lines %d to %d of %d)\n%s", from+1, to, len(lines), strings.Join(lines[from:to], "\n"))
}

// lensQuestion is what a reviewer's lens asks its judge, in its front
// matter (`judge.question`): the project's lens, else the role's; "" when
// it asks nothing of its own.
func lensQuestion(repo, rolesDir, lens string) string {
	for _, dir := range []string{filepath.Join(repo, ".workline", "roles", "reviewer"), filepath.Join(rolesDir, "reviewer")} {
		data, err := os.ReadFile(filepath.Join(dir, "lenses", lens+".md"))
		if err != nil {
			continue
		}
		rest, ok := strings.CutPrefix(strings.ReplaceAll(string(data), "\r\n", "\n"), "---\n")
		head, _, closed := strings.Cut(rest, "\n---\n")
		if !ok || !closed {
			return ""
		}
		var front struct {
			Judge struct {
				Question string `yaml:"question"`
			} `yaml:"judge"`
		}
		yaml.Unmarshal([]byte(head), &front)
		return strings.TrimSpace(front.Judge.Question)
	}
	return ""
}

// actFinding says an act's verdict, at its place.
func actFinding(a Act) verdict.Finding {
	where := a.Ref
	if a.Where != "" {
		where += " " + a.Where
	}
	who := ""
	if a.Judge != "" {
		who = fmt.Sprintf("; read by %s, independence: %s", a.Judge, a.Independence)
	}
	switch a.Verdict {
	case "true":
		return verdict.Finding{Rule: "act-stands", Where: where, Level: "warn", Message: actWords(a) + " — stands" + who}
	case "false":
		var lines []string
		for _, p := range a.Proofs {
			lines = append(lines, p.String())
		}
		return verdict.Finding{Rule: "act-false", Where: where, Message: actWords(a) + " — false: " + strings.Join(lines, "; ") + who}
	case "unproven":
		var lines []string
		for _, p := range a.Unproven {
			lines = append(lines, p.Why)
		}
		return verdict.Finding{Rule: "act-unproven", Where: where, Message: actWords(a) + " — said wrong, not proven: " + strings.Join(lines, "; ") + "; for a person" + who}
	}
	return verdict.Finding{Rule: "sample-not-read", Where: where, Message: actWords(a) + "; " + a.Why}
}
