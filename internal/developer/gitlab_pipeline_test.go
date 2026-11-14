package developer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// glHead is the head of the developer's merge request on the simulated
// GitLab, whose head pipeline is number 90.
const glHead = "5e59697bbc77f80b91c6ed9a1423d908ad71e49d"

// glPipeline is the head pipeline of merge request 7 as GitLab's REST
// API tells it: its status, its jobs, and each job's trace.
type glPipeline struct {
	status string
	jobs   []map[string]any
	traces map[int]string
}

func glJob(id int, name, status string) map[string]any {
	return map[string]any{"id": id, "name": name, "stage": "test", "status": status, "allow_failure": false,
		"pipeline": map[string]any{"id": 90, "sha": glHead}}
}

// gitlabHeadPipeline opens the GitLab forge on a server that answers the
// pipelines of the head, a pipeline's jobs and a job's trace, as GitLab's
// REST API does, apart from the machine's own settings.
func gitlabHeadPipeline(t *testing.T, p *glPipeline) forge.Forge {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		head := map[string]any{"id": 90, "iid": 5, "sha": glHead, "status": p.status}
		older := map[string]any{"id": 80, "iid": 4, "sha": "1111111111111111111111111111111111111111", "status": "success"}
		switch {
		case path == "/pipelines" || path == "/merge_requests/7/pipelines":
			if sha := r.URL.Query().Get("sha"); sha != "" && sha != glHead {
				_ = json.NewEncoder(w).Encode([]map[string]any{older})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{head, older})
		case path == "/pipelines/90" || path == "/pipelines/latest":
			_ = json.NewEncoder(w).Encode(head)
		case path == "/pipelines/90/jobs":
			_ = json.NewEncoder(w).Encode(p.jobs)
		case path == "/pipelines/80/jobs":
			_, _ = w.Write([]byte(`[]`))
		case path == "/merge_requests/7":
			_ = json.NewEncoder(w).Encode(map[string]any{"iid": 7, "sha": glHead, "source_branch": "workline/developer/3",
				"head_pipeline": head})
		case path == "/merge_requests/7/notes":
			_, _ = w.Write([]byte(`[]`))
		case path == "/api/v4/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "workline-bot"})
		case strings.HasPrefix(path, "/jobs/") && strings.HasSuffix(path, "/trace"):
			for _, j := range p.jobs {
				if path == "/jobs/"+json.Number(strings.TrimSuffix(strings.TrimPrefix(path, "/jobs/"), "/trace")).String()+"/trace" {
					id := int(j["id"].(int))
					if path == "/jobs/"+itoa(id)+"/trace" {
						_, _ = w.Write([]byte(p.traces[id]))
						return
					}
				}
			}
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	t.Setenv("WORKLINE_GITLAB_TOKEN", "")
	t.Setenv("CI_JOB_TOKEN", "")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	f, err := forge.Open("gitlab", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// Failed pipeline: given the developer's merge request on GitLab, when its
// head pipeline fails, then a round reads the failed jobs' traces: the
// project's job that failed is asked with the end of its trace; a job that
// passed, and workline's own, are not.
func TestGitLabFailedPipelineAsksTheFailedJobsTraces(t *testing.T) {
	f := gitlabHeadPipeline(t, &glPipeline{
		status: "failed",
		jobs: []map[string]any{
			glJob(501, "test", "failed"),
			glJob(502, "lint", "success"),
			glJob(503, "workline:judge", "failed"),
		},
		traces: map[int]string{
			501: "Running with gitlab-runner\n$ go test ./...\n--- FAIL: TestMulTwoByThree (0.01s)\n    mul 2 3: want 6, got 5\nERROR: Job failed: exit code 1\n",
			503: "the reviewer's own run failed\n",
		},
	})
	at, asked, unread := checksAsked(f, Act{}, glHead)
	if unread != "" {
		t.Fatalf("the pipeline was not read: %s", unread)
	}
	if at != glHead || len(asked) != 1 {
		t.Fatalf("read at %q, asked %d, want the head and the one failed job `test`: %+v", at, len(asked), asked)
	}
	if a := asked[0]; a.Where != "test" || a.Holds != checkHolds || a.Key != "checks|test" {
		t.Errorf("the failed job read as %+v", a)
	}
	checkLogs(f, asked)
	for _, want := range []string{"--- FAIL: TestMulTwoByThree (0.01s)", "mul 2 3: want 6, got 5", "ERROR: Job failed: exit code 1"} {
		if !strings.Contains(asked[0].Detail, want) {
			t.Errorf("its trace lacks %q:\n%s", want, asked[0].Detail)
		}
	}
	if strings.Contains(asked[0].Detail, "the reviewer's own run failed") {
		t.Errorf("workline's own job's trace was read:\n%s", asked[0].Detail)
	}
}

// Red gate: given a gate red on its head, when the round runs, then the
// gate's new findings are asked, once: the gates job of the head pipeline
// failed, its trace ends with the gates' summary; the head they were read
// at kept in the record, the same report asks nothing again.
func TestGitLabRedGateAsksItsNewFindingsOnce(t *testing.T) {
	f := gitlabHeadPipeline(t, &glPipeline{
		status: "failed",
		jobs: []map[string]any{
			glJob(601, "workline:gates", "failed"),
			glJob(602, "test", "success"),
		},
		traces: map[int]string{
			601: "workline: gates blocked\n### Gates: 1 new finding\n\n" +
				"On `5e59697`, only what this change adds since `8b3e072` counts; what was there before is counted, not shown.\n\n" +
				"- **`errcheck`**, 1 new: `cmd/workline/cli.go:22`. Error return value of `w.Close` is not checked\n",
		},
	})
	b, ok := f.(forge.Backlog)
	if !ok {
		t.Fatal("the GitLab forge keeps no backlog")
	}
	gates := func(r round) (rows []Asked) {
		for _, x := range r.Asked {
			if x.Holds == gateHolds {
				rows = append(rows, x)
			}
		}
		return rows
	}
	r, _, err := readAsked(f, b, forge.Kept{}, t.TempDir(), Act{}, 7, glHead)
	if err != nil {
		t.Fatal(err)
	}
	rows := gates(r)
	if len(rows) != 1 {
		t.Fatalf("asked %d gate findings, want the one new errcheck finding: %+v", len(rows), r.Asked)
	}
	if x := rows[0]; x.Where != "cmd/workline/cli.go:22" || x.Key != "gates|cmd/workline/cli.go|errcheck" {
		t.Errorf("the finding read as %+v", x)
	}
	if !sameCommit(r.GatesAt, glHead) {
		t.Fatalf("the head the gates were read at is %q, want %s", r.GatesAt, glHead)
	}
	again, _, err := readAsked(f, b, forge.Kept{}, t.TempDir(), Act{GatesAt: r.GatesAt}, 7, glHead)
	if err != nil {
		t.Fatal(err)
	}
	if rows := gates(again); len(rows) != 0 {
		t.Errorf("the same report asked again: %+v", rows)
	}
}

// Draft made green: given a Draft whose rounds make every job green, when
// the round runs, then the Draft is marked ready: with a job still red it
// is kept a Draft, with every job of its head pipeline green it is ready.
func TestGitLabDraftMadeGreenIsReady(t *testing.T) {
	p := &glPipeline{
		status: "failed",
		jobs:   []map[string]any{glJob(701, "test", "failed"), glJob(702, "lint", "success")},
	}
	f := gitlabHeadPipeline(t, p)
	if ready, kept := draftReady(f, Act{}, glHead); ready || kept != "" {
		t.Errorf("a job red: ready %v, kept %q, want a Draft left to its round", ready, kept)
	}
	p.status = "success"
	p.jobs = []map[string]any{glJob(701, "test", "success"), glJob(702, "lint", "success")}
	if ready, kept := draftReady(f, Act{}, glHead); !ready || kept != "" {
		t.Errorf("every job green: ready %v, kept %q, want it marked ready", ready, kept)
	}
}
