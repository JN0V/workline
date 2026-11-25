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

	"github.com/JN0V/workline/internal/kit/commitmsg"
	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/role"
)

// The developer's commits pass the project's commit check (#736): each
// subject reads `type(scope): summary`, its type from the issue's kind,
// its scope the folder the issue's Scope shares, its summary the issue's
// title, within the committer's limit and with no code internal to the
// project.

// commitType is a commit's type for an issue of this kind: a fix for a
// bug, a feature for a feature or no kind; for a task, a fix when its
// code changes a shipped file, else a chore (#2198). Release tools hide
// a chore from the notes and bump no version for it: a change to what a
// project ships needs both. The type says what the change does; the
// merge policy reads the issue's kind, never the type.
func commitType(kind string, code []string) string {
	switch kind {
	case devmark.KindFix:
		return "fix"
	case devmark.KindTask:
		if slices.ContainsFunc(code, shipped) {
			return "fix"
		}
		return "chore"
	}
	return "feat"
}

// notShipped are the top folders whose files no project ships: its
// docs and its CI.
var notShipped = []string{"docs", "doc", ".github", ".gitlab", ".circleci"}

// shipped: a code file (tests are apart already) other than docs, the
// CI, a page of prose or a tool's dotfile, all a release leaves out. A
// build or dependency file (CMakeLists.txt, requirements.txt) is
// shipped: it decides what the package builds.
func shipped(f string) bool {
	parts := strings.Split(path.Clean(f), "/")
	if slices.Contains(notShipped, strings.ToLower(parts[0])) {
		return false
	}
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}
	base := strings.ToLower(path.Base(f))
	switch path.Ext(base) {
	case ".md", ".rst", ".adoc":
		return false
	}
	return !slices.Contains([]string{"license", "jenkinsfile", "azure-pipelines.yml"}, base)
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
	issueRef    = regexp.MustCompile(`\s*\([^()]*#[0-9]+\b[^()]*\)|(^|\s)#[0-9]+\b`) // `(see #12)` whole
	emptyParens = regexp.MustCompile(`\(\s*\)|\[\s*\]`)
	spaces      = regexp.MustCompile(`\s+`)
	typeHead    = regexp.MustCompile(`(?i)^[a-z]+(\([^()\s]*\))?!?: `) // a summary's own `type(scope): `
)

// summaryOf is an issue's title as a subject's summary: its references
// to issues and the codes the committer refuses taken out, its first
// letter lower-case unless the word is an acronym, no closing dot.
func summaryOf(title string, s commitmsg.Settings) string {
	t := issueRef.ReplaceAllString(title, " ")
	codes, _ := commitmsg.InternalCodes(t, s)
	maybe, _ := commitmsg.InternalCodes(t, commitmsg.Settings{InternalCodes: s.InternalCodesMaybe, InternalCodesAllow: s.InternalCodesAllow, InternalCodesAllowPatterns: s.InternalCodesAllowPatterns})
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

// subject is `typ(scope): summary`, never cut: the scope dropped when
// the head would take half the committer's limit, the type the
// committer's first when it does not know typ.
func subject(typ, scope, summary string, s commitmsg.Settings) string {
	if len(s.Types) > 0 && !slices.Contains(s.Types, typ) {
		typ = s.Types[0]
	}
	head := typ + ": "
	if scope != "" {
		head = typ + "(" + scope + "): "
	}
	if scope != "" && 2*utf8.RuneCountInString(head) > s.SubjectMax && s.SubjectMax > 0 {
		return subject(typ, "", summary, s)
	}
	return head + summary
}

// committerSettings are the committer's settings for the project, its
// subject limit and the codes it refuses. Unread, the role's own as it
// ships them, else its limit alone, and the finding that says so: the
// subjects may then miss the project's rules.
func committerSettings(repo, where string) (commitmsg.Settings, []verdict.Finding) {
	unread := func(err error, fallback commitmsg.Settings) (commitmsg.Settings, []verdict.Finding) {
		return fallback, []verdict.Finding{{Rule: "commit-rules-unread", Level: "warn", Where: where,
			Message: "the project's commit rules could not be read (" + err.Error() + "): its commits' subjects follow the committer's own, which the project's check may refuse"}}
	}
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), "committer")
	if err != nil {
		return unread(err, commitmsg.Settings{SubjectMax: 72})
	}
	own, _ := commitmsg.SettingsFrom(r.MergedSettings(&role.ProjectConfig{}))
	if own.SubjectMax == 0 {
		own.SubjectMax = 72
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return unread(err, own)
	}
	s, err := commitmsg.SettingsFrom(r.MergedSettings(cfg))
	if err != nil {
		return unread(err, own)
	}
	return s, nil
}

// reviewSubject is the subject of a review round's fix commit.
func reviewSubject(issue, round int, s commitmsg.Settings) string {
	return subject("fix", "", fmt.Sprintf("the review's findings on #%d, round %d", issue, round), s)
}

// head is the subject's head before its agent's summary: a task's `chore`, the longer.
func head(b Bounds) string {
	return subject(commitType(b.Kind, nil), commitScope(b.Scope), "", commitmsg.Settings{})
}
