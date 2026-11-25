package evaluation

// The replay set (replay/README.md): real misses, each a case on the
// repository where it happened, at the commits it happened on. Read and
// pointed right here with no agent; played by TestReplay.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func loadReplay(t *testing.T) map[string]*caseFile {
	files, _ := filepath.Glob("replay/*/*.yaml")
	if len(files) == 0 {
		t.Fatal("no replay case (replay/<role>/<case>.yaml)")
	}
	cases := map[string]*caseFile{}
	for _, f := range files {
		c, err := readCase(f)
		if err != nil {
			t.Fatal(err)
		}
		cases[f] = &c
	}
	return cases
}

// TestReplayCasesLoad: every replay case reads, names its role and the
// real commits it starts from, and asks only checks that exist; a
// reviewer case plants a defect unless it is a clean change.
func TestReplayCasesLoad(t *testing.T) {
	for f, c := range loadReplay(t) {
		if !strings.HasPrefix(c.Case, c.Run.Role+"/") || !strings.HasPrefix(f, "replay/"+c.Run.Role+"/") {
			t.Errorf("%s: case %q, role %q, not in its role's folder", f, c.Case, c.Run.Role)
		}
		if c.Given.Clone.Repo == "" || len(c.Given.Clone.Head) < 7 {
			t.Errorf("%s: no given.clone repo and head", f)
		}
		for _, g := range c.Grade {
			for kind := range g {
				if !grades[kind] {
					t.Errorf("%s: unknown check %q", f, kind)
				}
			}
		}
		// A reviewer replay is a miss, a catch to keep (kept-), or a clean
		// change nobody sent back (clean-): only a clean one plants nothing.
		clean := strings.Contains(c.Case, "/clean-")
		if c.Review != nil && (len(c.Review.Defects) == 0) != clean {
			t.Errorf("%s: %d defects; only a clean- case plants none", f, len(c.Review.Defects))
		}
	}
}

// TestReplayCasesPointRight: each place a reviewer replay names holds its
// text at the case's head. It needs the repository's mirror in the cache,
// which a replay run fetches; without it, the case is skipped.
func TestReplayCasesPointRight(t *testing.T) {
	cache, _ := os.UserCacheDir()
	for f, c := range loadReplay(t) {
		if c.Review == nil {
			continue
		}
		t.Run(c.Case, func(t *testing.T) {
			mirror := filepath.Join(cache, "workline", "eval-mirrors", strings.ReplaceAll(c.Given.Clone.Repo, "/", "_")+".git")
			if _, err := os.Stat(mirror); err != nil {
				t.Skip("no mirror of " + c.Given.Clone.Repo + " in the cache: a replay run fetches it")
			}
			_, repo, _, err := build(t, c)
			if err != nil {
				t.Fatal(err)
			}
			var places []place
			for _, d := range append(slices.Clone(c.Review.Defects), c.Review.Outside...) {
				places = append(places, d.At...)
			}
			placesHold(t, f, c, repo, append(places, c.Review.Not...))
		})
	}
}
