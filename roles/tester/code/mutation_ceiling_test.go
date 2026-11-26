//go:build !windows

package tester

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// The two mutants the stand-in for Gremlins runs, as readMutants keys
// them (folder, file, line, column, kind).
const (
	runawayMutantKey = "./pkg/m.go:1:1:ARITHMETIC_BASE"
	normalMutantKey  = "./pkg/m.go:2:1:CONDITIONALS_NEGATION"
)

// curatorMutationLine is the curator's mutation command this repository
// configures, as its `.workline/config.yaml` holds it.
func curatorMutationLine(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".workline", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Roles struct {
			Tester struct {
				Settings struct {
					Mutation string `yaml:"curator-mutation-command"`
				} `yaml:"settings"`
			} `yaml:"tester"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.Roles.Tester.Settings.Mutation) == "" {
		t.Fatal("the repository's .workline/config.yaml sets no curator-mutation-command")
	}
	return cfg.Roles.Tester.Settings.Mutation
}

// runRunawayMutants runs the configured mutation command as the curator
// does, with a `go` on the PATH standing for Gremlins: it runs two
// prebuilt mutants, one growing a strings.Builder in a loop, one that
// ends at once, and writes Gremlins' JSON report, a mutant that exits
// non-zero KILLED, one that exits 0 LIVED. The runaway one stops by
// itself at 2 GiB, so a machine with no ceiling is not filled; a
// ceiling under that kills it first. ci says whether the run has the
// variables a CI runner sets. It returns the report's mutants by key.
func runRunawayMutants(t *testing.T, ci bool) map[string]string {
	t.Helper()
	line := curatorMutationLine(t)
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	build := func(name, body string) string {
		src := filepath.Join(dir, name+".go")
		if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		exe := filepath.Join(dir, name+".bin")
		cmd := exec.Command(goBin, "build", "-o", exe, src)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOFLAGS=")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building the mutant %s: %v\n%s", name, err, out)
		}
		return exe
	}
	runaway := build("runaway", `package main

import "strings"

func main() {
	var b strings.Builder
	chunk := strings.Repeat("x", 1<<20)
	for b.Len() < 2<<30 {
		b.WriteString(chunk)
	}
	println("survived", b.Len())
}
`)
	normal := build("normal", "package main\n\nfunc main() { println(\"fine\") }\n")

	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"out=\"\"\n" +
		"while [ $# -gt 0 ]; do if [ \"$1\" = \"-o\" ]; then out=\"$2\"; fi; shift; done\n" +
		"status() { if \"$1\" >/dev/null 2>&1; then echo LIVED; else echo KILLED; fi; }\n" +
		"a=$(status '" + runaway + "')\n" +
		"b=$(status '" + normal + "')\n" +
		"printf '{\"files\":[{\"file_name\":\"m.go\",\"mutations\":[" +
		"{\"type\":\"ARITHMETIC_BASE\",\"status\":\"%s\",\"line\":1,\"column\":1}," +
		"{\"type\":\"CONDITIONALS_NEGATION\",\"status\":\"%s\",\"line\":2,\"column\":1}]}]}' \"$a\" \"$b\" > \"$out\"\n"
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, k := range []string{"CI", "GITHUB_ACTIONS"} {
		if ci {
			t.Setenv(k, "true")
		} else {
			t.Setenv(k, "")
			_ = os.Unsetenv(k)
		}
	}

	report := filepath.Join(dir, "report.json")
	cmdLine := strings.NewReplacer("{package}", "./pkg", "{report}", report).Replace(line)
	res, late, runErr := shell(dir, cmdLine, time.Now().Add(3*time.Minute))
	if late || runErr != nil {
		t.Fatalf("the mutation command did not end well (late %v, err %v); output: %s", late, runErr, res.String())
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("the mutation command wrote no report: %v; output: %s", err, res.String())
	}
	got := map[string]string{}
	if err := readMutants(data, "./pkg", got); err != nil {
		t.Fatal(err)
	}
	return got
}

// Given a mutant that allocates without end (a strings.Builder grown in
// a loop), when the curator's mutation check runs it, then that mutant
// is killed at a memory ceiling and counted as killed, and the run goes
// on: the mutant after it is run and reported, and the command ends well.
func TestMutationCheckKillsARunawayMutantAndGoesOn(t *testing.T) {
	got := runRunawayMutants(t, false)
	if st, ok := got[runawayMutantKey]; !ok || !killed(st) {
		t.Errorf("the runaway mutant's status = %q (reported %v), want it killed at the memory ceiling", st, ok)
	}
	if st, ok := got[normalMutantKey]; !ok || st != "LIVED" {
		t.Errorf("the mutant after it = %q (reported %v), want it run and reported LIVED: the run goes on", st, ok)
	}
}

// Given a person's PC or a CI runner, when the check runs, then the same
// ceiling applies: the runaway mutant is killed and the other reported
// alike with the variables a CI runner sets and without them.
func TestMutationCeilingIsTheSameOnPCAndCI(t *testing.T) {
	local := runRunawayMutants(t, false)
	ci := runRunawayMutants(t, true)
	for _, key := range []string{runawayMutantKey, normalMutantKey} {
		if local[key] != ci[key] {
			t.Errorf("mutant %s: %q on a PC, %q on CI, want the same", key, local[key], ci[key])
		}
	}
	if !killed(local[runawayMutantKey]) || !killed(ci[runawayMutantKey]) {
		t.Errorf("the runaway mutant: %q on a PC, %q on CI, want it killed at the ceiling on both",
			local[runawayMutantKey], ci[runawayMutantKey])
	}
}
