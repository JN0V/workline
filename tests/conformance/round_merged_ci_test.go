package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// ci-github-round-on-a-merged-request (#1420): a review round started on
// an open pull request that merges, or whose branch is deleted, before
// the review job ends green and says so, asking no round. In the
// template and in workline's own.
func TestCIGitHubRoundOnAMergedRequest(t *testing.T) {
	for _, f := range []string{"../../ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					Name string `yaml:"name"`
					Run  string `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		script := ""
		for _, s := range wf.Jobs["review"].Steps {
			if s.Name == "The round (proposed only)" {
				script = s.Run
			}
		}
		if script == "" {
			t.Fatalf("%s: no step `The round (proposed only)` in the job review", f)
		}
		for _, c := range []struct {
			name, state string // state: what gh says of the pull request, "" when it cannot be read
			branch      bool
			say         string // what the step says; "" for a round asked
		}{
			{"merged, its branch kept", "MERGED", true, "pull request 33 is merged meanwhile: no round"},
			{"its branch deleted, the forge not read", "", false, "pull request 33: its branch workline/developer/26 is gone, merged or closed meanwhile: no round"},
			{"open", "OPEN", true, ""},
		} {
			t.Run(filepath.Base(filepath.Dir(f))+"/"+c.name, func(t *testing.T) {
				dir := t.TempDir()
				repo, bin := filepath.Join(dir, "repo"), filepath.Join(dir, "bin")
				write := func(path, text string, mode os.FileMode) {
					t.Helper()
					if err := os.WriteFile(path, []byte(text), mode); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.MkdirAll(bin, 0o755); err != nil {
					t.Fatal(err)
				}
				env := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
				sh := func(cmd string) {
					t.Helper()
					c := exec.Command("sh", "-c", cmd)
					c.Dir, c.Env = dir, env
					if out, err := c.CombinedOutput(); err != nil {
						t.Fatalf("%s: %v\n%s", cmd, err, out)
					}
				}
				sh("git init -q -b main origin && git -C origin -c user.name=f -c user.email=f@example.invalid commit -q --allow-empty -m start")
				if c.branch {
					sh("git -C origin branch workline/developer/26")
				}
				sh("git clone -q origin repo")
				gh := "#!/bin/sh\nexit 1\n"
				if c.state != "" {
					gh = "#!/bin/sh\necho " + c.state + "\n"
				}
				write(filepath.Join(bin, "gh"), gh, 0o755)
				write(filepath.Join(bin, "workline"), "#!/bin/sh\ntouch \""+filepath.Join(dir, "asked")+"\"\necho '{}'\n", 0o755)
				write(filepath.Join(dir, "round.sh"), script, 0o644)
				run := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", filepath.Join(dir, "round.sh"))
				run.Dir = repo
				run.Env = append(env, "PATH="+bin+":"+os.Getenv("PATH"), "PR=33", "BRANCH=workline/developer/26",
					"GITHUB_OUTPUT="+filepath.Join(dir, "output"), "GITHUB_STEP_SUMMARY="+filepath.Join(dir, "summary"), "CLAUDE_CODE_OAUTH_TOKEN=")
				out, err := run.CombinedOutput()
				if err != nil {
					t.Fatalf("the step failed, the run red: %v\n%s", err, out)
				}
				_, asked := os.Stat(filepath.Join(dir, "asked"))
				switch {
				case c.say != "" && (!strings.Contains(string(out), c.say) || asked == nil):
					t.Errorf("want %q and no round asked, got (round asked: %v):\n%s", c.say, asked == nil, out)
				case c.say == "" && asked != nil:
					t.Errorf("an open pull request: no round asked:\n%s", out)
				}
			})
		}
	}
}
