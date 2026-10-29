// Command workline runs the roles of the line.
//
// Every command and option is described in docs/usage.md.
//
//	workline run-role <role> --event <event> [--ai none|claude|cmd:<command>|fake:<file>|unavailable:<reason>]
//	                  [--repo <dir>] [--roles <dir>] [--forge ...] [--target ...] [--scope ...]
//	                  [--input name=value]... [--input-file name=path]... [--no-apply] [--json]
//	                  [--sarif <file>] [--code-quality <file>] [--summary <file>]
//	workline route <event> [same options as run-role, but --input-file]
//	workline review [--base <ref> | --spec <file> | --issue <n>] [--lenses all|<lens,...>] [--repo <dir>] [--ai ...] [--json]
//	workline item ready <id> [--repo <dir>] [--forge ...] [--json]
//	workline issues [list] [--repo <dir>] | show <n>|!<n> [--repo <dir>]   (the local forge)
//	workline apply <run-dir>... | --line <route result> [--json] [--summary <file>]
//	workline gate <name> [--repo <dir>] [--json]
//	workline hooks install|uninstall --global | --repo
//	workline setup [--hooks yes|no] [--ai none|claude|...] [--install <tool,...>|none] [--yes]
//	workline doctor [--repo <dir>] [--json]
//	workline docs [--repo <dir>] [--ai ...]
//	workline init [--repo <dir>] [--ai ...] [--roles <dir>] [--review] [--human-po yes|no] [--json]
//	workline hook <git-hook-name> [args]  (called by the installed hooks)
//	workline builtin <role> pre|post      (called by the shipped roles' scripts)
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/builtin/documentalist"
	"github.com/JN0V/workline/internal/builtin/inspector"
	"github.com/JN0V/workline/internal/builtin/productowner"
	"github.com/JN0V/workline/internal/builtin/reviewer"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/doctor"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/hooks"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/release"
	wlreport "github.com/JN0V/workline/internal/report"
	"github.com/JN0V/workline/internal/review"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/rolefs"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/sample"
	"github.com/JN0V/workline/internal/setup"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
	"go.yaml.in/yaml/v3"
)

// main runs the command the arguments name (cli.go: the commands, their
// help, and misuse, which exits 64).
func main() {
	os.Exit(dispatch(os.Args[1:]))
}

// followCmd rebuilds the roles' release merge requests on their base's new
// tip, run when it moves (ADR-0034); with --gardening, run when gardening
// runs, their gardening merge requests too, or closes them when nothing is
// left to merge. It calls no agent.
func followCmd(args []string) int {
	fs := newFlags("follow")
	repo := fs.String("repo", ".", "repository")
	base := fs.String("base", "", "the branch the merge requests go into (default: the branch checked out)")
	forgeSpec := fs.String("forge", "", "github, gitlab, local, cmd:<command> (default: the project's `forge` setting)")
	gardening := fs.Bool("gardening", false, "also the roles' gardening merge requests: rebuilt, or closed when nothing is left to merge (run when gardening runs)")
	verdictFile := fs.String("verdict", "", "with the project's `merge:` policy, keep first the line's verdict on a merge request's head, the file `workline route --verdict` wrote")
	verdictOnly := fs.Bool("verdict-only", false, "with --verdict: keep the verdict, and follow nothing else (a job no newer run cancels)")
	group := fs.String("merge-group", "", "a merge queue's group, its branch (gh-readonly-queue/<base>/pr-<n>-<head>): with the project's `merge:` policy, set on its commit (--head) the yes the head queued holds, and follow nothing else")
	groupHead := fs.String("head", "", "with --merge-group: the group's commit")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	var summaryFiles multi
	fs.Var(&summaryFiles, "summary", "also add a Markdown summary to this file, for a CI job's page (repeatable; a file named .html gets HTML)")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: not in a git repository")
		return 64
	}
	cfg, err := role.LoadProjectConfig(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	if *forgeSpec == "" {
		*forgeSpec = cfg.Forge
	}
	o := engine.FollowOptions{Repo: root, Forge: *forgeSpec, Base: *base, Gardening: *gardening, Verdict: *verdictFile, VerdictOnly: *verdictOnly,
		Group: *group, GroupHead: *groupHead}
	switch {
	case o.VerdictOnly && o.Verdict == "":
		fmt.Fprintln(os.Stderr, "workline: --verdict-only needs --verdict <file>")
		return 64
	case (o.Group == "") != (o.GroupHead == ""):
		fmt.Fprintln(os.Stderr, "workline: --merge-group and --head go together: the group's branch and its commit")
		return 64
	case o.Group != "" && (o.Verdict != "" || o.Gardening || o.Base != ""):
		fmt.Fprintln(os.Stderr, "workline: --merge-group follows nothing else: no --verdict, --gardening nor --base")
		return 64
	}
	if cfg.Merge != nil {
		o.Merge, o.Alone = cfg.Merge.Autonomy, cfg.Merge.Alone
	}
	if o.Verdict != "" {
		if o.Verdict, err = filepath.Abs(o.Verdict); err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 64
		}
	}
	gone, err := followSettings(&o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	res := engine.Follow(o)
	if gone {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "setting-gone", Level: "warn",
			Message: "the documentalist's update-branches is no longer read: workline merges nothing into a person's merge request (ADR-0051); take it off the project's settings"})
	}
	if err := wlreport.AppendSummary(summaryFiles, wlreport.RoleSummary("follow", "", res)); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return exitFor(res.Status)
}

// followSettings reads what `workline follow` takes of the documentalist's
// settings: which files are docs; and the release tool's branches, a
// bot's. gone says the project still sets `update-branches`, whose merges
// into people's merge requests ADR-0051 removed.
func followSettings(o *engine.FollowOptions) (gone bool, err error) {
	rolesDir, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		return false, err
	}
	r, err := role.Load(rolesDir, "documentalist")
	if err != nil {
		return false, err
	}
	cfg, err := role.LoadProjectConfig(o.Repo)
	if err != nil {
		return false, err
	}
	settings := r.MergedSettings(cfg)
	_, gone = settings["update-branches"]
	s, err := documentalist.SettingsFrom(settings)
	if err != nil {
		return false, fmt.Errorf(".workline/config.yaml: the documentalist's settings: %w", err)
	}
	rs, err := release.Load(o.Repo)
	if err != nil {
		return false, err
	}
	o.IsDoc, o.Release = s.IsDoc, rs.IsBranch
	o.Derive = func(dir string, docs map[string]string) bool {
		findings, stale := documentalist.Derive(dir, docs, s.Derive, false)
		for _, f := range findings {
			if f.Level != "" { // a command unknown or failing, a block unclosed
				return false
			}
		}
		return len(stale) == 0
	}
	return gone, nil
}

// version is the release, set when one is built (-X main.version=v0.1.0).
var version = ""

// engineVersion is the release this engine is, else what Go recorded when it
// was installed: a module version, or "(devel)" for a build from a checkout.
func engineVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
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
	fs := newFlags("run-role")
	event := fs.String("event", "", "event the role runs on")
	ai := fs.String("ai", "", "agent: none, claude, claude:<model>@<effort>, cmd:<command>, fake:<file>, unavailable:<reason> (default: the project's `ai` setting, else none)")
	repo := fs.String("repo", ".", "repository to work on")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles (default: the roles built into this binary)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	tamper := fs.Bool("test-tamper-before-apply", false, "conformance tests only")
	forgeSpec := fs.String("forge", "", "forge: github, gitlab, none, fake:<file> (default: the project's `forge` setting)")
	target := fs.String("target", "", "issue:<n> or merge-request:<n>, where comments and labels go")
	branch := fs.String("branch", "", "the branch the merge request --target names comes from, when the forge is not to be asked (default: the forge says)")
	var scope multi
	fs.Var(&scope, "scope", "a path pattern the task is about (repeatable)")
	noApply := fs.Bool("no-apply", false, "stop after judging; apply later with `workline apply <run-dir>`")
	openMR := fs.Bool("open-merge-request", false, "put what the patches write on a branch of the role, and open a merge request for it (needs a forge)")
	sarifFile := fs.String("sarif", "", "also write the findings to this file as SARIF, for code scanning")
	cqFile := fs.String("code-quality", "", "also write the findings to this file as a GitLab Code Quality report")
	var summaryFiles multi
	fs.Var(&summaryFiles, "summary", "also add a Markdown summary of the run to this file, for a CI job's page (repeatable; a file named .html gets HTML)")
	inputs, inputFiles := pairs{}, pairs{}
	fs.Var(inputs, "input", "input name=value (repeatable)")
	fs.Var(inputFiles, "input-file", "input name=path, written back by intentions that target it (repeatable)")
	name, code, done := parseAfter(fs, args, "a role")
	if done {
		return code
	}

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
		Forge: *forgeSpec, Target: t, Branch: *branch, Scope: scope, NoApply: *noApply, OpenMergeRequest: *openMR,
	})
	if err := wlreport.Write(absRepo, *event, wlreport.FromRole(name, res), *sarifFile, *cqFile); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if err := wlreport.AppendSummary(summaryFiles, wlreport.RoleSummary("run-role "+name+" ("+*event+")", name, res)); err != nil {
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
	for _, n := range r.Notes {
		fmt.Fprintf(os.Stderr, "  note from the agent: %s\n", strings.ReplaceAll(n, "\n", "\n    "))
	}
}

