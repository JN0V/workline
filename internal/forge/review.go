package forge

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LineComment is a comment on a line of a merge request's change: a
// reviewer's finding where it stands, for its author to answer or resolve
// there, as a person's review comment (#277).
type LineComment struct {
	Path string `json:"path" yaml:"path"`
	Line int    `json:"line" yaml:"line"` // in the file at the head of the change
	// Key is what it says, hidden in it (Marker("line="+Key)): while a
	// comment holding it is open on the merge request, it is not posted
	// again; resolved, it is.
	Key  string `json:"key,omitempty" yaml:"key,omitempty"`
	Body string `json:"body" yaml:"body"`
}

// What became of a line comment (Placed.State).
const (
	LinePosted  = "posted"  // posted now, in this review
	LineOpen    = "open"    // one holding its key is open on the merge request: not posted again
	LineOutside = "outside" // its line is not in the diff the forge shows, which takes no comment there
	LineRefused = "refused" // the forge refused it, Why says how
)

// Placed is what became of a line comment.
type Placed struct {
	LineComment
	State string `json:"state"`
	Why   string `json:"why,omitempty"`
}

// Reviewed is a review's outcome: where a person reads it — the review
// posted, else the merge request's changes, where its comments show —
// and each comment's fate, in the order given.
type Reviewed struct {
	URL    string   `json:"url,omitempty"`
	Placed []Placed `json:"placed"`
}

// LineReviewer is a forge that takes comments on the lines of a merge
// request, in its own review: GitHub's pull request review, GitLab's
// discussions on the diff, the simulated forge's threads. A review never
// approves nor asks for changes (ADR-0020). A comment on a line its diff
// does not show is left out, never forced elsewhere; one the forge
// refuses — a token that may not review — is said, never an error: only
// a forge out of reach is.
type LineReviewer interface {
	Review(id int, head, body string, comments []LineComment) (Reviewed, error)
}

// ReviewHere and ReviewEnd hold the region where a sticky comment carrying
// a review says what became of its comments on lines: the engine writes
// it there once the review is posted (docs/spec/role-outcome.md,
// `comment`). What the role writes between them is for an engine that
// posts no comment on a line, which leaves it as it is: each comment in
// full.
var (
	ReviewHere = Marker("review")
	ReviewEnd  = Marker("review-end")
)

// FillReview puts section in body's review region, the region's markers
// and what they held replaced; a body without the end marker has only
// ReviewHere replaced.
func FillReview(body, section string) string {
	from := strings.Index(body, ReviewHere)
	if from < 0 {
		return body
	}
	to := from + len(ReviewHere)
	if end := strings.Index(body[to:], ReviewEnd); end >= 0 {
		to += end + len(ReviewEnd)
	}
	if section == "" {
		return strings.TrimRight(body[:from], "\n") + "\n\n" + strings.TrimLeft(body[to:], "\n")
	}
	return body[:from] + section + body[to:]
}

// Unplaced is comments none of which is on its line, each for why.
func Unplaced(comments []LineComment, why string) Reviewed {
	r := Reviewed{}
	for _, c := range comments {
		r.Placed = append(r.Placed, Placed{LineComment: c, State: LineRefused, Why: why})
	}
	return r
}

// Section says where the comments on lines are, with a link to the
// review, and gives in full those not on their line, grouped by why: what
// a summary comment says of its review.
func (r Reviewed) Section() string {
	var b strings.Builder
	posted, open := 0, 0
	var off []Placed
	for _, p := range r.Placed {
		switch p.State {
		case LinePosted:
			posted++
		case LineOpen:
			open++
		default:
			off = append(off, p)
		}
	}
	if on := posted + open; on > 0 {
		review := "the review"
		if r.URL != "" {
			review = "[the review](" + r.URL + ")"
		}
		var parts []string
		if posted > 0 {
			parts = append(parts, fmt.Sprintf("%d posted now", posted))
		}
		if open > 0 {
			parts = append(parts, fmt.Sprintf("%d still open from before, not posted again", open))
		}
		comments := "comments"
		if on == 1 {
			comments = "comment"
		}
		fmt.Fprintf(&b, "**On their lines**: %d %s, in %s (%s): reply to each, or resolve it, there.\n\n",
			on, comments, review, strings.Join(parts, ", "))
	}
	var whys []string
	by := map[string][]Placed{}
	for _, p := range off {
		if _, ok := by[p.Why]; !ok {
			whys = append(whys, p.Why)
		}
		by[p.Why] = append(by[p.Why], p)
	}
	for _, why := range whys {
		fmt.Fprintf(&b, "**Here only**, not on their lines — %s:\n\n", strings.TrimSuffix(why, "."))
		for _, p := range by[why] {
			fmt.Fprintf(&b, "`%s:%d`\n\n> %s\n\n", p.Path, p.Line, strings.ReplaceAll(strings.TrimSpace(p.Body), "\n", "\n> "))
		}
	}
	return strings.TrimSuffix(b.String(), "\n\n")
}

// lineMarker hides a line comment's key in it.
func lineMarker(key string) string { return Marker("line=" + key) }

// withKey is a comment's body as posted: its key hidden after it.
func (c LineComment) withKey() string {
	if c.Key == "" {
		return c.Body
	}
	return c.Body + "\n\n" + lineMarker(c.Key)
}

var lineKey = regexp.MustCompile(`<!-- workline:line=(\S+) -->`)

// keysIn lists the line comments' keys a text holds.
func keysIn(body string) []string {
	var keys []string
	for _, m := range lineKey.FindAllStringSubmatch(body, -1) {
		keys = append(keys, m[1])
	}
	return keys
}

