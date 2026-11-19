package engine

// Every act a role applies carries one shared record (ADR-0070, §3): a
// hidden line in the comment the apply step writes, the same for every
// role, so the weekly sample draws a role's acts with no code written for
// that role. Only workline's own account's comments are read back.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
)

// ActMark is the key of the record's hidden line.
const ActMark = "act"

// ActItem is an item the run read, with its state.
type ActItem struct {
	Kind  string `json:"kind"` // issue, merge-request or comment
	ID    string `json:"id,omitempty"`
	State string `json:"state"`
}

// Act is the record of one act a role applied.
type Act struct {
	Role   string    `json:"role"`
	Kind   string    `json:"kind"`
	Run    string    `json:"run"`
	Day    string    `json:"day"` // the UTC date at apply
	Commit string    `json:"commit"`
	Models []string  `json:"models"`
	Items  []ActItem `json:"items"`
}

// Line is the hidden line that holds the record.
func (a Act) Line() string {
	data, _ := json.Marshal(a)
	return forge.Marker(ActMark + " " + string(data))
}

// ReadActs are the acts recorded in the comments of t, read only from
// workline's own account's, a comment last edited by another ignored.
func ReadActs(k forge.Kept, t forge.Target) []Act {
	_ = t
	var out []Act
	for _, c := range k.Each("<!-- workline:" + ActMark + " ") {
		for _, l := range strings.Split(c, "\n") {
			_, after, ok := strings.Cut(l, "<!-- workline:"+ActMark+" ")
			if !ok {
				continue
			}
			data, _, _ := strings.Cut(after, " -->")
			var a Act
			if json.Unmarshal([]byte(data), &a) == nil {
				out = append(out, a)
			}
		}
	}
	return out
}

// recordLine matches a line in the record's form.
var recordLine = regexp.MustCompile(`<!--\s*workline:` + ActMark + `\s*\{`)

// forgedRecord is the first line of a proposal in the record's form: only
// the apply step writes one.
func forgedRecord(in []intent.Intention) string {
	var find func(v any) string
	find = func(v any) string {
		switch x := v.(type) {
		case string:
			for _, l := range strings.Split(x, "\n") {
				if recordLine.MatchString(l) {
					return strings.TrimSpace(l)
				}
			}
		case map[string]any:
			for _, e := range x {
				if l := find(e); l != "" {
					return l
				}
			}
		case []any:
			for _, e := range x {
				if l := find(e); l != "" {
					return l
				}
			}
		}
		return ""
	}
	for _, i := range in {
		if l := find(i.Value); l != "" {
			return l
		}
	}
	return ""
}

// modelsUsed are the distinct models the run's calls used, each once.
func modelsUsed(calls []agent.Call) []string {
	out := []string{}
	for _, c := range calls {
		if c.Model == "" {
			continue
		}
		dup := false
		for _, m := range out {
			dup = dup || m == c.Model
		}
		if !dup {
			out = append(out, c.Model)
		}
	}
	return out
}

// record is the hidden line of the act of kind, applied on target now.
func (a *applier) record(kind string, target forge.Target) string {
	commit, _ := git(a.repo, nil, "rev-parse", "HEAD")
	act := Act{Role: a.role, Kind: kind, Run: a.runID, Day: clock.Now().UTC().Format("2006-01-02"),
		Commit: commit, Models: a.models, Items: []ActItem{}}
	if act.Models == nil {
		act.Models = []string{}
	}
	act.Items = a.itemsRead(target)
	return act.Line()
}

// itemsRead are the issue or merge request and the comments on it, with
// their states, as the forge tells them.
func (a *applier) itemsRead(t forge.Target) []ActItem {
	items := []ActItem{}
	if t.Kind == "issue" {
		state := "open"
		if is, err := a.forge.Issue(t.ID); err == nil && is != nil && is.Closed {
			state = "closed"
		}
		items = append(items, ActItem{Kind: "issue", ID: fmt.Sprint(t.ID), State: state})
	} else {
		items = append(items, ActItem{Kind: "merge-request", ID: fmt.Sprint(t.ID), State: "open"})
	}
	if b, ok := a.forge.(forge.Backlog); ok {
		if notes, err := b.Notes(t); err == nil || len(notes) > 0 {
			for i, n := range notes {
				id := n.ID
				if id == "" {
					id = fmt.Sprint(i + 1)
				}
				items = append(items, ActItem{Kind: "comment", ID: id, State: "posted"})
			}
		}
	}
	return items
}

// recorded is body followed by the act's record.
func (a *applier) recorded(body, kind string, target forge.Target) string {
	return strings.TrimRight(backlog.Inert(body), "\n") + "\n\n" + a.record(kind, target)
}

// recordBacklogActs sets the record of each act on an issue this run
// applied from the backlog's plan, which Persist writes in the state the
// role keeps there, its one comment: a comment of its own for each act
// would crowd the issue.
func (a *applier) recordBacklogActs(p *backlog.Plan) {
	if a.forge == nil || p == nil {
		return
	}
	for _, d := range p.Decisions {
		if d.Mode != backlog.Act || d.Act.Issue <= 0 || d.Act.Do == "" {
			continue
		}
		if p.Records == nil {
			p.Records = map[int][]string{}
		}
		p.Records[d.Act.Issue] = append(p.Records[d.Act.Issue], a.record(d.Act.Do, forge.Target{Kind: "issue", ID: d.Act.Issue}))
	}
}
