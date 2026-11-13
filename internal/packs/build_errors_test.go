package packs

import (
	"fmt"
	"strings"
	"testing"
)

// Given the kotlin-gradle pack listed and no build-errors set,
// When a new test breaks the build,
// Then the tester reads the pack's error rows: the listed packs give
// the tester's build-errors default, Kotlin's rows from kotlin-gradle.
func TestTesterDefaultsBuildErrorsFromPack(t *testing.T) {
	b := Builtin()
	d := TesterDefaults([]Pack{b["kotlin-gradle"]})
	rows, ok := d["build-errors"]
	if !ok {
		t.Fatalf("no build-errors default from the kotlin-gradle pack: %v", d)
	}
	text := fmt.Sprint(rows)
	if !strings.Contains(text, "Unresolved reference") || !strings.Contains(text, "e: ") {
		t.Errorf("the pack's Kotlin error rows are not in build-errors: %s", text)
	}
	if strings.Contains(text, "undefined: ") {
		t.Errorf("a pack not listed (go) gave its rows: %s", text)
	}
}
