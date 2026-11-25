package work

import (
	"slices"
	"testing"
)

// issue851 is the Validation of a real issue, as its author wrote it: each
// scenario a bold line, its steps bullets below.
const issue851 = "**Scenario: a leading `./` matches**\n- Given the pattern `./docs/archive/**`\n- When it is matched against `docs/archive/old.md`\n- Then it matches\n\n" +
	"**Scenario: a doubled slash matches**\n- Given the pattern `docs//archive/**`\n- When it is matched against `docs/archive/old.md`\n- Then it matches\n\n" +
	"**Scenario: a file at the root**\n- Given the pattern `./*.md`\n- When it is matched against `README.md`\n- Then it matches\n\n" +
	"**Scenario: other files still do not match**\n- Given the pattern `./docs/archive/**`\n- When it is matched against `src/a.go`\n- Then it does not match\n\n" +
	"**Scenario: today's patterns unchanged**\n- Given the patterns already tested, such as `docs/**` against `docs` and `src/**` against `../src/a`\n- When they are matched\n- Then each gives the same answer as today\n"

func TestScenariosForms(t *testing.T) {
	oldDebt := Scenario{Name: "Old debt", Text: "Given 140 findings, When one is added, Then only it blocks"}
	for _, tc := range []struct {
		form, validation string
		want             []Scenario
	}{
		{"workline's bullet", "Intro.\n\n- **Old debt:** Given 140 findings,\n  When one is added, Then only it blocks\n- Given no name, When read, Then named by its Then.\n",
			[]Scenario{{Name: "Old debt", Text: "Given 140 findings, When one is added, Then only it blocks"},
				{Name: "named by its Then", Text: "Given no name, When read, Then named by its Then."}}},
		{"colon after the bold", "- **Old debt**: Given 140 findings, When one is added, Then only it blocks\n", []Scenario{oldDebt}},
		{"numbered", "1. **Old debt:** Given 140 findings, When one is added, Then only it blocks\n", []Scenario{oldDebt}},
		{"bold scenario line, step bullets", "**Scenario: Old debt**\n- Given 140 findings\n- When one is added\n- Then only it blocks\n", []Scenario{oldDebt}},
		{"keyword out of the bold", "**Scenario:** Old debt\n- Given 140 findings\n- When one is added\n- Then only it blocks\n", []Scenario{oldDebt}},
		{"plain Gherkin lines", "Scenario: Old debt\n  Given 140 findings\n  When one is added\n  Then only it blocks\n", []Scenario{oldDebt}},
		{"fenced Gherkin", "```gherkin\nFeature: the gate\n\n  Background:\n    Given a repository\n\n  Scenario: Old debt\n    Given 140 findings\n    When one is added\n    Then only it blocks\n```\n", []Scenario{oldDebt}},
		{"a heading", "#### Scenario: Old debt\n\n- Given 140 findings\n- When one is added\n- Then only it blocks\n", []Scenario{oldDebt}},
		{"a heading with no keyword", "#### Old debt\n\nGiven 140 findings\nWhen one is added\nThen only it blocks\n", []Scenario{oldDebt}},
		{"a bullet, steps nested", "- **Scenario: Old debt**\n  - Given 140 findings\n  - When one is added\n  - Then only it blocks\n", []Scenario{oldDebt}},
		{"a bold name alone, step bullets", "**Old debt**\n- Given 140 findings\n- When one is added\n- Then only it blocks\n", []Scenario{oldDebt}},
		{"an outline", "Scenario Outline: Old debt\n  Given <n> findings\n  When one is added\n  Then only it blocks\n\n  Examples:\n    | n |\n    | 140 |\n",
			[]Scenario{{Name: "Old debt", Text: "Given <n> findings, When one is added, Then only it blocks Examples: | n | | 140 |"}}},
		{"a title over named bullets", "**Must hold:**\n\n- **Old debt:** Given 140 findings, When one is added, Then only it blocks\n", []Scenario{oldDebt}},
		{"a bullet named, nothing under it", "- **Scenario: Old debt**\n- **Moved code:**\n", nil},
		{"a heading over prose", "#### Notes\n\nThe total is right when Ann opens it.\n", nil},
		{"Given / When / Then with no name, named by its Then", "Given 140 findings, when one is added, then only it blocks.\n",
			[]Scenario{{Name: "only it blocks", Text: "Given 140 findings, when one is added, then only it blocks."}}},
		{"a bullet named Example", "- **Example:** Given 140 findings, When one is added, Then only it blocks\n",
			[]Scenario{{Name: "Example", Text: "Given 140 findings, When one is added, Then only it blocks"}}},
	} {
		if got := Scenarios(tc.validation); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %+v, want %+v", tc.form, got, tc.want)
		}
	}
}

func TestScenariosOfIssue851(t *testing.T) {
	got := Scenarios(issue851)
	names := []string{"a leading `./` matches", "a doubled slash matches", "a file at the root", "other files still do not match", "today's patterns unchanged"}
	if len(got) != len(names) {
		t.Fatalf("read %d scenarios, want %d: %+v", len(got), len(names), got)
	}
	for i, n := range names {
		if got[i].Name != n {
			t.Errorf("scenario %d named %q, want %q", i, got[i].Name, n)
		}
	}
	if want := "Given the pattern `./docs/archive/**`, When it is matched against `docs/archive/old.md`, Then it matches"; got[0].Text != want {
		t.Errorf("its text %q, want %q", got[0].Text, want)
	}
}

func TestHeadingNamingAScenarioIsNoSection(t *testing.T) {
	body := "## Validation\n\n### Scenario: Old debt\n\n- Given 140 findings\n- When one is added\n- Then only it blocks\n\n## Scope\n\nsrc/gate\n"
	if got := Scenarios(ByMeaning(body)[Validation].Text); len(got) != 1 || got[0].Name != "Old debt" {
		t.Errorf("a ### Scenario heading inside Validation: read %+v", got)
	}
}
