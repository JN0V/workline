---
sources: [cmd/workline, internal/role/config.go, internal/engine/engine.go, internal/hooks]
checked: 533a14a
verified: agent:documentalist
---
# Using workline

The commands, the files and the variables, as the engine reads them today.
What a role does is in its own README (`roles/<name>/README.md`).

## Commands

| Command | Does |
|---|---|
| `workline run-role <role> --event <event>` | runs one role: prepare, propose, judge, apply, again if `pre` left work for later (`in/more`), up to 5 rounds |
| `workline route <event>` | runs the steps routing names for the event, in order; the first that does not pass stops the line |
| `workline apply <run-dir>...` or `--line <file>` | applies runs judged with `--no-apply`, or resumes a run stopped while applying; `--line` takes the runs a `route --no-apply --json` result lists as `pending` |
| `workline gate <name>` | runs a gate declared in `.workline/config.yaml` |
| `workline item ready <id>` | moves a work item to `ready`, once its Need, Verification, Validation and Scope are written (`--forge` reads it from the forge) |
| `workline hooks install --global` / `uninstall --global` | takes `core.hooksPath` for every repository, and gives it back as it was; the hooks run the `commit-msg` and `pre-push` lines, then hand over to the hooks that were there |
| `workline hooks install --repo` | writes `.githooks/commit-msg` in this repository; remove that file to uninstall |

Options of `run-role` and `route`:

| Option | |
|---|---|
| `--repo <dir>` | the repository (default: here) |
| `--ai <agent>` | `none`, `claude`, or `claude:<model>@<effort>` to force a model (an alias or an exact id), an effort, or both, whatever the role's tier asks: `claude:opus`, `claude:@high`; or `cmd:<command>`, any other agent (below); default: `WORKLINE_AI` for the hook, else the project's `ai`, else yours, else none |
| `--input name=value` | an input for the role (`route`: for every step), e.g. `range=<base>..<head>` for the committer on a merge request |
| `--input-file name=path` | `run-role` only: an input read from a file, written back by the intention that targets it (the hook's message file) |
| `--forge <forge>` | `github` (needs `gh`), `gitlab` (needs `glab`), `none`; default: the project's `forge` |
| `--target issue:<n>` / `merge-request:<n>` | where comments and labels go |
| `--scope <glob>` | a path the task is about, repeatable; a patch outside it is refused |
| `--no-apply` | judge, then stop; apply later, in a job that holds the write token |
| `--roles <dir>` | a folder of roles used instead of the shipped ones |
| `--sarif <file>` / `--code-quality <file>` | also write the findings as SARIF (GitHub code scanning) or a GitLab Code Quality report; `route`: every step's (docs/spec/role-outcome.md) |
| `--json` | print the result as JSON: status, summary, findings, each agent call (agent, tier, effort, the exact model that answered, tokens in, cached and out, cost, seconds), and for a line its steps and pending runs |

## Another agent

`cmd:<command>` makes any command the agent: another provider's CLI, or a
wrapper around your own. It runs with `sh -c`, outside the repository:

- its input is the prompt, the role's persona first;
- `WORKLINE_TIER` and `WORKLINE_EFFORT` say what the role asks (`light`,
  `standard`, `frontier`; `none` to `max`), for it to pick its model;
- its output is the answer: the YAML list of proposals, a code fence tolerated;
- it may write what answered, in YAML, to the file `WORKLINE_CALL` names:
  `model`, `tokens-in`, `tokens-cached`, `tokens-out`, `cost-usd`;
- a non-zero exit, or no answer within the role's `model.timeout`, ends the
  run as `blocked-external`, as a used-up quota does.

Claude Code, run this way (the command the judge's trial used):

```sh
--ai 'cmd:echo "model: haiku" > "$WORKLINE_CALL"; claude -p --model haiku --tools ""'
```

## Exit codes

| Code | Status |
|---|---|
| 0 | `pass` (findings may still be shown) |
| 1 | `block`, or an error |
| 2 | `human`: a person must decide |
| 3 | `blocked-external`: something outside failed (agent quota, forge, a repository) — never a verdict on the work |
| 64 | the command was misused |

`workline gate` returns 0 or 1. An unknown option exits 2, from Go's option
parser, which a script cannot tell from `human` (docs/BACKLOG.md).

## Before a push

The `pre-push` hook runs the project's `pre-push` line on the commits being
pushed, with the person's agent — only when `.workline/config.yaml` routes it,
since the global hook reaches every repository:

```yaml
routing:
  events: {pre-push: [committer, documentalist]}
```

A step that blocks stops the push. So does a doc the documentalist patched: it
is in the working tree, to review, commit, and push again. Sizes and links the
line reports on every run are only counted. `git push --no-verify` skips the
hook.

## Files

| File | Holds |
|---|---|
| `.workline/config.yaml` | the project's settings, below |
| `.workline/roles/<role>/<facet>` | a facet replacing the shipped one (`policy.md`, `instruction.md`…) |
| `.workline/work/<id>.md` | a work item, without a forge |
| `.workline/issues/` | the issues roles open, without a forge |
| `.workline/off` | empty: the global hook skips this repository |
| `~/.config/workline/config.yaml` | `ai:` — your default agent, when a project does not say |
| `~/.cache/workline/models-seen.yaml` | the last model that answered each alias on this machine: when another one answers, a run reports `model-changed` once, without blocking (ADR-0004) |
| `~/.config/workline/roles/<role>/<facet>` | your own facets, used when the project has none |

Runs are kept in `.git/workline/runs/` (the last
<!-- workline:derive runs-kept -->50<!-- workline:end -->), never in the working tree.

## `.workline/config.yaml`

```yaml
ai: claude                  # the project's agent; none by default
forge: github               # github | gitlab | none
roles:
  committer:
    settings: {subject-max: 60}          # a role's settings, see its role.yaml
    enforce: {internal-code: warn}       # block (default) | warn | off, per rule
  documentalist:
    settings:
      derive: {cases: "ls tests/*.yaml | wc -l"}   # fills <!-- workline:derive cases -->…<!-- workline:end --> in a doc
routing:                    # replaces the shipped line, event by event
  events: {merge-request: [committer, documentalist, gate:merge]}
  handoffs: [{from: release-manager, to: documentalist}]
  max-handoffs: 3
gates:                      # docs/spec/gates.md
  merge:
    checks: [{id: secrets, run: "gitleaks detect --report-format sarif --report-path {out}/secrets.sarif", output: sarif, max: {error: 0}}]
repos:                      # other repositories docs may depend on (docs/spec/multi-repo.md)
  api: {url: "https://github.com/acme/api.git", branch: main}
```

A key the engine does not know blocks, with its line: an ignored setting is one
someone believes in and nothing applies. A project's `settings` for a role
replace the role's, key by key, one level deep.

## Environment

| Variable | |
|---|---|
| `WORKLINE_AI` | the agent the git hook uses |
| `WORKLINE_ROLES` | a folder of roles, as `--roles` |
| `WORKLINE_MODELS_SEEN` | the file keeping the last model that answered each model asked (default: `models-seen.yaml` in your cache folder, `~/.cache/workline/` on Linux) |
| `WORKLINE_RUNS_DIR` | where runs are kept, e.g. a folder a CI artifact carries to the job that applies |

A role's `pre` and `post` receive `WORKLINE_RUN_DIR`, `WORKLINE_EVENT`,
`WORKLINE_AI`, `WORKLINE_ROLE` and `WORKLINE_BIN`. A role run by a handoff
receives the inputs `handoff-from` and `handoff-reason`.
