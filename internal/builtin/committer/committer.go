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
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
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
	InternalCodesMaybe []string `json:"internal-codes-maybe"`
	AllowedIdentities  []string `json:"allowed-identities"`
	ForgeIdentities    []string `json:"forge-identities"`
	// Rewrite asks the agent to rewrite a refused message at a commit;
	// off by default, the hook refusing in about a second (ADR-0050).
	Rewrite bool `json:"rewrite"`
	// AIMaxSeconds is how long the rewrite may take; the engine holds it.
	AIMaxSeconds int `json:"ai-max-seconds"`

	// The lists InternalCodesAllow is made of, for a refusal to give the
	// line that lets a code through: the project's (the role's, when the
	// project sets none), and what the person's own config adds to it.
	projectAllow, personalAllow []string
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

	if second, ok := secondLine(message); ok && second != "" {
		add("blank-after-subject", "the second line must be empty: git reads the whole first paragraph as the subject")
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
		add("internal-code", fmt.Sprintf("%q means nothing outside the project; say what changed, and put the reference in a trailer (Refs: %s). If it names something public, a standard or a record anyone can read, %s", c, c, letThrough(c, s)))
	}
	maybe, err := possibleCodes(subject, s)
	if err != nil {
		return nil, err
	}
	for _, c := range maybe {
		add("possible-internal-code", fmt.Sprintf("%q may be a reference internal to the project (a ticket, a function, a requirement): if so, say what changed and put it in a trailer (Refs: %s). If it is a public name, a chip or a standard, keep it and %s", c, c, letThrough(c, s)))
	}
	return out, nil
}

// letThrough is how a code that is a public name gets past the check: the
// line to write in the person's own config, where a list adds to the
// project's, and in the project's, where a list replaces the role's whole.
func letThrough(code string, s Settings) string {
	project := s.projectAllow
	if project == nil {
		project = s.InternalCodesAllow
	}
	return fmt.Sprintf("let it through, for you, in %s under roles.committer.settings: `internal-codes-allow: %s`; for everyone, in .workline/config.yaml under roles.committer.settings: `internal-codes-allow: %s`",
		homeShort(role.PersonalConfig()), flowList(append(slices.Clone(s.personalAllow), code)), flowList(append(slices.Clone(project), code)))
}

// allowLists finds what InternalCodesAllow is made of, for letThrough: the
// role's list as the project sets it, and the person's own.
func (s *Settings) allowLists(repo string) {
	if r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), "committer"); err == nil {
		if cfg, err := role.LoadProjectConfig(repo); err == nil {
			s.projectAllow = settingList(r.MergedSettings(cfg)["internal-codes-allow"])
		}
	}
	if p, err := role.PersonalSettings("committer"); err == nil {
		s.personalAllow = settingList(p["internal-codes-allow"])
	}
}

// settingList is a setting's list, as strings.
func settingList(v any) []string {
	list, _ := v.([]any)
	out := []string{}
	for _, x := range list {
		out = append(out, fmt.Sprint(x))
	}
	return out
}

// flowList writes a list as YAML on one line, quoting what YAML would not
// read back as the same text.
func flowList(items []string) string {
	plain := regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it
		if !plain.MatchString(it) {
			out[i] = "'" + strings.ReplaceAll(it, "'", "''") + "'"
		}
	}
	return "[" + strings.Join(out, ", ") + "]"
}

// homeShort writes a path under the home folder from ~, as a person types it.
func homeShort(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}

