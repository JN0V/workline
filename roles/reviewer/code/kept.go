package reviewer

// A task that keeps the behaviour (docs/adr/0073-a-task-that-changes-no-behaviour-may-merge-alone.md): where the project lets a
// task merge alone (`merge: {autonomy: normal, alone: [task]}`), the
// reviewer asks the judge, once a run its lenses read, whether the whole
// change keeps the behaviour: no public interface, toolchain, build
// platform, nor chosen limit or value changed. A yes is written on the
// summary, by head, for `workline follow`; a no, unsure, or no answer,
// said, writes none: the task waits for a person's yes.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/kit/reviewmark"
	"github.com/JN0V/workline/internal/role"
)

const keptKey = "kept"

// keptQuestion is what the judge is asked of a task's whole change.
const keptQuestion = "The change below closes an issue labelled a task: work that should change no behaviour; the label may be wrong. " +
	"Does it keep the behaviour: can no user or caller observe anything it adds, removes or changes? " +
	"Answer no, naming what, when it changes, for example: a public interface (an exported name, a declaration in a public header, a command, option or output, a protocol message or topic, a stored key, file format or data layout); " +
	"the toolchain or build platform (a compiler, language or tool version, a build file, a dependency, a target); a limit or value a person chose (a timeout, size, threshold, default, text shown); " +
	"a test, fixture, golden file or snapshot that was there changed, or a new file that changes which tests run or how (a test main, conftest, test config). " +
	"Answer yes only when you are sure none of this happens; no when what is shown does not let you tell."

// askKept writes the judge's question on the change, on a merge request
// whose project lets a task merge alone and whose issue is a task: the
// issues it closes and its whole diff, base to head. A diff past
// diff-lines-max is not asked: said, the task waits for a person.
func (st *state) askKept(runDir, repo string, s Settings) error {
	if mergeRequest() == nil || !s.ForgeWrites || st.Spec != nil || st.Warn == warnOnly || !asksKept(repo) ||
		!slices.ContainsFunc(st.Issues, func(i ClosedIssue) bool { return i.Kind == backlog.KindTask }) {
		return nil
	}
	diff, err := git(repo, "diff", "--no-color", "--no-ext-diff", st.Base, st.Head)
	if err != nil {
		return err
	}
	if n := strings.Count(diff, "\n"); s.DiffLinesMax > 0 && n > s.DiffLinesMax {
		st.KeptNot = fmt.Sprintf("the change holds %d lines of diff, past diff-lines-max (%d)", n, s.DiffLinesMax)
		return nil
	}
	var b strings.Builder
	b.WriteString("## The issues it closes\n\n")
	for _, i := range st.Issues {
		b.WriteString(i.Text + "\n")
	}
	fmt.Fprintf(&b, "\n## The whole change\n\n```diff\n%s```\n", diff)
	dir := filepath.Join(runDir, "in", "judge", keptKey)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, _ := json.Marshal(map[string]string{"question": keptQuestion, "material": b.String()}) // JSON, which YAML reads
	st.Kept = true
	return os.WriteFile(filepath.Join(dir, "question.yaml"), data, 0o644)
}

// asksKept says whether the project lets a task merge alone, or asks
// the judge in shadow, nothing merging on its answer (`merge.shadow`).
func asksKept(repo string) bool {
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil || cfg.Merge == nil {
		return false
	}
	return cfg.Merge.Autonomy == "normal" && slices.Contains(cfg.Merge.Alone, backlog.KindTask) || slices.Contains(cfg.Merge.Shadow, backlog.KindTask)
}

// keptAnswer reads the judge's answer: what the summary says of it, the
// head on a yes, "" otherwise, and the answer as the judged record keeps
// it for the auditor's sample, nil when none.
func (st state) keptAnswer(runDir string) (said, head string, judged *reviewmark.Answered) {
	if st.KeptNot != "" {
		return "not asked: " + st.KeptNot + ". It waits for a person's yes.", "", nil
	}
	if !st.Kept {
		return "", "", nil
	}
	a, err := answerIn(filepath.Join(runDir, "in", "judge", keptKey))
	if err != nil || a.Yes == nil || a.Unsure {
		why := a.Error
		if why == "" {
			why = "the judge gave no answer"
		}
		return "not answered (" + why + "). It waits for a person's yes.", "", nil
	}
	why := strings.TrimRight(oneLine(a.Why, 300), ".")
	j := &reviewmark.Answered{Lens: keptKey, Where: st.issueRef(), At: st.Head, Yes: *a.Yes, Finder: a.Author, Judge: a.Model, Title: "the task keeps the behaviour", Why: oneLine(a.Why, 200)}
	if !*a.Yes {
		return fmt.Sprintf("no, the judge (%s): %s. It waits for a person's yes.", a.Model, why), "", j
	}
	return fmt.Sprintf("yes, the judge (%s): %s.", a.Model, why), st.Head, j
}

// issueRef names the task the change closes, `#4`.
func (st state) issueRef() string {
	for _, i := range st.Issues {
		if i.Kind == backlog.KindTask {
			return i.Ref()
		}
	}
	return ""
}

// keptSaid is the summary comment's section on the judge's answer, and
// the hidden line `workline follow` reads on a yes, above the head the
// summary names (forge.At).
func keptSaid(said string) string {
	if said == "" {
		return ""
	}
	return "**Behaviour kept**, for a task that may merge alone: " + said + "\n\n"
}
