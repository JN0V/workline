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
	// Sticky keeps one comment carrying marker on t, edited to body on each
	// run instead of a new one each time; with create false, it only edits
	// one already there.
	Sticky(t Target, body, marker string, create bool) error
	// Label adds and removes labels; adding one already there changes nothing.
	Label(t Target, add, remove []string) error
	// OpenIssue creates an issue, or comments on an open one with the same title.
	OpenIssue(title, body, marker string) (int, error)
	// KeepIssue rewrites the body of the open issue with this title, or opens
	// it when there is none and create is true: one issue, kept in place.
	KeepIssue(title, body string, create bool) (int, error)
	// Release publishes notes for an existing tag, unless already published.
	Release(tag, notes string) error
	// OpenMergeRequest opens a merge request from branch into base, or
	// updates the title and body of the one already open from branch.
	OpenMergeRequest(branch, base, title, body string) (int, error)
	// OpenMergeRequests lists the branches of the open merge requests whose
	// branch starts with prefix, sorted.
	OpenMergeRequests(prefix string) ([]string, error)
	// MergeRequestBranch is the branch a merge request comes from, and
	// whether it lives in this repository (not in a fork).
	MergeRequestBranch(id int) (branch string, here bool, err error)
}

// ErrUnreachable marks a forge that did not answer: the run is blocked by
// something outside the role.
var ErrUnreachable = errors.New("forge unreachable")

// Open returns the forge named by spec: "github", "gitlab", "local" (kept in
// the clone), "cmd:<command>" (another forge, plugged by a command),
// "fake:<file>", or "" / "none" for no forge: what needs one is refused, and
// says so.
func Open(spec, repo string) (Forge, error) {
	switch {
	case spec == "" || spec == "none":
		return nil, nil
	case spec == "github":
		return &github{repo: repo}, nil
	case spec == "gitlab":
		return &gitlab{repo: repo}, nil
	case spec == "local":
		return &Local{Repo: repo}, nil
	case strings.HasPrefix(spec, "cmd:"):
		if strings.TrimSpace(strings.TrimPrefix(spec, "cmd:")) == "" {
			return nil, fmt.Errorf("forge %q: name the command to run (cmd:<command>)", spec)
		}
		return &command{repo: repo, script: strings.TrimPrefix(spec, "cmd:")}, nil
	case strings.HasPrefix(spec, "fake:"):
		return &Fake{Path: strings.TrimPrefix(spec, "fake:")}, nil
	}
	return nil, fmt.Errorf("unknown forge %q (github, gitlab, local, cmd:<command>, fake:<file>, none)", spec)
}

// Missing says what to do when a write needs a forge and the project has none.
const Missing = "set `forge:` in .workline/config.yaml or pass --forge: github, gitlab, cmd:<command>, or `forge: local` to keep it in this clone"

// KeepsBranches says whether f's merge requests come from local branches:
// nothing is pushed to a remote, nor fetched from one.
func KeepsBranches(f Forge) bool {
	_, ok := f.(interface{ keepsBranches() })
	return ok
}

// Marker is the hidden text that makes a comment findable again.
func Marker(key string) string { return "<!-- workline:" + key + " -->" }
