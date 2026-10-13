package forge

import (
	"fmt"
	"os"
	"strings"
)

// Push is a head a merge request's branch was pushed to, and when.
type Push struct {
	Head string `json:"head"`
	At   string `json:"at"` // RFC 3339
}

// MergeState is what the merge policy weighs of an open merge request
// (ADR-0042), as its forge says it.
type MergeState struct {
	Head  string `json:"head"` // its head commit
	Draft bool   `json:"draft,omitempty"`
	// AutoMerge: the forge's auto-merge is on now. AutoMergeAt: when it
	// was last turned on, on now or not; "" never, or the forge does not
	// say. AutoMergeBy, AutoMergeBot: who turned it on now, and whether a
	// bot — workline's token, an app.
	AutoMerge    bool   `json:"auto-merge,omitempty"`
	AutoMergeAt  string `json:"auto-merge-at,omitempty"`
	AutoMergeBy  string `json:"auto-merge-by,omitempty"`
	AutoMergeBot bool   `json:"auto-merge-bot,omitempty"`
	// Checks: its head's checks as the forge sums them up — success,
	// pending, failure; "" none, or unread (ChecksUnread says why).
	Checks       string `json:"checks,omitempty"`
	ChecksUnread string `json:"checks-unread,omitempty"`
	// Labels are its labels set and taken off, oldest first, with who did
	// each, as Backlog's LabelEvents.
	Labels []LabelEvent `json:"label-events,omitempty"`
	// Pushes are the heads its branch was pushed to, oldest first; nil when
	// the forge does not say.
	Pushes []Push `json:"pushes,omitempty"`
}

// Merger is a forge whose own auto-merge the merge policy turns on and
// off (ADR-0042): the forge merges, checking its own rules.
type Merger interface {
	// MergeState reads an open merge request's state.
	MergeState(mr MergeRequest) (MergeState, error)
	// AutoMerge turns the forge's auto-merge on for the merge request at
	// head, or off. merged: the forge took it at once, its rules met
	// already (GitHub turns no auto-merge on for a pull request it could
	// merge now). Turning off one already off changes nothing.
	AutoMerge(mr MergeRequest, head string, on bool) (merged bool, err error)
	// Guarded says why a push after a person's yes could merge before
	// `follow` takes the yes back; "" when the forge holds it: GitHub when
	// the base requires the status Accepted, which a new head lacks;
	// GitLab, whose merge waits for the head's pipeline, `follow` in it.
	Guarded(base string) (string, error)
	// Accept marks a head accepted, or not, where the forge needs it to
	// hold a yes to that head only (GitHub: the status Accepted, by this
	// token); nothing where it does not.
	Accept(head string, ok bool, why string) error
	// KeepVerdict keeps the line's verdict on a head — its class, pass,
	// held, short or red, and why — where only this token writes it
	// (GitHub: the status VerdictStatus); Verdict reads it back, "" when
	// none is kept, or the forge keeps none (GitLab: the verdict is in hand
	// in the pipeline that judged the head).
	KeepVerdict(head, class, why string) error
	Verdict(head string) (class, why string, err error)
}

// The statuses workline sets on a head on GitHub, by its App's token: the
// line's verdict, and the yes the merge policy holds (ADR-0042).
const (
	VerdictStatus = "workline/verdict"
	Accepted      = "workline/accepted"
)

// statusText is a status's description: GitHub keeps 140 characters.
func statusText(s string) string {
	if r := []rune(s); len(r) > 140 {
		return string(r[:139]) + "…"
	}
	return s
}

// verdictText is a verdict kept in a status's description, "class: why";
// readVerdict reads it back.
func verdictText(class, why string) string { return statusText(class + ": " + why) }

func readVerdict(text string) (class, why string) {
	class, why, _ = strings.Cut(text, ": ")
	switch class {
	case "pass", "held", "short", "red":
		return class, why
	}
	return "", ""
}

// Refused says whether err is the forge refusing a request — a token that
// may not, a request it does not take — rather than not answering.
func Refused(err error) bool { return refusal(err) }

// Said is what the forge said refusing, without the words that only mark
// the error's kind.
func Said(err error) string {
	return strings.TrimPrefix(said(err), ErrUnreachable.Error()+": ")
}

// MergeRef is how f names a merge request: !7 on GitLab, #7 elsewhere.
func MergeRef(f any, id int) string {
	if _, ok := f.(*gitlab); ok {
		return fmt.Sprintf("!%d", id)
	}
	return fmt.Sprintf("#%d", id)
}

// MergeToken says why a merge f makes with its token would start no
// pipeline on the base — on GitHub Actions, the job's own token starts no
// workflow (ADR-0042) —, "" when it would.
func MergeToken(f any) string {
	if t, ok := f.(interface{ mergeToken() string }); ok {
		return t.mergeToken()
	}
	return ""
}

// mergeToken: in GitHub Actions, a token the workflow says is no GitHub
// App's (WORKLINE_APP, the app's slug, empty) is the job's own.
func (g *github) mergeToken() string {
	if os.Getenv("GITHUB_ACTIONS") == "true" && strings.TrimSpace(os.Getenv("WORKLINE_APP")) == "" {
		return "the job's own token (GITHUB_TOKEN): a merge it makes starts no workflow on the base — no follow, no release; give the job the GitHub App's token, and its slug in WORKLINE_APP"
	}
	return ""
}

// Self is who f's token writes as — on GitHub the App's bot, else the
// token's user —, "" when the forge does not say: a merge request it
// opened is workline's own, as one a bot opened (ADR-0042, `normal`).
func Self(f any) string {
	if s, ok := f.(interface{ self() (string, error) }); ok {
		if who, err := s.self(); err == nil {
			return who
		}
	}
	return ""
}

func (g *github) self() (string, error) { return g.me() }

// self: the simulated forge's token's user.
func (f *Fake) self() (string, error) { return fakeBot, nil }

// Every forge with an auto-merge workline speaks is a Merger; the local
// forge has none.
var _ = []Merger{&github{}, &gitlab{}, &command{}, &Fake{}}
