package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/devmark"
)

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
	p, c := prove(repo, tip, nil, nil, nil, Settings{ProofSeconds: 60})
	if p.Result != devmark.Red || !slices.Equal(p.Red, []string{"lib.ok"}) || c != nil {
		t.Fatalf("proof = %+v, cap = %+v; want red, lib.ok red, no cap", p, c)
	}
	if f := proofFindings("#1", p, nil); len(f) != 1 || f[0].Rule != "proof-red" || !strings.Contains(f[0].Message, "lib.ok") {
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
	p, _ := prove(repo, tip, nil, nil, nil, Settings{ProofSeconds: 60})
	if p.Result == devmark.Green {
		t.Fatalf("proof = %+v; want not green on a merge that conflicts", p)
	}
	for _, want := range []string{"conflict", "main", "lib.sh"} {
		if !strings.Contains(p.Why, want) {
			t.Errorf("why = %q, want it to say the conflict with main in lib.sh (missing %q)", p.Why, want)
		}
	}
}