// shown is what a merge request's diff shows of its files: each new line
// with its old one when the diff kept it as context, 0 when the change
// added it; and each file's old path, where it was renamed.
type shown struct {
	lines map[string]map[int]int
	old   map[string]string
}

func newShown() shown { return shown{lines: map[string]map[int]int{}, old: map[string]string{}} }

var hunkAt = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// add reads a file's hunks, as a forge gives its patch: GitHub's `patch`,
// GitLab's `diff`.
func (s shown) add(path, old, patch string) {
	if s.lines[path] == nil {
		s.lines[path] = map[int]int{}
	}
	if old != "" && old != path {
		s.old[path] = old
	}
	o, n, in := 0, 0, false
	for _, l := range strings.Split(patch, "\n") {
		if m := hunkAt.FindStringSubmatch(l); m != nil {
			o, _ = strconv.Atoi(m[1])
			n, _ = strconv.Atoi(m[2])
			in = true
			continue
		}
		if !in {
			continue
		}
		switch {
		case strings.HasPrefix(l, "+"):
			s.lines[path][n] = 0
			n++
		case strings.HasPrefix(l, "-"):
			o++
		case strings.HasPrefix(l, " "):
			s.lines[path][n] = o
			n++
			o++
		}
	}
}

// addGit reads `git diff` output, its files' headers included: what the
// simulated forges compute from the repository.
func (s shown) addGit(out string) {
	path, old, header := "", "", false
	var patch []string
	flush := func() {
		if path != "" && path != "/dev/null" {
			s.add(path, old, strings.Join(patch, "\n"))
		}
		path, old, patch = "", "", nil
	}
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			flush()
			header = true
		case header && strings.HasPrefix(l, "--- "):
			old = strings.TrimPrefix(strings.TrimPrefix(l, "--- "), "a/")
		case header && strings.HasPrefix(l, "+++ "):
			path = strings.TrimPrefix(strings.TrimPrefix(l, "+++ "), "b/")
		case hunkAt.MatchString(l):
			header = false
			patch = append(patch, l)
		case !header:
			patch = append(patch, l)
		}
	}
	flush()
}

// at says whether the diff shows line n of path, and its old line when
// kept (0 when added).
func (s shown) at(path string, n int) (old int, ok bool) {
	old, ok = s.lines[path][n]
	return old, ok
}

// oldPath is the path a file had before the change: its own unless renamed.
func (s shown) oldPath(path string) string {
	if o := s.old[path]; o != "" {
		return o
	}
	return path
}

// lineForge is what a review needs of a forge: the lines its diff shows,
// the keys its open line comments hold, a way to post comments, and the
// merge request's changes, where they show.
type lineForge interface {
	shownLines(id int) (shown, error)
	openKeys(id int) (map[string]bool, error)
	// postLines posts the comments on their lines; it returns the review's
	// address, if any, and why each comment it could not post was refused
	// ("" for one posted), or an error for a forge out of reach.
	postLines(id int, head, body string, at shown, comments []LineComment) (string, []string, error)
	changesPage(id int) string
}

// refusal says whether err is the forge refusing a write — a token that
// may not, a request it does not take — rather than not answering.
func refusal(err error) bool { return errors.Is(err, errForbidden) || errors.Is(err, errRefused) }

// errRefused marks a request the forge answered and did not take: a
// position it does not show, a value it does not accept (HTTP 400, 422).
var errRefused = fmt.Errorf("%w: refused", ErrUnreachable)

// said is what the forge said refusing: the error, without the words
// that only mark its kind.
func said(err error) string {
	return strings.TrimPrefix(strings.TrimPrefix(err.Error(), errForbidden.Error()+": "), errRefused.Error()+": ")
}

// review places each comment: open already, outside the diff, or posted
// in one review; what the forge refuses is said, each comment's why.
func review(f lineForge, id int, head, body string, comments []LineComment) (Reviewed, error) {
	r := Reviewed{Placed: make([]Placed, len(comments))}
	for i, c := range comments {
		r.Placed[i] = Placed{LineComment: c}
	}
	refuseAll := func(why string) (Reviewed, error) {
		for i := range r.Placed {
			if r.Placed[i].State == "" {
				r.Placed[i].State, r.Placed[i].Why = LineRefused, why
			}
		}
		if r.URL == "" {
			r.URL = f.changesPage(id)
		}
		return r, nil
	}
	if len(comments) == 0 {
		return r, nil
	}
	at, err := f.shownLines(id)
	if refusal(err) {
		return refuseAll("the forge refused to show the merge request's diff: " + said(err))
	}
	if err != nil {
		return r, err
	}
	open, err := f.openKeys(id)
	if refusal(err) {
		return refuseAll("the forge refused to list the merge request's comments on lines: " + said(err))
	}
	if err != nil {
		return r, err
	}
	var todo []LineComment
	var where []int
	for i, c := range comments {
		_, shows := at.at(c.Path, c.Line)
		switch {
		case c.Key != "" && open[c.Key]:
			r.Placed[i].State = LineOpen
		case !shows:
			r.Placed[i].State, r.Placed[i].Why = LineOutside, "its line is not in the merge request's diff, where the forge takes a comment"
		default:
			todo, where = append(todo, c), append(where, i)
		}
	}
	if len(todo) > 0 {
		url, whys, err := f.postLines(id, head, body, at, todo)
		if refusal(err) {
			return refuseAll("the forge refused the review: " + said(err))
		}
		if err != nil {
			return r, err
		}
		r.URL = url
		for j, i := range where {
			r.Placed[i].State = LinePosted
			if j < len(whys) && whys[j] != "" {
				r.Placed[i].State, r.Placed[i].Why = LineRefused, whys[j]
			}
		}
	}
	if r.URL == "" {
		r.URL = f.changesPage(id)
	}
	return r, nil
}
