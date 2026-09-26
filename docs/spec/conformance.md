---
sources: [tests/conformance/runner_test.go, tests/evaluation]
checked: fbbb1de
verified: agent:documentalist
---
# Conformance — v1 (draft)

The specs say what must happen; these tests prove it does. They are written
before the engine, and they belong to the contract, not to one engine: any
implementation that passes them is a valid engine (ADR-0001).

## Two levels

| Level | Agent | Runs | Answers |
|---|---|---|---|
| **Conformance** | a fake agent replaying recorded proposals | every change, in CI, free | Does the engine obey the contract? |
| **Evaluation** | a real agent | on demand, costs tokens | Does a role do its job well, with this model? |

Conformance never calls a model, so it is deterministic and costs nothing.
Evaluation grades a role on its own, on the same fixtures, with a real agent;
its score is tracked over time rather than passed or failed.

## Layout

```
tests/conformance/
  fixtures/repos/<name>.sh    builds a git repository from scratch, deterministically
  fixtures/agents/<name>.yaml proposals the fake agent returns
  cases/<area>/<case>.yaml    one behaviour each
```

Repositories are built by scripts rather than stored, because a git repository
cannot live inside another one, and because a script shows exactly what the
history contains.

## A case

```yaml
case: committer/internal-code-in-subject
about: A ticket code in the subject blocks the commit, even without AI.

given:
  repo: basic                       # fixtures/repos/basic.sh
  setup:                            # shell lines run in the repo after it is built
    - echo "change" >> main.go && git add main.go
  forge:                            # starting state of the simulated forge, if any
    issues: []
  config: {}                        # .workline/config.yaml for this case

run:
  role: committer                   # or: route: <event>, or: gate: <name>
  event: commit-msg
  input: {message: "fix: AC-3"}
  ai: none                          # none | fake:<fixture> | unavailable:<reason> | cmd:<command>

expect:
  status: block
  findings: [{rule: internal-code}]
  agent-calls: 0
  applied: []
```

`given` can also carry `models-seen`: the models this machine saw answer before
the run (`{"fake:light": {model: …}}`), in the file `WORKLINE_MODELS_SEEN` names.
It can also carry `repos`: other repositories to build first, each
exported by name as an environment variable holding its path, for
multi-repository cases.

`run` can also carry:

- `route: <event>` instead of `role` — run the event's whole line;
- `gate: <name>` instead of `role` — run one gate;
- `route: ready` with `item` — ask routing to move a work item;
- `target` — the issue or merge request comments and labels go on
  (`{merge-request: 1}`);
- `scope` — the run's scope, as a ready work item would give it;
- `no-apply: true` — judge, and stop before applying;
- `then: resume` — after the run, resume it with `workline apply`: the run, or
  every run a line judged and did not apply;
- `tamper: in/` — change the prepared input between prepare and apply.

`expect` lists only what the case is about; anything not listed is not checked.

- `findings` — each listed finding must be present (matched by `rule`, and by
  `where` if given).
- `agent-calls` — how many times the agent was called.
- `calls` — each call, in order: the fields listed must match (`agent`,
  `task`, `tier`, `effort`, `asked`, `model`).
- `applied` / `refused` — intentions applied or refused, by kind.
- `steps` — for a line, the steps that ran, in order.
- `forge` — fields the simulated forge must hold afterwards.
- `files` — paths that must exist, or contain a text, afterwards.

## The fakes

- **Fake agent.** `fake:<fixture>` returns the proposals in
  `fixtures/agents/<fixture>.yaml`, without reading the prompt. It checks the
  engine's handling of proposals, not their quality.
- **Unavailable agent.** `unavailable:quota` (or `auth`, `network`) fails the
  way a real agent does when a quota runs out.
- **Simulated forge.** An in-memory forge holding issues, labels, comments,
  merge requests and releases. It can be told to fail on the N-th write, to
  test recovery after a partial apply.

## Evaluation

An evaluation case uses the same `given` and `run`, with a real agent, and adds
`grade`: what a good answer contains ("the message names the changed
behaviour", "the product doc keeps every MUST"). Grading is done by checks where
possible, and by a judge from another provider where not (`independent-of` in
the model grid).

```
tests/evaluation/
  cases/<role>/<case>.yaml   one real situation each
  results.tsv                every run's score, appended: the history kept
  schedule/                  run.sh and systemd units: a run every week
  summary/                   go run ./tests/evaluation/summary: per case and models,
                             the runs, mean score, range, tokens and seconds; a
                             model that no longer answers is marked `replaced`
```

```sh
WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/
```

It runs only when `WORKLINE_EVAL` names the agent — it costs tokens — and says
so when skipped. The name is an `--ai` value, so `WORKLINE_EVAL=claude:haiku@medium`
runs every case on one model and effort, to compare them with the roles' own. Besides the fixtures, a case can start from this repository
itself: `given.workline-commit: <sha>` stages that commit's diff on its parent
(a message the hook refused, replayed with the author's words in
`run.message`); `given.workline-at: <sha>` checks it out as it was. Each check
in `grade` is a point: `status`, `agent-calls-max`, `subject-max`,
`keeps-words` (the share of the author's subject words kept), `no-vague-words`,
`file-contains`, `file-lacks`, `checked-is-head`, `body-unchanged`,
`lines-max`, `new-files-min`, and `judge`. A case's score is the points it earned; nothing
passes or fails, the scores are compared from one run to the next. Each line of
`results.tsv` also holds the exact models that answered (`>` for a step up),
the efforts asked, the tokens in and out, and the cost at list price (on a
subscription, the tokens are what counts): a score compares only with the same
models.

**The judge.** `judge: <question>` asks a yes-or-no question on what the role
produced — "does the rewritten subject keep the author's meaning?" — of the
agent `WORKLINE_JUDGE` names (an `--ai` value, often `cmd:`), with the case,
the author's words, the role's and the change. Its facets are in
`tests/evaluation/judge/`. A yes is a point, a no is lost with its reason. The
judge never runs on the graded agent's provider: without `WORKLINE_JUDGE`, or
with one of the same provider, the check is skipped, which is no point earned
or lost and is said in the `failed checks` column; the `judge` column names
the model that judged. A `cmd:` judge is taken to be of another provider:
whoever names the command vouches for it.

**Run often.** `tests/evaluation/schedule/run.sh` runs it on the local `main`,
in a worktree of its own, and appends to this repository's `results.tsv`
(`WORKLINE_EVAL_RESULTS`); the systemd user units next to it run it every
week, or at the next session when the machine was off. It skips the week when
the last commit changing the engine, the roles or the cases already has three
runs (`WORKLINE_EVAL_RUNS`) and no model changed since: the same code on the
same models adds little, and a subscription counts the tokens. The `workline`
column of `results.tsv` names that commit.

*Tried so far:* the judge through `cmd:`, with Claude standing in for another
provider (2026-09-26): it failed a rewrite that dropped a part of the subject,
and passed a fair cut that `keeps-words` fails. *Not yet:* a judge of another
provider, and the weekly run with one.
