package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GitHub's projects linked to a repository, as GraphQL answers them: a
// closed project left out, an open one's Status options and owner's
// kind, its workflows read apart (ADR-0057).
func TestGitHubBoards(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in\n*workflows*) cat <<'EOF'\n" +
		`{"data":{"repository":{"projectsV2":{"nodes":[{"number":3,"workflows":{"nodes":[{"name":"Item closed","enabled":true},{"name":"Pull request linked to issue","enabled":false}]}}]}}}}` +
		"\nEOF\n;;\n*) cat <<'EOF'\n" +
		`{"data":{"repository":{"projectsV2":{"nodes":[` +
		`{"number":2,"title":"Old","url":"u2","closed":true,"owner":{"__typename":"User"},"field":null},` +
		`{"number":3,"title":"Plan","url":"u3","closed":false,"owner":{"__typename":"Organization"},"field":{"options":[{"name":"Todo"},{"name":"In Progress"},{"name":"Done"}]}}]}}}}` +
		"\nEOF\n;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	boards, ok, err := Boards(&github{repo: t.TempDir()})
	if !ok || err != nil || len(boards) != 1 {
		t.Fatalf("boards = %+v, %v, %v", boards, ok, err)
	}
	b := boards[0]
	if b.Number != 3 || b.Owner != "organization" || strings.Join(b.Columns, ",") != "Todo,In Progress,Done" ||
		!b.Workflows["Item closed"] || b.Workflows["Pull request linked to issue"] {
		t.Errorf("board = %+v", b)
	}
}

// A token without read:project is said with the scope to add.
func TestGitHubBoardsUnread(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\necho \"Your token has not been granted the required scopes: ['read:project']\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, _, err := Boards(&github{repo: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "gh auth refresh -s read:project") {
		t.Errorf("err = %v", err)
	}
	if _, ok, _ := Boards(&Local{}); ok {
		t.Error("the local forge holds boards")
	}
}
