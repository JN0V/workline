package hooks

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// editors are the executables of the VS Code family, whose git extension
// answers askpass requests with an input box in the window (ADR-0008).
var editors = map[string]bool{
	"code": true, "code-insiders": true, "code-oss": true, "codium": true, "codium-insiders": true,
}

// gitSocket is the name VS Code gives the socket its git askpass listens on.
var gitSocket = regexp.MustCompile(`^vscode-git-[0-9a-f]+\.sock$`)

// editorSocket finds the socket of the editor window the push came from: the
// nearest ancestor of this process that is an editor, and the git socket it
// listens on, from pid up. Nothing the pushing process sets is read: an agent can make
// the question appear, not point it to a socket of its own. Linux only: it
// reads /proc.
func editorSocket(proc string, pid int) (string, bool) {
	for i := 0; i < 64 && pid > 1; i++ {
		exe, err := os.Readlink(filepath.Join(proc, strconv.Itoa(pid), "exe"))
		if err == nil && editors[filepath.Base(strings.TrimSuffix(exe, " (deleted)"))] {
			if s, ok := listening(proc, pid); ok {
				return s, true
			}
		}
		if pid, err = parentOf(proc, pid); err != nil {
			return "", false
		}
	}
	return "", false
}

// parentOf reads a process's parent from /proc/<pid>/stat, whose second
// field, the command, may hold spaces and parentheses.
func parentOf(proc string, pid int) (int, error) {
	data, err := os.ReadFile(filepath.Join(proc, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, err
	}
	i := bytes.LastIndexByte(data, ')')
	if i < 0 {
		return 0, fmt.Errorf("unreadable stat for %d", pid)
	}
	f := strings.Fields(string(data[i+1:]))
	if len(f) < 2 {
		return 0, fmt.Errorf("unreadable stat for %d", pid)
	}
	return strconv.Atoi(f[1])
}

// listening returns the git socket a process listens on: one of its file
// descriptors is a socket whose inode /proc/net/unix lists as listening, at
// a path VS Code names. Stale socket files of closed windows are never
// listened on, so the folder alone cannot tell.
func listening(proc string, pid int) (string, bool) {
	fds, err := os.ReadDir(filepath.Join(proc, strconv.Itoa(pid), "fd"))
	if err != nil {
		return "", false
	}
	inodes := map[string]bool{}
	for _, fd := range fds {
		l, err := os.Readlink(filepath.Join(proc, strconv.Itoa(pid), "fd", fd.Name()))
		if err == nil && strings.HasPrefix(l, "socket:[") {
			inodes[strings.TrimSuffix(strings.TrimPrefix(l, "socket:["), "]")] = true
		}
	}
	f, err := os.Open(filepath.Join(proc, "net", "unix"))
	if err != nil {
		return "", false
	}
	defer f.Close()
	return listeningIn(f, inodes)
}

// listeningIn reads /proc/net/unix: Num RefCount Protocol Flags Type St
// Inode Path; a listening socket has the flag __SO_ACCEPTCON (00010000).
func listeningIn(r io.Reader, inodes map[string]bool) (string, bool) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) < 8 || f[3] != "00010000" || !inodes[f[6]] || !gitSocket.MatchString(filepath.Base(f[7])) {
			continue
		}
		return f[7], true
	}
	return "", false
}

// askEditor puts one question in the editor's input box and returns what
// the person typed; Escape, or closing it, answers "". The request is the
// one the editor's own askpass sends for a credential: argv[2] is shown in
// the box, argv[4] above it. The word "password" would hide what is typed.
func askEditor(socket, question, title string, timeout time.Duration) (string, error) {
	body, _ := json.Marshal(map[string]any{"askpassType": "https", "argv": []string{"", "", question, "", title}})
	client := http.Client{Timeout: timeout, Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}}
	resp, err := client.Post("http://editor/askpass", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("the editor answered %s", resp.Status)
	}
	var answer string
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}
