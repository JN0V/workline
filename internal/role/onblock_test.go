package role

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// on-block names only what the role may propose: a kind outside its
// intentions is refused when the role loads, never applied on a block.
func TestOnBlockOutsideIntentionsRefused(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "r"), 0o755)
	write := func(onBlock string) {
		os.WriteFile(filepath.Join(dir, "r", "role.yaml"), []byte("contract: 1\nname: r\nintentions: [comment, note]\non-block: "+onBlock+"\n"), 0o644)
	}
	write("[comment]")
	if r, err := Load(dir, "r"); err != nil || len(r.OnBlock) != 1 {
		t.Fatalf("on-block [comment]: %v, %v", r, err)
	}
	write("[patch]")
	if _, err := Load(dir, "r"); err == nil || !strings.Contains(err.Error(), `on-block names "patch"`) {
		t.Fatalf("on-block [patch]: %v, want it refused", err)
	}
}

// on-moved-on names only what the role may propose, and never what
// changes the branch or the merge request: a patch, a label, a commit's
// message apply on the head judged only (#375).
func TestOnMovedOnNeverAPatchNorALabel(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "r"), 0o755)
	write := func(movedOn string) {
		os.WriteFile(filepath.Join(dir, "r", "role.yaml"), []byte("contract: 1\nname: r\nintentions: [comment, issue, patch, label]\non-moved-on: "+movedOn+"\n"), 0o644)
	}
	write("[comment, issue]")
	if r, err := Load(dir, "r"); err != nil || len(r.OnMovedOn) != 2 {
		t.Fatalf("on-moved-on [comment, issue]: %v, %v", r, err)
	}
	for _, k := range []string{"patch", "label"} {
		write("[" + k + "]")
		if _, err := Load(dir, "r"); err == nil || !strings.Contains(err.Error(), `on-moved-on names "`+k+`"`) {
			t.Errorf("on-moved-on [%s]: %v, want it refused", k, err)
		}
	}
	write("[note]")
	if _, err := Load(dir, "r"); err == nil || !strings.Contains(err.Error(), "not one of its intentions") {
		t.Errorf("on-moved-on [note]: %v, want it refused", err)
	}
}

// apart names only events the role runs on: what only warns runs apart
// from the verdict where the role runs at all (ADR-0050).
func TestApartOnlyOnItsEvents(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "r"), 0o755)
	write := func(apart string) {
		os.WriteFile(filepath.Join(dir, "r", "role.yaml"), []byte("contract: 1\nname: r\nevents: [review, merge-request]\napart: "+apart+"\n"), 0o644)
	}
	write("[merge-request]")
	if r, err := Load(dir, "r"); err != nil || !r.ApartOn("merge-request") || r.ApartOn("review") {
		t.Fatalf("apart [merge-request]: %v, %v", r, err)
	}
	write("[schedule]")
	if _, err := Load(dir, "r"); err == nil || !strings.Contains(err.Error(), `apart names "schedule", not one of its events`) {
		t.Fatalf("apart [schedule]: %v, want it refused", err)
	}
}
