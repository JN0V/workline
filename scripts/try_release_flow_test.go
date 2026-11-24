package scripts

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// forgeGh is a gh that answers from the JSON files of a folder: pr-<state>.json
// for `pr list`, checks.json, pr-<n>.json, issues.json, issue-<n>.json; each
// through the real jq, as gh's --jq does. It logs every call.
const forgeGh = `#!/usr/bin/env bash
dir=$(dirname "$0")
echo "$*" >> "$dir/gh.log"
jqx=. state= dl= pos=
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
  case ${args[i]} in
    --jq) jqx=${args[i+1]} ;;
    --state) state=${args[i+1]} ;;
    -D) dl=${args[i+1]} ;;
    --) pos=${args[i+1]} ;;
  esac
done
answer() { if [ -e "$dir/$1" ]; then jq -r "$jqx" "$dir/$1"; else jq -r "$jqx" <<<'[]'; fi; }
case "$1 $2" in
  "api repos/JN0V/TraceGlass-trial/commits/main") echo 0123456789abcdef ;;
  "api -X") ;;
  "api user") echo someone ;;
  "release download") cp "$dir"/rel/* "$dl"/ ;;
  "pr list") answer "pr-$state.json" ;;
  "pr checks") answer checks.json ;;
  "pr view") answer "pr-$pos.json" ;;
  "pr create") echo "https://github.com/JN0V/TraceGlass-trial/pull/7" ;;
  "issue list") answer issues.json ;;
  "issue create") echo "https://github.com/JN0V/TraceGlass-trial/issues/9" ;;
  "issue view") answer "issue-$pos.json" ;;
  "workflow run") ;;
  *) echo "unexpected gh $*" >&2; exit 1 ;;
esac
`

