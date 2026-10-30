package conformance

import (
	enc "encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The bounds step of workline-bounds.yml, run as GitHub runs it, its
// engine and git stand-ins (ADR-0058, step 2): the engine's verdict is
// the step's status output, its run named only when it is a run's name,
// and a finding's text, the pull request's file names, never a workflow
// command: the engine's output sits between stop-commands and their
// resume, each annotation escaped as GitHub's workflow commands ask.
func TestCIGitHubBoundsStepRuns(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	for _, f := range []string{"../../ci/github/workline-bounds.yml", "../../.github/workflows/workline-bounds.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					ID  string `yaml:"id"`
					Run string `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatal(err)
		}
		var script string
		for _, s := range wf.Jobs["bounds"].Steps {
			if s.ID == "bounds" {
				script = s.Run
			}
		}
		if script == "" {
			t.Fatalf("%s: no step `bounds`", f)
		}
		for _, c := range []struct{ run, wantRun string }{
			{".workline-runs/20261030T163402.410972975-developer", "20261030T163402.410972975-developer"},
			{".workline-runs/../../etc\nstatus=0", ""},
		} {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			// The engine's verdict: a block, a finding whose file name
			// would start a workflow command, its own output one too.
			data, _ := enc.Marshal(map[string]any{"status": "block", "summary": "the bounds of #3 refuse 1 of 2 files changed",
				"pending": []string{c.run},
				"findings": []map[string]string{{"rule": "acceptance-test-edit",
					"where": "a,b:%.go\n::warning::injected", "message": "frozen\r\n::error::injected"}}})
			engine := "#!/bin/sh\necho '::warning::from the engine' >&2\ncat <<'EOF'\n" + string(data) + "\nEOF\nexit 1\n"
			writeExec(t, filepath.Join(bin, "workline"), engine)
			writeExec(t, filepath.Join(bin, "git"), "#!/bin/sh\nexit 0\n")
			output, summary := filepath.Join(dir, "output"), filepath.Join(dir, "summary")
			cmd := exec.Command("bash", "-e", "-c", script)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GITHUB_OUTPUT="+output, "GITHUB_STEP_SUMMARY="+summary,
				"GH_TOKEN=t", "PR=4", "HEAD_REF=workline/developer/3", "HEAD_SHA=abc", "GITHUB_SHA=def")
			stdout, err := cmd.CombinedOutput() // GitHub reads commands on both
			if err != nil {
				t.Fatalf("%s: the step failed: %v\n%s", f, err, stdout)
			}
			got, _ := os.ReadFile(output)
			if !strings.Contains(string(got), "status=1\n") || !strings.Contains(string(got), "run="+c.wantRun+"\n") || strings.Count(string(got), "\n") != 2 {
				t.Errorf("%s: outputs %q, want status=1 and run=%q alone", f, got, c.wantRun)
			}
			if s, _ := os.ReadFile(summary); !strings.Contains(string(s), "refuse 1 of 2") {
				t.Errorf("%s: no summary on the job's page: %q", f, s)
			}
			// Outside stop-commands, the only command is the escaped annotation.
			stopped := ""
			var commands []string
			for l := range strings.Lines(string(stdout)) {
				l = strings.TrimRight(l, "\n")
				switch {
				case stopped == "" && strings.HasPrefix(l, "::stop-commands::"):
					stopped = strings.TrimPrefix(l, "::stop-commands::")
				case stopped != "" && l == "::"+stopped+"::":
					stopped = "done"
				case stopped == "done" && strings.HasPrefix(l, "::"):
					commands = append(commands, l)
				case stopped == "" && strings.HasPrefix(l, "::"):
					t.Errorf("%s: a workflow command before stop-commands: %q", f, l)
				}
			}
			want := "::error file=a%2Cb%3A%25.go%0A%3A%3Awarning%3A%3Ainjected,title=workline bounds::acceptance-test-edit: frozen%0D%0A::error::injected"
			if stopped != "done" || len(commands) != 1 || commands[0] != want {
				t.Errorf("%s: the engine's output not between stop-commands and their resume, or the annotation not escaped:\n%q\nwant %q", f, commands, want)
			}
		}
	}
}

func writeExec(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}
