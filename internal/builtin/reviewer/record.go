package reviewer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// Record is what the reviewer remembers between runs (ADR-0013's way): the
// commits a review answered whole, so they are not asked again, and how many
// reviews ran, which turns the lens a push gets. On a merge request it is
// hidden in the summary comment; on a machine, in the git directory.
type Record struct {
	Reviewed []string `json:"reviewed"`
	Runs     int      `json:"runs"`
}

// SummaryKey keys the reviewer's one comment on a merge request.
const SummaryKey = "summary"

var recordLine = regexp.MustCompile(`<!-- workline:record reviewed=([0-9a-f,]*) runs=(\d+) -->`)

// String is the record as the summary comment and the local file hold it.
func (r Record) String() string {
	return fmt.Sprintf("<!-- workline:record reviewed=%s runs=%d -->", strings.Join(r.Reviewed, ","), r.Runs)
}

// parseRecord reads a record from a text holding one; an empty one when none.
func parseRecord(text string) Record {
	m := recordLine.FindStringSubmatch(text)
	if m == nil {
		return Record{}
	}
	var r Record
	for _, c := range strings.Split(m[1], ",") {
		if c != "" {
			r.Reviewed = append(r.Reviewed, c)
		}
	}
	r.Runs, _ = strconv.Atoi(m[2])
	return r
}

// add records commits as reviewed, once each, and one more run.
func (r Record) add(commits []string) Record {
	out := Record{Reviewed: slices.Clone(r.Reviewed), Runs: r.Runs + 1}
	for _, c := range commits {
		if !slices.Contains(out.Reviewed, c) {
			out.Reviewed = append(out.Reviewed, c)
		}
	}
	return out
}

// localRecord is where a machine keeps the record: in the git directory,
// never tracked.
func localRecord(repo string) string {
	dir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return filepath.Join(repo, ".git", "workline", "reviewer-record")
	}
	return filepath.Join(strings.TrimSpace(dir), "workline", "reviewer-record")
}

// loadRecord reads the record: from the merge request's summary comment when
// the run is on one and may write there, else from the machine's. It says
// where it read it, and why it could not, when it could not.
func loadRecord(repo string, s Settings) (Record, string, error) {
	if t := mergeRequest(); t != nil && s.ForgeWrites {
		f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
		if err != nil || f == nil {
			return Record{}, "forge", fmt.Errorf("the forge: %v", err)
		}
		b, ok := f.(forge.Backlog)
		if !ok {
			return Record{}, "forge", fmt.Errorf("this forge cannot list a merge request's comments")
		}
		comments, err := b.Comments(*t)
		if err != nil {
			return Record{}, "forge", err
		}
		for _, c := range comments {
			if strings.Contains(c, forge.Marker("sticky=reviewer/"+SummaryKey)) {
				return parseRecord(c), "forge", nil
			}
		}
		return Record{}, "forge", nil
	}
	data, err := os.ReadFile(localRecord(repo))
	if os.IsNotExist(err) {
		return Record{}, "local", nil
	}
	return parseRecord(string(data)), "local", err
}

// saveLocal keeps the record on this machine.
func saveLocal(repo string, r Record) error {
	file := localRecord(repo)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(r.String()+"\n"), 0o644)
}

// mergeRequest is the merge request the run is on, when a forge and a
// target are given and the target is one.
func mergeRequest() *forge.Target {
	if os.Getenv("WORKLINE_FORGE") == "" {
		return nil
	}
	kind, id, ok := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	n, err := strconv.Atoi(id)
	if !ok || err != nil || kind != "merge-request" {
		return nil
	}
	return &forge.Target{Kind: kind, ID: n}
}
