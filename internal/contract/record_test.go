package contract

import "testing"

// A role's comment record merges the way that role offers; one that
// offers none, or is not joined, has none.
func TestARecordMergesTheWayItsRoleOffers(t *testing.T) {
	r := NewRegistry()
	r.Join("keeper", Role{Record: &Record{Merge: func(body, _, _ string) string { return body + "+merged" }}})
	r.Join("plain", Role{})
	if rec := r.RecordOf("keeper"); rec == nil || rec.Merge("b", "r", "f") != "b+merged" {
		t.Errorf("RecordOf(keeper) = %v", rec)
	}
	if r.RecordOf("plain") != nil || r.RecordOf("absent") != nil {
		t.Error("a role offering no record, or none joined, has one")
	}
}
