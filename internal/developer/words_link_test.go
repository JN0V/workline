package developer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
)

// TestAgentWordsNeverFollowALink: an objections file that is a link is
// neither followed nor read: what it leads to is never posted (#746).
func TestAgentWordsNeverFollowALink(t *testing.T) {
	run := t.TempDir()
	tree := filepath.Join(run, agent.WorktreeDir)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(tree, ".workline"), 0o755))
	secret := filepath.Join(t.TempDir(), "secret")
	must(os.WriteFile(secret, []byte("- test: SECRET-TEXT\n  line: 1\n"), 0o644))
	must(os.Symlink(secret, filepath.Join(tree, filepath.FromSlash(ObjectionsFile))))
	objections, unread, err := agentWords(run)
	must(err)
	if len(objections) != 0 || len(unread) != 1 || strings.Contains(strings.Join(unread, ""), "SECRET-TEXT") || !strings.Contains(unread[0], "not followed") {
		t.Errorf("a link read: %v, %q", objections, unread)
	}
	if _, err := os.Lstat(filepath.Join(tree, filepath.FromSlash(ObjectionsFile))); err == nil {
		t.Errorf("the link left in the worktree")
	}
}

// TestClaimHoldsUnreadWordsWithNoPullRequest: no code, so no pull
// request: the words that did not read go in the claim on the issue.
func TestClaimHoldsUnreadWordsWithNoPullRequest(t *testing.T) {
	a := Act{Act: "claim", Issue: 3, Unread: []string{"Its objections file could not be read. As it came:\n\n~~~\n- test: [x\n~~~\n"}}
	body := claimBody(a, Settings{ClaimLabel: "workline:doing"})
	if !strings.Contains(body, "Its agent's words, unread") || !strings.Contains(body, "- test: [x") {
		t.Errorf("unread words not in the claim:\n%s", body)
	}
	a.Proof = &Proof{Result: Green}
	if strings.Contains(claimBody(a, Settings{ClaimLabel: "workline:doing"}), "- test: [x") {
		t.Errorf("unread words in the claim beside a pull request")
	}
}
