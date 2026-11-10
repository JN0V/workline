package backlog

// A bug's parent (ADR-0029, amended 2026-11-08; #1212): the tester writes
// the reporter's symptom as a red test on the parent's Draft before it is
// split, and, once every part is closed, runs those tests over the
// default branch's current commit through its green proof. What it found
// is kept in one comment on the parent, workline's own; the engine closes
// the parent alone on it — green, every part closed when it ran, and a
// level past cautious — and proposes the closing to a person otherwise,
// the reason said.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// SymptomKey is the tester's comment on a bug's parent:
// <!-- workline:sticky=tester/symptom -->.
const SymptomKey = "symptom"

// SymptomInput is the input the line gives the tester on a bug's parent
// whose parts are all closed: the parts, by number, its symptom proven
// on the default branch with them in.
const SymptomInput = "symptom"

// Symptom is the tester's proof of a bug's parent: its Draft's tests,
// laid over the default branch's commit, green three runs out of three
// or not, and why.
type Symptom struct {
	Issue int `json:"issue"`
	// Draft: the tester's Draft the tests come from, 0 when none was found.
	Draft int `json:"draft,omitempty"`
	// Branch and Commit: the default branch and its commit the tests ran
	// over.
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit,omitempty"`
	// Parts: the parts closed when it ran: a part closed since is not in
	// the code it ran on.
	Parts []int `json:"parts"`
	// Tests: the ids its Draft's record keeps.
	Tests []string `json:"tests,omitempty"`
	Green bool     `json:"green"`
	// Why: what it found, said to a person: the tests green, or why not.
	Why string `json:"why"`
}

var symptomData = regexp.MustCompile(`<!-- workline:symptom (\{.*\}) -->`)

// Body is the comment a person reads, the record hidden at its end.
func (s Symptom) Body() string {
	var b strings.Builder
	if s.Green {
		fmt.Fprintf(&b, "**The reporter's symptom is gone**: %s.\n", strings.TrimSuffix(s.Why, "."))
	} else {
		fmt.Fprintf(&b, "**The reporter's symptom is not proven gone**: %s.\n", strings.TrimSuffix(s.Why, "."))
	}
	data, _ := json.Marshal(s)
	fmt.Fprintf(&b, "\n<!-- workline:symptom %s -->", data)
	return b.String()
}

// Said is what a closing says of it: the tests and the commit.
func (s Symptom) Said() string {
	if !s.Green {
		return s.Why
	}
	return fmt.Sprintf("its reproduction test%s %s green on %s at %s, three runs out of three", map[bool]string{true: "", false: "s"}[len(s.Tests) == 1], codeList(s.Tests), s.Branch, shortSHA(s.Commit))
}

// ReadSymptom reads the tester's proof on issue id from its comments,
// workline's own and edited by no one else (forge.Kept): found false when
// it has none.
func ReadSymptom(f forge.Forge, id int) (s Symptom, found bool, err error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return Symptom{}, false, err
	}
	body, ok := kept.Last(forge.Marker("sticky=tester/" + SymptomKey))
	if !ok {
		return Symptom{}, false, nil
	}
	m := symptomData.FindStringSubmatch(body)
	if m == nil || json.Unmarshal([]byte(m[1]), &s) != nil || s.Issue != id {
		return Symptom{}, false, nil
	}
	return s, true, nil
}

// Proves says why s does not prove the symptom of a parent whose parts
// are parts, "" when it does: green, run once all of them were closed.
func (s Symptom) Proves(parts []int) string {
	if !s.Green {
		return s.Why
	}
	var late []int
	for _, n := range parts {
		if !slices.Contains(s.Parts, n) {
			late = append(late, n)
		}
	}
	if len(late) > 0 {
		return fmt.Sprintf("its reproduction test ran before %s closed: not on the code that closed it", issueList(late))
	}
	return ""
}

func codeList(l []string) string {
	if len(l) == 0 {
		return "(none named)"
	}
	return "`" + strings.Join(l, "`, `") + "`"
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
