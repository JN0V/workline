// Package judge asks an agent one yes-or-no question on what a role
// produced, at the best independence available from the agent that produced
// it (ADR-0005): another provider, another model of it, or the same model in
// a context of its own. The engine asks it when a role's judge step has a
// question no check can answer; the evaluation, to grade. A judge asked so
// (AskReading) may first ask to read code, and answer unsure (ADR-0045).
package judge

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

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

// readingIntentions are what a judge that may read answers with, besides
// its note: its reads, and the quote a no rests on.
var readingIntentions = []string{"read", "quote"}

// Ask puts question and material to the agent spec names, in a run folder of
// its own, as the judge role.
func Ask(spec string, judgeRole *role.Role, question, material string) (Answer, error) {
	return ask(spec, judgeRole, question, material, false)
}

// AskReading puts a question whose judge may read before it answers
// (ADR-0045): it answers a note — yes, no or unsure — or asks to read, and
// quotes the lines a no rests on. The question says what it may still read.
func AskReading(spec string, judgeRole *role.Role, question, material string) (Answer, error) {
	return ask(spec, readingRole(judgeRole), question, material, true)
}

// readingRole is the judge as a question that lets it read asks it: its
// facets as they are, its output contract showing reads and a quote too.
// How to answer so is the question's to say, never the facets': every
// other judge's prompt stays as it was.
func readingRole(judgeRole *role.Role) *role.Role {
	reading := *judgeRole
	reading.Intentions = append(slices.Clone(judgeRole.Intentions), readingIntentions...)
	return &reading
}

// writeTask writes what a judge is asked: the question, then its material.
func writeTask(dir, question, material string) error {
	task := fmt.Sprintf("## Question\n\n%s\n\n%s\n", question, material)
	return os.WriteFile(filepath.Join(dir, "in", "task.md"), []byte(task), 0o644)
}

func ask(spec string, judgeRole *role.Role, question, material string, reading bool) (Answer, error) {
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
	if err := writeTask(dir, question, material); err != nil {
		return Answer{}, err
	}
	start := time.Now()
	call, err := ag.Propose(agent.Request{RunDir: dir, Repo: dir, Role: judgeRole})
	call.Task = "judge"
	call.Seconds = math.Round(time.Since(start).Seconds()*10) / 10 // what a judge took, in the run's summary (ADR-0050)
	a := Answer{Model: call.Model, Call: call}
	if errors.Is(err, agent.ErrInvalidOutput) {
		// A note left unquoted is no YAML ("- note: no: …"), yet it says its verdict.
		raw, _ := os.ReadFile(filepath.Join(dir, "out", "agent-answer.txt"))
		if reading { // its reads and its quote, item by item
			return answerReading(a, raw)
		}
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
	if reading {
		return answerReading(a, data)
	}
	var notes []map[string]string
	if err := yaml.Unmarshal(data, &notes); err != nil || len(notes) == 0 || notes[0]["note"] == "" {
		return a, fmt.Errorf("the judge answered no note: %.200s", data)
	}
	a.Yes, a.Why, err = Verdict(notes[0]["note"])
	return a, err
}

// answerReading reads the answer of a judge that may read: a yes or a no
// is its verdict, a no with the quote it rests on; else its reads, if it
// asks any; else unsure.
func answerReading(a Answer, data []byte) (Answer, error) {
	var items []map[string]any
	if err := yaml.Unmarshal(data, &items); err != nil {
		items = itemsOf(string(data))
	}
	note, said, why := "", "", ""
	var noteErr error
	if len(items) == 0 { // a note given bare, as today's judge's is read: "yes: …"
		note = string(data)
		said, why, noteErr = verdict(note, true)
	}
	for _, it := range items {
		switch {
		case it["note"] != nil && note == "":
			note = fmt.Sprint(it["note"])
			said, why, noteErr = verdict(note, true)
		case it["read"] != nil:
			if r, ok := readOf(it["read"]); ok {
				a.Reads = append(a.Reads, r)
			}
		case it["quote"] != nil && a.Quote == nil:
			if m, ok := it["quote"].(map[string]any); ok {
				a.Quote = &Quote{Path: strings.TrimSpace(fmt.Sprint(m["path"])), Text: text(m["text"])}
			}
		}
	}
	switch {
	case said == "yes" || said == "no":
		a.Yes, a.Why, a.Reads = said == "yes", why, nil
	case len(a.Reads) > 0:
		a.Why, a.Quote = why, nil
	case said == "unsure":
		a.Unsure, a.Why = true, why
	case note == "":
		return a, fmt.Errorf("the judge answered no note and asked no read: %.200s", data)
	default:
		return a, noteErr
	}
	return a, nil
}

// itemsOf reads an answer that is no YAML as a whole item by item, each
// starting with "- ": a note left unquoted ("- note: no: …") is kept as
// its text, so a broken item loses neither the reads nor the quote beside.
func itemsOf(raw string) []map[string]any {
	var chunks []string
	for _, l := range strings.SplitAfter(raw, "\n") {
		if strings.HasPrefix(l, "- ") || len(chunks) == 0 {
			chunks = append(chunks, l)
		} else {
			chunks[len(chunks)-1] += l
		}
	}
	var items []map[string]any
	for _, c := range chunks {
		var one []map[string]any
		switch {
		case yaml.Unmarshal([]byte(c), &one) == nil && len(one) == 1:
			items = append(items, one[0])
		case strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(c, "- ")), "note:"):
			items = append(items, map[string]any{"note": strings.TrimSpace(c)})
		}
	}
	return items
}

// readOf is one read a judge asked, from its YAML: false when it names
// nothing to read.
func readOf(v any) (Read, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return Read{}, false
	}
	r := Read{Path: text(m["path"]), Lines: strings.ReplaceAll(text(m["lines"]), " ", ""), Name: text(m["name"]), Grep: text(m["grep"]), Why: text(m["why"])}
	return r, r.Path != "" || r.Name != "" || r.Grep != ""
}

func text(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// Verdict reads the judge's note: "yes: why" or "no: why". The note may come
// raw, with its YAML around it.
func Verdict(note string) (yes bool, why string, err error) {
	said, why, err := verdict(note, false)
	return said == "yes", why, err
}

// verdict reads a note: yes, no, or, for a judge that may read, unsure.
func verdict(note string, reading bool) (said, why string, err error) {
	s := strings.TrimSpace(note)
	s = strings.TrimSpace(strings.TrimPrefix(s, "-"))
	s = strings.TrimSpace(strings.TrimPrefix(s, "note:")) // a note, as a list's item or not
	if len(s) > 1 && (s[0] == '"' || s[0] == '\'') {
		s = strings.TrimSuffix(s[1:], s[:1]) // the note quoted whole
	}
	v, reason, _ := strings.Cut(s, ":")
	reason = strings.Join(strings.Fields(reason), " ")
	switch v := strings.ToLower(strings.TrimSpace(v)); {
	case v == "yes", v == "no", v == "unsure" && reading:
		return v, reason, nil
	}
	if reading {
		return "", "", fmt.Errorf("the judge said neither yes, no nor unsure: %.200s", strings.Join(strings.Fields(note), " "))
	}
	return "", "", fmt.Errorf("the judge said neither yes nor no: %.200s", strings.Join(strings.Fields(note), " "))
}
