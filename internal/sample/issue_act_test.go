package sample

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
)

// commitAt is a repository whose only commit is dated `when` (RFC 3339):
// the commit an act's record keeps. Git's own settings never decide it.
func commitAt(t *testing.T, when string) (repo, sha string) {
	t.Helper()
	repo = t.TempDir()
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when)
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=" + os.DevNull}, args...)...)
		cmd.Dir, cmd.Env = repo, env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "chore: the code the product owner read")
	return repo, run("rev-parse", "HEAD")
}

// readerOn is a reader of the issue #7 of a simulated forge, with these
// comments, its body saying what the third scenario's question is about.
func readerOn(t *testing.T, repo string, comments []forge.FakeComment) *reader {
	t.Helper()
	state, err := json.Marshal(forge.FakeState{Issues: []forge.FakeItem{{
		ID: 7, Title: "Export the sales", Comments: comments,
		Body: "## Need\n\nEvery sale exported.\n\nDoes it include refunds?\n\n## Validation\n\n- **Three sales:** Then 3 rows.\n\n## Scope\n\nWriteRows.",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, state, 0o600); err != nil {
		t.Fatal(err)
	}
	return &reader{
		o:   Options{Repo: repo, RolesDir: "../../roles", Forge: &forge.Fake{Path: path, Repo: repo}},
		res: &Result{Week: "2026-W46"},
		cap: 1 << 30,
	}
}

// given is all the material shows: its about line and each item's text.
func given(m *material) string {
	var b strings.Builder
	b.WriteString(m.about + "\n")
	for _, it := range m.items {
		b.WriteString(it.text + "\n")
	}
	return b.String()
}

// Comment edited later: Given a comment created before the act and edited
// after it, When the auditor samples the act, Then it reads the comment as
// it was at the act, or leaves it out, said. The simulated forge keeps no
// edit history of a comment: the edited comment is left out, and the
// material says it was edited; a comment nobody edited is still given.
func TestIssueActLeavesOutACommentEditedLater(t *testing.T) {
	repo, sha := commitAt(t, "2026-11-10T10:00:00Z")
	rd := readerOn(t, repo, []forge.FakeComment{
		{Body: "Ann: the export is for CSV only.", Author: "ann", Created: "2026-11-08T09:00:00Z"},
		{Body: "Ann: EDITED-AFTER the export is for JSON too.", Author: "ann", Editor: "bob", Created: "2026-11-09T09:00:00Z"},
	})
	a := &Act{Role: productOwner, Kind: "ready", Ref: "#7", Day: "2026-11-10", Commit: sha, By: "alone", Line: "ready"}
	m := &material{holds: "true"}
	if why := rd.issueAct(a, m, "ready"); why != "" {
		t.Fatalf("issueAct: %s", why)
	}
	got := given(m)
	if !strings.Contains(got, "the export is for CSV only") {
		t.Errorf("a comment nobody edited is given:\n%s", got)
	}
	if strings.Contains(got, "EDITED-AFTER") {
		t.Errorf("a comment edited after the act, its forge keeping no history of it, is given as it reads now:\n%s", got)
	}
	if !strings.Contains(m.about, "edit") {
		t.Errorf("the material does not say a comment was left out as edited since: %q", m.about)
	}
}

// Comment posted later the same day: Given a comment posted two hours after
// the act, the same day, When the auditor samples the act, Then the comment
// is not given. The act's time is the date of the commit its record keeps.
func TestIssueActLeavesOutACommentPostedLaterTheSameDay(t *testing.T) {
	repo, sha := commitAt(t, "2026-11-10T10:00:00Z")
	rd := readerOn(t, repo, []forge.FakeComment{
		{Body: "Mia: LATER-SAME-DAY it must also pay refunds.", Author: "mia", Created: "2026-11-10T12:00:00Z"},
		{Body: "Mia: EARLIER-SAME-DAY it is for CSV.", Author: "mia", Created: "2026-11-10T08:00:00Z"},
	})
	a := &Act{Role: productOwner, Kind: "ready", Ref: "#7", Day: "2026-11-10", Commit: sha, By: "alone", Line: "ready"}
	m := &material{holds: "true"}
	if why := rd.issueAct(a, m, "ready"); why != "" {
		t.Fatalf("issueAct: %s", why)
	}
	got := given(m)
	if strings.Contains(got, "LATER-SAME-DAY") {
		t.Errorf("a comment posted two hours after the act, the same day, is given:\n%s", got)
	}
	if !strings.Contains(got, "EARLIER-SAME-DAY") {
		t.Errorf("a comment posted before the act, the same day, is not given:\n%s", got)
	}
}

// Question for a person: Given a spec review with one question for a person
// and "No important finding open", When the auditor samples the `ready` that
// followed, Then the question alone does not make the verdict false: the
// ready statement the panel is given, when the auditor says it false, no
// longer asks for nothing left open in the comments.
func TestReadyPanelStatementLeavesAQuestionForAPersonOut(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // the person's own facets stay out
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, sha := commitAt(t, "2026-11-10T10:00:00Z")
	rd := readerOn(t, repo, []forge.FakeComment{
		{Body: "**workline reviewer**, its spec — 0 findings on the spec, 1 question for a person.\n\n**Holds**: No important finding open, round 1 of 5.\n\n**Questions for a person**\n\n- `#7:5` (ambiguous): Does it include refunds?",
			Author: "workline-bot", Bot: true, Created: "2026-11-09T09:00:00Z"},
	})
	auditor, err := role.Load("../../roles", "auditor")
	if err != nil {
		t.Fatal(err)
	}
	rd.auditor = auditor
	panelIn := filepath.Join(t.TempDir(), "panel.txt")
	t.Setenv("PANEL_IN", panelIn)
	script := filepath.Join(t.TempDir(), "judge.sh")
	body := `in=$(cat)
case "$in" in
*"Two readings of the act below disagree"*)
  printf '%s' "$in" > "$PANEL_IN"
  printf -- '- note: "yes: it holds"\n' ;;
*)
  printf -- '- claim:\n    quote: "a question is open"\n    why: "a question is open"\n    source: {path: "#7", quote: "Does it include refunds?"}\n- note: "false: a question is open"\n' ;;
esac
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	rd.spec = "cmd:sh " + script
	rd.settings.Sample.Panel = []string{rd.spec}
	a := &Act{Role: productOwner, Kind: "ready", Ref: "#7", Day: "2026-11-10", Commit: sha, By: "alone", Line: "ready", key: "7-ready"}
	rd.readAct(a)
	asked, err := os.ReadFile(panelIn)
	if err != nil {
		t.Fatalf("the panel was never asked (verdict %s, why %q): %v", a.Verdict, a.Why, err)
	}
	if strings.Contains(string(asked), "nothing left open in its comments") {
		t.Errorf("the ready statement given to the panel still asks for nothing left open in its comments:\n%s", asked)
	}
}

// Older rule: Given a `ready` done on 2026-10-14, When the auditor samples
// it, Then it is judged by the meaning of ready on that day, said in its
// material.
func TestIssueActSaysWhatReadyMeantOnAnOlderDay(t *testing.T) {
	repo, sha := commitAt(t, "2026-10-14T10:00:00Z")
	rd := readerOn(t, repo, nil)
	a := &Act{Role: productOwner, Kind: "ready", Ref: "#7", Day: "2026-10-14", Commit: sha, By: "alone", Line: "ready"}
	m := &material{holds: "true"}
	if why := rd.issueAct(a, m, "ready"); why != "" {
		t.Fatalf("issueAct: %s", why)
	}
	if !strings.Contains(m.about, "Ready meant, on 2026-10-14:") {
		t.Errorf("the material does not say what ready meant on the act's day: %q", m.about)
	}
}

// askedPanelOnReady is what the panel is given when the auditor says a
// ready false because of a question for a person, the spec review's
// "No important finding open" on the issue.
func askedPanelOnReady(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, sha := commitAt(t, "2026-11-10T10:00:00Z")
	rd := readerOn(t, repo, []forge.FakeComment{
		{Body: "**workline reviewer**, its spec — 0 findings on the spec, 1 question for a person.\n\n**Holds**: No important finding open, round 1 of 5.\n\n**Questions for a person**\n\n- `#7:5` (ambiguous): Does it include refunds?",
			Author: "workline-bot", Bot: true, Created: "2026-11-09T09:00:00Z"},
	})
	auditor, err := role.Load("../../roles", "auditor")
	if err != nil {
		t.Fatal(err)
	}
	rd.auditor = auditor
	panelIn := filepath.Join(t.TempDir(), "panel.txt")
	t.Setenv("PANEL_IN", panelIn)
	script := filepath.Join(t.TempDir(), "judge.sh")
	body := `in=$(cat)
case "$in" in
*"Two readings of the act below disagree"*)
  printf '%s' "$in" > "$PANEL_IN"
  printf -- '- note: "yes: it holds"\n' ;;
*)
  printf -- '- claim:\n    quote: "a question is open"\n    why: "a question is open"\n    source: {path: "#7", quote: "Does it include refunds?"}\n- note: "false: a question is open"\n' ;;
esac
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	rd.spec = "cmd:sh " + script
	rd.settings.Sample.Panel = []string{rd.spec}
	a := &Act{Role: productOwner, Kind: "ready", Ref: "#7", Day: "2026-11-10", Commit: sha, By: "alone", Line: "ready", key: "7-ready"}
	rd.readAct(a)
	asked, err := os.ReadFile(panelIn)
	if err != nil {
		t.Fatalf("the panel was never asked (verdict %s, why %q): %v", a.Verdict, a.Why, err)
	}
	return string(asked)
}

// Question for a person: Given a spec review with one question for a person
// and "No important finding open", When the auditor samples the `ready` that
// followed, Then the question alone does not make the verdict false: the
// ready statement the panel is given names the important finding as what
// must not be left open, and says a question that holds nothing is no
// finding.
func TestReadyPanelStatementSaysAQuestionThatHoldsNothingIsNoFinding(t *testing.T) {
	asked := askedPanelOnReady(t)
	if !strings.Contains(asked, "no important finding left open") {
		t.Errorf("the ready statement given to the panel does not ask for no important finding left open:\n%s", asked)
	}
	if !strings.Contains(asked, "a question that holds nothing is no finding") {
		t.Errorf("the ready statement given to the panel does not say a question that holds nothing is no finding:\n%s", asked)
	}
}
