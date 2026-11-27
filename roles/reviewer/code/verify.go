package reviewer

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/kit/worktree"
)

// Verified is what the verifier (model.tasks.verify, tasks/verify.md)
// said in shadow: the points shown, C1 first, and its note, whole.
type Verified struct {
	Shown  []string `json:"shown"`
	Said   string   `json:"said,omitempty"`
	Unread string   `json:"unread,omitempty"` // why no answer was read
}

var verdictLine = regexp.MustCompile(`(?m)^[\s>*-]*\**(KEEP|DROP|BLOCK)\b`)

// askVerifier writes the verifier's task in shadow, for a review of the
// whole change that passes — an agent it cannot reach then changes no
// verdict — on no verdict's run of a review split (ADR-0050): every
// important finding and question the lenses raised, whatever its judge
// said; the change; a read tree at its head for an agent that reads files.
func askVerifier(runDir, repo string, s Settings, st state, c candidates, v Review) error {
	if s.Verifier != "shadow" || st.Warn == warnApart || v.Status != verdict.Pass || st.From != st.Base || st.Upto != "" {
		return nil
	}
	var points []Finding
	for _, g := range append(slices.Clone(c.Related), c.Outside...) {
		points = append(points, slices.DeleteFunc(members(g), func(f Finding) bool { return f.Severity != "important" })...)
	}
	var shown []string
	b := strings.Builder{}
	b.WriteString("## A first review\n\n")
	for i, f := range append(points, c.Questions...) {
		fmt.Fprintf(&b, "### C%d — %s lens, at %s\n\n%s\n\n%s\n\n", i+1, f.Lens, f.Where, f.Title, strings.TrimSpace(f.Why+"\n"+f.Decision))
		shown = append(shown, fmt.Sprintf("C%d %s, at %s: %s", i+1, f.Lens, f.Where, f.Title))
	}
	files, err := codeFiles(repo, s, st.Base, st.Head)
	if err != nil {
		return err
	}
	diff, err := git(repo, append([]string{"diff", "--no-color", "--no-ext-diff", "--find-renames", st.Base, st.Head, "--"}, pathspec(files)...)...)
	if err != nil {
		return err
	}
	where := "You have no files but this task."
	if agent.ReadsFiles(os.Getenv("WORKLINE_AI")) {
		if err := worktree.RunTree(repo, st.Head, runDir, agent.ReadTreeDir); err != nil {
			return err
		}
		where = "Your working directory holds the repository at the change's head."
	}
	head := fmt.Sprintf("# The change to verify\n\nThe commits %s..%s. %s\n\n%s\n%s", short(st.Base), short(st.Head), where, st.Testimony, whatFor(st.Issues, st.Beyond))
	room, err := taskRoom(runDir, repo, "verify")
	if left := room - len(head) - b.Len() - 100; err == nil && room > 0 && len(diff) > left {
		diff = diff[:max(left, 0)] + "\n… (cut: the rest is in the files)\n"
	}
	return errors.Join(err, writeJSON(filepath.Join(runDir, "in", "verify.json"), Verified{Shown: shown}),
		os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("verify\n"), 0o644),
		os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(head+"\n## The change\n\n```diff\n"+diff+"```\n\n"+b.String()), 0o644))
}

// verified records the verifier's note in the review and its summary,
// and takes it out of what the run applies: in shadow it changes nothing.
func verified(runDir string, v *Review) error {
	var vf Verified
	path := filepath.Join(runDir, "out", "intentions.yaml")
	intents, err := intent.Read(path)
	if readJSON(filepath.Join(runDir, "in", "verify.json"), &vf) != nil || err != nil {
		return nil // not asked; or an answer the engine said it could not read
	}
	intents = slices.DeleteFunc(intents, func(in intent.Intention) bool {
		if in.Kind == "note" {
			vf.Said += fmt.Sprintln(in.Value)
		}
		return in.Kind == "note"
	})
	n := map[string]int{}
	for _, m := range verdictLine.FindAllStringSubmatch(vf.Said, -1) {
		n[m[1]]++
	}
	if os.Getenv("WORKLINE_AGENT_ANSWERED") != "true" {
		vf.Unread = cmp.Or(os.Getenv("WORKLINE_AGENT_UNREAD"), os.Getenv("WORKLINE_AGENT_REFUSED"), "not asked: the agent unreachable, or ai-max-tokens spent")
		v.Summary += "; the verifier, in shadow, gave no answer that reads (" + oneLine(vf.Unread, 120) + ")"
	} else {
		v.Summary += fmt.Sprintf("; the verifier, in shadow (it changes nothing): of %d %s shown, keeps %d, drops %d; adds %d",
			len(vf.Shown), plural(len(vf.Shown), "point", "points"), n["KEEP"], n["DROP"], n["BLOCK"])
	}
	v.Verifier = &vf
	if len(intents) == 0 {
		return os.WriteFile(path, []byte("[]\n"), 0o644)
	}
	return intent.Write(path, intents)
}
