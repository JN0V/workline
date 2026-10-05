// Package productowner holds the deterministic steps of the product owner
// role (roles/product-owner): pre lists the open issues with what the engine
// knows of each; the acts the agent proposes are checked when the engine
// applies them (internal/backlog). The role's scripts call them through
// `workline builtin product-owner pre|post`.
package productowner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

const (
	exitNothing   = 10
	exitExternal  = 3
	bodyMax       = 3000 // characters of an issue's body given to the agent
	commentsMax   = 5    // its last comments given
	filesPerIssue = 3    // files an issue names, given whole
	codeLines     = 400  // lines of a file given
	relatedMax    = 6    // other issues on the same code, given whole
)

// Settings are the role's settings pre reads, as merged by the engine.
type Settings struct {
	IssuesPerRun int `json:"issues-per-run"` // issues read in one run
	CodeLinesMax int `json:"code-lines-max"` // lines of code given in one run, all files together
	Acts         struct {
		Ask struct {
			Rounds int `json:"rounds"` // times an issue's reporter is written to, then a person
		} `json:"ask"`
	} `json:"acts"`
}

// rounds is how many times an issue's reporter is written to, as the
// engine counts them (backlog.RoundsMax).
func (s Settings) rounds() int {
	return backlog.RoundsMax(map[string]backlog.Setting{"ask": {Rounds: s.Acts.Ask.Rounds}})
}

