---
sources: [cmd/workline, internal/role/config.go, internal/engine/engine.go, internal/hooks]
checked: 6013a50
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
| `workline setup` | sets up this machine, asking: the global hooks, your agent (`ai:` in your config), the tools the roles use, each installed with the command it shows; then prints `workline doctor`. Run again, it offers what is set up as the default. `--hooks yes\|no`, `--ai <agent>` and `--install <tool,...>\|all\|none` answer a question; `--yes` takes the defaults; without a terminal, every question must be answered so |
| `workline init` | adopts this repository: routes `pre-push` to the committer and the documentalist in `.workline/config.yaml`, unless the project routes it already, and has the agent propose the `sources` of each doc that says nothing of them, in the working tree for you to review and commit (roles/documentalist/README.md); without an agent, lists them. Run it again to do what is left |
| `workline docs` | has the docs your commits not pushed yet made suspect judged by your agent, then asks you, doc by doc, to keep or drop each change (`v` shows it); those kept go in one `docs:` commit, alone (ADR-0007). Without a terminal, the changes stay in the working tree for a person, who reviews them the same way with `workline docs --review`. Docs with changes not committed are refused, so the agent's are reviewed alone |
| `workline doctor` | says what is set up on this machine (git, the global hooks, the agent, the tools the roles use) and in this repository (whether the documentalist runs before a push, how many docs declare their sources), with the command that sets up each thing missing; changes nothing. It exits 1 only on an error — the agent named cannot be called, the config does not load — never for a tool left out; `--json` prints every check |

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
| `--push-to-merge-request` | with a forge and `--target merge-request:<n>`: what the patches write is committed to that merge request's branch and pushed, without force; from a fork, or when the branch moved on, the diff goes in one comment instead |
| `--open-merge-request` | with a forge: what the patches write goes on a branch `workline/<role>/<task>`, pushed, with a merge request opened or updated for it; the role is told how many of its merge requests are open (ADR-0006) |
| `--roles <dir>` | a folder of roles used instead of the shipped ones |
| `--sarif <file>` / `--code-quality <file>` | also write the findings as SARIF (GitHub code scanning) or a GitLab Code Quality report; `route`: every step's (docs/spec/role-outcome.md) |
| `--json` | print the result as JSON: status, summary, findings, the agent's notes, each agent call (agent, tier, effort, the exact model that answered, tokens in, cached and out, cost, seconds), and for a line its steps and pending runs |

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

**A person approves every push.** Once the line below has passed, the hook
lists the commits leaving, and the docs they made suspect and nobody judged
yet, and asks, on the terminal, `Push? [y]es / [N]o / [v]iew / [d]ocs`; `d`
is `workline docs` there and then — once the docs are committed, the push
stops and you push again, since it cannot carry a commit made during it; `v` writes a page showing each commit, its message and its changes
(`.git/workline/push.html`) and opens it with the system's default program.
No answer in ten minutes is a no. Without a terminal — an agent, an editor's
button — the push is refused, saying so. It holds in every repository the
global hooks reach, `.workline/off` too; only your own config turns it off,
`approve-push: false`, which no project can (ADR-0007). `git push
--no-verify` still skips it: it stops an agent pushing by habit, not one set
on it.

The `pre-push` hook runs the project's `pre-push` line on the commits being
pushed, with no agent — a push never waits on one — only when
`.workline/config.yaml` routes it, since the global hook reaches every
repository:

```yaml
routing:
  events: {pre-push: [committer, documentalist]}
```

A step that blocks stops the push. The docs the commits made suspect are
listed with the question, to judge with `d`; those made suspect before are
left for gardening. A derived block the documentalist regenerated stops the
push too: it is in the working tree, to review, commit, and push again. Sizes
and links the line reports on every run are only counted. `git push
--no-verify` skips the hook.

## Files

| File | Holds |
|---|---|
| `.workline/config.yaml` | the project's settings, below |
| `.workline/roles/<role>/<facet>` | a facet replacing the shipped one (`policy.md`, `instruction.md`…) |
| `.workline/work/<id>.md` | a work item, without a forge |
| `.workline/issues/` | the issues roles open, without a forge |
| `.workline/off` | empty: the global hook skips this repository |
| `~/.config/workline/config.yaml` | yours: `ai:`, your default agent when a project does not say; `approve-push: false` lets pushes leave without you approving them |
| `~/.cache/workline/models-seen.yaml` | the last model that answered each alias on this machine: when another one answers, a run reports `model-changed` once, without blocking (ADR-0004) |
| `~/.config/workline/roles/<role>/<facet>` | your own facets, used when the project has none |

Runs are kept in `.git/workline/runs/` (the last
<!-- workline:derive runs-kept -->50<!-- workline:end -->), never in the working tree: each
agent call with what it cost in `out/calls.jsonl`, each refused answer in
`out/refused-<n>.yaml`.

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
`WORKLINE_AI`, `WORKLINE_ROLE` and `WORKLINE_BIN`, `WORKLINE_FORGE` when a
forge is given, `WORKLINE_TARGET` (`merge-request:12`) with a target too, and
`WORKLINE_OPEN_MERGE_REQUESTS` with `--open-merge-request`. A role run by a handoff
receives the inputs `handoff-from` and `handoff-reason`.