// reviewCmd has the reviewer review a branch on this machine, at the end of
// development, before it is pushed: base..HEAD, every lens (ADR-0020). The
// findings are printed for the person; out/review.json in the run folder,
// or --json, gives them to the author's agent, to fix before pushing.
func reviewCmd(args []string) int {
	fs := newFlags("review")
	repo := fs.String("repo", ".", "repository to review")
	base := fs.String("base", "", "where the change starts: the branch it goes into (default: the reviewer's `base` setting, main)")
	lenses := fs.String("lenses", "", "all (the default on a machine), or the lenses to ask, separated by commas")
	spec := fs.String("spec", "", "review this spec file before it is built, through the spec lenses, not the code (#128)")
	issue := fs.String("issue", "", "review this issue, by its number, as a spec before it is built (#128)")
	ai := fs.String("ai", "", "agent: none, claude, claude:<model>@<effort>, cmd:<command>, fake:<file> (default: the project's `ai` setting, else yours)")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles (default: the roles built into this binary)")
	forgeSpec := fs.String("forge", "", "forge the issues for what lies outside the change go to (default: the project's `forge` setting)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	sarifFile := fs.String("sarif", "", "also write the findings to this file as SARIF")
	cqFile := fs.String("code-quality", "", "also write the findings to this file as a GitLab Code Quality report")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: not in a git repository")
		return 64
	}
	rolesDir, err := resolveRoles(*roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	absRoles, _ := filepath.Abs(rolesDir)
	inputs := map[string]string{}
	if *base != "" {
		inputs["range"] = *base + "..HEAD"
	}
	if *lenses != "" {
		inputs["lenses"] = *lenses
	}
	// A spec, not code: the reviewer's spec event, its input the file,
	// relative to the repository, or the issue's number (#128).
	event := "review"
	if *spec != "" || *issue != "" {
		if *spec != "" && *issue != "" || *base != "" {
			fmt.Fprintln(os.Stderr, "workline: --spec, --issue and --base each name what is reviewed: give one")
			return 64
		}
		event = "spec"
		if *issue != "" {
			inputs["issue"] = strings.TrimPrefix(*issue, "#")
		} else {
			rel, err := specPath(root, *spec)
			if err != nil {
				fmt.Fprintln(os.Stderr, "workline:", err)
				return 64
			}
			inputs["spec"] = rel
		}
	}
	res := engine.Run(engine.Options{Repo: root, RolesDir: absRoles, Role: "reviewer", Event: event, AI: *ai, DefaultAI: userDefaultAI(),
		Inputs: inputs, Forge: *forgeSpec})
	if err := wlreport.Write(root, event, wlreport.FromRole("reviewer", res), *sarifFile, *cqFile); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
		reviewTokens(filepath.Join(res.RunDir, "out", "review.json"))
		if _, err := os.Stat(filepath.Join(res.RunDir, "out", "review.json")); err == nil {
			fmt.Fprintf(os.Stderr, "For the author's agent: %s\n", filepath.Join(res.RunDir, "out", "review.json"))
		}
	}
	return exitFor(res.Status)
}

// specPath is a spec file given on the command line, from the current
// folder, as a path in the repository: the reviewer reads it there.
func specPath(root, file string) (string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("the spec: %v", err)
	}
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	if a, err := filepath.EvalSymlinks(abs); err == nil {
		abs = a
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("the spec %s lies outside the repository %s: the reviewer reads it against the code there", file, root)
	}
	return filepath.ToSlash(rel), nil
}

// reviewTokens prints what each call of a review used: the lenses' and
// each judge's, the total being in the summary (#147).
func reviewTokens(file string) {
	var v struct{ Calls []reviewer.Spent }
	if data, err := os.ReadFile(file); err != nil || json.Unmarshal(data, &v) != nil {
		return
	}
	for _, c := range v.Calls {
		of := c.Lenses
		if c.Where != "" {
			of += ", " + c.Where
		}
		fmt.Fprintf(os.Stderr, "  tokens, %s (%s): %d in, %d out\n", c.For, of, c.TokensIn, c.TokensOut)
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
	if code, done := positional(newFlags("builtin"), args, 2); done {
		return code
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
		return committer.Post(runDir, repo)
	case "documentalist pre":
		return documentalist.Pre(runDir, repo)
	case "documentalist post":
		return documentalist.Post(runDir, repo)
	case "product-owner pre":
		return productowner.Pre(runDir, repo)
	case "product-owner post":
		return productowner.Post(runDir, repo)
	case "reviewer pre":
		return reviewer.Pre(runDir, repo)
	case "reviewer post":
		return reviewer.Post(runDir, repo)
	case "inspector pre":
		return inspector.Pre(runDir, repo)
	case "inspector post":
		return inspector.Post(runDir, repo)
	case "developer pre":
		return developer.Pre(runDir, repo)
	case "developer post":
		return developer.Post(runDir, repo)
	case "tester pre":
		return tester.Pre(runDir, repo)
	case "tester post":
		return tester.Post(runDir, repo)
	}
	fmt.Fprintf(os.Stderr, "workline builtin: no built-in step %q\n", strings.Join(args, " "))
	return 99
}

func hooksCmd(args []string) int {
	fs := newFlags("hooks")
	if len(args) > 0 && isHelp(args[0]) {
		commandHelp(os.Stdout, fs)
		return 0
	}
	if len(args) != 2 || (args[0] != "install" && args[0] != "uninstall") || (args[1] != "--global" && args[1] != "--repo") {
		return misuse(fs, "install or uninstall, then --global or --repo")
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
	if len(args) == 0 || isHelp(args[0]) {
		// git always names the hook: help, or a person's mistake.
		if code, done := positional(newFlags("hook"), args, 1); done {
			return code
		}
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
	// A commit over its budget says so, quiet or not; its seconds go with
	// what the hook says, a rewrite's above all (ADR-0050).
	over := res.OverBudget()
	if quiet {
		if over != "" {
			fmt.Fprintln(os.Stderr, "workline: over its budget: "+over)
		}
		return 0
	}
	for _, s := range res.Steps {
		if s.Result != nil {
			fmt.Fprintf(os.Stderr, "workline %s: ", s.Name)
			report(s.Result)
		}
	}
	if t := res.Times(); t != "" {
		fmt.Fprintln(os.Stderr, "workline: "+t)
	}
	if over != "" {
		fmt.Fprintln(os.Stderr, "workline: over its budget: "+over)
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
	// On any refusal, a rewrite cut at its cap included (ADR-0050), the
	// person's words are kept, and the command that takes them up again
	// is given.
	fmt.Fprintf(os.Stderr, "commit aborted; your message is kept in %s: to edit it and commit again, git commit -e -F %s\n", args[1], args[1])
	return 1
}

// prePushQuiet are the findings a push does not need to show: the sizes and
// links of the docs it touched, which the merge request says again. They
// are counted, and `workline route pre-push` shows them. Passages written
// twice are not looked for at a push (ADR-0050).
var prePushQuiet = map[string]bool{
	"doc-too-long": true, "section-too-long": true, "card-too-short": true, "card-too-long": true,
	"folder-too-long": true, "agent-file-too-long": true, "links-not-checked": true, "nothing-tracked": true,
	"pending":        true, // due at a later moment, the release, which says so
	"derived-behind": true, // regenerated on the default branch after the merge (ADR-0027)
	"dead-link":      true, "cites-superseded": true,
}

// prePush runs the project's pre-push line on the commits being pushed, with
// no agent (ADR-0010), then, if the person asked for it, has them approve
// the push (ADR-0011: off unless their own config says `approve-push: true`).
// The line runs only when the project routes pre-push, since the global hook
// reaches every repository on the machine. A doc the line patched stops the
// push, so the person reviews it and pushes again.
func prePush(remote string) int {
	root, err := gitRoot(".")
	if err != nil {
		return 0
	}
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	refs, err := hooks.PushedRefs(root, remote, bytes.NewReader(stdin))
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	code := prePushLine(root, refs)
	if code != 0 {
		return code
	}
	if !userConfig().approvePush() {
		return 0
	}
	push := hooks.Push{Repo: root, Remote: remote, Refs: refs, Via: userConfig().ApproveVia}
	if hooks.Approve(push, os.Stderr) {
		return 0
	}
	return 1
}

// prePushLine runs the project's pre-push line on each range pushed, with no
// agent: a push never waits on one (ADR-0007). It counts the docs the
// commits made suspect, never lists them: they are judged on the merge
// request, by gardening or with `workline docs` (ADR-0010). The other docs
// are not read at a push: the merge request and gardening do (ADR-0050).
func prePushLine(root string, refs []hooks.Ref) int {
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
	rolesDir, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "workline: checking the commits pushed (%s), no agent…\n", strings.Join(cfg.Events["pre-push"], ", "))
	// The push's seconds, every range's steps together, and its budget
	// (ADR-0050): said before what stops it, never a block.
	start := time.Now()
	var runs []*line.Result
	times := func() {
		if len(runs) == 0 {
			return
		}
		all := line.Join(runs)
		all.Seconds = line.Since(start)
		line.Measure(all, cfg, "push")
		fmt.Fprintln(os.Stderr, "workline: "+all.Times())
		if over := all.OverBudget(); over != "" {
			fmt.Fprintln(os.Stderr, "workline: over its budget: "+over)
		}
	}
	var suspect []string
	for _, ref := range refs {
		rng := ref.Range
		if rng == "" {
			continue
		}
		res := line.Run("pre-push", engine.Options{
			Repo: root, RolesDir: rolesDir, AI: "none",
			Inputs: map[string]string{"range": rng},
		})
		runs = append(runs, res)
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
			if f.Rule == "suspect" { // counted, never listed: the push only counts (ADR-0010)
				if !contains(suspect, f.Where) {
					suspect = append(suspect, f.Where)
				}
				continue
			}
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
			times()
			fmt.Fprintf(os.Stderr, "workline: %s\npush stopped; `git push --no-verify` skips workline\n", res.Summary)
			return 1
		case patched:
			times()
			stat, _ := exec.Command("git", "-C", root, "-c", "core.quotePath=off", "diff", "--stat").Output()
			fmt.Fprintf(os.Stderr, "workline: the documentalist updated docs in your working tree:\n%s", stat)
			fmt.Fprintln(os.Stderr, "push stopped: review them (git diff), commit them, and push again")
			return 1
		}
	}
	times()
	if len(suspect) > 0 {
		fmt.Fprintf(os.Stderr, "workline: docs suspect: %d made so by these commits — `workline docs` judges them\n", len(suspect))
	}
	return 0
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// docsCmd has the docs the commits not pushed yet made suspect judged, then
// each change reviewed on the terminal and those kept committed (ADR-0007).
// Without a terminal, the changes stay in the working tree for a person.
func docsCmd(args []string) int {
	fs := newFlags("docs")
	repo := fs.String("repo", ".", "repository")
	ai := fs.String("ai", "", "agent judging the docs (default: the project's, else yours, else none)")
	reviewOnly := fs.Bool("review", false, "no agent: review, doc by doc, the doc changes already in the working tree")
	since := fs.String("since", "", "judge the commits after this one (default: where the docs were last judged, else the last tag, else all)")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: not in a git repository")
		return 64
	}
	if *reviewOnly {
		// What an agent left in the working tree, or gardening run by hand.
		changed, _ := exec.Command("git", "-C", root, "-c", "core.quotePath=off", "diff", "--name-only", "--", "*.md").Output()
		docs := strings.Fields(string(changed))
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		switch {
		case len(docs) == 0:
			fmt.Fprintln(os.Stderr, "workline: no doc changed in the working tree")
			return 0
		case err != nil:
			fmt.Fprintln(os.Stderr, "workline: reviewing needs a person, on a terminal")
			return 64
		}
		defer tty.Close()
		if _, err := review.Docs(root, docs, bufio.NewReader(tty), tty, review.Open); err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		return 0
	}
	start, from := docsJudgedFrom(root, *since)
	rng := "HEAD"
	if start != "" {
		if head := revParse(root, "HEAD"); head == revParse(root, start) {
			fmt.Fprintf(os.Stderr, "workline: no commit since the docs were last judged (%s)\n", from)
			return 0
		}
		rng = start + "..HEAD"
	}
	var in *bufio.Reader
	out := io.Writer(os.Stderr)
	if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		defer tty.Close()
		in, out = bufio.NewReader(tty), tty
	}
	fmt.Fprintf(out, "workline: judging the docs made suspect by the commits since %s\n", from)
	if judgeDocs(root, rng, *ai, in, out) {
		// Judged, and nothing left for a person: the next run starts here.
		head := revParse(root, "HEAD")
		if err := exec.Command("git", "-C", root, "update-ref", docsJudgedRef, head).Run(); err == nil {
			fmt.Fprintf(out, "workline: docs judged up to %.7s (%s)\n", head, docsJudgedRef)
		}
	}
	return 0
}

// sampleCmd draws the week's sample of every role's acts (ADR-0046) and
// has the docs of it read (ADR-0014, step 4), reading the forge and
// writing nothing; with --apply, it writes what a sample found to the
// forge, with no agent.
func sampleCmd(args []string) int {
	fs := newFlags("sample")
	repo := fs.String("repo", ".", "repository")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles")
	week := fs.String("week", "", "the ISO week sampled, 2026-W40 (default: the last whole week)")
	since := fs.String("since", "", "sample what was vouched for after this commit, up to HEAD, instead of a week")
	judgeSpec := fs.String("judge", "", "the agent reading the sample: claude:opus, cmd:… (default: WORKLINE_JUDGE, then the documentalist's `sample.judge` setting)")
	out := fs.String("out", "", "also write the result to this file, for --apply")
	apply := fs.String("apply", "", "write the result in this file to the forge: the tracking issue, and a merge request putting back a false `checked`")
	forgeSpec := fs.String("forge", "", "the forge the acts are drawn from, and --apply writes to: github, gitlab, local, cmd:…, fake:<file>, none (default: the project's `forge` setting)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	var summaryFiles multi
	fs.Var(&summaryFiles, "summary", "also add a Markdown summary of the sample to this file, for a CI job's page (repeatable; a file named .html gets HTML)")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: not in a git repository")
		return 64
	}
	var res *sample.Result
	spec := *forgeSpec
	if spec == "" {
		if cfg, err := role.LoadProjectConfig(root); err == nil {
			spec = cfg.Forge
		}
	}
	f, err := forge.Open(spec, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	if *apply != "" {
		res = sample.Apply(*apply, root, f)
	} else {
		rolesDir, err := resolveRoles(*roles)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		res = sample.Draw(sample.Options{Repo: root, RolesDir: rolesDir, Week: *week, Since: *since, Judge: *judgeSpec, Now: time.Now(), Forge: f})
		if *out != "" {
			data, _ := json.MarshalIndent(res, "", "  ")
			if err := os.WriteFile(*out, append(data, '\n'), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "workline:", err)
				return 1
			}
		}
	}
	title := "sample"
	if *apply != "" {
		title = "sample --apply"
	}
	if err := wlreport.AppendSummary(summaryFiles, wlreport.SampleSummary(title, res)); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if *asJSON {
		data, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(data))
	} else {
		report(&engine.Result{Status: res.Status, Summary: res.Summary, Findings: res.Findings, AgentCalls: res.AgentCalls, Calls: res.Calls, Applied: res.Applied})
	}
	return exitFor(res.Status)
}

