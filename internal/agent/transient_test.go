package agent

import (
	"errors"
	"fmt"
	"testing"
)

// A provider's overload or server error passes on its own; quota, login
// and a missing agent do not, nor any error but ErrUnavailable.
func TestTransient(t *testing.T) {
	for msg, want := range map[string]bool{
		"claude failed: API Error: 529 Overloaded. This is a server-side issue": true,
		"the command failed: API Error: 503 Service Unavailable":                true,
		"claude failed: overloaded_error":                                       true,
		"the command failed: the build server is overloaded":                    false,
		"quota":                   false,
		"claude is not installed": false,
		"claude did not answer within 3m0s (the role's model.timeout)": false,
	} {
		if got := Transient(fmt.Errorf("%w: %s", ErrUnavailable, msg)); got != want {
			t.Errorf("Transient(%q) = %v, want %v", msg, got, want)
		}
	}
	if Transient(errors.New("API Error: 529 Overloaded")) {
		t.Error("an error not ErrUnavailable is transient")
	}
}
