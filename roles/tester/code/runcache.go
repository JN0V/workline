package tester

// The tests a tree holds, kept (#1338): the green proof runs the base
// once only to tell the base's tests from the builder's new ones, by
// their ids. The tester's red proof already ran that same base, the same
// command, minutes before; a guard on the next push of the same pull
// request runs it again. A run of the same tree by the same command, its
// command exited 0, names the same tests: the ids it found are kept in
// the main clone's git folder, keyed by the tree's hash, the command and
// the report, and the base is read there rather than run again. Only ids
// are ever read back, never a result: what is proven green or red is
// always run.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// keptIDsMax is how many trees' ids are kept: the oldest go past it.
const keptIDsMax = 64

// keptIDs is what is kept of a run: the tree, the command, when, the ids.
type keptIDs struct {
	Tree    string    `json:"tree"`
	Command string    `json:"command"`
	Report  string    `json:"report"`
	At      time.Time `json:"at"`
	IDs     []string  `json:"ids"`
}

// idsDir is the main clone's .git/workline/test-ids, which its worktrees
// share; "" when repo is no git repository.
func idsDir(repo string) string {
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return ""
	}
	return filepath.Join(strings.TrimSpace(string(out)), "workline", "test-ids")
}

// idsKey is rev's tree with the command and report that ran on it; ""
// when rev names no tree.
func idsKey(repo, rev, command, report string) (key, tree string) {
	out, err := git(repo, "rev-parse", "--verify", "-q", rev+"^{tree}")
	if err != nil {
		return "", ""
	}
	tree = strings.TrimSpace(out)
	sum := sha256.Sum256([]byte(tree + "\x00" + command + "\x00" + report))
	return hex.EncodeToString(sum[:]), tree
}

// keepIDs keeps the ids r found on rev, run by command: only a run that
// exited 0 and wrote a report, the whole suite having run. A failure to
// keep them is no failure of the run.
func keepIDs(repo, rev, command, report string, r ran) {
	if r.failure != "" || len(r.reports) == 0 || len(r.results) == 0 {
		return
	}
	dir := idsDir(repo)
	key, tree := idsKey(repo, rev, command, report)
	if dir == "" || key == "" {
		return
	}
	k := keptIDs{Tree: tree, Command: command, Report: report, At: time.Now().UTC()}
	for id := range r.results {
		k.IDs = append(k.IDs, id)
	}
	slices.Sort(k.IDs)
	data, err := json.Marshal(k)
	if err != nil || os.MkdirAll(dir, 0o755) != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".kept-")
	if err != nil {
		return
	}
	_, werr := tmp.Write(data)
	if cerr := tmp.Close(); werr != nil || cerr != nil || os.Rename(tmp.Name(), filepath.Join(dir, key+".json")) != nil {
		_ = os.Remove(tmp.Name())
		return
	}
	pruneIDs(dir)
}

// readIDs is the ids kept for rev run by command, and when they were
// found; ok false when none is kept.
func readIDs(repo, rev, command, report string) (ids map[string]bool, at time.Time, ok bool) {
	dir := idsDir(repo)
	key, tree := idsKey(repo, rev, command, report)
	if dir == "" || key == "" {
		return nil, time.Time{}, false
	}
	data, err := os.ReadFile(filepath.Join(dir, key+".json"))
	if err != nil {
		return nil, time.Time{}, false
	}
	var k keptIDs
	if json.Unmarshal(data, &k) != nil || k.Tree != tree || k.Command != command || k.Report != report || len(k.IDs) == 0 {
		return nil, time.Time{}, false
	}
	ids = map[string]bool{}
	for _, id := range k.IDs {
		ids[id] = true
	}
	return ids, k.At, true
}

// pruneIDs removes the oldest kept files past keptIDsMax.
func pruneIDs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type kept struct {
		name string
		at   time.Time
	}
	var all []kept
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if fi, err := e.Info(); err == nil {
			all = append(all, kept{e.Name(), fi.ModTime()})
		}
	}
	if len(all) <= keptIDsMax {
		return
	}
	slices.SortFunc(all, func(a, b kept) int { return b.at.Compare(a.at) })
	for _, k := range all[keptIDsMax:] {
		_ = os.Remove(filepath.Join(dir, k.name))
	}
}
