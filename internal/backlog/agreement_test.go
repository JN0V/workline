package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

const role = "product-owner"

var proposal = "@zed, the product owner read this issue.\n\n" + engineBlock +
	"\n\n```yaml\nneed: Every row exported.\nscope: WriteRows.\n```\n</details>\n\n" + ProposalMarker(role, 1)

// Who agrees is the last word, after the last proposal, of the reporter
// or a person of the project: a stranger's or a bot's comment is not
// theirs to take back, nor to give.
func TestAgreement(t *testing.T) {
	is := forge.Issue{ID: 9, Author: "zed"}
	reporter := func(body string) forge.Note { return forge.Note{Body: body, Author: "zed"} }
	insider := func(body string) forge.Note { return forge.Note{Body: body, Author: "dev", Insider: true} }
	stranger := func(body string) forge.Note { return forge.Note{Body: body, Author: "mia"} }
	bot := forge.Note{Body: "agreed", Author: "project_1_bot_0a", Insider: true, Bot: true}
	ask := forge.Note{Body: "@zed, to refine this issue: which export?\n\n" + AskMarker(role, 2)}
	for _, c := range []struct {
		name  string
		notes []forge.Note
		want  string
	}{
		{"the reporter agrees", []forge.Note{{Body: proposal}, reporter("Agreed!\n\nThanks.")}, "zed"},
		{"a person of the project agrees", []forge.Note{{Body: proposal}, insider("agreed")}, "dev"},
		{"a stranger's +1 after it changes nothing", []forge.Note{{Body: proposal}, reporter("agreed"), stranger("+1")}, "zed"},
		{"a bot after it changes nothing", []forge.Note{{Body: proposal}, reporter("agreed"), {Body: "linked", Author: "ci", Bot: true}}, "zed"},
		{"the same people's later word takes it back", []forge.Note{{Body: proposal}, reporter("agreed"), reporter("Wait, not the JSON export.")}, ""},
		{"a stranger cannot agree", []forge.Note{{Body: proposal}, stranger("agreed")}, ""},
		{"nor a bot", []forge.Note{{Body: proposal}, bot}, ""},
		{"nor someone unnamed", []forge.Note{{Body: proposal}, {Body: "agreed"}}, ""},
		{"an agreement before the proposal is not to it", []forge.Note{reporter("agreed"), {Body: proposal}}, ""},
		{"a question is not agreed to", []forge.Note{{Body: proposal}, reporter("hm"), ask, reporter("agreed")}, ""},
		{"one final mark aside, no more", []forge.Note{{Body: proposal}, reporter("agreed!!!")}, ""},
		{"nor a spaced mark", []forge.Note{{Body: proposal}, reporter("agreed .")}, ""},
		{"more than the word is an answer", []forge.Note{{Body: proposal}, reporter("Agreed, but not the scope")}, ""},
	} {
		if got := Agreement(c.notes, is, role); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// The agreement a refine carries is checked again on the forge: only a
// refine carries one, and only with the sections last proposed.
func TestCheckAgreed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	state := forge.FakeState{Issues: []forge.FakeItem{{ID: 9, Author: "zed", Labels: []string{},
		Comments: []forge.FakeComment{{Body: proposal}, {Body: "agreed", Author: "zed"}}}}}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	p := &Plan{issues: map[int]forge.Issue{9: {ID: 9, Author: "zed"}}}

	ready := Proposal{Do: "ready", Issue: 9, Agreed: "zed"}
	if rule, _ := p.checkAgreed(f, role, &ready); rule != "" || ready.Agreed != "" {
		t.Errorf("a ready carrying an agreement: %q, agreed %q — want it cleared", rule, ready.Agreed)
	}
	same := Proposal{Do: "refine", Issue: 9, Agreed: "zed", Need: "Every row exported.", Scope: "WriteRows."}
	if rule, why := p.checkAgreed(f, role, &same); rule != "" {
		t.Errorf("the text agreed to: %s %s", rule, why)
	}
	other := Proposal{Do: "refine", Issue: 9, Agreed: "zed", Need: "Something else.", Scope: "WriteRows."}
	if rule, _ := p.checkAgreed(f, role, &other); rule != "not-agreed" {
		t.Errorf("another text: %q, want not-agreed", rule)
	}
	who := Proposal{Do: "refine", Issue: 9, Agreed: "dev", Need: "Every row exported.", Scope: "WriteRows."}
	if rule, _ := p.checkAgreed(f, role, &who); rule != "not-agreed" {
		t.Errorf("someone else named: %q, want not-agreed", rule)
	}
}