// Pre lists the open issues in in/task.md. An issue without a state comment
// gets one, through a fallback comment, and is judged from the next run; one
// whose state comment does not read is left out, and said.
func Pre(runDir, repo string) int {
	role := os.Getenv("WORKLINE_ROLE")
	if role == "" {
		role = "product-owner"
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	b, ok := f.(forge.Backlog)
	if f == nil || !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no backlog to keep",
			Findings: []verdict.Finding{{Rule: "no-forge", Message: "the product owner keeps a forge's issues; " + forge.Missing}}})
	}
	open, err := b.Issues()
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	if os.Getenv("WORKLINE_EVENT") == "import" {
		return preImport(runDir, repo, role, open)
	}
	head, err := exec.Command("git", "-C", repo, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return fail(fmt.Errorf("the commit the run is on: %v", err))
	}
	commit := strings.TrimSpace(string(head))
	s := Settings{IssuesPerRun: 8, CodeLinesMax: 1500}
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json")); err == nil {
		if err := json.Unmarshal(data, &s); err != nil {
			return fail(fmt.Errorf("settings: %v", err))
		}
	}
	var task strings.Builder
	var fallback []intent.Intention
	var findings []verdict.Finding
	judged := 0
	tracked := trackedFiles(repo)
	var code []string // the files the issues name, given once each
	var others []string
	type due struct {
		is       forge.Issue
		st       *backlog.State
		comments []string
		notes    []forge.Note // the comments with their authors
	}
	var readIDs []string                            // the issues read, for the plan (backlog.Decide)
	var again, never, changed, rest []due           // again: an act proposed only for the cap, read first
	reopened := backlog.ClosedByRole(b, role, open) // closed by the role, open again
	milestones, err := b.Milestones()
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	next := backlog.NextMilestone(repo, milestones) // where an issue that slipped goes
	capped := backlog.CappedByRole(b, role, open)   // an act proposed only for the cap
	for _, is := range open {
		if is.Title == backlog.ReportTitle(role) {
			continue
		}
		notes, err := b.Notes(forge.Target{Kind: "issue", ID: is.ID})
		comments := forge.Bodies(notes)
		if errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		}
		if err != nil {
			return fail(err)
		}
		st, found, err := backlog.ReadState(comments, role)
		// Only the text as last proposed, not answered since — an answer
		// may change it, the agent reads it first — and only while the body
		// still lacks what it adds.
		p := backlog.LastProposal(comments, role)
		agreed := ""
		if p != nil {
			_, adds, _ := backlog.Refine(is.Body, *p, role)
			if len(adds) > 0 {
				agreed = backlog.Agreement(notes, is, role)
			}
			if len(adds) == 0 || backlog.ReadExchange(comments, role).Answered {
				p = nil
			}
		}
		if found && err == nil && agreed != "" && !backlog.Accepted(is) {
			// Its reporter, or a person of the project, replied `agreed` to
			// the text last proposed: the engine writes what the body still
			// lacks, with no agent; ready, a split or a rename stay the
			// project's, by the label (ADR-0021).
			whose := "its reporter"
			if agreed != is.Author {
				whose = "a person of the project"
			}
			lp := backlog.LastProposal(comments, role)
			fallback = append(fallback, intent.Intention{Kind: "refine", Value: map[string]any{
				"issue": is.ID, "need": lp.Need, "verification": lp.Verification, "validation": lp.Validation,
				"scope": lp.Scope, "sources": lp.Sources, "own": true, "agreed": agreed,
				"why": fmt.Sprintf("the text proposed to its reporter, agreed to by @%s (%s) in a reply.", agreed, whose)}})
		}
		if found && err == nil && backlog.Accepted(is) && p != nil {
			// A refined text proposed to an outsider, agreed to by a person
			// of the project with the label: the engine writes what the body
			// still lacks, with no agent (ADR-0021).
			fallback = append(fallback, intent.Intention{Kind: "refine", Value: map[string]any{
				"issue": is.ID, "need": p.Need, "verification": p.Verification, "validation": p.Validation,
				"scope": p.Scope, "sources": p.Sources, "own": true,
				"why": "the text proposed to its reporter, agreed to by a person (" + backlog.LabelAccepted + ")"}})
		}
		if found && err == nil && backlog.Accepted(is) {
			// A person accepted its drafts, with the label: the engine moves
			// it to ready if its sections are there, with no agent.
			fallback = append(fallback, intent.Intention{Kind: "ready", Value: map[string]any{
				"issue": is.ID, "why": "its drafts accepted by a person (" + backlog.LabelAccepted + ")", "own": true}})
		}
		switch {
		case !found:
			fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{
				"issue": is.ID, "sticky": "state", "body": backlog.FormatState(backlog.State{Confirmed: commit})}})
			continue
		case err != nil:
			findings = append(findings, verdict.Finding{Rule: "state-broken", Where: fmt.Sprintf("#%d", is.ID),
				Message: "its state comment does not read (" + err.Error() + "): the issue is not judged, nothing is written on it"})
			continue
		}
		if backlog.Released(repo, is.Milestone) {
			// Its milestone's release is tagged: it slipped, and the engine
			// moves it to the next, with no agent (or proposes, with none).
			fallback = append(fallback, intent.Intention{Kind: "milestone", Value: map[string]any{
				"issue": is.ID, "milestone": next, "from": is.Milestone, "own": true,
				"why": fmt.Sprintf("slipped: %s is released (its tag exists).", is.Milestone)}})
		}
		switch {
		case agreed != "" && !backlog.Accepted(is):
			// Agreed to: written this run, not read again for that reply.
			rest = append(rest, due{is, st, comments, notes})
		case slices.Contains(capped, is.ID):
			again = append(again, due{is, st, comments, notes})
		case st.Judged == "":
			never = append(never, due{is, st, comments, notes})
		case sourcesChanged(repo, st),
			backlog.PeopleComments(comments) != st.Comments,         // someone wrote since it was read
			st.Body != "" && backlog.BodyDigest(is.Body) != st.Body, // someone changed its body
			slices.Contains(reopened, is.ID):
			changed = append(changed, due{is, st, comments, notes})
		default:
			rest = append(rest, due{is, st, comments, notes})
		}
	}
	// Those never read first, then those whose code changed since; an issue
	// whose code did not change is not read again (ADR-0018). Without an
	// agent, nothing is read, and no issue is said to be.
	toRead := append(append(again, never...), changed...)
	if os.Getenv("WORKLINE_AI") == "none" {
		toRead = nil
	}
	for i, d := range toRead {
		if i >= s.IssuesPerRun {
			rest = append(rest, d)
			continue
		}
		judged++
		readIDs = append(readIDs, strconv.Itoa(d.is.ID))
		files := named(repo, d.is, d.st, d.comments, tracked)
		writeIssue(&task, role, s.rounds(), d.is, d.st, d.notes, files)
		for _, f := range files {
			if !slices.Contains(code, f) {
				code = append(code, f)
			}
		}
		read := *d.st
		read.Judged, read.Comments, read.Body = commit, backlog.PeopleComments(d.comments), backlog.BodyDigest(d.is.Body)
		fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{
			"issue": d.is.ID, "sticky": "state", "update-only": true, "if-answered": true, "body": backlog.FormatState(read)}})
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "issues-read"), []byte(strings.Join(readIDs, "\n")), 0o644); err != nil {
		return fail(err)
	}
	if len(fallback) > 0 {
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
			return fail(err)
		}
	}
	if judged == 0 {
		if len(fallback) > 0 { // no task.md: the agent is not asked, the state comments are written
			return writeFindings(runDir, findings)
		}
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no issue to judge", Findings: findings})
	}
	intro := fmt.Sprintf("The run is on commit %s. %d open issues to read against the code.\n\n%s\n", commit, judged, releases(repo, b))
	// An issue not read in this run, on the same code as one read, is given
	// whole: it may be the original a duplicate is closed against, and a
	// duplicate's original is quoted (ADR-0018).
	var related strings.Builder
	shown := 0
	// The rest in the backlog's order: nearest milestone, priority, number.
	sort.SliceStable(rest, func(i, j int) bool { return backlog.Less(rest[i].is, rest[j].is) })
	for _, d := range rest {
		files := named(repo, d.is, d.st, d.comments, tracked)
		if shown < relatedMax && slices.ContainsFunc(files, func(f string) bool { return slices.Contains(code, f) }) {
			shown++
			fmt.Fprintf(&related, "## #%d %s\n\n%s\n\n", d.is.ID, d.is.Title, clip(d.is.Body, bodyMax/2))
			continue
		}
		others = append(others, fmt.Sprintf("- #%d %s%s", d.is.ID, d.is.Title, place(d.is)))
	}
	if shown > 0 {
		fmt.Fprintf(&task, "# Other open issues on the same code\n\nNot to judge in this run: given whole, as the original a duplicate would be closed against.\n\n%s", related.String())
	}
	if len(others) > 0 {
		fmt.Fprintf(&task, "# The other open issues, titles only\n\nNot read in this run; a duplicate may be one of them.\n\n%s\n\n", strings.Join(others, "\n"))
	}
	writeCode(&task, repo, code, s.CodeLinesMax)
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(intro+task.String()), 0o644); err != nil {
		return fail(err)
	}
	return writeFindings(runDir, findings)
}

