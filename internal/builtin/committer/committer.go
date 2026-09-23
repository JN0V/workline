// Package committer holds the deterministic checks of the committer role
// (roles/committer). The role's pre and post scripts call them through
// `workline builtin committer pre|post`.
package committer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Settings are the role's settings, as merged by the engine.
type Settings struct {
	SubjectMax         int      `json:"subject-max"`
	BodyMaxLines       int      `json:"body-max-lines"`
	Types              []string `json:"types"`
	InternalCodes      []string `json:"internal-codes"`
	InternalCodesAllow []string `json:"internal-codes-allow"`
}

// Check returns what is wrong with a commit message. An empty result means it passes.
func Check(message string, s Settings) ([]verdict.Finding, error) {
	subject, body := split(message)
	if skipped(subject) {
		return nil, nil
	}
	var out []verdict.Finding
	add := func(rule, msg string) {
		out = append(out, verdict.Finding{Rule: rule, Where: "subject", Message: msg})
	}

	format := regexp.MustCompile(`^(` + strings.Join(quoteAll(s.Types), "|") + `)(\([^()\s]+\))?!?: \S`)
	if !format.MatchString(subject) {
		add("format", fmt.Sprintf("the subject must read `type(scope): summary`, with a type among %s", strings.Join(s.Types, ", ")))
	}
	if n := utf8.RuneCountInString(subject); s.SubjectMax > 0 && n > s.SubjectMax {
		add("subject-length", fmt.Sprintf("the subject is %d characters; the limit is %d", n, s.SubjectMax))
	}
	if n := len(body); s.BodyMaxLines > 0 && n > s.BodyMaxLines {
		out = append(out, verdict.Finding{Rule: "body-length", Where: "body",
			Message: fmt.Sprintf("the body is %d lines; the limit is %d — explanations belong in the docs or the changelog", n, s.BodyMaxLines)})
	}
	codes, err := internalCodes(subject, s)
	if err != nil {
		return nil, err
	}
	for _, c := range codes {
		add("internal-code", fmt.Sprintf("%q means nothing outside the project; say what changed, and put the reference in a trailer (Refs: %s)", c, c))
	}
	return out, nil
}

// split returns the subject and the body lines that count against the limit:
// comments, blank lines and trailers (Refs:, Co-Authored-By:...) do not.
func split(message string) (string, []string) {
	var lines []string
	for _, l := range strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(l, "#") {
			lines = append(lines, strings.TrimRight(l, " \t"))
		}
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	if len(lines) == 0 {
		return "", nil
	}
	subject, rest := lines[0], lines[1:]
	trailer := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*: \S`)
	var paragraphs [][]string
	var cur []string
	for _, l := range rest {
		if l == "" {
			if len(cur) > 0 {
				paragraphs = append(paragraphs, cur)
				cur = nil
			}
			continue
		}
		cur = append(cur, l)
	}
	if len(cur) > 0 {
		paragraphs = append(paragraphs, cur)
	}
	if n := len(paragraphs); n > 0 && all(paragraphs[n-1], trailer.MatchString) {
		paragraphs = paragraphs[:n-1]
	}
	var body []string
	for _, p := range paragraphs {
		body = append(body, p...)
	}
	return subject, body
}

// skipped are messages git writes itself.
func skipped(subject string) bool {
	for _, p := range []string{"Merge ", "Revert \"", "fixup! ", "squash! ", "amend! "} {
		if strings.HasPrefix(subject, p) {
			return true
		}
	}
	return false
}

func internalCodes(subject string, s Settings) ([]string, error) {
	cleaned := subject
	for _, a := range s.InternalCodesAllow {
		re, err := regexp.Compile(`\b` + a + `\b`)
		if err != nil {
			return nil, fmt.Errorf("internal-codes-allow %q: %w", a, err)
		}
		cleaned = re.ReplaceAllString(cleaned, " ")
	}
	var found []string
	for _, c := range s.InternalCodes {
		re, err := regexp.Compile(c)
		if err != nil {
			return nil, fmt.Errorf("internal-codes %q: %w", c, err)
		}
		found = append(found, re.FindAllString(cleaned, -1)...)
	}
	return found, nil
}

// Pre is the role's prepare step: check the message; if it fails, ask the
// agent to rewrite it. Exit 10 when there is nothing to do.
func Pre(runDir, repo string) int {
	s, msg, err := load(runDir)
	if err != nil {
		return fail(err)
	}
	findings, err := Check(msg, s)
	if err != nil {
		return fail(err)
	}
	if len(findings) == 0 {
		return 10
	}
	if err := writeYAML(filepath.Join(runDir, "in", "findings.yaml"), findings); err != nil {
		return fail(err)
	}
	diff, _ := exec.Command("git", "-C", repo, "diff", "--cached", "--stat").Output()
	var task strings.Builder
	fmt.Fprintf(&task, "This commit message was refused:\n\n```\n%s\n```\n\nWhy:\n", msg)
	for _, f := range findings {
		fmt.Fprintf(&task, "- %s: %s\n", f.Rule, f.Message)
	}
	fmt.Fprintf(&task, "\nStaged changes:\n\n```\n%s```\n", diff)
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task.String()), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// Post is the role's judge step: a rewritten message must pass the same
// checks; with no rewrite, the original findings stand.
func Post(runDir string) int {
	s, _, err := load(runDir)
	if err != nil {
		return fail(err)
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	for _, in := range intents {
		switch in.Kind {
		case "commit-message":
			msg, _ := in.Value.(string)
			findings, err := Check(msg, s)
			if err != nil {
				return fail(err)
			}
			if len(findings) > 0 {
				return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the rewritten message still fails", Findings: findings})
			}
			return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "message rewritten"})
		case "note":
			return write(runDir, verdict.Verdict{Status: verdict.Human, Summary: fmt.Sprint(in.Value)})
		}
	}
	var findings []verdict.Finding
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "findings.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &findings); err != nil {
			return fail(err)
		}
	}
	return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "rewrite the commit message", Findings: findings})
}

func load(runDir string) (Settings, string, error) {
	var s Settings
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return s, "", err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, "", fmt.Errorf("settings: %w", err)
	}
	msg, err := os.ReadFile(filepath.Join(runDir, "in", "input", "message"))
	if err != nil {
		return s, "", fmt.Errorf("no commit message given: %w", err)
	}
	return s, string(msg), nil
}

func write(runDir string, v verdict.Verdict) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	switch v.Status {
	case verdict.Pass:
		return 0
	case verdict.Human:
		return 2
	}
	return 1
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "committer:", err)
	return 99
}

func quoteAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = regexp.QuoteMeta(s)
	}
	return out
}

func all(in []string, f func(string) bool) bool {
	for _, s := range in {
		if !f(s) {
			return false
		}
	}
	return true
}
