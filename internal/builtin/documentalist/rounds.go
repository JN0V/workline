package documentalist

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// A release's rounds, one fix (#572): on a release tool's merge request,
// a round that leaves docs for the next keeps its fix in the clone, and
// the next round proposes it with its own, in one merge request of its
// own (ADR-0017). The engine names those earlier rounds' folders in
// WORKLINE_EARLIER_ROUNDS, a path list as PATH is, oldest first: what
// they asked, kept and proposed is read there.

// earlierRounds are the run folders of the rounds before this one whose
// fix this round proposes with its own.
func earlierRounds() []string { return filepath.SplitList(os.Getenv("WORKLINE_EARLIER_ROUNDS")) }

// askedEarlier is a doc an earlier round of the run put before the agent
// and left suspect: not asked again before the next run, and still
// holding the release.
const askedEarlier = "(put before the agent in an earlier round of this run, which left it as it was: not asked again before the next run)"

// holdEarlierRounds keeps from the agent the docs an earlier round of this
// run put before it: what it fixed is in the clone, no longer suspect;
// what it left, asking again in this run would leave the same. Each still
// holds the release, as it did in that round.
func holdEarlierRounds(dirs []string, maps ...map[string]*suspectDoc) error {
	asked := map[string]bool{}
	for _, d := range dirs {
		var judged map[string]map[string]string
		data, err := os.ReadFile(filepath.Join(d, "in", "judged.yaml"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, &judged); err != nil {
			return err
		}
		for p := range judged {
			asked[p] = true
		}
	}
	for _, docs := range maps {
		for p, sd := range docs {
			if asked[p] && sd.note == "" {
				sd.note = askedEarlier
			}
		}
	}
	return nil
}

// earlierSticky is the body of the comment keyed key the latest earlier
// round proposed, newer than the forge's: ok is false when none did.
func earlierSticky(key string) (body string, ok bool, err error) {
	dirs := earlierRounds()
	for i := len(dirs) - 1; i >= 0; i-- {
		intents, err := intent.Read(filepath.Join(dirs[i], "out", "intentions.yaml"))
		if err != nil {
			return "", false, err
		}
		for _, in := range intents {
			if m, isMap := in.Value.(map[string]any); in.Kind == "comment" && isMap && m["sticky"] == key {
				b, _ := m["body"].(string)
				return b, true, nil
			}
		}
	}
	return "", false, nil
}

// releaseFix is what a release's round proposes in its fix's merge
// request (out/release-fix.yaml), for the round after it, which proposes
// it again with its own.
type releaseFix struct {
	Title    string            `yaml:"title"`
	Docs     map[string]string `yaml:"docs"` // each doc, what became of it ("" unsaid)
	Withheld []verdict.Finding `yaml:"withheld,omitempty"`
}

// earlierFix is what the latest earlier round proposed in the fix's merge
// request, nil when no round before this one did.
func earlierFix() (*releaseFix, error) {
	dirs := earlierRounds()
	for i := len(dirs) - 1; i >= 0; i-- {
		data, err := os.ReadFile(filepath.Join(dirs[i], "out", "release-fix.yaml"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var f releaseFix
		if err := yaml.Unmarshal(data, &f); err != nil {
			return nil, err
		}
		return &f, nil
	}
	return nil, nil
}

// withEarlier adds to what this round's fix says what the earlier rounds'
// said: their docs, what became of each, and the places left for a
// person. This round's word on a doc stands.
func (f *releaseFix) withEarlier(outcome map[string]string, judged map[string]map[string]string, withheld []verdict.Finding) (map[string]string, []verdict.Finding) {
	if f == nil {
		return outcome, withheld
	}
	merged := maps.Clone(outcome)
	if merged == nil {
		merged = map[string]string{}
	}
	for _, p := range slices.Sorted(maps.Keys(f.Docs)) {
		_, judgedNow := judged[p]
		if _, said := merged[p]; !said && !judgedNow {
			merged[p] = f.Docs[p]
		}
	}
	return merged, append(slices.Clone(f.Withheld), withheld...)
}

// titled is the fix's title once the earlier rounds' are with it: theirs
// and this round's the same, that one; else what the release waits for.
func (f *releaseFix) titled(title string) string {
	if f == nil || f.Title == "" || f.Title == title {
		return title
	}
	return "docs: what the release waits for"
}