// writeIssue gives one issue to the agent: what the engine knows of it, its
// body, its last comments, the engine's own left out.
func writeIssue(b *strings.Builder, role string, rounds int, is forge.Issue, st *backlog.State, notes []forge.Note, files []string) {
	comments := forge.Bodies(notes)
	fmt.Fprintf(b, "## #%d %s\n\n", is.ID, is.Title)
	if len(is.Labels) > 0 {
		fmt.Fprintf(b, "Labels: %s\n", strings.Join(is.Labels, ", "))
	}
	sources := "none named yet"
	if len(st.Sources) > 0 {
		sources = strings.Join(st.Sources, ", ")
	}
	if is.Milestone != "" {
		fmt.Fprintf(b, "Milestone: %s\n", is.Milestone)
	}
	if set := backlog.PriorityLabels(is); len(set) > 0 {
		whose := ""
		if len(set) != 1 || set[0] != st.Priority {
			whose = " (a person's: kept)"
		}
		fmt.Fprintf(b, "Priority: %d%s\n", backlog.Priority(is), whose)
	}
	if by := backlog.OpenedBy(is.Body); by != "" {
		// A role's finding, opened through the one way (ADR-0018): the
		// product owner takes it from here, as any other issue.
		fmt.Fprintf(b, "Opened by: the %s role, on a finding of its own — a draft to refine: its Need and Validation are not a person's yet\n", by)
	} else if is.Author != "" {
		outside := ""
		if !is.Insider {
			outside = ", without write access to the project"
		}
		fmt.Fprintf(b, "Opened by: %s%s\n", is.Author, outside)
	}
	if st.Title != "" && st.Title != is.Title {
		b.WriteString("Title: a person's, set after the role's (kept)\n")
	}
	if len(st.Split) > 0 {
		// Split already (ADR-0022): its children are issues of their own.
		var ids []string
		for _, id := range st.Split {
			ids = append(ids, fmt.Sprintf("#%d", id))
		}
		fmt.Fprintf(b, "Split into: %s (not split again)\n", strings.Join(ids, ", "))
	}
	fmt.Fprintf(b, "Sections: %s\n", sections(is.Body))
	fmt.Fprintf(b, "Sources: %s. Confirmed at: %s.\n", sources, st.Confirmed)
	if len(files) > 0 {
		fmt.Fprintf(b, "Code it names, given below: %s.\n", strings.Join(files, ", "))
	}
	if e := backlog.ReadExchange(comments, role); e.Rounds > 0 {
		// The conversation with its reporter so far: what was asked or
		// proposed is shown below, among the comments (ADR-0021).
		answer := "not answered since: nothing more is written to them before an answer"
		if e.Answered {
			answer = "answered since: read the answer — refine, mark ready, or ask what is still missing, never a question asked before"
		}
		fmt.Fprintf(b, "Written to its reporter: %d of %d times; %s.\n", e.Rounds, rounds, answer)
	}
	if !is.Insider && !backlog.Accepted(is) && backlog.OpenedBy(is.Body) == "" {
		fmt.Fprintf(b, "Its reporter is outside the project: a `refine` is proposed to them in a comment, not written in the body; `why` says what you understood of the issue; a `split`, a `rename` or `ready` is proposed to the project in the report.\n")
	}
	b.WriteString("\n")
	fmt.Fprintf(b, "%s\n\n", clip(is.Body, bodyMax))
	type said struct{ who, text string }
	var kept []said
	for _, n := range notes {
		switch {
		case backlog.Round(n.Body, role):
			kept = append(kept, said{"", "(The product owner wrote:) " + marker.ReplaceAllString(n.Body, "")})
		default:
			if _, engine := backlog.EngineMarker(n.Body); !engine {
				kept = append(kept, said{author(n, is), marker.ReplaceAllString(n.Body, "")}) // a person's, quoting the engine's maybe
			}
		}
	}
	if len(kept) > commentsMax {
		fmt.Fprintf(b, "(%d earlier comments left out)\n\n", len(kept)-commentsMax)
		kept = kept[len(kept)-commentsMax:]
	}
	for _, c := range kept {
		fmt.Fprintf(b, "Comment%s:\n> %s\n\n", c.who, strings.ReplaceAll(clip(c.text, bodyMax), "\n", "\n> "))
	}
}

