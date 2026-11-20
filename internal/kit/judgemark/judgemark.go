// Package judgemark is what a judge's question and answer are, as the
// engine, the sample and the roles that ask the judge read them: how far
// a judge stands from the work's author, which judge to pick, its answer,
// what it asks to read and the quote a no rests on (ADR-0005, ADR-0045).
// The judge itself is an engine service (internal/judge), not a role.
package judgemark

import (
	"strings"

	"github.com/JN0V/workline/internal/agent"
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
	// A judge that may read (AskReading, ADR-0045): instead of a verdict,
	// what it asks to read; a no's quote; or unsure, when it cannot tell.
	Reads  []Read
	Quote  *Quote
	Unsure bool
}

// Read is what a judge asks to read before it answers (ADR-0045): a
// file's lines, a definition by its name, or the places a word is
// written. The engine serves it from git, at the commit judged.
type Read struct {
	Path  string `yaml:"path,omitempty" json:"path,omitempty"`
	Lines string `yaml:"lines,omitempty" json:"lines,omitempty"` // "40-80", or one line; none: the whole file
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	Grep  string `yaml:"grep,omitempty" json:"grep,omitempty"`
	Why   string `yaml:"why,omitempty" json:"why,omitempty"`
}

// String is how a read is said: calc/sum.go:1-10, `sum`, grep `total`.
func (r Read) String() string {
	switch {
	case r.Path != "" && r.Lines != "":
		return r.Path + ":" + r.Lines
	case r.Path != "":
		return r.Path
	case r.Name != "":
		return "`" + r.Name + "`"
	case r.Grep != "":
		return "grep `" + r.Grep + "`"
	}
	return "nothing named"
}

// Quote is the code a no rests on, as it reads, with its path: the engine
// finds it again before it drops a finding (ADR-0045).
type Quote struct {
	Path string `yaml:"path" json:"path"`
	Text string `yaml:"text" json:"text"`
}
