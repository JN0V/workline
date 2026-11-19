package documentalist

import (
	"os"
	"os/exec"
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

// Given a doc whose source names a part of a Go file (`x.go#Target`) in a
// small repository, and another doc whose source names a part the file
// does not hold,
// When gardening plans the parts of each doc (planParts),
// Then the first reads exactly the lines of that part, from its first
// line to its last, not the whole file, and the second falls back on
// the whole file.
func TestPlanPartsReadsOnlyTheNamedPart(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q")
	write("x.go", "package x\n"+ // 1
		"\n"+ // 2
		"func Before() int {\n"+ // 3
		"\treturn 1\n"+ // 4
		"}\n"+ // 5
		"\n"+ // 6
		"func Target() int {\n"+ // 7
		"\ta := 2\n"+ // 8
		"\treturn a\n"+ // 9
		"}\n"+ // 10
		"\n"+ // 11
		"func After() int {\n"+ // 12
		"\treturn 3\n"+ // 13
		"}\n") // 14
	write("docs/spec/a.md", "# Spec\n\nIt says what Target returns.\n")
	run("add", "-A")
	run("commit", "-q", "-m", "fixture")

	one := func(source string) partFile {
		t.Helper()
		plans, why, err := planParts(&Doc{Path: "docs/spec/a.md", Sources: []string{source}}, repo, 0)
		if err != nil || why != "" {
			t.Fatalf("%s: planParts: why %q, err %v", source, why, err)
		}
		if len(plans) != 1 || len(plans[0].Files) != 1 {
			t.Fatalf("%s: want one part holding one file, got %+v", source, plans)
		}
		return plans[0].Files[0]
	}

	if got, want := one("x.go#Target"), (partFile{Path: "x.go", From: 7, To: 10, Lines: 14}); got != want {
		t.Errorf("x.go#Target: read %+v, want only the lines of Target: %+v", got, want)
	}
	if got, want := one("x.go#Missing"), (partFile{Path: "x.go", From: 1, To: 14, Lines: 14}); got != want {
		t.Errorf("x.go#Missing: read %+v, want the whole file: %+v", got, want)
	}
}
