package line

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/release"
	"github.com/JN0V/workline/internal/verdict"
)

// DryRun is a release's dry run (`route release --dry-run`, ADR-0050):
// what would hold the release were it cut now from HEAD, the default
// branch's tip at night. Each is said, none blocks.
type DryRun struct {
	Head  string `json:"head"`
	Since string `json:"since,omitempty"` // the last release it compares with; "" before the first
	// At: HEAD is this release already, nothing since to cut; no step run.
	At    string `json:"at,omitempty"`
	Holds []Hold `json:"holds"`
}

// Hold is one thing that would hold the release: a step's finding that
// blocks, or a step that does not pass with none, a person's decision
// or an outside failure.
type Hold struct {
	Step    string `json:"step"`
	Rule    string `json:"rule,omitempty"`
	Where   string `json:"where,omitempty"`
	Message string `json:"message,omitempty"`
}

// dryRunOn starts a release's dry run on the branch's tip, no agent,
// nothing applied. A tip that is a release already leaves nothing to cut
// (At).
func dryRunOn(o engine.Options) (engine.Options, *DryRun) {
	o.AI, o.NoApply = "none", true
	d := &DryRun{Holds: []Hold{}}
	if out, err := exec.Command("git", "-C", o.Repo, "rev-parse", "HEAD").Output(); err == nil {
		d.Head = strings.TrimSpace(string(out))
	}
	rs, err := release.Load(o.Repo)
	if err != nil {
		return o, d // the line's own config error says it
	}
	if at, err := release.AtHead(o.Repo, rs.Tags); err == nil && len(at) > 0 {
		d.At = at[0]
	}
	d.Since, _ = release.Last(o.Repo, rs.Tags)
	return o, d
}

// finishDryRun turns the line's run into the dry run's: what blocks is
// listed as what would hold the release, then lowered to a warning; the
// line passes, nothing is left to apply. A line that could not run holds
// the release too, said the same way.
func finishDryRun(res *Result, d *DryRun) {
	blocks := func(f verdict.Finding) bool { return f.Level == "" || f.Level == verdict.Block }
	for _, s := range res.Steps {
		if s.Status == verdict.Pass {
			continue
		}
		var fs []verdict.Finding
		said := ""
		switch {
		case s.Gate != nil:
			fs, said = s.Gate.Findings, s.Gate.Summary
		case s.Result != nil:
			fs, said = s.Result.Findings, s.Result.Summary
		}
		// What the step says blocks; else what it leaves to its verdict,
		// no level given; else the step, by its verdict.
		n := len(d.Holds)
		for _, level := range []string{verdict.Block, ""} {
			for _, f := range fs {
				if f.Level == level {
					d.Holds = append(d.Holds, Hold{Step: s.Name, Rule: f.Rule, Where: f.Where, Message: f.Message})
				}
			}
			if len(d.Holds) > n {
				break
			}
		}
		if len(d.Holds) == n {
			d.Holds = append(d.Holds, Hold{Step: s.Name, Rule: s.Status, Message: said})
		}
	}
	if len(res.Steps) == 0 && res.Status != verdict.Pass {
		for _, f := range res.Findings {
			if blocks(f) {
				d.Holds = append(d.Holds, Hold{Step: "routing", Rule: f.Rule, Where: f.Where, Message: f.Message})
			}
		}
	}
	lower := func(fs []verdict.Finding) {
		for i := range fs {
			if blocks(fs[i]) {
				fs[i].Level = "warn"
			}
		}
	}
	lower(res.Findings)
	for _, s := range res.Steps {
		if s.Gate != nil {
			lower(s.Gate.Findings)
		}
		if s.Result != nil {
			lower(s.Result.Findings)
			s.Result.ToApply, s.Result.Pending = false, nil
		}
	}
	res.Status, res.Pending, res.DryRun = verdict.Pass, nil, d
	res.Summary = DryRunSaid(d)
}

// DryRunSaid is a dry run said in one line: what would hold the release
// were it cut now, counted by step, or nothing.
func DryRunSaid(d *DryRun) string {
	from := "from " + short(d.Head)
	if d.Since != "" {
		from += ", since " + d.Since
	}
	switch {
	case d.At != "":
		return fmt.Sprintf("release, a dry run: %s is the release %s already, nothing since to cut", short(d.Head), d.At)
	case len(d.Holds) == 0:
		return fmt.Sprintf("release, a dry run %s: nothing would hold the release if it were cut now", from)
	}
	var by []string
	n := map[string]int{}
	for _, h := range d.Holds {
		if n[h.Step] == 0 {
			by = append(by, h.Step)
		}
		n[h.Step]++
	}
	for i, s := range by {
		by[i] = fmt.Sprintf("%s %d", s, n[s])
	}
	return fmt.Sprintf("release, a dry run %s: %d would hold the release if it were cut now (%s); said, never a block", from, len(d.Holds), strings.Join(by, ", "))
}

// short is a commit's first seven characters.
func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
