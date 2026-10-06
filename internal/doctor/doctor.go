// Package doctor says what workline needs on this machine and in a
// repository, what is missing, and the command that sets each one up, as
// `flutter doctor` and `brew doctor` do. It changes nothing.
package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/builtin/documentalist"
	"github.com/JN0V/workline/internal/hooks"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/tools"
)

// The levels of a check, from fine to broken.
const (
	OK    = "ok"    // set up
	Note  = "note"  // a choice, said so it is not mistaken for a pass
	Warn  = "warn"  // something does not run; workline still works
	Error = "error" // workline cannot do what it is set up to do
)

// Check is one thing looked at.
type Check struct {
	Area    string `json:"area"` // machine | repository
	Rule    string `json:"rule"`
	Where   string `json:"where,omitempty"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"` // the command that sets it up, or where to read how
}

// Report is what the doctor found. Findings are the checks that are not ok.
type Report struct {
	Status   string  `json:"status"` // pass, or block when a check is an error
	Summary  string  `json:"summary"`
	Checks   []Check `json:"checks"`
	Findings []Check `json:"findings"`
}

// Options say where to look.
type Options struct {
	Repo     string // a repository's root; empty: the machine only
	RolesDir string
	AI       string // WORKLINE_AI, which the hooks use first
	UserAI   string // `ai:` in the user's config
	// NoPushApproval: the person has not asked to approve each push (ADR-0011).
	NoPushApproval bool
}

// Run looks at the machine, then at the repository when there is one.
func Run(o Options) *Report {
	r := &Report{Checks: []Check{}, Findings: []Check{}}
	machine(r, o)
	if o.Repo != "" {
		repository(r, o)
	}
	errors, warns := 0, 0
	for _, c := range r.Checks {
		if c.Level == OK {
			continue
		}
		r.Findings = append(r.Findings, c)
		switch c.Level {
		case Error:
			errors++
		case Warn:
			warns++
		}
	}
	r.Status = "pass"
	switch {
	case errors > 0:
		r.Status = "block"
		r.Summary = fmt.Sprintf("%d problems, %d warnings", errors, warns)
	case warns > 0:
		r.Summary = fmt.Sprintf("%d warnings: some checks do not run", warns)
	default:
		r.Summary = "everything workline uses is set up"
	}
	return r
}

func (r *Report) add(c Check) { r.Checks = append(r.Checks, c) }

func machine(r *Report, o Options) {
	if v, err := exec.Command("git", "--version").Output(); err != nil {
		r.add(Check{Area: "machine", Rule: "git-missing", Level: Error, Message: "git is not on your PATH: workline needs it for everything",
			Fix: "see https://git-scm.com/downloads"})
	} else {
		r.add(Check{Area: "machine", Rule: "git", Level: OK, Message: strings.TrimSpace(string(v))})
	}

	if p, err := hooks.DefaultPaths(); err == nil {
		cur, _ := exec.Command("git", "config", "--global", "--get", "core.hooksPath").Output()
		switch got := strings.TrimSpace(string(cur)); {
		case got == "":
			r.add(Check{Area: "machine", Rule: "hooks-not-installed", Level: Warn,
				Message: "the global git hooks are not installed: commits are not checked as they are written",
				Fix:     "workline hooks install --global"})
		case filepath.Clean(got) != filepath.Clean(p.Hooks):
			r.add(Check{Area: "machine", Rule: "hooks-not-installed", Level: Warn,
				Message: fmt.Sprintf("the global core.hooksPath is %s, not workline's: commits are not checked as they are written", got),
				Fix:     "workline hooks install --global   # hands over to " + got})
		default:
			if _, err := os.Stat(filepath.Join(p.Hooks, "commit-msg")); err != nil {
				r.add(Check{Area: "machine", Rule: "hooks-broken", Level: Error,
					Message: "core.hooksPath points to " + p.Hooks + ", which holds no hook: git runs no hook at all",
					Fix:     "workline hooks install --global"})
			} else {
				r.add(Check{Area: "machine", Rule: "hooks", Level: OK, Message: "global git hooks installed in " + p.Hooks})
			}
		}
	}

	if o.NoPushApproval {
		r.add(Check{Area: "machine", Rule: "push-approval-off", Level: Note,
			Message: "pushes leave without a person approving them: the review is on the pull request (ADR-0011); `approve-push: true` in your config asks at each push"})
	} else {
		r.add(Check{Area: "machine", Rule: "push-approval", Level: OK,
			Message: "a person approves every push: on the terminal, else in the editor window, else in a dialog"})
	}

	agent(r, "machine", o.AI, "WORKLINE_AI", o.UserAI, "your config")

	for _, name := range UsedTools(o.RolesDir) {
		t := tools.Lookup(name)
		if t.Present() {
			r.add(Check{Area: "machine", Rule: "tool", Where: name, Level: OK, Message: name + " — " + orName(t.For)})
			continue
		}
		// A tool for what usually runs on a forge, gardening, is not missed here.
		level := Warn
		if !t.Local && t.For != "" {
			level = Note
		}
		r.add(Check{Area: "machine", Rule: "tool-missing", Where: name, Level: level,
			Message: name + " is not installed, so this does not run here: " + orName(t.For), Fix: t.Install()})
	}
}

