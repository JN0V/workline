package tests

import (
	"regexp"
	"strings"
	"testing"
)

// Given the role contract, When a person reads the shared pre and post,
// Then it names the cuts key, the failure on an unreadable file, and what
// in/task.md holds.
func TestRoleContractSaysWhatTheSharedStepsDo(t *testing.T) {
	doc := readText(t, "../docs/spec/role-contract.md")
	_, rest, ok := strings.Cut(doc, "Each is optional, on its own")
	if !ok {
		t.Fatal("role-contract.md: the paragraph on the optional pre and post is gone")
	}
	section, _, _ := strings.Cut(rest, "## Where roles come from")
	if !strings.Contains(section, "`cuts`") || !strings.Contains(section, "in/cuts") {
		t.Errorf("the shared pre and post: no word on the `cuts` key and the in/cuts it writes:\n%s", section)
	}
	if !regexp.MustCompile(`(?i)cannot be read|unreadable|not readable`).MatchString(section) {
		t.Errorf("the shared pre and post: no word on the failure when a file cannot be read:\n%s", section)
	}
	if strings.Contains(section, "from the role's `instruction.md`") {
		t.Errorf("the shared pre and post: still says in/task.md comes from the instruction:\n%s", section)
	}
	if !strings.Contains(section, "Do what your instruction says") {
		t.Errorf("the shared pre and post: does not say what in/task.md holds (\"Do what your instruction says.\"):\n%s", section)
	}
}
