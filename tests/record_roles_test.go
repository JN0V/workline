package tests

import (
	"testing"

	"github.com/JN0V/workline/internal/builtin/inspector"
	"github.com/JN0V/workline/internal/builtin/reviewer"
	"github.com/JN0V/workline/internal/contract"
)

// The roles whose sticky comment keeps a record their next run reads
// offer how it merges, each its own, both ways: the engine asks them,
// never imports them (#375, #598).
func TestTheRolesKeepingARecordOfferItsMerge(t *testing.T) {
	const body = "**body**\n\n<!-- workline:triaged {\"fingerprint\":\"a\"} -->"
	const found = "**found**\n\n<!-- workline:triaged {\"fingerprint\":\"b\"} -->"
	// On this input the two roles' merges answer apart: the inspector's
	// reads the triage records, the reviewer's reads none of them.
	if reviewer.MergeSummary(body, "", found) == inspector.MergeComment(body, "", found) ||
		reviewer.MergeInto(found, body, "", "m") == inspector.MergeInto(found, body, "", "m") {
		t.Fatal("the input does not tell the reviewer's merges from the inspector's")
	}
	for name, own := range map[string]contract.Record{
		"reviewer":  {Merge: reviewer.MergeSummary, Into: reviewer.MergeInto},
		"inspector": {Merge: inspector.MergeComment, Into: inspector.MergeInto},
	} {
		rec := contract.Builtins().RecordOf(name)
		if rec == nil {
			t.Fatalf("the %s offers no record", name)
		}
		if got, want := rec.Merge(body, "", found), own.Merge(body, "", found); got != want {
			t.Errorf("the %s's Merge is not its own: %q, want %q", name, got, want)
		}
		if got, want := rec.Into(found, body, "", "m"), own.Into(found, body, "", "m"); got != want {
			t.Errorf("the %s's Into is not its own: %q, want %q", name, got, want)
		}
	}
}

// The reviewer offers the review the engine asks, the documentalist the
// docs the doctor asks: one role each.
func TestTheReviewAndTheDocsAreOffered(t *testing.T) {
	if v, name := contract.Builtins().Review(); v == nil || v.ReviewedAll == nil || name != "reviewer" {
		t.Errorf("Review = %v, %q", v, name)
	}
	if d, name := contract.Builtins().Docs(); d == nil || d.Coverage == nil || d.UnreadDocs == nil || name != "documentalist" {
		t.Errorf("Docs = %v, %q", d, name)
	}
}
