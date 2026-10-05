package role

import (
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"
)

func settings(t *testing.T, src string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMergeSettings(t *testing.T) {
	defaults := `
issues-per-run: 8
acts:
  refine: {mode: act, max: 5}
  ask: {mode: act, max: 3, rounds: 3}
  close-obsolete: {mode: act, max: 3, exempt: [pinned, security]}
budgets: {doc-lines: 200, card-words: {min: 80, max: 400}}
types: [feat, fix]
`
	for _, tc := range []struct{ name, over, want string }{
		{"nothing set", ``, defaults},
		{"one act kind keeps the others",
			`acts: {refine: {mode: propose}}`,
			`
issues-per-run: 8
acts:
  refine: {mode: propose, max: 5}
  ask: {mode: act, max: 3, rounds: 3}
  close-obsolete: {mode: act, max: 3, exempt: [pinned, security]}
budgets: {doc-lines: 200, card-words: {min: 80, max: 400}}
types: [feat, fix]
`},
		{"a list replaced whole, a scalar replaced",
			`{types: [docs], issues-per-run: 2, acts: {close-obsolete: {exempt: [wontfix]}}}`,
			`
issues-per-run: 2
acts:
  refine: {mode: act, max: 5}
  ask: {mode: act, max: 3, rounds: 3}
  close-obsolete: {mode: act, max: 3, exempt: [wontfix]}
budgets: {doc-lines: 200, card-words: {min: 80, max: 400}}
types: [docs]
`},
		{"three levels deep",
			`budgets: {card-words: {max: 600}}`,
			`
issues-per-run: 8
acts:
  refine: {mode: act, max: 5}
  ask: {mode: act, max: 3, rounds: 3}
  close-obsolete: {mode: act, max: 3, exempt: [pinned, security]}
budgets: {doc-lines: 200, card-words: {min: 80, max: 600}}
types: [feat, fix]
`},
		{"a null removes the key, a new key is added",
			`{acts: {ask: null, split: {mode: off}}, budgets: {card-words: null}, history: [x]}`,
			`
issues-per-run: 8
acts:
  refine: {mode: act, max: 5}
  close-obsolete: {mode: act, max: 3, exempt: [pinned, security]}
  split: {mode: off}
budgets: {doc-lines: 200}
types: [feat, fix]
history: [x]
`},
		{"a map over a scalar replaces it, a scalar over a map too",
			`{issues-per-run: {a: 1}, budgets: 3}`,
			`
issues-per-run: {a: 1}
acts:
  refine: {mode: act, max: 5}
  ask: {mode: act, max: 3, rounds: 3}
  close-obsolete: {mode: act, max: 3, exempt: [pinned, security]}
budgets: 3
types: [feat, fix]
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := settings(t, defaults)
			got := MergeSettings(d, settings(t, tc.over))
			if want := settings(t, tc.want); !reflect.DeepEqual(got, want) {
				t.Errorf("got  %v\nwant %v", got, want)
			}
			if !reflect.DeepEqual(d, settings(t, defaults)) {
				t.Errorf("the defaults were changed: %v", d)
			}
		})
	}
}

func TestMergeSettingsNullInANewMap(t *testing.T) {
	d := settings(t, `{acts: {refine: {mode: act}}, budgets: 3}`)
	got := MergeSettings(d, settings(t, `{acts: {split: {mode: off, max: null}}, budgets: {doc-lines: 9, x: null}}`))
	want := settings(t, `{acts: {refine: {mode: act}, split: {mode: off}}, budgets: {doc-lines: 9}}`)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

func TestMergeSettingsSharesNothing(t *testing.T) {
	d := settings(t, `{acts: {refine: {mode: act}}, types: [feat], budgets: {card-words: {min: 1}}}`)
	o := settings(t, `{history: [x], derive: {a: b}}`)
	got := MergeSettings(d, o)
	got["acts"].(map[string]any)["refine"].(map[string]any)["mode"] = "off"
	got["types"].([]any)[0] = "fix"
	got["budgets"].(map[string]any)["card-words"].(map[string]any)["min"] = 2
	got["history"].([]any)[0] = "y"
	got["derive"].(map[string]any)["a"] = "c"
	if !reflect.DeepEqual(d, settings(t, `{acts: {refine: {mode: act}}, types: [feat], budgets: {card-words: {min: 1}}}`)) {
		t.Errorf("the defaults were changed: %v", d)
	}
	if !reflect.DeepEqual(o, settings(t, `{history: [x], derive: {a: b}}`)) {
		t.Errorf("the project's settings were changed: %v", o)
	}
}

func TestMergedSettingsWithoutProjectSettings(t *testing.T) {
	r := &Role{Name: "committer", Settings: map[string]any{"subject-max": 72}}
	got := r.MergedSettings(&ProjectConfig{})
	if !reflect.DeepEqual(got, map[string]any{"subject-max": 72}) {
		t.Errorf("got %v", got)
	}
}
