// Package setup sets workline up on a machine, asking what to enable: the
// global git hooks, the agent, the tools the roles use. Run again, it offers
// what is set up as the default, so it reconfigures. Each question has a flag
// answering it; without a terminal, every question must be answered so, or
// `--yes` takes the defaults (as gh and terraform do).
package setup

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/hooks"
	"github.com/JN0V/workline/internal/tools"
	"go.yaml.in/yaml/v3"
)

// Answers are the questions answered beforehand, by flags.
type Answers struct {
	Hooks   string   // yes | no; empty: ask
	AI      string   // none | claude | claude:<model>@<effort> | cmd:<command>; empty: ask
	Install []string // the tools to install; nil: ask; empty: none
	Yes     bool     // the default for every question not answered
}

// Machine is what setup changes, and how it talks.
type Machine struct {
	In          io.Reader
	Out         io.Writer
	Interactive bool // In is a terminal
	Bin         string
	Hooks       hooks.Paths
	UserConfig  string   // the file holding `ai:`
	Tools       []string // the tools the roles use
	// Install runs an install command, its output shown as it goes.
	Install func(command string) error
}

// Run asks, then sets up. It stops at the first thing that fails.
func Run(a Answers, m Machine) error {
	if !m.Interactive && !a.Yes {
		var missing []string
		if a.Hooks == "" {
			missing = append(missing, "--hooks yes|no")
		}
		if a.AI == "" {
			missing = append(missing, "--ai none|claude")
		}
		if a.Install == nil {
			missing = append(missing, "--install <tool,...>|none")
		}
		if len(missing) > 0 {
			return fmt.Errorf("no terminal to ask on: answer with %s, or take the defaults with --yes", strings.Join(missing, ", "))
		}
	}
	in := bufio.NewReader(m.In)
	ask := func(question, def string) string {
		fmt.Fprintf(m.Out, "%s [%s] ", question, def)
		if a.Yes || !m.Interactive {
			fmt.Fprintln(m.Out, def)
			return def
		}
		line, _ := in.ReadString('\n')
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
		return def
	}

	// 1. The global hooks: every commit checked as it is written.
	cur, err := gitGlobal("core.hooksPath")
	if err != nil {
		return err
	}
	installed := cur != "" && filepath.Clean(cur) == filepath.Clean(m.Hooks.Hooks)
	hooksOn := a.Hooks
	if hooksOn == "" {
		hooksOn = ask("Check every commit on this machine, with global git hooks?", "yes")
	}
	switch yes(hooksOn) {
	case true:
		msg, err := hooks.InstallGlobal(m.Hooks, m.Bin)
		if err != nil {
			return err
		}
		fmt.Fprintln(m.Out, "  "+msg)
	case false:
		if installed {
			msg, err := hooks.UninstallGlobal(m.Hooks)
			if err != nil {
				return err
			}
			fmt.Fprintln(m.Out, "  "+msg)
		}
	}

	// 2. The agent, for what needs judgement.
	was := readAI(m.UserConfig)
	ai := a.AI
	if ai == "" {
		def := was
		if def == "" {
			def = "none"
			if tools.Lookup("claude").Present() {
				def = "claude"
			}
		}
		ai = ask("Agent for what needs judgement (none, claude)?", def)
	}
	if name, _, _ := strings.Cut(ai, ":"); name != "none" && name != "claude" && name != "cmd" {
		return fmt.Errorf("agent %q: none, claude, claude:<model>@<effort> or cmd:<command>", ai)
	}
	if ai != was {
		if err := writeAI(m.UserConfig, ai); err != nil {
			return err
		}
		fmt.Fprintf(m.Out, "  ai: %s, in %s\n", ai, m.UserConfig)
	}

	// 3. The tools: those the roles use, and the agent's.
	names := append([]string{}, m.Tools...)
	if strings.HasPrefix(ai, "claude") {
		names = append(names, "claude")
	}
	for _, name := range names {
		t := tools.Lookup(name)
		if t.Present() {
			continue
		}
		command := t.Install()
		if !tools.Command(command) {
			fmt.Fprintf(m.Out, "%s is not installed (%s): %s\n", name, t.For, command)
			continue
		}
		var want bool
		if a.Install != nil {
			want = contains(a.Install, name)
		} else {
			def := "no"
			if t.Local {
				def = "yes"
			}
			want = yes(ask(fmt.Sprintf("Install %s — %s — with `%s`?", name, t.For, command), def))
		}
		if !want {
			continue
		}
		if err := m.Install(command); err != nil {
			return fmt.Errorf("installing %s: %w", name, err)
		}
		if !t.Present() {
			fmt.Fprintf(m.Out, "  %s is installed, but not on your PATH: add the folder it went to (`go env GOPATH`/bin for go install, ~/.cargo/bin for cargo)\n", name)
		}
	}
	return nil
}

// Shell runs a command with sh, its output shown as it goes.
func Shell(out io.Writer) func(string) error {
	return func(command string) error {
		fmt.Fprintln(out, "  $ "+command)
		cmd := exec.Command("sh", "-c", command)
		cmd.Stdout, cmd.Stderr = out, out
		return cmd.Run()
	}
}

func yes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "o", "oui", "true":
		return true
	}
	return false
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s || x == "all" {
			return true
		}
	}
	return false
}

func gitGlobal(key string) (string, error) {
	out, err := exec.Command("git", "config", "--global", "--get", key).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return "", nil // not set
	}
	return strings.TrimSpace(string(out)), err
}

// readAI returns `ai:` from the user's config; empty when not set.
func readAI(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	var c struct {
		AI string `yaml:"ai"`
	}
	if yaml.Unmarshal(data, &c) != nil {
		return ""
	}
	return c.AI
}

// writeAI sets `ai:` in the user's config, keeping whatever else it holds.
func writeAI(file, ai string) error {
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	top := doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: not a mapping", file)
	}
	set := false
	for i := 0; i+1 < len(top.Content); i += 2 {
		if top.Content[i].Value == "ai" {
			top.Content[i+1].Value, top.Content[i+1].Tag, set = ai, "!!str", true
		}
	}
	if !set {
		top.Content = append(top.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "ai"}, &yaml.Node{Kind: yaml.ScalarNode, Value: ai})
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, out.Bytes(), 0o644)
}
