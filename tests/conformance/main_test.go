package conformance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Main after a merge (ADR-0050): on each push to the default branch, the
// line of `main` runs once, in the job holding no secret: the gates every
// finding kept by the commit where main alone writes it, the whole repository's
// checks after them, no agent. A pull request based on that commit reads
// the kept findings as its merge base's. One job that writes keeps what
// main's run found in one issue, and only main's run: never a pull
// request's, nor another branch's push.

// A push to main as GitHub gives it to a workflow.
var mainPush = map[string]any{
	"github.event_name":   "push",
	"github.repository":   "owner/repo",
	"github.sha":          "a0a0",
	"github.event.before": "9f9f",
}

func TestGitHubMainAfterAMergeRunsOnce(t *testing.T) {
	type step struct {
		Name string         `yaml:"name"`
		Uses string         `yaml:"uses"`
		With map[string]any `yaml:"with"`
		Env  map[string]any `yaml:"env"`
		If   string         `yaml:"if"`
		Run  string         `yaml:"run"`
	}
	type job struct {
		If          string            `yaml:"if"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []step            `yaml:"steps"`
	}
	type workflow struct {
		On   map[string]any `yaml:"on"`
		Jobs map[string]job `yaml:"jobs"`
	}
	read := func(f string) workflow {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		return wf
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		wf := read(dir + "workline.yml")
		if fmt.Sprint(wf.On["push"]) != "map[branches:[main]]" {
			t.Errorf("%sworkline.yml: no run on a push to main (%v)", dir, wf.On["push"])
		}
		ctx := map[string]any{"steps.any.outputs.any": "true", "steps.base.outputs.sha": "9f9f", "steps.main.outputs.kept": "true"}
		for k, v := range mainPush {
			ctx[k] = v
		}
		if j := wf.Jobs["judge"]; truthy(eval(t, j.If, ctx)) {
			t.Errorf("%sworkline.yml: judge runs on main, beside the agent's token: %q", dir, j.If)
		}
		gates := wf.Jobs["gates"]
		if len(gates.Permissions) != 1 || gates.Permissions["contents"] != "read" {
			t.Errorf("%sworkline.yml: main's run holds %v, want contents: read alone", dir, gates.Permissions)
		}
		var restored, saved map[string]any
		mains := 0
		for _, s := range gates.Steps {
			if s.If != "" && !truthy(eval(t, s.If, ctx)) {
				continue
			}
			if env, _ := yaml.Marshal(s.Env); strings.Contains(string(env), "secrets.") {
				t.Errorf("%sworkline.yml: main's step %q holds a secret", dir, s.Name)
			}
			run := expand(t, s.Run, ctx)
			switch {
			case strings.HasPrefix(s.Uses, "actions/checkout@"):
				if ref := expand(t, fmt.Sprint(s.With["ref"]), ctx); ref != "a0a0" {
					t.Errorf("%sworkline.yml: main's run checks out %q, want the pushed commit", dir, ref)
				}
			case strings.HasPrefix(s.Uses, "actions/cache/restore@"):
				restored = s.With
			case strings.HasPrefix(s.Uses, "actions/cache/save@"):
				saved = s.With
			case strings.Contains(run, "--gates-only"):
				t.Errorf("%sworkline.yml: the merge request's gates run on main: %s", dir, s.Name)
			case strings.Contains(run, "workline route main"):
				mains++
				for _, want := range []string{"--ai none", "--forge none", `--gates-kept "$RUNNER_TEMP/workline-kept"`, "--input range=9f9f..a0a0"} {
					if !strings.Contains(run, want) {
						t.Errorf("%sworkline.yml: main's run without %s:\n%s", dir, want, run)
					}
				}
			}
		}
		if mains != 1 {
			t.Errorf("%sworkline.yml: %d steps run main's line on a push, want one", dir, mains)
		}
		// Read at main's tip before the merge, kept by the commit pushed,
		// in the same folder; never by a prefix, an older commit's.
		if restored == nil || saved == nil || restored["path"] != saved["path"] || fmt.Sprint(restored["path"]) != "${{ runner.temp }}/workline-kept" {
			t.Errorf("%sworkline.yml: main's findings not read and kept in one folder: %v, %v", dir, restored, saved)
		} else {
			if got := expand(t, fmt.Sprint(restored["key"]), ctx); got != "workline-gates-9f9f" {
				t.Errorf("%sworkline.yml: main's run reads the findings kept at %q, want main's tip before the merge", dir, got)
			}
			if got := expand(t, fmt.Sprint(saved["key"]), ctx); got != "workline-gates-a0a0" {
				t.Errorf("%sworkline.yml: main's run keeps its findings as %q, want the commit pushed", dir, got)
			}
			if _, ok := restored["restore-keys"]; ok {
				t.Errorf("%sworkline.yml: the findings are read by a prefix: another commit's could come back", dir)
			}
		}
		// A pull request reads them at its merge base.
		pr := map[string]any{"steps.any.outputs.any": "true", "steps.base.outputs.sha": "b0b0"}
		for k, v := range pullRequestPush {
			pr[k] = v
		}
		reads := false
		for _, s := range gates.Steps {
			if s.If != "" && !truthy(eval(t, s.If, pr)) {
				continue
			}
			if strings.HasPrefix(s.Uses, "actions/cache/save@") {
				t.Errorf("%sworkline.yml: a pull request's run keeps findings as main's", dir)
			}
			if run := expand(t, s.Run, pr); strings.Contains(run, "--gates-only") {
				reads = strings.Contains(run, `--gates-kept "$RUNNER_TEMP/workline-kept"`)
			}
			if s.Name == "The base's findings, kept by main" && (!strings.Contains(s.Run, `git merge-base "$BASE" "$HEAD"`) || fmt.Sprint(s.Env["HEAD"]) != "${{ github.event.pull_request.head.sha }}") {
				t.Errorf("%sworkline.yml: a pull request does not read the findings kept at its merge base: %s", dir, s.Run)
			}
		}
		if !reads {
			t.Errorf("%sworkline.yml: a pull request's gates do not read the findings main kept", dir)
		}

		// After main's run, one job keeps what it found in an issue; after a
		// push to another branch, none.
		apply := read(dir + "workline-apply.yml")
		for _, c := range []struct {
			branch string
			want   []string
		}{{"main", []string{"main"}}, {"feature", nil}} {
			run := map[string]any{"github.event_name": "workflow_run", "github.repository": "owner/repo",
				"github.event.workflow_run.name": "workline", "github.event.workflow_run.event": "push",
				"github.event.workflow_run.head_branch": c.branch, "github.event.repository.default_branch": "main",
				"github.event.workflow_run.conclusion": "failure", "github.event.workflow_run.head_repository.full_name": "owner/repo"}
			var ran []string
			for name, j := range apply.Jobs {
				if j.If != "" && truthy(eval(t, j.If, run)) && name != "follow" {
					ran = append(ran, name)
				}
			}
			slices.Sort(ran)
			if !slices.Equal(ran, c.want) {
				t.Errorf("%sworkline-apply.yml: after a push to %s, runs %v, want %v", dir, c.branch, ran, c.want)
			}
		}
		m := apply.Jobs["main"]
		if m.Permissions["issues"] != "write" || m.Permissions["contents"] != "read" || m.Permissions["pull-requests"] != "" {
			t.Errorf("%sworkline-apply.yml, job main: permissions %v, want issues: write and contents: read", dir, m.Permissions)
		}
		keeps := 0
		for _, s := range m.Steps {
			if strings.Contains(s.Run, "workline apply --main") {
				keeps++
				if !strings.Contains(s.Run, `-n workline-main -D "$RUNNER_TEMP/main"`) || !strings.Contains(s.Run, `--main "$RUNNER_TEMP/main/main.json" --forge github`) {
					t.Errorf("%sworkline-apply.yml, job main: does not keep main's own result, read outside the checkout:\n%s", dir, s.Run)
				}
			}
		}
		if keeps != 1 {
			t.Errorf("%sworkline-apply.yml, job main: %d steps keep main's findings, want one", dir, keeps)
		}
	}
}

func TestGitLabMainAfterAMergeRunsOnce(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	job := func(name string) map[string]any {
		j, _ := top[name].(map[string]any)
		if j == nil {
			t.Fatalf("no job %s", name)
		}
		return j
	}
	main := job("workline:main")
	script := fmt.Sprint(main["script"])
	if s, _ := main["script"].([]any); len(s) == 0 || fmt.Sprint(s[0]) != "unset CLAUDE_CODE_OAUTH_TOKEN WORKLINE_GITLAB_TOKEN GITLAB_TOKEN" {
		t.Errorf("workline:main: does not unset the agent's and the bot's tokens first: %v", main["script"])
	}
	for _, want := range []string{"workline route main --ai none --forge none --gates-kept .workline-kept", "--input range=$CI_COMMIT_BEFORE_SHA..$CI_COMMIT_SHA"} {
		if !strings.Contains(script, want) {
			t.Errorf("workline:main: runs without %s: %s", want, script)
		}
	}
	if reports, _ := main["artifacts"].(map[string]any)["reports"].(map[string]any); reports["codequality"] != nil {
		t.Errorf("workline:main: gives GitLab a Code Quality report, which the merge requests' widget would read old findings as fixed against")
	}
	// A base written by a branch is never read: the findings come from
	// main's own pipeline artifacts, by the default branch and its job,
	// workline:main; no cache holds them, which a merge request's
	// pipeline could write (`unprotect`). workline:base reads them, and
	// hands them to its own pipeline's job as an artifact, which no
	// other pipeline reads.
	const fromMain = `/jobs/artifacts/$CI_DEFAULT_BRANCH/raw/.workline-kept/$1?job=workline:main`
	for name, v := range top {
		j, _ := v.(map[string]any)
		if c, _ := yaml.Marshal(j["cache"]); strings.Contains(string(c), ".workline-kept") || strings.Contains(string(c), "unprotect") {
			t.Errorf("%s: keeps or reads the stored base in a cache, which a branch's pipeline could write: %s", name, c)
		}
		paths, _ := yaml.Marshal(j["artifacts"])
		if strings.Contains(string(paths), ".workline-kept/") != (name == "workline:main" || name == "workline:base") {
			t.Errorf("%s: its artifacts hold the stored base, the default branch's job's alone to write: %s", name, paths)
		}
	}
	// Read in a job of its own, with the bot's token, else the job's: the
	// jobs running the project's tools never hold the bot's (#520). It
	// checks nothing out, and runs no tool.
	base := job("workline:base")
	if script := fmt.Sprint(base["script"]); !strings.Contains(script, fromMain) || !strings.Contains(script, `PRIVATE-TOKEN: $WORKLINE_GITLAB_TOKEN`) ||
		!strings.Contains(script, `JOB-TOKEN: $CI_JOB_TOKEN`) || !strings.Contains(script, `kept "$base.json"`) || !strings.Contains(script, "kept commits") {
		t.Errorf("workline:base: does not read the merge base's findings, nor main's last ones, from main's own artifacts: %v", script)
	}
	if script := fmt.Sprint(base["script"]); strings.Contains(script, "workline route") || strings.Contains(script, "golangci") ||
		fmt.Sprint(base["variables"]) != "map[GIT_STRATEGY:none]" {
		t.Errorf("workline:base: runs a tool, or checks out the code, beside the bot's token: %v %v", base["variables"], script)
	}
	if !strings.Contains(fmt.Sprint(job("workline:gates")["script"]), "--gates-kept .workline-kept") {
		t.Errorf("workline:gates: does not read its merge base's findings as kept")
	}
	for _, name := range []string{"workline:gates", "workline:main"} {
		j := job(name)
		if fmt.Sprint(j["needs"]) != "[map[artifacts:true job:workline:base]]" {
			t.Errorf("%s: needs %v, not workline:base's artifact, the stored base", name, j["needs"])
		}
		if text := fmt.Sprint(j["before_script"], j["script"]); strings.Contains(text, "jobs/artifacts") {
			t.Errorf("%s: reads main's artifacts itself, beside the project's tools: %v", name, text)
		}
	}
	issue := job("workline:main:issue")
	if !strings.Contains(fmt.Sprint(issue["script"]), "export GITLAB_TOKEN=$WORKLINE_GITLAB_TOKEN") ||
		!strings.Contains(fmt.Sprint(issue["script"]), "workline apply --main main.json --forge gitlab") || fmt.Sprint(issue["needs"]) != "[workline:main]" {
		t.Errorf("workline:main:issue: does not keep main's run's findings with the bot's token: %v", issue["script"])
	}
	for _, name := range []string{"workline:main", "workline:main:issue"} {
		rules, _ := job(name)["rules"].([]any)
		if len(rules) != 1 || !strings.Contains(fmt.Sprint(rules[0]), `$CI_PIPELINE_SOURCE == "push" && $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH`) {
			t.Errorf("%s: runs on %v, want a push to the default branch alone", name, rules)
		}
	}
}

// GitLab's stored base as the template's own lines run it, a fake curl
// serving main's last artifact: workline:base reads it, and hands it on;
// workline:main carries the commits kept before it, 20 at most, its own
// first, a file it could not read left out; workline:gates is given its
// merge base's alone. The bot's token reads where set, else the job's; a
// file not served leaves none behind, and the gates' job says so.
func TestGitLabStoredBaseCarriedAndPruned(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	lines := func(job string, parts ...string) string {
		j, _ := top[job].(map[string]any)
		var b strings.Builder
		for _, part := range parts {
			items, _ := j[part].([]any)
			for _, l := range items {
				if anchored, ok := l.([]any); ok { // an anchor's lines, flattened as GitLab does
					for _, a := range anchored {
						fmt.Fprintln(&b, a)
					}
					continue
				}
				fmt.Fprintln(&b, l)
			}
		}
		return b.String()
	}
	bin, store := t.TempDir(), t.TempDir()
	// curl -fsSL --header H -o OUT URL: the file URL names, from the
	// store, else curl -f's 22; each header said.
	fake := "#!/bin/sh\nout=; url=; h=\n" +
		"while [ $# -gt 0 ]; do case $1 in -o) out=$2; shift ;; --header) h=$2; shift ;; -*) ;; *) url=$1 ;; esac; shift; done\n" +
		"echo \"$h\" >> \"$STORE/headers\"\n" +
		"name=${url##*/.workline-kept/}; name=${name%%\\?*}\n" +
		"[ -f \"$STORE/$name\" ] || exit 22\ncp \"$STORE/$name\" \"$out\"\n"
	os.WriteFile(filepath.Join(bin, "curl"), []byte(fake), 0o755)
	os.WriteFile(filepath.Join(bin, "workline"), []byte("#!/bin/sh\n[ \"$1 $2\" != \"route main\" ] || echo '{}' > \".workline-kept/$CI_COMMIT_SHA.json\"\n"), 0o755)
	var before []string
	for i := 1; i <= 25; i++ {
		c := fmt.Sprintf("c%02d", i)
		before = append(before, c)
		if c != "c03" { // listed, its file gone
			os.WriteFile(filepath.Join(store, c+".json"), []byte("{}"), 0o644)
		}
	}
	os.WriteFile(filepath.Join(store, "commits"), []byte(strings.Join(before, "\n")+"\n"), 0o644)
	// A pipeline's jobs, one after the other in one folder, each given
	// the next's artifacts as GitLab does; the bot's token, scoped to the
	// environment workline, to those that declare it alone (#520).
	pipeline := func(bot string, jobs []string, env ...string) (string, string) {
		dir := t.TempDir()
		os.Remove(filepath.Join(store, "headers"))
		var log strings.Builder
		for _, job := range jobs {
			cmd := exec.Command("sh", "-ec", lines(job, "before_script", "script"))
			cmd.Dir = dir
			cmd.Env = append([]string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"), "STORE=" + store,
				"CI_API_V4_URL=https://gitlab.invalid/api/v4", "CI_PROJECT_ID=1", "CI_DEFAULT_BRANCH=main", "CI_JOB_TOKEN=job",
				"CI_COMMIT_BEFORE_SHA=c01", "CI_COMMIT_SHA=head", "CI_MERGE_REQUEST_DIFF_BASE_SHA=c02"}, env...)
			if j, _ := top[job].(map[string]any); bot != "" && fmt.Sprint(j["environment"]) == "map[action:prepare name:workline]" {
				cmd.Env = append(cmd.Env, "WORKLINE_GITLAB_TOKEN="+bot)
			}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: %v\n%s", job, err, out)
			}
			log.Write(out)
		}
		return dir, log.String()
	}
	headers := func() string { h, _ := os.ReadFile(filepath.Join(store, "headers")); return strings.TrimSpace(string(h)) }
	dir, _ := pipeline("bot", []string{"workline:base", "workline:main"}, "CI_PIPELINE_SOURCE=push")
	got, _ := os.ReadFile(filepath.Join(dir, ".workline-kept", "commits"))
	want := append([]string{"head", "c01", "c02"}, before[3:19]...)
	if strings.Join(strings.Fields(string(got)), ",") != strings.Join(want, ",") {
		t.Errorf("workline:main keeps the commits %q, want %v", got, want)
	}
	files, _ := filepath.Glob(filepath.Join(dir, ".workline-kept", "*.json"))
	if len(files) != len(want) {
		t.Errorf("workline:main leaves %d files, want one for each of the %d commits listed", len(files), len(want))
	}
	if h := headers(); strings.Contains(h, "JOB-TOKEN") || !strings.Contains(h, "PRIVATE-TOKEN: bot") {
		t.Errorf("with the bot's token, read with %q", h)
	}
	pipeline("", []string{"workline:base", "workline:main"}, "CI_PIPELINE_SOURCE=push")
	if h := headers(); strings.Contains(h, "PRIVATE-TOKEN") || !strings.Contains(h, "JOB-TOKEN: job") {
		t.Errorf("with no bot's token, main's last findings read with %q, want the job's", h)
	}
	dir, _ = pipeline("bot", []string{"workline:base", "workline:gates"}, "CI_PIPELINE_SOURCE=merge_request_event")
	files, _ = filepath.Glob(filepath.Join(dir, ".workline-kept", "*"))
	if len(files) != 1 || filepath.Base(files[0]) != "c02.json" {
		t.Errorf("workline:gates was given %v, want its merge base's alone", files)
	}
	if h := headers(); h != "PRIVATE-TOKEN: bot" {
		t.Errorf("the merge base's findings read with %q, want the bot's token", h)
	}
	if _, log := pipeline("", []string{"workline:base", "workline:gates"}, "CI_PIPELINE_SOURCE=merge_request_event"); headers() != "JOB-TOKEN: job" || strings.Contains(log, "kept no findings") {
		t.Errorf("with no bot's token, read with %q, want the job's\n%s", headers(), log)
	}
	dir, log := pipeline("bot", []string{"workline:base", "workline:gates"}, "CI_PIPELINE_SOURCE=merge_request_event", "CI_MERGE_REQUEST_DIFF_BASE_SHA=c03")
	if files, _ = filepath.Glob(filepath.Join(dir, ".workline-kept", "*")); len(files) != 0 || !strings.Contains(log, "kept no findings for c03: the gates run again at the merge base") {
		t.Errorf("a base not served leaves %v behind, or is not said\n%s", files, log)
	}
	// In a merge train, the target's tip is the base.
	dir, _ = pipeline("bot", []string{"workline:base", "workline:gates"}, "CI_PIPELINE_SOURCE=merge_request_event",
		"CI_MERGE_REQUEST_EVENT_TYPE=merge_train", "CI_MERGE_REQUEST_TARGET_BRANCH_SHA=c04")
	if files, _ = filepath.Glob(filepath.Join(dir, ".workline-kept", "*")); len(files) != 1 || filepath.Base(files[0]) != "c04.json" {
		t.Errorf("a merge train's gates were given %v, want the target's tip's alone", files)
	}
}
