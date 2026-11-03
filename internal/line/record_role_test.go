package line

import (
	"strings"
	"testing"
)

// Role in plain words:
// Given an issue to refine whose step woke the product owner,
// When the line writes its comment,
// Then it says "last woke the product owner".
func TestRoleInPlainWords(t *testing.T) {
	body := recordBody(Record{State: "refine", Hops: 1}, "product-owner")
	if !strings.Contains(body, "last woke the product owner") {
		t.Errorf("the comment should say %q, got:\n%s", "last woke the product owner", body)
	}
}
