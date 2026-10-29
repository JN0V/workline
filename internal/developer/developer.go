// Package developer is the developer role's built-in steps (ADR-0054): it
// takes a ready issue to a pull request. Step 1, the pick: the first ready
// issue nobody claimed, in the backlog's order, claimed by a label the
// engine puts with a record on the issue. No agent writes code yet.
package developer

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

const (
	exitNothing  = 10 // pre: no question for the agent, its verdict final
	exitExternal = 3  // an outside service failed
	pickFile     = "pick.json"
	// ActMark opens the hidden line of the developer's record on an issue
	// it took, what the auditor reads (ADR-0046).
	ActMark = "workline:developer-act"
)

// Settings are the developer's settings, the project's over the role's.
type Settings struct {
	// ClaimLabel is the label that says the developer took an issue: the
	// forge's assignee where it cannot assign workline's App (GitHub).
	ClaimLabel string
}

// SettingsFrom reads the settings.
func SettingsFrom(m map[string]any) Settings {
	s := Settings{ClaimLabel: "workline:developing"}
	if v, ok := m["claim-label"].(string); ok && strings.TrimSpace(v) != "" {
		s.ClaimLabel = strings.TrimSpace(v)
	}
	return s
}

// Passed is a ready issue the pick passed over, and why.
type Passed struct {
	Issue int    `json:"issue"`
	Why   string `json:"why"`
}

// Act is the record of a pick, hidden in the comment it leaves on the
// issue: the commit the run read, the issue taken, the ready issues before
// it passed over and why.
type Act struct {
	Act    string   `json:"act"`
	At     string   `json:"at,omitempty"`
	Issue  int      `json:"issue"`
	Passed []Passed `json:"passed,omitempty"`
}

var actData = regexp.MustCompile(`<!-- ` + ActMark + ` (\{.*\}) -->`)

// Line is the hidden line a comment keeps the record in.
func (a Act) Line() string {
	data, _ := json.Marshal(a)
	return fmt.Sprintf("<!-- %s %s -->", ActMark, data)
}

// ReadAct reads a pick's record from a comment of the developer's.
func ReadAct(body string) (Act, bool) {
	m := actData.FindStringSubmatch(body)
	var a Act
	if m == nil || json.Unmarshal([]byte(m[1]), &a) != nil || a.Act == "" {
		return Act{}, false
	}
	return a, true
}

// pathWord is a word in backticks: a path, when it holds a slash or a dot.
var pathWord = regexp.MustCompile("`([^`\\s]+)`")

