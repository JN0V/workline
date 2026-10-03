package release

import "testing"

func TestVersion(t *testing.T) {
	for s, ok := range map[string]bool{"1.2.3": true, "0.10.0": true, "1.2": false, "1": false,
		"1.2.3-rc.1": false, "1.2.3+build": false, "01.2.3": false, "latest": false, "": false} {
		if got := version(s) != nil; got != ok {
			t.Errorf("version(%q) read = %v, want %v", s, got, ok)
		}
	}
	if !later(version("0.10.0"), version("0.9.9")) || later(version("1.0.0"), version("1.0.0")) {
		t.Error("versions compare as text, not as numbers")
	}
}

func TestIsBranch(t *testing.T) {
	s := Settings{Branches: Branches}
	for b, ok := range map[string]bool{"release-please--branches--main": true, "changeset-release/main": true,
		"release-plz-2026-10-03": true, "releaser-pleaser--branches--main": true, "feature": false, "release/1.1": false, "": false} {
		if s.IsBranch(b) != ok {
			t.Errorf("IsBranch(%q) = %v, want %v", b, !ok, ok)
		}
	}
}
