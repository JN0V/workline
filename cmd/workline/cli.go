package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// command is one command of the dispatcher: what `workline --help` lists,
// and what `workline <name> --help` shows above its options. Help is
// derived from this table only, so a command dispatched is a command listed.
type command struct {
	name, args, about string
	hidden            bool // run by the installed hooks or the shipped roles, not by a person
	run               func([]string) int
}

var commands []command

func init() {
	commands = []command{
		{name: "run-role", args: "<role> --event <event> [options]", about: "run one role on an event: prepare, propose, judge, apply", run: runRole},
		{name: "route", args: "<event> [options]", about: "run the steps routing names for an event, in order", run: routeCmd},
		{name: "apply", args: "<run-dir>... | --line <file> [options]", about: "apply runs judged with --no-apply, or resume a run stopped while applying", run: applyCmd},
		{name: "review", args: "[options]", about: "have the reviewer review this branch before it is pushed", run: reviewCmd},
		{name: "gate", args: "<name> [options]", about: "run a gate declared in .workline/config.yaml", run: gateCmd},
		{name: "item", args: "ready <id> [options]", about: "move a work item to ready, once its Need, Verification, Validation and Scope are written", run: itemCmd},
		{name: "issues", args: "[list] | show <n>|!<n> | import <file> [options]", about: "read the local forge's issues; import a roadmap or backlog file to the forge's issues", run: issuesCmd},
		{name: "docs", args: "[options]", about: "judge the docs made suspect since they were last judged, then review each change", run: docsCmd},
		{name: "sample", args: "[options] | --apply <file> [options]", about: "draw and read the week's sample of the docs the documentalist vouched for; --apply writes it to the forge", run: sampleCmd},
		{name: "follow", args: "[options]", about: "rebuild the roles' release merge requests on their base's new tip", run: followCmd},
		{name: "init", args: "[options]", about: "adopt this repository: route pre-push, have each doc's sources proposed", run: initCmd},
		{name: "setup", args: "[options]", about: "set workline up on this machine, asking what to enable", run: setupCmd},
		{name: "doctor", args: "[options]", about: "say what is set up on this machine and in this repository, and what is missing", run: doctorCmd},
		{name: "hooks", args: "install|uninstall --global|--repo", about: "install or remove workline's git hooks, for every repository or this one", run: hooksCmd},
		{name: "version", args: "", about: "print the engine's version", run: versionCmd},
		{name: "help", args: "[<command>]", about: "show this help, or a command's", run: helpCmd},
		{name: "hook", args: "<git-hook-name> [<args>]", about: "run by the installed git hooks", hidden: true, run: hookCmd},
		{name: "builtin", args: "<role> pre|post", about: "run by the shipped roles' scripts, inside a run", hidden: true, run: builtin},
	}
}

func lookupCommand(name string) *command {
	for i := range commands {
		if commands[i].name == name {
			return &commands[i]
		}
	}
	return nil
}

// dispatch runs the command args name, and returns the exit code: help is
// asked for, never misuse (0); misuse is 64, never 2, which says `human`.
func dispatch(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: workline <command> [<arguments>] [options]\nrun `workline --help` for the commands")
		return 64
	}
	switch args[0] {
	case "-h", "-help", "--help":
		return helpCmd(nil)
	case "--version":
		return versionCmd(nil)
	}
	if c := lookupCommand(args[0]); c != nil {
		return c.run(args[1:])
	}
	fmt.Fprintf(os.Stderr, "workline: unknown command %q; run `workline --help` for the commands\n", args[0])
	return 64
}

// topHelp is what `workline --help` prints: every command a person runs,
// one line each, then where to read more.
func topHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: workline <command> [<arguments>] [options]")
	fmt.Fprintln(w, "\nCommands:")
	var hidden []string
	for _, c := range commands {
		if c.hidden {
			hidden = append(hidden, c.name)
			continue
		}
		fmt.Fprintf(w, "  %-9s %s\n", c.name, c.about)
	}
	fmt.Fprintf(w, "\nInternal, run by the installed hooks and the shipped roles: %s.\n", strings.Join(hidden, ", "))
	fmt.Fprintln(w, "\n`workline <command> --help` (or `workline help <command>`) shows its arguments and options.")
	fmt.Fprintln(w, "Exit codes: 0 pass, 1 block or error, 2 human, 3 blocked-external, 64 misuse.")
	fmt.Fprintln(w, "More: https://github.com/JN0V/workline/blob/main/docs/usage.md")
}

func helpCmd(args []string) int {
	if len(args) == 0 || isHelp(args[0]) {
		topHelp(os.Stdout)
		return 0
	}
	c := lookupCommand(args[0])
	if c == nil || c.name == "help" {
		if c == nil {
			fmt.Fprintf(os.Stderr, "workline help: unknown command %q; run `workline --help` for the commands\n", args[0])
			return 64
		}
		topHelp(os.Stdout)
		return 0
	}
	return c.run([]string{"--help"})
}