// author says who wrote a comment, as the agent is told: its reporter, a
// person of the project, a bot, or someone else — whose word decides
// nothing (ADR-0021); "" when the forge does not say.
func author(n forge.Note, is forge.Issue) string {
	switch {
	case n.Author == "":
		return ""
	case n.Bot:
		return " by @" + n.Author + ", a bot"
	case n.Author == is.Author:
		return " by @" + n.Author + ", its reporter"
	case n.Insider:
		return " by @" + n.Author + ", of the project"
	}
	return " by @" + n.Author + ", outside the project"
}

// place says an issue's milestone and priority, when it has them.
func place(is forge.Issue) string {
	var out []string
	if is.Milestone != "" {
		out = append(out, is.Milestone)
	}
	if n := backlog.Priority(is); n > 0 {
		out = append(out, fmt.Sprintf("priority %d", n))
	}
	if len(out) == 0 {
		return ""
	}
	return " (" + strings.Join(out, ", ") + ")"
}

// sections says which of the four sections an issue's body has, and which
// are drafts no person made theirs yet.
func sections(body string) string {
	have := work.Sections(body)
	var there, missing []string
	for _, name := range backlog.Sections {
		text := strings.TrimSpace(have[name])
		switch {
		case text == "":
			missing = append(missing, name)
		case strings.Contains(text, backlog.DraftMarker):
			there = append(there, name+" (draft)")
		default:
			there = append(there, name)
		}
	}
	out := "none"
	if len(there) > 0 {
		out = strings.Join(there, ", ")
	}
	if len(missing) > 0 {
		out += "; missing: " + strings.Join(missing, ", ")
	}
	return out
}