// scopePaths are the paths an issue's Scope names, in backticks.
func scopePaths(body string) []string {
	var out []string
	for _, m := range pathWord.FindAllStringSubmatch(work.ByMeaning(body)[work.Scope].Text, -1) {
		p := strings.TrimSuffix(path.Clean(m[1]), "/")
		if strings.ContainsAny(m[1], "/.") && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// overlap is the first path of a under or over one of b's, "" when none.
func overlap(a, b []string) string {
	for _, p := range a {
		for _, q := range b {
			if p == q || strings.HasPrefix(p, q+"/") || strings.HasPrefix(q, p+"/") {
				return p
			}
		}
	}
	return ""
}

// pick is the issue the developer takes: the first, in the backlog's
// order, offered to whoever builds (ready, waiting on no open issue, no
// parts), assigned to nobody, not claimed already, no open merge request
// closing it, its Scope's paths clear of the issues the developer holds.
// only, when not 0, is the one issue looked at. passed are the issues
// offered it passed over, why each.
func pick(issues []forge.Issue, mrs []forge.MergeRequest, s Settings, only int) (id int, passed []Passed) {
	open := map[int]bool{}
	for _, is := range issues {
		open[is.ID] = true
	}
	closer := map[int]int{}
	for _, m := range mrs {
		for _, n := range forge.Closes(m.Title, m.Body) {
			if _, ok := closer[n]; !ok {
				closer[n] = m.ID
			}
		}
	}
	held := map[int][]string{} // the issues the developer holds: their Scope's paths
	for _, is := range issues {
		if slices.Contains(is.Labels, s.ClaimLabel) {
			held[is.ID] = scopePaths(is.Body)
		}
	}
	backlog.Order(issues)
	for _, is := range issues {
		if only != 0 && is.ID != only || !backlog.Offered(is, open) {
			continue
		}
		why := ""
		switch {
		case len(is.Assignees) > 0:
			why = "assigned to " + strings.Join(is.Assignees, ", ")
		case slices.Contains(is.Labels, s.ClaimLabel):
			why = "taken already, labelled " + s.ClaimLabel
		case closer[is.ID] != 0:
			why = fmt.Sprintf("the open merge request %d closes it", closer[is.ID])
		default:
			mine := scopePaths(is.Body)
			for _, n := range slices.Sorted(maps.Keys(held)) {
				if p := overlap(mine, held[n]); p != "" {
					why = fmt.Sprintf("its %s's `%s` overlaps #%d's, taken", work.Name(work.Scope), p, n)
					break
				}
			}
		}
		if why == "" {
			return is.ID, passed
		}
		passed = append(passed, Passed{Issue: is.ID, Why: why})
	}
	return 0, passed
}

// Pre picks the issue: with no agent, it names it and stops, nothing
// claimed; with one, it leaves the claim to the engine — the label and the
// comment holding the record — and no question: the code is a later
// step's (ADR-0054, build plan).
func Pre(runDir, repo string) int {
	s, err := settingsIn(runDir)
	if err != nil {
		return fail(err)
	}
	only := 0
	if kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":"); kind == "issue" {
		only, _ = strconv.Atoi(num)
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge to read the issues from",
			Findings: []verdict.Finding{{Rule: "no-forge", Message: "the developer takes a ready issue from a forge; " + forge.Missing}}})
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the forge lists no issues",
			Findings: []verdict.Finding{{Rule: "no-backlog", Message: "the developer picks a ready issue from the forge's open issues, which this forge does not list"}}})
	}
	issues, err := b.Issues()
	if err != nil {
		return external(err)
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return external(err)
	}
	id, passed := pick(issues, mrs, s, only)
	var findings []verdict.Finding
	for _, p := range passed {
		findings = append(findings, verdict.Finding{Rule: "issue-passed", Level: "info", Where: fmt.Sprintf("#%d", p.Issue), Message: p.Why})
	}
	if id == 0 {
		summary := "no ready issue waits for the developer"
		if only != 0 {
			summary = fmt.Sprintf("#%d is not one the developer may take", only)
			if len(passed) == 0 {
				findings = append(findings, verdict.Finding{Rule: "issue-not-offered", Level: "info", Where: fmt.Sprintf("#%d", only),
					Message: "not open and " + backlog.LabelReady + ", or it waits on an open issue, or it has parts"})
			}
		}
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: summary, Findings: findings})
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		findings = append(findings, verdict.Finding{Rule: "no-agent", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: "with no agent, the developer names the issue it would take, and claims and opens nothing (principle 5)"})
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("would take #%d: no agent, nothing claimed", id), Findings: findings})
	}
	at, _ := gitOut(repo, "rev-parse", "HEAD")
	act := Act{Act: "pick", At: at, Issue: id, Passed: passed}
	data, _ := json.Marshal(act)
	if err := os.WriteFile(filepath.Join(runDir, "in", pickFile), data, 0o644); err != nil {
		return fail(err)
	}
	claim := []intent.Intention{
		{Kind: "label", Value: map[string]any{"issue": id, "add": []string{s.ClaimLabel}}},
		{Kind: "comment", Value: map[string]any{"issue": id, "sticky": "claim", "body": claimBody(act, s)}},
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), claim); err != nil {
		return fail(err)
	}
	return 0
}

// claimBody is the comment the claim leaves on the issue, its record hidden.
func claimBody(a Act, s Settings) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The developer** takes this issue: the first ready one nobody took, in the backlog's order. "+
		"`%s` says so; a person who takes it back removes the label, or assigns themselves.\n", s.ClaimLabel)
	if len(a.Passed) > 0 {
		b.WriteString("\nPassed over before it:\n\n")
		for _, p := range a.Passed {
			fmt.Fprintf(&b, "- #%d: %s\n", p.Issue, p.Why)
		}
	}
	fmt.Fprintf(&b, "\n%s\n", a.Line())
	return b.String()
}

// Post says what the run took; the claim is the engine's to apply.
func Post(runDir, repo string) int {
	var a Act
	data, err := os.ReadFile(filepath.Join(runDir, "in", pickFile))
	if err != nil {
		return fail(err)
	}
	if err := json.Unmarshal(data, &a); err != nil {
		return fail(err)
	}
	where := fmt.Sprintf("#%d", a.Issue)
	v := verdict.Verdict{Status: verdict.Pass, Summary: "took " + where}
	for _, p := range a.Passed {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "issue-passed", Level: "info", Where: fmt.Sprintf("#%d", p.Issue), Message: p.Why})
	}
	v.Findings = append(v.Findings, verdict.Finding{Rule: "code-not-built", Level: "warn", Where: where,
		Message: "the developer claimed the issue; writing its code and opening its pull request are not built yet (ADR-0054, steps 2 to 4): a person takes it from here, or removes the label"})
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return 0
}

func settingsIn(runDir string) (Settings, error) {
	m := map[string]any{}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return Settings{}, err
	}
	return SettingsFrom(m), nil
}

func final(runDir string, v verdict.Verdict) int {
	v.Final = true
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return exitNothing
}

// external says a forge that did not answer, never read as no issue.
func external(err error) int {
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	return fail(err)
}

func gitOut(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "developer:", err)
	return 1
}
