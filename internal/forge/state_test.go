package forge

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// historyLine matches a line of a comment's history: a date, then a time.
var historyLine = regexp.MustCompile(`(?m)^.*\d{4}-\d{2}-\d{2}\D.*\d{2}:\d{2}.*$`)

// stateFake is a simulated forge holding issue #7 and merge request #8,
// the merge request starting with the comments given.
func stateFake(t *testing.T, onMR ...FakeComment) (f *Fake, issue, mr Target) {
	t.Helper()
	t.Setenv("WORKLINE_APP_ID", "")
	if onMR == nil {
		onMR = []FakeComment{}
	}
	f = &Fake{Path: filepath.Join(t.TempDir(), "forge.json")}
	s := &FakeState{
		Issues:        []FakeItem{{ID: 7, Title: "An issue", Labels: []string{}, Comments: []FakeComment{}}},
		MergeRequests: []FakeItem{{ID: 8, Branch: "feat", Base: "main", Title: "A change", Labels: []string{}, Comments: onMR}},
	}
	if err := f.save(s); err != nil {
		t.Fatal(err)
	}
	return f, Target{Kind: "issue", ID: 7}, Target{Kind: "merge-request", ID: 8}
}

// itemOf reads the item as the forge holds it: its comments as posted,
// not as a role reads them.
func itemOf(t *testing.T, f *Fake, at Target) FakeItem {
	t.Helper()
	s, err := f.loadState()
	if err != nil {
		t.Fatal(err)
	}
	it, err := s.item(at)
	if err != nil {
		t.Fatal(err)
	}
	return *it
}

// One comment: an issue the product owner, the reviewer, the tester and
// the developer each wrote on holds one comment of workline's, with a
// section for each role, each read back as the comment it was.
func TestStateCommentOnePerItem(t *testing.T) {
	f, issue, _ := stateFake(t)
	roles := []string{"product-owner", "reviewer", "tester", "developer"}
	for _, r := range roles {
		if err := f.Sticky(issue, "The "+r+" wrote its record.", Marker("sticky="+r+"/acts"), true); err != nil {
			t.Fatal(err)
		}
	}
	it := itemOf(t, f, issue)
	if len(it.Comments) != 1 || it.Posted != 1 {
		t.Fatalf("%d comments, %d posted, want one comment posted once: %+v", len(it.Comments), it.Posted, it.Comments)
	}
	body := it.Comments[0].Body
	for _, r := range roles {
		if !strings.Contains(body, r) || !strings.Contains(body, "The "+r+" wrote its record.") {
			t.Errorf("no section for %s in:\n%s", r, body)
		}
	}
	k, err := ReadKept(f, issue)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles {
		got, ok := k.Last(Marker("sticky=" + r + "/acts"))
		if !ok || !strings.Contains(got, "The "+r+" wrote its record.") {
			t.Errorf("%s's record read back as %q, %v", r, got, ok)
		}
	}
}

// Edited, not posted: with workline's comment on the issue, another role
// writing edits it, posts no new one, and a role writing again replaces
// its own section.
func TestStateCommentEditedNotPosted(t *testing.T) {
	f, issue, _ := stateFake(t)
	if err := f.Sticky(issue, "The reviewer found two problems.", Marker("sticky=reviewer/acts"), true); err != nil {
		t.Fatal(err)
	}
	if err := f.Sticky(issue, "The tester ran the tests.", Marker("sticky=tester/acts"), true); err != nil {
		t.Fatal(err)
	}
	if err := f.Sticky(issue, "The reviewer found none left.", Marker("sticky=reviewer/acts"), true); err != nil {
		t.Fatal(err)
	}
	it := itemOf(t, f, issue)
	if len(it.Comments) != 1 || it.Posted != 1 {
		t.Fatalf("%d comments, %d posted, want the one comment edited twice, never posted again", len(it.Comments), it.Posted)
	}
	k, err := ReadKept(f, issue)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := k.Last(Marker("sticky=reviewer/acts")); !ok || !strings.Contains(got, "found none left") {
		t.Errorf("the reviewer's section: %q, %v, want its new text", got, ok)
	}
	if got, ok := k.Last(Marker("sticky=tester/acts")); !ok || !strings.Contains(got, "ran the tests") {
		t.Errorf("the tester's section: %q, %v", got, ok)
	}
}