// byName is the one tracked file with this name, or "".
func byName(name string, tracked map[string]bool) string {
	found := ""
	for f := range tracked {
		if f == name || strings.HasSuffix(f, "/"+name) {
			if found != "" {
				return ""
			}
			found = f
		}
	}
	return found
}

// trackedFiles lists the files of the commit the run is on.
func trackedFiles(repo string) map[string]bool {
	out, _ := exec.Command("git", "-C", repo, "ls-files").Output()
	files := map[string]bool{}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		files[f] = true
	}
	return files
}

// provenance is the line the engine ends an imported issue with.
var provenance = regexp.MustCompile(`(?m)^Opened from .* by the [\w -]+ role\.\r?$`)

// marker is the engine's hidden text in a body: the file it was imported
// from, a key.
var marker = regexp.MustCompile(`<!-- workline:[^>]*-->`)

// symbol is a name of the code an issue quotes as code: `WriteRows`,
// `Core::publish()`.
var symbol = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*(?:::[A-Za-z_][A-Za-z0-9_]*)*)(?:\\(\\))?`")

var pathLike = regexp.MustCompile(`[\w.-]+(?:/[\w.-]+)+|[\w-]+\.[A-Za-z]{1,5}\b`)

// named lists the files an issue is about: its sources, then the paths its
// title and body name that the commit holds, at most filesPerIssue. A file
// named alone (`csv.go:5`, as a roadmap writes it) is the one file of the
// commit with that name; a name two files share is not guessed.
func named(repo string, is forge.Issue, st *backlog.State, comments []string, tracked map[string]bool) []string {
	var out []string
	add := func(p string) {
		p, _, _ = strings.Cut(p, "#")
		p = strings.Trim(p, "./`'\"")
		if !tracked[p] && !strings.Contains(p, "/") {
			p = byName(p, tracked)
		}
		if tracked[p] && !slices.Contains(out, p) && len(out) < filesPerIssue {
			out = append(out, p)
		}
	}
	for _, s := range st.Sources {
		add(s)
	}
	// The file an issue was imported from is where it was written, not
	// the code it is about.
	body := marker.ReplaceAllString(provenance.ReplaceAllString(is.Body, ""), "")
	for _, c := range comments {
		if !strings.Contains(c, "<!-- workline:") { // a person's: an answer may name the code
			body += "\n" + c
		}
	}
	for _, m := range pathLike.FindAllString(is.Title+"\n"+body, -1) {
		add(m)
	}
	// A symbol quoted as code: the file of the commit that holds it, when
	// one or two do, docs left out — the issue's code, found for it.
	for _, m := range symbol.FindAllStringSubmatch(is.Title+"\n"+body, -1) {
		name := m[1]
		if i := strings.LastIndex(name, "::"); i >= 0 {
			name = name[i+2:]
		}
		if len(out) >= filesPerIssue || len(name) < 4 {
			break
		}
		for _, f := range holding(repo, name) {
			add(f)
		}
	}
	return out
}

