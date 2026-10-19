package forge

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Push is a head a merge request's branch was pushed to, and when.
type Push struct {
	Head string `json:"head"`
	At   string `json:"at"` // RFC 3339
}

// CheckRun is one run of a check on a head, as the forge shows it: its
// name, and its state — success, failure, pending, or cancelled.
type CheckRun struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// sumRuns sums up a head's check runs: failure when one failed, else
// pending when one runs, else success; "" none. A cancelled run is no
// result, neither success nor failure: another run of the same check —
// the same name, as the forge's required checks match them — counts
// instead; a check only ever cancelled counts as one still running, the
// forge's own required checks still guarding the merge.
func sumRuns(runs []CheckRun) string {
	other := map[string]bool{} // a check with a run not cancelled
	for _, r := range runs {
		if r.State != "cancelled" {
			other[r.Name] = true
		}
	}
	sum := ""
	for _, r := range runs {
		state := r.State
		if state == "cancelled" {
			if other[r.Name] {
				continue
			}
			state = "pending"
		}
		switch {
		case state == "failure":
			return "failure"
		case state == "pending" || sum == "pending":
			sum = "pending"
		case state == "success":
			sum = "success"
		}
	}
	return sum
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
	// Pushers are who pushed to its branch, each push, as the forge names
	// them — GitHub's repository activity, GitLab's push events —; nil
	// when the forge does not say (ADR-0042, `normal`).
	Pushers []string `json:"pushers,omitempty"`
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
	// held, short, waits (a draft's, ADR-0050) or red, and why — where
	// only this token writes it
	// (GitHub: the status VerdictStatus); Verdict reads it back, "" when
	// none is kept, or the forge keeps none (GitLab: the verdict is in hand
	// in the pipeline that judged the head).
	KeepVerdict(head, class, why string) error
	Verdict(head string) (class, why string, err error)
}

// QueueGuard is a forge whose merge queue tests a group's commit against
// the checks its base requires, the yes Accept sets among them (GitHub,
// ADR-0050). AcceptedOn reads that yes back on a head, this token's only:
// whether it holds one, and why not when Accept said.
type QueueGuard interface {
	AcceptedOn(head string) (ok bool, why string, err error)
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
	case "pass", "held", "short", "waits", "red":
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

// App is the bot workline's token writes as — GitHub's App
// (WORKLINE_APP), GitLab's project bot —, and the name its jobs commit
// with; "" when the token is a person's, or the forge does not say. Only
// what it opened, pushed and committed is workline's own work at
// `normal` (ADR-0042): a person's token is the one agents use too.
func App(f any) (login, committer string, err error) {
	if a, ok := f.(interface {
		app() (string, string, error)
	}); ok {
		return a.app()
	}
	return "", "", nil
}

// app: the App the workflow names (WORKLINE_APP), whose jobs commit as
// it; none with a person's token.
func (g *github) app() (string, string, error) {
	if app := strings.TrimSpace(os.Getenv("WORKLINE_APP")); app != "" {
		return app + "[bot]", app + "[bot]", nil
	}
	return "", "", nil
}

// app: the token's user when GitLab says it is a bot (a project or group
// access token's); the template's jobs commit as "workline"
// (ci/gitlab/workline.gitlab-ci.yml).
func (g *gitlab) app() (string, string, error) {
	out, err := g.api("user")
	if err != nil {
		return "", "", err
	}
	var u struct {
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	}
	if err := decode(out, &u); err != nil {
		return "", "", err
	}
	if !u.Bot && !botName.MatchString(u.Username) {
		return "", "", nil
	}
	return u.Username, "workline", nil
}

// app: the simulated forge's token's user, committing under its name.
func (f *Fake) app() (string, string, error) { return fakeBot, fakeBot, nil }

// closing: the issue's last `closed` event, its actor's write access read
// as LabelEvents reads it.
func (g *github) closing(id int) (LabelEvent, bool, error) {
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/events?per_page=100", id),
		"--jq", `.[] | select(.event == "closed") | {author: (.actor.login // ""), bot: (.actor.type == "Bot"), created: .created_at}`)
	if err != nil {
		return LabelEvent{}, false, err
	}
	closed, err := lines[LabelEvent](out)
	if err != nil || len(closed) == 0 {
		return LabelEvent{}, false, err
	}
	e := closed[len(closed)-1]
	if e.Author != "" && !e.Bot {
		if e.Writer, err = g.writer(e.Author); err != nil {
			return LabelEvent{}, false, err
		}
	}
	return e, true, nil
}

