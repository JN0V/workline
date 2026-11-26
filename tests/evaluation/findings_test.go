package evaluation

// What the meaning scorer reads (tests/evaluation/meaning): a reviewer
// run's findings as their author reads them, a JSON line a case run, kept
// when WORKLINE_EVAL_FINDINGS names a file. The run folder goes with the
// test, and the log holds titles only.

import (
	"encoding/json"
	"os"
	"strings"

	reviewer "github.com/JN0V/workline/roles/reviewer/code"
)

// findingBody is a finding's words past its title.
func findingBody(f reviewer.Finding) string {
	parts := []string{strings.TrimSpace(f.Why)}
	if f.Fix != "" {
		parts = append(parts, "Fix: "+strings.TrimSpace(f.Fix))
	}
	if f.Seen != "" {
		parts = append(parts, "Seen: "+strings.TrimSpace(f.Seen))
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// keepFindings appends the run's findings to the file
// WORKLINE_EVAL_FINDINGS names; unset, nothing is kept. The arm is
// WORKLINE_EVAL_ARM, or the agent asked: the scorer's judge never sees it.
func keepFindings(c *caseFile, r *run, rr *reviewRun) error {
	file := os.Getenv("WORKLINE_EVAL_FINDINGS")
	if file == "" {
		return nil
	}
	type finding struct {
		Place    string `json:"place"`
		Severity string `json:"severity"`
		Refused  bool   `json:"refused"`
		Title    string `json:"title"`
		Body     string `json:"body"`
	}
	line := struct {
		Arm       string    `json:"arm"`
		Case      string    `json:"case"`
		TokensIn  int       `json:"tokens_in"`
		TokensOut int       `json:"tokens_out"`
		Findings  []finding `json:"findings"`
	}{Arm: os.Getenv("WORKLINE_EVAL_ARM"), Case: strings.TrimPrefix(c.Case, "reviewer/"), Findings: []finding{}}
	if line.Arm == "" {
		line.Arm = os.Getenv("WORKLINE_EVAL")
	}
	for _, call := range r.res.Calls {
		line.TokensIn, line.TokensOut = line.TokensIn+call.TokensIn, line.TokensOut+call.TokensOut
	}
	for i, f := range rr.Findings {
		body := ""
		if i < len(rr.Bodies) {
			body = rr.Bodies[i]
		}
		line.Findings = append(line.Findings, finding{f.Where, f.Severity, f.Judged == "no", f.Title, body})
	}
	data, err := json.Marshal(line)
	if err != nil {
		return err
	}
	record.Lock()
	defer record.Unlock()
	out, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = out.Write(append(data, '\n'))
	return err
}
