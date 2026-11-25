package work

import (
	"reflect"
	"testing"
)

// Given a Validation written as Given / When / Then with no name — a
// bullet each, a line each, or a step a bullet —, when it is read, then
// each scenario is named by the first words of its Then, never dropped.
func TestScenariosUnnamedNamedByTheirThen(t *testing.T) {
	for name, c := range map[string]struct {
		in   string
		want []string
	}{
		"bullets":       {"- Given the library, When I multiply 2 by 3, Then I get 6.\n- Given the library, When I multiply 5 by 0, Then I get 0.", []string{"I get 6", "I get 0"}},
		"lines":         {"Given a day of three sales, when Ann opens it, then she counts three.", []string{"she counts three"}},
		"steps":         {"- Given a day of three sales\n- When Ann opens the report\n- Then she counts three sales in it today", []string{"she counts three sales in it"}},
		"no then":       {"- Given a day of three sales, When Ann opens it.", nil},
		"given prose":   {"Given the cache is warm, a test checks the total when it is opened.", nil},
		"then in prose": {"The total is given to Ann when she opens it, then she leaves.", nil},
		"prose after":   {"Given a day\nWhen Ann opens it\nThen she counts three\n\nThe report is shared with the team.", []string{"she counts three"}},
		"note after":    {"- Given a day\n- When Ann opens it\n- Then she counts three\n\nNotes: then the team reads it.", []string{"she counts three"}},
		"alike":         {"- Given three, When Ann opens it, Then the total shown on the report is 3.\n- Given none, When Ann opens it, Then the total shown on the report is 0.", []string{"the total shown on the report", "the total shown on the report (2)"}},
		"after a named": {"- **A:** Given a, When b, Then c.\n- Given d, When e, Then f.", []string{"A", "f"}},
		"then in when":  {"- Given a day, When Ann opens it then closes it, Then she counts three.", []string{"she counts three"}},
		"numbered":      {"1. Given a day, When Ann opens it, Then she counts three.", []string{"she counts three"}},
		"named":         {"- **Three sales:** Given a day of three sales, When Ann opens it, Then she counts three.", []string{"Three sales"}},
		"prose":         {"Ann sees three sales.", nil},
	} {
		var got []string
		for _, s := range Scenarios(c.in) {
			got = append(got, s.Name)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: names %q, want %q", name, got, c.want)
		}
	}
}
