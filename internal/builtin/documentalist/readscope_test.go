package documentalist

import (
	"strings"
	"testing"
)

// A doc not given every source whole says what was read and what was not,
// in the order its header names them, and nothing is said found wrong of the
// rest (#120).
func TestReadScopeSaysWhatWasNotRead(t *testing.T) {
	sd := &suspectDoc{doc: &Doc{Sources: []string{"src/a.go", "src/b.go", "docs/x.md#y", "src/gone.go"}},
		read: []sourceRead{
			{src: "src/a.go", what: "what changed in src/a.go since abc1234", part: true},
			{src: "docs/x.md#y", what: "docs/x.md, judged in this same task"},
			{src: "src/gone.go", what: "what changed in src/gone.go since abc1234"},
		}}
	want := "read: what changed in src/a.go since abc1234, docs/x.md, judged in this same task, what changed in src/gone.go since abc1234; not read: the rest of src/a.go, src/b.go"
	if got := sd.scope().String(); got != want {
		t.Fatalf("scope = %q, want %q", got, want)
	}
	none := (&suspectDoc{doc: &Doc{Sources: []string{"src/a.go"}}}).scope().String()
	if none != "read: none of its sources; not read: src/a.go" {
		t.Fatalf("scope of a doc given none of its sources = %q", none)
	}
	if abbrev("0123456789abcdef") != "0123456" || abbrev("abc") != "abc" {
		t.Fatal("a commit is said in 7 characters")
	}
}

func TestNotVouchedSaysWhatWasRead(t *testing.T) {
	read := "read: what changed in src/a.go since abc1234; not read: the rest of src/a.go"
	for _, c := range []struct {
		fixed  string
		partly bool
	}{{"", false}, {"2 places", false}, {"", true}, {"2 places", true}} {
		got := notVouched(c.fixed, c.partly, read)
		if !strings.Contains(got, read) || strings.Contains(got, "nothing found wrong,") {
			t.Errorf("notVouched(%q, %v) = %q: it must say what was read, never nothing found wrong of the rest", c.fixed, c.partly, got)
		}
		if whole := notVouched(c.fixed, c.partly, ""); strings.Contains(whole, "not read") {
			t.Errorf("notVouched(%q, %v) of a doc read whole = %q", c.fixed, c.partly, whole)
		}
	}
	if got := notVouched("", false, read); !strings.HasPrefix(got, "(nothing found wrong in what was read,") {
		t.Errorf("notVouched = %q", got)
	}
}

func TestOutcomesSayWhatWasRead(t *testing.T) {
	judged := map[string]map[string]string{"docs/whole.md": {}, "docs/diff.md": {}, "docs/unrecorded.md": {}}
	patched := map[string]bool{"docs/whole.md": false, "docs/diff.md": false, "docs/unrecorded.md": false}
	read := map[string]string{"docs/diff.md": "read: what changed in src/a.go since abc1234; not read: the rest of src/a.go", "docs/unrecorded.md": readUnrecorded}
	got := outcomes(judged, nil, nil, patched, map[string]bool{}, nil, func(p string) string { return read[p] })
	if got["docs/whole.md"] != "nothing found wrong, `judged` recorded, `checked` not moved" {
		t.Errorf("whole: %q", got["docs/whole.md"])
	}
	if want := "nothing found wrong in what was read, `judged` recorded, `checked` not moved — " + read["docs/diff.md"]; got["docs/diff.md"] != want {
		t.Errorf("diff: %q, want %q", got["docs/diff.md"], want)
	}
	if o := got["docs/unrecorded.md"]; strings.Contains(o, "nothing found wrong,") || !strings.Contains(o, "not read: the rest") {
		t.Errorf("unrecorded: %q", o)
	}
}
