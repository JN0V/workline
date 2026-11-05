package tester

// A tester short of code (ADR-0063, §6): its agent writes no test and
// names the files it lacks, `needs: <paths> — why`. A needs-input of the
// engine, never "no test possible" (#1047, #1059): the issue keeps its
// state, said on it; the line gives those files at the step's next run.

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// needsWord opens the note naming the files the agent lacks.
const needsWord = "needs:"

// needsOf reads the files a note says the agent lacks, and why; false
// when no note opens with `needs:`.
func needsOf(notes []string) (paths []string, why string, ok bool) {
	for _, n := range notes {
		rest, found := strings.CutPrefix(strings.TrimSpace(n), needsWord)
		if !found {
			continue
		}
		list, reason, _ := strings.Cut(rest, "—")
		if reason == "" {
			list, reason, _ = strings.Cut(rest, " - ")
		}
		for _, p := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
			if p = strings.Trim(p, "`'\""); p != "" && !slices.Contains(paths, p) {
				paths = append(paths, p)
			}
		}
		return paths, strings.TrimSpace(reason), true
	}
	return nil, "", false
}

// needed ends a run whose agent named the files it lacks: those of the
// repository, said on the issue, its state kept; the finding the line
// gives them by (verdict.NeedsCode). Naming none there, the run is
// refused: nothing the engine can give.
func needed(runDir, repo string, t task, intents []intent.Intention, paths []string, why string) int {
	where := fmt.Sprintf("#%d", t.Issue)
	var known []string
	for _, p := range paths {
		clean := path.Clean(strings.TrimPrefix(p, "./"))
		if clean == "." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
			continue
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(clean))); err == nil && !slices.Contains(known, clean) {
			known = append(known, clean)
		}
	}
	if len(known) == 0 {
		return refusePost(runDir, t.Issue, verdict.Verdict{Status: verdict.Block, Summary: "the files it lacks are none of the repository",
			Findings: []verdict.Finding{{Rule: "needs-unknown", Where: where,
				Message: fmt.Sprintf("its agent said it lacks %s, none a file or folder of the repository: nothing to give it", strings.Join(paths, ", "))}}})
	}
	said := "`" + strings.Join(known, "`, `") + "`"
	if why != "" {
		said += ": " + why
	}
	body := backlog.Said("The tester", "wrote no tests for this issue yet: it lacks code", said,
		"the line gives it these files at its next run; said lacking again, the issue moves to Blocked for a person to name its files")
	intents = append(slices.DeleteFunc(intents, func(i intent.Intention) bool { return i.Kind == "note" }),
		intent.Intention{Kind: "comment", Value: map[string]any{"issue": t.Issue, "sticky": "issue", "body": body}})
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return fail(err)
	}
	return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no test written: it lacks " + strings.Join(known, ", "),
		Findings: []verdict.Finding{{Rule: verdict.NeedsCode, Level: "info", Where: where, Message: strings.Join(known, ", ")}}})
}

// neededFiles are the files the line gives a run again for the code its
// agent said it lacked (the input `needs`), as sources: none when the
// run was not given any.
func neededFiles(runDir string) []string {
	data, err := os.ReadFile(filepath.Join(runDir, "in", "input", "needs"))
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range strings.Split(string(data), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
