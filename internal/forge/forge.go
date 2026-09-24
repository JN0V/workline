// Package forge applies what reaches a forge — comments, labels, issues,
// releases — on GitHub, GitLab, or a simulated forge for the tests. Every
// write is idempotent, so a run interrupted half-way can be resumed.
package forge

import (
	"errors"
	"fmt"
	"strings"
)

// Target is what a comment or a label goes on.
type Target struct {
	Kind string `json:"kind" yaml:"kind"` // "issue" or "merge-request"
	ID   int    `json:"id" yaml:"id"`
}

func (t Target) String() string { return fmt.Sprintf("%s #%d", t.Kind, t.ID) }

// Issue is what the line reads from a work item.
type Issue struct {
	ID     int      `json:"id"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

// Forge is what the engine needs from one.
type Forge interface {
	Issue(id int) (*Issue, error)
	// Comment posts body on t unless a comment carrying marker is already there.
	Comment(t Target, body, marker string) error
	// Label adds and removes labels; adding one already there changes nothing.
	Label(t Target, add, remove []string) error
	// OpenIssue creates an issue, or comments on an open one with the same title.
	OpenIssue(title, body, marker string) (int, error)
	// Release publishes notes for an existing tag, unless already published.
	Release(tag, notes string) error
}

// ErrUnreachable marks a forge that did not answer: the run is blocked by
// something outside the role.
var ErrUnreachable = errors.New("forge unreachable")

// Open returns the forge named by spec: "github", "gitlab", "fake:<file>",
// or "" / "none" for no forge.
func Open(spec, repo string) (Forge, error) {
	switch {
	case spec == "" || spec == "none":
		return nil, nil
	case spec == "github":
		return &github{repo: repo}, nil
	case spec == "gitlab":
		return &gitlab{repo: repo}, nil
	case strings.HasPrefix(spec, "fake:"):
		return &Fake{Path: strings.TrimPrefix(spec, "fake:")}, nil
	}
	return nil, fmt.Errorf("unknown forge %q (github, gitlab, fake:<file>, none)", spec)
}

// Marker is the hidden text that makes a comment findable again.
func Marker(key string) string { return "<!-- workline:" + key + " -->" }
