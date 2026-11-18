package devmark

import (
	"fmt"
	"strings"
)

// Proof is what the project's tests said on the developer's code.
type Proof struct {
	Result  string   `json:"result"`
	Command string   `json:"command,omitempty"`
	Passed  int      `json:"passed"`
	Red     []string `json:"red,omitempty"`
	// Why: unproven, why it did not run whole; red with no test red, why
	// the command failed.
	Why string `json:"why,omitempty"`
	// Tree: where it ran, "fresh" — a checkout of At apart, never its
	// agent's worktree; At: that commit.
	Tree string `json:"tree,omitempty"`
	At   string `json:"at,omitempty"`
	// Scope: "touched", the proof-command on Packages, the folders of
	// its code and of the acceptance tests (#703); "whole", the
	// test-command, Why so in Whole.
	Scope    string   `json:"scope,omitempty"`
	Packages []string `json:"packages,omitempty"`
	Whole    string   `json:"whole,omitempty"`
	// Missing: the tools the test command's output says were missing
	// where it ran (#760): a red from the environment, not the code.
	Missing []string `json:"missing,omitempty"`
	// Test, Runs and Ran: a flaky-n-of-n proof (ADR-0059, decision 4),
	// the test it ran, the runs asked, the runs made; green only when
	// each of Runs was green, Test passing.
	Test string `json:"test,omitempty"`
	Runs int    `json:"runs,omitempty"`
	Ran  int    `json:"ran,omitempty"`
}

// The proof's scopes.
const (
	Touched = "touched"
	Whole   = "whole"
)

// Cap is a bound on the run reached: the setting, what was spent, what it
// allows.
type Cap struct {
	Setting string `json:"setting"`
	Spent   int    `json:"spent"`
	Allows  int    `json:"allows"`
}

// NOfN says a green flaky-n-of-n proof's runs; "" for another proof.
func (p Proof) NOfN() string {
	if p.Runs == 0 || p.Result != Green {
		return ""
	}
	return fmt.Sprintf("; `%s` passed %d runs of %d", p.Test, p.Ran, p.Runs)
}

// ProofLines are a proof's lines, as its pull request's body says them.
func ProofLines(p Proof, caps []Cap) string {
	var b strings.Builder
	switch p.Result {
	case Green:
		fmt.Fprintf(&b, "- `%s`: %d test%s passed, none red%s.\n", p.Command, p.Passed, Plural(p.Passed), p.NOfN())
	case Red:
		fmt.Fprintf(&b, "- `%s`: %d passed, %d red.\n", p.Command, p.Passed, len(p.Red))
		for _, t := range p.Red {
			fmt.Fprintf(&b, "  - red: `%s`\n", t)
		}
		if p.Why != "" {
			fmt.Fprintf(&b, "  - %s\n", p.Why)
		}
		if len(p.Missing) > 0 {
			fmt.Fprintf(&b, "  - Not found where it ran: `%s`. A tool missing there is a red from the environment, not the code: the project's setup steps install what its test job has.\n", strings.Join(p.Missing, "`, `"))
		}
	default:
		fmt.Fprintf(&b, "- The project's tests did not run whole: %s\n", p.Why)
		if len(p.Missing) > 0 {
			fmt.Fprintf(&b, "  - Not found where it ran: `%s`.\n", strings.Join(p.Missing, "`, `"))
		}
	}
	switch p.Scope {
	case Touched:
		fmt.Fprintf(&b, "- On the folders its code and the acceptance tests touch: `%s`. This pull request's own CI runs the whole suite.\n", strings.Join(p.Packages, " "))
	case Whole:
		fmt.Fprintf(&b, "- The whole suite: %s (the developer's `proof-command`, with `{packages}`, runs the touched folders alone).\n", p.Whole)
	}
	if p.Tree == "fresh" && p.At != "" {
		fmt.Fprintf(&b, "- Run on a fresh checkout of %s, its code laid on it: not its agent's worktree.\n", ShortSHA(p.At))
	}
	for _, c := range caps {
		fmt.Fprintf(&b, "- Cap reached: %s, %d spent of %d.\n", c.Setting, c.Spent, c.Allows)
	}
	return b.String()
}

func ShortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func Plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Said is the proof in a few words: its tests passed, or what failed.
func (p Proof) Said() string {
	switch {
	case p.Result == Green:
		return fmt.Sprintf("%d test%s passed", p.Passed, Plural(p.Passed))
	case len(p.Red) > 0:
		return fmt.Sprintf("%d test%s red: %s", len(p.Red), Plural(len(p.Red)), strings.Join(p.Red, ", "))
	case p.Why != "":
		return p.Why
	}
	return p.Result
}
