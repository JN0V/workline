package line

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/routing"
)

// A GitLab note event names an issue only when it is the reviewer's
// comment on the spec of an issue in refine (#846): any other comment,
// on an issue or not, runs nothing, which keeps the engine's own comments
// from starting a pipeline that starts another (#813).
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
		{"a person's comment", note("Looks right to me.", "Issue", []string{"workline:to-refine"}), 0, "not the reviewer's on its spec"},
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
