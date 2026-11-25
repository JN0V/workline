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
	"github.com/JN0V/workline/internal/role"
)

const keptKey = "kept"

// keptQuestion is what the judge is asked of a task's whole change.
const keptQuestion = "The change below closes an issue labelled a task: work that should change no behaviour. " +
	"Does it keep the behaviour? Answer yes only when you are sure it changes none of these: a public interface (an exported name, signature, command, option, file format, or output others rely on); " +
	"the toolchain or build platform (compiler, language or tool version, build file, dependency, target); a limit or value a person chose (a timeout, size, threshold, default, text shown). " +
	"Answer no, naming which, when it changes one, or when what is shown does not let you tell."

// askKept writes the judge's question on the change, on a merge request
// whose project lets a task merge alone and whose issue is a task: the
// issues it closes and its whole diff, base to head. A diff past
// diff-lines-max is not asked: said, the task waits for a person.
func (st *state) askKept(runDir, repo string, s Settings) error {
	if mergeRequest() == nil || !s.ForgeWrites || st.Spec != nil || st.Warn == warnOnly || !taskAlone(repo) ||
		!slices.ContainsFunc(st.Issues, func(i ClosedIssue) bool { return i.Kind == backlog.KindTask }) {
		return nil
	}
	diff, err := git(repo, "diff", st.Base, st.Head)
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

// taskAlone says whether the project lets a task merge alone.
func taskAlone(repo string) bool {
	cfg, err := role.LoadProjectConfig(repo)
	return err == nil && cfg.Merge != nil && cfg.Merge.Autonomy == "normal" && slices.Contains(cfg.Merge.Alone, backlog.KindTask)
}

// keptAnswer reads the judge's answer: what the summary says of it, and
// the head on a yes, "" otherwise.
func (st state) keptAnswer(runDir string) (said, head string) {
	if st.KeptNot != "" {
		return "not asked: " + st.KeptNot + ". It waits for a person's yes.", ""
	}
	if !st.Kept {
		return "", ""
	}
	a, err := answerIn(filepath.Join(runDir, "in", "judge", keptKey))
	switch {
	case err != nil || a.Yes == nil || a.Unsure:
		why := a.Error
		if why == "" {
			why = "the judge gave no answer"
		}
		return "not answered (" + why + "). It waits for a person's yes.", ""
	case !*a.Yes:
		return fmt.Sprintf("no, the judge (%s): %s. It waits for a person's yes.", a.Model, strings.TrimRight(oneLine(a.Why, 300), ".")), ""
	}
	return fmt.Sprintf("yes, the judge (%s): %s.", a.Model, strings.TrimRight(oneLine(a.Why, 300), ".")), st.Head
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
