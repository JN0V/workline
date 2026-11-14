package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/wake"
)

// wakeCmd prints where the line may be drifting after a merge to the
// default branch, read from git and the forge with no AI (ADR-0068 §8);
// with --issue, keeps it in one issue on the forge. It never blocks: the
// triggers met exit 0, and so does a forge that does not answer, said.
func wakeCmd(args []string) int {
	fs := newFlags("wake")
	repo := fs.String("repo", ".", "repository")
	rng := fs.String("range", "", "the default branch's tip before the merge and after it, <before>..<after> (default: HEAD's first parent..HEAD)")
	forgeSpec := fs.String("forge", "", "github, gitlab, cmd:<command>, none (default: the project's `forge` setting)")
	issue := fs.Bool("issue", false, "also keep what it prints in one issue on the forge, edited in place, and create the label "+wake.Label+" when missing")
	asJSON := fs.Bool("json", false, "print what it read as JSON")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		return notInRepo()
	}
	cfg, err := role.LoadProjectConfig(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	if *forgeSpec == "" {
		*forgeSpec = cfg.Forge
	}
	before, after := "", "HEAD"
	if *rng != "" {
		var ok bool
		if before, after, ok = strings.Cut(*rng, ".."); !ok || after == "" {
			return misuse(fs, fmt.Sprintf("--range %q: give <before>..<after>", *rng))
		}
	}
	f, err := forge.Open(*forgeSpec, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	b, isBacklog := f.(forge.Backlog)
	if *issue && (f == nil || !isBacklog) {
		fmt.Fprintln(os.Stderr, "workline: --issue needs a forge (--forge github, gitlab…) to keep its issue")
		return 64
	}
	r, err := wake.Read(wake.Options{Repo: root, Before: before, After: after, Forge: f})
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	now, id := "", 0
	if *issue {
		open, err := b.Issues()
		if err != nil {
			r.Unread = append(r.Unread, "the wake's issue: "+forge.Said(err))
		}
		for _, is := range open {
			if is.Title == wake.Title && !is.Closed {
				now, id = is.Body, is.ID
				break
			}
		}
	}
	body, changed, any := wake.Issue(r, now)
	if *asJSON {
		out, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(out))
	} else {
		// The issue's record of what it lists is for the next run, not a
		// reader.
		said, _, _ := strings.Cut(body, wake.Marker)
		fmt.Print(said)
	}
	if !*issue {
		return 0
	}
	// What it keeps on the forge is said on stderr; a write that fails is
	// said, never a red run: the wake holds nothing (principle 15).
	if err := b.EnsureLabel(wake.Label, "d93f0b", "anyone's cord: step back and look at where the line drifts (ADR-0068)"); err != nil {
		fmt.Fprintln(os.Stderr, "workline: the label "+wake.Label+" not created:", forge.Said(err))
	}
	switch {
	case !changed && id != 0:
		fmt.Fprintf(os.Stderr, "workline: nothing new nor gone: issue #%d left as it is\n", id)
	case !changed:
		fmt.Fprintln(os.Stderr, "workline: no trigger met: no issue opened")
	default:
		n, err := f.KeepIssue(wake.Title, backlog.Inert(body), any)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline: the wake's issue not kept:", forge.Said(err))
			return 0
		}
		fmt.Fprintf(os.Stderr, "workline: issue #%d kept\n", n)
	}
	return 0
}
