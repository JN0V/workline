package forge

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFakeCountsReads: with its reads log there, the fake forge counts
// each read, a byte each, and no write's own read; with none, it writes
// no log.
func TestFakeCountsReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(`{"issues": [{"id": 1, "title": "t", "body": "b"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &Fake{Path: path}
	if _, err := f.Issue(1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ReadsLog(path)); !os.IsNotExist(err) {
		t.Fatalf("a reads log with none asked: %v", err)
	}
	if err := os.WriteFile(ReadsLog(path), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Issue(1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Issues(); err != nil {
		t.Fatal(err)
	}
	if err := f.SetTitle(1, "new"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(ReadsLog(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Errorf("reads counted %d, want 2: two reads, the write's own read not one", len(data))
	}
}
