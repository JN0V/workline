package productowner

// The kind's light read (#878): an open issue no kind label tells, that
// the run does not read in full, is read for its kind alone — its title
// and the start of its body, a few a run, many in one agent call. Without
// it, an issue read before the kind act existed keeps no kind until it
// changes, and a bug's fix waits for a person (ADR-0060, decision 4).

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

const (
	// kindTask names, in in/task-kind, a run that reads kinds alone: the
	// agent reads its own short instruction (model.tasks.kind).
	kindTask = "kind"
	// kindBodyMax is the characters of a body a light read gives.
	kindBodyMax = 600
)

// unkinded says whether an issue is one a light read may label: no kind
// label tells it, and the role never set one — a person's change or
// removal stands (ADR-0039 (c)).
func unkinded(is forge.Issue, st *backlog.State) bool {
	k, _ := backlog.KindOf(is.Labels)
	return k == "" && (st == nil || st.Kind == "")
}

// kindMode is the kind act's mode and cap in this run, the kinds a person
// undid proposed.
func kindMode(cfg backlog.Config, demoted []string) (mode string, max int) {
	for _, m := range cfg.Modes(demoted) {
		if m.Kind == kindTask {
			return m.Mode, m.Max
		}
	}
	return backlog.Off, 0
}

// lightKinds are the issues read for their kind alone, those nearest a
// merge first, then in the backlog's order, as many as the kind act's cap leaves once the issues read in full
// with no kind are counted; none when the act is off, or proposes while
// the issues waiting on a person are at proposals-max.
func lightKinds(candidates []forge.Issue, full []due, mode string, max int, waiting bool) []forge.Issue {
	if mode == backlog.Off || mode == backlog.Propose && waiting {
		return nil
	}
	read := map[int]bool{}
	for _, d := range full {
		read[d.is.ID] = true
		if unkinded(d.is, d.st) {
			max-- // the full read proposes its kind: the same cap
		}
	}
	list := slices.DeleteFunc(slices.Clone(candidates), func(is forge.Issue) bool { return read[is.ID] })
	if max <= 0 || len(list) == 0 {
		return nil
	}
	backlog.Order(list)
	// A merge waits on the kind (ADR-0060, decision 4): the issues a
	// developer is on first, then those ready, then the rest.
	slices.SortStableFunc(list, func(a, b forge.Issue) int { return nearMerge(b) - nearMerge(a) })
	return list[:min(max, len(list))]
}

// claimLabel is the developer's claim, its default (developer's
// claim-label): a pull request may wait on the issue's kind.
const claimLabel = "workline:developing"

// nearMerge says how near an issue is to a merge that waits on its kind:
// 2 a developer is on it, 1 ready, 0 otherwise.
func nearMerge(is forge.Issue) int {
	switch {
	case slices.Contains(is.Labels, claimLabel):
		return 2
	case slices.Contains(is.Labels, backlog.LabelReady):
		return 1
	}
	return 0
}

var htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)

// kindSection gives the issues read for their kind alone: each title and
// the start of its body, its HTML comments left out.
func kindSection(list []forge.Issue) string {
	if len(list) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Issues read for their kind alone\n\n")
	b.WriteString("No kind label tells these, and they are not read in full in this run: by their title and the start of their body, propose `kind` on each — bug, feature or task — and nothing else on them.\n\n")
	for _, is := range list {
		body := strings.TrimSpace(htmlComment.ReplaceAllString(is.Body, ""))
		if body == "" {
			body = "(no body)"
		}
		fmt.Fprintf(&b, "## #%d %s\n\n%s\n\n", is.ID, is.Title, clip(body, kindBodyMax))
	}
	return b.String()
}

// kindIntro opens a task that reads kinds alone.
func kindIntro(commit string, n int) string {
	return fmt.Sprintf("The run is on commit %s. Open issues to label with their kind: %d.\n\n", commit, n)
}
