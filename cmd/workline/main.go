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
//	workline setup [--hooks yes|no] [--ai none|claude|...] [--install <tool,...>|none] [--yes]
//	workline doctor [--repo <dir>] [--json]
//	workline docs [--repo <dir>] [--ai ...]
//	workline init [--repo <dir>] [--ai ...] [--roles <dir>] [--json]
//	workline hook <git-hook-name> [args]  (called by the installed hooks)
//	workline builtin <role> pre|post      (called by the shipped roles' scripts)
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/builtin/documentalist"
	"github.com/JN0V/workline/internal/builtin/releasemanager"
	"github.com/JN0V/workline/internal/doctor"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/hooks"
	"github.com/JN0V/workline/internal/line"
	wlreport "github.com/JN0V/workline/internal/report"
	"github.com/JN0V/workline/internal/review"
	"github.com/JN0V/workline/internal/rolefs"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/setup"
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
	case "doctor":
		os.Exit(doctorCmd(os.Args[2:]))
	case "init":
		os.Exit(initCmd(os.Args[2:]))
	case "setup":
		os.Exit(setupCmd(os.Args[2:]))
	case "docs":
		os.Exit(docsCmd(os.Args[2:]))
	}
	usage()
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: workline run-role <role> --event <event> [options]\n       workline hooks install|uninstall --global|--repo\n       workline setup [--hooks yes|no] [--ai <agent>] [--install <tool,...>|none] [--yes]\n       workline doctor [--repo <dir>] [--json]\n       workline init [--repo <dir>] [--ai <agent>] [--json]")
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
	openMR := fs.Bool("open-merge-request", false, "put what the patches write on a branch of the role, and open a merge request for it (needs a forge)")
	pushMR := fs.Bool("push-to-merge-request", false, "commit what the patches write to the branch of the merge request --target names; from a fork, a comment (needs a forge)")
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
		Forge: *forgeSpec, Target: t, Scope: scope, NoApply: *noApply, OpenMergeRequest: *openMR, PushToMergeRequest: *pushMR,
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
	for _, n := range r.Notes {
		fmt.Fprintf(os.Stderr, "  note from the agent: %s\n", strings.ReplaceAll(n, "\n", "\n    "))
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
		return committer.Post(runDir, repo)
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
	"pending": true, // due at a later moment, the release, which says so
	// About the whole repository, not what is pushed: gardening's.
	"dead-link": true, "duplicate": true, "cites-superseded": true,
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
// agent: a push never waits on one (ADR-0007). It counts the suspect docs,
// never lists them: they are judged on the merge request, by gardening or
// with `workline docs` (ADR-0010).
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
	var suspect, before []string
	for _, ref := range refs {
		rng := ref.Range
		if rng == "" {
			continue
		}
		res := line.Run("pre-push", engine.Options{
			Repo: root, RolesDir: rolesDir, AI: "none",
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
			if f.Rule == "suspect" { // counted, never listed: the push only counts (ADR-0010)
				into := &suspect
				if strings.Contains(f.Message, "left for gardening") {
					into = &before
				}
				if !contains(*into, f.Where) {
					*into = append(*into, f.Where)
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
			fmt.Fprintf(os.Stderr, "workline: %s\npush stopped; `git push --no-verify` skips workline\n", res.Summary)
			return 1
		case patched:
			stat, _ := exec.Command("git", "-C", root, "diff", "--stat").Output()
			fmt.Fprintf(os.Stderr, "workline: the documentalist updated docs in your working tree:\n%s", stat)
			fmt.Fprintln(os.Stderr, "push stopped: review them (git diff), commit them, and push again")
			return 1
		}
	}
	if len(suspect)+len(before) > 0 {
		fmt.Fprintf(os.Stderr, "workline: docs suspect: %d made so by these commits, %d before — `workline docs` judges them\n", len(suspect), len(before))
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
	fs := flag.NewFlagSet("docs", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository")
	ai := fs.String("ai", "", "agent judging the docs (default: the project's, else yours, else none)")
	reviewOnly := fs.Bool("review", false, "no agent: review, doc by doc, the doc changes already in the working tree")
	since := fs.String("since", "", "judge the commits after this one (default: where the docs were last judged, else the last tag, else all)")
	_ = fs.Parse(args)
	root, err := gitRoot(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline: not in a git repository")
		return 64
	}
	if *reviewOnly {
		// What an agent left in the working tree, or gardening run by hand.
		changed, _ := exec.Command("git", "-C", root, "diff", "--name-only", "--", "*.md").Output()
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

// docsJudgedRef is where `workline docs` records the commit the docs were
// last judged up to (ADR-0010): a repository pushed to main with no merge
// request judges from there, pushed or not.
const docsJudgedRef = "refs/workline/docs-judged"

// docsJudgedFrom is the commit `workline docs` judges after, and how to say
// it: the one asked for; else where the docs were last judged — the point
// HEAD shares with it, after a rebase or on another branch; else the last
// tag; else none, every commit.
func docsJudgedFrom(root, since string) (string, string) {
	if since != "" {
		return since, since
	}
	if ref := revParse(root, docsJudgedRef); ref != "" {
		if base, err := exec.Command("git", "-C", root, "merge-base", ref, "HEAD").Output(); err == nil {
			return strings.TrimSpace(string(base)), "they were last judged, " + docsJudgedRef
		}
	}
	if tag, err := exec.Command("git", "-C", root, "describe", "--tags", "--abbrev=0").Output(); err == nil {
		t := strings.TrimSpace(string(tag))
		return t, "the last tag, " + t
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
	if dirty, _ := exec.Command("git", "-C", root, "status", "--porcelain", "--", "*.md").Output(); len(dirty) > 0 {
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
		if f.Rule == "suspect" && !strings.Contains(f.Message, "left for gardening") || f.Level == "block" {
			fmt.Fprintf(out, "  %s %s: %s\n", f.Rule, f.Where, strings.SplitN(f.Message, "\n", 2)[0])
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
	changed, _ := exec.Command("git", "-C", root, "diff", "--name-only", "--", "*.md").Output()
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
	left, _ := exec.Command("git", "-C", root, "diff", "--name-only", "--", "*.md").Output()
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
	openMR := fs.Bool("open-merge-request", false, "put what the patches write on a branch of the role, and open a merge request for it (needs a forge)")
	pushMR := fs.Bool("push-to-merge-request", false, "commit what the patches write to the branch of the merge request --target names; from a fork, a comment (needs a forge)")
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
		Forge: *forgeSpec, Target: t, Scope: scope, NoApply: *noApply, OpenMergeRequest: *openMR, PushToMergeRequest: *pushMR})
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

// setupCmd sets workline up on this machine, asking what to enable, then
// shows the doctor's report. Run again, it reconfigures.
func setupCmd(args []string) int {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	hooksOn := fs.String("hooks", "", "yes or no: the global git hooks (default: ask)")
	ai := fs.String("ai", "", "your agent: none, claude, claude:<model>@<effort>, cmd:<command> (default: ask)")
	install := fs.String("install", "", "the tools to install, comma-separated, all, or none (default: ask)")
	yes := fs.Bool("yes", false, "take the default of every question not answered by a flag")
	asJSON := fs.Bool("json", false, "print the doctor's report, at the end, as JSON")
	_ = fs.Parse(args)
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
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository to look at; outside one, the machine only")
	asJSON := fs.Bool("json", false, "print the report as JSON")
	_ = fs.Parse(args)
	rolesDir, err := resolveRoles(os.Getenv("WORKLINE_ROLES"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 1
	}
	root, _ := gitRoot(*repo)
	r := doctor.Run(doctor.Options{Repo: root, RolesDir: rolesDir, AI: os.Getenv("WORKLINE_AI"), UserAI: userDefaultAI(),
		NoPushApproval: !userConfig().approvePush()})
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
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository to adopt")
	ai := fs.String("ai", "", "agent proposing each doc's sources (default: the project's, else yours, else none)")
	roles := fs.String("roles", os.Getenv("WORKLINE_ROLES"), "folder holding the roles")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	_ = fs.Parse(args)
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
	if stat, _ := exec.Command("git", "-C", root, "status", "--short").Output(); len(stat) > 0 {
		fmt.Fprintf(os.Stderr, "\nIn your working tree:\n%s", stat)
		fmt.Fprintln(os.Stderr, "Review it (git diff): commit what is right, restore what is not (git restore <file>).")
	}
	if len(suspect) > 0 {
		fmt.Fprintf(os.Stderr, "%d docs are suspect already: code they describe changed since they were last edited. The next push has them judged.\n", len(suspect))
	}
	return exitFor(res.Status)
}
