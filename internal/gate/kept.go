package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/linters"
)

// The findings of main's gates, kept by commit (ADR-0050): main's run
// after a merge writes its head's, every finding of each SARIF check; a
// merge request based on that commit reads them as its merge base's,
// rather than running the tools there again. Kept only for the same
// command, config and tool: else the rerun stays (ADR-0049, section 6).

// keptVersion is the form of a kept file: one of another form is not read.
const keptVersion = 1

// keptFile is a commit's kept findings, by gate and check.
type keptFile struct {
	V      int                              `json:"v"`
	Commit string                           `json:"commit"`
	Gates  map[string]map[string]*keptCheck `json:"gates"`
}

// keptCheck is what one SARIF check found at the commit, and what it ran.
type keptCheck struct {
	// Command: what tells the findings, hashed: the check's command, its
	// output, its `config`.
	Command string `json:"command"`
	// Tool: the tools and versions its SARIF names; Binaries, the tools
	// its command runs, as found on the PATH, by their content's hash.
	Tool     string       `json:"tool,omitempty"`
	Binaries string       `json:"binaries,omitempty"`
	Seconds  float64      `json:"seconds"` // its run there
	Results  []keptResult `json:"results"`
}

// keptResult is a finding as the comparison reads it: its identity, the
// file as the commit names it.
type keptResult struct {
	Tool string `json:"tool,omitempty"`
	Rule string `json:"rule,omitempty"`
	Path string `json:"path,omitempty"`
	Rest string `json:"rest"` // the tool's fingerprints, else the message and the line's text
}

// keptPath is where a commit's findings are kept in dir.
func keptPath(dir, commit string) string { return filepath.Join(dir, commit+".json") }

// readKept reads the findings kept at commit; nil when none are, or in a
// form this engine does not read.
func readKept(dir, commit string) *keptFile {
	if dir == "" || commit == "" {
		return nil
	}
	data, err := os.ReadFile(keptPath(dir, commit))
	if err != nil {
		return nil
	}
	var k keptFile
	if json.Unmarshal(data, &k) != nil || k.V != keptVersion || k.Commit != commit {
		return nil
	}
	return &k
}

// keep writes a check's findings at commit, beside those other checks
// and gates kept there.
func keep(dir, commit, gate, id string, c *keptCheck) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	k := readKept(dir, commit)
	if k == nil {
		k = &keptFile{V: keptVersion, Commit: commit}
	}
	if k.Gates == nil {
		k.Gates = map[string]map[string]*keptCheck{}
	}
	if k.Gates[gate] == nil {
		k.Gates[gate] = map[string]*keptCheck{}
	}
	k.Gates[gate][id] = c
	data, err := json.Marshal(k)
	if err != nil {
		return err
	}
	tmp := keptPath(dir, commit) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, keptPath(dir, commit))
}

// commandOf is what tells a check's findings, hashed: two checks alike
// here find the same in the same tree with the same tool.
func commandOf(c Check) string {
	sum := sha256.Sum256([]byte(strings.Join(append([]string{c.Run, c.Output}, c.Config...), "\x00")))
	return hex.EncodeToString(sum[:])
}

// binaries are the tools a check's command runs, as found on the PATH,
// each by its content's hash: the same name may be another version.
func binaries(c Check) string {
	var out []string
	for _, t := range linters.Ran(c.Run) {
		p, err := exec.LookPath(t)
		if err != nil {
			continue
		}
		if r, err := filepath.EvalSymlinks(p); err == nil {
			p = r
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err == nil {
			out = append(out, t+"="+hex.EncodeToString(h.Sum(nil))[:16])
		}
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// keptResults are a run's findings as kept, read in its tree.
func keptResults(rs []*result, l *lines) []keptResult {
	out := make([]keptResult, 0, len(rs))
	for _, r := range rs {
		out = append(out, keptResult{Tool: r.tool, Rule: r.rule, Path: r.uri, Rest: r.rest(l)})
	}
	return out
}

// keptBase is the merge base's findings of a check, as kept on main, when
// they stand for a run there: the same command, none of its config
// changed by the change, the same tool. Else nil, and why, for the
// summary: the check then runs again at the merge base.
func keptBase(k *keptFile, gate string, c Check, ch *change, tool, bins string) (*keptCheck, string) {
	if k == nil {
		return nil, ""
	}
	kc := k.Gates[gate][c.ID]
	switch {
	case kc == nil:
		return nil, "not kept there"
	case kc.Command != commandOf(c):
		return nil, "kept for another command"
	}
	config := configFor([]Check{c})
	var touched []string
	for file := range ch.files {
		if config(file) {
			touched = append(touched, file)
		}
	}
	if len(touched) > 0 {
		sort.Strings(touched)
		return nil, "the change touches its config, " + strings.Join(touched, ", ")
	}
	switch {
	case tool == "" && bins == "":
		return nil, "its tool's version is not known"
	case kc.Tool != tool:
		return nil, fmt.Sprintf("kept with %s, the head ran %s", orNone(kc.Tool), orNone(tool))
	case kc.Binaries != bins:
		return nil, "kept with another build of its tool"
	}
	return kc, ""
}

func orNone(s string) string {
	if s == "" {
		return "a tool naming no version"
	}
	return s
}

// clean says whether the working tree is HEAD's, every tracked file as
// committed: findings kept by HEAD's commit must be that commit's.
func clean(repo string) error {
	out, err := git(repo, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return errors.New("the tree differs from HEAD's commit")
	}
	return nil
}
