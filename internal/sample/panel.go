package sample

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/work"
)

// A false verdict is two models disagreeing: the role's and the auditor's.
// Three judges settle it (ADR-0046), each given the material and both
// readings as "a" and "b", never whose, the order drawn from the act; the
// majority settles, a 2–1 vote flagged. Their calls count in the week's
// tokens.

// Panel is how a dispute was settled.
type Panel struct {
	For    string   `json:"for,omitempty"` // act or auditor: the majority's reading; "" when none
	Votes  []string `json:"votes"`         // each judge's: act, auditor, or "" when it did not answer
	Models []string `json:"models"`        // each judge's, as it answered
	Why    string   `json:"why,omitempty"` // why it did not settle
}

// Words say the panel's vote: "2–1 for the act (flagged)".
func (p *Panel) Words() string {
	if p == nil {
		return "—"
	}
	act, auditor := 0, 0
	for _, v := range p.Votes {
		switch v {
		case "act":
			act++
		case "auditor":
			auditor++
		}
	}
	switch p.For {
	case "act":
		return fmt.Sprintf("%d–%d for the act%s", act, auditor, flag(auditor))
	case "auditor":
		return fmt.Sprintf("%d–%d for the auditor%s", auditor, act, flag(act))
	}
	return "no majority: " + p.Why
}

func flag(minority int) string {
	if minority > 0 {
		return " (flagged)"
	}
	return ""
}

// panelSpecs are the three judges: the setting's, else Claude's three
// models when the judge is Claude's; none else, said.
func (rd *reader) panelSpecs() []string {
	if p := rd.settings.Sample.Panel; len(p) > 0 {
		return p
	}
	if judge.Provider(rd.spec) == "claude" {
		return []string{"claude:opus", "claude:sonnet", "claude:haiku"}
	}
	return nil
}

// noPanel is why a false verdict stands as the auditor's alone.
const noPanel = "no panel to settle it (`sample.panel`): the auditor's reading stands, for a person"

// dispute has a disagreement settled: the act's reading against the
// auditor's, on the material the auditor read.
func (rd *reader) dispute(key, actReading, auditorReading, material string) *Panel {
	p := &Panel{}
	if rd.judgeRole == nil {
		r, err := role.Load(rd.o.RolesDir, "judge")
		if err != nil {
			p.Why = "the judge role: " + err.Error()
			return p
		}
		rd.judgeRole = r
	}
	// Which reading is "a" is drawn from the week and the act: never always
	// the role's, so a judge's leaning favours neither.
	sum := sha256.Sum256([]byte(rd.res.Week + "\x00" + key))
	a, b, aIs := actReading, auditorReading, "act"
	if sum[0]%2 == 1 {
		a, b, aIs = auditorReading, actReading, "auditor"
	}
	question := fmt.Sprintf("Two readings of the act below disagree.\n\nReading a: %s\n\nReading b: %s\n\nOn the material below alone, is reading a the right one rather than reading b? Answer yes for reading a, no for reading b.", a, b)
	for _, spec := range rd.panelSpecs() {
		if why := rd.over(); why != "" {
			p.Why = why
			break
		}
		ans, err := judge.Ask(spec, rd.judgeRole, question, material)
		rd.called(ans.Call)
		p.Models = append(p.Models, orUnknown(ans.Call.Agent, ans.Model))
		vote := ""
		switch {
		case err != nil:
			p.Why = "a judge did not answer: " + err.Error()
		case ans.Yes == (aIs == "act"):
			vote = "act"
		default:
			vote = "auditor"
		}
		p.Votes = append(p.Votes, vote)
	}
	act, auditor := 0, 0
	for _, v := range p.Votes {
		switch v {
		case "act":
			act++
		case "auditor":
			auditor++
		}
	}
	switch {
	case act >= 2:
		p.For, p.Why = "act", ""
	case auditor >= 2:
		p.For, p.Why = "auditor", ""
	case p.Why == "":
		p.Why = "the judges did not agree"
	}
	return p
}

// statements are what each kind of act says, as a dispute's readings put
// it: the act's reading says it is so, or, for a finding dropped, that it
// is not; the auditor's, the other, with the material's words.
var statements = map[string]string{
	"checked": "every passage of the doc is true of its sources, given whole as they were at the commit its `checked` names",
	"finding": "the code, as it reads, fails the way the finding says",
	"rewrite": "the commit message says what its diff does, and nothing more",
	"refine":  "the sections the product owner wrote say what the person asked, and nothing more",
	"sources": "the code the product owner named holds what the issue is about",
	"order":   "the issue's words do not contradict the priority, effort or milestone set",
	"ask":     "the question the product owner asked was not already answered on the issue",
	"closing": "the closing holds: the code does what the issue asks, or the other issue holds the same need",
	"split":   "each part holds one need, and together the parent's",
}

// settleByPanel has a false verdict's dispute settled, and says it: the
// act stands when the panel chose its reading. With no panel, the
// auditor's reading stands, for a person.
func (rd *reader) settleByPanel(key, kind string, actSaysSo bool, proofs []Proof, material string, verdict, why *string) *Panel {
	if len(rd.panelSpecs()) == 0 {
		*why = noPanel
		return nil
	}
	said := statements[kind]
	if kind == "ready" {
		said = fmt.Sprintf("the issue was ready: one need, a %s a person wrote, a %s proving it, nothing left open in its comments", work.Name(work.Validation), work.Name(work.Verification))
	}
	so := "Yes: " + said + "."
	not := "No: it is not so that " + said
	var lines []string
	for _, p := range proofs {
		lines = append(lines, p.String())
	}
	act, auditor := so, not+", as these words show: "+strings.Join(lines, "; ")
	if !actSaysSo {
		act, auditor = not+".", strings.TrimSuffix(so, ".")+", as these words show: "+strings.Join(lines, "; ")
	}
	p := rd.dispute(key, act, auditor, material)
	switch p.For {
	case "act":
		*verdict, *why = "true", "the panel chose the act's reading, "+p.Words()
	case "auditor":
		*why = "the panel chose the auditor's reading, " + p.Words()
	default:
		*why = "no panel majority (" + p.Why + "): for a person"
	}
	return p
}
