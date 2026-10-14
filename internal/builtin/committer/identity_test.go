package committer

import "testing"

// A forge identity is an address written whole: a pattern is refused, so
// naming one bot never lets every bot in.
func TestForgeIdentitiesAreAddresses(t *testing.T) {
	for _, f := range []string{"12345+example-app[bot]@users.noreply.github.com", "noreply@github.com"} {
		if err := forgeIdentities(Settings{ForgeIdentities: []string{f}}); err != nil {
			t.Errorf("%q refused: %v", f, err)
		}
	}
	for _, f := range []string{`.*\[bot\]@users\.noreply\.github\.com`, `^noreply@github\.com$`, "github-actions[bot]", "a b@c.d"} {
		if err := forgeIdentities(Settings{ForgeIdentities: []string{f}}); err == nil {
			t.Errorf("%q taken, want refused", f)
		}
	}
}
