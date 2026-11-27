package reviewer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/worktree"
)

// Verified is what the verifier (tasks/verify.md) said in shadow, asked
// once the review was posted: it changes nothing of the review.
type Verified struct {
	Shown    []string `json:"shown,omitempty"`     // the points shown, C1 first
	Kept     []string `json:"kept,omitempty"`      // C<n> — why
	Dropped  []string `json:"dropped,omitempty"`   // C<n> — why
	Added    []string `json:"added,omitempty"`     // where — why
	NoAnswer string   `json:"no-answer,omitempty"` // why it gave none
}

// askVerifier leaves the engine, in shadow, the verifier's task in out/shadow,
// on a review of the whole change but the run apart (ADR-0050).
func askVerifier(runDir, repo string, v *Review) {
	st, c, ai, b, shown := state{}, candidates{}, os.Getenv("WORKLINE_AI"), strings.Builder{}, []string{}
	s, err := settings(runDir)
	if err != nil || s.Verifier != "shadow" || v.Subject == "spec" || readJSON(filepath.Join(runDir, "in", "review-state.json"), &st) != nil ||
		st.Warn == warnOnly || st.From != st.Base || st.Upto != "" || readJSON(filepath.Join(runDir, "in", "candidates.json"), &c) != nil {
		return
	}
	for _, g := range append(append(slices.Clone(c.Related), c.Outside...), c.Questions...) { // every point but nits, whatever its judge said
		for _, f := range slices.DeleteFunc(members(g), func(f Finding) bool { return f.Severity == "nit" }) {
			fmt.Fprintf(&b, "### C%d — %s lens, at %s\n\n%s\n\n%s\n\n", len(shown)+1, f.Lens, f.Where, f.Title, strings.TrimSpace(f.Why+"\n"+f.Decision))
			shown = append(shown, fmt.Sprintf("C%d %s, at %s: %s", len(shown)+1, f.Lens, f.Where, f.Title))
		}
	}
	files, err := codeFiles(repo, s, st.Base, st.Head)
	diff, err2 := git(repo, append([]string{"diff", "--no-color", "--no-ext-diff", "--find-renames", st.Base, st.Head, "--"}, pathspec(files)...)...)
	dir, where := filepath.Join(runDir, "out", "shadow"), "You have no files but this task."
	err3 := os.MkdirAll(filepath.Join(dir, "in"), 0o755)
	if strings.HasPrefix(ai, "cmd:") {
		err = errors.New("a `cmd:` agent reads no file of the repository")
	} else if agent.ReadsFiles(ai) && errors.Join(err, err2, err3) == nil {
		where, err3 = "Your working directory holds the repository at the change's head.", worktree.RunTree(repo, st.Head, dir, agent.ReadTreeDir)
	}
	task := fmt.Sprintf("# The change to verify\n\nThe commits %s..%s. %s\n\n%s\n%s\n## The change\n\n```diff\n%s```\n\n## A first review\n\n%s",
		short(st.Base), short(st.Head), where, st.Testimony, whatFor(st.Issues, st.Beyond), capLines(diff, s.DiffLinesMax, "the rest is in the files"), b.String())
	if err = errors.Join(err, err2, err3); err == nil {
		err = errors.Join(os.WriteFile(filepath.Join(dir, "in", "task-kind"), []byte("verify\n"), 0o644), os.WriteFile(filepath.Join(dir, "in", "task.md"), []byte(task), 0o644))
	}
	if v.Verifier = (&Verified{Shown: shown}); err != nil {
		*v = unverified(*v, s, err.Error())
	}
}

// unverified says, in shadow, that the verifier gave no answer, and why.
func unverified(v Review, s Settings, why string) Review {
	if s.Verifier == "shadow" {
		v.Verifier, v.Summary = &Verified{NoAnswer: why}, v.Summary+"; the verifier gave no answer: "+oneLine(why, 200)
	}
	return v
}

// verified is post run again once the verifier was asked (WORKLINE_SHADOW:
// answered, or why not): its answer, or a refusal, is recorded.
func verified(runDir, why string) int {
	var v Review
	if err := readJSON(filepath.Join(runDir, "out", "review.json"), &v); err != nil || v.Verifier == nil {
		return fail(errors.Join(errors.New("no review to record the verifier's answer in"), err))
	}
	dir, vf, said := filepath.Join(runDir, "out", "shadow"), v.Verifier, ""
	intents, err := intent.Read(filepath.Join(dir, "out", "intentions.yaml"))
	if err != nil {
		said = err.Error()
	}
	for _, in := range intents {
		said += fmt.Sprintln(in.Value)
	}
	to := map[string]*[]string{"KEEP": &vf.Kept, "DROP": &vf.Dropped, "BLOCK": &vf.Added}
	for _, l := range strings.Split(said, "\n") {
		if f := strings.Fields(strings.TrimLeft(l, " >*-")); len(f) > 1 && to[f[0]] != nil {
			*to[f[0]] = append(*to[f[0]], strings.Join(f[1:], " "))
		}
	}
	if why == "answered" && len(vf.Kept)+len(vf.Dropped)+len(vf.Added) == 0 && !strings.Contains(said, "no blocking defect") {
		why = "refused, in none of its forms: " + oneLine(said, 200)
	}
	line := fmt.Sprintf("verifier (shadow): %d kept, %d dropped, %d added", len(vf.Kept), len(vf.Dropped), len(vf.Added))
	if why != "answered" {
		*vf, line = Verified{Shown: vf.Shown, NoAnswer: why}, "the verifier gave no answer: "+oneLine(why, 200)
	}
	if err := errors.Join(writeJSON(filepath.Join(runDir, "out", "review.json"), v), os.WriteFile(filepath.Join(dir, "said"), []byte(line+"\n"), 0o644)); err != nil {
		return fail(err)
	}
	return 0
}