// Dated history: a comment edited by 3 runs ends with 3 lines, each with
// the date, the time (UTC) and the role, after the sections.
func TestStateCommentHistoryDated(t *testing.T) {
	f, issue, _ := stateFake(t)
	before := time.Now().UTC().Format("2006-01-02")
	roles := []string{"reviewer", "tester", "developer"}
	for _, r := range roles {
		if err := f.Sticky(issue, "The "+r+" is done.", Marker("sticky="+r+"/acts"), true); err != nil {
			t.Fatal(err)
		}
	}
	after := time.Now().UTC().Format("2006-01-02")
	body := itemOf(t, f, issue).Comments[0].Body
	lines := historyLine.FindAllString(body, -1)
	if len(lines) != 3 {
		t.Fatalf("%d history lines, want 3, in:\n%s", len(lines), body)
	}
	for i, r := range roles {
		if !strings.Contains(lines[i], r) {
			t.Errorf("line %d = %q, want the role %s", i+1, lines[i], r)
		}
		if !strings.Contains(lines[i], before) && !strings.Contains(lines[i], after) {
			t.Errorf("line %d = %q, want today's date in UTC (%s)", i+1, lines[i], after)
		}
		if at := strings.Index(body, "The "+r+" is done."); at < 0 || at > strings.Index(body, lines[0]) {
			t.Errorf("%s's section is not before the history, at the end of:\n%s", r, body)
		}
	}
}

// History kept short: after 12 edits the last 10 lines show, the first
// two cut.
func TestStateCommentHistoryCutAtTen(t *testing.T) {
	f, issue, _ := stateFake(t)
	for i := 1; i <= 12; i++ {
		role := fmt.Sprintf("role%02d", i)
		if err := f.Sticky(issue, "The "+role+" wrote.", Marker("sticky="+role+"/acts"), true); err != nil {
			t.Fatal(err)
		}
	}
	it := itemOf(t, f, issue)
	if len(it.Comments) != 1 {
		t.Fatalf("%d comments, want one", len(it.Comments))
	}
	lines := historyLine.FindAllString(it.Comments[0].Body, -1)
	if len(lines) != 10 {
		t.Fatalf("%d history lines, want 10:\n%s", len(lines), it.Comments[0].Body)
	}
	for j, l := range lines {
		if want := fmt.Sprintf("role%02d", j+3); !strings.Contains(l, want) {
			t.Errorf("line %d = %q, want the %s's, the last ten in order", j+1, l, want)
		}
	}
}

// Older comments folded: a pull request with 10 comments from an earlier
// workline, written by its App, holds one comment of workline's once the
// line writes on it, the others gone; a person's comment stays.
func TestStateCommentFoldsOlder(t *testing.T) {
	var old []FakeComment
	for n := 0; n < 10; n++ {
		old = append(old, FakeComment{Body: fmt.Sprintf("Old record %d.\n\n%s", n, Marker(fmt.Sprintf("sticky=role%d/acts", n))), Author: fakeBot, Bot: true})
	}
	old = append(old, FakeComment{Body: "I agree.", Author: "dana", Insider: true})
	f, _, mr := stateFake(t, old...)
	if err := f.Sticky(mr, "Fresh record.", Marker("sticky=tester/acts"), true); err != nil {
		t.Fatal(err)
	}
	it := itemOf(t, f, mr)
	var ours []string
	persons := 0
	for _, c := range it.Comments {
		switch {
		case strings.Contains(c.Body, "<!-- workline:"):
			ours = append(ours, c.Body)
		case c.Author == "dana" && c.Body == "I agree.":
			persons++
		}
	}
	if len(ours) != 1 || persons != 1 || len(it.Comments) != 2 {
		t.Fatalf("%d comments, %d of workline's, %d of the person's: want one of workline's and the person's alone", len(it.Comments), len(ours), persons)
	}
	for n := 0; n < 10; n++ {
		if !strings.Contains(ours[0], fmt.Sprintf("Old record %d.", n)) {
			t.Errorf("Old record %d lost in the one comment:\n%s", n, ours[0])
		}
	}
	if !strings.Contains(ours[0], "Fresh record.") {
		t.Errorf("the new record is not in the one comment:\n%s", ours[0])
	}
}

