package documentalist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specDocs reads, in this repository, the docs under docs/spec/ that
// declare their sources, with the repository's root and the files it
// tracks.
func specDocs(t *testing.T) (string, Tree, []*Doc) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	globs := []string{"docs/spec/**"}
	tree, err := loadTree(root, globs)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := trackedDocs(tree, globs)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) == 0 {
		t.Fatal("no doc under docs/spec/ declares its sources")
	}
	return root, tree, docs
}

// Given the 1.0 release is out,
// When the documentalist looks for the docs made suspect
// (`workline docs --ai none`: a source changed since the doc's `checked`),
// Then no doc under `docs/spec/` is suspect.
func TestSpecDocsAreNotSuspect(t *testing.T) {
	root, tree, docs := specDocs(t)
	for _, d := range docs {
		for _, src := range d.Sources {
			name, path, anchor := splitSource(src)
			if name != "" {
				continue
			}
			checked := d.Checked[""]
			if checked == "" || checked == "HEAD" {
				t.Errorf("%s: no commit recorded in `checked` for %s", d.Path, src)
				continue
			}
			if !tree.exists(path) {
				t.Errorf("%s: names %s as a source, which no longer exists", d.Path, src)
				continue
			}
			a := anchorFor(root, "HEAD", d.Path, checked)
			if a.err != nil || a.cut || !a.ok {
				t.Errorf("%s: cannot tell whether %s changed since %s (err %v, shallow clone %v)", d.Path, src, checked, a.err, a.cut)
				continue
			}
			commits, err := changed(root, a.commit, "HEAD", path, anchor, docNames(tree.Docs[d.Path]))
			if err != nil {
				t.Errorf("%s: %s: %v", d.Path, src, err)
				continue
			}
			if commits != "" {
				t.Errorf("%s is suspect: %s changed since %s:\n%s", d.Path, src, checked, commits)
			}
		}
	}
}

// Given a specification doc split into pages with precise `sources`,
// When gardening runs,
// Then each page is judged against its own sources, with no person
// needed: no page is too long itself to be judged in parts, and the
// sources of each fit the parts a doc may take.
func TestSpecDocsAreJudgedInParts(t *testing.T) {
	root, _, docs := specDocs(t)
	for _, d := range docs {
		_, why, err := partsRoom(d, root)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(why, "too long itself") {
			t.Errorf("%s: %s", d.Path, why)
			continue
		}
		if why != "" { // sources in another repository: not this scenario's
			continue
		}
		plans, why, err := planParts(d, root, 0)
		if err != nil {
			t.Fatal(err)
		}
		if why == "" && len(plans) > partsMaxDefault {
			t.Errorf("%s: its sources take %d parts, more than the %d a doc may take: narrow them", d.Path, len(plans), partsMaxDefault)
		}
	}
}
