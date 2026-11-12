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

// The tests a tree holds, by their ids, carried between CI jobs (#1407):
// the tester's guard reads the base's rather than run the whole suite
// again, when the tester's red proof on the default branch, or the
// merge request's own run before, found them. Only from where no other
// branch can write: GitHub's cache, a pull request's own or the default
// branch's; GitLab's artifacts of the default branch's or the merge
// request's own pipelines, never GitLab's cache, which every branch
// shares. Never kept from a run of the developer's code.

const testIDsPath = ".git/workline/test-ids"

func TestGitHubTestIDsCarriedByTheCache(t *testing.T) {
	type step struct {
		Name string         `yaml:"name"`
		ID   string         `yaml:"id"`
		Uses string         `yaml:"uses"`
		With map[string]any `yaml:"with"`
		If   string         `yaml:"if"`
		Run  string         `yaml:"run"`
	}
	type job struct {
		Steps []step `yaml:"steps"`
	}
	type workflow struct {
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
	// The cache steps of job that run in ctx: restored, saved.
	cached := func(j job, ctx map[string]any) (restored, saved []step) {
		for _, s := range j.Steps {
			if fmt.Sprint(s.With["path"]) != testIDsPath || s.If != "" && !truthy(eval(t, s.If, ctx)) {
				continue
			}
			switch {
			case strings.HasPrefix(s.Uses, "actions/cache/restore@"):
				restored = append(restored, s)
			case strings.HasPrefix(s.Uses, "actions/cache/save@"):
				saved = append(saved, s)
			default:
				t.Errorf("%q keeps the test ids with %s: restore and save apart, never the whole job's end", s.Name, s.Uses)
			}
		}
		return restored, saved
	}
	ofStep := func(j job, id string) step {
		for _, s := range j.Steps {
			if s.ID == id {
				return s
			}
		}
		return step{}
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		// A pull request reads them by a prefix, its own newest else the
		// default branch's, and keeps what is new under a key of its run.
		judge := read(dir + "workline.yml").Jobs["judge"]
		pr := map[string]any{"steps.found.outputs.new": "true", "github.run_id": "11", "github.run_attempt": "1"}
		for k, v := range pullRequestPush {
			pr[k] = v
		}
		restored, saved := cached(judge, pr)
		if len(restored) != 1 || len(saved) != 1 {
			t.Fatalf("%sworkline.yml: a pull request's judge restores %d and saves %d test-id caches, want one each", dir, len(restored), len(saved))
		}
		if got := fmt.Sprint(restored[0].With["restore-keys"]); got != "workline-test-ids-" {
			t.Errorf("%sworkline.yml: the test ids read by %q, want the prefix workline-test-ids-", dir, got)
		}
		if k := expand(t, fmt.Sprint(saved[0].With["key"]), pr); k != "workline-test-ids-11-1" || expand(t, fmt.Sprint(restored[0].With["key"]), pr) != k {
			t.Errorf("%sworkline.yml: the test ids kept as %q, want the run's own key", dir, k)
		}
		if !strings.Contains(ofStep(judge, "found").Run, `= "$READ"`) {
			t.Errorf("%sworkline.yml: the test ids kept even when none is new", dir)
		}
		// A queue's group keeps none: its own scope, read by nothing.
		group := map[string]any{}
		for k, v := range mergeGroup {
			group[k] = v
		}
		if r, s := cached(judge, group); len(r)+len(s) != 0 || truthy(eval(t, ofStep(judge, "found").If, group)) {
			t.Errorf("%sworkline.yml: a queue's group reads or keeps test ids", dir)
		}

		// The tester's red proof on the default branch keeps them there,
		// only when the tester ran and the developer did not; a person's
		// label's run may not write the cache (#853).
		dev := read(dir + "workline-develop.yml").Jobs["judge"]
		dispatch := map[string]any{"github.event_name": "workflow_dispatch", "steps.due.outputs.due": "true", "steps.found.outputs.new": "true",
			"github.run_id": "12", "github.run_attempt": "1"}
		restored, saved = cached(dev, dispatch)
		if len(restored) != 1 || len(saved) != 1 || fmt.Sprint(restored[0].With["restore-keys"]) != "workline-test-ids-" {
			t.Errorf("%sworkline-develop.yml: the issue's run restores %d and saves %d test-id caches, want one each, read by the prefix", dir, len(restored), len(saved))
		}
		found := ofStep(dev, "found")
		if !strings.Contains(found.Run, `.name == "tester"`) || !strings.Contains(found.Run, `.name != "developer"`) {
			t.Errorf("%sworkline-develop.yml: the test ids kept whichever role ran:\n%s", dir, found.Run)
		}
		label := map[string]any{"github.event_name": "issues", "steps.due.outputs.due": "true", "steps.line.outcome": "success"}
		if truthy(eval(t, found.If, label)) {
			t.Errorf("%sworkline-develop.yml: a person's label's run tries to keep the test ids, which it may not (#853)", dir)
		}
	}
}

// On GitLab: workline:base reads the ids from the default branch's
// workline:develop:judge and the merge request's own workline:judge, only
// names of the engine's form; workline:judge lays them where the engine
// reads them and keeps what its runs found; workline:develop:judge keeps
// its own only when the tester ran, else carries the job before's.
func TestGitLabTestIDsCarriedByArtifacts(t *testing.T) {
	top := readGitLab(t)
	for name, job := range top {
		if j, ok := job.(map[string]any); ok {
			if _, cache := j["cache"]; cache {
				t.Errorf("GitLab: %s uses a cache, which every branch writes", name)
			}
		}
	}
	lines := func(job string, parts ...string) string {
		j, _ := top[job].(map[string]any)
		var b strings.Builder
		for _, part := range parts {
			items, _ := j[part].([]any)
			for _, l := range items {
				if anchored, ok := l.([]any); ok {
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
	if j, _ := top["workline:judge"].(map[string]any); !strings.Contains(fmt.Sprint(j["needs"]), "job:workline:base") {
		t.Errorf("GitLab: workline:judge is not given workline:base's artifacts: %v", j["needs"])
	}
	key := strings.Repeat("a", 64) + ".json"
	served := strings.Repeat("b", 64) + ".json"
	bin, store := t.TempDir(), t.TempDir()
	// curl: the file after .workline-test-ids/ in the store under the ref
	// and job asked; -o or standard output; each URL logged.
	fake := "#!/bin/sh\nout=; url=\n" +
		"while [ $# -gt 0 ]; do case $1 in -o) out=$2; shift ;; --header) shift ;; -*) ;; *) url=$1 ;; esac; shift; done\n" +
		"echo \"$url\" >> \"$STORE/urls\"\n" +
		"case $url in *.workline-test-ids/*) ;; *) exit 22 ;; esac\n" +
		"ref=${url#*/jobs/artifacts/}; ref=${ref%%/raw/*}; job=${url#*job=}; job=${job%%&*}\n" +
		"name=${url##*/.workline-test-ids/}; name=${name%%\\?*}\n" +
		"f=\"$STORE/$(echo \"$ref\" | tr / _)/$job/$name\"\n" +
		"[ -f \"$f\" ] || exit 22\nif [ -n \"$out\" ]; then cp \"$f\" \"$out\"; else cat \"$f\"; fi\n"
	if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	// workline: lists the ids it was given, then finds one tree more.
	os.WriteFile(filepath.Join(bin, "workline"), []byte("#!/bin/sh\n[ \"$1 $2\" = \"route merge-request\" ] || [ \"$1 $2\" = \"route issue\" ] || exit 0\n"+
		"ls .git/workline/test-ids > given 2>/dev/null; echo '{}' > \".git/workline/test-ids/$FOUND\"; echo \"$LINE\"\n"), 0o755)
	serve := func(ref, job string, files ...string) {
		d := filepath.Join(store, strings.ReplaceAll(ref, "/", "_"), job)
		os.MkdirAll(d, 0o755)
		os.WriteFile(filepath.Join(d, "index"), []byte(strings.Join(append(files, "../../evil.json", "x.json"), "\n")+"\n"), 0o644)
		for _, f := range files {
			os.WriteFile(filepath.Join(d, f), []byte("{}"), 0o644)
		}
		os.WriteFile(filepath.Join(d, "x.json"), []byte("{}"), 0o644)
	}
	serve("main", "workline:develop:judge", key)
	serve("refs/merge-requests/7/head", "workline:judge", served)
	run := func(dir, job string, env ...string) string {
		script := lines(job, "script")
		if job == "workline:develop:judge" { // from the ids on: the agent and the project's setup are not this test's
			script = script[strings.Index(script, "ids() {"):]
		}
		cmd := exec.Command("sh", "-ec", script)
		cmd.Dir = dir
		cmd.Env = append([]string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"), "STORE=" + store,
			"CI_API_V4_URL=https://gitlab.invalid/api/v4", "CI_PROJECT_ID=1", "CI_DEFAULT_BRANCH=main", "CI_JOB_TOKEN=job",
			"WORKLINE_GITLAB_TOKEN=bot", "CI_MERGE_REQUEST_IID=7", "CI_MERGE_REQUEST_DIFF_BASE_SHA=c02", "FOUND=" + strings.Repeat("c", 64) + ".json"}, env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", job, err, out)
		}
		return string(out)
	}
	ls := func(dir string) []string {
		var names []string
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			names = append(names, e.Name())
		}
		slices.Sort(names)
		return names
	}

	// A merge request's pipeline: base, then judge in the same folder, as
	// GitLab hands base's artifacts on.
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	run(dir, "workline:base", "CI_PIPELINE_SOURCE=merge_request_event")
	urls, _ := os.ReadFile(filepath.Join(store, "urls"))
	if !strings.Contains(string(urls), "/jobs/artifacts/main/raw/.workline-test-ids/index?job=workline:develop:judge") ||
		!strings.Contains(string(urls), "/jobs/artifacts/refs/merge-requests/7/head/raw/.workline-test-ids/index?job=workline:judge") {
		t.Errorf("workline:base reads the test ids from:\n%s", urls)
	}
	if got := ls(filepath.Join(dir, ".workline-test-ids")); !slices.Equal(got, []string{key, served}) {
		t.Errorf("workline:base hands on %v, want the two trees served, no other name", got)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "evil.json")); err == nil {
		t.Errorf("a name out of the folder was written")
	}
	run(dir, "workline:judge", `LINE={"steps":[{"name":"tester"}]}`)
	if given, _ := os.ReadFile(filepath.Join(dir, "given")); strings.Join(strings.Fields(string(given)), ",") != key+","+served {
		t.Errorf("the engine was given %q in .git/workline/test-ids, want the two trees read", given)
	}
	if got := ls(filepath.Join(dir, ".workline-test-ids")); len(got) != 4 || got[3] != "index" {
		t.Errorf("workline:judge keeps %v, want the three trees and their index", got)
	}

	// A merge train reads none: it runs no judge.
	os.Remove(filepath.Join(store, "urls"))
	run(t.TempDir(), "workline:base", "CI_PIPELINE_SOURCE=merge_request_event", "CI_MERGE_REQUEST_EVENT_TYPE=merge_train")
	if urls, _ := os.ReadFile(filepath.Join(store, "urls")); strings.Contains(string(urls), "test-ids") {
		t.Errorf("a merge train reads test ids:\n%s", urls)
	}

	// The tester's run on the default branch keeps what it found; the
	// developer's carries the job before's, never its own.
	for _, c := range []struct {
		line string
		want int
	}{
		{`{"steps":[{"name":"tester"}]}`, 2},
		{`{"steps":[{"name":"developer"}]}`, 1},
	} {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
		run(dir, "workline:develop:judge", "LINE="+c.line, "ai=none")
		got := ls(filepath.Join(dir, ".workline-test-ids"))
		if len(got) != c.want+1 || got[0] != key {
			t.Errorf("after %s, workline:develop:judge keeps %v, want %d trees and their index", c.line, got, c.want)
		}
	}
}
