package developer

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
)

// The developer's commits pass the project's commit check (#736): each
// subject reads `type(scope): summary`, its type from the issue's kind,
// its scope the folder the issue's Scope shares, its summary the issue's
// title, within the committer's limit and with no code internal to the
// project.

// commitType is a commit's type for an issue of this kind: a fix for a
// bug, a chore for a task, a feature otherwise.
func commitType(kind string) string {
	switch kind {
	case KindFix:
		return "fix"
	case KindTask:
		return "chore"
	}
	return "feat"
}

// commitScope is the name of the folder every path of the Scope sits in,
// "" when they share none: a file's folder, a path with no dot in its
// last name taken for a folder.
func commitScope(scope []string) string {
	var common []string
	for i, p := range scope {
		dir := path.Clean(p)
		if strings.Contains(path.Base(dir), ".") {
			dir = path.Dir(dir)
		}
		parts := strings.Split(dir, "/")
		if dir == "." {
			parts = nil
		}
		if i == 0 {
			common = parts
			continue
		}
		n := 0
		for n < len(common) && n < len(parts) && common[n] == parts[n] {
			n++
		}
		common = common[:n]
	}
	if len(common) == 0 {
		return ""
	}
	name := strings.ToLower(common[len(common)-1])
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`).MatchString(name) {
		return ""
	}
	return name
}

var (
	issueRef    = regexp.MustCompile(`(^|\s)#[0-9]+\b`)
	emptyParens = regexp.MustCompile(`\(\s*\)|\[\s*\]`)
	spaces      = regexp.MustCompile(`\s+`)
)

// summaryOf is an issue's title as a subject's summary: its references
// to issues and the codes the committer refuses taken out, its first
// letter lower-case unless the word is an acronym, no closing dot.
func summaryOf(title string, s committer.Settings) string {
	t := issueRef.ReplaceAllString(title, " ")
	codes, _ := committer.InternalCodes(t, s)
	maybe, _ := committer.InternalCodes(t, committer.Settings{InternalCodes: s.InternalCodesMaybe, InternalCodesAllow: s.InternalCodesAllow})
	for _, c := range append(codes, maybe...) {
		t = regexp.MustCompile(`\b`+regexp.QuoteMeta(c)+`\b`).ReplaceAllString(t, " ")
	}
	t = emptyParens.ReplaceAllString(t, " ")
	t = strings.TrimSpace(spaces.ReplaceAllString(t, " "))
	t = strings.Trim(t, " :;,.-–—")
	t = strings.ReplaceAll(t, " ,", ",")
	if first, n := utf8.DecodeRuneInString(t); n > 0 {
		second, _ := utf8.DecodeRuneInString(t[n:])
		if !unicode.IsUpper(second) {
			t = string(unicode.ToLower(first)) + t[n:]
		}
	}
	return t
}

// subject is `typ(scope): summary`, cut on a word to the committer's
// limit; the scope dropped when the head would take half of it, the
// type the committer's first when it does not know typ.
func subject(typ, scope, summary string, s committer.Settings) string {
	if len(s.Types) > 0 && !slices.Contains(s.Types, typ) {
		typ = s.Types[0]
	}
	if summary == "" {
		summary = "the issue's change"
	}
	head := typ + ": "
	if scope != "" {
		head = typ + "(" + scope + "): "
	}
	limit := s.SubjectMax
	if limit <= 0 {
		return head + summary
	}
	if scope != "" && 2*utf8.RuneCountInString(head) > limit {
		return subject(typ, "", summary, s)
	}
	room := limit - utf8.RuneCountInString(head)
	if utf8.RuneCountInString(summary) <= room {
		return head + summary
	}
	// Whole words while they fit; a first word too long, cut.
	var cut string
	for _, w := range strings.Fields(summary) {
		next := strings.TrimSpace(cut + " " + w)
		if utf8.RuneCountInString(next) > room {
			break
		}
		cut = next
	}
	if cut == "" {
		cut = string([]rune(summary)[:max(room, 1)])
	}
	return head + strings.TrimRight(cut, " :;,.-–—")
}

// committerSettings are the committer's settings for the project, its
// subject limit and the codes it refuses. Unread, the role's own as it
// ships them, else its limit alone, and the finding that says so: the
// subjects may then miss the project's rules.
func committerSettings(repo, where string) (committer.Settings, []verdict.Finding) {
	unread := func(err error, fallback committer.Settings) (committer.Settings, []verdict.Finding) {
		return fallback, []verdict.Finding{{Rule: "commit-rules-unread", Level: "warn", Where: where,
			Message: "the project's commit rules could not be read (" + err.Error() + "): its commits' subjects follow the committer's own, which the project's check may refuse"}}
	}
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), "committer")
	if err != nil {
		return unread(err, committer.Settings{SubjectMax: 72})
	}
	own, _ := committer.SettingsFrom(r.MergedSettings(&role.ProjectConfig{}))
	if own.SubjectMax == 0 {
		own.SubjectMax = 72
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return unread(err, own)
	}
	s, err := committer.SettingsFrom(r.MergedSettings(cfg))
	if err != nil {
		return unread(err, own)
	}
	return s, nil
}

// reviewSubject is the subject of a review round's fix commit.
func reviewSubject(issue, round int, s committer.Settings) string {
	return subject("fix", "", fmt.Sprintf("the review's findings on #%d, round %d", issue, round), s)
}
