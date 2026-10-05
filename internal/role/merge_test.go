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

func TestMergedSettingsLevel(t *testing.T) {
	r := &Role{Name: "po", Settings: settings(t, `
autonomy: normal
acts:
  rename: {mode: act, max: 5}
  split: {mode: act, max: 2}
`), Levels: map[string]map[string]map[string]any{"autonomy": {
		"cautious":     settings(t, `acts: {rename: {mode: propose}, split: {mode: propose}}`),
		"enterprising": settings(t, `acts: {rename: {max: 10}}`),
	}}}
	project := func(src string) *ProjectConfig {
		var c ProjectConfig
		if err := yaml.Unmarshal([]byte("roles: {po: {settings: "+src+"}}"), &c); err != nil {
			t.Fatal(err)
		}
		return &c
	}
	for _, tc := range []struct{ name, over, want, level string }{
		{"the default level lays nothing", `{}`,
			`{autonomy: normal, acts: {rename: {mode: act, max: 5}, split: {mode: act, max: 2}}}`,
			`{autonomy: normal, acts: {rename: {mode: act, max: 5}, split: {mode: act, max: 2}}}`},
		{"a level is laid over the defaults", `{autonomy: cautious}`,
			`{autonomy: cautious, acts: {rename: {mode: propose, max: 5}, split: {mode: propose, max: 2}}}`,
			`{autonomy: normal, acts: {rename: {mode: propose, max: 5}, split: {mode: propose, max: 2}}}`},
		{"the project's own setting over the level", `{autonomy: cautious, acts: {rename: {mode: act}}}`,
			`{autonomy: cautious, acts: {rename: {mode: act, max: 5}, split: {mode: propose, max: 2}}}`,
			`{autonomy: normal, acts: {rename: {mode: propose, max: 5}, split: {mode: propose, max: 2}}}`},
		{"enterprising", `{autonomy: enterprising}`,
			`{autonomy: enterprising, acts: {rename: {mode: act, max: 10}, split: {mode: act, max: 2}}}`,
			`{autonomy: normal, acts: {rename: {mode: act, max: 10}, split: {mode: act, max: 2}}}`},
		{"a level the role does not have lays nothing", `{autonomy: bold}`,
			`{autonomy: bold, acts: {rename: {mode: act, max: 5}, split: {mode: act, max: 2}}}`,
			`{autonomy: normal, acts: {rename: {mode: act, max: 5}, split: {mode: act, max: 2}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := r.MergedSettings(project(tc.over))
			level := got[ByLevel]
			delete(got, ByLevel)
			if want := settings(t, tc.want); !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
			if want := settings(t, tc.level); !reflect.DeepEqual(level, want) {
				t.Errorf("by level: got %v, want %v", level, want)
			}
		})
	}
}

// Two settings picking levels that set the same field: laid in the order
// of the settings' names, the same on every run.
func TestMergedSettingsTwoLevelsInOrder(t *testing.T) {
	r := &Role{Name: "po", Settings: settings(t, `{autonomy: normal, pace: slow, max: 1}`),
		Levels: map[string]map[string]map[string]any{
			"autonomy": {"enterprising": settings(t, `{max: 5}`)},
			"pace":     {"fast": settings(t, `{max: 9}`)},
		}}
	var c ProjectConfig
	if err := yaml.Unmarshal([]byte("roles: {po: {settings: {autonomy: enterprising, pace: fast}}}"), &c); err != nil {
		t.Fatal(err)
	}
	for range 20 { // a map's order changes from run to run: the result does not
		if got := r.MergedSettings(&c)["max"]; got != 9 {
			t.Fatalf("max %v, want 9: pace laid after autonomy", got)
		}
	}
}
