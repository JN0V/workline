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
	"issue":          `- issue: {title: "short title", body: "what is wrong, where, how it was noticed"}`,
	"release":        `- release: {version: "the version you were given", notes: "release notes"}`,
	"handoff":        `- handoff: {role: "next role", reason: "why"}`,
	"note":           `- note: "a message for a person"`,
}

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
		b.WriteString("# Your previous answer was refused\n\n" + string(fb) + "\n")
	}
	b.WriteString("# Answer\n\nAnswer with YAML only, no prose and no code fence: a list of proposals, each item holding exactly one of:\n\n")
	for _, k := range req.Role.Intentions {
		b.WriteString(contracts[k] + "\n")
	}
	b.WriteString("\nYou cannot act, only propose; anything else is refused.\n\n")
	if policy, ok := facet(req, "policy.md"); ok {
		b.WriteString("# Rules\n\n" + policy)
	}
	user = b.String()
	if budget := req.Role.Context.Budget; budget > 0 {
		if tokens := (len(system) + len(user)) / 4; tokens > budget {
			return "", "", fmt.Errorf("prompt is about %d tokens, over the role's budget of %d", tokens, budget)
		}
	}
	return system, user, nil
}
