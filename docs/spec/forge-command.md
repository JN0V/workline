---
sources: [internal/forge/command.go, internal/forge/forge.go, ci/forgejo]
checked: 0560057
judged: bca29fb
verified: agent:documentalist
---
# A forge plugged by a command — v1 (draft)

workline writes where the project lives (ADR-0016): GitHub and GitLab it
speaks natively, a project with none keeps its writes in the clone
(`forge: local`). Any other forge — Gitea, Forgejo, Bitbucket, one of your
own — is plugged by a command, the way a `cmd:` agent is:

```yaml
# .workline/config.yaml
forge: 'cmd:sh ci/forgejo/workline-forge.sh'
```

or `--forge 'cmd:<command>'` on a command line. A sample for Forgejo and
Gitea, whose API is GitHub's shape, is
[ci/forgejo/workline-forge.sh](../../ci/forgejo/workline-forge.sh).

## One request, one run

Each operation the engine needs runs the command once, through `sh -c`, **in
the repository** (a forge CLI reads the project from its remote), with:

- on its input, one JSON object: `operation` and that operation's arguments;
- `WORKLINE_FORGE_OPERATION` naming the operation, for a script that
  dispatches before reading its input;
- the engine's environment, tokens included: the command holds them, as
  `gh` or `glab` would.

It answers on its output with one JSON object; nothing at all is read as
`{}`.

| The command | Read as |
|---|---|
| exits 0 with an answer | done; the answer's fields below |
| exits 0 with `{"error": "…"}` | the forge refused: the run blocks with those words |
| exits other than 0, or answers no JSON | the forge unreachable: `blocked-external`, its last line said; the run resumes with `workline apply <run>` |
| no answer within 2 minutes | the forge unreachable |

## Operations

Each write is **idempotent**: the engine may run it again after a run
stopped half-way, and the forge must end the same. A marker is hidden text
(`<!-- workline:… -->`) the comment carries; finding it again is how a
comment is not posted twice. A target is `{"kind": "issue" | "merge-request",
"id": <n>}`.

| `operation` | Arguments | Does | Answers |
|---|---|---|---|
| `issue` | `id` | reads an issue | `{id, title, body, labels: [names]}` |
| `comment` | `target`, `body`, `marker` | posts `body` + a blank line + `marker` on the target, unless a comment there holds `marker` | `{}` |
| `sticky` | `target`, `body`, `marker`, `create` | edits the comment holding `marker` to `body` + `marker`; with none, posts it when `create` is true | `{}` |
| `label` | `target`, `add`, `remove` | adds the labels (creating them on the forge if needed), removes the others; one already there, or already gone, changes nothing | `{}` |
| `open-issue` | `title`, `body`, `marker` | on the open issue with this title, comments as `comment` does; with none, opens one, its body `body` + `marker` | `{id}` |
| `keep-issue` | `title`, `body`, `create` | rewrites the body of the open issue with this title; with none, opens it when `create` is true | `{id}`, 0 when none was opened |
| `release` | `tag`, `notes` | publishes the notes for the existing tag, unless a release of it is already there | `{}` |
| `open-merge-request` | `branch`, `base`, `title`, `body` | opens a merge request from `branch` (already pushed to `origin` by the engine) into `base`, or updates the title and body of the one open from `branch` | `{id}` |
| `open-merge-requests` | `prefix` | lists the open merge requests whose branch starts with `prefix` | `{branches: [names]}` |
| `merge-request-branch` | `id` | the branch a merge request comes from, and whether it lives in this repository rather than a fork | `{branch, here}` |

Arguments not listed are not sent; an operation the command does not know
answers `{"error": …}`. The engine keeps, of `open-merge-requests`, only the
branches under the prefix, sorted.

## What stays with the engine

Git is the engine's: it commits and pushes the branches of merge requests
(`--open-merge-request`, `--push-to-merge-request`, the weekly sample) to
`origin` itself; the command is asked only for what is the forge's — issues,
comments, labels, merge requests, releases. A command is not sandboxed: what
it does with the tokens it is given is its own promise, not the engine's.

## Tested

The conformance cases `forge/cmd-*` plug a fake script
(tests/conformance/fixtures/forges/logged.sh) that records each request: a
merge request's comment, a release, a work item read and labelled, the
weekly sample written, a forge failing (`blocked-external`) and refusing
(`block`). The Forgejo sample was run against a mock of the API only, not
yet on a live instance.
