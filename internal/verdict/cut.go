package verdict

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// What a role's pre cut of what it gives its agent, past one of its caps
// (#1159): said to the agent in the task, and to the person in the run's
// verdict, never to the agent alone. Pre writes each cut, one a line, in
// its run's in/cuts; the engine says each as a finding of the run.

// CutRule is the finding a cut is said by: a warning, never a block.
const CutRule = "input-cut"

// cutsFile is where pre writes its cuts, under the run's in/.
const cutsFile = "cuts"

// SayCut records a cut of what the run gives its agent: what was cut,
// from what size to what cap, and the setting that raises it. Said once,
// however many times pre writes it.
func SayCut(runDir, said string) error {
	said = strings.Join(strings.Fields(said), " ")
	if said == "" || slices.Contains(Cuts(runDir), said) {
		return nil
	}
	f, err := os.OpenFile(filepath.Join(runDir, "in", cutsFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(said + "\n"); err != nil {
		_ = f.Close() // the write's error is the one said
		return err
	}
	return f.Close()
}

// Cuts are the cuts a run's pre said, in order.
func Cuts(runDir string) []string {
	data, err := os.ReadFile(filepath.Join(runDir, "in", cutsFile))
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// CutFindings are a run's cuts as findings, one each, those already in
// had left out.
func CutFindings(runDir string, had []Finding) []Finding {
	var out []Finding
	for _, c := range Cuts(runDir) {
		if !slices.ContainsFunc(had, func(f Finding) bool { return f.Rule == CutRule && f.Message == c }) {
			out = append(out, Finding{Rule: CutRule, Level: "warn", Message: c})
		}
	}
	return out
}

// Setting names a role's setting as a project raises it.
func Setting(role, name string) string {
	return "roles." + role + ".settings." + name + " in .workline/config.yaml"
}
