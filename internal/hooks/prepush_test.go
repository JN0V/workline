package hooks

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPushRanges(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.invalid")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "first")
	root := run("rev-parse", "HEAD")
	run("commit", "-q", "--allow-empty", "-m", "second")
	pushed := run("rev-parse", "HEAD")
	run("update-ref", "refs/remotes/origin/main", pushed) // what the remote holds
	run("commit", "-q", "--allow-empty", "-m", "third")
	head := run("rev-parse", "HEAD")

	for _, c := range []struct{ name, stdin, want string }{
		{"a branch the remote has", "refs/heads/main " + head + " refs/heads/main " + pushed, pushed + ".." + head},
		{"a new branch", "refs/heads/topic " + head + " refs/heads/topic " + zero, pushed + ".." + head},
		{"a remote tip not fetched", "refs/heads/main " + head + " refs/heads/main 1234567890123456789012345678901234567890", pushed + ".." + head},
		{"nothing new", "refs/heads/old " + pushed + " refs/heads/old " + zero, ""},
		{"a deleted ref", "(delete) " + zero + " refs/heads/gone " + pushed, ""},
	} {
		got, err := PushRanges(dir, "origin", strings.NewReader(c.stdin+"\n"))
		if err != nil || strings.Join(got, " ") != c.want {
			t.Errorf("%s: %v, %v; want %q", c.name, got, err, c.want)
		}
	}
	run("update-ref", "-d", "refs/remotes/origin/main")
	got, _ := PushRanges(dir, "origin", strings.NewReader("refs/heads/main "+head+" refs/heads/main "+zero+"\n"))
	if strings.Join(got, " ") != head {
		t.Errorf("a first push of the whole history: %v; want %q (root %s)", got, head, root)
	}
}
