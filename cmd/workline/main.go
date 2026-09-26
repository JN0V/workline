// Command workline runs the roles of the line.
//
// Every command and option is described in docs/usage.md.
//
//	workline run-role <role> --event <event> [--ai none|claude|cmd:<command>|fake:<file>|unavailable:<reason>]
//	                  [--repo <dir>] [--roles <dir>] [--forge ...] [--target ...] [--scope ...]
//	                  [--input name=value]... [--input-file name=path]... [--no-apply] [--json]
//	                  [--sarif <file>] [--code-quality <file>]
//	workline route <event> [same options as run-role, but --input-file]
//	workline item ready <id> [--repo <dir>] [--forge ...] [--json]
//	workline apply <run-dir>... | --line <route result> [--json]
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
	wlreport "github.com/JN0V/workline/internal/report"
	"github.com/JN0V/workline/internal/rolefs"
	"github.com/JN0V/workline/internal/routing"
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
	ai := fs.String("ai", "", "agent: none, claude, claude:<model>@<effort>, cmd:<command>, fake:<file>, unavailable:<reason> (default: the project's `ai` setting, else none)")
	repo := fs.String("repo", ".", "repository to work on")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles (default: the roles built into this binary)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	tamper := fs.Bool("test-tamper-before-apply", false, "conformance tests only")
	forgeSpec := fs.String("forge", "", "forge: github, gitlab, none, fake:<file> (default: the project's `forge` setting)")
	target := fs.String("target", "", "issue:<n> or merge-request:<n>, where comments and labels go")
	var scope multi
	fs.Var(&scope, "scope", "a path pattern the task is about (repeatable)")
	noApply := fs.Bool("no-apply", false, "stop after judging; apply later with `workline apply <run-dir>`")
	sarifFile := fs.String("sarif", "", "also write the findings to this file as SARIF, for code scanning")
	cqFile := fs.String("code-quality", "", "also write the findings to this file as a GitLab Code Quality report")
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
		Forge: *forgeSpec, Target: t, Scope: scope, NoApply: *noApply,
	})
	if err := wlreport.Write(absRepo, wlreport.FromRole(name, res), *sarifFile, *cqFile); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
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
		level, where := "", ""
		if f.Level != "" {
			level = " (" + f.Level + ")"
		}
		if f.Where != "" {
			where = " " + f.Where
		}
		fmt.Fprintf(os.Stderr, "  %s%s%s: %s\n", f.Rule, where, level, f.Message)
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
		return documentalist.Post(runDir, repo)
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

