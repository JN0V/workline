package engine

// A run folder is dropped 50 runs later (keepRuns); what its agent calls
// spent is kept first, one line in the runs folder's tally, so a line's
// tokens can be summed over a week however many runs it made (#1360).

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/forge"
)

// TallyFile is the file in a runs folder holding a RunTally a line for
// each run dropped that had called an agent; never pruned.
const TallyFile = "tally.jsonl"

// RunTally is what a run's agent calls spent, and on what.
type RunTally struct {
	Run          string        `json:"run"` // its folder's name: when it started, and its role
	Role         string        `json:"role,omitempty"`
	Target       *forge.Target `json:"target,omitempty"` // as its out/run.yaml says; none before apply
	Calls        int           `json:"calls"`
	Fresh        int           `json:"fresh"` // agent.Call.Fresh, summed
	TokensIn     int           `json:"tokens-in,omitempty"`
	TokensCached int           `json:"tokens-cached,omitempty"`
	TokensOut    int           `json:"tokens-out,omitempty"`
	Seconds      float64       `json:"seconds,omitempty"`
}

// Started is when the run started, from its folder's name; zero when the
// name does not say.
func (t RunTally) Started() time.Time {
	stamp, _, _ := strings.Cut(t.Run, "-")
	at, err := time.Parse("20060102T150405.000000000", stamp)
	if err != nil {
		return time.Time{}
	}
	return at
}

// tallyOf reads a run folder: its out/run.yaml and out/calls.jsonl; an
// error when its calls are there but could not be read whole.
func tallyOf(dir string) (RunTally, error) {
	t := RunTally{Run: filepath.Base(dir)}
	if _, role, ok := strings.Cut(t.Run, "-"); ok {
		t.Role = role
	}
	if data, err := os.ReadFile(filepath.Join(dir, "out", "run.yaml")); err == nil {
		var run struct {
			Role   string        `yaml:"role"`
			Target *forge.Target `yaml:"target"`
		}
		if yaml.Unmarshal(data, &run) == nil {
			if run.Role != "" {
				t.Role = run.Role
			}
			t.Target = run.Target
		}
	}
	file, err := os.Open(filepath.Join(dir, "out", "calls.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return t, err
	}
	defer func() { _ = file.Close() }()
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	unread := 0
	for sc.Scan() {
		var c agent.Call
		if len(strings.TrimSpace(sc.Text())) == 0 {
			continue
		}
		if json.Unmarshal(sc.Bytes(), &c) != nil {
			unread++
			continue
		}
		t.Calls++
		t.Fresh += c.Fresh()
		t.TokensIn += c.TokensIn
		t.TokensCached += c.TokensCached
		t.TokensOut += c.TokensOut
		t.Seconds += c.Seconds
	}
	if err := sc.Err(); err != nil {
		return t, err
	}
	if unread > 0 {
		return t, fmt.Errorf("%s: %d calls unread", filepath.Join(dir, "out", "calls.jsonl"), unread)
	}
	return t, nil
}

// keepTally adds the run in dir to its runs folder's tally, before the
// folder is dropped; a run that called no agent spent nothing, and is
// left out. An error: the folder is kept.
func keepTally(base, dir string) error {
	t, err := tallyOf(dir)
	if err != nil || t.Calls == 0 {
		return err
	}
	line, err := json.Marshal(t)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(base, TallyFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return errors.Join(err, f.Close())
}

// Tallies are the runs a runs folder knows of: those its tally kept, then
// those whose folder is still there, each once, by its folder's name. A
// folder that is not there: none.
func Tallies(base string) ([]RunTally, error) {
	var out []RunTally
	seen := map[string]bool{}
	if f, err := os.Open(filepath.Join(base, TallyFile)); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			var t RunTally
			if json.Unmarshal(sc.Bytes(), &t) == nil && t.Run != "" && !seen[t.Run] {
				seen[t.Run] = true
				out = append(out, t)
			}
		}
		err := sc.Err()
		_ = f.Close()
		if err != nil {
			return out, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	for _, e := range entries {
		if e.IsDir() && !seen[e.Name()] {
			seen[e.Name()] = true
			t, _ := tallyOf(filepath.Join(base, e.Name())) // a folder kept: what could be read of it
			out = append(out, t)
		}
	}
	return out, nil
}

// WriteTally writes to file the tally of the runs whose folder base still
// holds that called an agent, a line each, as TallyFile holds them: what
// a CI job keeps once its runs folder is gone (ADR-0019), no prompt nor
// diff. The lines of base's own tally are not written again: in the
// weekly sample's job, they are the tallies it gathered. None called an
// agent: no file, and 0.
func WriteTally(base, file string) (int, error) {
	entries, err := os.ReadDir(base)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	var out []byte
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, _ := tallyOf(filepath.Join(base, e.Name())) // what could be read of it, as Tallies
		if t.Calls == 0 {
			continue
		}
		line, err := json.Marshal(t)
		if err != nil {
			return 0, err
		}
		out = append(append(out, line...), '\n')
		n++
	}
	if n == 0 {
		return 0, nil
	}
	return n, os.WriteFile(file, out, 0o644)
}
