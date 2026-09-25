package evaluation

import "testing"

func TestChecks(t *testing.T) {
	if k := keptWords("ci: run the project's line in the templates", "ci: run the project's line in templates"); k < 0.85 {
		t.Errorf("cutting a word keeps the others: %.2f", k)
	}
	if k := keptWords("ci: run the project's line in the templates", "ci: refactor templates to use line routing"); k > 0.5 {
		t.Errorf("rewording keeps few of them: %.2f", k)
	}
	c := &caseFile{}
	c.Run.Message = "ci: run the line"
	if why := check("no-vague-words", true, c, &run{message: "ci: refactor the line"}); why == "" {
		t.Error("a vague word brought in is caught")
	}
	if why := check("no-vague-words", true, c, &run{message: "ci: run the line in the templates"}); why != "" {
		t.Error(why)
	}
}

func TestAnsweredBy(t *testing.T) {
	m, e, c := answeredBy([]call{{"low", "claude-haiku-4-5", 0.01}, {"low", "claude-sonnet-5", 0.02}})
	if m != "claude-haiku-4-5>claude-sonnet-5" || e != "low" || c != "0.0300" {
		t.Errorf("got %q %q %q", m, e, c)
	}
}
