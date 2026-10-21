package forge

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Own is workline's own account on a forge: the one its applying jobs
// write with. A role reads its records back only from a comment that
// account wrote, and KeepOwn edits only such a comment (ADR-0051, step
// 1): anyone may comment on a public merge request, a comment another
// wrote stays theirs to edit, and a record read from it could make a
// role take commits as already read.
type Own struct {
	logins []string // its logins
	app    string   // GitHub: the App's id, as a comment names the App that wrote it
	all    bool     // every comment: the local forge's, kept in the clone, the person's own
	said   bool     // the comments the forge says are its own (Note.Own): a command's
	// unsigned: a comment whose author is not given, the simulated
	// forge's body alone, is its own; never on a real forge, which names
	// every comment's author.
	unsigned bool
	name     string // how a message names it
}

// Wrote says whether workline's own account wrote n.
func (o Own) Wrote(n Note) bool {
	switch {
	case o.all:
		return true
	case o.said:
		return n.Own
	case o.app != "" && n.App == o.app:
		return true
	case n.Author == "":
		return o.unsigned && !n.Bot && !n.Insider
	}
	return slices.Contains(o.logins, n.Author)
}

func (o Own) String() string { return o.name }

// OwnAccount is workline's own account on f, as f knows it: GitHub's App
// or the token's user, GitLab's bot, the simulated forge's token's user,
// every comment of the local forge's; a command's, the comments it says
// are (docs/spec/forge-command.md).
func OwnAccount(f any) (Own, error) {
	if o, ok := f.(interface{ own() (Own, error) }); ok {
		return o.own()
	}
	return Own{name: "none: this forge does not say who workline writes as"}, nil
}

// KeepOwn keeps one comment carrying marker on t, edited to body on each
// run as Sticky does, but only one workline's own account wrote: the one
// a role reads its record back from (ReadKept). Another's carrying the
// marker is left, and workline's own posted after it, even with create
// false. A forge that keeps its comments itself, a command's, keeps it
// as its Sticky does.
func KeepOwn(f Forge, t Target, body, marker string, create bool) error {
	if o, ok := f.(interface {
		stickyOwn(t Target, body, marker string, create bool) error
	}); ok {
		return o.stickyOwn(t, body, marker, create)
	}
	return f.Sticky(t, body, marker, create)
}

// Kept is a merge request's or an issue's comments as a role reads its
// records back from them: only those workline's own account wrote.
type Kept struct {
	notes []Note
	own   Own
}

// ReadKept lists t's comments, and who workline's own account is. A
// token that may not tell who is of the project still reads them: who
// wrote each is all a record needs.
func ReadKept(f Forge, t Target) (Kept, error) {
	b, ok := f.(Backlog)
	if !ok {
		return Kept{}, errors.New("this forge cannot list a merge request's comments")
	}
	notes, err := b.Notes(t)
	if err != nil && !errors.Is(err, ErrAuthorsUnknown) {
		return Kept{}, err
	}
	own, err := OwnAccount(f)
	if err != nil {
		return Kept{}, fmt.Errorf("who workline's own account is could not be read: %w", err)
	}
	return Kept{notes: notes, own: own}, nil
}

// Last is the last comment carrying marker that workline's own account
// wrote: the one KeepOwn edits.
func (k Kept) Last(marker string) (string, bool) {
	for i := len(k.notes) - 1; i >= 0; i-- {
		if n := k.notes[i]; strings.Contains(n.Body, marker) && k.own.Wrote(n) {
			return n.Body, true
		}
	}
	return "", false
}

