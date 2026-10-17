package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A comment renewed on GitHub is posted anew first, then each comment
// carrying the marker before it is deleted: the state is never lost
// between the two, and a new comment mails the issue's subscribers
// (ADR-0038, amended). gh is replaced by a script logging each call and
// printing the ids the listing asks for.
func TestGitHubRenew(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	script := "#!/bin/sh\necho \"$*\" >> " + log + "\ncase \"$*\" in *--paginate*) printf '11\\n12\\n' ;; esac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	if err := g.Renew(Target{Kind: "issue", ID: 9}, "state", Marker("sticky=po/state")); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	var calls []string // the body's own lines left out
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "api ") {
			calls = append(calls, l)
		}
	}
	want := []string{"--paginate repos/{owner}/{repo}/issues/9/comments", "-X POST repos/{owner}/{repo}/issues/9/comments",
		"-X DELETE repos/{owner}/{repo}/issues/comments/11", "-X DELETE repos/{owner}/{repo}/issues/comments/12"}
	if len(calls) != len(want) {
		t.Fatalf("calls:\n%s\nwant %d", data, len(want))
	}
	for i, w := range want {
		if !strings.Contains(calls[i], w) {
			t.Errorf("call %d = %q, want it to hold %q", i+1, calls[i], w)
		}
	}
}
