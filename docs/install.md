---
sources:
  - .goreleaser.yaml#builds
  - .goreleaser.yaml#archives
  - go.mod
  - cmd/workline/cli.go#topHelp
  - cmd/workline/cli.go#versionCmd
  - cmd/workline/main.go#setupCmd
  - cmd/workline/main.go#doctorCmd
  - internal/setup/setup.go#Run
  - internal/setup/setup.go#Shell
  - internal/tools/tools.go#known
  - internal/agent/claude.go#claudeSession
  - internal/agent/claude.go#claudeEnv
checked: 542fc9cb6
verified: agent:claude-code
---
# Install

workline runs in two places, each on its own or both:

- **your machine**: every commit checked as you write it, a branch read
  before you push it;
- **CI**: every merge request judged, gardening and the weekly sample on a
  schedule.

You need git. The agent is a choice: without one, every check on facts
still runs, and each role says what it skipped
([principle 5](PRINCIPLES.md)).

## On your machine

### 1. The binary

Each [release](https://github.com/JN0V/workline/releases) holds the binary
for Linux, macOS and Windows. On Linux or macOS, into `~/.local/bin`:

```sh
os=$(uname -s | tr A-Z a-z); arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
curl -fsSL "https://github.com/JN0V/workline/releases/latest/download/workline_${os}_${arch}.tar.gz" | tar -xz -C ~/.local/bin workline
workline version
```

Or, with [Go](https://go.dev/dl/) 1.27.1 or later (into `$(go env GOPATH)/bin`,
usually `~/go/bin`: add it to your `PATH`):

```sh
go install github.com/JN0V/workline/cmd/workline@latest
```

To update, run the same command again, into the same place
([why](install-hooks.md#update-in-place)).

### 2. Set the machine up

```sh
workline setup     # asks what to enable, shows each command before it runs it
workline doctor    # at any time: what is set up, what is missing, the command for each
workline --help    # every command; `workline <command> --help`, its options
```

`setup` asks:

1. whether to check every commit on this machine (the global hooks, below);
2. which agent judges (below);
3. whether to install the tools the roles use and are missing: gitleaks
   (the committer's secrets check), lychee (links to other sites, when
   gardening), and the agent's CLI if you chose `claude`;
4. in a GitLab clone, whether to create the webhook and trigger token
   that start the line on an issue's label
   ([how](gitlab/line-of-states.md)).

Run it again to change an answer; `--yes` takes the defaults; `--hooks`,
`--ai`, `--install` and `--issue-hook` answer without a terminal.

### 3. The agent: your choice

| Answer | What judges |
|---|---|
| `none` | no AI: the checks still run, each role says what it skipped |
| `claude` | [Claude Code](https://docs.claude.com/en/docs/claude-code/setup): `setup` offers to install its CLI; run `claude` once to log in (below) |
| `cmd:<command>` | any command: another provider's CLI, or a wrapper of your own ([the contract](usage.md#another-agent)) |

- Claude logs in with a subscription or an API key; workline calls
  `claude` with that login and holds no key of its own.
- Other agents built in (Codex, OpenCode…) are parked until a project needs
  one; `cmd:` runs any of them meanwhile
  ([another agent](usage.md#another-agent)).

- The answer goes to `ai:` in `~/.config/workline/config.yaml` (macOS:
  `~/Library/Application Support/workline/config.yaml`), for all your
  repositories.
- Which answer wins, the project's, yours or `WORKLINE_AI` for one
  commit: [which agent answers](config.md).

### 4. Every commit checked

The global git hooks, installed by `setup` or by hand, and what a commit
refused looks like: [every commit checked](install-hooks.md).

### 5. Adopt a repository

`workline init` in a repository, then the product owner on one issue:
[adopt a repository](install-repo.md).

## In CI

The workflows to copy or the pipeline to include, and what they need:
[install in CI](install-ci.md). What the jobs are: [ci.md](ci.md).
