package forge

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// A finding left as a conversation on a merge request's line is one more
// thing a base requiring every conversation resolved holds the merge on:
// code scanning opens one for each alert on a line the change touched,
// whatever its level (#412, docs/research/ci-and-forge.md). What needs no
// answer is shown beside its line instead, as a check's annotation, which
// opens none.

// Annotation is a finding shown beside its line on a commit's changes.
type Annotation struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Level   string `json:"level"` // notice, warning or failure
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// FindingsCheck is the check the documentalist's findings on a merge
// request are annotations of.
const FindingsCheck = "workline findings"

// Annotator is a forge that shows findings beside a commit's lines with
// no conversation to resolve: GitHub's check runs, the simulated forge's
// checks. A check written again on the same head is a newer one; the
// forge shows the latest of a name.
type Annotator interface {
	// Annotate writes a completed check of this name on head, neutral,
	// holding the annotations; it returns the page that shows it. A token
	// that may not write checks is a refusal: Refused says so.
	Annotate(head, name, summary string, notes []Annotation) (string, error)
}

// Conversations is what a branch asks of a merge request's conversations
// before it merges.
type Conversations struct {
	Branch   string // the branch read: the default one when none is named
	Required bool   // every conversation must be resolved first
	// Unread says what of the branch's rules could not be read, and why:
	// Required then tells only what was read.
	Unread string
}

// ConversationRules is a forge whose branches may require every
// conversation of a merge request resolved before it merges: GitHub's
// rulesets and branch protection, the simulated forge's.
type ConversationRules interface {
	Conversations(branch string) (Conversations, error)
}

var (
	_ = []Annotator{&github{}, &Fake{}}
	_ = []ConversationRules{&github{}, &Fake{}}
)

// annotationsPerCall is how many annotations GitHub takes in one write of
// a check run: more are added by updating it.
const annotationsPerCall = 50

// Annotate creates the check run, completed and neutral, its first fifty
// annotations with it, and adds the rest fifty an update. Creating one
// needs a GitHub App's token with checks: write, the job's own included.
func (g *github) Annotate(head, name, summary string, notes []Annotation) (string, error) {
	type annotation struct {
		Path    string `json:"path"`
		Start   int    `json:"start_line"`
		End     int    `json:"end_line"`
		Level   string `json:"annotation_level"`
		Title   string `json:"title,omitempty"`
		Message string `json:"message"`
	}
	type output struct {
		Title       string       `json:"title"`
		Summary     string       `json:"summary"`
		Annotations []annotation `json:"annotations"`
	}
	title := fmt.Sprintf("%d %s", len(notes), map[bool]string{true: "finding", false: "findings"}[len(notes) == 1])
	batch := func(from int) output {
		o := output{Title: title, Summary: summary, Annotations: []annotation{}}
		for _, n := range notes[from:min(from+annotationsPerCall, len(notes))] {
			o.Annotations = append(o.Annotations, annotation{Path: n.Path, Start: n.Line, End: n.Line, Level: n.Level, Title: n.Title, Message: n.Message})
		}
		return o
	}
	input, _ := json.Marshal(struct {
		Name       string `json:"name"`
		Head       string `json:"head_sha"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		Output     output `json:"output"`
	}{name, head, "completed", "neutral", batch(0)})
	out, err := runInput(g.repo, input, "gh", "api", "-X", "POST", "repos/{owner}/{repo}/check-runs", "--input", "-")
	if err != nil {
		return "", checksRight(err)
	}
	var created struct {
		ID  int64  `json:"id"`
		URL string `json:"html_url"`
	}
	if err := decode(out, &created); err != nil {
		return "", err
	}
	for from := annotationsPerCall; from < len(notes); from += annotationsPerCall {
		input, _ := json.Marshal(struct {
			Output output `json:"output"`
		}{batch(from)})
		if _, err := runInput(g.repo, input, "gh", "api", "-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/check-runs/%d", created.ID), "--input", "-"); err != nil {
			return created.URL, checksRight(err)
		}
	}
	return created.URL, nil
}

// checksRight says which right a refused check needs.
func checksRight(err error) error {
	if refusal(err) && strings.Contains(err.Error(), "HTTP 403") {
		return fmt.Errorf("%w (the token needs checks: write; a GitHub App's, the permission Checks: read and write)", err)
	}
	return err
}

// Conversations reads the rules on the branch, the default one when none
// is named: its rulesets', which any token reads, then its classic
// protection's, which only an admin's does — said when it could not.
func (g *github) Conversations(branch string) (Conversations, error) {
	if branch == "" {
		out, err := g.api("repos/{owner}/{repo}", "--jq", ".default_branch")
		if err != nil {
			return Conversations{}, err
		}
		branch = strings.TrimSpace(string(out))
	}
	c := Conversations{Branch: branch}
	out, err := g.api("repos/{owner}/{repo}/rules/branches/"+url.PathEscape(branch), "--jq",
		`[.[] | select(.type == "pull_request") | .parameters.required_review_thread_resolution == true] | any`)
	if err != nil && !errors.Is(err, errNotFound) {
		return c, err
	}
	c.Required = strings.TrimSpace(string(out)) == "true"
	if c.Required {
		return c, nil
	}
	out, err = g.api("repos/{owner}/{repo}/branches/"+url.PathEscape(branch)+"/protection", "--jq",
		`.required_conversation_resolution.enabled == true`)
	switch {
	case err == nil:
		c.Required = strings.TrimSpace(string(out)) == "true"
	case strings.Contains(err.Error(), "Branch not protected"):
	case errors.Is(err, errNotFound) || refusal(err) || strings.Contains(err.Error(), "HTTP 401"):
		c.Unread = "its classic branch protection, which only an admin's token reads"
	default:
		return c, err
	}
	return c, nil
}

// Annotate keeps the check on head, neutral, as GitHub does; refused
// whole when the forge refuses checks.
func (f *Fake) Annotate(head, name, summary string, notes []Annotation) (string, error) {
	if s, err := f.load(); err != nil || s.ChecksRefused != "" {
		if err == nil {
			err = fmt.Errorf("%w: %s", errForbidden, s.ChecksRefused)
		}
		return "", err
	}
	err := f.write(func(s *FakeState) error {
		s.Checks = append(s.Checks, FakeCheck{Commit: head, Name: name, Conclusion: "neutral", Summary: summary, Annotations: append([]Annotation{}, notes...)})
		return nil
	})
	return "", err
}

// Conversations reads the branches the simulated forge says require them;
// its default branch is main.
func (f *Fake) Conversations(branch string) (Conversations, error) {
	s, err := f.load()
	if err != nil {
		return Conversations{}, err
	}
	if branch == "" {
		branch = "main"
	}
	c := Conversations{Branch: branch, Unread: s.ConversationsUnread}
	for _, b := range s.ConversationsRequired {
		c.Required = c.Required || b == branch
	}
	return c, nil
}
