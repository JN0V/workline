package committer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// termList is the repository's own gitleaks config. A project may commit one
// to share its allowlist; when it is gitignored, or a link, it is a private
// list of terms, and gitleaks lets its own config through.
const termList = ".gitleaks.toml"

// Secrets scans what a commit adds (rng empty: the staged changes) or every
// commit of rng with gitleaks: secrets, and the terms of the user's lists.
// gitleaks takes, first found: GITLEAKS_CONFIG, the repository's
// .gitleaks.toml — which may extend the common list — then the user's common
// list (<config folder>/workline/gitleaks.toml), then its own rules. Matches
// are never printed. Without gitleaks, it says the check did not run.
func Secrets(repo, rng string) ([]verdict.Finding, error) {
	var findings []verdict.Finding
	if rng == "" {
		staged, private := stagedTermList(repo)
		if staged && private {
			findings = append(findings, verdict.Finding{Rule: "term-list-staged", Where: termList,
				Message: "this is a private list of terms (gitignored, or a link): it must never be committed; unstage it with git restore --staged " + termList})
		}
	}
	bin, err := exec.LookPath("gitleaks")
	if err != nil {
		return append(findings, verdict.Finding{Rule: "secrets-not-checked", Level: "warn",
			Message: "gitleaks is not installed, so neither secrets nor forbidden terms were looked for (https://github.com/gitleaks/gitleaks)"}), nil
	}
	args := []string{"git", "--no-banner", "--redact", "--log-level", "error", "--report-format", "json", "--report-path", "-", "--exit-code", "0"}
	if rng == "" {
		args = append(args, "--staged")
	} else {
		args = append(args, "--log-opts", rng)
	}
	if common := commonList(repo); common != "" {
		args = append(args, "--config", common)
	}
	cmd := exec.Command(bin, append(args, repo)...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gitleaks failed: %v: %s", err, strings.TrimSpace(errOut.String()))
	}
	var leaks []struct {
		RuleID    string `json:"RuleID"`
		File      string `json:"File"`
		StartLine int    `json:"StartLine"`
		Commit    string `json:"Commit"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &leaks); err != nil {
		return nil, fmt.Errorf("gitleaks answered no report: %v", err)
	}
	for _, l := range leaks {
		in := ""
		if l.Commit != "" {
			in = " in commit " + l.Commit[:min(7, len(l.Commit))]
		}
		findings = append(findings, verdict.Finding{Rule: "leak", Where: fmt.Sprintf("%s:%d", l.File, l.StartLine),
			Message: fmt.Sprintf("gitleaks rule %q%s: a secret, or a term that must not reach this repository; the match is not shown", l.RuleID, in)})
	}
	return findings, nil
}

// commonList is the user's common list, when gitleaks would not find a
// config of its own: none named by the environment, none in the repository.
func commonList(repo string) string {
	if os.Getenv("GITLEAKS_CONFIG") != "" || os.Getenv("GITLEAKS_CONFIG_TOML") != "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(repo, termList)); err == nil {
		return ""
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, "workline", "gitleaks.toml")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// stagedTermList says whether the commit adds the repository's gitleaks
// config, and whether that config is a private one.
func stagedTermList(repo string) (staged, private bool) {
	out, _ := exec.Command("git", "-C", repo, "diff", "--cached", "--name-only", "--diff-filter=ACMR", "--", termList).Output()
	if strings.TrimSpace(string(out)) == "" {
		return false, false
	}
	if exec.Command("git", "-C", repo, "check-ignore", "-q", "--no-index", "--", termList).Run() == nil {
		return true, true
	}
	mode, _ := exec.Command("git", "-C", repo, "ls-files", "-s", "--", termList).Output()
	return true, strings.HasPrefix(string(mode), "120000")
}
