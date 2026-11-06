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

// Size is the machine error of a task too big for the role's context:
// the same input gives the same prompt, so a run again fails the same
// (#1160). It is never retried: the issue waits for a person at once.
const Size = "size"

// Retried says whether the line runs the step again for this outcome: a
// machine error a run again may mend, never one of size.
func (o Outcome) Retried() bool { return o.Machine() && o.Of != Size }

// machineKinds are the finding codes that are a machine error, by its
// kind: a run that failed, an answer the engine could not read or use,
// code or tests that do not build, a budget spent before the agent was
// asked. Each moved a state before, as a verdict on the issue (#1048,
// #1075, #1017, #953): now retried, never a move; but a size, which waits
// for a person at once (#1160).
var machineKinds = map[string]string{
	"engine-error":         "run",    // the run failed: any shape, a new one too (principle 12)
	"agent-invalid-output": "answer", // the agent's answer did not read
	"no-test-written":      "answer", // the tester's agent answered nothing it could use, or was not asked
	"patch-unreadable":     "patch",  // a patch that does not apply
	"build-broken":         "build",  // tests or code that do not build
	"no-new-test":          "report", // the test report names none of the tests written
	"prompt-over-budget":   Size,     // the task over the role's context budget: the agent not asked
}

// NeedsCode is the finding a role leaves when its agent named code it
// lacked, read at its next run: the engine gives it then.
const NeedsCode = "needs-code"

// OutcomeOf is the outcome of a run that ended with status and findings:
// a machine error by the first finding whose code is one, whatever the
// status says else — a run blocked for it, or one that passed saying it;
// a needs-input of the engine when a finding says the code it needs; a
// progress for a run that passed; else, a block on the issue, the
// person's to settle.
func OutcomeOf(status string, findings []Finding) Outcome {
	if status == Block {
		// A run that blocked, its agent not asked for its prompt's size:
		// that is why, whatever else it says (#1160). Said by the role's
		// own finding when it gives one, the engine's else. The engine's
		// is a warning, counted on purpose: it is said on the first call
		// alone, the input's own size; a call asked again over the budget
		// with its feedback is prompt-over-budget-asked-again, not this.
		size := -1
		for i, f := range findings {
			if f.Rule == "prompt-over-budget" && (size < 0 || f.Blocks() && !findings[size].Blocks()) {
				size = i
			}
		}
		if size >= 0 {
			return Outcome{Kind: MachineError, Of: Size, Said: findings[size].Message}
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
	said := ""
	for _, f := range findings {
		if f.Blocks() {
			said = f.Message
			break
		}
	}
	return Outcome{Kind: NeedsInput, Of: FromPerson, Said: said}
}
