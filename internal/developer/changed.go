package developer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// Diffs gives the files an open merge request changes, its diff from the
// merge base with its base branch, and whether they were read (#1256).
// nil reads none: each held issue then holds its Scope's prediction.
type Diffs func(forge.MergeRequest) ([]string, bool)

// Changed reads merge requests' changed files with git in repo, each
// once: its branch in repo itself where the forge keeps branches there
// (keeps, the local forge), else its branch fetched from origin, or a
// fork's head ref. Nothing is written but FETCH_HEAD. A head
// that cannot be read is not: the caller falls back on the prediction.
func Changed(repo string, keeps bool) Diffs {
	if repo == "" {
		return nil
	}
	type read struct {
		files []string
		ok    bool
	}
	seen := map[string]read{}
	return func(m forge.MergeRequest) ([]string, bool) {
		key := m.Branch + "\x00" + m.Head
		if r, ok := seen[key]; ok {
			return r.files, r.ok
		}
		files, ok := changedFiles(repo, keeps, m)
		seen[key] = read{files, ok}
		return files, ok
	}
}

func changedFiles(repo string, keeps bool, m forge.MergeRequest) ([]string, bool) {
	tip, base := "", ""
	switch {
	case keeps && m.Here && m.Branch != "":
		t, err := gitOut(repo, "rev-parse", "-q", "--verify", "refs/heads/"+m.Branch+"^{commit}")
		if err != nil {
			return nil, false
		}
		tip = t
	default:
		src := m.Head // a fork's head, kept on this repository
		if m.Here && m.Branch != "" {
			src = "refs/heads/" + m.Branch
		}
		if src == "" {
			return nil, false
		}
		// Its base fetched with it, as the forge has it now; alone when
		// the base cannot be fetched, the clone's own then read.
		refs := []string{src}
		if m.Base != "" {
			refs = append(refs, "refs/heads/"+m.Base)
		}
		heads, err := fetched(repo, refs)
		if err != nil && len(refs) > 1 {
			heads, err = fetched(repo, refs[:1])
		}
		if err != nil || len(heads) == 0 {
			return nil, false
		}
		tip = heads[0]
		if len(heads) > 1 {
			base = heads[1]
		}
	}
	if base == "" {
		base = localBase(repo, m.Base)
	}
	out, err := gitOut(repo, "diff", "--name-only", "-z", "--no-renames", base+"..."+tip)
	if err != nil {
		return nil, false
	}
	files := []string{}
	for _, f := range strings.Split(out, "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, true
}

// fetched fetches refs from origin into FETCH_HEAD alone, and is the
// commits they are at, in their order.
func fetched(repo string, refs []string) ([]string, error) {
	if _, err := gitOut(repo, append([]string{"fetch", "-q", "--no-tags", "origin"}, refs...)...); err != nil {
		return nil, err
	}
	p, err := gitOut(repo, "rev-parse", "--git-path", "FETCH_HEAD")
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(repo, p)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var heads []string
	for _, l := range strings.Split(string(b), "\n") {
		if f := strings.Fields(l); len(f) > 0 {
			heads = append(heads, f[0])
		}
	}
	return heads, nil
}

// localBase is the clone's base branch: origin's as last fetched, else
// its own branch, else HEAD.
func localBase(repo, base string) string {
	if base != "" {
		for _, r := range []string{"refs/remotes/origin/" + base, "refs/heads/" + base} {
			if _, err := gitOut(repo, "rev-parse", "-q", "--verify", r+"^{commit}"); err == nil {
				return r
			}
		}
	}
	return "HEAD"
}
