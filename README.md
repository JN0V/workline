# workline

A software factory for AI-assisted development: each role — committer, release
manager, documentalist… — does one job with only the context it needs, tools do
the mechanical work, and an AI is called only when a decision needs judgement.
Everything keeps working without AI.

Status (2026-09-24): used daily on its author's machine; 38 conformance cases
green in CI.

| Works | Not yet |
|---|---|
| **Committer**: checks every commit (global git hook) and every commit of a merge request; Claude rewrites refused messages | |
| **Release manager**: semver, calver, several packages in one repository, generated changelog, tag, forge release | the merge-request flow, build metadata |
| **Documentalist**: finds docs whose sources changed (code, sections, other repositories); cuts cascades | size budgets, duplicates, dead links, AI judgement of suspect docs |
| **Gates**, **routing** and handoffs, **work items** (local or forge issues) | |
| **Forges**: GitHub (tried live), simulated; GitLab written | GitLab never run |
| Agents: Claude Code | Codex, Antigravity, OpenCode; the generated model grid |

## Try it

```sh
go build -o ~/.local/bin/workline ./cmd/workline

workline hooks install --global     # check every commit message on this machine
workline hooks uninstall --global   # give core.hooksPath back as it was
```

Global hooks hand over to whatever held `core.hooksPath` before, then to each
repository's own hooks; nothing that ran before stops running. A repository
opts out with an empty `.workline/off` file.

By default no AI is used: a refused message is explained, and you rewrite it.
To let Claude Code rewrite it, add `ai: claude` to `~/.config/workline/config.yaml`
(your default) or to a project's `.workline/config.yaml` (which wins).

## Develop

```sh
go test -count=1 ./...    # unit tests and the conformance suite
```

`-count=1` matters: the conformance suite builds the engine itself, which Go's
test cache does not see.

## Read next

- [Principles](docs/PRINCIPLES.md) — the rules every choice is checked against
- [Role contract](docs/spec/role-contract.md) — what a role is
- [Decisions](docs/adr/) · [Research](docs/research/) · [Backlog](docs/BACKLOG.md)
