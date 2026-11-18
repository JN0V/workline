package tester

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// brokenBuild is a repository whose base holds one passing test, and the
// settings and patch of a tester whose new test does not build there
// (a call with too many arguments, no name the issue adds): a build
// broken past the fast path, the judge's to read. An agent is on.
func brokenBuild(t *testing.T) (repo, base string, s Settings, patches []intent.Intention) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("WORKLINE_AI", "claude")
	t.Setenv("WORKLINE_JUDGED", "")
	repo, run := gitRepo(t, map[string]string{"go.mod": "module example.com/m\n"})
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No gate and no review on the tests: only the red proof runs.
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("routing: {events: {merge-request: []}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "add", ".")
	run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "config")
	base = run(repo, "rev-parse", "HEAD")
	s = Settings{
		Report:          "junit.xml",
		Paths:           []string{"**/*_test.go"},
		JudgeReadRounds: 1,
		JudgeReadMax:    10,
		BuildErrors:     []BuildErrors{{Error: `^(?P<file>\S+\.go):\d+:\d+: (?P<message>.+)$`, TestName: `(?m)^func (Test\w+)\(`}},
		Command: `if [ -f calc/new_test.go ]; then echo "calc/new_test.go:5:9: too many arguments in call to Add"; exit 1; fi; ` +
			`printf '<testsuite name="old"><testcase classname="old" name="TestOld"/></testsuite>' > {report}`,
	}
	patches = []intent.Intention{{Kind: "patch", Value: map[string]any{
		"file":    "calc/new_test.go",
		"content": "package calc\n\nfunc TestNew(t *testing.T) { check(Add(1, 2, 3)) }\n",
	}}}
	return repo, base, s, patches
}

// judgedOnce runs the tester in runDir on the broken build: the judge is
// asked, answers no, and the engine runs the tester again with its
// answer, build-broken.
func judgedOnce(t *testing.T, runDir, repo, base string, s Settings, patches []intent.Intention) {
	t.Helper()
	_, _, _, asked, err := proveJudged(runDir, repo, base, patches, s)
	if err != nil || !asked {
		t.Fatalf("first run: asked %v, err %v; want the judge asked", asked, err)
	}
	answer := "yes: false\nwhy: a call the issue never asks for\nlevel: model\nmodel: m\n"
	if err := os.WriteFile(filepath.Join(runDir, "out", "judge", redProof, "answer.yaml"), []byte(answer), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKLINE_JUDGED", "answered")
	v, _, _, asked, err := proveJudged(runDir, repo, base, patches, s)
	t.Setenv("WORKLINE_JUDGED", "")
	if err != nil || asked || v == nil || v.Status != verdict.Block {
		t.Fatalf("first run, answered: %v, asked %v, err %v; want build-broken", v, asked, err)
	}
}

// notAskedAgain checks that proveJudged in runDir gives the first answer
// as build-broken, the judge not asked.
func notAskedAgain(t *testing.T, runDir, repo, base string, s Settings, patches []intent.Intention) {
	t.Helper()
	v, _, _, asked, err := proveJudged(runDir, repo, base, patches, s)
	if err != nil {
		t.Fatal(err)
	}
	if asked {
		t.Fatal("the judge was asked again on the build it already judged")
	}
	if _, err := os.Stat(filepath.Join(runDir, "out", "judge", redProof, "question.yaml")); err == nil {
		t.Error("a question was written for the judge")
	}
	var found *verdict.Finding
	if v != nil {
		for i, f := range v.Findings {
			if f.Rule == "build-broken" {
				found = &v.Findings[i]
			}
		}
	}
	if v == nil || v.Status != verdict.Block || found == nil || !strings.Contains(found.Message, "the judge: a call the issue never asks for") {
		t.Errorf("the first answer is not used: %+v", v)
	}
}

// Same build, new tester run: given a tester whose tests do not build,
// judged once, when the tester runs again in the same step on the same
// broken build, then the judge is not asked again and its first answer
// is used.
func TestJudgedSameBuildNewTesterRun(t *testing.T) {
	repo, base, s, patches := brokenBuild(t)
	runs := filepath.Join(repo, ".workline-runs")
	first, second := filepath.Join(runs, "20260101T000000-tester"), filepath.Join(runs, "20260101T000100-tester")
	judgedOnce(t, first, repo, base, s, patches)
	notAskedAgain(t, second, repo, base, s, patches)
}

// Same build, next tick: given the same judged broken build, when the
// next tick runs the tester again on it, then the judge is not asked
// again.
func TestJudgedSameBuildNextTick(t *testing.T) {
	repo, base, s, patches := brokenBuild(t)
	runs := filepath.Join(repo, ".workline-runs")
	first, next := filepath.Join(runs, "20260101T000000-tester"), filepath.Join(runs, "20260102T000000-tester")
	judgedOnce(t, first, repo, base, s, patches)
	// The tick is over: its run folder is gone.
	if err := os.RemoveAll(first); err != nil {
		t.Fatal(err)
	}
	notAskedAgain(t, next, repo, base, s, patches)
}
