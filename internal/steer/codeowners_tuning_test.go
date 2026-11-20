package steer

import (
	"os"
	"path/filepath"
	"testing"
)

// Tuning takes the normal path, authority stays guarded: given workline's
// CODEOWNERS and the files the project's config is split into in
// .workline/, when a pull request changes only a tuning file, then the
// merge policy holds nothing of it, and when it changes the guarded
// config.yaml, then it waits for the person's label.
func TestWorklineCodeownersGuardsAuthorConfigOnly(t *testing.T) {
	data, err := os.ReadFile("../../CODEOWNERS")
	if err != nil {
		t.Fatal(err)
	}
	co := ParseCodeowners("CODEOWNERS", data)
	files, err := filepath.Glob("../../.workline/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 2 {
		t.Fatalf(".workline/ holds %d config files, want the guarded config.yaml and a tuning file beside it", len(files))
	}
	for _, f := range files {
		name := ".workline/" + filepath.Base(f)
		if filepath.Base(f) == "config.yaml" {
			if !co.Holds(name) {
				t.Errorf("%s is not held by CODEOWNERS: authority settings must wait for the person's label", name)
			}
			continue
		}
		if co.Holds(name) {
			t.Errorf("%s is held by CODEOWNERS: a tuning file takes the line's normal path", name)
		}
	}
}