// docsJudgedRef is where `workline docs` records the commit the docs were
// last judged up to (ADR-0010): a repository pushed to main with no merge
// request judges from there, pushed or not.
const docsJudgedRef = "refs/workline/docs-judged"

// docsJudgedFrom is the commit `workline docs` judges after, and how to say
// it: the one asked for; else where the docs were last judged — the point
// HEAD shares with it, after a rebase or on another branch; else the last
// release (internal/release); else none, every commit.
func docsJudgedFrom(root, since string) (string, string) {
	if since != "" {
		return since, since
	}
	if ref := revParse(root, docsJudgedRef); ref != "" {
		if base, err := exec.Command("git", "-C", root, "merge-base", ref, "HEAD").Output(); err == nil {
			return strings.TrimSpace(string(base)), "they were last judged, " + docsJudgedRef
		}
	}
	if rs, err := release.Load(root); err == nil {
		if t, err := release.Last(root, rs.Tags); err == nil && t != "" {
			return t, "the last release, " + t
		}
	}
	return "", "the first commit"
}

// revParse is the commit rev names, or "" when it names none.
func revParse(root, rev string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "-q", "--verify", rev+"^{commit}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// judgeDocs runs the documentalist on the range, with the agent, then has
// the person review each doc it changed; in is nil without a terminal. It
// says whether the docs are judged: the run passed, and every change it
// made was reviewed, or there was none.
func judgeDocs(root, rng, ai string, in *bufio.Reader, out io.Writer) bool {
	if dirty, _ := exec.Command("git", "-C", root, "-c", "core.quotePath=off", "status", "--porcelain", "--", "*.md").Output(); len(dirty) > 0 {
		fmt.Fprintf(out, "workline: docs have changes not committed:\n%scommit or stash them first, so what the documentalist proposes is reviewed alone\n", dirty)
		return false
	}
	rolesDir, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		fmt.Fprintln(out, "workline:", err)
		return false
	}
	fmt.Fprintln(out, "workline: the documentalist judges the docs these commits made suspect…")
	res := engine.Run(engine.Options{Repo: root, RolesDir: rolesDir, Role: "documentalist", Event: "pre-push",
		AI: ai, DefaultAI: userDefaultAI(), Inputs: map[string]string{"range": rng}})
	for _, f := range res.Findings {
		// A suspect doc of the range, and what stopped the run: a fix the
		// judge refused, however often the agent was asked.
		if f.Rule == "suspect" && !strings.Contains(f.Message, "left for gardening") || f.Level == "block" || f.Where == "patch" || f.Rule == "left-out" {
			lines := strings.Split(strings.TrimSpace(f.Message), "\n")
			msg := lines[0]
			if last := lines[len(lines)-1]; len(lines) > 1 && strings.HasPrefix(last, "(") {
				msg += " " + last // what became of it: a person judges it, a later round
			}
			fmt.Fprintf(out, "  %s %s: %s\n", f.Rule, f.Where, msg)
		}
	}
	for _, n := range res.Notes {
		fmt.Fprintf(out, "  note from the agent: %s\n", strings.ReplaceAll(n, "\n", "\n    "))
	}
	// A doc of the range no agent judged yet — none was asked, or it waits
	// for a later round — keeps the ref where it is; one a person must read
	// (too large, judged without being vouched for) does not: it is said.
	waiting := 0
	for _, f := range res.Findings {
		if (f.Rule == "suspect" || f.Rule == "stale") && !strings.Contains(f.Message, "left for gardening") && !strings.Contains(f.Message, "a person") {
			waiting++
		}
	}
	if res.Status != verdict.Pass {
		// The run stopped at the round it was in: the docs of that round,
		// and those left for later rounds, were not judged.
		fmt.Fprintf(out, "workline: %s — %s; the docs it was judging, and those after them, are not judged yet: run `workline docs` again\n", res.Status, res.Summary)
	}
	changed, _ := exec.Command("git", "-C", root, "-c", "core.quotePath=off", "diff", "--name-only", "--", "*.md").Output()
	docs := strings.Fields(string(changed))
	if len(docs) == 0 {
		fmt.Fprintf(out, "workline: %s — no doc changed\n", res.Status)
		return res.Status == verdict.Pass && waiting == 0
	}
	if in == nil {
		fmt.Fprintf(out, "workline: %d doc(s) changed in your working tree, for a person to review (git diff), commit what is right, restore the rest\n", len(docs))
		return false
	}
	if _, err := review.Docs(root, docs, in, out, review.Open); err != nil {
		fmt.Fprintln(out, "workline:", err)
		return false
	}
	left, _ := exec.Command("git", "-C", root, "-c", "core.quotePath=off", "diff", "--name-only", "--", "*.md").Output()
	return res.Status == verdict.Pass && waiting == 0 && len(strings.TrimSpace(string(left))) == 0
}

func gitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// personal is the person's own config (~/.config/workline/config.yaml): what
// they want wherever they work, which no project can change.
type personal struct {
	AI          string `yaml:"ai"`           // the agent, when a project does not say
	ApprovePush *bool  `yaml:"approve-push"` // a person approves every push; off unless true (ADR-0011)
	// ApproveVia: where a push may be approved, among terminal, editor and
	// dialog (ADR-0008); all of them unless said.
	ApproveVia []string `yaml:"approve-push-via"`
}

func (p personal) approvePush() bool { return p.ApprovePush != nil && *p.ApprovePush }

func userConfig() personal {
	var c personal
	dir, err := os.UserConfigDir()
	if err != nil {
		return c
	}
	data, err := os.ReadFile(filepath.Join(dir, "workline", "config.yaml"))
	if err != nil {
		return c
	}
	yaml.Unmarshal(data, &c)
	return c
}

// userDefaultAI is the agent a person wants by default when a project does
// not say.
func userDefaultAI() string { return userConfig().AI }

