package hooks

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A fake /proc: the hook (pid 40) runs under git (30), under a shell (20),
// under the editor's extension host (10), which listens on its git socket;
// a closed window's socket is still in the folder, and listened on by none.
func fakeProc(t *testing.T, editorExe string) string {
	proc := t.TempDir()
	for _, p := range []struct {
		pid, parent int
		exe         string
		sockets     []string
	}{
		{40, 30, "/usr/bin/workline", nil},
		{30, 20, "/usr/bin/git", []string{"901"}},
		{20, 10, "/usr/bin/bash", nil},
		{10, 1, editorExe, []string{"900", "777"}},
	} {
		dir := filepath.Join(proc, strconv.Itoa(p.pid))
		os.MkdirAll(filepath.Join(dir, "fd"), 0o755)
		os.Symlink(p.exe, filepath.Join(dir, "exe"))
		os.WriteFile(filepath.Join(dir, "stat"), []byte(strconv.Itoa(p.pid)+" (a (name)) S "+strconv.Itoa(p.parent)+" 1 1 0"), 0o644)
		for i, s := range p.sockets {
			os.Symlink("socket:["+s+"]", filepath.Join(dir, "fd", strconv.Itoa(i+3)))
		}
	}
	os.MkdirAll(filepath.Join(proc, "net"), 0o755)
	os.WriteFile(filepath.Join(proc, "net", "unix"), []byte(`Num       RefCount Protocol Flags    Type St Inode Path
0000000000000000: 00000002 00000000 00010000 0001 01 900 /run/user/1000/vscode-git-c0488bf336.sock
0000000000000000: 00000002 00000000 00010000 0001 01 777 /run/user/1000/vscode-ipc-1.sock
0000000000000000: 00000003 00000000 00000000 0001 03 901 /run/user/1000/vscode-git-0000000000.sock
`), 0o644)
	return proc
}

func TestEditorSocketFromAncestors(t *testing.T) {
	if s, ok := editorSocket(fakeProc(t, "/usr/share/codium/codium"), 40); !ok || s != "/run/user/1000/vscode-git-c0488bf336.sock" {
		t.Fatalf("socket = %q, %v", s, ok)
	}
	// No editor among the ancestors: an agent's shell in a terminal, CI.
	if s, ok := editorSocket(fakeProc(t, "/usr/bin/tmux"), 40); ok {
		t.Fatalf("socket = %q from no editor", s)
	}
}

// editorServer answers askpass requests as VS Code does, with the answers
// given, and keeps what it was asked.
func editorServer(t *testing.T, answers ...string) (string, *[][]string) {
	socket := filepath.Join(t.TempDir(), "vscode-git-1234567890.sock")
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	asked := &[][]string{}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type string   `json:"askpassType"`
			Argv []string `json:"argv"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.Method != "POST" || r.URL.Path != "/askpass" || req.Type != "https" || len(req.Argv) < 5 {
			http.Error(w, "not an askpass request", 500) // as VS Code fails on fewer than five
			return
		}
		*asked = append(*asked, req.Argv)
		answer := ""
		if len(answers) > 0 {
			answer, answers = answers[0], answers[1:]
		}
		json.NewEncoder(w).Encode(answer)
	})}
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })
	return socket, asked
}

func TestAskInTheEditor(t *testing.T) {
	repo, refs := pushRepo(t)
	for _, c := range []struct {
		answers []string
		want    bool
		viewed  bool
	}{
		{answers: []string{"y"}, want: true},
		{answers: []string{""}, want: false}, // Escape, or Enter on nothing
		{answers: []string{"no"}, want: false},
		{answers: []string{"v", "Y"}, want: true, viewed: true},
	} {
		socket, asked := editorServer(t, c.answers...)
		viewed := false
		var out strings.Builder
		got := askOnce(Push{Repo: repo, Remote: "origin", Refs: refs}, &out,
			func(q, title string) (string, error) { return askEditor(socket, q, title, answerTimeout) },
			func(string) error { viewed = true; return nil })
		if got != c.want || viewed != c.viewed {
			t.Errorf("answers %q: approved %v, viewed %v; want %v, %v\n%s", c.answers, got, viewed, c.want, c.viewed, out.String())
		}
		argv := (*asked)[0]
		if strings.Contains(strings.ToLower(argv[2]), "password") || !strings.Contains(argv[4], "push 1 commit(s) to origin main") {
			t.Errorf("asked %q", argv)
		}
		if !strings.Contains(out.String(), "feat: the change to push") {
			t.Errorf("the commits are not listed where git shows them:\n%s", out.String())
		}
	}
}

// The editor gone, or failing: no answer, no push.
func TestEditorFailingStopsThePush(t *testing.T) {
	repo, refs := pushRepo(t)
	var out strings.Builder
	got := askOnce(Push{Repo: repo, Remote: "origin", Refs: refs}, &out,
		func(q, title string) (string, error) {
			return askEditor(filepath.Join(t.TempDir(), "gone.sock"), q, title, answerTimeout)
		},
		func(string) error { return errors.New("unused") })
	if got || !strings.Contains(out.String(), "push stopped") {
		t.Fatalf("approved %v\n%s", got, out.String())
	}
}
