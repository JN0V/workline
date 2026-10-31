package line

// GitLab's issue webhook, pointed at the project's own trigger URL, starts
// a pipeline whose event is in the file $TRIGGER_PAYLOAD (ADR-0059,
// decision 8): the engine reads it itself. A state's label put on an
// issue names that issue; any other event — a comment, an edit, a label
// that is no state — runs nothing, at once.

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/routing"
)

// PayloadEnv names the variable GitLab gives a pipeline a trigger
// started: the file holding the webhook's event, or the event itself.
const PayloadEnv = "TRIGGER_PAYLOAD"

// payload is what the engine reads of a GitLab issue event.
type payload struct {
	Kind  string `json:"object_kind"`
	Attrs struct {
		IID int `json:"iid"`
	} `json:"object_attributes"`
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	Changes struct {
		Labels *struct {
			Previous []struct {
				Title string `json:"title"`
			} `json:"previous"`
			Current []struct {
				Title string `json:"title"`
			} `json:"current"`
		} `json:"labels"`
	} `json:"changes"`
}

// readPayload is the issue a webhook's event names, when it put a state's
// label on it; 0 and why not, else. An event that does not read is an
// error.
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
	if p.Kind != "issue" {
		return 0, fmt.Sprintf("a %q event, not an issue's: nothing to run", p.Kind), nil
	}
	if p.Changes.Labels == nil {
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
