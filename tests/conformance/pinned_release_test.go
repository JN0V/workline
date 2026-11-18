package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// workline's own line runs a published release, pinned, as a user's does
// (ADR-0071 §1): every job with a write token, and gates and judge, take
// the engine's action with the version WORKLINE_VERSION pins; warn, test
// and the head pass run the head's engine. The pin is one value across the
// workflows, and release-please never moves it.

// The workflows and jobs that run the pinned release; any other job taking
// the engine runs the head's.
var pinnedJobs = map[string][]string{
	"workline-answer.yml":    nil, // nil: every job taking the engine
	"workline-apply.yml":     nil,
	"workline-bounds.yml":    nil,
	"workline-develop.yml":   nil,
	"workline-gardening.yml": nil,
	"workline-merge.yml":     nil,
	"workline-sample.yml":    nil,
	"workline.yml":           {"gates", "judge", "pinned"},
}

// The workflows running a release ahead of the pin, and why: a command
// the pin does not hold yet. Their WORKLINE_VERSION is their own, a
// release, never behind the pin; the pin's pull request catches them up.
var aheadJobs = map[string]string{
	"workline-loop.yml": "tick, first released in v0.79.0 (#1665; ADR-0071's N-1 rule)",
}

var headJobs = map[string][]string{
	"workline.yml":      {"head-gates", "head-judge"},
	"workline-warn.yml": nil,
}

const pinRef = "${{ env.WORKLINE_VERSION }}"

