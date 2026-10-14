package sample

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A reply naming rows and no week answers the week it follows only, never
// an earlier one; one naming a week answers that week, wherever it stands.
func TestWeeksReplies(t *testing.T) {
	file := filepath.Join(t.TempDir(), "forge.json")
	state := `{"issues": [{"id": 1, "title": "t", "labels": [], "comments": [
		"rows <!-- workline:rows [{\"n\":1},{\"n\":2}] -->\n\n<!-- workline:sample:2026-W40 -->",
		"rows <!-- workline:rows [{\"n\":1},{\"n\":2}] -->\n\n<!-- workline:sample:2026-W41 -->",
		{"body": "not 2", "author": "lead"},
		{"body": "2026-W40: not 1", "author": "lead"}],
		"reactions-by": {"1": [{"emoji": "-1", "author": "lead", "writer": true}], "2": [{"emoji": "-1", "author": "lead", "writer": true}]}}],
		"merge-requests": []}`
	if err := os.WriteFile(file, []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := weeks(&forge.Fake{Path: file, Repo: t.TempDir()}, 1)
	if err != nil || len(ws) != 2 {
		t.Fatalf("weeks: %+v %v", ws, err)
	}
	if fmt.Sprint(ws[0].Contested) != "map[1:true]" || fmt.Sprint(ws[1].Contested) != "map[2:true]" {
		t.Errorf("W40 contested %v (want 1), W41 %v (want 2)", ws[0].Contested, ws[1].Contested)
	}
}
