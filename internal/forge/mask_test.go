package forge

import (
	"strings"
	"testing"
)

// The values of the forge's tokens and of the variables named like a
// secret are masked, the longest first; a short value, a name, and
// other text are kept.
func TestSecretsMasked(t *testing.T) {
	env := []string{
		"GH_TOKEN=fake-value-for-test-1215",
		"MY_api_key=fake-key-1215",
		"DB_PASSWORD=fake-pw-1215-fake-pw-1215-long",
		"SHORT_SECRET=1234567", // under minSecret: kept
		"HOME=/home/someone-long",
		"PEM_CREDENTIALS=line-one-of-it\nline-two-of-it",
	}
	secrets := Secrets(env)
	if secrets[0] != "fake-pw-1215-fake-pw-1215-long" {
		t.Errorf("longest not first: %q", secrets)
	}
	got := MaskWith("error: login refused for fake-value-for-test-1215 (GH_TOKEN), key fake-key-1215, pw fake-pw-1215-fake-pw-1215-long, pin 1234567, in /home/someone-long, pem line-two-of-it", secrets)
	want := "error: login refused for *** (GH_TOKEN), key ***, pw ***, pin 1234567, in /home/someone-long, pem ***"
	if got != want {
		t.Errorf("masked:\n got %q\nwant %q", got, want)
	}
	plain := "e: a/T.kt:1:2 Unresolved reference: login"
	if got := MaskWith(plain, secrets); got != plain {
		t.Errorf("a line with no secret changed: %q", got)
	}
}

// Mask reads the process's environment.
func TestMaskReadsTheEnvironment(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "fake-value-for-test-1215")
	if got := Mask("token=fake-value-for-test-1215"); strings.Contains(got, "fake-value") || got != "token=***" {
		t.Errorf("Mask = %q", got)
	}
}
