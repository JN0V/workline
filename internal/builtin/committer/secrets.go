package committer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/tools"
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
	if _, err := exec.LookPath("gitleaks"); err != nil {
		return append(findings, verdict.Finding{Rule: "secrets-not-checked", Level: "warn",
			Message: "gitleaks is not installed, so neither secrets nor forbidden terms were looked for; to install it: " + tools.Lookup("gitleaks").Install()}), nil
	}
	args := []string{"git"}
	if rng == "" {
		args = append(args, "--staged")
	} else {
		args = append(args, "--log-opts", rng)
	}
	leaks, err := gitleaks(repo, append(args, repo), "")
	if err != nil {
		return nil, err
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

// A Message is a commit message, and the commit it belongs to (empty for
// the one being written).
type Message struct{ Commit, Text string }

// MessageLeaks scans commit messages with gitleaks, with the same lists as
// Secrets: gitleaks reads what commits change, never what they say. All the
// messages go through one scan; each finding names its commit and line.
// Without gitleaks it finds nothing: Secrets already says so.
func MessageLeaks(repo string, messages []Message) ([]verdict.Finding, error) {
	if _, err := exec.LookPath("gitleaks"); err != nil || len(messages) == 0 {
		return nil, nil
	}
	var text strings.Builder
	var starts []int // the first line of each message in text
	lines := 0
	for _, m := range messages {
		starts = append(starts, lines+1)
		body := scannedPart(m.Text)
		text.WriteString(body)
		text.WriteString("\n\n")
		lines += strings.Count(body, "\n") + 2
	}
	leaks, err := gitleaks(repo, []string{"stdin"}, text.String())
	if err != nil {
		return nil, err
	}
	var findings []verdict.Finding
	for _, l := range leaks {
		i := len(starts) - 1
		for i > 0 && starts[i] > l.StartLine {
			i--
		}
		where := fmt.Sprintf("commit message, line %d", l.StartLine-starts[i]+1)
		if c := messages[i].Commit; c != "" {
			where = fmt.Sprintf("commit %s, message line %d", c, l.StartLine-starts[i]+1)
		}
		findings = append(findings, verdict.Finding{Rule: "leak", Where: where,
			Message: fmt.Sprintf("gitleaks rule %q: a secret, or a term that must not reach this repository, in the commit message; the match is not shown", l.RuleID)})
	}
	return findings, nil
}

// scannedPart is what git keeps of a message: the lines before the scissors
// of commit -v, comment lines blanked so that line numbers still hold.
func scannedPart(message string) string {
	lines := strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") && strings.Contains(l, " >8 ") {
			lines = lines[:i]
			break
		}
		if strings.HasPrefix(l, "#") {
			lines[i] = ""
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

type leak struct {
	RuleID    string `json:"RuleID"`
	File      string `json:"File"`
	StartLine int    `json:"StartLine"`
	Commit    string `json:"Commit"`
}

// gitleaks runs one scan in the repository, where gitleaks finds its
// .gitleaks.toml, and reads its report; matches are redacted.
func gitleaks(repo string, args []string, stdin string) ([]leak, error) {
	args = append(args, "--no-banner", "--redact", "--log-level", "error", "--report-format", "json", "--report-path", "-", "--exit-code", "0")
	if common := commonList(repo); common != "" {
		args = append(args, "--config", common)
	}
	cmd := exec.Command("gitleaks", args...)
	cmd.Dir = repo
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gitleaks failed: %v: %s", err, strings.TrimSpace(errOut.String()))
	}
	var leaks []leak
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &leaks); err != nil {
		return nil, fmt.Errorf("gitleaks answered no report: %v", err)
	}
	return leaks, nil
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
	out, _ := exec.Command("git", "-C", repo, "-c", "core.quotePath=off", "diff", "--cached", "--name-only", "--diff-filter=ACMR", "--", termList).Output()
	if strings.TrimSpace(string(out)) == "" {
		return false, false
	}
	if exec.Command("git", "-C", repo, "check-ignore", "-q", "--no-index", "--", termList).Run() == nil {
		return true, true
	}
	mode, _ := exec.Command("git", "-C", repo, "-c", "core.quotePath=off", "ls-files", "-s", "--", termList).Output()
	return true, strings.HasPrefix(string(mode), "120000")
}
