package line

// GitLab's issue webhook, pointed at the project's own trigger URL, starts
// a pipeline whose event is in the file $TRIGGER_PAYLOAD (ADR-0059,
// decision 8): the engine reads it itself. A state's label put on an
// issue names that issue; so does the reviewer's comment on the spec of
// an issue in refine, which the product owner reads next (#846), and a
// person's comment or body edit on an issue on the line, an answer the
// line may wait on (#1414). Any other event — the engine's other
// comments, an edit of an issue off the line, a label that is no state
// — runs nothing, at once.

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/routing"
)

// PayloadEnv names the variable GitLab gives a pipeline a trigger
// started: the file holding the webhook's event, or the event itself.
const PayloadEnv = "TRIGGER_PAYLOAD"

// title is a label as an event names it.
type title struct {
	Title string `json:"title"`
}

// payload is what the engine reads of a GitLab issue or note event.
type payload struct {
	Kind  string `json:"object_kind"`
	Attrs struct {
		IID          int    `json:"iid"`
		Note         string `json:"note"`
		NoteableType string `json:"noteable_type"`
	} `json:"object_attributes"`
	// Labels: on an issue event, the issue's labels after it.
	Labels *[]title `json:"labels"`
	// Issue: on a note event, the issue commented on, its labels as they
	// were then.
	Issue *struct {
		IID    int      `json:"iid"`
		Labels *[]title `json:"labels"`
	} `json:"issue"`
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	Changes struct {
		Description *json.RawMessage `json:"description"`
		Labels      *struct {
			Previous []title `json:"previous"`
			Current  []title `json:"current"`
		} `json:"labels"`
	} `json:"changes"`
}

// readPayload is the issue a webhook's event names, when it put a state's
// label on it, or is the reviewer's spec review on it in refine; 0 and
// why not, else. An event that does not read is an error.
func readPayload(raw string, cfg *routing.Config) (int, string, error) {
	data := []byte(raw)
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		var err error
		if data, err = os.ReadFile(raw); err != nil {
			return 0, "", fmt.Errorf("$%s: %w", PayloadEnv, err)
		}
	}
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return 0, "", fmt.Errorf("$%s does not read as a webhook's event: %w", PayloadEnv, err)
	}
	if p.Kind == "note" {
		n, why := readNote(p, cfg)
		return n, why, nil
	}
	if p.Kind != "issue" {
		return 0, fmt.Sprintf("a %q event, not an issue's: nothing to run", p.Kind), nil
	}
	if p.Changes.Labels == nil {
		// A body edited on an issue on the line (#1414): a person's answer
		// to a draft or a block, perhaps; the line reads the issue and
		// says whether its step is due, with no agent.
		if p.Changes.Description != nil {
			if onLine(p.Labels, cfg) {
				return p.Attrs.IID, "", nil
			}
			return 0, fmt.Sprintf("#%d's body edited, the issue on no state: nothing to run", p.Attrs.IID), nil
		}
		return 0, fmt.Sprintf("#%d changed, its labels did not: nothing to run", p.Attrs.IID), nil
	}
	var before []string
	for _, l := range p.Changes.Labels.Previous {
		before = append(before, l.Title)
	}
	for _, l := range p.Changes.Labels.Current {
		if slices.Contains(before, l.Title) {
			continue
		}
		for _, s := range cfg.States {
			if s.Label == l.Title {
				return p.Attrs.IID, "", nil
			}
		}
	}
	return 0, fmt.Sprintf("#%d got no state's label (by %s): nothing to run", p.Attrs.IID, p.User.Username), nil
}

// readNote is the issue a comment names: the reviewer's on the spec of an
// issue in refine, whose step it makes due again (#128), the product
// owner reading it then (#846); a person's on an issue on the line, an
// answer the line may wait on, a block's or a draft's (#1414); 0 and why
// not, else. The engine's other comments — its records and states, each
// holding a workline marker — run nothing: the issue's step is what it
// was before them (one event, at most one run; #813). Who wrote it is not
// read further: the line reads the issue and its record, and a step not
// due runs nothing.
func readNote(p payload, cfg *routing.Config) (int, string) {
	if p.Attrs.NoteableType != "Issue" || p.Issue == nil {
		return 0, fmt.Sprintf("a comment on a %s, not on an issue: nothing to run", cmp.Or(p.Attrs.NoteableType, "thing unnamed"))
	}
	n := p.Issue.IID
	if !strings.Contains(p.Attrs.Note, backlog.SpecMarker) {
		if strings.Contains(p.Attrs.Note, "<!-- workline:") {
			return 0, fmt.Sprintf("#%d got a comment, not the reviewer's on its spec: nothing to run", n)
		}
		if onLine(p.Issue.Labels, cfg) {
			return n, ""
		}
		return 0, fmt.Sprintf("#%d got a person's comment, the issue on no state: nothing to run", n)
	}
	if p.Issue.Labels == nil { // an event naming no labels: the line reads the issue and says
		return n, ""
	}
	for _, s := range cfg.States {
		if s.Name == refineState && slices.ContainsFunc(*p.Issue.Labels, func(l title) bool { return l.Title == s.Label }) {
			return n, ""
		}
	}
	return 0, fmt.Sprintf("#%d got the reviewer's comment on its spec, not in %s: nothing to run", n, refineState)
}

// onLine says whether an issue bearing labels is on the line: one of
// them a state's; labels not named, the line reads the issue and says.
func onLine(labels *[]title, cfg *routing.Config) bool {
	if labels == nil {
		return true
	}
	return slices.ContainsFunc(cfg.States, func(s routing.State) bool {
		return slices.ContainsFunc(*labels, func(l title) bool { return l.Title == s.Label })
	})
}
