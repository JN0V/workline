package tests

import (
	"testing"

	"github.com/JN0V/workline/internal/contract"
)

// The documentalist offers the weekly sample what it asks of the docs,
// whole (contract.DocsSample): the sample never imports it.
func TestTheDocsSampleIsOffered(t *testing.T) {
	if d, name := contract.Builtins().DocsSample(); d == nil || name != "documentalist" {
		t.Errorf("DocsSample = %v, %q", d, name)
	}
}
