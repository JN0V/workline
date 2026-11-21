package developer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// threadForge writes a simulated forge holding the developer's pull
// request 1 with its record, which has heard the ids in heard, and one
// review thread: the reviewer's comment, workline's own account, and
// the replies under it.
func threadForge(t *testing.T, heard []string, replies []forge.FakeComment) (*forge.Fake, forge.Backlog) {
	t.Helper()
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "WORKLINE_APP_ID"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	record := "**The developer's record**\n\n" + devmark.Act{Act: "code", Issue: 1, Heard: heard}.Line() + "\n\n" + forge.Marker("sticky=developer/"+devmark.RecordKey)
	state := forge.FakeState{
		Issues: []forge.FakeItem{{ID: 1, Title: "Multiply", Body: "I want to multiply.", Labels: []string{}, Comments: []forge.FakeComment{}}},
		MergeRequests: []forge.FakeItem{{
			ID: 2, Branch: "workline/developer/1", Base: "main", Title: "feat: multiply", Body: "Closes #1", Labels: []string{},
			Comments: []forge.FakeComment{{Body: record, Author: "workline-bot", Bot: true, ID: "9001"}},
			Threads: []forge.FakeThread{{
				ID: "501", Path: "lib.sh", Line: 2, Author: "workline-bot", Bot: true,
				Body:    "**workline reviewer** · **blocks** · correctness\n\n**Important: mul adds**\n\nIt gives 5.\n\n<!-- workline:line=correctness/abc -->",
				Replies: replies,
			}},
		}},
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	b, ok := forge.Forge(f).(forge.Backlog)
	if !ok {
		t.Fatal("the simulated forge lists no comments")
	}
	return f, b
}

// askedPeople are the keys of what a round asks of a person.
func askedPeople(r round) []string {
	var keys []string
	for _, a := range r.Asked {
		if a.Holds == "a person" {
			keys = append(keys, a.Key)
		}
	}
	return keys
}

// Reply in a thread: Given an open reviewer thread on the developer's
// pull request, When a person of the project replies in it, Then the
// developer's next round lists that reply among what it is asked, and
// acts on it.
func TestReadAskedListsAThreadReply(t *testing.T) {
	f, b := threadForge(t, nil, []forge.FakeComment{
		{Body: "Please also handle the empty list.", Author: "dana", Insider: true, ID: "502"},
	})
	target := forge.Target{Kind: "merge-request", ID: 2}
	kept, err := forge.ReadKept(f, target)
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := readAsked(f, b, kept, t.TempDir(), devmark.Act{Act: "code", Issue: 1}, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	var got *devmark.Asked
	for i, a := range r.Asked {
		if a.Key == "person|502" {
			got = &r.Asked[i]
		}
	}
	if got == nil {
		t.Fatalf("the reply in the thread is not asked: %+v", r.Asked)
	}
	if got.Where != "@dana" || got.Holds != "a person" || got.Text != "Please also handle the empty list." {
		t.Errorf("the reply asked: %+v", *got)
	}
	task := roundTask(r, forge.Issue{ID: 1, Title: "Multiply"}, Bounds{Issue: 1}, Settings{Rounds: 3, MaxTurns: 40}, "", nil)
	if !strings.Contains(task, "Please also handle the empty list.") {
		t.Errorf("the round's task lacks the reply's words:\n%s", task)
	}
}

// Read once: Given a reply the developer already acted on, When the next
// round runs, Then that reply is not asked again.
func TestReadAskedHearsAThreadReplyOnce(t *testing.T) {
	f, b := threadForge(t, nil, []forge.FakeComment{
		{Body: "Please also handle the empty list.", Author: "dana", Insider: true, ID: "502"},
	})
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: 2})
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := readAsked(f, b, kept, t.TempDir(), devmark.Act{Act: "code", Issue: 1}, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(first.Heard, "502") {
		t.Fatalf("the reply's id is not recorded as heard: %v", first.Heard)
	}
	// The round kept what it heard in its record, as its post does.
	next := devmark.Act{Act: "code", Issue: 1, Heard: first.Heard}
	second, _, err := readAsked(f, b, kept, t.TempDir(), next, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if keys := askedPeople(second); len(keys) != 0 || len(second.Heard) != 0 {
		t.Errorf("a reply already acted on is asked again: %v, heard %v", keys, second.Heard)
	}
}

// The App's own reply: Given the same thread, When the App replies in
// it, Then no round is due from that reply.
func TestRoundDueNotFromTheAppsOwnThreadReply(t *testing.T) {
	replies := []forge.FakeComment{
		{Body: "Fixed: it multiplies now.", Author: "workline-bot", Insider: true, ID: "503"},
		{Body: "Build is green.", Author: "ci[bot]", Insider: true, Bot: true, ID: "504"},
		{Body: "Thanks, but keep the old name.", Author: "dana", Insider: true, ID: "505"},
	}
	// Of the three replies, a person's alone is asked.
	f, b := threadForge(t, nil, replies)
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: 2})
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := readAsked(f, b, kept, t.TempDir(), devmark.Act{Act: "code", Issue: 1}, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if keys := askedPeople(r); len(keys) != 1 || keys[0] != "person|505" || len(r.Heard) != 1 || r.Heard[0] != "505" {
		t.Errorf("asked %v, heard %v: want only the person's reply, 505", keys, r.Heard)
	}
	// The person's reply acted on: what is left is the App's and the bot's.
	f, _ = threadForge(t, []string{"505"}, replies)
	mr, err := f.MergeRequest(2)
	if err != nil {
		t.Fatal(err)
	}
	due, err := RoundDue(f, t.TempDir(), mr)
	if err != nil || due {
		t.Errorf("RoundDue = %v, %v; want no round due from the App's reply or a bot's", due, err)
	}
}
