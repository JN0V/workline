package developer

// A test the tester rewrote (ADR-0055, #939): the developer objects to a
// frozen acceptance test, and the tester rewrites it on its Draft's
// branch, its own commit, its record naming the new blob. The developer's
// pull request started from the Draft before it and holds the old test:
// its next round takes the rewrite into its branch, as the tester's record
// names it, its agent's worktree holding it already, and proves it with
// the rest. Its record then starts from the Draft's new head.

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/tester"
)

// testerHolds is who asks a rewritten test into a round: the tester.
const testerHolds = "the tester"

// rewritesAsked are the acceptance tests of a's issue the tester's
// branch holds otherwise than the head, as the tester's record on its
// Draft names them: each a round asks (its Detail the blob). Git first:
// the forge is read only when the branches differ. A blob the record
// does not name is never asked, so it makes no round due again and
// again. None when the head or the tester's branch is not known in repo.
func rewritesAsked(f forge.Forge, repo string, a Act, head string) ([]Asked, error) {
	if head == "" || len(a.Acceptance) == 0 {
		return nil, nil
	}
	tip := testerTip(repo, a.Issue)
	if tip == "" {
		return nil, nil
	}
	var out []Asked
	for _, t := range a.Acceptance {
		blob, _ := gitOut(repo, "rev-parse", "-q", "--verify", tip+":"+t)
		if blob == "" {
			continue
		}
		if have, _ := gitOut(repo, "rev-parse", "-q", "--verify", head+":"+t); have == blob {
			continue
		}
		out = append(out, Asked{Key: "tester|" + t + "|" + blob, Where: t, Holds: testerHolds, Detail: blob,
			Text: "rewritten on an objection, on its Draft"})
	}
	if len(out) == 0 {
		return nil, nil
	}
	files, err := testerFiles(f, a.Issue)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(out, func(x Asked) bool { return files[x.Where] != x.Detail }), nil
}

// testerFiles are the frozen tests the record on the tester's open Draft
// for issue names, each its blob; none without a Draft or a record read.
func testerFiles(f forge.Forge, issue int) (map[string]string, error) {
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return nil, err
	}
	for _, mr := range mrs {
		if testerIssue(mr.Branch) != issue {
			continue
		}
		r, ok, err := tester.ReadRecord(f, mr.ID)
		if err != nil || !ok {
			return nil, err
		}
		return r.Files, nil
	}
	return nil, nil
}

// testerTip is the commit of the tester's branch for issue, as fetched
// from origin, else local; "" when neither is there.
func testerTip(repo string, issue int) string {
	b := fmt.Sprintf("%s%d", tester.TestsBranch, issue)
	for _, ref := range []string{"refs/remotes/origin/" + b, "refs/heads/" + b} {
		if h, err := gitOut(repo, "rev-parse", "-q", "--verify", ref+"^{commit}"); err == nil && h != "" {
			return h
		}
	}
	return ""
}

// takeRewrites keeps, of the round's asks, a rewrite whose blob the
// tester's record names (b.Blobs): the frozen test is the record's, never
// a branch's word alone. It returns the asks kept and the tests taken,
// each its blob.
func takeRewrites(asked []Asked, b Bounds) (kept []Asked, taken map[string]string) {
	taken = map[string]string{}
	for _, x := range asked {
		if x.Holds != testerHolds {
			kept = append(kept, x)
			continue
		}
		if b.Blobs[x.Where] == x.Detail {
			taken[x.Where] = x.Detail
			kept = append(kept, x)
		}
	}
	return kept, taken
}

// withRewrites is a commit on at whose tree holds the tests taken, each
// its blob: where its agent's worktree starts, so that it reads them as
// rewritten and its diff holds its own code alone. A commit of no
// branch: the round pushes the tests itself (rewritePatches).
func withRewrites(repo, at string, taken map[string]string) (string, error) {
	tmp, err := os.MkdirTemp("", "workline-rewrite-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index"),
		"GIT_AUTHOR_NAME=workline", "GIT_AUTHOR_EMAIL=workline@localhost",
		"GIT_COMMITTER_NAME=workline", "GIT_COMMITTER_EMAIL=workline@localhost")
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := run("read-tree", at); err != nil {
		return "", err
	}
	for _, t := range slices.Sorted(maps.Keys(taken)) {
		mode := "100644" // its mode on the branch kept: an executable test stays one
		if out, _ := run("ls-tree", at, "--", t); strings.HasPrefix(out, "100755 ") {
			mode = "100755"
		}
		if _, err := run("update-index", "--add", "--cacheinfo", mode+","+taken[t]+","+t); err != nil {
			return "", err
		}
	}
	tree, err := run("write-tree")
	if err != nil {
		return "", err
	}
	return run("commit-tree", tree, "-p", at, "-m", "the tester's rewritten tests")
}

// rewritePatches are the tests taken, each whole as the tester's record
// names it, for the round to push before its agent's code.
func rewritePatches(repo string, taken map[string]string) ([]intent.Intention, error) {
	var out []intent.Intention
	for _, t := range slices.Sorted(maps.Keys(taken)) {
		data, err := exec.Command("git", "-C", repo, "cat-file", "blob", taken[t]).Output()
		if err != nil {
			return nil, fmt.Errorf("the tester's rewrite of %s, blob %s: %w", t, shortSHA(taken[t]), err)
		}
		out = append(out, intent.Intention{Kind: "patch", Value: map[string]any{"file": t, "content": string(data)}})
	}
	return out, nil
}

// rewriteLine is a rewrite asked, as the round's task and comment say it.
func rewriteLine(a Asked) string {
	return fmt.Sprintf("the tester rewrote `%s` on an objection, on its Draft: taken into this branch", a.Where)
}