// agent says which agent is used, and whether it can be called. The first of
// the two given that is set wins, as the hooks read them.
func agent(r *Report, area, first, firstFrom, second, secondFrom string) {
	spec, from := first, firstFrom
	if spec == "" {
		spec, from = second, secondFrom
	}
	if spec == "" || spec == "none" {
		r.add(Check{Area: area, Rule: "agent-none", Level: Note,
			Message: "no agent: every check runs, and what needs judgement goes to a person (a refused message is explained, not rewritten)",
			Fix:     "workline setup"})
		return
	}
	name, _, _ := strings.Cut(spec, ":")
	switch name {
	case "claude":
		t := tools.Lookup("claude")
		if !t.Present() {
			r.add(Check{Area: area, Rule: "agent-missing", Level: Error,
				Message: fmt.Sprintf("the agent is %s (%s), but `claude` is not on your PATH: every call would fail", spec, from), Fix: t.Install()})
			return
		}
		r.add(Check{Area: area, Rule: "agent", Level: OK, Message: fmt.Sprintf("agent %s (%s); run `claude` once to log in, if you have not", spec, from)})
	case "cmd":
		r.add(Check{Area: area, Rule: "agent", Level: OK, Message: fmt.Sprintf("agent %s (%s): a command, not checked here", spec, from)})
	default:
		r.add(Check{Area: area, Rule: "agent-unknown", Level: Error,
			Message: fmt.Sprintf("the agent %q (%s) is not one workline knows: none, claude, claude:<model>@<effort>, cmd:<command>", spec, from)})
	}
}

