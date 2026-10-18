package gate

import (
	"testing"
)

func TestPatchReadByItsHunks(t *testing.T) {
	c := &change{gone: map[string]int{}}
	c.readPatch(`diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -3,0 +4,2 @@ func f() {
+++counter // a line that starts with ++
+	os.Remove(p) //nolint
@@ -9 +11 @@
-	old()
+	now()
diff --git "a/sp ace.go" "b/sp ace.go"
--- "a/sp ace.go"
+++ "b/sp ace.go"
@@ -1,0 +2 @@
+x
`)
	want := []line{{"x.go", 4, "++counter // a line that starts with ++"}, {"x.go", 5, "\tos.Remove(p) //nolint"}, {"x.go", 11, "\tnow()"}, {"sp ace.go", 2, "x"}}
	if len(c.added) != len(want) {
		t.Fatalf("added = %+v", c.added)
	}
	for i, l := range want {
		if c.added[i] != l {
			t.Errorf("added[%d] = %+v, want %+v", i, c.added[i], l)
		}
	}
	if c.gone["old()"] != 1 {
		t.Errorf("gone = %v", c.gone)
	}
}

func TestResultFilesFromTheTreesRoot(t *testing.T) {
	for uri, want := range map[string]string{
		"internal/x.go":            "internal/x.go",
		"./internal/x.go":          "internal/x.go",
		"file:///work/repo/a/b.go": "a/b.go",
		"/work/repo/a/b.go":        "a/b.go",
		"a/sp%20ace.go":            "a/sp ace.go",
		"/elsewhere/c.go":          "/elsewhere/c.go",
	} {
		if got := relative(uri, "/work/repo"); got != want {
			t.Errorf("relative(%q) = %q, want %q", uri, got, want)
		}
	}
}