// Unread says why no record was read when only comments workline's own
// account did not write carry marker: who wrote them, and who is read;
// nil when one of its own carries it, or none does.
func (k Kept) Unread(marker string) error {
	if _, ok := k.Last(marker); ok {
		return nil
	}
	var who []string
	for _, n := range k.notes {
		if !strings.Contains(n.Body, marker) {
			continue
		}
		w := "a comment whose author the forge does not say"
		if n.Author != "" {
			w = "@" + n.Author + "'s comment"
		}
		if !slices.Contains(who, w) {
			who = append(who, w)
		}
	}
	if len(who) == 0 {
		return nil
	}
	verb := "carries"
	if len(who) > 1 {
		verb = "carry"
	}
	return fmt.Errorf("%s %s its marker: only workline's own account's is read (%s)", strings.Join(who, " and "), verb, k.own)
}

// own: the App the workflow names, by its slug (WORKLINE_APP) or by its
// id (WORKLINE_APP_ID), which a job holding no App token knows too —
// one judging a pull request —; else the token's user; in GitHub
// Actions, where GitHub does not say who the job's own token is,
// github-actions[bot]. Read once.
func (g *github) own() (Own, error) {
	if g.owner != nil {
		return *g.owner, nil
	}
	var o Own
	if app := strings.TrimSpace(os.Getenv("WORKLINE_APP")); app != "" {
		o.logins, o.name = []string{app + "[bot]"}, app+"[bot], the GitHub App WORKLINE_APP names"
	}
	if id := strings.TrimSpace(os.Getenv("WORKLINE_APP_ID")); id != "" {
		o.app = id
		if o.name == "" {
			o.name = "the GitHub App WORKLINE_APP_ID names (" + id + ")"
		}
	}
	if o.name == "" {
		login, err := g.me()
		switch {
		case err == nil:
			o.logins, o.name = []string{login}, login+", the token's user"
		case errors.Is(err, errForbidden) && os.Getenv("GITHUB_ACTIONS") == "true":
			o.logins = []string{"github-actions[bot]"}
			o.name = "github-actions[bot], the job's own token; with a GitHub App, give the job its id in WORKLINE_APP_ID"
		default:
			return Own{}, err
		}
	}
	g.owner = &o
	return o, nil
}

// own: the users the project names bots (`gitlab: {bots: […]}`), and the
// user of the bot's token, WORKLINE_GITLAB_TOKEN, which the applying jobs
// write with, as GitLab says (GET /user); with no bot's token, the user
// of the token in use, a person's on their machine. The job's own token
// does not say who it is. Read once.
func (g *gitlab) own() (Own, error) {
	if g.owner != nil {
		return *g.owner, nil
	}
	if err := g.connect(); err != nil {
		return Own{}, err
	}
	g.botNamed("") // the project's named bots, read
	o := Own{logins: slices.Clone(g.named)}
	u := *g
	if bot := os.Getenv("WORKLINE_GITLAB_TOKEN"); bot != "" {
		u.header, u.token = "PRIVATE-TOKEN", bot
	}
	if u.header != "JOB-TOKEN" {
		out, err := u.api("user")
		if err != nil {
			return Own{}, err
		}
		var me struct {
			Username string `json:"username"`
		}
		if err := decode(out, &me); err != nil {
			return Own{}, err
		}
		if me.Username != "" {
			o.logins = append(o.logins, me.Username)
		}
	}
	o.name = "none: the job's own token does not say who it is; set WORKLINE_GITLAB_TOKEN, or name the bot in gitlab: {bots: […]}"
	if len(o.logins) > 0 {
		o.name = strings.Join(o.logins, ", ")
	}
	g.owner = &o
	return o, nil
}

// own: the simulated forge's token's user; a comment its file gives as
// its body alone is what the engine wrote before, its own too.
func (f *Fake) own() (Own, error) {
	return Own{logins: []string{fakeBot}, unsigned: true, name: fakeBot}, nil
}

// own: the local forge keeps no author: every comment in the clone is
// the person's, workline's on their behalf.
func (l *Local) own() (Own, error) { return Own{all: true, name: "this clone's"}, nil }

// own: the comments the command says workline's own account wrote
// (`own`, docs/spec/forge-command.md).
func (c *command) own() (Own, error) {
	return Own{said: true, name: "the comments the forge command says are its own"}, nil
}