// possibleCodes finds what looks like a code with no dash, F207: often
// internal, as often a public name, nRF52. Which one needs judgement.
func possibleCodes(subject string, s Settings) ([]string, error) {
	codes, err := internalCodes(subject, Settings{InternalCodes: s.InternalCodesMaybe, InternalCodesAllow: s.InternalCodesAllow})
	var out []string
	for _, c := range codes {
		if sure, _ := internalCodes(c, Settings{InternalCodes: s.InternalCodes}); len(sure) == 0 {
			out = append(out, c)
		}
	}
	return out, err
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

// InternalCodes finds the codes internal to the project in a text, by the
// committer's settings: the reviewer looks for them in the comments a change
// adds (roles/reviewer).
func InternalCodes(text string, s Settings) ([]string, error) { return internalCodes(text, s) }

func internalCodes(subject string, s Settings) ([]string, error) {
	cleaned := subject
	for _, a := range s.InternalCodesAllow {
		re, err := regexp.Compile(`(?i)\b` + a + `\b`)
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
	if rng := inputValue(runDir, "range"); rng != "" {
		return checkRange(runDir, repo, rng)
	}
	s, msg, err := load(runDir)
	if err != nil {
		return fail(err)
	}
	s.allowLists(repo)
	findings, err := Check(msg, s)
	if err != nil {
		return fail(err)
	}
	// What the commit carries is checked before its message: no rewrite
	// fixes a secret, so the agent is not asked. Nor is it for one in the
	// message: the message would carry it to the agent.
	secrets, err := scan(repo, "", []Message{{Text: msg}})
	if err != nil {
		return fail(err)
	}
	who, err := Identity(repo, "", s)
	if err != nil {
		return fail(err)
	}
	secrets = append(secrets, who...)
	var advisories, blocking []verdict.Finding
	for _, f := range secrets {
		if f.Level == "warn" {
			advisories = append(advisories, f)
		} else {
			blocking = append(blocking, f)
		}
	}
	if len(blocking) > 0 {
		v := verdict.Verdict{Status: verdict.Block, Summary: "the commit carries what must not reach the repository, or an identity not allowed",
			Findings: append(append(blocking, findings...), advisories...)}
		if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
			return fail(err)
		}
		return 10
	}
	if len(advisories) > 0 { // shown with whatever verdict the run ends on
		if err := writeYAML(filepath.Join(runDir, "in", "advisories.yaml"), advisories); err != nil {
			return fail(err)
		}
	}
	if len(findings) == 0 {
		if len(advisories) > 0 {
			v := verdict.Verdict{Status: verdict.Pass, Summary: "the message is clear", Findings: advisories}
			if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
				return fail(err)
			}
		}
		return 10
	}
	if !s.Rewrite {
		// The agent's rewrite is opt-in (ADR-0050): the person rewrites, the
		// reasons in hand, their message kept.
		v := verdict.Verdict{Status: verdict.Block, Summary: "rewrite the commit message", Findings: append(findings, advisories...)}
		if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
			return fail(err)
		}
		return 10
	}
	if err := writeYAML(filepath.Join(runDir, "in", "findings.yaml"), findings); err != nil {
		return fail(err)
	}
	if said := rewriting(os.Getenv("WORKLINE_AI"), s.AIMaxSeconds); said != "" {
		fmt.Fprintln(os.Stderr, said) // before the wait, not after it
	}
	stat, _ := exec.Command("git", "-C", repo, "-c", "core.quotePath=off", "diff", "--cached", "--stat").Output()
	diff, _ := exec.Command("git", "-C", repo, "-c", "core.quotePath=off", "diff", "--cached", "--unified=2").Output()
	var task strings.Builder
	fmt.Fprintf(&task, "This commit message was refused:\n\n```\n%s\n```\n\nWhy:\n", msg)
	for _, f := range findings {
		fmt.Fprintf(&task, "- %s: %s\n", f.Rule, f.Message)
	}
	fmt.Fprintf(&task, "\nStaged changes:\n\n```\n%s```\n\n```diff\n%s```\n", stat, capText(string(diff), maxDiff))
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task.String()), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// Post is the role's judge step: a rewritten message must pass the same
// checks, and carry no secret or term; with no rewrite, the original
// findings stand.
func Post(runDir, repo string) int {
	s, original, err := load(runDir)
	if err != nil {
		return fail(err)
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	messages := 0
	for _, in := range intents {
		if in.Kind == "commit-message" {
			messages++
		}
	}
	if messages > 1 {
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "one commit, one message",
			Findings: []verdict.Finding{{Rule: "several-messages", Where: "proposal",
				Message: fmt.Sprintf("%d messages were proposed for one commit; to split it, return a note saying how", messages)}}})
	}
	for _, in := range intents {
		switch in.Kind {
		case "commit-message":
			msg, _ := in.Value.(string)
			checked, err := Check(msg, s)
			if err != nil {
				return fail(err)
			}
			var findings []verdict.Finding
			for _, f := range checked {
				if f.Rule != "possible-internal-code" { // the agent judged it: kept, a public name
					findings = append(findings, f)
				}
			}
			findings = append(findings, keepsHeader(original, msg, s)...)
			said, err := MessageLeaks(repo, []Message{{Text: msg}})
			if err != nil {
				return fail(err)
			}
			findings = append(findings, said...)
			kept := map[string]bool{}
			for _, t := range Trailers(msg) {
				kept[t] = true
			}
			for _, t := range Trailers(original) {
				if !kept[t] {
					findings = append(findings, verdict.Finding{Rule: "trailer-dropped", Where: "trailers",
						Message: fmt.Sprintf("the rewrite dropped %q, or did not keep it on its own line in the last paragraph; trailers are kept as written", t)})
				}
			}
			// workline's own trailers say who made or rewrote a commit:
			// the engine writes them, never the agent (ADR-0046).
			had := map[string]bool{}
			for _, l := range ownLines(original) {
				had[l] = true
			}
			for _, l := range ownLines(msg) {
				if !had[l] {
					findings = append(findings, verdict.Finding{Rule: "trailer-added", Where: "trailers",
						Message: fmt.Sprintf("the rewrite added %q; workline's own trailers are the engine's to write", l)})
				}
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
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "advisories.yaml")); err == nil {
		var advisories []verdict.Finding
		if err := yaml.Unmarshal(data, &advisories); err != nil {
			return fail(err)
		}
		v.Findings = append(v.Findings, advisories...)
	}
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

// maxDiff keeps the diff within the role's context budget; the stat above it
// still lists every file.
const maxDiff = 8000

func capText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (diff cut here; the stat above lists every file)\n"
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

// ownLine is a line git could read as one of workline's own trailers,
// whatever its letter case or spaces, anywhere in a message.
var ownLine = regexp.MustCompile(`(?i)^\s*workline-[a-z0-9-]*\s*:`)

// ownLines are a message's lines that could be read as workline's own
// trailers, their key in lower case, no space about its colon.
func ownLines(message string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n") {
		if ownLine.MatchString(l) {
			key, value, _ := strings.Cut(l, ":")
			out = append(out, strings.ToLower(strings.TrimSpace(key))+": "+strings.TrimSpace(value))
		}
	}
	return out
}