func gateCmd(args []string) int {
	fs := newFlags("gate")
	repo := fs.String("repo", ".", "repository to check")
	base := fs.String("base", "", "count only the findings the change adds since the merge base of this revision and HEAD, as on a merge request (default: none, every finding counts)")
	kept := fs.String("gates-kept", "", "with --base: the folder of the gates' findings kept by commit, where the merge base's are read when kept with the same command, config and tool, rather than running the tools again")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	name, code, done := parseAfter(fs, args, "a gate's name")
	if done {
		return code
	}
	switch {
	case isSet(fs, "base") && strings.TrimSpace(*base) == "":
		// An unset variable in a CI job: never read as no base at all.
		return misuse(fs, "--base names no revision (an unset variable?): give one, or leave --base out to count every finding")
	case *kept != "" && *base == "":
		return misuse(fs, "--gates-kept reads the merge base's findings: it goes with --base")
	}
	abs, _ := filepath.Abs(*repo)
	var v *verdict.Verdict
	if *base != "" {
		// As on a merge request (ADR-0049): only what the change adds
		// since the merge base counts, the base's findings read as main
		// kept them or run again there.
		if *kept != "" {
			*kept, _ = filepath.Abs(*kept)
		}
		v, _ = gate.RunWith(abs, name, gate.Options{Since: *base, Kept: *kept})
	} else {
		v = gate.Run(abs, name)
	}
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

// testCmd runs the tester's checks that need no agent (ADR-0055): check,
// the red-first proof on the change since --base; measure, the curator's
// measures of the corpus on HEAD; and the runs with an agent: write,
// answer, curate (step 7: a few redundant tests removed, proven).
func testCmd(args []string) int {
	fs := newFlags("test")
	repo := fs.String("repo", ".", "repository to check")
	base := fs.String("base", "", "check: the revision the change is proven red on: its merge base with HEAD, as on a merge request")
	issue := fs.Int("issue", 0, "write: the ready issue whose scenarios the tests check")
	mr := fs.Int("merge-request", 0, "answer: the tester's Draft whose objections it answers, the first waiting")
	ai := fs.String("ai", "", "write, answer, curate: agent: none, claude, cmd:<command> (default: the project's `ai` setting, else none)")
	forgeSpec := fs.String("forge", "", "write, answer, curate: forge: github, gitlab, local, fake:<file> (default: the project's `forge` setting)")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	noApply := fs.Bool("no-apply", false, "write, answer, curate: judge, apply nothing; a job apart applies it with `workline apply <run>`, holding the write token and no agent's key")
	history := fs.String("history", "", "measure, curate: the curator's history file, read before and written after (curate: the folder it weighed); CI passes it from one run to the next as an artifact (missing: the first measure)")
	var summaryFiles multi
	fs.Var(&summaryFiles, "summary", "measure: also add the measures to this file, for a CI job's page (repeatable; a file named .html gets HTML)")
	sub, code, done := parseAfter(fs, args, "check, measure, curate, write or answer")
	if done {
		return code
	}
	switch {
	case sub == "write":
		if *issue <= 0 {
			return misuse(fs, "write needs --issue <n>, the ready issue to write the tests of")
		}
		return testRun(*repo, *roles, *ai, *forgeSpec, "issue", &forge.Target{Kind: "issue", ID: *issue}, *noApply, *asJSON)
	case sub == "answer":
		if *mr <= 0 {
			return misuse(fs, "answer needs --merge-request <n>, the tester's Draft whose objections it answers")
		}
		return testRun(*repo, *roles, *ai, *forgeSpec, "objection", &forge.Target{Kind: "merge-request", ID: *mr}, *noApply, *asJSON)
	case sub == "curate":
		// The measures it weighs, and where it keeps the folder it weighed.
		if *history != "" {
			h, _ := filepath.Abs(*history)
			_ = os.Setenv(tester.HistoryEnv, h)
		}
		return testRun(*repo, *roles, *ai, *forgeSpec, "curate", nil, *noApply, *asJSON)
	case sub == "measure":
		if *noApply {
			return misuse(fs, "measure applies nothing: --no-apply goes with write, answer or curate")
		}
	case sub != "check":
		return misuse(fs, fmt.Sprintf("unknown command %q: check, measure, curate, write or answer", sub))
	case *noApply:
		return misuse(fs, "check proves and applies nothing: --no-apply goes with write, answer or curate")
	case strings.TrimSpace(*base) == "":
		// An unset variable in a CI job: never read as no base at all.
		return misuse(fs, "--base names no revision (an unset variable?): give the base the change is proven red on")
	}
	abs, _ := filepath.Abs(*repo)
	rolesDir, err := resolveRoles(*roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	r, err := role.Load(rolesDir, "tester")
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	cfg, err := role.LoadProjectConfig(abs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	s, err := tester.SettingsFrom(r.MergedSettings(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if sub == "measure" {
		m := tester.Measure(abs, *history, s)
		res := &engine.Result{Status: m.Status, Summary: m.Summary, Findings: m.Findings, Applied: []string{}, Refused: []string{}}
		sum := wlreport.RoleSummary("test measure", "", res)
		sum.Lines = m.Lines
		if err := wlreport.AppendSummary(summaryFiles, sum); err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		if *asJSON {
			out, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(out))
		} else {
			report(res)
			for _, l := range m.Lines {
				fmt.Println(l)
			}
		}
		return exitFor(m.Status)
	}
	v := tester.Check(abs, *base, s)
	res := &engine.Result{Status: v.Status, Summary: v.Summary, Findings: v.Findings, Applied: []string{}, Refused: []string{}}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return exitFor(v.Status)
}

func developCmd(args []string) int {
	fs := newFlags("develop")
	repo := fs.String("repo", ".", "repository to work on")
	issue := fs.Int("issue", 0, "the ready issue to take, rather than the first in the backlog's order")
	ai := fs.String("ai", "", "agent: none, claude, cmd:<command> (default: the project's `ai` setting, else none); with none, the issue is named, nothing claimed")
	forgeSpec := fs.String("forge", "", "forge: github, gitlab, local, fake:<file> (default: the project's `forge` setting)")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	noApply := fs.Bool("no-apply", false, "judge, apply nothing; a job apart applies the claim with `workline apply <run>`, holding the write token and no agent's key")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	var target *forge.Target
	if *issue > 0 {
		target = &forge.Target{Kind: "issue", ID: *issue}
	}
	abs, _ := filepath.Abs(*repo)
	rolesDir, err := resolveRoles(*roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	absRoles, _ := filepath.Abs(rolesDir)
	res := engine.Run(engine.Options{Repo: abs, RolesDir: absRoles, Role: "developer", Event: "issue", AI: *ai,
		Forge: *forgeSpec, Target: target, NoApply: *noApply})
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return exitFor(res.Status)
}

// testRun runs the tester on one target (ADR-0055). On an issue, write
// (step 2): the agent writes a test a scenario, post proves them red on
// the base, and the engine commits them on workline/tester/issue-<n>,
// opened as a Draft. On its Draft, answer (step 3): the first objection
// waiting, kept or rewritten, its own commit pushed on the branch. With no
// target, curate (step 7): a folder's redundant tests removed, proven with
// no agent, on workline/tester/curate-<folder>.
func testRun(repo, roles, ai, forgeSpec, event string, target *forge.Target, noApply, asJSON bool) int {
	abs, _ := filepath.Abs(repo)
	rolesDir, err := resolveRoles(roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	absRoles, _ := filepath.Abs(rolesDir)
	res := engine.Run(engine.Options{Repo: abs, RolesDir: absRoles, Role: "tester", Event: event, AI: ai,
		Forge: forgeSpec, Target: target, NoApply: noApply, OpenMergeRequest: event == "issue" || event == "curate"})
	if asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return exitFor(res.Status)
}

func routeCmd(args []string) int {
	fs := newFlags("route")
	repo := fs.String("repo", ".", "repository to work on")
	ai := fs.String("ai", "", "agent for every role (default: the project's, else yours, else none)")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	forgeSpec := fs.String("forge", "", "forge: github, gitlab, none, fake:<file> (default: the project's `forge` setting)")
	target := fs.String("target", "", "issue:<n> or merge-request:<n>, where comments and labels go")
	branch := fs.String("branch", "", "the branch the merge request --target names comes from, when the forge is not to be asked (default: the forge says)")
	var scope multi
	fs.Var(&scope, "scope", "a path pattern the task is about (repeatable)")
	noApply := fs.Bool("no-apply", false, "judge every step, apply none; apply later with `workline apply` and the runs listed as pending")
	openMR := fs.Bool("open-merge-request", false, "put what the patches write on a branch of the role, and open a merge request for it (needs a forge)")
	sarifFile := fs.String("sarif", "", "also write every step's findings to this file as SARIF, for code scanning")
	cqFile := fs.String("code-quality", "", "also write every step's findings to this file as a GitLab Code Quality report")
	var summaryFiles multi
	fs.Var(&summaryFiles, "summary", "also add a Markdown summary of the line to this file, for a CI job's page (repeatable; a file named .html gets HTML)")
	verdictFile := fs.String("verdict", "", "also write the line's verdict on the head it judged (the range's end) to this file, for `workline follow --verdict` and the merge policy")
	verdictHead := fs.String("verdict-head", "", "with --verdict: the head the verdict is for, when the commit judged is a merge result (GitLab's merged results pipelines)")
	gatesOnly := fs.Bool("gates-only", false, "run the event's gate steps alone, each one: a CI job of their own, with no write token and no agent's (the result, --json, is what --gates-from and `apply --gates` read)")
	gatesFrom := fs.String("gates-from", "", "take each gate step's verdict from this file, the result a run with --gates-only printed, rather than running it; with --warn-only, the gates' findings for what triages them (the inspector), never their verdict")
	gatesKept := fs.String("gates-kept", "", "the folder of the gates' findings kept by commit: on main, the head's are written there, the next base; on a merge request, the merge base's are read there when kept with the same command, config and tool, rather than running the tools again")
	warnApart := fs.Bool("warn-apart", false, "leave what only warns to a run apart, which --warn-only makes: the reviewer's lenses that warn, on a merge request; the verdict waits only for what can block")
	warnOnly := fs.Bool("warn-only", false, "the run apart from the verdict: only what a line with --warn-apart left, posted when done; never the head's verdict")
	dryRun := fs.Bool("dry-run", false, "on release: what would hold the release if it were cut now, every step run with no agent, nothing applied, said as warnings, never a block (`workline apply --release-dry-run` posts it)")
	releaseWait := fs.Duration("release-wait", 0, "on the release, wait this long for the checks not done on the tip it is cut from, read again till they end, rather than block at once (e.g. 30m, a CI job's)")
	inputs := pairs{}
	fs.Var(inputs, "input", "input name=value, given to every step (repeatable)")
	event, code, done := parseAfter(fs, args, "an event")
	if done {
		return code
	}
	// A release's dry run (ADR-0050) asks no agent, applies and writes
	// nothing: it says what would hold the release cut from the branch
	// checked out.
	if *dryRun {
		switch {
		case event != "release":
			return misuse(fs, "--dry-run goes with the event release: what would hold the release")
		case *ai != "" && *ai != "none":
			return misuse(fs, "a release's dry run asks no agent: no --ai "+*ai)
		case *target != "" || *openMR:
			return misuse(fs, "a release's dry run writes nothing: no --target nor --open-merge-request (`workline apply --release-dry-run` posts what it found)")
		case *verdictFile != "" || *gatesOnly || *gatesFrom != "" || *warnApart || *warnOnly:
			return misuse(fs, "a release's dry run runs the release's whole line, giving no verdict: no --verdict, --gates-only, --gates-from, --warn-apart nor --warn-only")
		}
	}
	if *gatesOnly && *gatesFrom != "" {
		return misuse(fs, "--gates-only runs the gates, --gates-from reads them: one or the other")
	}
	flagged := map[bool]string{true: line.WarnApart, false: line.WarnOnly}[*warnApart]
	switch given, told := inputs[line.WarnInput]; {
	case *warnApart && *warnOnly:
		return misuse(fs, "--warn-apart leaves what only warns, --warn-only runs it alone: one or the other")
	case (*warnApart || *warnOnly) && told && given != flagged:
		return misuse(fs, fmt.Sprintf("--warn-%s and --input %s=%s say two runs: one or the other", flagged, line.WarnInput, given))
	case *warnApart || *warnOnly:
		inputs[line.WarnInput] = flagged
	}
	switch w := inputs[line.WarnInput]; {
	case w != "" && w != line.WarnApart && w != line.WarnOnly:
		return misuse(fs, fmt.Sprintf("--input %s=%s: %s or %s", line.WarnInput, w, line.WarnApart, line.WarnOnly))
	case w == line.WarnOnly && *gatesOnly:
		return misuse(fs, "--warn-only runs no gate: no --gates-only (--gates-from gives the gates' findings to what triages them)")
	case w == line.WarnOnly && *verdictFile != "":
		return misuse(fs, "--warn-only gives no verdict: what only warns never holds the head's (--verdict goes with the line's own run)")
	}
	// A merge queue's or train's run (ADR-0050) guards the merged result:
	// the head's verdict and the person's yes stay the merge request's
	// (ADR-0042), never the queue's to write.
	switch q := inputs[line.QueueInput]; {
	case q != "" && q != "true" && q != "false":
		return misuse(fs, fmt.Sprintf("--input %s=%s: true or false", line.QueueInput, q))
	case q == "true" && *verdictFile != "":
		return misuse(fs, "a queue's run (--input queue=true) gives no verdict on the head: the head's stays its merge request's (--verdict goes with that line's run)")
	case q == "true" && inputs[line.WarnInput] == line.WarnOnly:
		return misuse(fs, "--warn-only reads what only warns, a queue's run (--input queue=true) only the gates: one or the other")
	}
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
	if *gatesFrom != "" {
		*gatesFrom, _ = filepath.Abs(*gatesFrom)
	}
	if *gatesKept != "" {
		*gatesKept, _ = filepath.Abs(*gatesKept)
	}
	res := line.RunWith(event, engine.Options{Repo: abs, RolesDir: rolesDir, AI: *ai, DefaultAI: userDefaultAI(), Inputs: inputs,
		Forge: *forgeSpec, Target: t, Branch: *branch, Scope: scope, NoApply: *noApply, OpenMergeRequest: *openMR,
		DryRun: *dryRun, ReleaseWait: *releaseWait},
		&line.Gates{Only: *gatesOnly, From: *gatesFrom, Kept: *gatesKept})
	if err := wlreport.Write(abs, event, wlreport.FromLine(res), *sarifFile, *cqFile); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if err := wlreport.AppendSummary(summaryFiles, wlreport.LineSummary("route "+event, res)); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if *verdictFile != "" {
		rng := inputs["range"]
		if *verdictHead != "" {
			rng = *verdictHead
		}
		if err := writeVerdict(abs, *verdictFile, rng, res); err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
	}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Fprintf(os.Stderr, "%s — %s\n", res.Status, res.Summary)
		for _, s := range res.Steps {
			fmt.Fprintf(os.Stderr, "  %-28s %s\n", s.Name, s.Status)
		}
		if t := res.Times(); t != "" {
			fmt.Fprintf(os.Stderr, "  seconds: %s\n", t)
		}
		if over := res.OverBudget(); over != "" {
			fmt.Fprintf(os.Stderr, "  over its budget: %s\n", over)
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

// writeVerdict writes the line's verdict on the head it judged — the
// range's end, else HEAD — for the merge policy (ADR-0042).
func writeVerdict(repo, file, rng string, res *line.Result) error {
	head := "HEAD"
	if rng != "" {
		head = gitrange.Head(rng)
	}
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", head+"^{commit}").Output()
	switch {
	case err == nil:
		head = strings.TrimSpace(string(out))
	case len(head) != 40 || strings.Trim(head, "0123456789abcdef") != "":
		return fmt.Errorf("--verdict: the head judged, %s, is no commit here", head)
	} // a full id not fetched here: the source head of a merge result
	v := engine.HeadVerdict{Head: head, Status: res.Status, Findings: res.Findings}
	for _, s := range res.Steps {
		v.Steps = append(v.Steps, engine.StepStatus{Name: s.Name, Status: s.Status})
	}
	return engine.WriteVerdict(file, v)
}

func itemCmd(args []string) int {
	fs := newFlags("item")
	repo := fs.String("repo", ".", "repository holding .workline/work/")
	forgeSpec := fs.String("forge", "", "read the item from this forge instead of .workline/work/")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if len(args) > 0 && args[0] == "ready" {
		args = args[1:]
	} else if len(args) == 0 || !strings.HasPrefix(args[0], "-") {
		return misuse(fs, "the only action is ready")
	}
	item, code, done := parseAfter(fs, args, "an item id")
	if done {
		return code
	}
	abs, _ := filepath.Abs(*repo)
	var v *verdict.Verdict
	if *forgeSpec != "" && *forgeSpec != "none" {
		// A forge's item is a number, whole: never "abc" read as 0, nor
		// "12abc" as 12.
		id, err := strconv.Atoi(strings.TrimPrefix(item, "#"))
		if err != nil || id < 1 {
			return misuse(fs, fmt.Sprintf("%q is not an item number", item))
		}
		f, err := forge.Open(*forgeSpec, abs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 64
		}
		v = work.ReadyOnForge(f, id)
	} else {
		v = work.Ready(abs, item)
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

// issuesCmd reads the local forge (`forge: local`): the issues and merge
// requests kept in the clone, listed, or one shown whole. It writes nothing.
func issuesCmd(args []string) int {
	what, id := "list", ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		what, args = args[0], args[1:]
	}
	if (what == "show" || what == "import") && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id, args = args[0], args[1:]
	}
	fs := newFlags("issues")
	repo := fs.String("repo", ".", "repository")
	apply := fs.Bool("apply", false, "import: open the issues; without it, only say which")
	forgeSpec := fs.String("forge", "", "import: github, gitlab, local, cmd:<command> (default: the project's `forge` setting)")
	ai := fs.String("ai", "", "import: the agent that reads the file (default: the project's, then yours)")
	share := fs.Int("lines", 300, "import: lines of the file read in one call")
	roles := fs.String("roles", "", "import: folder of roles (default: the shipped ones)")
	asJSON := fs.Bool("json", false, "import: print the result as JSON")
	var summaryFiles multi
	fs.Var(&summaryFiles, "summary", "import: also add a Markdown summary, the map with it, to this file, for a CI job's page (repeatable; a file named .html gets HTML)")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: not in a git repository")
		return 64
	}
	if what == "import" {
		return importIssues(root, id, *forgeSpec, *ai, *roles, *share, *apply, *asJSON, summaryFiles)
	}
	l := &forge.Local{Repo: root}
	switch {
	case what == "list":
		n := 0
		for _, kind := range []string{"issue", "merge-request"} {
			items, err := l.Items(kind)
			if err != nil {
				fmt.Fprintln(os.Stderr, "workline:", err)
				return 1
			}
			for _, it := range items {
				n++
				sign, more := "#", ""
				if kind == "merge-request" {
					sign, more = "!", fmt.Sprintf("  (%s into %s)", it.Branch, it.Base)
				}
				labels := ""
				if len(it.Labels) > 0 {
					labels = "  [" + strings.Join(it.Labels, ", ") + "]"
				}
				fmt.Printf("%s%-4d %-7s %s%s%s\n", sign, it.ID, l.StateOf(it), it.Title, labels, more)
			}
		}
		if n == 0 {
			fmt.Fprintln(os.Stderr, "workline: no issue nor merge request in this clone's local forge (forge: local)")
		}
		return 0
	case what == "show" && id != "":
		kind := "issue"
		if rest, ok := strings.CutPrefix(id, "!"); ok {
			kind, id = "merge-request", rest
		}
		var n int
		if _, err := fmt.Sscan(strings.TrimPrefix(id, "#"), &n); err != nil {
			fmt.Fprintln(os.Stderr, "workline: show <n> for an issue, show !<n> for a merge request")
			return 64
		}
		it, err := l.Item(kind, n)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		fmt.Printf("%s — %s", it.Title, l.StateOf(it))
		if len(it.Labels) > 0 {
			fmt.Printf("  [%s]", strings.Join(it.Labels, ", "))
		}
		if it.Branch != "" {
			fmt.Printf("  (%s into %s)", it.Branch, it.Base)
		}
		fmt.Printf("\n\n%s\n", it.Body)
		for _, c := range it.Comments {
			fmt.Printf("\n---\n\n%s\n", c)
		}
		return 0
	}
	fmt.Fprintln(os.Stderr, "usage: workline issues [list] [--repo <dir>] | workline issues show <n>|!<n> [--repo <dir>]")
	return 64
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
	fs := newFlags("apply")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	lineFile := fs.String("line", "", "apply the runs `workline route --no-apply --json` listed as pending in this file")
	var summaryFiles multi
	fs.Var(&summaryFiles, "summary", "also add a Markdown summary of what was applied to this file, for a CI job's page (repeatable; a file named .html gets HTML)")
	fromMR := fs.Int("from-merge-request", 0, "the runs come from this merge request's own CI run, which its code can change: only their proposals are read, found by name in the runs folder; the repository, the roles and the forge are this job's, and only what that merge request's line may do is applied, to it")
	head := fs.String("head", "", "with --from-merge-request: the commit its run judged; nothing is applied once its branch moved on")
	repo := fs.String("repo", ".", "with --from-merge-request: this job's checkout of the base, whose settings are read")
	forgeSpec := fs.String("forge", "", "with --from-merge-request: github, gitlab, cmd:<command> (default: the project's `forge` setting)")
	gatesFile := fs.String("gates", "", "with --from-merge-request: show on it what its gates found the change adds, from the result `route --gates-only` printed in its own run: a check's annotations and one summary comment")
	mainFile := fs.String("main", "", "keep what main's run after a merge found, the result `route main --json` printed, in one issue, edited when what it lists changes; with --forge, --repo")
	dryFile := fs.String("release-dry-run", "", "post what a release's dry run found, the result `route release --dry-run --json` printed, in the default branch's issue, edited when what it lists changes; with --forge, --repo")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	if *dryFile != "" {
		if *mainFile != "" || *lineFile != "" || *gatesFile != "" || *fromMR != 0 || len(dirs) > 0 {
			fmt.Fprintln(os.Stderr, "workline: --release-dry-run posts what a release's dry run found, and applies nothing else: no run, --main, --line, --gates nor --from-merge-request")
			return 64
		}
		return keepDryRun(*dryFile, *repo, *forgeSpec, summaryFiles, *asJSON)
	}
	if *mainFile != "" {
		if *lineFile != "" || *gatesFile != "" || *fromMR != 0 || len(dirs) > 0 {
			fmt.Fprintln(os.Stderr, "workline: --main keeps main's findings in its issue, and applies nothing else: no run, --line, --gates nor --from-merge-request")
			return 64
		}
		return keepMain(*mainFile, *repo, *forgeSpec, summaryFiles, *asJSON)
	}
	if *gatesFile != "" && *fromMR == 0 {
		fmt.Fprintln(os.Stderr, "workline: --gates goes with --from-merge-request: its findings are shown on that merge request")
		return 64
	}
	resume := engine.Resume
	var from engine.FromMergeRequest
	if *fromMR != 0 {
		var err error
		if from, err = fromMergeRequest(*fromMR, *head, *repo, *forgeSpec); err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 64
		}
		resume = func(listed string) *engine.Result { return engine.ResumeFrom(listed, from) }
	} else if *head != "" {
		fmt.Fprintln(os.Stderr, "workline: --head goes with --from-merge-request")
		return 64
	}
	// What the line's documentalist found on a merge request, shown beside
	// its lines on the head judged, a check's annotations (#412); what its
	// gates found the change adds, so too, with a summary comment (ADR-0049).
	var noted []verdict.Finding
	if *gatesFile != "" {
		shown, err := showGates(from, *gatesFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		noted = append(noted, shown...)
	}
	if *lineFile == "" && *gatesFile != "" && len(dirs) == 0 {
		return nothingToApply(noted, summaryFiles, *asJSON)
	}
	// The runs named here, never a line's result's: one with
	// --from-merge-request is found by name (ResumeFrom), its refusal said.
	var named []string
	if *fromMR == 0 {
		named = append(named, dirs...)
	}
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
		if *fromMR != 0 {
			noted = append(engine.Annotate(from, func(read func(string) ([]byte, error)) ([]forge.Annotation, int) {
				var found []wlreport.Item
				for _, i := range wlreport.FromLine(&l) {
					if i.Role == "documentalist" {
						found = append(found, i)
					}
				}
				return wlreport.Annotations(read, found)
			}), noted...)
		}
		dirs = append(dirs, l.Pending...)
		if len(dirs) == 0 {
			return nothingToApply(noted, summaryFiles, *asJSON)
		}
	}
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: workline apply <run-dir>... | --line <route result> [--json] [--summary <file>]")
		return 64
	}
	for _, d := range named {
		if !engine.RunName(filepath.Base(d)) {
			fmt.Fprintf(os.Stderr, "workline: %q names no run's folder (<date>T<time>.<ns>-<role>): nothing applied\n", d)
			return 64
		}
	}
	res := resume(dirs[0])
	stopped := len(dirs) - 1 // the last run applied
	for i, d := range dirs[1:] {
		if res.Status != verdict.Pass {
			stopped = i
			break
		}
		next := resume(d)
		res.Status, res.Summary, res.RunDir = next.Status, next.Summary, next.RunDir
		res.Applied = append(res.Applied, next.Applied...)
		res.Refused = append(res.Refused, next.Refused...)
		res.Findings = append(res.Findings, next.Findings...)
		res.Handoffs = append(res.Handoffs, next.Handoffs...)
		if next.Issues != "" {
			res.Issues, res.Waiting = next.Issues, next.Waiting
		}
	}
	res.Findings = append(noted, res.Findings...)
	// The runs after the one that stopped are not applied: said, with the
	// command that resumes them all, never left out of the job's page.
	if left := dirs[stopped+1:]; len(left) > 0 {
		runs := "runs"
		if len(left) == 1 {
			runs = "run"
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "runs-not-applied", Where: dirs[stopped], Level: "error",
			Message: fmt.Sprintf("%d %s after it not applied; once it is fixed, resume with: workline apply %s",
				len(left), runs, strings.Join(dirs[stopped:], " "))})
	}
	if err := wlreport.AppendSummary(summaryFiles, wlreport.RoleSummary("apply", "", res)); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if *asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return exitFor(res.Status)
}

// keepMain keeps what main's run after a merge found in one issue
// (ADR-0050): opened when there is something to list, edited when what
// it lists changes, left as it is when nothing is new or gone. The run's
// result is data its job wrote: what it says is cut and defused.
func keepMain(file, repo, forgeSpec string, summaryFiles []string, asJSON bool) int {
	fail := func(err error) int {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return fail(err)
	}
	var r line.Result
	if err := json.Unmarshal(data, &r); err != nil {
		return fail(fmt.Errorf("--main %s: %v", file, err))
	}
	if r.Event != line.MainEvent {
		return fail(fmt.Errorf("--main %s: not the result of `workline route main`", file))
	}
	root, err := gitRoot(repo)
	if err != nil {
		return fail(errors.New("not in a git repository"))
	}
	if forgeSpec == "" {
		cfg, err := role.LoadProjectConfig(root)
		if err != nil {
			return fail(err)
		}
		forgeSpec = cfg.Forge
	}
	f, err := forge.Open(forgeSpec, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	b, ok := f.(forge.Backlog)
	if f == nil || !ok {
		fmt.Fprintln(os.Stderr, "workline: --main needs a forge (--forge github, gitlab…) to keep its issue")
		return 64
	}
	open, err := b.Issues()
	if err != nil {
		return fail(err)
	}
	now, id := "", 0
	for _, is := range open {
		if is.Title == wlreport.MainTitle && !is.Closed {
			now, id = is.Body, is.ID
			break
		}
	}
	body, changed, found := wlreport.MainIssue(&r, now)
	res := &engine.Result{Status: verdict.Pass, Applied: []string{}, Refused: []string{}}
	switch {
	case !changed && id != 0:
		res.Summary = fmt.Sprintf("nothing new nor gone on the default branch: issue #%d left as it is", id)
	case !changed:
		res.Summary = "nothing found on the default branch: no issue opened"
	default:
		n, err := f.KeepIssue(wlreport.MainTitle, backlog.Inert(body), found)
		if err != nil {
			return fail(err)
		}
		res.Summary = fmt.Sprintf("issue #%d kept: what the default branch's run found changed", n)
		res.Applied = append(res.Applied, fmt.Sprintf("issue #%d", n))
		id = n
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "main-issue", Message: res.Summary})
	// Red, each open merge request into it is told, once a red spell: a
	// check red there for it is the base's (ADR-0051, cut 2).
	if since := wlreport.RedSince(body); since != "" && id != 0 && r.Main != nil {
		n, err := noteBaseRed(f, root, r.Main.Head, since, id)
		if err != nil {
			return fail(err)
		}
		if n > 0 {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "base-red", Message: fmt.Sprintf("the default branch red since %s: the open merge requests into it told, once each: %d", since, n)})
		}
	}
	if err := wlreport.AppendSummary(summaryFiles, wlreport.RoleSummary("apply --main", "", res)); err != nil {
		return fail(err)
	}
	if asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return 0
}

// noteBaseRed says, on each open merge request whose base holds head, that
// the default branch's run is red since a commit, from merges each green
// alone: a check red there for it is not the merge request's own. One
// comment each, by the commit main turned red at. It returns how many it
// looked at.
func noteBaseRed(f forge.Forge, repo, head, since string, issue int) (int, error) {
	open, err := f.ListMergeRequests()
	if err != nil {
		return 0, err
	}
	holds, n := map[string]bool{}, 0
	for _, mr := range open {
		held, read := holds[mr.Base]
		if !read {
			for _, ref := range []string{"refs/remotes/origin/" + mr.Base, "refs/heads/" + mr.Base} {
				if exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", head, ref).Run() == nil {
					held = true
					break
				}
			}
			holds[mr.Base] = held
		}
		if !held {
			continue
		}
		body := fmt.Sprintf("**Base red, not yours**: the default branch's run is red since `%.7s`, from merges each green alone (#%d). A check red here for that is the base's, not this merge request's: it clears once the base is fixed.", since, issue)
		if err := f.Comment(forge.Target{Kind: "merge-request", ID: mr.ID}, body, forge.Marker("base-red="+since)); err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}

// keepDryRun posts what a release's dry run found (ADR-0050) in a part of
// the default branch's issue, put in, edited, or taken out when nothing
// holds. The run's result is data its job wrote: what it says is cut and
// defused.
func keepDryRun(file, repo, forgeSpec string, summaryFiles []string, asJSON bool) int {
	fail := func(err error) int {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return fail(err)
	}
	var r line.Result
	if err := json.Unmarshal(data, &r); err != nil {
		return fail(fmt.Errorf("--release-dry-run %s: %v", file, err))
	}
	if r.DryRun == nil {
		return fail(fmt.Errorf("--release-dry-run %s: not the result of `workline route release --dry-run`", file))
	}
	d := r.DryRun
	root, err := gitRoot(repo)
	if err != nil {
		return fail(errors.New("not in a git repository"))
	}
	if forgeSpec == "" {
		cfg, err := role.LoadProjectConfig(root)
		if err != nil {
			return fail(err)
		}
		forgeSpec = cfg.Forge
	}
	f, err := forge.Open(forgeSpec, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	b, ok := f.(forge.Backlog)
	if f == nil || !ok {
		fmt.Fprintln(os.Stderr, "workline: --release-dry-run needs a forge (--forge github, gitlab…) to post what it found")
		return 64
	}
	res := &engine.Result{Status: verdict.Pass, Applied: []string{}, Refused: []string{}}
	open, err := b.Issues()
	if err != nil {
		return fail(err)
	}
	now, id := "", 0
	for _, is := range open {
		if is.Title == wlreport.MainTitle && !is.Closed {
			now, id = is.Body, is.ID
			break
		}
	}
	var said []string
	if body, changed := wlreport.WithDryRun(now, d); changed && (id != 0 || len(d.Holds) > 0) {
		n, err := f.KeepIssue(wlreport.MainTitle, backlog.Inert(body), true)
		if err != nil {
			return fail(err)
		}
		said = append(said, fmt.Sprintf("issue #%d kept: what would hold the release changed", n))
		res.Applied = append(res.Applied, fmt.Sprintf("issue #%d", n))
	} else if id != 0 {
		said = append(said, fmt.Sprintf("nothing new nor gone for the release: issue #%d left as it is", id))
	} else {
		said = append(said, "nothing would hold the release: no issue opened")
	}
	res.Summary = strings.Join(said, "; ")
	res.Findings = append(res.Findings, verdict.Finding{Rule: "release-dry-run", Message: res.Summary})
	if err := wlreport.AppendSummary(summaryFiles, wlreport.RoleSummary("apply --release-dry-run", "", res)); err != nil {
		return fail(err)
	}
	if asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return 0
}

// nothingToApply ends an apply the line proposed nothing to, its findings
// said.
func nothingToApply(noted []verdict.Finding, summaryFiles []string, asJSON bool) int {
	res := &engine.Result{Status: verdict.Pass, Summary: "nothing to apply: the line proposed nothing",
		Applied: []string{}, Refused: []string{}, Findings: noted}
	if err := wlreport.AppendSummary(summaryFiles, wlreport.Summary{Title: "apply", Status: verdict.Pass, NothingToApply: true, Findings: noted}); err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	if asJSON {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		report(res)
	}
	return 0
}

// showGates shows on the merge request what its gates found the change
// adds, from the result a run of the gates alone printed in its own CI
// run: data that run's code could have written, capped and defused. The
// summary comment lists as many rules as the reviewer's `findings-max`
// lets it list findings (ADR-0049).
func showGates(from engine.FromMergeRequest, file string) ([]verdict.Finding, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var r line.Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("--gates %s: %v", file, err)
	}
	g := wlreport.Gates(&r, from.Head)
	entries := 10 // the reviewer's default
	if cfg, err := role.LoadProjectConfig(from.Repo); err == nil {
		// A whole number from 1, however it was read: YAML's int, JSON's
		// float; any other value leaves the default.
		switch n := cfg.Roles["reviewer"].Settings["findings-max"].(type) {
		case int:
			if n >= 1 {
				entries = n
			}
		case int64:
			if n >= 1 {
				entries = int(n)
			}
		case float64:
			if n >= 1 && n == float64(int(n)) {
				entries = int(n)
			}
		}
	}
	return engine.ShowGates(from, engine.GatesToShow{Notes: g.Notes, Title: g.Title(), Summary: g.Summary(),
		Comment: func(url string) string { return g.Comment(url, entries) }, Worth: g.Worth(), New: g.New}), nil
}

// fromMergeRequest is what `apply --from-merge-request` takes from this
// job: its checkout of the base, its engine's roles, its forge.
func fromMergeRequest(id int, head, repo, forgeSpec string) (engine.FromMergeRequest, error) {
	if head == "" {
		return engine.FromMergeRequest{}, errors.New("--from-merge-request needs --head, the commit its run judged")
	}
	root, err := gitRoot(repo)
	if err != nil {
		return engine.FromMergeRequest{}, errors.New("not in a git repository")
	}
	if forgeSpec == "" {
		cfg, err := role.LoadProjectConfig(root)
		if err != nil {
			return engine.FromMergeRequest{}, err
		}
		forgeSpec = cfg.Forge
	}
	roles, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		return engine.FromMergeRequest{}, err
	}
	return engine.FromMergeRequest{Repo: root, RolesDir: roles, Forge: forgeSpec, ID: id, Head: head}, nil
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

// setupCmd sets workline up on this machine, asking what to enable, then
// shows the doctor's report. Run again, it reconfigures.
func setupCmd(args []string) int {
	fs := newFlags("setup")
	hooksOn := fs.String("hooks", "", "yes or no: the global git hooks (default: ask)")
	ai := fs.String("ai", "", "your agent: none, claude, claude:<model>@<effort>, cmd:<command> (default: ask)")
	install := fs.String("install", "", "the tools to install, comma-separated, all, or none (default: ask)")
	yes := fs.Bool("yes", false, "take the default of every question not answered by a flag")
	asJSON := fs.Bool("json", false, "print the doctor's report, at the end, as JSON")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	var a setup.Answers
	a.Hooks, a.AI, a.Yes = *hooksOn, *ai, *yes
	if *install != "" {
		a.Install = []string{}
		if *install != "none" {
			a.Install = strings.Split(*install, ",")
		}
	}
	bin, err := os.Executable()
	if err == nil {
		bin, err = filepath.EvalSymlinks(bin)
	}
	var paths hooks.Paths
	if err == nil {
		paths, err = hooks.DefaultPaths()
	}
	rolesDir := ""
	if err == nil {
		rolesDir, err = resolveRoles(os.Getenv("WORKLINE_ROLES"))
	}
	cfgDir, _ := os.UserConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	err = setup.Run(a, setup.Machine{
		In: os.Stdin, Out: os.Stderr, Interactive: terminal(os.Stdin),
		Bin: bin, Hooks: paths, UserConfig: filepath.Join(cfgDir, "workline", "config.yaml"),
		Tools: doctor.UsedTools(rolesDir), Install: setup.Shell(os.Stderr),
	})
	if err != nil {
		if *asJSON {
			out, _ := json.MarshalIndent(&engine.Result{Status: verdict.Block, Summary: err.Error(), Applied: []string{}, Refused: []string{},
				Findings: []verdict.Finding{{Rule: "setup-stopped", Message: err.Error()}}}, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Fprintln(os.Stderr, "workline:", err)
		}
		if strings.HasPrefix(err.Error(), "no terminal") {
			return 64
		}
		return 1
	}
	if *asJSON {
		return doctorCmd([]string{"--json"})
	}
	fmt.Println()
	return doctorCmd(nil)
}

// terminal says whether f is a terminal a person types in: a character
// device, but not /dev/null, which is one too.
func terminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(st, null)
}

// doctorCmd says what is set up on this machine and in the repository, and
// the command that sets up what is missing. It exits 1 only on an error:
// something missing that workline was set up to use.
func doctorCmd(args []string) int {
	fs := newFlags("doctor")
	repo := fs.String("repo", ".", "repository to look at; outside one, the machine only")
	asJSON := fs.Bool("json", false, "print the report as JSON")
	forgeSpec := fs.String("forge", "", "the forge whose rules are read: whether the default branch requires every conversation resolved (default: the project's `forge` setting; github when the repository's GitHub workflows run workline)")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	rolesDir, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	root, _ := gitRoot(*repo)
	r := doctor.Run(doctor.Options{Repo: root, RolesDir: rolesDir, AI: os.Getenv("WORKLINE_AI"), UserAI: userDefaultAI(),
		NoPushApproval: !userConfig().approvePush(), Forge: *forgeSpec})
	if *asJSON {
		out, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(out))
	} else {
		mark := map[string]string{doctor.OK: "✓", doctor.Note: "·", doctor.Warn: "!", doctor.Error: "✗"}
		area := ""
		for _, c := range r.Checks {
			if c.Area != area {
				area = c.Area
				title := "This machine"
				if area == "repository" {
					title = "This repository (" + root + ")"
				}
				fmt.Printf("%s\n", title)
			}
			fmt.Printf("  %s %s\n", mark[c.Level], c.Message)
			if c.Fix != "" {
				fmt.Printf("      %s\n", c.Fix)
			}
		}
		fmt.Printf("\n%s\n", r.Summary)
	}
	if r.Status != verdict.Pass {
		return 1
	}
	return 0
}

// initCmd adopts a repository: pre-push is routed to the committer and the
// documentalist, and the docs that say nothing of their sources are put
// before the agent, whose proposals land in the working tree for a person to
// review. Running it again only does what is left.
func initCmd(args []string) int {
	fs := newFlags("init")
	repo := fs.String("repo", ".", "repository to adopt")
	ai := fs.String("ai", "", "agent proposing each doc's sources (default: the project's, else yours, else none)")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	review := fs.Bool("review", false, "also have the reviewer review each merge request (ADR-0020): added to the project's merge-request line")
	humanPO := fs.String("human-po", "", "yes when a person is the project's Product Owner: the product owner role proposes what sets direction (autonomy: cautious, ADR-0026); asked on a terminal")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: not in a git repository")
		return 64
	}
	rolesDir, err := resolveRoles(*roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	steps, changed, err := routing.RoutePrePush(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	config := "pre-push runs " + strings.Join(steps, ", ") + ", as the project set it"
	if changed {
		config = "pre-push now runs " + strings.Join(steps, ", ") + " (.workline/config.yaml)"
	}
	if *review {
		steps, changed, err := routing.AddReviewer(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		if changed {
			config += "; merge requests now run " + strings.Join(steps, ", ")
		} else {
			config += "; merge requests run " + strings.Join(steps, ", ") + ", the reviewer among them already"
		}
	}
	if *humanPO == "" && !*asJSON && terminal(os.Stdin) {
		fmt.Fprint(os.Stderr, "Is there a human Product Owner on this project? [no] ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		*humanPO = strings.TrimSpace(line)
	}
	switch strings.ToLower(*humanPO) {
	case "", "no", "n", "false":
	case "yes", "y", "true":
		// A person sets the direction: the role proposes it (ADR-0026).
		level, changed, err := routing.SetAutonomy(root, "cautious")
		if err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		if changed {
			config += "; the product owner proposes what sets direction, a person being the Product Owner (autonomy: cautious)"
		} else {
			config += "; the product owner's autonomy is " + level + ", as the project set it"
		}
	default:
		fmt.Fprintf(os.Stderr, "workline: --human-po %q: yes or no\n", *humanPO)
		return 64
	}
	res := engine.Run(engine.Options{Repo: root, RolesDir: rolesDir, Role: "documentalist", Event: "init", AI: *ai, DefaultAI: userDefaultAI()})
	if *asJSON {
		out, _ := json.MarshalIndent(struct {
			*engine.Result
			Config string `json:"config"`
		}{res, config}, "", "  ")
		fmt.Println(string(out))
		return exitFor(res.Status)
	}
	// Only what adoption is about: the rest is what any run reports, and
	// `workline doctor` or the next push says it.
	fmt.Fprintln(os.Stderr, "workline init:", config)
	fmt.Fprintf(os.Stderr, "workline init: %s — %s\n", res.Status, res.Summary)
	suspect := map[string]bool{}
	for _, f := range res.Findings {
		switch {
		case f.Rule == "suspect":
			suspect[f.Where] = true
		case f.Rule == "no-sources" || res.Status != verdict.Pass && f.Level == "" && !prePushQuiet[f.Rule]:
			fmt.Fprintf(os.Stderr, "  %s %s: %s\n", f.Rule, f.Where, f.Message)
		}
	}
	for _, n := range res.Notes {
		fmt.Fprintf(os.Stderr, "  note from the agent: %s\n", strings.ReplaceAll(n, "\n", "\n    "))
	}
	if stat, _ := exec.Command("git", "-C", root, "-c", "core.quotePath=off", "status", "--short").Output(); len(stat) > 0 {
		fmt.Fprintf(os.Stderr, "\nIn your working tree:\n%s", stat)
		fmt.Fprintln(os.Stderr, "Review it (git diff): commit what is right, restore what is not (git restore <file>).")
	}
	if len(suspect) > 0 {
		fmt.Fprintf(os.Stderr, "%d docs are suspect already: code they describe changed since they were last edited. The next push lists them; `workline docs` has them judged, or a merge request.\n", len(suspect))
	}
	return exitFor(res.Status)
}

// importIssues moves a file — a roadmap, a backlog — to the forge's
// issues: the product owner reads it a share at a time and proposes an
// issue for each item still to do, its text quoted; the engine checks each
// quote and opens it once (docs/spec/backlog-acts.md, "Importing a file").
// Without apply, it only says what it would open.
func importIssues(root, file, forgeSpec, ai, roles string, share int, apply, asJSON bool, summaryFiles []string) int {
	out := os.Stdout
	if asJSON {
		out = os.Stderr // the text goes aside, the result alone on stdout
	}
	total := &engine.Result{Status: verdict.Pass, Applied: []string{}, Refused: []string{}}
	var coverage *backlog.Coverage // the map: each item of the file to its issue, or why none
	finish := func(code int) int {
		total.ToApply = len(total.Pending) > 0
		s := wlreport.RoleSummary("issues import", "product-owner", total)
		if coverage != nil {
			s.Map = coverage.Entries()
		}
		if err := wlreport.AppendSummary(summaryFiles, s); err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 1
		}
		if asJSON {
			data, _ := json.MarshalIndent(struct {
				*engine.Result
				Coverage *backlog.Coverage `json:"coverage,omitempty"`
			}{total, coverage}, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Fprintln(out, total.Summary)
			if len(total.Pending) > 0 {
				fmt.Fprintf(out, "  to apply: workline apply %s\n", strings.Join(total.Pending, " "))
			}
		}
		return code
	}
	if file == "" || share < 20 {
		fmt.Fprintln(os.Stderr, "usage: workline issues import <file> [--apply] [--ai <agent>] [--forge <forge>] [--lines <n>, 20 or more] [--json]")
		return 64
	}
	data, err := exec.Command("git", "-C", root, "show", "HEAD:"+file).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "workline: %s is not in the last commit: an import reads what is committed\n", file)
		return 64
	}
	rolesDir, err := resolveRoles(roles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	absRoles, _ := filepath.Abs(rolesDir)
	n := strings.Count(string(data), "\n")
	if !strings.HasSuffix(string(data), "\n") {
		n++
	}
	// The forge's issues, open and closed, before and after: the map tells
	// an issue this import opened from one that held the item already.
	var backlogOf forge.Backlog
	spec := forgeSpec
	if spec == "" {
		if cfg, err := role.LoadProjectConfig(root); err == nil {
			spec = cfg.Forge
		}
	}
	if f, err := forge.Open(spec, root); err == nil && f != nil {
		backlogOf, _ = f.(forge.Backlog)
	}
	allIssues := func() ([]forge.Issue, error) {
		if backlogOf == nil {
			return nil, nil
		}
		return backlogOf.AllIssues()
	}
	before, err := allIssues()
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: the forge's issues:", err)
		return 3
	}
	opened, proposed := 0, 0
	var shares []backlog.ImportShare
	for from := 1; from <= n; {
		to := min(from+share-1, n)
		res := engine.Run(engine.Options{Repo: root, RolesDir: absRoles, Role: "product-owner", Event: "import",
			AI: ai, DefaultAI: userDefaultAI(), Forge: forgeSpec, NoApply: !apply,
			Inputs: map[string]string{"file": file, "from": fmt.Sprint(from), "to": fmt.Sprint(to)}})
		total.AgentCalls += res.AgentCalls
		total.Calls = append(total.Calls, res.Calls...)
		total.Findings = append(total.Findings, res.Findings...)
		total.Applied = append(total.Applied, res.Applied...)
		// Judged only, a share's run waits for `workline apply`, in the job
		// that holds the forge's write token and no AI key, as a line's.
		switch {
		case len(res.Pending) > 0:
			total.Pending = append(total.Pending, res.Pending...)
		case res.ToApply:
			total.Pending = append(total.Pending, res.RunDir)
		}
		for _, k := range res.Applied {
			if k == "open" {
				opened++
			}
		}
		sh := importAnswer(res.RunDir)
		sh.From, sh.To, sh.Own = from, to, n
		if len(shares) > 0 {
			shares[len(shares)-1].Own = from - 1
		}
		shares = append(shares, sh) // what would be opened is the map's, its quotes checked
		fmt.Fprintf(out, "lines %d to %d: %s\n", from, to, res.Status)
		if res.Status != verdict.Pass {
			total.Status = res.Status
			total.Summary = fmt.Sprintf("stopped at lines %d to %d (%s); %d issues opened before — run it again: what is open is not opened twice", from, to, res.Summary, opened)
			return finish(1)
		}
		if to == n {
			break
		}
		from = to - share/5 + 1 // a fifth read again: an item cut at the end is whole in the next share
	}
	after := before
	if apply {
		if after, err = allIssues(); err != nil {
			fmt.Fprintln(os.Stderr, "workline: the forge's issues:", err)
			return 3
		}
	}
	var findings []verdict.Finding
	coverage, findings = backlog.Cover(root, file, strings.Split(string(data), "\n"), shares, before, after, apply)
	total.Findings = append(total.Findings, findings...)
	fmt.Fprint(out, coverage.Text())
	opened, proposed = 0, 0
	for _, m := range coverage.Items {
		switch m.State {
		case backlog.ItemOpened:
			opened++
		case backlog.ItemWouldOpen:
			proposed++
		}
	}
	if apply {
		total.Summary = fmt.Sprintf("%s read; %d issues opened", file, opened)
	} else {
		total.Summary = fmt.Sprintf("%s read; %d issues would be opened — nothing written: --apply opens them, or `workline apply` the runs pending", file, proposed)
	}
	if k := len(coverage.NotCovered); k > 0 {
		// A requirement left with no issue and no reason is a person's to
		// read: never passed silently.
		total.Status = verdict.Human
		total.Summary += fmt.Sprintf("; %d %s not covered — no issue, no reason: see Not covered", k, map[bool]string{true: "item", false: "items"}[k == 1])
		return finish(exitFor(verdict.Human))
	}
	return finish(0)
}

// importAnswer reads what the agent answered on one share of an import —
// the items to open, as the engine decided them, and the items skipped —
// from the share's run folder.
func importAnswer(runDir string) backlog.ImportShare {
	var sh backlog.ImportShare
	if runDir == "" {
		return sh
	}
	var plan backlog.Plan
	planned := false
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "acts.yaml")); err == nil {
		planned = yaml.Unmarshal(data, &plan) == nil
	}
	recorded := backlog.RecordedOpenings(runDir)
	ins, _ := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	for i, in := range ins {
		if in.Kind != "open" {
			continue
		}
		var c backlog.Proposal
		if data, err := yaml.Marshal(in.Value); err != nil || yaml.Unmarshal(data, &c) != nil || c.Quote == nil {
			continue
		}
		o := backlog.ImportOpen{Title: c.Title, Quote: *c.Quote}
		if planned {
			d := plan.Decision(i)
			o.Dropped = d == nil || d.Mode == backlog.Off
			o.Capped = d != nil && d.Mode == backlog.Propose
		}
		if r, ok := recorded[i]; ok {
			o.Opening = &r
		}
		sh.Opens = append(sh.Opens, o)
	}
	skips, _ := intent.Read(filepath.Join(runDir, "out", "skips.yaml"))
	for _, in := range skips {
		var s backlog.Skip
		if data, err := yaml.Marshal(in.Value); err == nil && yaml.Unmarshal(data, &s) == nil {
			sh.Skips = append(sh.Skips, s)
		}
	}
	return sh
}
