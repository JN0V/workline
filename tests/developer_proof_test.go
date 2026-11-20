package tests

// The developer's proof on the real tester (docs/spec/layout.md, Tests):
// the developer asks the tester to run the project's tests through the
// registry (contract.Tests), so these tests link both, as the binary
// does. Moved from the developer's proof*_test.go, as they were.

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/devmark"
	_ "github.com/JN0V/workline/roles" // the compiled-in roles: the tester runs the proof
	developer "github.com/JN0V/workline/roles/developer/code"
)

// proofRepo is a repository holding files, committed on main.
func proofRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for f, content := range files {
		full := filepath.Join(repo, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "-A"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "init"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return repo
}

// A project whose test command is the tester's setting: run-tests.sh
// writes a JUnit report, one test that passes when lib.sh says "ok".
func provingRepo(t *testing.T, command string) string {
	t.Helper()
	roles, err := filepath.Abs("../roles")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKLINE_ROLES_DIR", roles)
	script := `r=$(sh lib.sh)
if [ "$r" = ok ]; then c='<testcase classname="lib" name="ok"/>'; else c='<testcase classname="lib" name="ok"><failure message="no"/></testcase>'; fi
printf '<testsuite name="lib">%s</testsuite>\n' "$c" > "$1"
[ "$r" = ok ]
`
	return proofRepo(t, map[string]string{
		"lib.sh":                "echo no\n",
		"run-tests.sh":          script,
		".workline/config.yaml": "roles:\n  tester:\n    settings:\n      test-command: " + command + "\n",
	})
}

func lib(says string) []intent.Intention {
	return []intent.Intention{{Kind: "patch", Value: map[string]any{"file": "lib.sh", "content": "echo " + says + "\n"}}}
}

func TestProve(t *testing.T) {
	repo := provingRepo(t, `"sh run-tests.sh {report}"`)
	if p, c := developer.Prove(repo, "", lib("ok"), nil, nil, developer.Settings{ProofSeconds: 60}); p.Result != devmark.Green || p.Passed != 1 || c != nil {
		t.Errorf("green: proof = %+v, cap = %+v; want green, 1 passed, no cap", p, c)
	}
	p, c := developer.Prove(repo, "", lib("no"), nil, nil, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Red || len(p.Red) != 1 || p.Red[0] != "lib.ok" || c != nil {
		t.Errorf("red: proof = %+v, cap = %+v; want red, lib.ok red, no cap", p, c)
	}
	if lines := devmark.ProofLines(p, nil); !strings.Contains(lines, "0 passed, 1 red") || !strings.Contains(lines, "red: `lib.ok`") {
		t.Errorf("red: Verification lines %q lack what was red", lines)
	}
	if f := developer.ProofFindings("#3", p, nil); len(f) != 1 || f[0].Rule != "proof-red" {
		t.Errorf("red: findings = %+v, want one proof-red", f)
	}
}

// A flaky-n-of-n proof keeps its test and runs, proven or not: a review
// round runs the same N runs again from the record.
func TestProveFlakyKeepsItsRunsForTheReview(t *testing.T) {
	repo := provingRepo(t, `"sh run-tests.sh {report}"`)
	p, _ := developer.Prove(repo, "", lib("ok"), nil, &developer.Flaky{Test: "ok", Runs: 3}, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Green || p.Ran != 3 || !strings.Contains(devmark.ProofLines(p, nil), "`ok` passed 3 runs of 3") {
		t.Errorf("green: proof = %+v; want green, 3 runs of 3 said", p)
	}
	if n := developer.Again(&p); n == nil || *n != (developer.Flaky{Test: "ok", Runs: 3}) {
		t.Errorf("again(green) = %+v, want ok, 3 runs", n)
	}
	p, _ = developer.Prove(repo, "", lib("ok"), nil, &developer.Flaky{Test: "a b", Runs: 3}, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Unproven || developer.Again(&p) == nil {
		t.Errorf("a name a shell reads: proof = %+v; want unproven, its runs kept", p)
	}
	t.Setenv("WORKLINE_ROLES_DIR", t.TempDir()) // no tester's role: its settings unread
	p, _ = developer.Prove(repo, "", lib("ok"), nil, &developer.Flaky{Test: "ok", Runs: 3}, developer.Settings{})
	if p.Result != devmark.Unproven || developer.Again(&p) == nil {
		t.Errorf("settings unread: proof = %+v; want unproven, its runs kept", p)
	}
	if developer.Again(&devmark.Proof{Result: devmark.Green}) != nil {
		t.Error("again(not flaky) is not nil")
	}
}

// No report written: the tests did not run whole, never green.
func TestProveNotRunWhole(t *testing.T) {
	repo := provingRepo(t, `"true"`)
	p, c := developer.Prove(repo, "", lib("ok"), nil, nil, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Unproven || p.Why == "" || c != nil {
		t.Errorf("proof = %+v, cap = %+v; want unproven, why said, no cap", p, c)
	}
	if f := developer.ProofFindings("#3", p, nil); len(f) != 1 || f[0].Rule != "proof-not-run" {
		t.Errorf("findings = %+v, want one proof-not-run", f)
	}
}

// A report naming no test proves nothing: unproven, never red.
func TestProveEmptyReportIsUnproven(t *testing.T) {
	repo := provingRepo(t, `"printf '<testsuite name=\"lib\"></testsuite>' > {report}"`)
	p, c := developer.Prove(repo, "", lib("ok"), nil, nil, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Unproven || p.Why != "the report names no test" || c != nil {
		t.Errorf("proof = %+v, cap = %+v; want unproven, the report naming no test", p, c)
	}
}

// A command that fails, its report naming no test, and a tool not found in
// its output: unproven, and the tool is held in Missing (#774).
func TestProveUnprovenHoldsTheMissingTool(t *testing.T) {
	repo := provingRepo(t, `"printf '<testsuite name=\"lib\"></testsuite>' > {report}; echo 'run.sh: 1: nosuchtool774: not found'; exit 1"`)
	p, c := developer.Prove(repo, "", lib("ok"), nil, nil, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Unproven || p.Why != "the report names no test" || c != nil {
		t.Fatalf("proof = %+v, cap = %+v; want unproven, the report naming no test", p, c)
	}
	if !slices.Equal(p.Missing, []string{"nosuchtool774"}) {
		t.Errorf("Missing = %v, want [nosuchtool774]", p.Missing)
	}
}

// The same failure with no tool named in its output: Missing is empty.
func TestProveUnprovenWithNoToolMissingNamesNone(t *testing.T) {
	repo := provingRepo(t, `"printf '<testsuite name=\"lib\"></testsuite>' > {report}; exit 1"`)
	p, _ := developer.Prove(repo, "", lib("ok"), nil, nil, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Unproven || len(p.Missing) != 0 {
		t.Errorf("proof = %+v; want unproven, no Missing", p)
	}
}

// Past proof-seconds, the command is stopped: unproven, and the cap said.
func TestProveStoppedPastItsSeconds(t *testing.T) {
	repo := provingRepo(t, `"exec sleep 30"`)
	p, c := developer.Prove(repo, "", lib("ok"), nil, nil, developer.Settings{ProofSeconds: 1})
	if p.Result != devmark.Unproven || c == nil || c.Setting != "proof-seconds" || c.Allows != 1 {
		t.Fatalf("proof = %+v, cap = %+v; want unproven, the proof-seconds cap", p, c)
	}
	f := developer.ProofFindings("#3", p, []devmark.Cap{*c})
	if len(f) != 2 || f[1].Rule != "cap-reached" || f[1].Where != "proof-seconds" {
		t.Errorf("findings = %+v, want proof-not-run then cap-reached on proof-seconds", f)
	}
}

// One case, its group: given a change touching one conformance case
// (tests/conformance/cases/dev/x.yaml), when the developer's proof runs,
// then the proof command runs that case's group: its package carries a
// `-run` naming the group `dev`, not the whole suite.
func TestProveOneCaseRunsItsGroup(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	roles, err := filepath.Abs("../roles")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKLINE_ROLES_DIR", roles)
	// The script writes one red test whose name is the arguments it got
	// after the report path.
	script := `f=$1
shift
printf '<testsuite name="s"><testcase classname="c" name="args %s"><failure message="x"/></testcase></testsuite>\n' "$*" > "$f"
exit 1
`
	repo := proofRepo(t, map[string]string{
		"run-tests.sh":                         script,
		"tests/conformance/runner_test.go":     "package conformance\n",
		"tests/conformance/cases/dev/x.yaml":   "case: x\n",
		"tests/conformance/cases/other/y.yaml": "case: y\n",
		".workline/config.yaml":                "roles:\n  tester:\n    settings:\n      test-command: \"sh run-tests.sh {report}\"\n",
	})
	patches := []intent.Intention{{Kind: "patch", Value: map[string]any{
		"file": "tests/conformance/cases/dev/x.yaml", "content": "case: x\nabout: changed\n",
	}}}
	s := developer.Settings{ProofSeconds: 60, ProofCommand: "sh run-tests.sh {report} {packages}"}
	p, c := developer.Prove(repo, "", patches, nil, nil, s)
	if c != nil || len(p.Red) != 1 {
		t.Fatalf("proof = %+v, cap = %+v; want one red test naming the arguments", p, c)
	}
	args := p.Red[0]
	if !strings.Contains(args, "./tests/conformance") {
		t.Errorf("arguments %q lack the package ./tests/conformance", args)
	}
	if !strings.Contains(args, "-run") || !strings.Contains(args, "dev") {
		t.Errorf("arguments %q: want a -run naming the group dev, not the whole suite", args)
	}
	if strings.Contains(args, "other") {
		t.Errorf("arguments %q name a group the change does not touch", args)
	}
}

// provedGit runs git in repo, cut off from the machine's git config and
// hooks, each commit with its own name and email.
func provedGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	full := append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "core.hooksPath=/dev/null"}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// staleBranch is a project whose test lib.ok is green on a developer
// branch's base, the branch holding its own files; main then moves on,
// its lib.sh saying mainLib. It returns the repository, left on main,
// and the branch's tip.
func staleBranch(t *testing.T, branch map[string]string, mainLib string) (string, string) {
	t.Helper()
	repo := provingRepo(t, `"sh run-tests.sh {report}"`)
	put := func(files map[string]string, msg string) {
		for f, content := range files {
			if err := os.WriteFile(filepath.Join(repo, f), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		provedGit(t, repo, "add", "-A")
		provedGit(t, repo, "commit", "-q", "-m", msg)
	}
	put(map[string]string{"lib.sh": "echo ok\n"}, "the base: lib is ok")
	provedGit(t, repo, "checkout", "-q", "-b", "workline/developer/1")
	put(branch, "the developer's code")
	tip := provedGit(t, repo, "rev-parse", "HEAD")
	provedGit(t, repo, "checkout", "-q", "main")
	put(map[string]string{"lib.sh": mainLib}, "main moves on")
	return repo, tip
}

// Stale base, red on main
// Given a developer branch behind main, and a test green on its base but
// red on the merge with main,
// When the developer's proof runs,
// Then it is red and names that test.
func TestProveRunsOnTheMergeWithMain(t *testing.T) {
	repo, tip := staleBranch(t, map[string]string{"extra.txt": "the branch's own work\n"}, "echo no\n")
	p, c := developer.Prove(repo, tip, nil, nil, nil, developer.Settings{ProofSeconds: 60})
	if p.Result != devmark.Red || !slices.Equal(p.Red, []string{"lib.ok"}) || c != nil {
		t.Fatalf("proof = %+v, cap = %+v; want red, lib.ok red, no cap", p, c)
	}
	if f := developer.ProofFindings("#1", p, nil); len(f) != 1 || f[0].Rule != "proof-red" || !strings.Contains(f[0].Message, "lib.ok") {
		t.Errorf("findings = %+v, want one proof-red naming lib.ok", f)
	}
}

// Merge in conflict
// Given a branch whose merge with main conflicts,
// When the proof would run,
// Then it does not say green; the conflict is said, as a merge round
// would say it.
func TestProveSaysTheConflictWithMain(t *testing.T) {
	repo, tip := staleBranch(t, map[string]string{"lib.sh": "echo ok # the branch's own change\n"}, "echo no\n")
	p, _ := developer.Prove(repo, tip, nil, nil, nil, developer.Settings{ProofSeconds: 60})
	if p.Result == devmark.Green {
		t.Fatalf("proof = %+v; want not green on a merge that conflicts", p)
	}
	for _, want := range []string{"conflict", "main", "lib.sh"} {
		if !strings.Contains(p.Why, want) {
			t.Errorf("why = %q, want it to say the conflict with main in lib.sh (missing %q)", p.Why, want)
		}
	}
}
