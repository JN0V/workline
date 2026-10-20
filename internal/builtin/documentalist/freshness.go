package documentalist

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/verdict"
)

// Freshness says how long a doc stays trusted without being read again.
type Freshness struct {
	StaleAfterDays int `json:"stale-after-days"`
}

// staleDocs reports the docs last confirmed too long ago: the newest commit
// their `checked` names is older than stale-after-days. A doc is dated by its
// confirmation, not by its last edit: a doc nobody reads again drifts from a
// world that changed around it, even when its sources did not (Google's
// freshness dates). Docs already suspect or pending are being handled, and a
// commit that cannot be read is reported by the suspect check.
func staleDocs(docs []*Doc, pl *places, f Freshness, now time.Time, skip func(string) bool) []verdict.Finding {
	if f.StaleAfterDays <= 0 {
		return []verdict.Finding{{Rule: "setting-missing", Level: "block",
			Message: "freshness.stale-after-days is not set, so no doc was checked for freshness; set it, or turn the rule off with `enforce`"}}
	}
	var out []verdict.Finding
	for _, d := range docs {
		if skip(d.Path) {
			continue
		}
		var newest time.Time
		var commit string
		for name, checked := range d.Checked {
			where, err := pl.get(name)
			if err != nil || checked == "" || checked == "HEAD" {
				continue
			}
			ct, err := git(where.dir, "log", "-1", "--format=%ct", checked+"^{commit}", "--")
			if err != nil {
				continue
			}
			sec, err := strconv.ParseInt(ct, 10, 64)
			if err != nil {
				continue
			}
			if t := time.Unix(sec, 0); t.After(newest) {
				newest, commit = t, checked
			}
		}
		if newest.IsZero() {
			continue
		}
		days := int(now.Sub(newest).Hours() / 24)
		if days > f.StaleAfterDays {
			out = append(out, verdict.Finding{Rule: "stale", Where: d.Path,
				Message: fmt.Sprintf("last confirmed %d days ago (checked: %s, %s), past freshness.stale-after-days (%d): read it again against its sources and what it describes, then move `checked`",
					days, commit, newest.UTC().Format("2006-01-02"), f.StaleAfterDays)})
		}
	}
	return out
}

// wholeCharsDefault caps what the sources of one doc may take in a task,
// unless the project sets `whole-chars`. A doc whose sources do not fit is
// left for a person, or judged in parts: confirming it without reading them
// would only fake its freshness. Characters, not tokens: the engine measures
// a task before the call, with no tokenizer (agent.Tokens).
const wholeCharsDefault = 20000

// wholeChars is the project's cap on the sources of one doc judged whole.
func wholeChars(s Settings) int {
	if s.WholeChars > 0 {
		return s.WholeChars
	}
	return wholeCharsDefault
}

// wholeCharsMax is the most `whole-chars` may be: its task, the facets and
// an answer asked again fit the role's context budget, 48000 tokens at
// agent.Tokens' ratio (TestLargestTaskFitsTheBudget). A project asking for
// more is refused, said, rather than every doc's prompt refused (#235).
const wholeCharsMax = 25000

// checkWholeChars refuses a `whole-chars` whose task would not fit.
func checkWholeChars(s Settings) error {
	if s.WholeChars > wholeCharsMax {
		return fmt.Errorf("whole-chars %d: over %d, a task would not fit the role's context budget (tokens estimated at 711 + 0.82 a character); set it at most %d", s.WholeChars, wholeCharsMax, wholeCharsMax)
	}
	return nil
}

// taskChars caps a task: the room the doc and the words around its sources
// take is kept, whatever the sources may take.
func taskChars(s Settings) int {
	return taskMaxChars + max(0, wholeChars(s)-wholeCharsDefault)
}