type forge struct {
	t          *testing.T
	gh, gitURL string
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitEnv() []string {
	return append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
}

// repo makes a bare repository at <gitURL>/<name>.git from files, tagging
// each commit given.
func repo(t *testing.T, base, name string, commits ...map[string]string) {
	t.Helper()
	work := t.TempDir()
	git(t, work, "init", "-q", "-b", "main")
	for _, files := range commits {
		tag := files["tag"]
		for p, c := range files {
			if p != "tag" {
				write(t, filepath.Join(work, p), c)
			}
		}
		git(t, work, "add", "-A")
		git(t, work, "commit", "-q", "-m", "c")
		if tag != "" {
			git(t, work, "tag", tag)
		}
	}
	git(t, base, "clone", "-q", "--bare", work, filepath.Join(base, name+".git"))
}

func template(version string) string {
	return "name: workline\nenv:\n  WORKLINE_VERSION: " + version + "   # x-release-please-version\njobs: {}\n"
}

// newForge: the release v9.9.9 (a fake binary, its checksum), workline's
// templates at v9.9.8 and v9.9.9, the copy pinned to v9.9.8 with a line of
// its own.
func newForge(t *testing.T) forge {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not on the PATH")
	}
	bin, base := t.TempDir(), t.TempDir()
	write(t, filepath.Join(bin, "gh"), forgeGh)
	asset := "workline_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	archive := filepath.Join(bin, "rel", asset)
	write(t, archive, "")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(f)
	gz := tar.NewWriter(zw)
	body := "#!/bin/sh\ncase $1 in version) echo v9.9.9 ;; doctor) echo '✓ all set' ;; esac\n"
	if err := gz.WriteHeader(&tar.Header{Name: "workline", Mode: 0o755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	_, _ = gz.Write([]byte(body))
	for _, c := range []func() error{gz.Close, zw.Close, f.Close} {
		if err := c(); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(archive)
	sum := sha256.Sum256(data)
	write(t, filepath.Join(bin, "rel", "checksums.txt"), hex.EncodeToString(sum[:])+"  "+asset+"\n")

	write(t, filepath.Join(base, "JN0V", ".keep"), "")
	repo(t, filepath.Join(base, "JN0V"), "workline",
		map[string]string{"defaults/ci/github/workline.yml": template("v9.9.8"), "defaults/ci/github/optional.yml": "name: optional\n", "tag": "v9.9.8"},
		map[string]string{"defaults/ci/github/workline.yml": template("v9.9.9"), "defaults/ci/github/new.yml": "name: new\n", "tag": "v9.9.9"})
	own := strings.Replace(template("v9.9.8"), "jobs: {}", "jobs: {}\n# the copy's own line", 1)
	repo(t, filepath.Join(base, "JN0V"), "TraceGlass-trial",
		map[string]string{".github/workflows/workline.yml": own})
	return forge{t: t, gh: bin, gitURL: "file://" + base}
}

func (f forge) set(name, json string) { write(f.t, filepath.Join(f.gh, name), json) }

func (f forge) run(args ...string) (string, int) {
	f.t.Helper()
	return f.runEnv(nil, args...)
}

// runEnv runs the script with env added to the forge's.
func (f forge) runEnv(env []string, args ...string) (string, int) {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", append([]string{"try-release.sh", "--minutes", "0"}, args...)...)
	cmd.Env = append(gitEnv(), "PATH="+f.gh+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TRY_GIT_URL="+f.gitURL, "TRY_POLL=0", "TRY_LABEL_MINUTES=0", "TRY_REPO=", "TRY_WORKLINE=")
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	if e, ok := err.(*exec.ExitError); ok {
		return string(out), e.ExitCode()
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return string(out), 0
}

func (f forge) calls() string { return calls(f.t, filepath.Join(f.gh, "gh.log")) }

const lockDeleted = "api -X DELETE repos/JN0V/TraceGlass-trial/git/refs/tags/workline-try-lock"

func TestTryReleaseNoMergedUpgradeWaitsForThePersonsLabel(t *testing.T) {
	f := newForge(t)
	f.set("pr-open.json", `[{"number":7,"url":"https://github.com/JN0V/TraceGlass-trial/pull/7"}]`)
	f.set("checks.json", `[{"bucket":"pass"},{"bucket":"skipping"}]`)
	f.set("pr-7.json", `{"mergeable":"MERGEABLE","state":"OPEN"}`)
	out, code := f.run("v9.9.9")
	if code != 2 {
		t.Fatalf("exit %d, want 2 (waiting on the person):\n%s", code, out)
	}
	for _, want := range []string{"checksum ok, says v9.9.9", "doctor: clean", "checks: green; mergeable", "put the label workline:accepted on https://github.com/JN0V/TraceGlass-trial/pull/7"} {
		if !strings.Contains(out, want) {
			t.Errorf("the run misses %q:\n%s", want, out)
		}
	}
	c := f.calls()
	if strings.Contains(c, "issue create") || strings.Contains(c, "label") && !strings.Contains(c, "labels") {
		t.Errorf("an issue opened or a label was put before the upgrade merged:\n%s", c)
	}
	if !strings.Contains(c, lockDeleted) {
		t.Errorf("the lock was not released at exit:\n%s", c)
	}
	// The branch pushed: the release's pin, the copy's own line kept.
	show := exec.Command("git", "--git-dir", strings.TrimPrefix(f.gitURL, "file://")+"/JN0V/TraceGlass-trial.git",
		"show", "refs/heads/workline-v9.9.9:.github/workflows/workline.yml")
	got, err := show.Output()
	if err != nil {
		t.Fatalf("no branch workline-v9.9.9 pushed to the copy: %v", err)
	}
	if !strings.Contains(string(got), "WORKLINE_VERSION: v9.9.9") || !strings.Contains(string(got), "# the copy's own line") {
		t.Errorf("the upgrade does not take the release's pin and keep the copy's line:\n%s", got)
	}
	ls, err := exec.Command("git", "--git-dir", strings.TrimPrefix(f.gitURL, "file://")+"/JN0V/TraceGlass-trial.git",
		"ls-tree", "--name-only", "refs/heads/workline-v9.9.9", ".github/workflows/").Output()
	if err != nil {
		t.Fatal(err)
	}
	if files := string(ls); !strings.Contains(files, "new.yml") || strings.Contains(files, "optional.yml") {
		t.Errorf("want the workflow new in the release, not the one the copy left out:\n%s", files)
	}
}

func TestTryReleaseGreenWhenTheCannedIssueMergesByPolicy(t *testing.T) {
	f := newForge(t)
	f.set("pr-merged.json", `[{"number":5,"url":"https://github.com/JN0V/TraceGlass-trial/pull/5"}]`)
	// The first canned bug is fixed already; the second is opened.
	f.set("issues.json", `[{"number":3,"title":"An outline of 65536 by 65536 pixels with no pixels crashes instead of being refused","state":"CLOSED","stateReason":"COMPLETED","body":""}]`)
	f.set("issue-9.json", `{"state":"CLOSED","closedByPullRequestsReferences":[{"number":10},{"number":11}]}`)
	f.set("pr-10.json", `{"url":"https://github.com/JN0V/TraceGlass-trial/pull/10","state":"CLOSED","labels":[]}`)
	f.set("pr-11.json", `{"url":"https://github.com/JN0V/TraceGlass-trial/pull/11","state":"MERGED","labels":[{"name":"bug"}]}`)
	out, code := f.run("v9.9.9")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	for _, want := range []string{"green: v9.9.9", "upgrade: https://github.com/JN0V/TraceGlass-trial/pull/5", "merged: https://github.com/JN0V/TraceGlass-trial/pull/11"} {
		if !strings.Contains(out, want) {
			t.Errorf("the verdict misses %q:\n%s", want, out)
		}
	}
	c := f.calls()
	if !strings.Contains(c, "issue create -R JN0V/TraceGlass-trial --title A matrix of 8 numbers crashes the composition with an index error --label workline:needs-tests --label bug") {
		t.Errorf("the second canned issue was not opened test-first with its kind:\n%s", c)
	}
	if strings.Contains(c, "workflow run") {
		t.Errorf("the merge workflow was dispatched for an issue closed by a merge:\n%s", c)
	}
}

func TestTryReleaseRedWhenTheFixMergedOnAPersonsLabel(t *testing.T) {
	f := newForge(t)
	f.set("pr-merged.json", `[{"number":5,"url":"https://github.com/JN0V/TraceGlass-trial/pull/5"}]`)
	f.set("issues.json", `[{"number":8,"title":"An outline of 65536 by 65536 pixels with no pixels crashes instead of being refused","state":"CLOSED","stateReason":"COMPLETED","body":"<!-- workline-try: v9.9.9 -->"}]`)
	f.set("issue-8.json", `{"state":"CLOSED","closedByPullRequestsReferences":[{"number":12}]}`)
	f.set("pr-12.json", `{"url":"https://github.com/JN0V/TraceGlass-trial/pull/12","state":"MERGED","labels":[{"name":"workline:accepted"}]}`)
	out, code := f.run("v9.9.9")
	if code != 1 || !strings.Contains(out, "merged on a person's label, not by policy") {
		t.Fatalf("exit %d, want 1 and a red for the label:\n%s", code, out)
	}
	if c := f.calls(); strings.Contains(c, "issue create") {
		t.Errorf("a run after this version's issue closed opened another:\n%s", c)
	}
}

// A rerun with the upgrade branch already holding the rebuilt tree pushes
// nothing: a git on the PATH logs each call before the real one runs.
func TestTryReleaseRerunWithNothingChangedPushesNothing(t *testing.T) {
	f := newForge(t)
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	gitLog := filepath.Join(f.gh, "git.log")
	write(t, filepath.Join(f.gh, "git"), "#!/usr/bin/env bash\necho \"$*\" >> \""+gitLog+"\"\nexec \""+real+"\" \"$@\"\n")
	f.set("pr-open.json", `[{"number":7,"url":"https://github.com/JN0V/TraceGlass-trial/pull/7"}]`)
	f.set("checks.json", `[{"bucket":"pass"}]`)
	f.set("pr-7.json", `{"mergeable":"MERGEABLE","state":"OPEN"}`)
	if out, code := f.run("v9.9.9"); code != 2 {
		t.Fatalf("first run: exit %d, want 2:\n%s", code, out)
	}
	if !strings.Contains(calls(t, gitLog), "push") {
		t.Fatalf("the first run pushed no upgrade branch:\n%s", calls(t, gitLog))
	}
	if err := os.Remove(gitLog); err != nil {
		t.Fatal(err)
	}
	out, code := f.run("v9.9.9")
	if code != 2 {
		t.Fatalf("rerun: exit %d, want 2:\n%s", code, out)
	}
	if g := calls(t, gitLog); strings.Contains(g, "push") {
		t.Errorf("a rerun with nothing changed pushed:\n%s", g)
	}
	if c := f.calls(); strings.Contains(c, "pr create") {
		t.Errorf("a rerun opened a second upgrade pull request:\n%s", c)
	}
}

// A command's own failure (tar's 2, git's 128) is red, exit 1, never read
// as "waiting" (2); the lock is released all the same.
func TestTryReleaseAFailingCommandIsRed(t *testing.T) {
	t.Run("tar", func(t *testing.T) {
		f := newForge(t)
		asset := "workline_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
		garbage := "not an archive\n"
		sum := sha256.Sum256([]byte(garbage))
		f.set(filepath.Join("rel", asset), garbage)
		f.set(filepath.Join("rel", "checksums.txt"), hex.EncodeToString(sum[:])+"  "+asset+"\n")
		out, code := f.run("v9.9.9")
		if code != 1 || !strings.Contains(out, "red: a command failed (exit 2)") {
			t.Errorf("exit %d, want 1 and a red naming tar's code:\n%s", code, out)
		}
		if c := f.calls(); !strings.Contains(c, lockDeleted) {
			t.Errorf("the lock was not released after a red:\n%s", c)
		}
	})
	t.Run("git clone", func(t *testing.T) {
		f := newForge(t)
		f.gitURL = "file:///nonexistent-try-release"
		out, code := f.run("v9.9.9")
		if code != 1 || !strings.Contains(out, "red: a command failed (exit 128)") {
			t.Errorf("exit %d, want 1 and a red naming git's code:\n%s", code, out)
		}
		if c := f.calls(); !strings.Contains(c, lockDeleted) {
			t.Errorf("the lock was not released after a red:\n%s", c)
		}
	})
}

// The fix green and accepted by policy, not merged: the script dispatches
// the copy's merge workflow once, and never labels. A fix not accepted
// yet, or accepted on a person's label, is left as it is.
func TestTryReleaseDispatchesTheMergeOnceForAFixAcceptedByPolicy(t *testing.T) {
	green := `{"__typename":"CheckRun","name":"test","conclusion":"SUCCESS"}`
	accepted := `{"__typename":"StatusContext","context":"workline/accepted","state":"SUCCESS"}`
	for _, c := range []struct {
		name, pr string
		want     bool
	}{
		{"accepted by policy", `{"state":"OPEN","isDraft":false,"labels":[{"name":"bug"}],"statusCheckRollup":[` + green + `,` + accepted + `]}`, true},
		{"not accepted yet", `{"state":"OPEN","isDraft":false,"labels":[],"statusCheckRollup":[` + green + `]}`, false},
		{"a check pending", `{"state":"OPEN","isDraft":false,"labels":[],"statusCheckRollup":[{"__typename":"CheckRun","name":"test","conclusion":""},` + accepted + `]}`, false},
		{"a person's label", `{"state":"OPEN","isDraft":false,"labels":[{"name":"workline:accepted"}],"statusCheckRollup":[` + green + `,` + accepted + `]}`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newForge(t)
			f.set("pr-merged.json", `[{"number":5,"url":"https://github.com/JN0V/TraceGlass-trial/pull/5"}]`)
			f.set("issues.json", `[{"number":8,"title":"An outline of 65536 by 65536 pixels with no pixels crashes instead of being refused","state":"OPEN","stateReason":null,"body":"<!-- workline-try: v9.9.9 -->"}]`)
			f.set("issue-8.json", `{"state":"OPEN","closedByPullRequestsReferences":[{"number":10}]}`)
			f.set("pr-10.json", c.pr)
			out, code := f.runEnv([]string{"TRY_DISPATCH_MINUTES=0"}, "v9.9.9")
			if code != 1 || !strings.Contains(out, "no pull request merged for the canned issue") {
				t.Fatalf("exit %d, want 1 when the time runs out:\n%s", code, out)
			}
			calls := f.calls()
			if got := strings.Count(calls, "workflow run workline-merge.yml -R JN0V/TraceGlass-trial"); got != map[bool]int{true: 1, false: 0}[c.want] {
				t.Errorf("dispatched %d times, want it %v:\n%s", got, c.want, calls)
			}
			if said := strings.Contains(out, "dispatched the copy's workline merge workflow once"); said != c.want {
				t.Errorf("the output says the dispatch %v, want %v:\n%s", said, c.want, out)
			}
			if strings.Contains(calls, "edit") || strings.Contains(calls, "merge ") {
				t.Errorf("the script labelled or merged:\n%s", calls)
			}
		})
	}
}
