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
	m, e, in, out, c := answeredBy([]call{{"low", "claude-haiku-4-5", 1000, 50, 0.01}, {"low", "claude-sonnet-5", 2000, 70, 0.02}})
	if m != "claude-haiku-4-5>claude-sonnet-5" || e != "low" || in != "3000" || out != "120" || c != "0.0300" {
		t.Errorf("got %q %q %q %q %q", m, e, in, out, c)
	}
}

func TestJudge(t *testing.T) {
	c := &caseFile{}
	c.About, c.Run.Message = "a rewrite", "fix: stop crashing on an empty file"
	r := &run{repo: t.TempDir(), message: "refactor: file handling"}
	answer := func(note string) string {
		return `cmd:cat > /dev/null; echo 'model: a-judge' > "$WORKLINE_CALL"; echo '- note: "` + note + `"'`
	}
	t.Setenv("WORKLINE_EVAL", "claude")
	t.Setenv("WORKLINE_JUDGE", answer("no: the crash is gone from it"))
	if why, err := judge("Same meaning?", c, r); err != nil || why != "the crash is gone from it" || r.judgedBy != "a-judge" {
		t.Errorf("a no is a lost point, with its reason: %q %v %q", why, err, r.judgedBy)
	}
	t.Setenv("WORKLINE_JUDGE", answer("yes: same meaning"))
	if why, err := judge("Same meaning?", c, r); err != nil || why != "" {
		t.Errorf("a yes is a point: %q %v", why, err)
	}
	t.Setenv("WORKLINE_JUDGE", answer("maybe"))
	if _, err := judge("Same meaning?", c, r); err == nil {
		t.Error("neither yes nor no is no verdict")
	}
	t.Setenv("WORKLINE_JUDGE", "claude:opus")
	if _, err := judge("Same meaning?", c, r); err == nil {
		t.Error("a judge of the graded agent's provider is refused")
	}
	t.Setenv("WORKLINE_JUDGE", "")
	c.Grade = []map[string]any{{"judge": "Same meaning?"}}
	if passed, failed, skipped := grade(c, r); passed != 0 || len(failed) != 0 || len(skipped) != 1 {
		t.Errorf("without a judge, the check is skipped: %d %v %v", passed, failed, skipped)
	}
}