// staleForAgent picks the stale docs to put before the agent, each with its
// sources as they are now; a doc whose sources do not fit says so.
func staleForAgent(findings []verdict.Finding, byPath map[string]*Doc, pl *places, s Settings) map[string]*suspectDoc {
	out := map[string]*suspectDoc{}
	for i := range findings {
		f := &findings[i]
		d := byPath[f.Where]
		if f.Rule != "stale" || d == nil {
			continue
		}
		now, err := sourcesNow(d, pl, wholeChars(s))
		// Too large: a person reads it again, or it is judged in parts.
		sd := &suspectDoc{doc: d, why: []string{f.Message}, evidence: now, tooLarge: err == errTooLarge, whole: err == nil && len(now) > 0}
		if err != nil && err != errTooLarge {
			sd.note = unreadNote(err)
		}
		out[d.Path] = sd
	}
	return out
}

// errTooLarge: a doc's sources take more than limit characters.
var errTooLarge = errors.New("its sources do not fit in a task whole")

// unreadNote says why a doc one of whose sources cannot be read is not put
// before the agent: judged against what is not there, it would be confirmed
// unread (#239). A person judges it.
func unreadNote(err error) string {
	return fmt.Sprintf("(a source cannot be read — %v: not put before the agent; "+personJudges+", naming in `sources` what replaced it)", err)
}

// sourcesNow is each source of a doc as it is now: a section of a doc, or
// every text file under a path. err is errTooLarge past limit characters,
// or says which source cannot be read.
func sourcesNow(d *Doc, pl *places, limit int) ([]string, error) {
	return sourcesAs(d, pl, limit, "as it is now")
}

// sourcesAs is sourcesNow at the revisions pl reads, each source titled as
// it is said there ("as it is now", "at the commit `checked` names").
func sourcesAs(d *Doc, pl *places, limit int, as string) ([]string, error) {
	var evidence []string
	size := 0
	add := func(title, lang, text string) bool {
		size += len(text)
		evidence = append(evidence, fmt.Sprintf("%s, %s:\n\n```%s\n%s\n```", title, as, lang, strings.TrimRight(text, "\n")))
		return size <= limit
	}
	for _, src := range d.Sources {
		name, path, anchor := splitSource(src)
		where, err := pl.get(name)
		if err != nil {
			return nil, err
		}
		named := path
		if name != "" {
			named = name + ":" + path
		}
		if strings.HasSuffix(path, ".md") {
			content, err := git(where.dir, "show", where.rev+":"+path)
			if err != nil {
				return nil, fmt.Errorf("%s cannot be read", named)
			}
			// A heading gone is a section not there, never an empty one.
			section, found := Section(content, anchor)
			if !found {
				return nil, fmt.Errorf("no heading of %s has the anchor #%s", named, anchor)
			}
			if !add(src, "markdown", section) {
				return nil, errTooLarge
			}
			continue
		}
		files, err := git(where.dir, "ls-tree", "-z", "-r", "--name-only", where.rev, "--", path)
		if err != nil || files == "" {
			return nil, fmt.Errorf("%s holds no file", named)
		}
		for _, f := range pathList(files) {
			title := f
			if name != "" {
				title = name + ":" + f
			}
			content, err := git(where.dir, "show", where.rev+":"+f)
			if err != nil {
				return nil, fmt.Errorf("%s cannot be read", title)
			}
			if strings.ContainsRune(content, 0) {
				continue // not text
			}
			if !add(title, "", content) {
				return nil, errTooLarge
			}
		}
	}
	return evidence, nil
}

// staleTask asks the agent to read stale docs again against their sources.
func staleTask(stale map[string]*suspectDoc, s Settings, pl *places, repo string) (string, map[string]map[string]string, error) {
	return docTask(`Kind: stale

No source of these docs changed since they were last confirmed, but that was
long ago. Read each one again against its sources as they are now, given in
full below. If it is still true, return a patch that only sets `+"`checked`"+`
and `+"`verified`"+`. If not, the patch also fixes what is now wrong, and nothing
else. If the sources given do not let you tell, return a note instead.

`, stale, s, pl, repo)
}