// holding lists the files of the commit holding a name, as a word, docs
// left out; none when more than two do: too common to tell.
func holding(repo, name string) []string {
	out, err := exec.Command("git", "-C", repo, "grep", "-l", "-w", "-F", name, "HEAD", "--", ".", ":!*.md").Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		files = append(files, strings.TrimPrefix(l, "HEAD:"))
	}
	if len(files) > 2 {
		return nil
	}
	return files
}

// writeCode gives each file named, whole up to codeLines, with its last
// commits: what changed it, so an issue the code solved can be told.
func writeCode(b *strings.Builder, repo string, files []string, budget int) {
	if len(files) == 0 {
		return
	}
	b.WriteString("# Code the issues name, as it is at this commit\n\n")
	for _, f := range files {
		data, err := exec.Command("git", "-C", repo, "show", "HEAD:"+f).Output()
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		more := ""
		if n := min(codeLines, budget); len(lines) > n {
			more = fmt.Sprintf("\n(%d more lines left out)", len(lines)-n)
			lines = lines[:n]
		}
		budget -= len(lines)
		if len(lines) == 0 {
			fmt.Fprintf(b, "## %s\n\nLeft out: the run's code budget is spent.\n\n", f)
			continue
		}
		for i := range lines {
			lines[i] = fmt.Sprintf("%4d  %s", i+1, lines[i])
		}
		log, _ := exec.Command("git", "-C", repo, "log", "-5", "--format=%h %as %s", "--", f).Output()
		fmt.Fprintf(b, "## %s\n\nLast commits:\n%s\n```\n%s\n```%s\n\n", f, strings.TrimSpace(string(log)), strings.Join(lines, "\n"), more)
	}
}

// releases says where the project stands: its last release, and the
// milestones open, for the issues to be put in.
func releases(repo string, b forge.Backlog) string {
	last := "none yet"
	if out, err := exec.Command("git", "-C", repo, "describe", "--tags", "--abbrev=0").Output(); err == nil {
		last = strings.TrimSpace(string(out))
	}
	open := "none"
	if ms, err := b.Milestones(); err == nil && len(ms) > 0 {
		open = strings.Join(ms, ", ")
	}
	return fmt.Sprintf("Last release: %s. Open milestones: %s.\n", last, open)
}

// sourcesChanged says whether a commit since the issue was last read
// touched one of its sources; an issue with none named is not read again.
func sourcesChanged(repo string, st *backlog.State) bool {
	var paths []string
	for _, s := range st.Sources {
		p, _, _ := strings.Cut(s, "#")
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		return false
	}
	out, err := exec.Command("git", append([]string{"-C", repo, "log", "-1", "--format=%h", st.Judged + "..HEAD", "--"}, paths...)...).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + " […]"
}

// Post passes: each act is checked when the engine applies it, against the
// code and the forge as they are then.
func Post(runDir, repo string) int {
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		v = &verdict.Verdict{}
	}
	v.Status, v.Summary = verdict.Pass, "the acts are checked as they are applied"
	return write(runDir, *v)
}

func writeFindings(runDir string, findings []verdict.Finding) int {
	if len(findings) == 0 {
		return 0
	}
	return write(runDir, verdict.Verdict{Status: verdict.Pass, Findings: findings})
}

func final(runDir string, v verdict.Verdict) int {
	if code := write(runDir, v); code != 0 {
		return code
	}
	return exitNothing
}

func write(runDir string, v verdict.Verdict) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return 0
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "product-owner:", err)
	return 1
}

