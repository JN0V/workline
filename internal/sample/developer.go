package sample

import (
	"fmt"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// The developer's code in the weekly sample (ADR-0046, the developer's
// row), on each merge request of its merged in the window: its commits on
// the head, `Workline-Role: developer`, and the record on its pull request.
// Not drawn: its pick and its refusals at apply, the engine's own moves.

const devRole = "developer"

// devAct is what the developer's code is re-checked against, read with it.
type devAct struct {
	record   developer.Act
	recorded bool
	first    string // its oldest commit on the head
}

// developerActs are the developer's code on its merge requests merged in
// the window, one act each, and what could not be read of them.
func developerActs(repo string, f forge.Forge, mrs []forge.MergeRequest) ([]Act, []string, error) {
	var out []Act
	var unread []string
	for _, m := range mrs {
		if !strings.HasPrefix(m.Branch, developer.Branch) {
			continue
		}
		ref := forge.MergeRef(f, m.ID)
		list, found, err := mergedCommits(repo, m)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			unread = append(unread, fmt.Sprintf("the developer's code on %s: its head is not in this clone", ref))
			continue
		}
		var tip, first, subject, by, day string
		for _, c := range strings.Fields(list) { // newest first
			info, err := git(repo, "log", "-1", "--format=%(trailers:key=Workline-Role,valueonly,separator=%x2C)%x00%(trailers:key=Workline-Model,valueonly,separator=%x2C%x20)%x00%aI%x00%s", c)
			if err != nil {
				return nil, nil, err
			}
			p := strings.Split(info, "\x00")
			if len(p) != 4 || !hasValue(p[0], devRole) {
				continue
			}
			first = c
			if tip == "" { // its code's commit, its tests' before it
				tip, by, subject = c, p[1], p[3]
				if t, err := time.Parse(time.RFC3339, p[2]); err == nil {
					day = t.UTC().Format(time.DateOnly)
				}
			}
		}
		if tip == "" {
			unread = append(unread, fmt.Sprintf("the developer's code on %s: no commit of its on the head", ref))
			continue
		}
		r, ok, err := developer.ReadRecord(f, m.ID)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, Act{Role: devRole, Kind: "code", Ref: ref, Commit: tip, Day: day, By: by, Line: subject, key: tip,
			developer: &devAct{record: r, recorded: ok, first: first}})
	}
	return out, unread, nil
}

// developerMaterial gathers what the developer's code was written for: the
// issue, the diff of its own commits, its proof.
func (rd *reader) developerMaterial(a *Act, m *material) string {
	d := a.developer
	if d == nil {
		return "its record was not read with it"
	}
	if !d.recorded {
		return fmt.Sprintf("the developer's record of its code on %s cannot be read", a.Ref)
	}
	m.author, m.acted = actedBy(a.By)
	is, err := rd.o.Forge.Issue(d.record.Issue)
	if err != nil {
		return fmt.Sprintf("#%d cannot be read: %v", d.record.Issue, err)
	}
	diff, err := git(rd.o.Repo, "diff", "--no-color", d.first+"^", a.Commit)
	if err != nil {
		return fmt.Sprintf("the diff of %.7s cannot be read", a.Commit)
	}
	diff = cutDiff(diff)
	m.items = append(m.items, issueItem(is), item{name: "diff", text: diff, full: diff})
	if d.record.Proof != nil {
		proof := developer.ProofLines(*d.record.Proof, d.record.Caps)
		m.items = append(m.items, item{name: "proof", text: proof, full: proof})
	}
	m.act = fmt.Sprintf("The developer's code for #%d, its commits up to %.7s, opened as %s.", d.record.Issue, a.Commit, a.Ref)
	m.about = "The issue as it reads now, the diff of the developer's own commits (its code, its tests), and what the project's tests said on it."
	// Its review rounds (ADR-0054, step 6): what each was asked and how it
	// ended, its fixes among the commits drawn.
	if rounds := developer.RoundLines(d.record); rounds != "" {
		m.items = append(m.items, item{name: "rounds", text: rounds, full: rounds})
		m.act = fmt.Sprintf("The developer's code for #%d, its commits up to %.7s, opened as %s, then %s.", d.record.Issue, a.Commit, a.Ref, plural(len(d.record.Rounds), "review round"))
		m.about += " Then its review rounds: what the reviewer or a person asked each, and how it ended."
	}
	return ""
}