func versionCmd(args []string) int {
	fs := newFlags("version")
	if code, done := parseFlags(fs, args); done {
		return code
	}
	if fs.NArg() > 0 {
		return misuse(fs, "takes no argument")
	}
	fmt.Println(engineVersion())
	return 0
}

func isHelp(a string) bool { return a == "-h" || a == "-help" || a == "--help" }

// newFlags is a command's option set. It never exits on its own: parseFlags
// says what to do on --help and on a bad option.
func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

// parseFlags parses args; done says the command ends there, with code: 0
// after printing its help on stdout (-h, --help), 64 after naming the bad
// option and printing its usage on stderr.
func parseFlags(fs *flag.FlagSet, args []string) (code int, done bool) {
	err := fs.Parse(args)
	switch {
	case err == nil:
		return 0, false
	case errors.Is(err, flag.ErrHelp):
		commandHelp(os.Stdout, fs)
		return 0, true
	}
	return misuse(fs, err.Error()), true
}

// parseOnly is parseFlags for a command that takes options alone: an
// argument left over is misuse, never silently dropped with the options
// after it.
func parseOnly(fs *flag.FlagSet, args []string) (code int, done bool) {
	if code, done := parseFlags(fs, args); done {
		return code, done
	}
	if fs.NArg() > 0 {
		return misuse(fs, fmt.Sprintf("unexpected argument %q", fs.Arg(0))), true
	}
	return 0, false
}

// parseAfter is parseOnly for a command that takes one argument first, what
// it names (a role, an event): it returns that argument.
func parseAfter(fs *flag.FlagSet, args []string, what string) (arg string, code int, done bool) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		if code, done := parseFlags(fs, args); done {
			return "", code, done
		}
		return "", misuse(fs, what+" is required, before the options"), true
	}
	code, done = parseOnly(fs, args[1:])
	return args[0], code, done
}

// positional checks a command that takes n arguments and no option: its
// help, else misuse when they are not n.
func positional(fs *flag.FlagSet, args []string, n int) (code int, done bool) {
	switch {
	case len(args) > 0 && isHelp(args[0]):
		commandHelp(os.Stdout, fs)
		return 0, true
	case len(args) != n:
		return misuse(fs, fmt.Sprintf("takes %d arguments, got %d", n, len(args))), true
	}
	return 0, false
}

// misuse says what was wrong, then the command's usage, on stderr: 64.
func misuse(fs *flag.FlagSet, msg string) int {
	fmt.Fprintf(os.Stderr, "workline %s: %s\n", fs.Name(), msg)
	commandHelp(os.Stderr, fs)
	return 64
}

// hiddenFlags are options for the tests alone, left out of the help.
var hiddenFlags = map[string]bool{"test-tamper-before-apply": true}

// flagValue is the placeholder an option's value is shown with.
var flagValue = map[string]string{
	"repo": "<dir>", "roles": "<dir>", "ai": "<agent>", "judge": "<agent>", "forge": "<forge>",
	"event": "<event>", "target": "<issue:n|merge-request:n>", "branch": "<branch>", "base": "<ref>",
	"scope": "<glob>", "input": "<name=value>", "input-file": "<name=path>", "lenses": "<lens,...>",
	"sarif": "<file>", "code-quality": "<file>", "summary": "<file>", "out": "<file>", "line": "<file>",
	"week": "<year-Wnn>", "since": "<rev>", "lines": "<n>", "hooks": "yes|no", "human-po": "yes|no",
	"install": "<tool,...>|all|none",
}

// commandHelp prints the command's usage line, what it does, and its
// options, with the -- every doc uses (Go takes - and -- alike).
func commandHelp(w io.Writer, fs *flag.FlagSet) {
	name := fs.Name()
	c := lookupCommand(name)
	if c == nil {
		fmt.Fprintf(w, "usage: workline %s [options]\n", name)
	} else {
		fmt.Fprintf(w, "usage: %s\n\n%s.\n", strings.TrimSpace("workline "+c.name+" "+c.args), upperFirst(c.about))
	}
	type opt struct{ left, usage string }
	var opts []opt
	width := 0
	fs.VisitAll(func(f *flag.Flag) {
		if hiddenFlags[f.Name] {
			return
		}
		left := "--" + f.Name
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); !ok || !b.IsBoolFlag() {
			v := flagValue[f.Name]
			if f.Name == "apply" && name == "sample" {
				v = "<file>"
			}
			if v == "" {
				v = "<value>"
			}
			left += " " + v
		}
		usage := f.Usage
		if d := f.DefValue; d != "" && d != "false" && d != "[]" && d != "map[]" && !strings.Contains(usage, "default") {
			usage += fmt.Sprintf(" (default %s)", d)
		}
		opts = append(opts, opt{left, usage})
		width = max(width, len(left))
	})
	if len(opts) == 0 {
		return
	}
	width = min(width, 28)
	fmt.Fprintln(w, "\nOptions:")
	for _, o := range opts {
		if len(o.left) > width {
			fmt.Fprintf(w, "  %s\n  %-*s  %s\n", o.left, width, "", o.usage)
			continue
		}
		fmt.Fprintf(w, "  %-*s  %s\n", width, o.left, o.usage)
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
