// Command workline runs the roles of the line.
//
//	workline run-role <role> --event <event> [--ai none|fake:<file>|unavailable:<reason>]
//	                  [--repo <dir>] [--roles <dir>]
//	                  [--input name=value]... [--input-file name=path]... [--json]
//	workline route <event> [--repo <dir>] [--ai ...] [--input name=value]... [--json]
//	workline item ready <id> [--repo <dir>] [--forge ...] [--json]
//	workline apply <run-dir> [--json]    (resume a run stopped while applying)
//	workline gate <name> [--repo <dir>] [--json]
//	workline hooks install|uninstall --global | --repo
//	workline hook <git-hook-name> [args]  (called by the installed hooks)
//	workline builtin <role> pre|post      (called by the shipped roles' scripts)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/builtin/documentalist"
	"github.com/JN0V/workline/internal/builtin/releasemanager"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/hooks"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/rolefs"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
	"go.yaml.in/yaml/v3"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "run-role":
		os.Exit(runRole(os.Args[2:]))
	case "builtin":
		os.Exit(builtin(os.Args[2:]))
	case "hooks":
		os.Exit(hooksCmd(os.Args[2:]))
	case "hook":
		os.Exit(hookCmd(os.Args[2:]))
	case "gate":
		os.Exit(gateCmd(os.Args[2:]))
	case "route":
		os.Exit(routeCmd(os.Args[2:]))
	case "item":
		os.Exit(itemCmd(os.Args[2:]))
	case "apply":
		os.Exit(applyCmd(os.Args[2:]))
	}
	usage()
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: workline run-role <role> --event <event> [options]\n       workline hooks install|uninstall --global|--repo")
	os.Exit(64)
}

type pairs map[string]string

func (p pairs) String() string { return fmt.Sprint(map[string]string(p)) }
func (p pairs) Set(s string) error {
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return fmt.Errorf("expected name=value, got %q", s)
	}
	p[k] = v
	return nil
}

