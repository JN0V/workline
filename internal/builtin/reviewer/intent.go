package reviewer

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// closing is an issue closed by a keyword, as GitHub and GitLab read it:
// `Closes #4`, `fixes: #4`, `Resolved #4`. `Refs #4` closes nothing.
var closing = regexp.MustCompile(`(?i)\b(?:clos(?:e|es|ed|ing)|fix(?:es|ed|ing)?|resolv(?:e|es|ed|ing)|implement(?:s|ed|ing)?):?\s+#(\d+)\b`)

// closes are the issues the texts say the change closes, once each, in order.
func closes(texts ...string) []int {
	var ids []int
	for _, t := range texts {
		for _, m := range closing.FindAllStringSubmatch(t, -1) {
			if n, err := strconv.Atoi(m[1]); err == nil && !slices.Contains(ids, n) {
				ids = append(ids, n)
			}
		}
	}
	return ids
}

// ClosedIssue is an issue the change says it closes, as the lenses read it.
type ClosedIssue struct {
	ID   int    `json:"id"`
	Text string `json:"text"` // its title, Need, Verification and Scope, as given
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

// whatFor is the section the lenses and a judge read of the issues closed.
func whatFor(issues []ClosedIssue) string {
	if len(issues) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## What the change is for: the issue the change closes\n\nAs the issue reads now. A finding quoting it gives its number as the path: `path: \"#4\"`.\n\n")
	for _, is := range issues {
		b.WriteString(is.Text + "\n")
	}
	return b.String()
}

// intentSections are what of an issue a change is compared with: what it
// asks, how it is proven, and what it leaves out (#126).
var intentSections = []string{"Need", "Verification", "Scope"}

// issueText is what the lenses read of an issue: its title, then its Need,
// Verification and Scope; its whole body when it has none of them.
func issueText(is *forge.Issue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### Issue #%d: %s\n", is.ID, is.Title)
	sections := work.Sections(is.Body)
	any := false
	for _, name := range intentSections {
		if body := strings.TrimSpace(sections[name]); body != "" {
			fmt.Fprintf(&b, "\n#### %s\n\n%s\n", name, body)
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
		n := strings.Count(text, "\n")
		if limit > 0 && n > left {
			if left < 8 { // too little left to say what it asks
				unread(id, fmt.Sprintf("the issues closed hold more than issue-lines-max (%d) lines", limit))
				continue
			}
			text = capLines(text, left, "the issue is cut here, past issue-lines-max")
			n = left
		}
		left -= n
		out = append(out, ClosedIssue{ID: id, Text: text})
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
// what the change does. A key without a dash (`Closes: #4`) is kept.
var trailer = regexp.MustCompile(`^  [A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)+: `)

// testimony is what the author says of the change, for the lenses to check
// against it: the messages of the commits not reviewed yet, whole, and the
// merge request's title and body; the commits reviewed before named.
// Capped at limit lines.
func testimony(repo string, st state, mr string, limit int) (string, error) {
	log, err := git(repo, "log", "--format=- %h %s%n%w(0,2,2)%b", st.From+".."+st.Head)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("## What the author says\n\nTestimony, not evidence: what the author says the change does. Check each claim against the code.\n\n### The commits\n\n")
	for _, l := range strings.Split(strings.TrimSpace(log), "\n") {
		if strings.TrimSpace(l) != "" && !trailer.MatchString(l) {
			b.WriteString(l + "\n")
		}
	}
	if st.From != st.Base {
		before, err := git(repo, "log", "--format=- %h %s", st.Base+".."+st.From)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "\nReviewed before, not shown here:\n\n%s", before)
	}
	if mr != "" {
		fmt.Fprintf(&b, "\n### The merge request\n\n%s\n", mr)
	}
	return capLines(b.String(), limit, "what the author says is cut here, past testimony-lines-max"), nil
}
