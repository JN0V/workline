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
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/commitmsg"
	"github.com/JN0V/workline/internal/kit/verdict"
	"go.yaml.in/yaml/v3"
)

// The role joins the engine by name (internal/contract).
func init() { contract.Join("committer", contract.Role{Pre: Pre, Post: Post}) }

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
	s.AllowLists(repo)
	findings, err := commitmsg.Check(msg, s)
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
		// The agent's rewrite turned off (ADR-0050): the person rewrites,
		// the reasons in hand, their message kept.
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
	if len(diff) > maxDiff {
		// Said to the person too, not to the agent alone (#1159).
		if err := verdict.SayCut(runDir, fmt.Sprintf("the staged diff given to the committer's agent is cut from %d to %d characters, a cap no setting changes: the stat above it lists every file", len(diff), maxDiff)); err != nil {
			return fail(err)
		}
	}
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
			checked, err := commitmsg.Check(msg, s)
			if err != nil {
				return fail(err)
			}
			var findings []verdict.Finding
			for _, f := range checked {
				if f.Rule != "possible-internal-code" { // the agent judged it: kept, a public name
					findings = append(findings, f)
				}
			}
			findings = append(findings, commitmsg.KeepsHeader(original, msg, s)...)
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

func load(runDir string) (commitmsg.Settings, string, error) {
	var s commitmsg.Settings
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
func checkRange(runDir, repo, rng string) int {
	var s commitmsg.Settings
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return fail(err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return fail(err)
	}
	s.AllowLists(repo)
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
		f, err := commitmsg.Check(msg, s)
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
	title := squashed(repo, rng, s)
	v := verdict.Verdict{Status: verdict.Pass, Summary: "every commit message is clear"}
	switch blocking := slices.ContainsFunc(title, func(f verdict.Finding) bool { return f.Level != "warn" }); {
	case len(findings) > 0 && blocking:
		v = verdict.Verdict{Status: verdict.Block, Summary: "some commit messages need rewriting (git rebase -i), and the title the squash lands: " + titleFix}
	case len(findings) > 0:
		v = verdict.Verdict{Status: verdict.Block, Summary: "some commit messages need rewriting (git rebase -i)"}
	case blocking:
		v = verdict.Verdict{Status: verdict.Block, Summary: "the squash would land a subject the rules refuse: " + titleFix}
	}
	v.Findings = append(findings, title...)
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

// titleFix is how a refused title is fixed: an edit to the title runs no
// check, so the job is run again by hand, or by a push (ADR-0051).
const titleFix = "edit the merge request's title, then re-run this job (one click) or push"

// squashed checks, by the same rules, the subject a squash would land on
// the base (ADR-0051): the merge request's title, or its one commit's
// subject where the forge takes that, room left for what the forge adds
// after it, GitHub's " (#N)". Only on a merge request a forge may squash;
// what could not be read is said, never read as a pass.
func squashed(repo, rng string, s commitmsg.Settings) []verdict.Finding {
	kind, id, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	n, err := strconv.Atoi(id)
	if kind != "merge-request" || err != nil {
		return nil
	}
	unread := func(why string) []verdict.Finding {
		return []verdict.Finding{{Rule: "title-unread", Level: "warn", Where: "title",
			Message: "the subject a squash would land is not checked: " + why}}
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return unread("the forge could not be opened (" + err.Error() + ")")
	}
	if f == nil {
		return nil // no forge: no merge request to read
	}
	sq, ok := f.(forge.Squasher)
	if !ok {
		return unread("this forge does not say whether it squashes")
	}
	how, err := sq.Squash(n)
	if err != nil {
		return unread("whether the forge squashes could not be read (" + err.Error() + ")")
	}
	if !how.On {
		return nil
	}
	mr, err := f.MergeRequest(n)
	if err != nil {
		return unread("the merge request could not be read (" + err.Error() + ")")
	}
	subject, where := mr.Title, "title"
	if how.LoneCommit {
		out, err := exec.Command("git", append([]string{"-C", repo, "log", "--format=%s"}, gitrange.Args(rng)...)...).Output()
		if lines := strings.Split(strings.TrimSpace(string(out)), "\n"); err == nil && len(lines) == 1 && lines[0] != "" {
			subject, where = lines[0], "title (the squash takes the one commit's subject)"
		}
	}
	room := s
	if s.SubjectMax > 0 {
		room.SubjectMax = s.SubjectMax - utf8.RuneCountInString(how.Suffix)
	}
	found, err := commitmsg.Check(subject, room)
	if err != nil {
		return unread(err.Error())
	}
	for i := range found {
		found[i].Where = where
		switch found[i].Rule {
		case "subject-length":
			found[i].Message = fmt.Sprintf("the squash lands %q: %d characters, over the limit of %d", subject+how.Suffix, utf8.RuneCountInString(subject+how.Suffix), s.SubjectMax)
		case "possible-internal-code":
			found[i].Level = "warn" // as on commits already made: said
		}
	}
	return found
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