// A person's token left alone: a role's comment written under a person's
// account stays as it is when the line writes on the item; workline's
// own sections are one comment beside it.
func TestStateCommentPersonsTokenLeft(t *testing.T) {
	mine := FakeComment{Body: "Review by hand.\n\n" + Marker("sticky=reviewer/acts"), Author: "dana", Insider: true}
	f, _, mr := stateFake(t, mine)
	if err := KeepOwn(f, mr, "Machine review.", Marker("sticky=reviewer/acts"), true); err != nil {
		t.Fatal(err)
	}
	if err := KeepOwn(f, mr, "Tests ran.", Marker("sticky=tester/acts"), true); err != nil {
		t.Fatal(err)
	}
	it := itemOf(t, f, mr)
	if len(it.Comments) != 2 {
		t.Fatalf("%d comments, want the person's and one of workline's: %+v", len(it.Comments), it.Comments)
	}
	if c := it.Comments[0]; c.Body != mine.Body || c.Author != "dana" {
		t.Errorf("the person's comment changed: %+v", c)
	}
	if body := it.Comments[1].Body; !strings.Contains(body, "Machine review.") || !strings.Contains(body, "Tests ran.") {
		t.Errorf("workline's comment lacks a section:\n%s", body)
	}
}

// Too long for one comment: sections together past 65,536 characters
// leave two comments of workline's, none past the limit, the history in
// the first, the largest section in the second, nothing lost.
func TestStateCommentPastSizeLimit(t *testing.T) {
	f, issue, _ := stateFake(t)
	for _, s := range []struct {
		role, fill string
		n          int
	}{{"reviewer", "a", 20000}, {"tester", "b", 30000}, {"developer", "c", 25000}} {
		if err := f.Sticky(issue, "Record: "+strings.Repeat(s.fill, s.n), Marker("sticky="+s.role+"/acts"), true); err != nil {
			t.Fatalf("%s's write failed: %v", s.role, err)
		}
	}
	it := itemOf(t, f, issue)
	if len(it.Comments) != 2 {
		t.Fatalf("%d comments, want the main one and one overflow", len(it.Comments))
	}
	a, b, c := strings.Repeat("a", 20000), strings.Repeat("b", 30000), strings.Repeat("c", 25000)
	large := -1
	for i, cm := range it.Comments {
		if n := utf8.RuneCountInString(cm.Body); n > 65536 {
			t.Errorf("comment %d holds %d characters, past 65,536", i+1, n)
		}
		if strings.Contains(cm.Body, b) {
			large = i
		}
	}
	if large < 0 {
		t.Fatal("the largest section is in no comment: lost")
	}
	other := it.Comments[1-large].Body
	if !strings.Contains(other, a) || !strings.Contains(other, c) {
		t.Errorf("the other sections are not together in the main comment")
	}
	if strings.Contains(it.Comments[large].Body, a) || strings.Contains(it.Comments[large].Body, c) {
		t.Errorf("the overflow holds more than the largest section")
	}
	if historyLine.MatchString(it.Comments[large].Body) {
		t.Errorf("the history is in the overflow comment, not the main one")
	}
}