func repository(r *Report, o Options) {
	where := o.Repo
	if _, err := os.Stat(filepath.Join(o.Repo, ".workline", "off")); err == nil {
		r.add(Check{Area: "repository", Rule: "repository-off", Where: where, Level: Note,
			Message: ".workline/off is here: the global hooks skip this repository", Fix: "rm .workline/off"})
		return
	}
	cfg, err := role.LoadProjectConfig(o.Repo)
	var line *routing.Config
	if err == nil {
		line, err = routing.Load(o.Repo)
	}
	if err != nil {
		r.add(Check{Area: "repository", Rule: "config-invalid", Where: where, Level: Error, Message: err.Error()})
		return
	}
	if cfg.AI != "" {
		agent(r, "repository", o.AI, "WORKLINE_AI", cfg.AI, ".workline/config.yaml")
	}
	if cfg.Forge == "github" { // GitLab is reached through its API, with nothing installed
		if t := tools.Lookup("gh"); !t.Present() {
			r.add(Check{Area: "repository", Rule: "tool-missing", Where: "gh", Level: Warn,
				Message: "gh is not installed, so this does not run here: " + t.For, Fix: t.Install()})
		}
	}

	// Where the docs are judged (ADR-0010): on each merge request in CI, by
	// gardening, at the release. The push only counts them, when routed.
	routes := func(event string) bool {
		for _, s := range line.Events[event] {
			if s == "documentalist" {
				return true
			}
		}
		return false
	}
	var judged []string
	if routes("merge-request") && ciRuns(o.Repo, "workline route merge-request") {
		judged = append(judged, "on each merge request, in CI")
	}
	if routes("schedule") && ciRuns(o.Repo, "workline route schedule") {
		judged = append(judged, "by gardening, on a schedule")
	}
	if routes("release") {
		if len(judged) == 0 {
			judged = append(judged, "at the release only: `workline docs` judges them before")
		} else {
			judged = append(judged, "at the release")
		}
	}
	if len(judged) > 0 {
		level := OK
		if len(judged) == 1 && routes("release") {
			level = Note // no CI judges them: fine for a solo project, worth knowing
		}
		r.add(Check{Area: "repository", Rule: "docs-judged", Where: where, Level: level,
			Message: "docs are judged " + strings.Join(judged, "; ")})
	} else {
		r.add(Check{Area: "repository", Rule: "docs-judged-nowhere", Where: where, Level: Warn,
			Message: "the documentalist runs on no merge request in CI, no gardening, and not at the release: docs go stale unseen, unless someone runs `workline docs`",
			Fix:     "copy the CI templates (ci/github, ci/gitlab), or route `release` to the documentalist"})
	}
	if routes("pre-push") {
		r.add(Check{Area: "repository", Rule: "docs-counted-at-push", Where: where, Level: OK,
			Message: "the push counts the docs its commits make suspect"})
	}
	// A shallow clone lacks the commits docs were checked against, and the
	// last release: the documentalist and the release say so and stop.
	if out, err := exec.Command("git", "-C", o.Repo, "rev-parse", "--is-shallow-repository").Output(); err == nil && strings.TrimSpace(string(out)) == "true" {
		r.add(Check{Area: "repository", Rule: "shallow-clone", Where: where, Level: Warn,
			Message: "this clone is shallow: a doc checked against a commit it lacks cannot be judged, and the release cannot find the last one",
			Fix:     "git fetch --unshallow   # in CI: actions/checkout's `fetch-depth: 0`, GitLab's `GIT_DEPTH: 0`"})
	}

	d, err := role.Load(o.RolesDir, "documentalist")
	if err != nil {
		r.add(Check{Area: "repository", Rule: "roles-unreadable", Where: o.RolesDir, Level: Error, Message: err.Error()})
		return
	}
	globs := stringList(d.MergedSettings(cfg)["docs"])
	tracked, none, untracked, err := documentalist.Coverage(o.Repo, globs)
	if err != nil {
		r.add(Check{Area: "repository", Rule: "docs-unreadable", Where: where, Level: Error, Message: err.Error()})
		return
	}
	total := len(tracked) + len(none) + len(untracked)
	described := fmt.Sprintf("%d docs declare their sources", len(tracked))
	if len(none) > 0 {
		described += fmt.Sprintf(", %d describe no code (`sources: []`)", len(none))
	}
	switch {
	case total == 0:
		r.add(Check{Area: "repository", Rule: "no-docs", Where: where, Level: Note,
			Message: fmt.Sprintf("no doc under %v: the documentalist has nothing to look after", globs)})
	case len(untracked) == 0:
		r.add(Check{Area: "repository", Rule: "docs-tracked", Where: where, Level: OK, Message: described})
	case len(tracked) == 0:
		r.add(Check{Area: "repository", Rule: "nothing-tracked", Where: where, Level: Warn,
			Message: fmt.Sprintf("none of the %d docs declares its sources, so none can be found suspect: the documentalist only checks sizes and links", total),
			Fix:     "workline init   # proposes their sources, with an agent; without one, lists them"})
	default:
		r.add(Check{Area: "repository", Rule: "docs-untracked", Where: where, Level: Warn,
			Message: fmt.Sprintf("%s; %d say nothing, so they are never found suspect: %s", described, len(untracked), sample(untracked)),
			Fix:     "workline init   # proposes their sources, with an agent; without one, lists them"})
	}
	if msg, err := documentalist.UnreadDocs(o.Repo, globs); err != nil {
		r.add(Check{Area: "repository", Rule: "docs-unreadable", Where: where, Level: Error, Message: err.Error()})
	} else if msg != "" {
		r.add(Check{Area: "repository", Rule: "docs-not-read", Where: where, Level: Warn, Message: msg})
	}
}

// UsedTools lists the tools the roles use, each once, sorted.
func UsedTools(rolesDir string) []string {
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, e := range entries {
		ro, err := role.Load(rolesDir, e.Name())
		if err != nil {
			continue
		}
		for _, u := range ro.Uses {
			seen[u] = true
		}
	}
	out := make([]string, 0, len(seen))
	for u := range seen {
		out = append(out, u)
	}
	sort.Strings(out)
	return out
}

func stringList(v any) []string {
	var out []string
	if l, ok := v.([]any); ok {
		for _, x := range l {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func sample(paths []string) string {
	if len(paths) <= 5 {
		return strings.Join(paths, ", ")
	}
	return strings.Join(paths[:5], ", ") + fmt.Sprintf(", and %d more", len(paths)-5)
}

// ciRuns says whether a CI configuration of the repository runs the command:
// a GitHub workflow, or GitLab's pipeline.
func ciRuns(repo, command string) bool {
	files, _ := filepath.Glob(filepath.Join(repo, ".github", "workflows", "*.y*ml"))
	files = append(files, filepath.Join(repo, ".gitlab-ci.yml"))
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil && strings.Contains(string(data), command) {
			return true
		}
	}
	return false
}

func orName(s string) string {
	if s == "" {
		return "a role uses it"
	}
	return s
}
