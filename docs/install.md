---
sources: [cmd/workline, internal/setup, internal/hooks, internal/tools, internal/agent/agent.go, ci/github, ci/gitlab]
checked: bd66bb99
judged: a8574a5
verified: agent:claude-code
---
# Install

workline runs in two places, each on its own or both:

- **your machine**: every commit checked as you write it, a branch read
  before you push it;
- **CI**: every merge request judged, gardening and the weekly sample on a
  schedule.

You need git. No AI is needed for any step: without an agent every check
still runs, and what needs judgement is listed for a person.

## On your machine

### 1. The binary

Each [release](https://github.com/JN0V/workline/releases) holds the binary
for Linux, macOS and Windows. On Linux or macOS, into `~/.local/bin`:

```sh
os=$(uname -s | tr A-Z a-z); arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
curl -fsSL "https://github.com/JN0V/workline/releases/latest/download/workline_${os}_${arch}.tar.gz" | tar -xz -C ~/.local/bin workline
workline version
```

Or, with [Go](https://go.dev/dl/) 1.21 or later (into `$(go env GOPATH)/bin`,
usually `~/go/bin`: add it to your `PATH`):

```sh
go install github.com/JN0V/workline/cmd/workline@latest
```

To update, run the same command again, into the same place: the global
hooks call the binary where it was when they were installed (`command -v
workline` says where).

### 2. Set the machine up

```sh
workline setup     # asks three questions, shows each command before it runs it
workline doctor    # at any time: what is set up, what is missing, the command for each
workline --help    # every command; `workline <command> --help`, its options
```

`setup` asks:

1. whether to check every commit on this machine (the global hooks, below);
2. which agent judges (below);
3. whether to install the tools the roles use and are missing: gitleaks
   (the committer's secrets check), lychee (links to other sites, when
   gardening), and the agent's CLI if you chose `claude`.

Run it again to change an answer; `--yes` takes the defaults; `--hooks`,
`--ai` and `--install` answer without a terminal.

### 3. The agent: your choice

| Answer | What judges |
|---|---|
| `none` | no AI: a refused message is explained, you rewrite it; suspect docs are listed for you |
| `claude` | [Claude Code](https://docs.claude.com/en/docs/claude-code/setup): `setup` offers to install its CLI; run `claude` once to log in (below) |
| `cmd:<command>` | any command: another provider's CLI, or a wrapper of your own ([the contract](usage.md#another-agent)) |

Claude logs in with a subscription or an API key; workline calls `claude -p`
with that login and holds no key of its own.

Other agents built in (Codex, OpenCode…) are parked until a project needs
one; `cmd:` runs any of them meanwhile
([#86](https://github.com/JN0V/workline/issues/86)).

- The answer goes to `ai:` in `~/.config/workline/config.yaml` (macOS:
  `~/Library/Application Support/workline/config.yaml`), for all your
  repositories.
- A project's own `.workline/config.yaml` wins over yours.
- `WORKLINE_AI=none git commit …` turns the agent off for one commit.

### 4. Every commit checked

`workline setup` does this when you say yes; by hand:

```sh
workline hooks install --global     # git's global core.hooksPath now goes through workline
workline hooks uninstall --global   # gives core.hooksPath back as it was
```

- Each global hook hands over to the one that held `core.hooksPath` before,
  or else to the repository's own (`.githooks/`, `.git/hooks/`), from a
  linked worktree (`git worktree add`), a submodule or a bare repository
  too: nothing that ran before stops running.
- Hooks written by another version of workline: `workline doctor` says
  `hooks-outdated`; run `workline hooks install --global` again.
- A repository that sets its own `core.hooksPath` bypasses the global
  hooks: `workline doctor` there says so and prints the fix
  ([troubleshooting](troubleshooting.md#a-hook-does-not-fire)).
- A repository opts out with an empty `.workline/off` file.

Try it in any repository:

```sh
git commit --allow-empty -m "AC-3 fix the thing"
# workline committer: block — rewrite the commit message
#   format subject: the subject must read `type(scope): summary`…
# workline: 0.6 s in all — committer 0.5 s
# commit aborted; your message is kept in .git/COMMIT_EDITMSG: to edit it and commit again, git commit -e -F .git/COMMIT_EDITMSG
```

- **With no agent**, or `rewrite: false` under
  `roles: {committer: {settings: …}}` (the project's
  `.workline/config.yaml`, or your own config), it refuses in about a
  second, as above.
- **With an agent**, the same commit goes through, rewritten, 30 s at most
  ([the committer](../roles/committer/README.md#rewrite-the-agent-30-s-at-most),
  [ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md)):

```text
workline committer: the message is refused; the agent (claude) rewrites it, 30 s at most…
workline committer: pass — message rewritten
workline: 14 s in all — committer 14 s (rewrite 13 s)
workline: the commit goes on with this message instead of yours:
  │ fix: fix the thing
  │
  │ Refs: AC-3
```

### 5. Adopt a repository

```sh
cd your-repo
workline init      # routes pre-push; proposes each doc's sources, for you to review and commit
```

- From then on, each push runs the committer and the documentalist on the
  commits leaving, with no agent: one line counts the docs they made
  suspect.
- Until a doc names its `sources`, nothing tells when it goes wrong.
- `--review` puts the reviewer on each merge request too; `workline review`
  reads a branch before you push it.
- `--human-po yes` says a person is the project's Product Owner: the
  product owner role then proposes what sets direction rather than doing it
  ([ADR-0038](adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).

Step by step, in ten minutes: [the quickstart](quickstart.md).

## In CI

What the jobs are: [ci.md](ci.md); every detail on your forge's page. In short:

- **GitHub**: copy the workflows of [ci/github](../ci/github/) into
  `.github/workflows/` — [workline.yml](../ci/github/workline.yml) (each
  pull request, and main after each merge), [workline-warn.yml](../ci/github/workline-warn.yml) (what
  only warns, beside it: the reviewer's lenses that warn, the inspector's
  triage), [workline-apply.yml](../ci/github/workline-apply.yml)
  (what it proposed applied, main's findings in one issue, and each push
  to main for `workline follow`), [workline-fork.yml](../ci/github/workline-fork.yml) (a
  fork's), [workline-gardening.yml](../ci/github/workline-gardening.yml),
  [workline-sample.yml](../ci/github/workline-sample.yml),
  [workline-objection.yml](../ci/github/workline-objection.yml) and
  [workline-answer.yml](../ci/github/workline-answer.yml) (with the
  [tester](../roles/tester/README.md#when-it-runs)),
  [workline-develop.yml](../ci/github/workline-develop.yml) (with the
  [developer](../roles/developer/docs/triggers.md)),
  [workline-merge.yml](../ci/github/workline-merge.yml) (with the
  [merge policy](merge-policy.md)),
  [workline-release.yml](../ci/github/workline-release.yml) (with a
  release tool: [what holds a release](release-holds.md))
  ([ci-github.md](ci-github.md)).
- **GitLab**: include
  [ci/gitlab/workline.gitlab-ci.yml](../ci/gitlab/workline.gitlab-ci.yml),
  then add two pipeline schedules, gardening and the sample
  ([ci-gitlab.md](ci-gitlab.md#gitlabcom); a [self-managed
  instance](ci-gitlab-self-managed.md)).
- **Another forge** (Gitea, Forgejo…) or [none](ci-other-forges.md#no-forge): [ci-other-forges.md](ci-other-forges.md).
- **A board**, GitHub's project or GitLab's: what to switch on so the
  roles move its cards, [boards](boards.md#what-you-switch-on).

What the templates need:

| | GitHub | GitLab |
|---|---|---|
| the agent, your choice | its credentials; for Claude, which the templates install: secret `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`) | the same as a variable, masked, not protected, scoped to the environment `workline` ([why](gitlab/environment.md)) |
| writing | a [GitHub App](ci-github-app.md): variable `WORKLINE_APP_ID`, secret `WORKLINE_APP_PRIVATE_KEY` in the environment `workline-app` | variable `WORKLINE_GITLAB_TOKEN`: a project access token, `api` and `write_repository`, scoped the same |
| the history | `fetch-depth: 0`, set | `GIT_DEPTH: 0`, set |
| the sample's judge | variable `WORKLINE_JUDGE` (optional) | the same, and `WORKLINE_TASK=sample` on its schedule |

- Without the agent's token, the templates run with no agent: every check
  still runs. For another agent, change their `--ai` to a `cmd:`.
- The whole history is needed: docs are compared since their `checked`,
  the release since its tag. A job cloning too shallow reports
  `shallow-clone`, never a pass.
- The templates run the release of workline named by `WORKLINE_VERSION` at
  their top: update it to take a newer one.
- A pipeline started by a tool rather than by the forge (GitLab's trigger
  API, another pipeline): [gitlab-trigger.md](gitlab-trigger.md). The bare
  command for each event, for any other trigger: [triggers.md](triggers.md).
