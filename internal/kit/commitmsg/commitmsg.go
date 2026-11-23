// Package commitmsg holds the committer's message checks: the rules a
// commit message is held to, which other roles use too (the developer
// writes its subjects to them, the reviewer looks for internal codes).
// The committer role (roles/committer/code) runs them on commits.
package commitmsg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/role"
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
	// Rewrite asks the agent to rewrite a refused message at a commit, on
	// by default within AIMaxSeconds; off, the hook refuses in about a
	// second (ADR-0050).
	Rewrite bool `json:"rewrite"`
	// AIMaxSeconds is how long the rewrite may take; the engine holds it.
	AIMaxSeconds int `json:"ai-max-seconds"`

	// The lists InternalCodesAllow is made of, for a refusal to give the
	// line that lets a code through: the project's (the role's, when the
	// project sets none), and what the person's own config adds to it.
	projectAllow, personalAllow []string
}

// SettingsFrom reads the role's settings, as the engine merges them: the
// developer writes its commits' subjects to them (roles/developer/code).
func SettingsFrom(m map[string]any) (Settings, error) {
	var s Settings
	data, err := json.Marshal(m)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("the committer's settings: %w", err)
	}
	return s, nil
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
	if project == nil { // the project's could not be read: the merged list, the person's taken out
		project = slices.DeleteFunc(slices.Clone(s.InternalCodesAllow), func(c string) bool { return slices.Contains(s.personalAllow, c) })
	}
	yours := homeShort(role.PersonalConfig())
	if yours == "" {
		yours = "your own config (workline/config.yaml in your config folder)"
	}
	return fmt.Sprintf("let it through, for you, in %s under roles.committer.settings: `internal-codes-allow: %s`; for everyone, in .workline/config.yaml under roles.committer.settings: `internal-codes-allow: %s`",
		yours, flowList(append(slices.Clone(s.personalAllow), code)), flowList(append(project, code)))
}

// AllowLists finds what InternalCodesAllow is made of, for letThrough: the
// role's list as the project sets it, and the person's own.
func (s *Settings) AllowLists(repo string) {
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

func isWordByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func internalCodes(subject string, s Settings) ([]string, error) {
	cleaned := subject
	for _, a := range s.InternalCodesAllow {
		// an entry is literal text, not a pattern; \b only beside a word
		// character, as it never matches beside punctuation such as `+`.
		pat := regexp.QuoteMeta(a)
		if a != "" && isWordByte(a[0]) {
			pat = `\b` + pat
		}
		if a != "" && isWordByte(a[len(a)-1]) {
			pat += `\b`
		}
		re, err := regexp.Compile(`(?i)` + pat)
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

// KeepsHeader refuses a rewrite that changes the type, the scope or the breaking
// mark the author chose, when the original subject was well formed: only its
// length, body or codes were refused, so what it says it is stays the author's
// call. Adding a scope the author left out only makes it more precise.
func KeepsHeader(original, rewrite string, s Settings) []verdict.Finding {
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
