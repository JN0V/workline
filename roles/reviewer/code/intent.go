package reviewer

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/work"
)

// ClosedIssue is an issue the change says it closes, as the lenses read it.
type ClosedIssue struct {
	ID        int             `json:"id"`
	Text      string          `json:"text"`                // its title and the sections of intentSections, as given
	Scenarios []work.Scenario `json:"scenarios,omitempty"` // its Validation's scenarios, each the intent lens answers
}

// Ref is how a finding names the issue as its cause's path: `#4`.
func (i ClosedIssue) Ref() string { return fmt.Sprintf("#%d", i.ID) }

// issueOf is the issue the change closes that a cause's path names (`#4`),
// or nil.
func issueOf(st state, path string) *ClosedIssue {
	for i := range st.Issues {
		if st.Issues[i].Ref() == path {
			return &st.Issues[i]
		}
	}
	return nil
}

// whatFor is the section the lenses and a judge read of the issues
// closed, and of the files the author lists beyond their Scope.
func whatFor(issues []ClosedIssue, beyond string) string {
	if len(issues) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## What the change is for: the issue the change closes\n\nAs the issue reads now. A finding quoting it gives its number as the path: `path: \"#4\"`.\n\n")
	for _, is := range issues {
		b.WriteString(is.Text + "\n")
	}
	if beyond != "" {
		fmt.Fprintf(&b, "\n### %s, as the author lists them\n\nThe %s is the product owner's hint, not a fence (ADR-0065): each file below was changed beyond it, its reason the author's. Judge each against the issue.\n\n%s", beyondHeading, work.Name(work.Scope), beyond)
	}
	return b.String()
}

// beyondHeading heads the developer's list of the files it changed
// beyond the issue's Scope, on its pull request and in a round's commit
// (roles/developer/code, BeyondHeading).
var beyondHeading = "Beyond the " + work.Name(work.Scope)

