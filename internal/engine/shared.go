package engine

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
)

// The shared pre and post (ADR-0070): a role with no `pre` script, or no
// `post`, runs on these. They are the engine's own plumbing, run in its
// process, so a role that is a prompt adds a folder of files and no code.
//
// pre reads the inputs the role declares in role.yaml (`inputs`), and
// writes the task: the inputs, whole, in in/task.md; the role's skills in
// in/skills. post says the verdict: it holds nothing back that the engine
// checks itself — each proposal against the role's `intentions` and
// `duties`, before post — and applies nothing: the trusted apply step does.

// Inputs are the inputs a role may declare, each read by the shared pre
// (`issues:<label>` too, the open issues carrying a label); `wake`, `sample` and `runs` of the ADR are not read yet, and said so. One not named
// here is refused, said: never read as nothing.
var Inputs = []string{"target", "commits", "issues"}

// sharedStep runs the shared pre or post for a role with no such script.
// It returns the exit code the script would, as script does.
func sharedStep(r *role.Role, step, repo string, env []string) (int, error) {
	get := func(k string) string {
		for i := len(env) - 1; i >= 0; i-- {
			if v, ok := strings.CutPrefix(env[i], k+"="); ok {
				return v
			}
		}
		return ""
	}
	runDir := get("WORKLINE_RUN_DIR")
	phase(r.Name, step+" (shared)")
	if step == "pre" {
		return sharedPre(r, repo, runDir, get)
	}
	return sharedPost(r, runDir, get)
}

// hasScript says whether the role's folder holds that step's script.
func hasScript(r *role.Role, step string) bool {
	_, err := os.Stat(filepath.Join(r.Dir, step))
	return err == nil
}

func sharedPre(r *role.Role, repo, runDir string, get func(string) string) (int, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s, on %s\n\n%s\n\n", r.Name, get("WORKLINE_EVENT"), strings.TrimSpace(r.Mission))
	if len(r.Inputs) > 0 {
		b.WriteString("## What you read\n\n")
	}
	for _, in := range r.Inputs {
		text, err := readInput(in, repo, runDir, get)
		if err != nil {
			return 0, fmt.Errorf("role %q: input %q not read: %w", r.Name, in, err)
		}
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", in, strings.TrimSpace(text))
	}
	task := filepath.Join(runDir, "in", "task.md")
	if err := os.WriteFile(task, []byte(b.String()), 0o644); err != nil {
		return 0, err
	}
	if len(r.Context.Skills) > 0 {
		if err := os.WriteFile(filepath.Join(runDir, "in", agent.SkillsFile), []byte(strings.Join(r.Context.Skills, "\n")+"\n"), 0o644); err != nil {
			return 0, err
		}
	}
	// A prompt the role names and does not have is said now, with any
	// agent, never run on without it. Too large a prompt is the run's to
	// say once the agent is asked.
	if _, _, err := agent.Prompt(agent.Request{RunDir: runDir, Repo: repo, Role: r}); err != nil && !errors.Is(err, agent.ErrOverBudget) {
		return 0, fmt.Errorf("role %q: %w", r.Name, err)
	}
	return exitOK, nil
}

// readInput is one declared input, as text for the task.
func readInput(name, repo, runDir string, get func(string) string) (string, error) {
	base, label, _ := strings.Cut(name, ":")
	var f forge.Forge
	needForge := func() (forge.Forge, error) {
		if f != nil {
			return f, nil
		}
		spec := get("WORKLINE_FORGE")
		if spec == "" {
			return nil, fmt.Errorf("no forge to read it from (--forge, or the project's forge)")
		}
		var err error
		if f, err = forge.Open(spec, repo); err != nil {
			return nil, err
		}
		if f == nil {
			return nil, fmt.Errorf("no forge to read it from (--forge, or the project's forge)")
		}
		return f, nil
	}
	issueText := func(is forge.Issue) string {
		labels := ""
		if len(is.Labels) > 0 {
			labels = " [" + strings.Join(is.Labels, ", ") + "]"
		}
		return fmt.Sprintf("- #%d %s%s\n\n%s\n", is.ID, is.Title, labels, indent(is.Body))
	}
	switch base {
	case "issues":
		fg, err := needForge()
		if err != nil {
			return "", err
		}
		b, ok := fg.(forge.Backlog)
		if !ok {
			return "", fmt.Errorf("this forge keeps no issues to list")
		}
		all, err := b.Issues()
		if err != nil {
			return "", err
		}
		var out []string
		for _, is := range all {
			if label == "" || slices.Contains(is.Labels, label) {
				out = append(out, issueText(is))
			}
		}
		if len(out) == 0 {
			return "none", nil
		}
		return strings.Join(out, "\n"), nil
	case "target":
		t := get("WORKLINE_TARGET")
		_, id, _ := strings.Cut(t, ":")
		n, convErr := strconv.Atoi(id)
		if t == "" || convErr != nil {
			return "", fmt.Errorf("the run is on no issue nor merge request (--target issue:<number>)")
		}
		fg, err := needForge()
		if err != nil {
			return "", err
		}
		is, err := fg.Issue(n)
		if err != nil {
			return "", err
		}
		return issueText(*is), nil
	case "commits":
		data, err := os.ReadFile(filepath.Join(runDir, "in", "input", "range"))
		if err != nil || strings.TrimSpace(string(data)) == "" {
			return "", fmt.Errorf("the run is given no commit range (input range)")
		}
		rng := strings.TrimSpace(string(data))
		out, err := exec.Command("git", "-C", repo, "log", "--format=%h %s", rng).Output()
		if err != nil {
			return "", fmt.Errorf("the commits of %s cannot be read: %w", rng, err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("not one the engine reads (%s)", strings.Join(Inputs, ", "))
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

func sharedPost(r *role.Role, runDir string, get func(string) string) (int, error) {
	v := &verdict.Verdict{Status: verdict.Pass}
	code := exitOK
	switch {
	case get("WORKLINE_AGENT_UNREAD") != "":
		v.Status, code = verdict.Block, exitBlock
		v.Summary = "the agent's answer could not be read"
		v.Findings = []verdict.Finding{{Rule: "agent-invalid-output", Message: get("WORKLINE_AGENT_UNREAD")}}
	case get("WORKLINE_AGENT_REFUSED") != "":
		v.Status, code = verdict.Block, exitBlock
		v.Summary = "the agent's proposals were refused"
		v.Findings = []verdict.Finding{{Rule: "intention-refused", Message: get("WORKLINE_AGENT_REFUSED")}}
	case get("WORKLINE_AGENT_OVER_BUDGET") != "":
		v.Summary = "the agent was not asked: " + get("WORKLINE_AGENT_OVER_BUDGET")
	case get("WORKLINE_AGENT_ANSWERED") == "true":
		v.Summary = "the agent answered; its proposals are held to the role's intentions and duties, applied by the apply step"
	case cmp.Or(get("WORKLINE_AI"), "none") == "none":
		switch r.WithoutAI {
		case "block":
			v.Status, code = verdict.Block, exitBlock
			v.Summary = fmt.Sprintf("skipped: %s asks an agent and none is set (without-ai: block)", r.Name)
		case "pass":
			v.Summary = "nothing to do without an agent"
		default:
			v.Summary = fmt.Sprintf("skipped: %s asks an agent, agent none: its inputs read, nothing proposed, nothing written", r.Name)
		}
	default:
		v.Status, code = verdict.Block, exitExternal
		v.Summary = "the agent did not answer"
	}
	if code == exitExternal {
		v.Status = verdict.BlockedExternal
	}
	return code, verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), v)
}
