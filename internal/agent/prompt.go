package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// contracts describes each intention's shape, for the generated output contract.
var contracts = map[string]string{
	"commit-message": "- commit-message: |\n    type(scope): subject\n\n    Body, if any.\n\n    Trailer: value",
	"patch":          "- patch: |\n    a unified diff",
	"comment":        `- comment: "text of the comment"`,
	"label":          `- label: {add: [name], remove: [name]}`,
	"issue":          `- issue: {title: "short title", body: "what is wrong, where, how it was noticed", at: {path: "src/file.go", text: "the line it is about, as it reads"}}  # at: when it is about code; an issue open or closed on that line already, it is not opened again`,
	"close":          `- close: {issue: 12, reason: duplicate, duplicate-of: 7, quote: {issue: 7, text: "the words, as they read"}, why: "why"}  # or reason: obsolete, quote: {path: "src/file.go", text: "the code, as it reads"}`,
	"keep":           `- keep: {issue: 12, why: "what shows it is not solved"}  # an issue the task says is announced obsolete, kept open: a reply or the code shows it is still true`,
	"sources":        `- sources: {issue: 12, sources: ["src/file.go"], quote: {path: "src/file.go", text: "the code, as it reads"}, why: "why this is the code the issue is about"}`,
	"open":           `- open: {title: "short title", quote: {path: "docs/ROADMAP.md", text: "the item's text, as the file has it"}}`,
	"skip":           `- skip: {lines: "12-14", reason: done, quote: {path: "docs/ROADMAP.md", text: "the words that say it is done"}}  # importing only: or reason: held, issue: 7 (an issue holds it); or reason: not-item, why: "an introduction"`,
	"milestone":      `- milestone: {issue: 12, milestone: "v2.14.0", why: "why it belongs to that release"}`,
	"order":          `- order: {issue: 12, priority: 1, why: "why it comes before the others"}  # 1 the most pressing, to 4`,
	"refine":         `- refine: {issue: 12, need: "one line: who needs what, and why (a draft)", example: "one real case, today and wanted (a draft)", validation: "named scenarios, Given / When / Then (a draft)", verification: "the tests that will prove it", scope: "the part of the code", sources: ["src/file.go"], why: "why", questions: "what is still missing (an outsider's issue only)"}  # only the sections it lacks; a bug: steps instead of example — the steps, Expected, Actual, the version (a draft); description: the text above its sections in plain words, only where the task says it is a role's`,
	"ready":          `- ready: {issue: 12, why: "its four sections are there, Need and Validation a person's"}`,
	"unready":        `- unready: {issue: 12, why: "what in the change it no longer fits"}  # a ready issue back to refine: always proposed to a person, never done by the role`,
	"ask":            `- ask: {issue: 12, questions: "what its reporter should say", why: "why the issue cannot be refined without it"}`,
	"split":          `- split: {issue: 12, into: [{title: "a short title", need: "who needs this part, and why (a draft)", verification: "the tests that will prove it", validation: "who accepts it (a draft)", scope: "the part of the code", sources: ["src/file.go"]}, {title: "…", …}], why: "why it is several needs"}  # 2 to 6 children, each its four sections`,
	"depend":         `- depend: {issue: 14, blocked-by: [13], why: "what #14 needs from #13 before it can start"}  # only what it cannot start without; a split's child says it with after: [1], the children it waits on by their place`,
	"undepend":       `- undepend: {issue: 14, blocked-by: [13], why: "why #14 no longer needs #13"}  # only a link the role set; a closed blocker's link the engine takes off itself; proposed to a person`,
	"rename":         `- rename: {issue: 12, title: "about ten words: the problem, not the fix", why: "why the title it has does not say it"}`,
	"handoff":        `- handoff: {role: "next role", reason: "why"}`,
	"note":           `- note: "a message for a person"`,
	"finding":        `- finding: {lens: correctness, severity: important, title: "what is wrong, in a few words", why: "how it fails, and when", cause: {path: "src/file.go", quote: "the line that causes it, as it reads"}, symptom: {path: "src/other.go", quote: "where it shows, as it reads"}, fix: "what would fix it, in a sentence", seen: "what a person using the project sees when it fails, in plain words"}  # lens: the one that found it; severity: important or nit; symptom only when elsewhere; claim: "the author's words, as they read", only for a lens checking claims; a cause in an issue the change closes: path "#4"; decision: "the question for a person", only for a choice that is not a defect`,
	"claim":          `- claim: {lines: "12-14", status: contradicted, quote: "the doc's words, as they read", source: {path: "src/file.go", lines: "40-41", quote: "the source's words, as they read"}, why: "why they disagree"}`,
}

