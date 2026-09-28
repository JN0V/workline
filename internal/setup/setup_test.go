package setup

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/hooks"
)

// machine is a machine of its own: its git config, its config folder, and a
// PATH holding git and go — the way to install gitleaks — and no tool the
// roles use.
func machine(t *testing.T, typed string) (Machine, *[]string, *bytes.Buffer) {
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "gitconfig"))
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git")
	}
	bin := filepath.Join(dir, "bin")
	os.MkdirAll(bin, 0o755)
	os.Symlink(git, filepath.Join(bin, "git"))
	os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", bin)
	var ran []string
	out := &bytes.Buffer{}
	return Machine{
		In: strings.NewReader(typed), Out: out, Interactive: typed != "",
		Bin:        "/usr/local/bin/workline",
		Hooks:      hooks.Paths{Hooks: filepath.Join(dir, "hooks"), Next: filepath.Join(dir, "next")},
		UserConfig: filepath.Join(dir, "config.yaml"),
		Tools:      []string{"gitleaks"},
		Install:    func(c string) error { ran = append(ran, c); return nil },
	}, &ran, out
}

func hooksPath(t *testing.T) string {
	out, _ := exec.Command("git", "config", "--global", "--get", "core.hooksPath").Output()
	return strings.TrimSpace(string(out))
}

func TestRunAgainReconfigures(t *testing.T) {
	m, _, _ := machine(t, "")
	os.WriteFile(m.UserConfig, []byte("# mine\nai: claude\nother: kept\n"), 0o644)
	if err := Run(Answers{Hooks: "yes", AI: "none", Install: []string{}}, m); err != nil {
		t.Fatal(err)
	}
	if hooksPath(t) != m.Hooks.Hooks {
		t.Fatalf("hooks not installed: core.hooksPath = %q", hooksPath(t))
	}
	data, _ := os.ReadFile(m.UserConfig)
	if !strings.Contains(string(data), "ai: none") || !strings.Contains(string(data), "other: kept") || !strings.Contains(string(data), "# mine") {
		t.Errorf("config rewritten badly:\n%s", data)
	}
	if err := Run(Answers{Hooks: "no", AI: "none", Install: []string{}}, m); err != nil {
		t.Fatal(err)
	}
	if hooksPath(t) != "" {
		t.Errorf("hooks still installed: core.hooksPath = %q", hooksPath(t))
	}
}

func TestTypedAnswers(t *testing.T) {
	// hooks: default; agent: none; gitleaks: yes.
	m, ran, out := machine(t, "\nnone\ny\n")
	m.Interactive = true
	if err := Run(Answers{}, m); err != nil {
		t.Fatal(err)
	}
	if hooksPath(t) != m.Hooks.Hooks {
		t.Errorf("the default, yes, did not install the hooks")
	}
	if len(*ran) != 1 || !strings.Contains((*ran)[0], "gitleaks") {
		t.Errorf("installed %v, want gitleaks\n%s", *ran, out)
	}
	if !strings.Contains(out.String(), "not on your PATH") {
		t.Errorf("a tool still missing after its install is not said:\n%s", out)
	}
}

func TestWithoutTerminal(t *testing.T) {
	m, ran, _ := machine(t, "")
	err := Run(Answers{AI: "none"}, m)
	if err == nil || !strings.Contains(err.Error(), "--hooks") || strings.Contains(err.Error(), "--ai") {
		t.Fatalf("want the flags left unanswered named, got %v", err)
	}
	if hooksPath(t) != "" || len(*ran) > 0 {
		t.Error("something changed before every answer was known")
	}
	if err := Run(Answers{Yes: true}, m); err != nil {
		t.Fatal(err)
	}
	if len(*ran) != 1 {
		t.Errorf("--yes: a tool used on this machine is installed by default; ran %v", *ran)
	}
}