// closing: the issue's last state event closing it, by whom, a Developer
// or above writing.
func (g *gitlab) closing(id int) (LabelEvent, bool, error) {
	out, err := g.api("--paginate", path(Target{Kind: "issue", ID: id})+"/resource_state_events?per_page=100")
	if err != nil {
		return LabelEvent{}, false, err
	}
	found, err := pages[struct {
		State   string `json:"state"`
		Created string `json:"created_at"`
		User    *struct {
			Username string `json:"username"`
		} `json:"user"`
	}](out)
	if err != nil {
		return LabelEvent{}, false, err
	}
	var e LabelEvent
	ok := false
	for _, s := range found {
		if s.State == "closed" {
			e, ok = LabelEvent{Note: Note{Created: s.Created}}, true
			if s.User != nil {
				e.Author, e.Bot = s.User.Username, botName.MatchString(s.User.Username)
			}
		}
	}
	if ok && e.Author != "" {
		if e.Insider, err = g.insider(e.Author); err != nil {
			return LabelEvent{}, false, err
		}
		e.Writer = !e.Bot && g.members[e.Author] >= developer
	}
	return e, ok, nil
}

// closing: as the file says who closed it (`closing`).
func (f *Fake) closing(id int) (LabelEvent, bool, error) {
	s, err := f.load()
	if err != nil {
		return LabelEvent{}, false, err
	}
	it, err := s.item(Target{Kind: "issue", ID: id})
	if err != nil || !it.Closed || it.Closing == nil {
		return LabelEvent{}, false, err
	}
	return *it.Closing, true, nil
}

// ReadStatus is the status the merge policy's App sets on the base commit
// it read for reverts, at `normal` (ADR-0042): its mark, read back from
// the App only, as VerdictStatus.
const ReadStatus = "workline/merge-read"

// Mark keeps on commit, where only this token writes — a status of the
// App's —, what the merge policy read there: text, the demotion issues in
// force. A forge that keeps no such mark answers ErrNoMark.
func Mark(f any, commit, text string) error {
	if m, ok := f.(interface {
		mark(commit, text string) error
	}); ok {
		return m.mark(commit, text)
	}
	return ErrNoMark
}

// Marked reads the mark this token kept on commit: ok false when none.
func Marked(f any, commit string) (text string, ok bool, err error) {
	if m, can := f.(interface {
		marked(commit string) (string, bool, error)
	}); can {
		return m.marked(commit)
	}
	return "", false, ErrNoMark
}

// ErrNoMark: the forge keeps no mark only workline's App writes.
var ErrNoMark = fmt.Errorf("this forge keeps no mark only workline's App writes")

func (g *github) mark(commit, text string) error {
	return g.setStatus(commit, ReadStatus, "success", text)
}

func (g *github) marked(commit string) (string, bool, error) {
	state, text, err := g.ownStatus(commit, ReadStatus)
	return text, err == nil && state != "", err
}

func (f *Fake) mark(commit, text string) error {
	return f.setStatus(commit, ReadStatus, "success", text)
}

func (f *Fake) marked(commit string) (string, bool, error) {
	st, ok, err := f.ownStatus(commit, ReadStatus)
	return st.Description, ok, err
}

// mark: a commit status of the project bot's — on a commit with a
// pipeline, an external job of it.
func (g *gitlab) mark(commit, text string) error {
	_, err := g.api("-X", "POST", "projects/:id/statuses/"+commit, "-f", "state=success", "-f", "name="+ReadStatus, "-f", "description="+statusText(text))
	return err
}

// marked: the newest status of that name on commit set by the project's
// bot, the token's user; one anyone else set is never read.
func (g *gitlab) marked(commit string) (string, bool, error) {
	app, _, err := g.app()
	if err != nil || app == "" {
		return "", false, err
	}
	out, err := g.api("projects/:id/repository/commits/" + commit + "/statuses?all=true&per_page=100&name=" + url.QueryEscape(ReadStatus))
	if err != nil {
		return "", false, err
	}
	var found []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Created     string `json:"created_at"`
		Author      struct {
			Username string `json:"username"`
		} `json:"author"`
	}
	if err := decode(out, &found); err != nil {
		return "", false, err
	}
	text, at, ok := "", "", false
	for _, s := range found {
		if s.Name == ReadStatus && s.Author.Username == app && s.Created >= at {
			text, at, ok = s.Description, s.Created, true
		}
	}
	return text, ok, nil
}

// mark asks mark: {commit, text}, answered {}.
func (c *command) mark(commit, text string) error {
	return c.call("mark", map[string]any{"commit": commit, "text": text}, nil)
}

// marked asks marked: {commit}, answered {text} — "" none.
func (c *command) marked(commit string) (string, bool, error) {
	var a struct {
		Text string `json:"text"`
	}
	err := c.call("marked", map[string]any{"commit": commit}, &a)
	return a.Text, a.Text != "", err
}

// Closing is who closed an issue last, as the forge says: a LabelEvent's
// Author, Bot and Writer; ok false when it is open, or the forge does not
// say. A demotion is lifted only by a person who may push and merge
// (ADR-0042, `normal`).
func Closing(f any, id int) (who LabelEvent, ok bool, err error) {
	if c, can := f.(interface {
		closing(id int) (LabelEvent, bool, error)
	}); can {
		return c.closing(id)
	}
	return LabelEvent{}, false, nil
}

// Every forge with an auto-merge workline speaks is a Merger; the local
// forge has none.
var _ = []Merger{&github{}, &gitlab{}, &command{}, &Fake{}}