// BlockScalars is how a text holding code is written so that it reads: a
// plain text starting with a backtick, or holding `: ` or ` #`, does not
// (#138), and a block scalar takes code as it is, with no escaping. Said in
// the prompt, and again when an answer did not read.
const BlockScalars = "An item may be written over several lines, one key a line. A text holding code, a quote, a backtick, `: ` or ` #`, or several lines is written as a block scalar: the key, then `|`, then the text on the lines below, indented under the key with spaces (its first line never starting with a tab)."

// facet finds a facet file: the project's copy first, then the user's, then
// the role's own (docs/spec/role-contract.md, "Facets").
func facet(req Request, name string) (string, bool) {
	home, _ := os.UserHomeDir()
	for _, dir := range []string{
		filepath.Join(req.Repo, ".workline", "roles", req.Role.Name),
		filepath.Join(home, ".config", "workline", "roles", req.Role.Name),
		req.Role.Dir,
	} {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return string(data), true
		}
	}
	return "", false
}

// Prompt assembles what the agent reads, in the contract's order: persona
// (returned apart, as the system prompt), knowledge, instruction, the task,
// the output contract, and the policy last.
func Prompt(req Request) (system, user string, err error) {
	system, _ = facet(req, "persona.md")
	if system == "" {
		system = fmt.Sprintf("You are the %s of a software project. %s", req.Role.Name, req.Role.Mission)
	}
	var b strings.Builder
	for _, k := range req.Role.Context.Knowledge {
		text, ok := facet(req, filepath.Join("knowledge", k))
		if !ok {
			return "", "", fmt.Errorf("knowledge file %q not found", k)
		}
		b.WriteString(text + "\n\n")
	}
	instruction, ok := facet(req, "instruction.md")
	if !ok {
		return "", "", fmt.Errorf("role %q has no instruction.md", req.Role.Name)
	}
	b.WriteString(instruction + "\n\n")
	task, err := os.ReadFile(filepath.Join(req.RunDir, "in", "task.md"))
	if err != nil {
		return "", "", err
	}
	b.WriteString("# Task\n\n" + string(task) + "\n\n")
	if fb, err := os.ReadFile(filepath.Join(req.RunDir, "out", "feedback.md")); err == nil {
		b.WriteString("# Your previous answer was refused\n\n" + string(fb) + "\n" +
			"Your new answer replaces it whole: give again, unchanged, every proposal and every change of it not refused above, and change only what is refused.\n\n")
	}
	b.WriteString("# Answer\n\nAnswer with YAML only, no prose and no code fence: a list of proposals, each item holding exactly one of:\n\n")
	for _, k := range req.Role.Intentions {
		b.WriteString(contracts[k] + "\n")
	}
	b.WriteString("\n" + BlockScalars + "\n\nYou cannot act, only propose; anything else is refused.\n\n")
	if policy, ok := facet(req, "policy.md"); ok {
		b.WriteString("# Rules\n\n" + policy)
	}
	user = b.String()
	if budget := req.Role.Context.Budget; budget > 0 {
		chars := len(system) + len(user)
		if tokens := Tokens(chars); tokens > budget {
			return "", "", fmt.Errorf("%w: about %d tokens (%d characters at %s), over the role's budget of %d", ErrOverBudget, tokens, chars, TokensRatio, budget)
		}
	}
	return system, user, nil
}

// Tokens estimates what a prompt of so many characters (bytes) costs, before
// the call, with no tokenizer: 711 + 0.82 a character. Fitted on ten real
// calls of the reviewer, Go code and diffs, Claude Sonnet and Opus, 87k to
// 117k characters (#147, roles/reviewer/docs/tried.md, 2026-10-06), within
// 2% of what Claude reported. On the documentalist's evaluation calls
// (Sonnet and Opus, 6k to 43k characters) it reads Markdown 5 to 15% high
// and C++ sources 10% low (#235): one ratio, code and prose alike, since
// prose here is far from four characters a token.
func Tokens(chars int) int { return 711 + chars*82/100 }

// TokensRatio says the estimate, in the refusal.
const TokensRatio = "711 + 0.82 a character"