func runRole(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		usage()
	}
	name := args[0]
	fs := flag.NewFlagSet("run-role", flag.ExitOnError)
	event := fs.String("event", "", "event the role runs on")
	ai := fs.String("ai", "", "agent: none, claude, fake:<file>, unavailable:<reason> (default: the project's `ai` setting, else none)")
	repo := fs.String("repo", ".", "repository to work on")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles (default: the roles built into this binary)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	tamper := fs.Bool("test-tamper-before-apply", false, "conformance tests only")
	forgeSpec := fs.String("forge", "", "forge: github, gitlab, none, fake:<file> (default: the project's `forge` setting)")
	target := fs.String("target", "", "issue:<n> or merge-request:<n>, where comments and labels go")
	var scope multi
	fs.Var(&scope, "scope", "a path pattern the task is about (repeatable)")
	inputs, inputFiles := pairs{}, pairs{}
	fs.Var(inputs, "input", "input name=value (repeatable)")
	fs.Var(inputFiles, "input-file", "input name=path, written back by intentions that target it (repeatable)")
	_ = fs.Parse(args[1:])

	targets := map[string]string{}
	for k, p := range inputFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		inputs[k] = string(data)
		targets[k] = p
	}
	absRepo, _ := filepath.Abs(*repo)
	rolesDir, err := resolveRoles(*roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	absRoles, _ := filepath.Abs(rolesDir)
	t, err := parseTarget(*target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	res := engine.Run(engine.Options{
		Repo: absRepo, RolesDir: absRoles, Role: name, Event: *event, AI: *ai,
		Inputs: inputs, Targets: targets, TamperBeforeApply: *tamper,
		Forge: *forgeSpec, Target: t, Scope: scope,
	})
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	switch res.Status {
	case verdict.Pass:
		return 0
	case verdict.Human:
		return 2
	case verdict.BlockedExternal:
		return 3
	}
	return 1
}

func report(r *engine.Result) {
	fmt.Fprintf(os.Stderr, "%s", r.Status)
	if r.Summary != "" {
		fmt.Fprintf(os.Stderr, " — %s", r.Summary)
	}
	fmt.Fprintln(os.Stderr)
	for _, h := range r.Handoffs {
		fmt.Fprintf(os.Stderr, "  handoff: %v\n", h)
	}
	for _, f := range r.Findings {
		level := ""
		if f.Level != "" {
			level = " (" + f.Level + ")"
		}
		fmt.Fprintf(os.Stderr, "  %s%s: %s\n", f.Rule, level, f.Message)
	}
}

// resolveRoles returns the given folder, or the built-in roles extracted to the cache.
func resolveRoles(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	return rolefs.Dir()
}

func builtin(args []string) int {
	if len(args) != 2 {
		usage()
	}
	runDir := os.Getenv("WORKLINE_RUN_DIR")
	if runDir == "" {
		fmt.Fprintln(os.Stderr, "workline builtin: WORKLINE_RUN_DIR is not set; this command is run by the engine")
		return 99
	}
	repo, _ := os.Getwd()
	switch args[0] + " " + args[1] {
	case "committer pre":
		return committer.Pre(runDir, repo)
	case "committer post":
		return committer.Post(runDir)
	case "documentalist pre":
		return documentalist.Pre(runDir, repo)
	case "documentalist post":
		return documentalist.Post(runDir)
	case "release-manager pre":
		return releasemanager.Pre(runDir, repo)
	case "release-manager post":
		return releasemanager.Post(runDir)
	}
	fmt.Fprintf(os.Stderr, "workline builtin: no built-in step %q\n", strings.Join(args, " "))
	return 99
}

func hooksCmd(args []string) int {
	if len(args) != 2 || (args[0] != "install" && args[0] != "uninstall") || (args[1] != "--global" && args[1] != "--repo") {
		usage()
	}
	bin, err := os.Executable()
	if err == nil {
		bin, err = filepath.EvalSymlinks(bin)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	var msg string
	switch args[0] + " " + args[1] {
	case "install --global":
		var p hooks.Paths
		if p, err = hooks.DefaultPaths(); err == nil {
			msg, err = hooks.InstallGlobal(p, bin)
		}
	case "uninstall --global":
		var p hooks.Paths
		if p, err = hooks.DefaultPaths(); err == nil {
			msg, err = hooks.UninstallGlobal(p)
		}
	case "install --repo":
		var root string
		if root, err = gitRoot("."); err == nil {
			msg, err = hooks.InstallRepo(root, bin)
		}
	default:
		err = fmt.Errorf("uninstall --repo: delete .githooks/commit-msg")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	fmt.Println(msg)
	return 0
}

// hookCmd is what the installed hooks call. Today only commit-msg does work.
func hookCmd(args []string) int {
	if len(args) == 0 {
		usage()
	}
	if args[0] != "commit-msg" || len(args) < 2 {
		return 0
	}
	root, err := gitRoot(".")
	if err != nil {
		return 0 // not in a repository: nothing to check
	}
	if _, err := os.Stat(filepath.Join(root, ".workline", "off")); err == nil {
		return 0 // this repository opted out
	}
	rolesDir, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	msgFile, _ := filepath.Abs(args[1])
	data, err := os.ReadFile(msgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	res := line.Run("commit-msg", engine.Options{
		Repo: root, RolesDir: rolesDir, AI: os.Getenv("WORKLINE_AI"), DefaultAI: userDefaultAI(),
		Inputs: map[string]string{"message": string(data)}, Targets: map[string]string{"message": msgFile},
	})
	quiet := res.Status == verdict.Pass && len(res.Findings) == 0
	for _, s := range res.Steps {
		if s.Result != nil && (len(s.Result.Applied) > 0 || len(s.Result.Findings) > 0) {
			quiet = false
		}
	}
	if quiet {
		return 0
	}
	for _, s := range res.Steps {
		if s.Result != nil {
			fmt.Fprintf(os.Stderr, "workline %s: ", s.Name)
			report(s.Result)
		}
	}
	if res.Status == verdict.Pass {
		return 0
	}
	if len(res.Steps) == 0 || res.Steps[len(res.Steps)-1].Result == nil {
		fmt.Fprintln(os.Stderr, "workline:", res.Summary)
		for _, f := range res.Findings {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", f.Rule, f.Message)
		}
	}
	fmt.Fprintln(os.Stderr, "commit aborted; your message is kept in", msgFile)
	return 1
}

func gitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// userDefaultAI reads `ai:` from ~/.config/workline/config.yaml, the agent a
// person wants by default when a project does not say.
func userDefaultAI() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(cfg, "workline", "config.yaml"))
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

func gateCmd(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		usage()
	}
	fs := flag.NewFlagSet("gate", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository to check")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	_ = fs.Parse(args[1:])
	abs, _ := filepath.Abs(*repo)
	v := gate.Run(abs, args[0])
	res := &engine.Result{Status: v.Status, Summary: v.Summary, Findings: v.Findings, Applied: []string{}, Refused: []string{}}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	if v.Status == verdict.Pass {
		return 0
	}
	return 1
}

func routeCmd(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		usage()
	}
	fs := flag.NewFlagSet("route", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository to work on")
	ai := fs.String("ai", "", "agent for every role (default: the project's, else yours, else none)")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	inputs := pairs{}
	fs.Var(inputs, "input", "input name=value (repeatable)")
	_ = fs.Parse(args[1:])
	abs, _ := filepath.Abs(*repo)
	rolesDir, err := resolveRoles(*roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	res := line.Run(args[0], engine.Options{Repo: abs, RolesDir: rolesDir, AI: *ai, DefaultAI: userDefaultAI(), Inputs: inputs})
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Fprintf(os.Stderr, "%s — %s\n", res.Status, res.Summary)
		for _, s := range res.Steps {
			fmt.Fprintf(os.Stderr, "  %-28s %s\n", s.Name, s.Status)
		}
		for _, f := range res.Findings {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", f.Rule, f.Message)
		}
	}
	return exitFor(res.Status)
}

func itemCmd(args []string) int {
	if len(args) < 2 || args[0] != "ready" {
		fmt.Fprintln(os.Stderr, "usage: workline item ready <id> [--repo <dir>] [--json]")
		return 64
	}
	fs := flag.NewFlagSet("item", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository holding .workline/work/")
	forgeSpec := fs.String("forge", "", "read the item from this forge instead of .workline/work/")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	_ = fs.Parse(args[2:])
	abs, _ := filepath.Abs(*repo)
	var v *verdict.Verdict
	if *forgeSpec != "" && *forgeSpec != "none" {
		f, err := forge.Open(*forgeSpec, abs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 64
		}
		var id int
		fmt.Sscan(args[1], &id)
		v = work.ReadyOnForge(f, id)
	} else {
		v = work.Ready(abs, args[1])
	}
	res := &engine.Result{Status: v.Status, Summary: v.Summary, Findings: v.Findings, Applied: []string{}, Refused: []string{}}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return exitFor(v.Status)
}

func exitFor(status string) int {
	switch status {
	case verdict.Pass:
		return 0
	case verdict.Human:
		return 2
	case verdict.BlockedExternal:
		return 3
	}
	return 1
}

func applyCmd(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "usage: workline apply <run-dir> [--json]")
		return 64
	}
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	_ = fs.Parse(args[1:])
	res := engine.Resume(args[0])
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return exitFor(res.Status)
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

func parseTarget(s string) (*forge.Target, error) {
	if s == "" {
		return nil, nil
	}
	kind, num, ok := strings.Cut(s, ":")
	var id int
	if _, err := fmt.Sscan(num, &id); !ok || err != nil || (kind != "issue" && kind != "merge-request") {
		return nil, fmt.Errorf("--target must be issue:<n> or merge-request:<n>, not %q", s)
	}
	return &forge.Target{Kind: kind, ID: id}, nil
}
