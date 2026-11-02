package developer

// A project's check red on the developer's pull request is a round, like
// a gate's finding (#917): its tests, run whole in CI, fail where the
// developer's proof ran a part of them. The round asks each check red at
// its branch's head once all the project's checks there are done, the
// end of its failed steps' log in its task. Its record keeps the head
// the checks were read at: the same head is never asked twice.

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// checkHolds is how a round says a project check's red.
const checkHolds = "a project check"

// The bounds of a check's log in a round's task: its last lines, at most
// so many bytes. The end of a failed step holds the failures and the
// error that stopped it; its whole log would fill the agent's context.
const (
	logLines = 80
	logBytes = 6000
)

// checksAsked reads the project's checks at head: none asked when head
// is not known, the forge reads none, its last round read them at this
// head (a.ChecksAt), one still runs, or none is red. It returns the head
// they were read at, and why the forge did not answer.
func checksAsked(f forge.Forge, a Act, head string) (at string, asked []Asked, unread string) {
	pc, ok := f.(forge.ProjectChecked)
	if !ok || head == "" || sameCommit(head, a.ChecksAt) {
		return "", nil, ""
	}
	cs, err := pc.ProjectChecks(head)
	if err != nil {
		return "", nil, err.Error()
	}
	for _, c := range cs {
		if c.State == "pending" {
			return "", nil, "" // the round reads them all once done: the last one's end wakes it
		}
	}
	for _, c := range cs {
		if c.State != "failure" {
			continue
		}
		asked = append(asked, Asked{Key: "checks|" + c.Name, Where: c.Name, Text: "red at `" + head[:min(7, len(head))] + "`",
			Holds: checkHolds, Check: &c})
	}
	if len(asked) == 0 {
		return "", nil, ""
	}
	return head, asked, ""
}

// checkLogs gives each red check the end of its failed steps' log, read
// from the forge; one it cannot read says so.
func checkLogs(f forge.Forge, asked []Asked) {
	pc, ok := f.(forge.ProjectChecked)
	if !ok {
		return
	}
	for i, a := range asked {
		if a.Holds != checkHolds || a.Check == nil {
			continue
		}
		log, err := pc.CheckLog(*a.Check)
		switch {
		case err != nil:
			asked[i].Detail = "its log could not be read (" + err.Error() + "): run the project's tests whole to see what fails"
		case strings.TrimSpace(log) == "":
			asked[i].Detail = "the forge keeps no log of it: run the project's tests whole to see what fails"
		default:
			asked[i].Detail = logEnd(log)
		}
	}
}

var (
	ansi  = regexp.MustCompile("\x1b\\[[0-9;]*[A-Za-z]|\\^\\[\\[[0-9;]*m")
	stamp = regexp.MustCompile(`^\x{feff}?\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?Z ?`)
)

// logEnd is a log's end, bounded: its last lines, no time stamps nor
// colours nor the runner's group marks, at most logBytes.
func logEnd(log string) string {
	var keep []string
	for _, l := range strings.Split(strings.ReplaceAll(log, "\r", ""), "\n") {
		l = ansi.ReplaceAllString(stamp.ReplaceAllString(l, ""), "")
		if strings.HasPrefix(l, "##[group]") || strings.HasPrefix(l, "##[endgroup]") {
			continue
		}
		keep = append(keep, strings.TrimRight(l, " \t"))
	}
	for len(keep) > 0 && strings.TrimSpace(keep[len(keep)-1]) == "" {
		keep = keep[:len(keep)-1]
	}
	cut := len(keep) > logLines
	if cut {
		keep = keep[len(keep)-logLines:]
	}
	out := strings.Join(keep, "\n")
	if len(out) > logBytes {
		i := len(out) - logBytes
		for i < len(out) && !utf8.RuneStart(out[i]) {
			i++ // never inside a character
		}
		out, cut = out[i:], true
		if i := strings.IndexByte(out, '\n'); i >= 0 && i < len(out)-1 {
			out = out[i+1:]
		}
	}
	if cut {
		out = "…\n" + out
	}
	return out
}

// checkLine is a red check as a round's record and comment say it.
func checkLine(a Asked) string {
	return fmt.Sprintf("%s `%s`: %s", checkHolds, a.Where, a.Text)
}

// checksUnread says the project's checks were not read: a red one asks
// nothing this round (principle 12).
func checksUnread(where, why string) verdict.Finding {
	return verdict.Finding{Rule: "checks-unread", Level: "warn", Where: where,
		Message: "the project's checks at its head were not read, a red one not asked: " + why}
}
