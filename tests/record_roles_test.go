package tests

import (
	"testing"

	"github.com/JN0V/workline/internal/contract"
)

// The roles whose sticky comment keeps a record their next run reads
// offer how it merges, both ways: the engine asks them, never imports
// them (#375, #598).
func TestTheRolesKeepingARecordOfferItsMerge(t *testing.T) {
	for _, name := range []string{"reviewer", "inspector"} {
		rec := contract.Builtins().RecordOf(name)
		if rec == nil || rec.Merge == nil || rec.Into == nil {
			t.Errorf("the %s offers %+v", name, rec)
		}
	}
}
