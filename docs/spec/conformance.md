---
sources: [tests/conformance/runner_test.go, tests/evaluation]
checked: 1b41bbf
verified: agent:claude-code
judged: fa1d682
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
  fixtures/roles/<name>/      roles of the tests only, beside the shipped ones
  cases/<area>/<case>.yaml    one behaviour each
```

A role of the tests tries a mechanism no shipped role uses: `hands-over`
asks for a handoff, which the routing cases follow.

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

What else `given`, `run` and `expect` can carry is in [the case's fields](../conformance/case-fields.md).

## The fakes

- **Fake agent.** `fake:<fixture>` returns the proposals in
  `fixtures/agents/<fixture>.yaml`, without reading the prompt. It checks the
  engine's handling of proposals, not their quality. A fixture holding
  `calls: [[…], […]]` answers each call in turn — a part asked, then the fix
  — and past the last, with the last.
- **Command agent.** `cmd:sh "$FIXTURES/agents/<script>.sh"` runs a script
  of the fixtures as the agent: one that reads the prompt, for a case where
  each call asks something else (`FIXTURES` names the fixtures folder).
- **Fake tools.** `fixtures/bin/` holds tools that need the network, replaying
  an answer the real one gave: `lychee` (recorded from 0.24.2). A case puts
  them first: `env: {PATH: "$FIXTURES/bin:$PATH"}`.
- **Unavailable agent.** `unavailable:quota` (or `auth`, `network`) fails the
  way a real agent does when a quota runs out.
- **Command forge.** `cmd:sh "$FIXTURES/forges/logged.sh"` plugs a script
  as the forge (docs/spec/forge-command.md): it writes each request it gets
  to `.git/forge-requests.jsonl`, for `files` to check, and answers as a
  small forge would; `FORGE_FAIL` makes it fail, `FORGE_REFUSE` refuse.
- **Simulated forge.** A forge kept in a JSON file, holding issues, labels,
  comments, merge requests and milestones (`milestone-due` gives one its
  due date, by title); with `sub-issues: true`, it
  links sub-issues as GitHub does, without, it has none; with
  `dependencies: true`, it keeps what an issue waits on (`blocked-by`) as
  GitHub does, without, it refuses the relation. It can be told to
  fail on the N-th write, to test recovery after a partial apply. An
  issue's `closed-by` lists what closed it, `{kind, ref, text}`;
  `label-events`, its labels set and taken off with who (`{label, added,
  author, insider, bot, created}`). A comment is its text, or `{body,
  author, insider, bot, id}`, its id its place from 1 when not given; with
  `scoped-labels: true`, the forge's labels are scoped, as GitLab's.
- **Simulated GitLab.** `forge: gitlab` runs the engine's own GitLab forge
  against a mock of GitLab's REST API (tests/conformance/gitlab_test.go)
  over the same file: its `members` (`{username, access_level}`) are the
  project's, what the engine writes is the token user's, a note another
  user wrote refuses an edit (403), an `is_blocked_by` link is refused for
  its license (403) as on GitLab Free, and GraphQL refuses all.

## Evaluation

An evaluation case uses the same `given` and `run`, with a real agent, and adds
`grade`: what a good answer contains ("the message names the changed
behaviour", "the product doc keeps every MUST"). Grading is done by checks where
possible, and by a judge model where not, as far from the graded one as can
be (below).

```
tests/evaluation/
  cases/<role>/<case>.yaml   one real situation each
  results.tsv                every run's score, appended: the history kept
  schedule/                  run.sh and systemd units: a run every week
  testdata/                  fake agents proving the cases grade as they should
  summary/                   go run ./tests/evaluation/summary: per case and models,
                             the runs, the pass rate (runs earning every point),
                             mean score, range, tokens and seconds; fewer than
                             five runs are marked `few runs`, a model that no
                             longer answers `replaced`
```

```sh
WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/
```

It runs only when `WORKLINE_EVAL` names the agent — it costs tokens — and says
so when skipped. The name is an `--ai` value, so `WORKLINE_EVAL=claude:haiku@medium`
runs every case on one model and effort, to compare them with the roles' own.
The reviewer's cases ask each lens apart; `WORKLINE_EVAL_LENSES` names the
lenses asked (`diff-alone`: the facet alone) and `WORKLINE_EVAL_REVIEWER`
sets its settings, a YAML map (`{ai-max-tokens: 10000}`). Besides the fixtures, a case can start from this repository
itself: `given.workline-commit: <sha>` stages that commit's diff on its parent
(a message the hook refused, replayed with the author's words in
`run.message`); `given.workline-at: <sha>` checks it out as it was. Each check
in `grade` is a point: `status`, `agent-calls-max`, `subject-max`,
`keeps-words` (the share of the author's subject words kept), `no-vague-words`,
`file-contains`, `file-lacks`, `checked-is-head`, `judged-is-head` (the
doc's `judged` names the commit judged), `checked-unchanged` (the doc keeps
the `checked` it had), `body-unchanged`, `never-confirms` (a claim fixed, or
left unconfirmed, never checked again as it is), `finding` and `no-finding`
(a finding of the run, or none, by `rule`, a part of `where` and of
`message`, each when given), `lines-max`, `new-files-min`, and `judge`. A case's score is the points it earned; nothing
passes or fails, the scores are compared from one run to the next. Each line of
`results.tsv` also holds the exact models that answered (`>` for a step up),
the efforts asked, the tokens in and out, and the cost at list price (on a
subscription, the tokens are what counts): a score compares only with the same
models.

How the evaluation is built — real shapes, the reviewer's measure, pass rates, the judge, the weekly run — is in [Evaluation in detail](../conformance/evaluation.md).
