package line

// Staleness, per state (ADR-0059, decision 7): an issue left in a state
// past its age limit (routing.stale), counted from when the forge says
// its label was put on, whose Scope names a path or a code name the
// repository no longer holds, goes back to refine, the reason said in
// the line's comment; the product owner, woken by refine, reads it
// again. The check asks no agent, and reads the Scope first: it costs
// no API call, so the label's date is read only for an issue whose
// Scope names something gone.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	wlclock "github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// clock is the line's time, read through the one seam (ADR-0063).
func clock() time.Time { return wlclock.Now() }

var (
	// codeName is a word a Scope names code by: `stateOf`,
	// `Config.Wakes`, `multiply_all()`.
	codeName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*(\(\))?$`)
	// fileExt ends a word naming a file: `board.go`, `config.yaml`.
	fileExt = regexp.MustCompile(`\.(go|md|ya?ml|json|toml|sh|bash|py|js|mjs|cjs|ts|tsx|jsx|rs|c|h|cc|cpp|hpp|java|kt|rb|php|cs|swift|txt|html|css|scss|sql|xml|proto|mod|sum|cfg|ini|lock|tf|gradle)$`)
)

// scopeIndex answers, for the run, whether the repository still holds
// what a Scope names: its files listed once, each name searched once.
type scopeIndex struct {
	repo  string
	files []string
	read  bool
	err   error
	said  bool // the index could not be read, said once
	names map[string]bool
	// builtins are the listed packs' built-in names, read once.
	builtins []string
	packed   bool
}

func newScopeIndex(repo string) *scopeIndex {
	return &scopeIndex{repo: repo, names: map[string]bool{}}
}

// scopeKind says what a Scope's word names: "path" (a file or a folder),
// "name" (code: a word with a capital, an underscore, a dot or "()"),
// or "" for any other word, never checked — a label, a setting, a flag,
// a plain word.
func scopeKind(w string) string {
	switch {
	case strings.ContainsAny(w, ":*?[]{}<>$=@#~|\"',;!%^&+\\") || strings.HasPrefix(w, "-") ||
		strings.HasPrefix(w, "/") || strings.HasPrefix(w, "../"):
		return ""
	case strings.Contains(w, "/") || fileExt.MatchString(w):
		return "path"
	case codeName.MatchString(w) && (strings.ContainsAny(w, "_.(") || strings.ToLower(w) != w):
		return "name"
	}
	return ""
}

// gone are the words of body's Scope the repository no longer holds, in
// their order: a path no file or folder has, a code name no file but
// Markdown holds as a word (docs and decision records keep old names).
func (x *scopeIndex) gone(body string) ([]string, error) {
	var out []string
	for _, w := range scopeWordsHeld(work.ByMeaning(body)[work.Scope].Text) {
		kind := scopeKind(w)
		if kind == "" || slices.Contains(out, w) {
			continue
		}
		if err := x.load(); err != nil {
			return nil, err
		}
		if work.InRepo(w, x.files) != "" {
			continue
		}
		switch kind {
		case "path":
			if !x.folder(w) {
				out = append(out, w)
			}
		case "name":
			if x.builtin(w) {
				continue
			}
			held, err := x.holds(w)
			if err != nil {
				return nil, err
			}
			if !held {
				out = append(out, w)
			}
		}
	}
	return out, nil
}

// scopeWordsHeld are a Scope's words but the paths of a line `New:`:
// files still to be written are not in the code yet, by design. The
// code names on such a line are still checked.
func scopeWordsHeld(scope string) []string {
	var out []string
	for _, line := range strings.Split(scope, "\n") {
		isNew := work.MarksNew(line)
		for _, w := range work.ScopeWords(line) {
			if isNew && scopeKind(w) == "path" {
				continue
			}
			out = append(out, w)
		}
	}
	return out
}

// builtin says whether w, a code name, is one the project's listed
// language packs give as the language's own (`builtin-names`): the
// whole word, or its last part. A pack that cannot be read gives none.
func (x *scopeIndex) builtin(w string) bool {
	if !x.packed {
		x.packed = true
		x.builtins = role.PackBuiltinNames(x.repo)
	}
	name := strings.TrimSuffix(w, "()")
	return slices.Contains(x.builtins, name) || slices.Contains(x.builtins, name[strings.LastIndex(name, ".")+1:])
}

func (x *scopeIndex) load() error {
	if x.read {
		return x.err
	}
	x.read = true
	// -z: each name as it is, never quoted (core.quotepath).
	out, err := exec.Command("git", "-C", x.repo, "ls-files", "-z").Output()
	if err != nil {
		x.err = fmt.Errorf("the repository's files could not be listed (git ls-files: %v)", err)
		return x.err
	}
	x.files = strings.FieldsFunc(string(out), func(r rune) bool { return r == 0 })
	return nil
}

// folder says whether a file lies under the folder w names: its path, or
// a path ending with it.
func (x *scopeIndex) folder(w string) bool {
	p := path.Clean(w)
	return slices.ContainsFunc(x.files, func(f string) bool {
		return strings.HasPrefix(f, p+"/") || strings.Contains(f, "/"+p+"/")
	})
}

// holds says whether a file of the repository but Markdown holds the
// code name w as a word: its last part, `Config.Wakes` searched as
// Wakes, `add()` as add.
func (x *scopeIndex) holds(w string) (bool, error) {
	name := strings.TrimSuffix(w, "()")
	name = name[strings.LastIndex(name, ".")+1:]
	if held, ok := x.names[name]; ok {
		return held, nil
	}
	err := exec.Command("git", "-C", x.repo, "grep", "-q", "-w", "-F", "-e", name, "--", ".", ":(exclude)*.md").Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		x.names[name] = true
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		x.names[name] = false
	default:
		return false, fmt.Errorf("the repository could not be searched for %s (git grep: %v)", name, err)
	}
	return x.names[name], nil
}

// stale checks issue is, in state s, for staleness: its Scope first, then
// its age, then the line's record. A write moving it back to refine; or,
// when a person put it back after the line's move, one saying it once
// and leaving it; nil when it is not stale, or that was said.
func stale(res *Result, cfg *routing.Config, b forge.Backlog, x *scopeIndex, is forge.Issue, s routing.State) (*StateWrite, error) {
	days := cfg.StaleDays(s.Name)
	if _, ok := cfg.State(refineState); days == 0 || s.Name == refineState || !ok || x == nil {
		return nil, nil
	}
	where := fmt.Sprintf("#%d", is.ID)
	gone, err := x.gone(is.Body)
	if err != nil {
		if !x.said {
			x.said = true
			res.Findings = append(res.Findings, verdict.Finding{Rule: "stale-unchecked", Level: "warn", Where: where,
				Message: err.Error() + ": staleness not checked this run"})
		}
		return nil, nil
	}
	if len(gone) == 0 {
		return nil, nil
	}
	t, err := b.Trail(is.ID, s.Label)
	if err != nil {
		return nil, err
	}
	at, known := labeledAt(t.Labeled)
	if !known {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "stale-unknown", Level: "info", Where: where,
			Message: fmt.Sprintf("its %s names %s, no longer in the code, but the forge does not say when it got %s: not said stale", work.Name(work.Scope), said(gone), s.Label)})
		return nil, nil
	}
	if clock().Sub(at) < time.Duration(days)*24*time.Hour {
		return nil, nil
	}
	rec, had, _, _, err := readRecordSpec(b, is.ID)
	if err != nil {
		return nil, err
	}
	why := fmt.Sprintf("in %s since %s, past its limit of %d days (routing.stale), and its %s names %s, no longer in the code",
		s.Name, at.UTC().Format("2006-01-02"), days, work.Name(work.Scope), said(gone))
	if had && rec.Stale == s.Name {
		// Moved out as stale once, and a person put it back: the move
		// undone is theirs to make again (ADR-0059, amended).
		if rec.Kept == t.Labeled {
			return nil, nil
		}
		why = fmt.Sprintf("a person put %s back after the line moved it out as stale; %s", s.Label, why)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-stale-kept", Level: "info", Where: where, Message: why + ": left as it is"})
		rec.Kept = t.Labeled
		return &StateWrite{Issue: is.ID, Record: rec, Kept: why}, nil
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-stale", Level: "warn", Where: where, Message: why + ": back to refine"})
	now := Record{State: s.Name, Body: bodyHash(is.Body), Labeled: t.Labeled, Hops: rec.Hops, Reached: rec.Reached, Stale: s.Name, Spent: rec.Spent, Counted: rec.Counted, Sized: rec.Sized, Last: rec.Last}
	return &StateWrite{Issue: is.ID, Record: now, Stale: why}, nil
}

// labeledAt reads a label's date as the forge gives it, RFC 3339 or
// YYYY-MM-DD; false when it gives none.
func labeledAt(s string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	t, err := time.Parse(time.DateOnly, s)
	return t, err == nil
}

// said lists words in backticks: `a`, `b` and `c`.
func said(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = "`" + w + "`"
	}
	if len(q) == 1 {
		return q[0]
	}
	return strings.Join(q[:len(q)-1], ", ") + " and " + q[len(q)-1]
}

// staleBody writes a stale issue's move back to refine — its state's
// labels, its card to the board's refine column — and is the line's
// comment saying why; or, for one a person put back, the comment alone.
func staleBody(f forge.Forge, cfg *routing.Config, repo string, t forge.Target, w StateWrite) (string, error) {
	left, _ := cfg.State(w.Record.Stale)
	if w.Kept != "" {
		return backlog.Said("workline's line", "finds this issue stale, and leaves it", w.Kept,
			"a person decides: moves it to refine by its label, or brings its "+work.Name(work.Scope)+" up to date") + "\n\n" + recordMarker(w.Record), nil
	}
	add, remove, err := cfg.Move(refineState)
	if err != nil {
		return "", err
	}
	card := board.Along(f, repo, refineState)
	add, remove = card.Fold(add, remove)
	if err := f.Label(t, add, remove); err != nil {
		return "", err
	}
	card.Write(f, t.ID)
	return backlog.Said("workline's line", "moves this issue back to refine, stale", w.Stale,
		fmt.Sprintf("the product owner reads it again; to undo, put `%s` back", left.Label)) + "\n\n" + recordMarker(w.Record), nil
}

// recordMarker is the record alone, as the engine reads it back.
func recordMarker(r Record) string {
	data, _ := json.Marshal(r) // strings and an int: never fails
	return "<!-- workline:" + recordMark + string(data) + " -->"
}
