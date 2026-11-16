package conformance

import (
	enc "encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
)

// The apply job's room step, run as GitHub runs it, on the engine and a
// fake forge, with a gh stand-in that keeps the gardening workflow's
// latest run (#1549): a backlog run starts when Ready, below its target,
// dropped since the last one started, or that run readied an issue; three
// drops while one runs give one waiting, two runs in all; a last run that
// readied none, Ready unchanged since, starts none, said; and gh failing,
// or no gardening workflow, never fails the step.
func TestRoomStepStartsTheBacklogRunOnlyWhenDue(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	const run = "2026-11-16T10:00:00Z/2026-11-16T10:05:00Z"
	ready := func(added bool, at string) forge.LabelEvent {
		return forge.LabelEvent{Label: "workline:ready", Added: added, Note: forge.Note{Created: at}}
	}
	cases := []struct {
		name    string
		latest  string // the gardening workflow's latest run: status start/updated; "" none; "fail" the list fails
		taken   []string
		applies int
		want    int    // runs dispatched
		said    string // in the step's summary
	}{
		{"three drops while one runs", "in_progress " + run, []string{"2026-11-16T11:00:00Z", "2026-11-16T11:10:00Z", "2026-11-16T11:20:00Z"}, 3, 1, "dropped below its target"},
		{"the last readied none, Ready unchanged", "completed " + run, []string{"2026-11-16T09:00:00Z"}, 2, 0, "readied none, and none has left Ready since"},
		{"the last readied one", "completed " + run, nil, 1, 1, "readied an issue"},
		{"no gardening run ever", "", nil, 1, 1, "no backlog run before"},
		{"gh fails, or no gardening workflow", "fail", []string{"2026-11-16T11:00:00Z"}, 1, 0, "could not be read"},
	}
	for _, f := range []string{"../../defaults/ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
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
		for _, s := range wf.Jobs["apply"].Steps {
			if s.ID == "room" {
				script = s.Run
			}
		}
		if script == "" {
			t.Fatalf("%s: no step `room` in the apply job", f)
		}
		for _, c := range cases {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.MkdirAll(filepath.Join(dir, ".workline"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			config := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
			if err := os.WriteFile(filepath.Join(dir, ".workline", "config.yaml"), []byte(config), 0o644); err != nil {
				t.Fatal(err)
			}
			var st forge.FakeState
			for i := range 8 { // 8 ready of 10
				it := forge.FakeItem{ID: i + 1, Labels: []string{"workline:ready"}, Comments: []forge.FakeComment{},
					LabelEvents: []forge.LabelEvent{ready(true, "2026-11-01T09:00:00Z")}}
				if c.name == "the last readied one" && i == 0 {
					it.LabelEvents = []forge.LabelEvent{ready(true, "2026-11-16T10:03:00Z")}
				}
				st.Issues = append(st.Issues, it)
			}
			for i, at := range c.taken { // taken by the developer, at
				st.Issues = append(st.Issues, forge.FakeItem{ID: 20 + i, Labels: []string{"workline:developing"}, Comments: []forge.FakeComment{},
					LabelEvents: []forge.LabelEvent{ready(true, "2026-11-01T09:00:00Z"), ready(false, at)}})
			}
			body, _ := enc.Marshal(st)
			fake := filepath.Join(dir, "forge.json")
			if err := os.WriteFile(fake, body, 0o644); err != nil {
				t.Fatal(err)
			}
			latest := filepath.Join(dir, "latest")
			if err := os.WriteFile(latest, []byte(c.latest), 0o644); err != nil {
				t.Fatal(err)
			}
			calls := filepath.Join(dir, "calls")
			// gh run list: the latest run through --jq, or a failure;
			// gh workflow run workline-gardening.yml: logged, now the latest, queued.
			writeExec(t, filepath.Join(bin, "gh"), `#!/bin/bash
echo "$*" >> "`+calls+`"
latest=$(cat "`+latest+`")
case "$1 $2" in
"run list")
  [ "$latest" != fail ] || { echo "could not find any workflows named workline-gardening.yml" >&2; exit 1; }
  for i in "$@"; do [ "$prev" = --jq ] && q=$i; prev=$i; done
  if [ -z "$latest" ]; then echo '[]'; else
    set -- $latest; echo "[{\"status\":\"$1\",\"startedAt\":\"${2%/*}\",\"updatedAt\":\"${2#*/}\"}]"; fi | jq -r "$q" ;;
"workflow run")
  echo "queued -" > "`+latest+`" ;;
esac
`)
			writeExec(t, filepath.Join(bin, "workline"), `#!/bin/bash
args=(); for a in "$@"; do [ "$a" = github ] && a="fake:`+fake+`"; args+=("$a"); done
exec "`+engineBin+`" "${args[@]}"
`)
			summary := filepath.Join(dir, "summary")
			for range c.applies {
				cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+summary, "GH_TOKEN=t",
					"ISSUE=20", "GITHUB_REF_NAME=main", "GITHUB_WORKFLOW_REF=o/r/.github/workflows/workline-develop.yml@refs/heads/main")
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("%s, %s: the step failed: %v\n%s", f, c.name, err, out)
				}
			}
			log, _ := os.ReadFile(calls)
			if n := strings.Count(string(log), "workflow run workline-gardening.yml"); n != c.want {
				t.Errorf("%s, %s: %d backlog runs started, want %d; gh calls:\n%s", f, c.name, n, c.want, log)
			}
			said, _ := os.ReadFile(summary)
			if !strings.Contains(string(said), c.said) {
				t.Errorf("%s, %s: the summary does not say %q:\n%s", f, c.name, c.said, said)
			}
		}
	}
}