// Trailers returns the trailer lines of a message (Refs:, Co-Authored-By:...).
func Trailers(message string) []string {
	var lines []string
	for _, l := range strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(l, "#") {
			lines = append(lines, strings.TrimRight(l, " \t"))
		}
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	trailer := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*: \S`)
	var out []string
	for i := len(lines) - 1; i > 0 && trailer.MatchString(lines[i]); i-- {
		out = append([]string{lines[i]}, out...)
	}
	if len(out) > 0 && len(lines) > len(out) && lines[len(lines)-len(out)-1] != "" {
		return nil // not a separate last paragraph: not trailers
	}
	return out
}

// checkRange checks every commit of a merge request (base..head). Commits
// already pushed are not rewritten: the findings go to the people who wrote
// them, so no agent is asked.
// keepsHeader refuses a rewrite that changes the type, the scope or the breaking
// mark the author chose, when the original subject was well formed: only its
// length, body or codes were refused, so what it says it is stays the author's
// call. Adding a scope the author left out only makes it more precise.
func keepsHeader(original, rewrite string, s Settings) []verdict.Finding {
	format := regexp.MustCompile(`^(` + strings.Join(quoteAll(s.Types), "|") + `)(\([^()\s]+\))?(!?): \S`)
	subject, _ := split(original)
	was := format.FindStringSubmatch(subject)
	if was == nil {
		return nil // the header itself was wrong: the rewrite may change it
	}
	newSubject, _ := split(rewrite)
	now := format.FindStringSubmatch(newSubject)
	if now == nil || now[1] == was[1] && (now[2] == was[2] || was[2] == "") && now[3] == was[3] {
		return nil // a malformed rewrite is already refused by Check
	}
	return []verdict.Finding{{Rule: "header-changed", Where: "subject",
		Message: fmt.Sprintf("the rewrite changed %q into %q; keep the type and scope the author chose, and fix only what was refused", was[1]+was[2]+was[3], now[1]+now[2]+now[3])}}
}

func checkRange(runDir, repo, rng string) int {
	var s Settings
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return fail(err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return fail(err)
	}
	s.allowLists(repo)
	out, err := exec.Command("git", append([]string{"-C", repo, "log", "--format=%h%x1f%B%x1e"}, gitrange.Args(rng)...)...).Output()
	if err != nil {
		return fail(fmt.Errorf("cannot read the commits of %s: %v", rng, err))
	}
	var findings, advisories []verdict.Finding
	var messages []Message
	for _, rec := range strings.Split(string(out), "\x1e") {
		hash, msg, ok := strings.Cut(strings.TrimSpace(rec), "\x1f")
		if !ok {
			continue
		}
		messages = append(messages, Message{Commit: hash, Text: msg})
		f, err := Check(msg, s)
		if err != nil {
			return fail(err)
		}
		for _, x := range f {
			x.Where = "commit " + hash + ", " + x.Where
			if x.Rule == "possible-internal-code" {
				x.Level = "warn" // judged when it was committed, by an agent or a person
				advisories = append(advisories, x)
				continue
			}
			findings = append(findings, x)
		}
	}
	secrets, err := scan(repo, rng, messages)
	if err != nil {
		return fail(err)
	}
	who, err := Identity(repo, rng, s)
	if err != nil {
		return fail(err)
	}
	secrets = append(secrets, who...)
	v := verdict.Verdict{Status: verdict.Pass, Summary: "every commit message is clear"}
	if len(findings) > 0 {
		v = verdict.Verdict{Status: verdict.Block, Summary: "some commit messages need rewriting (git rebase -i)", Findings: findings}
	}
	v.Findings = append(v.Findings, advisories...)
	for _, f := range secrets {
		v.Findings = append(v.Findings, f)
		if f.Level != "warn" {
			v.Status, v.Summary = verdict.Block, "a commit carries what must not reach the repository, or an identity not allowed: rewrite the history before it is pushed"
		}
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return 10
}

// rewriting is what the hook says before the agent rewrites a refused
// message, while the person waits: which agent, and how long at most.
// Nothing when no agent is asked.
func rewriting(ai string, seconds int) string {
	if ai == "" || ai == "none" {
		return ""
	}
	name, _, _ := strings.Cut(ai, ":")
	said := fmt.Sprintf("workline committer: the message is refused; the agent (%s) rewrites it", name)
	if seconds > 0 {
		said += fmt.Sprintf(", %d s at most", seconds)
	}
	return said + "…"
}

func inputValue(runDir, name string) string {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
	return strings.TrimSpace(string(data))
}

// secondLine returns the line after the subject, ignoring comments; ok is
// false for a one-line message.
func secondLine(message string) (string, bool) {
	var lines []string
	for _, l := range strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(l, "#") {
			lines = append(lines, strings.TrimRight(l, " \t"))
		}
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) < 2 {
		return "", false
	}
	return lines[1], true
}
