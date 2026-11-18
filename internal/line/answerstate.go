package line

import (
	"fmt"
	"github.com/JN0V/workline/internal/kit/testmark"
	"os/exec"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// answerStateMarker keys the line's word on a tester's Draft whose answer
// run ran (#1701): the Draft as that run read it, its tip and each
// objection waiting. On the same Draft, the reconciler starts no answer
// again, nor does a chained run: only a new comment in a waiting thread,
// a new objection or a new head does.
var answerStateMarker = forge.Marker("sticky=line/answer-state")

// AnswerState is a tester's Draft as its answer run reads it: its
// branch's tip in repo and each objection waiting, by thread and its
// comments' count, in a line; waiting, whether one waits at all; tried,
// whether an answer run already ran on that same state.
func AnswerState(repo string, f forge.Forge, id int) (state string, waiting, tried bool, err error) {
	th, ok := f.(forge.Threaded)
	if !ok {
		return "", false, false, fmt.Errorf("this forge lists no conversation on a line")
	}
	mr, err := f.MergeRequest(id)
	if err != nil {
		return "", false, false, err
	}
	threads, err := th.Threads(id)
	if err != nil {
		return "", false, false, err
	}
	return answerState(repo, f, mr, threads)
}

func answerState(repo string, f forge.Forge, mr forge.MergeRequest, threads []forge.Thread) (state string, waiting, tried bool, err error) {
	var ids []string
	for _, t := range threads {
		if testmark.ObjectionWaits(t) {
			ids = append(ids, fmt.Sprintf("%s.%d", t.ID, len(t.Comments)))
		}
	}
	if len(ids) == 0 {
		return "", false, false, nil
	}
	slices.Sort(ids)
	tip := remoteTip(repo, mr.Branch)
	if tip == "" {
		tip = "-"
	}
	state = "at=" + tip + " threads=" + strings.Join(ids, ",")
	b, ok := f.(forge.Backlog)
	if !ok {
		return state, true, false, nil
	}
	comments, err := b.Comments(forge.Target{Kind: "merge-request", ID: mr.ID})
	if err != nil {
		return state, true, false, err
	}
	said := "<!-- workline:answer-state " + state + " -->"
	tried = slices.ContainsFunc(comments, func(c string) bool { return strings.Contains(c, said) })
	return state, true, tried, nil
}

// MarkAnswerTried writes on the tester's Draft id the state its answer
// run ran on, whatever it wrote: answered, refused, failed.
func MarkAnswerTried(f forge.Forge, id int, state string) error {
	at, _, _ := strings.Cut(strings.TrimPrefix(state, "at="), " ")
	body := fmt.Sprintf("**workline's line** ran the tester's answer on the objections here, at %s. "+
		"It runs it again on a new comment in their threads, a new objection or a new head; never on the same ones.\n\n"+
		"<!-- workline:answer-state %s -->\n%s", short(at), state, answerStateMarker)
	return f.Sticky(forge.Target{Kind: "merge-request", ID: id}, body, answerStateMarker, true)
}

// remoteTip is branch's commit as this clone last fetched it from its
// remote, else its local branch; "" when neither is there.
func remoteTip(repo, branch string) string {
	for _, ref := range []string{"refs/remotes/origin/" + branch, "refs/heads/" + branch} {
		out, err := exec.Command("git", "-C", repo, "rev-parse", "-q", "--verify", ref+"^{commit}").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}
