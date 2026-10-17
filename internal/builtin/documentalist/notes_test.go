package documentalist

import (
	"reflect"
	"sort"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// A note names its doc in `doc`, or, given as text, by the doc's path or an
// end of it no other doc of the task shares (#449).
func TestDoubtedDocs(t *testing.T) {
	judged := map[string]map[string]string{
		"docs/clock/status.md": {}, "docs/clock/README.md": {}, "docs/net/README.md": {}, "docs/clock/events.md": {},
	}
	note := func(v any) intent.Intention { return intent.Intention{Kind: "note", Value: v} }
	for _, c := range []struct {
		name string
		note intent.Intention
		want []string
	}{
		{"the run's note, a file name", note(`In status.md, the "33 tests" count could not be checked; so it is kept.`), []string{"docs/clock/status.md"}},
		{"a path", note("docs/clock/events.md: the second topic is not in the code given."), []string{"docs/clock/events.md"}},
		{"an end of a path", note("In clock/README.md the default could not be confirmed."), []string{"docs/clock/README.md"}},
		{"a file name two docs share names neither", note("The README.md version could not be confirmed."), nil},
		{"a path another doc's file name ends", note("docs/other/status.md says 33 tests."), nil},
		{"a longer name", note("status.mdx and events.md.bak are not docs of the task."), nil},
		{"two docs named", note("Neither status.md nor events.md could be confirmed whole."), []string{"docs/clock/events.md", "docs/clock/status.md"}},
		{"no doc named", note("The version macro is the only change."), nil},
		{"in doc", note(map[string]any{"doc": "docs/clock/status.md", "text": "The test count is in no source given."}), []string{"docs/clock/status.md"}},
		{"in doc, its text naming another", note(map[string]any{"doc": "./docs/clock/status.md", "text": "events.md says the same."}), []string{"docs/clock/status.md"}},
		{"in doc, a list", note(map[string]any{"doc": []any{"docs/clock/status.md", "docs/net/README.md"}, "text": "Kept."}), []string{"docs/clock/status.md", "docs/net/README.md"}},
		{"in doc, a doc not put before the agent", note(map[string]any{"doc": "docs/clock/tried.md", "text": "A record."}), nil},
		{"the engine's own mark", note(map[string]any{"own": true, "reader-read": true}), nil},
	} {
		got := doubtedDocs([]intent.Intention{c.note, {Kind: "patch", Value: "status.md"}}, judged)
		var docs []string
		for p := range got {
			docs = append(docs, p)
		}
		sort.Strings(docs)
		if !reflect.DeepEqual(docs, c.want) {
			t.Errorf("%s: doubted %v, want %v", c.name, docs, c.want)
		}
	}
}