// Two writers at once: two roles read the comment, then both write, one
// after the other: both sections are in the comment.
func TestStateCommentTwoWriters(t *testing.T) {
	f, issue, _ := stateFake(t)
	if err := f.Sticky(issue, "The developer opened it.", Marker("sticky=developer/acts"), true); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadKept(f, issue); err != nil { // the reviewer reads
		t.Fatal(err)
	}
	if _, err := ReadKept(f, issue); err != nil { // the tester reads
		t.Fatal(err)
	}
	if err := f.Sticky(issue, "The reviewer found it sound.", Marker("sticky=reviewer/acts"), true); err != nil {
		t.Fatal(err)
	}
	if err := f.Sticky(issue, "The tester found it green.", Marker("sticky=tester/acts"), true); err != nil {
		t.Fatal(err)
	}
	it := itemOf(t, f, issue)
	if len(it.Comments) != 1 {
		t.Fatalf("%d comments, want one", len(it.Comments))
	}
	for _, w := range []string{"The developer opened it.", "The reviewer found it sound.", "The tester found it green."} {
		if !strings.Contains(it.Comments[0].Body, w) {
			t.Errorf("%q lost from the comment:\n%s", w, it.Comments[0].Body)
		}
	}
}

// A person's edit: a person edited the reviewer's section of the comment;
// the tester's and the developer's records are still read, the reviewer's
// is said unread.
func TestStateCommentPersonsEdit(t *testing.T) {
	f, issue, _ := stateFake(t)
	for _, s := range [][2]string{{"reviewer", "Reviewer found two."}, {"tester", "Tester ran it."}, {"developer", "Developer fixed it."}} {
		if err := f.Sticky(issue, s[1], Marker("sticky="+s[0]+"/acts"), true); err != nil {
			t.Fatal(err)
		}
	}
	s, err := f.loadState()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Issues[0].Comments) != 1 {
		t.Fatalf("%d comments, want one to edit", len(s.Issues[0].Comments))
	}
	c := &s.Issues[0].Comments[0]
	edited := strings.Replace(c.Body, "Reviewer found two.", "Reviewer found none, says a person.", 1)
	if edited == c.Body {
		t.Fatalf("the reviewer's section not found in:\n%s", c.Body)
	}
	c.Body, c.Editor = edited, "dana"
	if err := f.save(s); err != nil {
		t.Fatal(err)
	}
	k, err := ReadKept(f, issue)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := k.Last(Marker("sticky=reviewer/acts")); ok {
		t.Errorf("the edited section is read: %q", got)
	}
	if k.Unread(Marker("sticky=reviewer/acts")) == nil {
		t.Error("the edited section is not said unread")
	}
	for _, s := range [][2]string{{"tester", "Tester ran it."}, {"developer", "Developer fixed it."}} {
		if got, ok := k.Last(Marker("sticky=" + s[0] + "/acts")); !ok || !strings.Contains(got, s[1]) {
			t.Errorf("%s's record read as %q, %v, want it read", s[0], got, ok)
		}
	}
}

// Waits keep their mail: a wait said on a pull request is a comment of its
// own, beside the one holding the roles' sections.
func TestStateCommentWaitsKeepTheirOwn(t *testing.T) {
	f, _, mr := stateFake(t)
	if err := f.Sticky(mr, "The reviewer read it.", Marker("sticky=reviewer/acts"), true); err != nil {
		t.Fatal(err)
	}
	wait := Marker("merge-waits=feature")
	if err := f.Sticky(mr, "**Waiting for a person's yes.**", wait, true); err != nil {
		t.Fatal(err)
	}
	it := itemOf(t, f, mr)
	if len(it.Comments) != 2 || it.Posted != 2 {
		t.Fatalf("%d comments, %d posted, want the wait posted as its own", len(it.Comments), it.Posted)
	}
	var waits, others int
	for _, c := range it.Comments {
		if strings.Contains(c.Body, wait) {
			waits++
			if strings.Contains(c.Body, "reviewer") {
				t.Errorf("the wait shares its comment with a section:\n%s", c.Body)
			}
		} else if strings.Contains(c.Body, "The reviewer read it.") {
			others++
		}
	}
	if waits != 1 || others != 1 {
		t.Errorf("%d waits and %d section comments in %+v", waits, others, it.Comments)
	}
}