// beyondListed are the bullets listed under a heading or a line
// "Beyond the Scope" in the texts given, the merge request's body and its
// commits' messages, each once, in order: the files changed beyond the
// issue's Scope and why, as the author says (ADR-0065, change 2).
func beyondListed(texts ...string) string {
	var out []string
	for _, t := range texts {
		in := false
		for _, l := range strings.Split(strings.ReplaceAll(t, "\r\n", "\n"), "\n") {
			l = strings.TrimSpace(l)
			switch {
			case strings.TrimSuffix(strings.TrimSpace(strings.TrimLeft(l, "#")), ":") == beyondHeading:
				in = true
			case !in || l == "":
			case strings.HasPrefix(l, "- "):
				if !slices.Contains(out, l) {
					out = append(out, l)
				}
			default:
				in = strings.HasPrefix(l, "Files changed beyond") // the list's own lead line
			}
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

// intentSections are what of an issue a change is compared with: what it
// asks, its example and a bug's steps, the scenarios that accept it, how
// it is proven, and what it leaves out (#126).
var intentSections = []work.Meaning{work.Need, work.Example, work.Steps, work.Validation, work.Verification, work.Scope}

// issueText is what the lenses read of an issue: its title, then each of
// intentSections it holds, under its heading; its whole body when it has
// none of them.
func issueText(is *forge.Issue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### Issue #%d: %s\n", is.ID, is.Title)
	sections := work.ByMeaning(is.Body)
	any := false
	for _, m := range intentSections {
		if body := strings.TrimSpace(sections[m].Text); body != "" {
			fmt.Fprintf(&b, "\n#### %s\n\n%s\n", sections[m].Heading, body)
			any = true
		}
	}
	if !any {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(is.Body))
	}
	return b.String()
}

// readIssues reads the issues the change closes from the forge, up to
// limit lines all together; one it cannot read is said, and the intent
// lens is then not asked of it.
func readIssues(repo string, ids []int, limit int) ([]ClosedIssue, []verdict.Finding) {
	if len(ids) == 0 {
		return nil, nil
	}
	var said []verdict.Finding
	unread := func(id int, why string) {
		said = append(said, verdict.Finding{Rule: "issue-unread", Where: fmt.Sprintf("#%d", id), Level: "warn",
			Message: fmt.Sprintf("the change says it closes #%d, but %s: the intent lens is not asked of it", id, why)})
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil || f == nil {
		why := "no forge to read it from"
		if err != nil {
			why = "the forge: " + err.Error()
		}
		for _, id := range ids {
			unread(id, why)
		}
		return nil, said
	}
	var out []ClosedIssue
	left := limit
	for _, id := range ids {
		is, err := f.Issue(id)
		if err != nil {
			unread(id, "it could not be read ("+err.Error()+")")
			continue
		}
		text := issueText(is)
		scenarios := work.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)
		n := strings.Count(text, "\n")
		if limit > 0 && n > left {
			if left < 8 { // too little left to say what it asks
				unread(id, fmt.Sprintf("the issues closed hold more than issue-lines-max (%d) lines", limit))
				continue
			}
			text = capLines(text, left, "the issue is cut here, past issue-lines-max")
			said = append(said, cutSaid(fmt.Sprintf("#%d", id), "the issue the change closes", n, left, "issue-lines-max"))
			n = left
		}
		left -= n
		// A scenario the cut left out is not asked of the lens.
		scenarios = slices.DeleteFunc(scenarios, func(sc work.Scenario) bool { return len(locate(strings.Split(text, "\n"), sc.Name)) == 0 })
		out = append(out, ClosedIssue{ID: id, Text: text, Scenarios: scenarios})
	}
	return out, said
}

// mergeRequestText is the title and body of the merge request the run is
// on: what its author says of it, and the issues it closes. "" on a machine.
func mergeRequestText(repo string) (string, error) {
	t := mergeRequest()
	if t == nil {
		return "", nil
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil || f == nil {
		return "", err
	}
	mr, err := f.MergeRequest(t.ID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.TrimSpace(mr.Title) + "\n\n" + strings.TrimSpace(mr.Body)), nil
}

// trailer is a commit message's trailer line, as indented in the log:
// `Co-Authored-By: …`, `Signed-off-by: …`, `Workline-Role: …` — who, not
// what the change does. Only these keys: `Follow-up: no change` is a claim.
var trailer = regexp.MustCompile(`(?i)^  (?:co-authored-by|signed-off-by|reviewed-by|workline-[a-z-]+): `)

// testimonyHead opens what the lenses read of the author; never the
// author's own words, so never a claim found again.
const testimonyHead = "## What the author says\n\nTestimony, not evidence: what the author says the change does. Check each claim against the code.\n\n"

// saidByAuthor is the testimony without its head: where a claim is found
// again.
func saidByAuthor(t string) []string {
	return strings.Split(strings.TrimPrefix(t, testimonyHead), "\n")
}

// testimony is what the author says of the change, for the lenses to check
// against it: the messages of the commits not reviewed yet, whole, and the
// merge request's title and body; the commits reviewed before named. The
// base's commits a merge brought in are not the author's (#134), nor those
// left past commits-per-push (ADR-0050). Capped at limit lines; cut, the
// lines it held.
func testimony(repo string, st state, mr string, limit int) (string, int, error) {
	log, err := git(repo, "log", "--format=- %h %s%n%w(0,2,2)%b", st.From+".."+st.lensTo(), "^"+st.Base)
	if err != nil {
		return "", 0, err
	}
	var b strings.Builder
	b.WriteString(testimonyHead + "### The commits\n\n")
	for _, l := range strings.Split(strings.TrimSpace(log), "\n") {
		if strings.TrimSpace(l) != "" && !trailer.MatchString(l) {
			b.WriteString(l + "\n")
		}
	}
	if st.From != st.Base {
		before, err := git(repo, "log", "--format=- %h %s", st.Base+".."+st.From)
		if err != nil {
			return "", 0, err
		}
		fmt.Fprintf(&b, "\nReviewed before, not shown here:\n\n%s", before)
	}
	if mr != "" {
		fmt.Fprintf(&b, "\n### The merge request\n\n%s\n", mr)
	}
	t, cut := cutLines(b.String(), limit, "what the author says is cut here, past testimony-lines-max")
	return t, cut, nil
}

// scenarioOf is the scenario of an issue the change closes a name gives,
// written as the issue writes it; the issue is the one a ref names (`#4`),
// or, none named, the first holding a scenario of that name.
func scenarioOf(st state, ref, name string) (*ClosedIssue, *work.Scenario) {
	for i := range st.Issues {
		is := &st.Issues[i]
		if ref != "" && is.Ref() != ref {
			continue
		}
		for j := range is.Scenarios {
			if strings.EqualFold(norm(is.Scenarios[j].Name), norm(name)) {
				return is, &is.Scenarios[j]
			}
		}
	}
	return nil, nil
}

// shown reads the intent lens's answer that a scenario of the issue is
// done and proven: the code doing it and the test proving it, each quoted
// and found again at the head. One missing either, or whose quote is not
// found again, shows nothing: the scenario is then judged as done or
// proven by nothing shown (unshown).
func (c *candidates) shown(repo string, st state, intentAsked bool, v any) {
	var raw struct {
		Scenario string `json:"scenario"`
		Issue    any    `json:"issue"` // "#4", or 4
		Code     *Quote `json:"code"`
		Test     *Quote `json:"test"`
	}
	data, _ := json.Marshal(v)
	if err := json.Unmarshal(data, &raw); err != nil {
		c.Logged = append(c.Logged, verdict.Finding{Rule: "scenario-unshown", Level: "warn",
			Message: fmt.Sprintf("a scenario answered does not read (%v): shown done by nothing", err)})
		return
	}
	said := func(why string) {
		c.Logged = append(c.Logged, verdict.Finding{Rule: "scenario-unshown", Level: "warn",
			Message: fmt.Sprintf("the scenario %q, shown done: %s", raw.Scenario, why)})
	}
	if !intentAsked {
		said("answered by a lens other than intent, which alone answers scenarios: not read")
		return
	}
	ref := ""
	if raw.Issue != nil {
		ref = "#" + strings.TrimPrefix(strings.TrimSpace(fmt.Sprint(raw.Issue)), "#")
	}
	is, sc := scenarioOf(st, ref, raw.Scenario)
	if sc == nil {
		said("no scenario of that name in the Validation of the issues the change closes")
		return
	}
	for _, q := range []struct {
		what string
		at   *Quote
	}{{"code doing it", raw.Code}, {"test proving it", raw.Test}} {
		if q.at == nil || strings.TrimSpace(q.at.Quote) == "" {
			said(fmt.Sprintf("it quotes no %s: judged as shown by nothing", q.what))
			return
		}
		path := clean(q.at.Path)
		if to, ok := st.Change.Moved[path]; ok {
			path = to
		}
		text, ok := fileAt(repo, st.Head, path)
		if !ok || len(locate(strings.Split(text, "\n"), q.at.Quote)) == 0 {
			said(fmt.Sprintf("its %s is not found in %s at %s: judged as shown by nothing", q.what, path, short(st.Head)))
			return
		}
	}
	if key := is.Ref() + " " + sc.Name; !slices.Contains(c.Shown, key) {
		c.Shown = append(c.Shown, key)
	}
}

// unshown makes a finding of each scenario of the issues the change
// closes that the intent lens, asked and answering, neither showed done
// and proven, its code and test found again, nor found missing: an
// important one, its cause the scenario's name in the issue, which its
// judge, shown the issue and the change, keeps or drops. A "no finding"
// is never taken on the lens's silence.
func (c *candidates) unshown(repo string, s Settings, st state) {
	if st.Spec != nil || !slices.Contains(st.Lenses, "intent") || slices.Contains(c.Failed, "intent") {
		return
	}
	for _, is := range st.Issues {
		for _, sc := range is.Scenarios {
			if slices.Contains(c.Shown, is.Ref()+" "+sc.Name) || c.foundOn(is, sc) {
				continue
			}
			f, why := found(repo, st, "intent", false, map[string]any{
				"severity": "important",
				"title":    fmt.Sprintf("No code and test shown for the scenario %q", sc.Name),
				"why":      fmt.Sprintf("the intent lens quoted no code at the head doing this scenario of %s and no test proving it, nor found either missing: %s", is.Ref(), oneLine(sc.Text, 300)),
				"fix":      "do the scenario, and a test that proves it",
				"cause":    map[string]any{"path": is.Ref(), "quote": sc.Name},
			}, "")
			if why != "" {
				c.Logged = append(c.Logged, verdict.Finding{Rule: "scenario-unshown", Level: "warn",
					Message: fmt.Sprintf("the scenario %q of %s, shown by nothing: %s", sc.Name, is.Ref(), why)})
				continue
			}
			c.put(s, f)
		}
	}
}

// foundOn says whether a finding or a question of the intent lens is on
// a scenario: its cause quoted from the issue naming the scenario, or
// quoting words of this scenario no other scenario of the issue holds.
func (c *candidates) foundOn(is ClosedIssue, sc work.Scenario) bool {
	name := norm(sc.Name)
	only := func(q string) bool {
		if !strings.Contains(norm(sc.Name+" "+sc.Text), q) {
			return false
		}
		for _, o := range is.Scenarios {
			if o.Name != sc.Name && strings.Contains(norm(o.Name+" "+o.Text), q) {
				return false
			}
		}
		return true
	}
	on := func(f Finding) bool {
		q := norm(f.Cause.Quote)
		return f.Lens == "intent" && f.Cause.Path == is.Ref() && q != "" && (strings.Contains(q, name) || only(q))
	}
	for _, f := range slices.Concat(c.Related, c.Questions) {
		if on(f) || slices.ContainsFunc(f.Also, on) {
			return true
		}
	}
	return false
}
