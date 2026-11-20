package documentalist

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Records: docs that say what was true when they were written — a
// changelog, release notes, a decision record, a tried or research record —
// where an old count or version is the record, not a mistake. They are
// known by the names the usual tools give them, and by the project's
// `history` and `decisions` globs, added to those. Our own names alone
// missed backstage's 296 `docs/releases/v1.x.0-changelog.md` and its
// `architecture-decisions/adr001-…` (docs/research/documentalist-genericity.md).

// recordGlobs are the project's own: history docs, and decision records.
// Set once a run, from the settings (useRecords).
var recordGlobs struct{ history, decisions []string }

// useRecords takes the project's `history` and `decisions` globs.
func useRecords(s Settings) {
	recordGlobs.history, recordGlobs.decisions = s.History, s.Decisions
}

// decisionFolders are the folders the usual tools keep decisions in:
// adr-tools and log4brains (doc/adr, docs/adr), MADR (docs/decisions),
// backstage (docs/architecture-decisions), decision-records; and any folder
// named `adr-…`.
var decisionFolders = map[string]bool{"adr": true, "adrs": true, "decisions": true,
	"architecture-decisions": true, "decision-records": true, "decision-log": true}

// historyFolders hold history docs only: release notes, changelogs, research.
var historyFolders = map[string]bool{"research": true, "releases": true,
	"release-notes": true, "changelog": true, "changelogs": true, "news": true}

// historyStems are the names of history docs, `-` and `_` alike.
var historyStems = map[string]bool{"changelog": true, "changes": true, "history": true,
	"news": true, "release-notes": true, "releasenotes": true, "tried": true}

// decisionFile is a record's name: its number first (`0007-…`), or after
// `adr` (`adr001-…`, `ADR-7-…`).
var decisionFile = regexp.MustCompile(`(?i)^(?:adr[-_]?)?(\d+)[-_.][^/]*\.md$`)

// isDecisionFolder says whether a folder's name is one decisions are kept in.
func isDecisionFolder(dir string) bool {
	dir = strings.ToLower(dir)
	return decisionFolders[dir] || strings.HasPrefix(dir, "adr-") || strings.HasPrefix(dir, "adr_")
}

// decisionNumber is a decision record's number, when p is one: a numbered
// file in a decisions folder, or matching the project's `decisions` globs
// (the first number in its name).
func decisionNumber(p string) (int, bool) {
	base := path.Base(p)
	if matchAny(recordGlobs.decisions, p) {
		if m := number.FindString(base); m != "" {
			n, _ := strconv.Atoi(m)
			return n, true
		}
		return 0, false
	}
	m := decisionFile.FindStringSubmatch(base)
	if m == nil || !isDecisionFolder(path.Base(path.Dir(p))) {
		return 0, false
	}
	n, _ := strconv.Atoi(m[1])
	return n, true
}

// isHistory says whether a doc is a record: its name holds `changelog` or
// `release-notes`, or is history, news, changes or tried; a folder of its
// path holds decisions, release notes or research; or the project's
// `history` or `decisions` globs name it.
func isHistory(p string) bool {
	if matchAny(recordGlobs.history, p) || matchAny(recordGlobs.decisions, p) {
		return true
	}
	base := strings.ToLower(path.Base(p))
	stem := strings.ReplaceAll(strings.TrimSuffix(base, path.Ext(base)), "_", "-")
	if historyStems[stem] || strings.Contains(stem, "changelog") ||
		strings.Contains(stem, "release-notes") || strings.Contains(stem, "releasenotes") {
		return true
	}
	for _, dir := range strings.Split(path.Dir(p), "/") {
		if isDecisionFolder(dir) || historyFolders[strings.ToLower(dir)] ||
			strings.HasPrefix(strings.ToLower(dir), "research-") {
			return true
		}
	}
	return false
}
