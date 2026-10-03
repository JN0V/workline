// Package release says what a project's releases are, for the roles that
// hold them (ADR-0017): the branches a release tool opens its pull request
// from, and the last release, one lookup for every reader.
package release

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
	"go.yaml.in/yaml/v3"
)

// Branches are the head branches of the release pull requests the known
// tools open: release-please, releaser-pleaser, release-plz, changesets.
var Branches = []string{"release-please--*", "releaser-pleaser--*", "release-plz-*", "changeset-release/*"}

// Tags is the pattern of a release's tag when the project names none.
const Tags = "v*"

// Settings are the project's `release` key in .workline/config.yaml.
type Settings struct {
	Branches []string `yaml:"branches"` // replace the tools' when set
	Tags     string   `yaml:"tags"`
}

// Load reads the project's release settings, the defaults where it says
// nothing. The config was checked by whoever loaded it first.
func Load(repo string) (Settings, error) {
	var c struct {
		Release Settings `yaml:"release"`
	}
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if err != nil && !os.IsNotExist(err) {
		return c.Release, err
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return c.Release, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	if len(c.Release.Branches) == 0 {
		c.Release.Branches = Branches
	}
	if c.Release.Tags == "" {
		c.Release.Tags = Tags
	}
	return c.Release, nil
}

// IsBranch says whether a merge request from this branch is a release
// tool's: the release, held as one.
func (s Settings) IsBranch(branch string) bool {
	return branch != "" && pathglob.Any(s.Branches, branch)
}

// ErrShallow is a clone missing history: the last release may be in what
// was not fetched, and starting from no tag would take every doc ever
// suspect for the release's.
var ErrShallow = errors.New("this clone is shallow: the last release may be in the history it lacks; fetch it whole (actions/checkout's `fetch-depth: 0`, GitLab's `GIT_DEPTH: 0`)")

// Last is the last release HEAD holds: the highest version among the tags
// matching the pattern and merged into HEAD — not the nearest, which a
// hotfix merged back would be. A prerelease (v1.2.0-rc.1) and an alias (v1,
// latest) are not releases. "" when there is none.
func Last(repo, pattern string) (string, error) {
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--is-shallow-repository").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	if strings.TrimSpace(string(out)) == "true" {
		return "", ErrShallow
	}
	out, err = exec.Command("git", "-C", repo, "tag", "--merged", "HEAD", "--list", pattern).Output()
	if err != nil {
		return "", fmt.Errorf("git tag: %w", err)
	}
	prefix, _, _ := strings.Cut(pattern, "*")
	best, bestV := "", []int(nil)
	for _, tag := range strings.Fields(string(out)) {
		v := version(strings.TrimPrefix(tag, prefix))
		if v != nil && (bestV == nil || later(v, bestV)) {
			best, bestV = tag, v
		}
	}
	return best, nil
}

// version reads major.minor.patch, nil for anything else.
func version(s string) []int {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil
	}
	v := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || p != strconv.Itoa(n) {
			return nil
		}
		v[i] = n
	}
	return v
}

func later(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}
