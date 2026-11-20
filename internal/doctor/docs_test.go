package doctor

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// The doctor says what the role keeping the docs answers: the docs that
// declare their sources and those that say nothing, and the files read
// as docs that it does not read.
func TestDocsSaysWhatTheDocsRoleAnswers(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	o := Options{Repo: t.TempDir(), RolesDir: filepath.Join(filepath.Dir(here), "..", "..", "roles")}
	fixtureDocs.tracked, fixtureDocs.none, fixtureDocs.untracked = []string{"a.md"}, []string{"b.md"}, []string{"c.md"}
	fixtureDocs.unread = "1 file read as a doc that the documentalist does not read"
	defer func() {
		fixtureDocs.tracked, fixtureDocs.none, fixtureDocs.untracked, fixtureDocs.unread = nil, nil, nil, ""
	}()
	var r Report
	docs(&r, o, &role.ProjectConfig{})
	rules := map[string]Check{}
	for _, c := range r.Checks {
		rules[c.Rule] = c
	}
	if c, ok := rules["docs-untracked"]; !ok || c.Level != Warn {
		t.Errorf("one doc of three saying nothing: %+v", r.Checks)
	}
	if c, ok := rules["docs-not-read"]; !ok || c.Message != fixtureDocs.unread {
		t.Errorf("the files not read: %+v", r.Checks)
	}
}
