package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GitHub's issue fields, as GraphQL answered them for a jn0v-lab
// repository on 2026-10-18: each single-select field with its options and
// visibility, the repository public, the token's user a member; a date
// field left out. A job's or an App's token is not shown to read values
// visible to members only: blind (ADR-0040). A value is written by REST,
// the option by name, with the API version header.
func TestGitHubIssueFields(t *testing.T) {
	bin := t.TempDir()
	args := filepath.Join(bin, "args")
	input := filepath.Join(bin, "input")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> " + args + "\ncat > " + input + "\n" + `cat <<'EOF'
{"data":{"repository":{"isPrivate":false,"owner":{"viewerIsAMember":true},"issueFields":{"nodes":[
{"fullDatabaseId":"47714576","name":"Priority","visibility":"ORG_ONLY","options":[{"name":"Urgent"},{"name":"High"},{"name":"Medium"},{"name":"Low"}]},
{},
{"fullDatabaseId":"47714579","name":"Effort","visibility":"ALL","options":[{"name":"High"},{"name":"Medium"},{"name":"Low"}]}]}}}}
EOF
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("WORKLINE_APP", "")
	g := &github{repo: t.TempDir()}
	fs, err := g.issueFields()
	if err != nil {
		t.Fatal(err)
	}
	if !fs.Public || fs.Blind != "" || len(fs.List) != 2 {
		t.Fatalf("fields = %+v", fs)
	}
	if p := fs.List[0]; p.ID != 47714576 || p.Name != "Priority" || p.Public || strings.Join(p.Options, ",") != "Urgent,High,Medium,Low" {
		t.Errorf("Priority = %+v", p)
	}
	if e := fs.List[1]; !e.Public || e.Options[0] != "High" {
		t.Errorf("Effort = %+v", e)
	}
	t.Setenv("GITHUB_ACTIONS", "true")
	if fs, _ := g.issueFields(); !strings.Contains(fs.Blind, "GITHUB_TOKEN") {
		t.Errorf("the job's token: %q", fs.Blind)
	}
	t.Setenv("WORKLINE_APP", "workline-jn0v")
	if fs, _ := g.issueFields(); !strings.Contains(fs.Blind, "App") {
		t.Errorf("an App's token: %q", fs.Blind)
	}
	if err := g.setField(3, Field{ID: 47714579, Name: "Effort"}, "Low"); err != nil {
		t.Fatal(err)
	}
	asked, _ := os.ReadFile(args)
	sent, _ := os.ReadFile(input)
	for _, w := range []string{"repos/{owner}/{repo}/issues/3/issue-field-values", "X-GitHub-Api-Version: 2026-03-10", "POST"} {
		if !strings.Contains(string(asked), w) {
			t.Errorf("gh not asked for %q: %s", w, asked)
		}
	}
	if string(sent) != `{"issue_field_values":[{"field_id":47714579,"value":"Low"}]}` {
		t.Errorf("sent %s", sent)
	}
}

// A user's repository answers issueFields null: no field, no error.
func TestGitHubNoIssueFields(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\ncat <<'EOF'\n" + `{"data":{"repository":{"isPrivate":false,"owner":{},"issueFields":null}}}` + "\nEOF\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	fs, err := (&github{repo: t.TempDir()}).issueFields()
	if err != nil || len(fs.List) != 0 {
		t.Errorf("fields = %+v, %v", fs, err)
	}
}
