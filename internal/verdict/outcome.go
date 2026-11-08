package verdict

// A run's typed outcome (ADR-0063, §6): beside its finding codes, one
// kind the line moves an issue by. Only the codes that move a state get
// a kind, in the table below; the others stay information.

// The outcome kinds.
const (
	// Progress: the run moved the issue, or did its part.
	Progress = "progress"
	// NeedsInput: another role, a person or the engine must act; Of says
	// which.
	NeedsInput = "needs-input"
	// MachineError: the engine's or the agent's run failed; Of says how.
	// It never moves an issue: it is retried under a budget, then waits
	// for a person (ADR-0063, I3).
	MachineError = "machine-error"
)

// What a needs-input waits on (Outcome.Of).
const (
	// FromEngine: the agent lacked what the engine can give, said in its
	// answer: the code it names.
	FromEngine = "engine"
	// FromPerson: a person decides, the run's reason said.
	FromPerson = "person"
	// SizeAsk: the run was too big for its role (ADR-0064, step 3): the
	// product owner is asked on the issue's size first, never a run again
	// on the same input, which crosses the same.
	SizeAsk = "product-owner: size"
)

// Outcome is a run's typed outcome: its kind, what it is of — a machine
// error's kind, a needs-input's source —, and what the run said of it.
type Outcome struct {
	Kind string `json:"kind"`
	Of   string `json:"of,omitempty"`
	Said string `json:"said,omitempty"`
}

// String is the outcome as said: `machine-error(build)`.
func (o Outcome) String() string {
	if o.Of == "" {
		return o.Kind
	}
	return o.Kind + "(" + o.Of + ")"
}

// Machine says whether the outcome is a machine error.
func (o Outcome) Machine() bool { return o.Kind == MachineError }

// Crossed says whether the run was too big for its role: the product
// owner is asked on the issue's size (ADR-0064, §2).
func (o Outcome) Crossed() bool { return o.Kind == NeedsInput && o.Of == SizeAsk }

// Size was the machine error of a task too big for the role's context
// (#1160), as a record from before step 3 of ADR-0064 keeps it: read as
// spent at once, never run again.
const Size = "size"

// machineKinds are the finding codes that are a machine error, by its
// kind: a run that failed, an answer the engine could not read or use,
// code or tests that do not build. Each moved a state before, as a
// verdict on the issue (#1048, #1075, #1017, #953): now retried, never a
// move. A size is none: the product owner's to read (ADR-0064, step 3).
var machineKinds = map[string]string{
	"engine-error":         "run",    // the run failed: any shape, a new one too (principle 12)
	"agent-invalid-output": "answer", // the agent's answer did not read
	"no-test-written":      "answer", // the tester's agent answered nothing it could use, or was not asked
	"patch-unreadable":     "patch",  // a patch that does not apply
	"build-broken":         "build",  // tests or code that do not build
	"no-new-test":          "report", // the test report names none of the tests written
}

// NeedsCode is the finding a role leaves when its agent named code it
// lacked, read at its next run: the engine gives it then.
const NeedsCode = "needs-code"

// The findings a run too big for its role leaves: its agent not asked for
// its prompt's size, on the first call alone (a call asked again over the
// budget with its feedback is prompt-over-budget-asked-again, not this);
// its token cap reached.
const (
	overBudgetRule = "prompt-over-budget"
	tokenCapRule   = "ai-max-tokens"
)

// OutcomeOf is the outcome of a run that ended with status and findings
// (ADR-0063, §6; ADR-0064, §2):
//   - the product owner asked on its size when its agent was not asked
//     for its prompt's size, whatever the status says: it judged nothing;
//     or when it blocked at its token cap;
//   - else a machine error by the first finding whose code is one, a run
//     blocked for it;
//   - a needs-input of the engine when a finding says the code it needs;
//   - a progress for a run that passed;
//   - the product owner asked on its size for a blocked run that cut what
//     its agent read: its block may be the cut's;
//   - else, a block on the issue, the person's to settle.
func OutcomeOf(status string, findings []Finding) Outcome {
	if o, ok := sized(findings, overBudgetRule); ok {
		return o
	}
	if status == Block {
		if o, ok := sized(findings, tokenCapRule); ok {
			return o
		}
		for _, f := range findings {
			if kind, ok := machineKinds[f.Rule]; ok && f.Blocks() {
				return Outcome{Kind: MachineError, Of: kind, Said: f.Message}
			}
		}
	}
	for _, f := range findings {
		if f.Rule == NeedsCode {
			return Outcome{Kind: NeedsInput, Of: FromEngine, Said: f.Message}
		}
	}
	if status == Pass {
		return Outcome{Kind: Progress}
	}
	if o, ok := sized(findings, CutRule); ok {
		return o
	}
	said := ""
	for _, f := range findings {
		if f.Blocks() {
			said = f.Message
			break
		}
	}
	return Outcome{Kind: NeedsInput, Of: FromPerson, Said: said}
}

// sized is the product owner asked on the run's size when a finding of
// rule is among findings: said by the role's own, which blocks, when it
// gives one; the engine's warning else.
func sized(findings []Finding, rule string) (Outcome, bool) {
	at := -1
	for i, f := range findings {
		if f.Rule == rule && (at < 0 || f.Blocks() && !findings[at].Blocks()) {
			at = i
		}
	}
	if at < 0 {
		return Outcome{}, false
	}
	return Outcome{Kind: NeedsInput, Of: SizeAsk, Said: findings[at].Message}, true
}
