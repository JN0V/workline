package role

import (
	"strings"
	"testing"
)

func TestNumbersNotAsWrittenRefused(t *testing.T) {
	for text, refused := range map[string]bool{
		"7515148":   false, // read as written: a setting holding a commit reads it as text
		"'0123456'": false,
		"0123456":   true, // octal
		"1234e56":   true, // a float
		"1234567890123456789012345678901234567890": true, // past an integer
		"0.85":   false, // not a commit's form
		"400000": false,
	} {
		err := CheckConfig([]byte("roles: {documentalist: {settings: {sample: {after: " + text + "}}}}"))
		if got := err != nil; got != refused {
			t.Errorf("%s: refused %v, want %v (%v)", text, got, refused, err)
		}
		if err != nil && !strings.Contains(err.Error(), "quote it") {
			t.Errorf("%s: %v does not say to quote it", text, err)
		}
	}
}