// preImport gives the agent a share of a file — lines from to to, as the
// command importing it cut them — to tell its items still to do, each to
// open as an issue, its text quoted (docs/spec/backlog-acts.md, "Importing
// a file"). The file's format is the agent's to read, not pre's.
func preImport(runDir, repo, role string, open []forge.Issue) int {
	input := func(name string) string {
		data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
		return strings.TrimSpace(string(data))
	}
	file := input("file")
	var from, to int
	fmt.Sscan(input("from"), &from)
	fmt.Sscan(input("to"), &to)
	if file == "" || from < 1 || to < from {
		return fail(fmt.Errorf("an import names a file and its lines: --input file=<path> from=<n> to=<n>"))
	}
	data, err := exec.Command("git", "-C", repo, "show", "HEAD:"+file).Output()
	if err != nil {
		return fail(fmt.Errorf("%s is not in the commit the run is on", file))
	}
	lines := strings.Split(string(data), "\n")
	if from > len(lines) {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "nothing left to read in " + file})
	}
	to = min(to, len(lines))
	var b strings.Builder
	fmt.Fprintf(&b, "Import: lines %d to %d of `%s`, of %d. Each item still to do becomes an issue.\n\n", from, to, file, len(lines))
	var titles []string
	for _, is := range open {
		if is.Title != backlog.ReportTitle(role) {
			titles = append(titles, fmt.Sprintf("- #%d %s", is.ID, is.Title))
		}
	}
	if len(titles) > 0 {
		fmt.Fprintf(&b, "# The open issues, titles only\n\nAn item one of them already holds is not opened again.\n\n%s\n\n", strings.Join(titles, "\n"))
	}
	fmt.Fprintf(&b, "# `%s`, lines %d to %d\n\n```\n", file, from, to)
	for i := from; i <= to; i++ {
		fmt.Fprintf(&b, "%5d  %s\n", i, lines[i-1])
	}
	b.WriteString("```\n")
	b.WriteString(elsewhere(lines, from, to))
	if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("import\n"), 0o644); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(b.String()), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

const (
	mentionsPerID = 4   // lines given for one id, those of a table or a heading first
	mentionsIDs   = 40  // ids looked for in one share
	mentionChars  = 240 // characters of a line given
)

var itemID = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:-[A-Z][A-Z0-9]*)*-[0-9]+\b`)

// elsewhere gives the lines outside a share that name an id the share
// holds: a file often says an item is done far from the item itself — a
// table of what shipped, a summary — and a share alone would open it again.
// Which line says what is the agent's to read; pre only finds them.
func elsewhere(lines []string, from, to int) string {
	var ids []string
	for _, l := range lines[from-1 : to] {
		for _, id := range itemID.FindAllString(l, -1) {
			if !slices.Contains(ids, id) && len(ids) < mentionsIDs {
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	var b strings.Builder
	for _, id := range ids {
		named := regexp.MustCompile(`(^|[^A-Za-z0-9-])` + regexp.QuoteMeta(id) + `($|[^0-9])`)
		var structure, prose []int
		for i, l := range lines {
			if i+1 >= from && i+1 <= to || !named.MatchString(l) {
				continue
			}
			if t := strings.TrimSpace(l); strings.HasPrefix(t, "|") || strings.HasPrefix(t, "#") {
				structure = append(structure, i)
			} else {
				prose = append(prose, i)
			}
		}
		at := append(structure, prose...)
		if len(at) == 0 {
			continue
		}
		more := ""
		if len(at) > mentionsPerID {
			more = fmt.Sprintf("    (%d more lines name it)\n", len(at)-mentionsPerID)
			at = at[:mentionsPerID]
		}
		sort.Ints(at)
		fmt.Fprintf(&b, "- %s\n", id)
		for _, i := range at {
			fmt.Fprintf(&b, "%6d  %s\n", i+1, clip(lines[i], mentionChars))
		}
		b.WriteString(more)
	}
	if b.Len() == 0 {
		return ""
	}
	return "\n# What the rest of the file says of this share's items\n\nThe lines outside the share that name an id the share holds. An item they say is done, merged or dropped is not opened.\n\n" + b.String()
}
