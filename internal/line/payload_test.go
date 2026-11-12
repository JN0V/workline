package line

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/routing"
)

// A GitLab note event names an issue when it is the reviewer's comment
// on the spec of an issue in refine (#846), or a person's on an issue on
// the line (#1414): the engine's other comments run nothing, which keeps
// them from starting a pipeline that starts another (#813).
func TestReadPayloadNotes(t *testing.T) {
	cfg := &routing.Config{States: []routing.State{
		{Name: "refine", Label: "workline:to-refine", Role: "product-owner"},
		{Name: "tests", Label: "workline:needs-tests", Role: "tester"},
	}}
	spec := "**The reviewer** read it.\n\n<!-- workline:spec-review body=ab round=1 open=1 in=Need -->\n<!-- workline:sticky=reviewer/spec -->"
	note := func(body, on string, labels []string) string {
		ev := map[string]any{"object_kind": "note", "user": map[string]any{"username": "bot"},
			"object_attributes": map[string]any{"note": body, "noteable_type": on}}
		if on == "Issue" {
			is := map[string]any{"iid": 7}
			if labels != nil {
				var ls []map[string]string
				for _, l := range labels {
					ls = append(ls, map[string]string{"title": l})
				}
				is["labels"] = ls
			}
			ev["issue"] = is
		}
		data, _ := json.Marshal(ev)
		return string(data)
	}
	for _, c := range []struct {
		name, raw string
		want      int
		why       string
	}{
		{"the spec review in refine", note(spec, "Issue", []string{"bug", "workline:to-refine"}), 7, ""},
		{"the spec review, no labels named", note(spec, "Issue", nil), 7, ""},
		{"the spec review out of refine", note(spec, "Issue", []string{"workline:needs-tests"}), 0, "not in refine"},
		{"the line's record", note("<!-- workline:sticky=line/state -->", "Issue", []string{"workline:to-refine"}), 0, "not the reviewer's on its spec"},
		{"a person's comment on the line", note("Looks right to me.", "Issue", []string{"workline:to-refine"}), 7, ""},
		{"a person's comment, no labels named", note("src/a.kt, src/b.kt", "Issue", nil), 7, ""},
		{"a person's comment off the line", note("Looks right to me.", "Issue", []string{"bug"}), 0, "the issue on no state"},
		{"on a merge request", note(spec, "MergeRequest", nil), 0, "a comment on a MergeRequest, not on an issue"},
	} {
		t.Run(c.name, func(t *testing.T) {
			n, why, err := readPayload(c.raw, cfg)
			if err != nil || n != c.want || !strings.Contains(why, c.why) || (c.want != 0) != (why == "") {
				t.Fatalf("got #%d, %q, %v; want #%d, %q", n, why, err, c.want, c.why)
			}
		})
	}
}

// A GitLab issue event that edited a body names the issue when it is on
// the line (#1414): a person's answer to a draft or a block; the line
// then says whether its step is due. Off the line, or no body edited
// and no state's label put on, nothing.
func TestReadPayloadBodyEdits(t *testing.T) {
	cfg := &routing.Config{States: []routing.State{{Name: "blocked", Label: "workline:blocked", Role: "product-owner"}}}
	for _, c := range []struct {
		name, raw string
		want      int
		why       string
	}{
		{"edited on the line", `{"object_kind":"issue","object_attributes":{"iid":4},"labels":[{"title":"workline:blocked"}],"changes":{"description":{"previous":"a","current":"b"}}}`, 4, ""},
		{"edited, no labels named", `{"object_kind":"issue","object_attributes":{"iid":4},"changes":{"description":{"previous":"a","current":"b"}}}`, 4, ""},
		{"edited off the line", `{"object_kind":"issue","object_attributes":{"iid":4},"labels":[{"title":"bug"}],"changes":{"description":{"previous":"a","current":"b"}}}`, 0, "the issue on no state"},
		{"its title changed", `{"object_kind":"issue","object_attributes":{"iid":4},"labels":[{"title":"workline:blocked"}],"changes":{"title":{"previous":"a","current":"b"}}}`, 0, "its labels did not"},
	} {
		t.Run(c.name, func(t *testing.T) {
			n, why, err := readPayload(c.raw, cfg)
			if err != nil || n != c.want || !strings.Contains(why, c.why) || (c.want != 0) != (why == "") {
				t.Fatalf("got #%d, %q, %v; want #%d, %q", n, why, err, c.want, c.why)
			}
		})
	}
}