// hookCmd is what the installed hooks call: commit-msg and pre-push do work,
// the others hand over.
func hookCmd(args []string) int {
	if len(args) == 0 {
		usage()
	}
	if args[0] == "pre-push" && len(args) >= 2 {
		return prePush(args[1])
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
		// A rewrite replaced what the person wrote: show what is committed.
		if now, err := os.ReadFile(msgFile); err == nil && string(now) != string(data) {
			fmt.Fprintln(os.Stderr, "workline: the commit goes on with this message instead of yours:")
			for _, l := range strings.Split(strings.TrimRight(string(now), "\n"), "\n") {
				if !strings.HasPrefix(l, "#") {
					fmt.Fprintln(os.Stderr, "  │ "+l)
				}
			}
		}
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

// prePushQuiet are the findings a push does not need to show: sizes and links
// the line reports on every run, whatever is being pushed. They are counted,
// and `workline route pre-push` shows them.
var prePushQuiet = map[string]bool{
	"doc-too-long": true, "section-too-long": true, "card-too-short": true, "card-too-long": true,
	"folder-too-long": true, "agent-file-too-long": true, "links-not-checked": true, "nothing-tracked": true,
}

// prePush runs the project's pre-push line on the commits being pushed, with
// the person's agent. Nothing runs unless the project routes pre-push: the
// global hook reaches every repository on the machine. A doc the line patched
// stops the push, so the person reviews it and pushes again.
func prePush(remote string) int {
	root, err := gitRoot(".")
	if err != nil {
		return 0
	}
	if _, err := os.Stat(filepath.Join(root, ".workline", "off")); err == nil {
		return 0
	}
	cfg, err := routing.Load(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		fmt.Fprintln(os.Stderr, "push stopped; `git push --no-verify` skips workline")
		return 1
	}
	if _, routed := cfg.Events["pre-push"]; !routed {
		return 0
	}
	ranges, err := hooks.PushRanges(root, remote, os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	rolesDir, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	for _, rng := range ranges {
		res := line.Run("pre-push", engine.Options{
			Repo: root, RolesDir: rolesDir, AI: os.Getenv("WORKLINE_AI"), DefaultAI: userDefaultAI(),
			Inputs: map[string]string{"range": rng},
		})
		patched, advisory := false, 0
		for _, s := range res.Steps {
			if s.Result == nil {
				continue
			}
			for _, a := range s.Result.Applied {
				patched = patched || a == "patch"
			}
		}
		for _, f := range res.Findings {
			if prePushQuiet[f.Rule] {
				advisory++
				continue
			}
			where := ""
			if f.Where != "" {
				where = " " + f.Where
			}
			fmt.Fprintf(os.Stderr, "workline: %s%s: %s\n", f.Rule, where, strings.ReplaceAll(f.Message, "\n", "\n    "))
		}
		if advisory > 0 {
			fmt.Fprintf(os.Stderr, "workline: %d more findings that do not stop the push: workline route pre-push --input range=%s\n", advisory, rng)
		}
		switch {
		case res.Status != verdict.Pass:
			fmt.Fprintf(os.Stderr, "workline: %s\npush stopped; `git push --no-verify` skips workline\n", res.Summary)
			return 1
		case patched:
			stat, _ := exec.Command("git", "-C", root, "diff", "--stat").Output()
			fmt.Fprintf(os.Stderr, "workline: the documentalist updated docs in your working tree:\n%s", stat)
			fmt.Fprintln(os.Stderr, "push stopped: review them (git diff), commit them, and push again")
			return 1
		}
	}
	return 0
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
	forgeSpec := fs.String("forge", "", "forge: github, gitlab, none, fake:<file> (default: the project's `forge` setting)")
	target := fs.String("target", "", "issue:<n> or merge-request:<n>, where comments and labels go")
	var scope multi
	fs.Var(&scope, "scope", "a path pattern the task is about (repeatable)")
	noApply := fs.Bool("no-apply", false, "judge every step, apply none; apply later with `workline apply` and the runs listed as pending")
	sarifFile := fs.String("sarif", "", "also write every step's findings to this file as SARIF, for code scanning")
	cqFile := fs.String("code-quality", "", "also write every step's findings to this file as a GitLab Code Quality report")
	inputs := pairs{}
	fs.Var(inputs, "input", "input name=value, given to every step (repeatable)")
	_ = fs.Parse(args[1:])
	abs, _ := filepath.Abs(*repo)
	rolesDir, err := resolveRoles(*roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	t, err := parseTarget(*target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	res := line.Run(args[0], engine.Options{Repo: abs, RolesDir: rolesDir, AI: *ai, DefaultAI: userDefaultAI(), Inputs: inputs,
		Forge: *forgeSpec, Target: t, Scope: scope, NoApply: *noApply})
	if err := wlreport.Write(abs, wlreport.FromLine(res), *sarifFile, *cqFile); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Fprintf(os.Stderr, "%s — %s\n", res.Status, res.Summary)
		for _, s := range res.Steps {
			fmt.Fprintf(os.Stderr, "  %-28s %s\n", s.Name, s.Status)
		}
		for _, f := range res.Findings {
			where := ""
			if f.Where != "" {
				where = " " + f.Where
			}
			fmt.Fprintf(os.Stderr, "  %s%s: %s\n", f.Rule, where, f.Message)
		}
		if len(res.Pending) > 0 {
			fmt.Fprintf(os.Stderr, "  to apply: workline apply %s\n", strings.Join(res.Pending, " "))
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

// applyCmd applies judged runs, in the order given — a line's pending runs
// are listed in the line's order — and stops at the first that does not pass.
func applyCmd(args []string) int {
	var dirs []string
	for len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dirs, args = append(dirs, args[0]), args[1:]
	}
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	lineFile := fs.String("line", "", "apply the runs `workline route --no-apply --json` listed as pending in this file")
	_ = fs.Parse(args)
	if *lineFile != "" {
		data, err := os.ReadFile(*lineFile)
		var l line.Result
		if err == nil {
			err = json.Unmarshal(data, &l)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		dirs = append(dirs, l.Pending...)
		if len(dirs) == 0 {
			fmt.Fprintln(os.Stderr, "pass — nothing to apply: the line proposed nothing")
			return 0
		}
	}
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: workline apply <run-dir>... | --line <route result> [--json]")
		return 64
	}
	res := engine.Resume(dirs[0])
	for _, d := range dirs[1:] {
		if res.Status != verdict.Pass {
			break
		}
		next := engine.Resume(d)
		res.Status, res.Summary, res.RunDir = next.Status, next.Summary, next.RunDir
		res.Applied = append(res.Applied, next.Applied...)
		res.Findings = append(res.Findings, next.Findings...)
		res.Handoffs = append(res.Handoffs, next.Handoffs...)
	}
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
