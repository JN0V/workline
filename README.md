<!-- workline
sources: [cmd/workline, ci, routing.default.yaml, internal/forge, internal/agent/agent.go]
checked: fbbb1de
verified: agent:documentalist
-->
<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/logo-dark.svg">
    <img src="docs/assets/logo.svg" alt="workline" width="360">
  </picture>
</h1>

A software factory for AI-assisted development: each role — committer, release
manager, documentalist… — does one job with only the context it needs, tools do
the mechanical work, and an AI is called only when a decision needs judgement.
Everything keeps working without AI.

Status (2026-09-24): used daily on its author's machine;
<!-- workline:derive conformance-cases -->65<!-- workline:end --> conformance cases green in CI.

| Works | Not yet |
|---|---|
| **Committer**: checks every commit (global git hook) and every commit of a merge request; Claude rewrites refused messages | |
| **Release manager**: semver, calver, several packages in one repository, generated changelog, tag, forge release | the merge-request flow, build metadata |
| **Documentalist**: finds docs whose sources changed (code, sections, other repositories); cuts cascades; size budgets, duplicates, dead links inside the repository, identifiers gone from the code; Claude judges suspect docs, and its patches are checked | links to other sites, freshness, derived blocks, condensing and splitting by AI |
| **Gates**, **routing** and handoffs, on a machine or judged on a forge and applied later | |
| **Work items** (local files or forge issues): the check that moves one to `ready` | the rest of the item's life |
| **Forges**: GitHub (comments and labels tried live), simulated; GitLab written | GitLab never run; GitHub issues and releases never run live |
| Agents: Claude Code | Codex, Antigravity, OpenCode; the generated model grid |

## Install

You need git and [Go](https://go.dev/dl/) 1.21 or later (Go fetches the
version workline needs by itself). There are no released binaries yet.

```sh
go install github.com/JN0V/workline/cmd/workline@latest
```

The binary goes to `$(go env GOPATH)/bin`, usually `~/go/bin`, which Go does
not put on your `PATH`. If `workline` is not found, add it (`~/.zshrc` for zsh):

```sh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc && source ~/.bashrc
```

Run the same `go install` again to update.

### Check every commit on this machine

```sh
workline hooks install --global     # git's global core.hooksPath now goes through workline
workline hooks uninstall --global   # gives core.hooksPath back as it was
```

Each global hook hands over to the one that held `core.hooksPath` before, or,
when there was none for that hook, to the repository's own (`.githooks/`,
`.git/hooks/`): nothing that ran before stops running. A repository opts out
with an empty `.workline/off` file.

Try it in any repository:

```sh
git commit --allow-empty -m "AC-3 fix the thing"
# workline committer: block — rewrite the commit message
#   format subject: the subject must read `type(scope): summary`…
```

### Let an AI rewrite what is refused (optional)

By default no AI is used: a refused message is explained, and you rewrite it.
To let an AI rewrite it:

1. Install [Claude Code](https://docs.claude.com/en/docs/claude-code/setup),
   run `claude` once and log in (a Claude subscription or an API key). workline
   calls `claude -p` with that login: it holds no key of its own.
2. Tell workline to use it, for all your repositories:

   ```sh
   mkdir -p ~/.config/workline
   echo 'ai: claude' >> ~/.config/workline/config.yaml
   ```

   On macOS the file is `~/Library/Application Support/workline/config.yaml`.
   A project's own `.workline/config.yaml` wins over yours, and
   `WORKLINE_AI=none git commit …` turns the AI off for one commit.

The same commit now goes through, rewritten:

```text
workline committer: pass — message rewritten
workline: the commit goes on with this message instead of yours:
  │ fix: fix the thing
  │
  │ Refs: AC-3
```

Claude Code is the only agent so far.

## Where it runs

On your machine, on a forge, or both: each place fires its own events, and one
routing (`routing.default.yaml`, changed in `.workline/config.yaml`) says which
roles each event runs. A role behaves the same wherever it runs; only the
trigger, the agent at hand and the way proposals are applied differ.

```mermaid
flowchart LR
  routing[("one routing<br/>.workline/config.yaml")]
  subgraph machine["Your machine"]
    commit["git commit"] -- commit-msg --> committer1["committer"]
    push["git push"] -- pre-push --> local["the project's pre-push line"]
  end
  subgraph forge["Forge: GitHub or GitLab"]
    mr["merge request"] -- merge-request --> judge["judge job<br/>committer, documentalist<br/>no write token"]
    judge -- proposals --> apply["apply job<br/>no AI key"]
    schedule["schedule<br/>a pipeline you add"] -- schedule --> documentalist["documentalist"]
  end
  machine -- git push --> forge
  routing -.-> committer1
  routing -.-> local
  routing -.-> judge
  routing -.-> documentalist
```

| Event | Fired by | Roles by default |
|---|---|---|
| `commit-msg` | your machine: the global git hook | committer |
| `pre-push` | your machine, before the commits leave it, if the project routes it | none by default; workline itself: committer, documentalist |
| `merge-request` | the forge: [GitHub Actions](ci/github/workline.yml) or [GitLab CI](ci/gitlab/workline.gitlab-ci.yml) template | committer, documentalist |
| `schedule` | you, or a scheduled pipeline you add (the templates have none yet) | documentalist |
| `release` | wherever you run `workline route release`: it tags and publishes at once (`flow: direct`) | release-manager |

So the committer checks your messages as you write them, and again on the merge
request for those without the hook; the documentalist runs on the forge. On a
forge, one job judges without a write token and another applies without an AI
key (`--no-apply`, then `workline apply`).

The CI templates run `workline route merge-request`, so a project's `routing:`
reaches its CI too.

## Take only a part

workline is one binary with its roles inside, and needs nothing but git (and
`gh` or `glab` to reach a forge). Any existing pipeline can call one role, and
leave the rest:

```sh
workline run-role committer --event merge-request --ai none \
  --input range=origin/main..HEAD --json    # exit code: 0 pass, 1 block, 2 human, 3 external
workline run-role documentalist --event schedule --ai none --json
workline gate release --json   # a gate declared in .workline/config.yaml: your scanners' SARIF, with thresholds
```

- The exit code carries the verdict; `--json` gives the findings to whatever
  reads them.
- The global hook hands over to the hooks that were there before; nothing
  stops running.
- `--no-apply` and `workline apply` fit a pipeline that keeps tokens apart.
- A role is a folder (`role.yaml`, facets, `pre` and `post` in any language):
  a team can replace one facet (`.workline/roles/<role>/`), or run its own
  roles with `--roles <dir>` ([role contract](docs/spec/role-contract.md)).

Not there yet: verdicts as SARIF, for code-scanning and code-quality views;
an agent other than Claude Code built in (`--ai cmd:<command>` runs any).

## Develop

```sh
git clone https://github.com/JN0V/workline && cd workline
go build -o ~/.local/bin/workline ./cmd/workline   # or anywhere on your PATH
go test -count=1 ./...    # unit tests and the conformance suite
```

`-count=1` matters: the conformance suite builds the engine itself, which Go's
test cache does not see. The evaluation grades the roles with a real agent on
real cases, and costs tokens, so it runs only when asked:
`WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/`
(docs/spec/conformance.md, "Evaluation").

## Read next

- [Usage](docs/usage.md) — commands, options, exit codes, files, variables
- [Principles](docs/PRINCIPLES.md) — the rules every choice is checked against
- [Role contract](docs/spec/role-contract.md) — what a role is
- [Decisions](docs/adr/) · [Research](docs/research/) · [Backlog](docs/BACKLOG.md)
