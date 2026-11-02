package developer

import "testing"

// The gates' summary comment of pull request #884, as GitHub showed it:
// one new errcheck finding, the round asks it at its place.
func TestGateRowsReadTheGatesComment(t *testing.T) {
	c := "### Gates: 2 new findings\n\nOn `5e59697`, only what this change adds since `8b3e072` counts; what was there before is counted, not shown.\n\n" +
		"| Gate | Check | New | Before it |\n|---|---|---|---|\n| `lint` | `golangci-lint` | 1 | 395 |\n| `vet` | `go-vet` | did not run | – |\n\n" +
		"- **`go-vet`** did not run: no go\n" +
		"- **`gofmt`** did not pass; fix: `gofmt -w .`\n" +
		"- **`errcheck`**, 1 new: `cmd/workline/cli_version_plain_test.go:22`. Error return value of `w.Close` is not checked\n" +
		"- 2 more rules, 3 findings, not listed here: the reviewer's `findings-max` (10)\n\n" +
		"Each is beside its line in the check [workline gates](https://example.invalid/runs/1), on the changed files; none is a conversation to resolve.\n\n<!-- workline:sticky=gates/summary -->"
	head, asked := gateRows(c)
	if head != "5e59697" {
		t.Errorf("head %q, want 5e59697", head)
	}
	if len(asked) != 2 {
		t.Fatalf("asked %d, want 2 (a check that did not run, and the rules not listed, ask nothing): %+v", len(asked), asked)
	}
	if a := asked[1]; a.Key != "gates|cmd/workline/cli_version_plain_test.go|errcheck" || a.Where != "cmd/workline/cli_version_plain_test.go:22" || a.Holds != gateHolds {
		t.Errorf("the errcheck finding read as %+v", a)
	}
	if a := asked[0]; a.Key != "gates||gofmt" || a.Where != "" {
		t.Errorf("the check that did not pass read as %+v", a)
	}
	if h, a := gateRows("### Gates: no new finding\n\n<!-- workline:sticky=gates/summary -->"); h != "" || a != nil {
		t.Errorf("a comment with no head read as %q %+v", h, a)
	}
}

func TestSameCommit(t *testing.T) {
	for _, c := range []struct {
		a, b string
		same bool
	}{
		{"5e59697", "5e59697bbc77f80b91c6ed9a1423d908ad71e49d", true},
		{"5e59697bbc77f80b91c6ed9a1423d908ad71e49d", "5e59697", true},
		{"5e59697", "4d2d6b1", false},
		{"", "5e59697", false},
		{"5e5", "5e59697", false},
	} {
		if got := sameCommit(c.a, c.b); got != c.same {
			t.Errorf("sameCommit(%q, %q) = %v", c.a, c.b, got)
		}
	}
}