func TestOwnLineRunsThePinnedRelease(t *testing.T) {
	own, _ := filepath.Glob("../../.github/workflows/*.yml")
	version := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	pins, ahead := map[string]string{}, map[string]string{}
	for _, f := range own {
		base := filepath.Base(f)
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "x-release-please-version") {
			t.Errorf("%s: carries release-please's marker: a release would move the line's pin, untried", base)
		}
		var wf struct {
			Env  map[string]string `yaml:"env"`
			Jobs map[string]struct {
				ContinueOnError bool         `yaml:"continue-on-error"`
				Steps           []engineStep `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		pinned, isPinned := pinnedJobs[base]
		heads, isHead := headJobs[base]
		_, isAhead := aheadJobs[base]
		if isAhead {
			if !version.MatchString(wf.Env["WORKLINE_VERSION"]) {
				t.Errorf("%s: WORKLINE_VERSION %q is not a release", base, wf.Env["WORKLINE_VERSION"])
			}
			ahead[base] = wf.Env["WORKLINE_VERSION"]
		}
		if isPinned {
			if !version.MatchString(wf.Env["WORKLINE_VERSION"]) {
				t.Errorf("%s: WORKLINE_VERSION %q is not a release", base, wf.Env["WORKLINE_VERSION"])
			}
			pins[base] = wf.Env["WORKLINE_VERSION"]
		}
		for name, job := range wf.Jobs {
			for _, s := range job.Steps {
				if s.Uses != engineAction {
					continue
				}
				wantPin := isAhead || isPinned && (pinned == nil || slices.Contains(pinned, name))
				wantHead := isHead && (heads == nil || slices.Contains(heads, name))
				switch {
				case wantPin && s.With["version"] != pinRef:
					t.Errorf("%s, job %s: runs the engine with version %q, not the pinned release (%s)", base, name, s.With["version"], pinRef)
				case !wantPin && s.With["version"] != "":
					t.Errorf("%s, job %s: runs a release; ADR-0071 gives it the head's engine", base, name)
				case !wantPin && !wantHead:
					t.Errorf("%s, job %s: takes the engine, in no list of ADR-0071's table", base, name)
				}
			}
			// The head pass never holds the line's verdict.
			if isHead && base == "workline.yml" && slices.Contains(heads, name) && !job.ContinueOnError {
				t.Errorf("%s, job %s: the head pass is not continue-on-error: it would block the line", base, name)
			}
		}
	}
	if len(pins) != len(pinnedJobs) {
		t.Errorf("found the pin in %d workflows, want %d: %v", len(pins), len(pinnedJobs), pins)
	}
	first := ""
	for f, v := range pins {
		if first == "" {
			first = v
		}
		if v != first {
			t.Errorf("the pins differ: %v (%s); one release runs the line", pins, f)
			break
		}
	}

	for f, v := range ahead {
		if first != "" && semverLess(v, first) {
			t.Errorf("%s: WORKLINE_VERSION %s is behind the pin, %s: it runs ahead of it, for %s", f, v, first, aheadJobs[f])
		}
	}

	data, err := os.ReadFile("../../release-please-config.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), ".github/") {
		t.Errorf("release-please-config.json lists a file under .github/: a release would move the line's pin, untried")
	}
}

// The release step runs as the action holds it, curl a stand-in serving a
// release's files: an archive whose checksum does not match, or a binary
// saying another version, puts nothing on PATH.
func TestOwnPinnedReleaseIsCheckedOrNothingRuns(t *testing.T) {
	var run string
	for _, s := range readEngineAction(t) {
		if s.Name == "The pinned release" {
			run = s.Run
			if s.If != "inputs.version != ''" || s.Env["WORKLINE_VERSION"] != "${{ inputs.version }}" {
				t.Errorf("the release step is not the version input's (if %q, env %v)", s.If, s.Env)
			}
		}
	}
	if run == "" {
		t.Fatal("the engine's action holds no step taking the pinned release")
	}
	for _, c := range []struct {
		name, says string
		badSum     bool
		runs       bool
	}{
		{name: "the release pinned", says: "v1.2.3", runs: true},
		{name: "a checksum that does not match", says: "v1.2.3", badSum: true},
		{name: "a binary saying another version", says: "v1.2.2"},
	} {
		dir := t.TempDir()
		served := filepath.Join(dir, "served")
		stage := filepath.Join(dir, "stage")
		for _, d := range []string{served, stage, filepath.Join(dir, "fake")} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(stage, "workline"), []byte("#!/bin/sh\necho "+c.says+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		archive := filepath.Join(served, "workline_linux_amd64.tar.gz")
		if out, err := exec.Command("tar", "-czf", archive, "-C", stage, "workline").CombinedOutput(); err != nil {
			t.Fatalf("tar: %v %s", err, out)
		}
		data, _ := os.ReadFile(archive)
		sum := sha256.Sum256(data)
		if c.badSum {
			sum[0] ^= 0xff
		}
		sums := hex.EncodeToString(sum[:]) + "  workline_linux_amd64.tar.gz\n"
		if err := os.WriteFile(filepath.Join(served, "checksums.txt"), []byte(sums), 0o644); err != nil {
			t.Fatal(err)
		}
		// curl -fsSLO <url> saves the file named by the URL's end; curl
		// -fsSL <url> prints it. Only the pinned version's URL is served.
		curl := `#!/bin/sh
out=; for a in "$@"; do case "$a" in -*O*) out=1 ;; -*) ;; *) url=$a ;; esac; done
case "$url" in https://github.com/JN0V/workline/releases/download/v1.2.3/*) ;; *) exit 22 ;; esac
f="` + served + `/${url##*/}"; [ -f "$f" ] || exit 22
if [ -n "$out" ]; then cp "$f" .; else cat "$f"; fi
`
		if err := os.WriteFile(filepath.Join(dir, "fake", "curl"), []byte(curl), 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "path")
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", run)
		cmd.Env = []string{"PATH=" + filepath.Join(dir, "fake") + ":" + os.Getenv("PATH"), "WORKLINE_VERSION=v1.2.3", "RUNNER_TEMP=" + dir, "GITHUB_PATH=" + path}
		out, err := cmd.CombinedOutput()
		added, _ := os.ReadFile(path)
		ran := err == nil && slices.Contains(strings.Fields(string(added)), filepath.Join(dir, "workline-release"))
		if ran != c.runs {
			t.Errorf("%s: runs %v (%v), want %v\n%s", c.name, ran, err, c.runs, out)
		}
		if !c.runs && len(added) > 0 {
			t.Errorf("%s: refused, the release put on PATH still:\n%s", c.name, out)
		}
	}
}

// semverLess says whether release a (v1.2.3) comes before b.
func semverLess(a, b string) bool {
	num := func(v string) [3]int {
		var n [3]int
		for i, p := range strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3) {
			n[i], _ = strconv.Atoi(p)
		}
		return n
	}
	x, y := num(a), num(b)
	return slices.Compare(x[:], y[:]) < 0
}
