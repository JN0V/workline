package tester

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// Long thread, last reply read: an objection's thread of 250 comments ends
// with the tester's own reply, recognised by its author login. When the
// tester answers the Draft's objections, nothing is posted.
func TestAnswerLongThreadEndingWithTheTestersReplyPostsNothing(t *testing.T) {
	const bot = "workline-bot" // the simulated forge's own account
	rec := record{Issue: 790, Files: map[string]string{"a_test.go": "abc1234"}}
	replies := []forge.FakeComment{}
	for i := 2; i <= 250; i++ {
		c := forge.FakeComment{Body: fmt.Sprintf("builder: comment %d", i), Author: "builder"}
		if i%2 == 1 {
			c = forge.FakeComment{Body: fmt.Sprintf("tester: reply %d\n\n%s", i, answerMark), Author: bot}
		}
		replies = append(replies, c)
	}
	// 250 comments: the objection, then 249 replies, the last the tester's.
	replies[len(replies)-1] = forge.FakeComment{Body: "tester: the last reply\n\n" + answerMark, Author: bot}
	state := forge.FakeState{MergeRequests: []forge.FakeItem{{
		ID: 7, Branch: branchPrefix + "790", Labels: []string{},
		Comments: []forge.FakeComment{{Body: rec.body() + "\n" + forge.Marker("sticky=tester/"+recordKey), Author: bot}},
		Threads: []forge.FakeThread{{ID: "T1", Path: "a_test.go", Line: 3, Author: "builder",
			Body: ObjectionWord + "\ntest: a_test.go", Replies: replies}},
	}}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "forge.json")
	run := filepath.Join(dir, "run")
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(run, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, content := range map[string][]byte{fakePath: data, filepath.Join(run, "in", "settings.json"): []byte("{}")} {
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("WORKLINE_TARGET", "merge-request:7")
	t.Setenv("WORKLINE_FORGE", "fake:"+fakePath)
	f := &forge.Fake{Path: fakePath, Repo: dir}
	th, err := f.Threads(7)
	if err != nil || len(th) != 1 || len(th[0].Comments) != 250 {
		t.Fatalf("the fixture's thread: %d threads, %v", len(th), err)
	}
	before, _ := os.ReadFile(fakePath)

	if code := answerPre(run, dir, Settings{}); code != exitNothing {
		t.Fatalf("answerPre exited %d, want %d: nothing waits", code, exitNothing)
	}
	v, err := verdict.Read(filepath.Join(run, "out", "verdict.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != verdict.Pass || !strings.Contains(v.Summary, "no objection waits") {
		t.Fatalf("verdict %s %q, want a pass: no objection waits", v.Status, v.Summary)
	}
	if _, err := os.Stat(filepath.Join(run, "in", answerFile)); err == nil {
		t.Fatal("an answer was prepared for a thread whose last reply is the tester's")
	}
	after, _ := os.ReadFile(fakePath)
	if string(before) != string(after) {
		t.Fatal("the forge changed: something was posted")
	}
}
