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

	agent(r, "machine", o.AI, "WORKLINE_AI", o.UserAI, "your config")

	for _, name := range usedTools(o.RolesDir) {
		t := tools.Lookup(name)
		if t.Present() {
			r.add(Check{Area: "machine", Rule: "tool", Where: name, Level: OK, Message: name + " — " + orName(t.For)})
			continue
		}
		r.add(Check{Area: "machine", Rule: "tool-missing", Where: name, Level: Warn,
			Message: name + " is not installed, so this does not run: " + orName(t.For), Fix: t.Install()})
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
			Fix:     "echo 'ai: claude' >> " + userConfig()})
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
	switch cfg.Forge {
	case "github", "gitlab":
		name := map[string]string{"github": "gh", "gitlab": "glab"}[cfg.Forge]
		if t := tools.Lookup(name); !t.Present() {
			r.add(Check{Area: "repository", Rule: "tool-missing", Where: name, Level: Warn,
				Message: name + " is not installed, so this does not run here: " + t.For, Fix: t.Install()})
		}
	}

	prePush := line.Events["pre-push"]
	runs := map[string]bool{}
	for _, s := range prePush {
		runs[s] = true
	}
	if runs["documentalist"] {
		r.add(Check{Area: "repository", Rule: "documentalist-before-push", Where: where, Level: OK,
			Message: "before a push: " + strings.Join(prePush, ", ")})
	} else {
		r.add(Check{Area: "repository", Rule: "documentalist-not-before-push", Where: where, Level: Warn,
			Message: "the documentalist does not run before a push here (pre-push routes: " + orNone(prePush) + "): docs go stale unseen until a merge request, if CI runs workline",
			Fix:     "add `routing: {events: {pre-push: [committer, documentalist]}}` to .workline/config.yaml"})
	}

	d, err := role.Load(o.RolesDir, "documentalist")
	if err != nil {
		r.add(Check{Area: "repository", Rule: "roles-unreadable", Where: o.RolesDir, Level: Error, Message: err.Error()})
		return
	}
	globs := stringList(d.MergedSettings(cfg)["docs"])
	tracked, untracked, err := documentalist.Coverage(o.Repo, globs)
	if err != nil {
		r.add(Check{Area: "repository", Rule: "docs-unreadable", Where: where, Level: Error, Message: err.Error()})
		return
	}
	total := len(tracked) + len(untracked)
	switch {
	case total == 0:
		r.add(Check{Area: "repository", Rule: "no-docs", Where: where, Level: Note,
			Message: fmt.Sprintf("no doc under %v: the documentalist has nothing to look after", globs)})
	case len(untracked) == 0:
		r.add(Check{Area: "repository", Rule: "docs-tracked", Where: where, Level: OK,
			Message: fmt.Sprintf("%d docs declare their sources, all of them", total)})
	case len(tracked) == 0:
		r.add(Check{Area: "repository", Rule: "nothing-tracked", Where: where, Level: Warn,
			Message: fmt.Sprintf("none of the %d docs declares its sources, so none can be found suspect: the documentalist only checks sizes and links", total),
			Fix:     "declare the code each doc describes, as `sources` in its header (workline's roles/documentalist/README.md)"})
	default:
		// Some docs describe no code (decisions, a backlog): not declaring
		// sources may be right, so it is said, not warned about.
		r.add(Check{Area: "repository", Rule: "docs-untracked", Where: where, Level: Note,
			Message: fmt.Sprintf("%d of %d docs declare their sources; these do not, so they are never found suspect: %s", len(tracked), total, sample(untracked)),
			Fix:     "declare the code each doc describes, as `sources` in its header (workline's roles/documentalist/README.md)"})
	}
}

// userConfig is the user's config file, where `ai:` names their agent.
func userConfig() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "~/.config/workline/config.yaml"
	}
	return filepath.Join(dir, "workline", "config.yaml")
}

// usedTools lists the tools the roles use, each once, sorted.
func usedTools(rolesDir string) []string {
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

func orNone(l []string) string {
	if len(l) == 0 {
		return "none"
	}
	return strings.Join(l, ", ")
}

func orName(s string) string {
	if s == "" {
		return "a role uses it"
	}
	return s
}
