package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
)

// ActsFile is where the acts of the roles that run on the shared pre and
// post are kept, in the runs folder (RunsDir), one JSON line each: what the
// weekly sample draws from, by role and kind, with no code of the role's own
// (ADR-0070).
const ActsFile = "acts.jsonl"

// ActRecord is one act a role did, applied: its role, its kind (the
// intention's), what it was on, and what it said or wrote.
type ActRecord struct {
	Run    string `json:"run"`
	Index  int    `json:"index"`
	Role   string `json:"role"`
	Kind   string `json:"kind"`
	Day    string `json:"day"`              // YYYY-MM-DD, UTC
	Ref    string `json:"ref"`              // the issue or merge request it was on, else the files it wrote
	Target string `json:"target,omitempty"` // issue or merge-request, with Number
	Number int    `json:"number,omitempty"`
	By     string `json:"by,omitempty"`   // the agents and models that proposed it
	Line   string `json:"line,omitempty"` // what it said, or the diff it wrote, cut
}

// maxActText bounds the text an act keeps.
const maxActText = 8000

// keepAct appends an applied intention to the runs folder's record.
func keepAct(repo, runID, role string, st runState, index int, in intent.Intention, f forge.Forge) error {
	if in.Kind == "note" {
		return nil // a message for a person: no act on the project
	}
	rec := ActRecord{Run: runID, Index: index, Role: role, Kind: in.Kind, Day: time.Now().UTC().Format(time.DateOnly),
		By: strings.Join(st.Models, ", ")}
	if st.Target != nil {
		rec.Target, rec.Number = st.Target.Kind, st.Target.ID
		rec.Ref = fmt.Sprintf("#%d", st.Target.ID)
		if st.Target.Kind == "merge-request" {
			rec.Ref = forge.MergeRef(f, st.Target.ID)
		}
	}
	switch v := in.Value.(type) {
	case string:
		rec.Line = v
	default:
		if data, err := json.Marshal(v); err == nil {
			rec.Line = string(data)
		}
	}
	if in.Kind == "patch" {
		rec.Line = fmt.Sprint(in.Value)
		if rec.Ref == "" {
			rec.Ref = strings.Join(intent.PatchFiles(in.Value), ", ")
		}
	}
	if rec.Ref == "" {
		rec.Ref = runID
	}
	if len(rec.Line) > maxActText {
		rec.Line = rec.Line[:maxActText]
	}
	base := RunsDir(repo)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(base, ActsFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	if cerr := file.Close(); err == nil {
		err = cerr
	}
	return err
}

// ReadActs are the acts kept in a runs folder, oldest first; none when
// nothing was kept.
func ReadActs(base string) ([]ActRecord, error) {
	f, err := os.Open(filepath.Join(base, ActsFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []ActRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var a ActRecord
		if json.Unmarshal(sc.Bytes(), &a) == nil && a.Role != "" && a.Kind != "" {
			out = append(out, a)
		}
	}
	return out, sc.Err()
}
