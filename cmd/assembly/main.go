// Command assembly runs the roles of the line.
//
//	assembly run-role <role> --event <event> [--ai none|fake:<file>|unavailable:<reason>]
//	                  [--repo <dir>] [--roles <dir>]
//	                  [--input name=value]... [--input-file name=path]... [--json]
//	assembly builtin <role> pre|post    (called by the shipped roles' scripts)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/assembly-line/internal/builtin/committer"
	"github.com/JN0V/assembly-line/internal/engine"
	"github.com/JN0V/assembly-line/internal/verdict"
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
	}
	usage()
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: assembly run-role <role> --event <event> [options]  |  assembly builtin <role> pre|post")
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
	ai := fs.String("ai", "none", "agent: none, fake:<file>, unavailable:<reason>")
	repo := fs.String("repo", ".", "repository to work on")
	roles := fs.String("roles", defaultRoles(), "folder holding the roles")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	tamper := fs.Bool("test-tamper-before-apply", false, "conformance tests only")
	inputs, inputFiles := pairs{}, pairs{}
	fs.Var(inputs, "input", "input name=value (repeatable)")
	fs.Var(inputFiles, "input-file", "input name=path, written back by intentions that target it (repeatable)")
	_ = fs.Parse(args[1:])

	targets := map[string]string{}
	for k, p := range inputFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "assembly:", err)
			return 1
		}
		inputs[k] = string(data)
		targets[k] = p
	}
	absRepo, _ := filepath.Abs(*repo)
	absRoles, _ := filepath.Abs(*roles)
	res := engine.Run(engine.Options{
		Repo: absRepo, RolesDir: absRoles, Role: name, Event: *event, AI: *ai,
		Inputs: inputs, Targets: targets, TamperBeforeApply: *tamper,
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
	for _, f := range r.Findings {
		level := ""
		if f.Level != "" {
			level = " (" + f.Level + ")"
		}
		fmt.Fprintf(os.Stderr, "  %s%s: %s\n", f.Rule, level, f.Message)
	}
}

// defaultRoles is $ASSEMBLY_ROLES, else the roles folder next to the binary's module.
func defaultRoles() string {
	if d := os.Getenv("ASSEMBLY_ROLES"); d != "" {
		return d
	}
	return "roles"
}

func builtin(args []string) int {
	if len(args) != 2 {
		usage()
	}
	runDir := os.Getenv("ASSEMBLY_RUN_DIR")
	if runDir == "" {
		fmt.Fprintln(os.Stderr, "assembly builtin: ASSEMBLY_RUN_DIR is not set; this command is run by the engine")
		return 99
	}
	repo, _ := os.Getwd()
	switch args[0] + " " + args[1] {
	case "committer pre":
		return committer.Pre(runDir, repo)
	case "committer post":
		return committer.Post(runDir)
	}
	fmt.Fprintf(os.Stderr, "assembly builtin: no built-in step %q\n", strings.Join(args, " "))
	return 99
}
