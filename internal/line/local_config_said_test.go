package line

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Config said: given an uncommitted change to the project's config, when
// reconcile reviews a local merge request, then the local guide says which
// config the run takes, and the run takes that one.
func TestLocalReviewUncommittedConfigSaid(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "main")
	cfg := filepath.Join(repo, ".workline", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("forge: local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "first")
	head := run("rev-parse", "HEAD")
	if err := os.WriteFile(cfg, []byte("forge: local\nautonomy: normal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree, done, err := HeadTree(repo, head)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	got, err := os.ReadFile(filepath.Join(tree, ".workline", "config.yaml"))
	if err != nil || string(got) != "forge: local\nautonomy: normal\n" {
		t.Fatalf("the run's config = %q, %v: want the clone's uncommitted one", got, err)
	}
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "local-line.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, step5, ok := strings.Cut(string(doc), "## 5. Review the merge request")
	if !ok {
		t.Fatal("no step 5 in docs/local-line.md")
	}
	step5, _, _ = strings.Cut(step5, "## 6.")
	if !strings.Contains(strings.ToLower(step5), "uncommitted") {
		t.Errorf("step 5 does not say which config the run takes (an uncommitted one)")
	}
}
