package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// The developer's record names the person: Given the developer's record
// of a round that waits on a person, When it is posted, Then it names
// the person with no mention.
func TestRoundNoteNamesTheWaitedPersonWithoutMention(t *testing.T) {
	f := threadForge(t, nil, []forge.FakeComment{
		{Body: "Please also handle the empty list.", Author: "dana", Insider: true, ID: "502"},
	})
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: 2})
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := readAsked(f, kept, t.TempDir(), devmark.Act{Act: "code", Issue: 1}, forge.MergeRequest{ID: 2}, "")
	if err != nil {
		t.Fatal(err)
	}
	x := devmark.Round{N: 1, Result: "no-code", Asked: texts(r.Asked)}
	body := roundBody(x, Settings{Rounds: 3}, nil, "", nil, nil)
	if !strings.Contains(body, "a person, ") || !strings.Contains(body, "dana") {
		t.Errorf("the round note does not name the person it waits on:\n%s", body)
	}
	if strings.Contains(body, "@dana") {
		t.Errorf("the round note mentions the person, it sends a notification:\n%s", body)
	}
}
