// Package judge asks an agent one yes-or-no question on what a role
// produced, at the best independence available from the agent that produced
// it (ADR-0005): another provider, another model of it, or the same model in
// a context of its own. The engine asks it when a role's judge step has a
// question no check can answer; the evaluation, to grade.
package judge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
	"go.yaml.in/yaml/v3"
)

// Levels of independence, strongest first (ADR-0005).
var Levels = map[string]int{"provider": 3, "model": 2, "context": 1}

// Provider is the provider part of an agent spec ("claude" in "claude:opus").
func Provider(spec string) string {
	p, _, _ := strings.Cut(spec, ":")
	return p
}

// Independence is how far a judge stands from the models that produced the
// work: another provider, another model of it, or the same model.
func Independence(judgeSpec, judgeModel, authorSpec string, authorModels []string) string {
	if p := Provider(judgeSpec); p == "cmd" || p != Provider(authorSpec) {
		return "provider"
	}
	for _, m := range authorModels {
		if m == judgeModel {
			return "context"
		}
	}
	return "model"
}

// Pick is the judge for work an agent produced, when no judge is named: of
// Claude, the other of its two strongest models; otherwise the same agent,
// in a context of its own.
func Pick(authorSpec, authorModel string) string {
	if Provider(authorSpec) != "claude" {
		return authorSpec
	}
	if strings.Contains(authorModel, "opus") {
		return "claude:sonnet"
	}
	return "claude:opus"
}

// Answer is what the judge said.
type Answer struct {
	Yes   bool
	Why   string     // the reason, one line
	Model string     // the model that answered
	Call  agent.Call // the call, with its tokens
}

// Ask puts question and material to the agent spec names, in a run folder of
// its own, as the judge role.
func Ask(spec string, judgeRole *role.Role, question, material string) (Answer, error) {
	ag, err := agent.Parse(spec)
	if err != nil {
		return Answer{}, err
	}
	if ag == nil {
		return Answer{}, errors.New("no agent to judge with")
	}
	dir, err := os.MkdirTemp("", "workline-judge-")
	if err != nil {
		return Answer{}, err
	}
	defer os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, "in"), 0o755)
	os.MkdirAll(filepath.Join(dir, "out"), 0o755)
	task := fmt.Sprintf("## Question\n\n%s\n\n%s\n", question, material)
	if err := os.WriteFile(filepath.Join(dir, "in", "task.md"), []byte(task), 0o644); err != nil {
		return Answer{}, err
	}
	call, err := ag.Propose(agent.Request{RunDir: dir, Repo: dir, Role: judgeRole})
	call.Task = "judge"
	a := Answer{Model: call.Model, Call: call}
	if errors.Is(err, agent.ErrInvalidOutput) {
		// A note left unquoted is no YAML ("- note: no: …"), yet it says its verdict.
		raw, _ := os.ReadFile(filepath.Join(dir, "out", "agent-answer.txt"))
		a.Yes, a.Why, err = Verdict(string(raw))
		return a, err
	}
	if err != nil {
		return a, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "out", "intentions.yaml"))
	if err != nil {
		return a, err
	}
	var notes []map[string]string
	if err := yaml.Unmarshal(data, &notes); err != nil || len(notes) == 0 || notes[0]["note"] == "" {
		return a, fmt.Errorf("the judge answered no note: %.200s", data)
	}
	a.Yes, a.Why, err = Verdict(notes[0]["note"])
	return a, err
}

// Verdict reads the judge's note: "yes: why" or "no: why". The note may come
// raw, with its YAML around it.
func Verdict(note string) (yes bool, why string, err error) {
	s := strings.TrimSpace(note)
	s = strings.TrimSpace(strings.TrimPrefix(s, "- note:"))
	if len(s) > 1 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		s = s[1 : len(s)-1] // the note quoted whole
	}
	v, reason, _ := strings.Cut(s, ":")
	reason = strings.Join(strings.Fields(reason), " ")
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes":
		return true, reason, nil
	case "no":
		return false, reason, nil
	}
	return false, "", fmt.Errorf("the judge said neither yes nor no: %.200s", strings.Join(strings.Fields(note), " "))
}
