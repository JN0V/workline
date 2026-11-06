package simulator

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
)

// splitConfig is the trial copy's line in round 5: the reviewer reads the
// spec the product owner refined, and a draft waits 5 min.
var splitConfig = strings.NewReplacer(
	"roles:\n", "roles:\n  product-owner:\n    settings:\n      acts: {ask: {wait: 5m}}\n",
	"issue: [product-owner, tester, developer]", "issue: [product-owner, reviewer, tester, developer]",
).Replace(config)

// moment is the forge as the trial's #56 stood when its next step was
// asked (traces/trial-56-split.json): its body, comments, labels and
// their dates, each as GitHub's edit history kept them at that time.
type moment struct {
	At    string          `json:"at"`
	Forge forge.FakeState `json:"forge"`
}

// recorder answers every call with no proposal, and keeps the roles asked.
type recorder struct{ roles []string }

func (r *recorder) Propose(req agent.Request) (agent.Call, error) {
	r.roles = append(r.roles, req.Role.Name)
	return agent.Call{Agent: "fake", Model: "sim"}, os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), []byte("[]\n"), 0o644)
}

// TestTrial56Split replays round 5's #56 (#1150) as CI runs it: a judge
// job (product owner, then the reviewer, nothing applied), an apply job,
// then the issue's next step asked of the line. The first reviewer read
// the body before the apply wrote the drafts, so each next step found
// drafts with no review: the product owner was called twice for nothing,
// at 13:13:22 and 13:16:25. Each moment the line is asked: its step due,
// and the roles its judge job calls.
func TestTrial56Split(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("traces", "trial-56-split.json"))
	if err != nil {
		t.Fatal(err)
	}
	var moments map[string]moment
	if err := json.Unmarshal(data, &moments); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		moment string
		due    bool     // its next step started
		calls  []string // the roles its judge job asks
	}{
		// Call 1 applied: drafts, no review on them. The reviewer's turn,
		// not the product owner's.
		{"1-after-drafts", true, []string{"reviewer"}},
		// The Verification rewritten after an important finding, the
		// review on the text before: the reviewer again, alone.
		{"2-after-revise", true, []string{"reviewer"}},
		// The review with nothing important, the delay not passed: the
		// drafts wait, no call (#1127).
		{"3-after-review", false, nil},
	}
	for _, c := range cases {
		t.Run(c.moment, func(t *testing.T) {
			m, ok := moments[c.moment]
			if !ok {
				t.Fatalf("no moment %s in the trace", c.moment)
			}
			at, err := time.Parse(time.RFC3339, m.At)
			if err != nil {
				t.Fatal(err)
			}
			work := t.TempDir()
			hermetic(t, work)
			repo := filepath.Join(work, "repo")
			if out, err := exec.Command("cp", "-a", repoTemplate, repo).CombinedOutput(); err != nil {
				t.Fatalf("the repository: %v\n%s", err, out)
			}
			if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(splitConfig), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := gitIn(repo, "commit", "-q", "--no-verify", "-am", "chore: the trial's line"); err != nil {
				t.Fatal(err)
			}
			f :=filepath.Join(work, "forge.json")
			state, _ := json.Marshal(m.Forge)
			if err := os.WriteFile(f, state, 0o644); err != nil {
				t.Fatal(err)
			}
			roles, _ := filepath.Abs(filepath.Join("..", "..", "roles"))
			t.Setenv(clock.Fixed, at.Format(time.RFC3339))
			rec := &recorder{}
			defer agent.Hook("split", rec)()
			base := engine.Options{Repo: repo, RolesDir: roles, Forge: "fake:" + f, OpenMergeRequest: true,
				Target: &forge.Target{Kind: "issue", ID: 56}}

			// The apply job's next step: the line asked, with no agent.
			ask := base
			ask.AI = "none"
			ask.Inputs = map[string]string{line.ReconcileInput: "list"}
			res := line.RunWith(line.IssueEvent, ask, &line.Gates{})
			due := slices.Contains(res.Due, 56)
			if due != c.due {
				t.Fatalf("at %s, #56 due: %v, want %v: %s", m.At, due, c.due, res.Summary)
			}
			if !due {
				return
			}
			// The judge job it starts: nothing applied.
			judge := base
			judge.AI, judge.NoApply = "hook:split", true
			res = line.RunWith(line.IssueEvent, judge, &line.Gates{})
			if !slices.Equal(uniq(rec.roles), c.calls) {
				t.Fatalf("at %s, the judge job asked %v, want %v: %s", m.At, rec.roles, c.calls, res.Summary)
			}
			// The apply job: its review written, nothing important in it.
			out, _ := json.Marshal(res)
			lj := filepath.Join(work, "line.json")
			if err := os.WriteFile(lj, out, 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(engine.Self(), "apply", "--line", lj)
			cmd.Dir = repo
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("the apply: %v\n%s", err, out)
			}
			// Its next step asked: the drafts wait on a person, no call.
			res = line.RunWith(line.IssueEvent, ask, &line.Gates{})
			waits := slices.ContainsFunc(res.Findings, func(f verdict.Finding) bool { return f.Rule == "state-waits" })
			if slices.Contains(res.Due, 56) || !waits {
				t.Errorf("after the review's apply, #56 does not wait on a person: %s", res.Summary)
			}
		})
	}
}

// uniq is roles, each once, in the order first asked.
func uniq(roles []string) []string {
	var out []string
	for _, r := range roles {
		if !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	return out
}
