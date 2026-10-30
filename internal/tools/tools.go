// Package tools knows the programs workline calls but does not ship — the
// optional tools a role `uses`, the agents, the forges' CLIs — and how each
// one is installed, so a missing one is named with the command that installs
// it on this machine.
package tools

import (
	"os/exec"
	"runtime"
	"strings"
)

// Tool is a program workline may call.
type Tool struct {
	Name string
	// For says who needs it and what goes unchecked without it.
	For string
	// Local: what runs on a machine uses it (a commit, a push); otherwise it
	// serves what usually runs on a forge, gardening, and is offered only.
	Local bool
	// Ways to install it, most fitting first; the first whose program is on
	// this machine is the one given.
	Ways []Way
	Page string // where it is documented, when no way fits
}

// Way is one command installing a tool, and the program it needs.
type Way struct {
	Needs   string // a program on PATH: brew, go, cargo, npm, curl
	OS      string // runtime.GOOS it is for; empty: any
	Command string
}

var known = map[string]Tool{
	"gitleaks": {Name: "gitleaks", For: "the committer: secrets and forbidden terms", Local: true,
		Page: "https://github.com/gitleaks/gitleaks/releases",
		Ways: []Way{
			{Needs: "brew", Command: "brew install gitleaks"},
			// Not in its README, but tried: workline is installed with Go, so
			// this way is nearly always there.
			{Needs: "go", Command: "go install github.com/zricethezav/gitleaks/v8@latest"},
		}},
	"lychee": {Name: "lychee", For: "the documentalist: links to other sites, when gardening", Local: true,
		Page: "https://lychee.cli.rs/installation/",
		Ways: []Way{
			{Needs: "brew", Command: "brew install lychee"},
			{Needs: "cargo", Command: "cargo install lychee"},
			// No brew, no cargo: the pinned release binary, its checksum
			// checked, put in ~/.local/bin. WORKLINE_LYCHEE_RELEASE_URL
			// replaces the download site (tests use a fake one).
			{Needs: "curl", OS: "linux", Command: lycheeRelease},
		}},
	"claude": {Name: "claude", For: "the agent `claude`: refused messages rewritten, suspect docs judged", Local: true,
		Page: "https://docs.claude.com/en/docs/claude-code/setup",
		Ways: []Way{
			// The installer the setup page recommends; it keeps itself up to date.
			{Needs: "curl", OS: "linux", Command: "curl -fsSL https://claude.ai/install.sh | bash"},
			{Needs: "curl", OS: "darwin", Command: "curl -fsSL https://claude.ai/install.sh | bash"},
			{Needs: "brew", Command: "brew install --cask claude-code"},
			{Needs: "npm", Command: "npm install -g @anthropic-ai/claude-code"},
		}},
	"gh": {Name: "gh", For: "the forge `github`: comments, labels, issues, merge requests",
		Page: "https://cli.github.com",
		Ways: []Way{{Needs: "brew", Command: "brew install gh"}}},
}

// lycheeRelease downloads lychee's pinned release, checks its sha256 and
// installs it in ~/.local/bin.
const lycheeRelease = `set -e; b="${WORKLINE_LYCHEE_RELEASE_URL:-https://github.com/lycheeverse/lychee/releases/download}"; ` +
	`a="lychee-$(uname -m)-unknown-linux-gnu.tar.gz"; u="$b/lychee-v0.21.0"; d="$(mktemp -d)"; ` +
	`curl -fsSL -o "$d/$a" "$u/$a"; curl -fsSL -o "$d/$a.sha256" "$u/$a.sha256"; ` +
	`(cd "$d" && echo "$(cut -d' ' -f1 "$a.sha256")  $a" | sha256sum -c -); ` +
	`tar -xzf "$d/$a" -C "$d"; mkdir -p "$HOME/.local/bin"; ` +
	`install -m 755 "$d/lychee" "$HOME/.local/bin/lychee"; rm -rf "$d"`

// Lookup returns what workline knows of a tool; an unknown one has its name only.
func Lookup(name string) Tool {
	if t, ok := known[name]; ok {
		return t
	}
	return Tool{Name: name}
}

// Present says whether the tool is on PATH.
func (t Tool) Present() bool {
	_, err := exec.LookPath(t.Name)
	return err == nil
}

// Install returns the command that installs the tool on this machine, or,
// when none fits, where to read how.
func (t Tool) Install() string {
	for _, w := range t.Ways {
		if w.OS != "" && w.OS != runtime.GOOS {
			continue
		}
		if _, err := exec.LookPath(w.Needs); err == nil {
			return w.Command
		}
	}
	if t.Page != "" {
		return "see " + t.Page
	}
	return "install " + t.Name + " and put it on your PATH"
}

// Command says whether an install answer is a command to run, rather than a
// page to read.
func Command(install string) bool {
	return !strings.HasPrefix(install, "see ") && !strings.HasPrefix(install, "install ")
}
