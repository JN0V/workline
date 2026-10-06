---
sources: [tests/conformance/runner_test.go, tests/evaluation]
checked: ce2783e
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

`given` can also carry `models-seen`: the models this machine saw answer before
the run (`{"fake:light": {model: …}}`), in the file `WORKLINE_MODELS_SEEN` names.
It can also carry `repos`: other repositories to build first, each
exported by name as an environment variable holding its path, for
multi-repository cases. `env` sets variables for the run (`$VAR` expands):
a `PATH` without a tool, or with a fake one first.

`run` can also carry:

- `route: <event>` instead of `role` — run the event's whole line;
- `gate: <name>` instead of `role` — run one gate;
- `doctor: true` instead of `role` — run `workline doctor` on the repository;
- `init: true` instead of `role` — run `workline init` on the repository,
  with `init-options: [<option>...]` if given (`--human-po yes`);
- `setup: [<option>...]` instead of `role` — run `workline setup` with these
  options, on a git config of the case's own;
- `issues-import: [<argument>...]` instead of `role` — `workline issues
  import` on the repository, on the simulated forge;
- `review: [<option>...]` instead of `role` — `workline review` on the
  repository, with these options;
- `sample: [<option>...]` instead of `role` — draw and read the weekly
  sample (`workline sample`), with no forge; `then: apply` then writes what
  it found with `workline sample --apply`, on the simulated forge;
- `follow: [<option>...]` instead of `role` — `workline follow` on the
  repository (`[--base, main]`), on the simulated forge, or GitLab's with
  `forge: gitlab`;
- `cli: [<argument>...]` instead of `role` — `workline` with these
  arguments, as typed, in the repository: help, misuse; checked by `exit`,
  `stdout` and `stderr` only;
- `route: ready` with `item` — ask routing to move a work item;
- `target` — the issue or merge request comments and labels go on
  (`{merge-request: 1}`); `branch` — the branch that merge request comes from;
- `open-merge-request: true` or `push-to-merge-request: true` — put what the
  patches write on a merge request;
- `reports: true` — also write the findings as SARIF and Code Quality;
- `summary: true` — also write `--summary`, Markdown and HTML, and give the
  same files to the `apply` or `sample --apply` that follows;
- `scope` — the run's scope, as a ready work item would give it;
- `no-apply: true` — judge, and stop before applying;
- `forge` — a forge spec given as `--forge` instead of the simulated forge
  (`local`, `none`, `cmd:<command>`, `gitlab`), the sample's `then: apply`
  included;
- `then: resume` — after the run, resume it with `workline apply`: the run, or
  every run a line judged and did not apply; `then: resume-elsewhere` — the
  same with another cache and the roles built into the engine, as CI's
  second job;
- `tamper: in/` — change the prepared input between prepare and apply.

`expect` lists only what the case is about; anything not listed is not checked.

- `findings` — each listed finding must be present (matched by `rule`, and by
  `where` and a part of its `message` and `fix` if given).
- `checks` — the doctor's checks, ok ones included: each listed must be
  present, by `rule`, and by `level`, `where`, a part of `message` and of
  `fix` if given.
- `no-findings` — findings that must not be there (by `rule` and `where`,
  and a part of the message when given).
- `agent-calls` — how many times the agent was called.
- `calls` — each call, in order: the fields listed must match (`agent`,
  `task`, `tier`, `effort`, `asked`, `model`).
- `applied` / `refused` — intentions applied or refused, by kind.
- `steps` — for a line, the steps that ran, in order.
- `forge` — fields the simulated forge must hold afterwards: per item, by
  `id`, `comments` (a count), `labels` (the exact set), `comment-contains`
  (a text some comment holds, or a list of them) and `comment-lacks` (a
  text none does), `closed`
  and the `reason` it was closed for, its `milestone`, `branch`, `base`,
  `title` and the `parent` it is a sub-issue of (0 for none), `blocked-by`
  the issues it waits on in the forge's relation, `body-contains` and `body-lacks` (a text its body holds, or does
  not); `absent: true` — no item with that id. `labels` there lists the
  labels the forge defines, their order as id, by `title`.
- `pushed` / `pushed-message` — a text a file holds on a branch of the
  case's `origin`, or the message of that branch's tip.
- `not-pushed` — branches of the case's `origin` the run left where they
  were; `on-top` — a branch of `origin` whose tip holds another's
  (`{workline/documentalist/release: main}`).
- `branches` — a text a file holds on a local branch of the repository.
- `issues-listed` — texts `workline issues list` prints afterwards.
- `summary` — a text the result's summary holds.
- `coverage` / `not-covered` — with `issues-import`, entries the import's
  map must hold, each matched on the fields given (`lines`, `state`,
  `issue`; `words` and `why` by a part of them); with `then: resume`,
  the judge's map, `workline apply` printing none.
- `notes` — texts the agent's notes must hold, all rounds together.
- `sarif` / `code-quality` / `left-out` — with `reports`, results each report
  must hold, and the places neither may name.
- `summary-file` / `summary-html` — with `summary`, texts the Markdown and
  the HTML summary must hold, in this order.
- `refused-kept` — how many refused answers the run folders keep.
- `calls-kept` — how many agent calls the run folders record.
- `run-files` — files of the run folder (`out/claims.yaml`), each holding a
  text (`contains`), or not (`not-contains`).
- `files` — paths that must exist, or contain a text or each of a list of
  texts, or lack one (`lacks`), afterwards.
- `exit` / `stdout` / `stderr` — with `cli`: the exit code, and texts
  printed on each stream.

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
  comments, merge requests and milestones; with `sub-issues: true`, it
  links sub-issues as GitHub does, without, it has none; with
  `dependencies: true`, it keeps what an issue waits on (`blocked-by`) as
  GitHub does, without, it refuses the relation. It can be told to
  fail on the N-th write, to test recovery after a partial apply. An
  issue's `closed-by` lists what closed it, `{kind, ref, text}`. A
  comment is its text, or `{body, author, insider, bot}`.
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
runs every case on one model and effort, to compare them with the roles' own. Besides the fixtures, a case can start from this repository
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

**Real shapes, both sides graded** (ADR-0014, step 1). The `drifted`
fixture (tests/conformance/fixtures/repos/drifted.sh) holds docs drifted as
DomoticsCore's and workline's did, every one `checked`, its sources fitting
whole in a task: frozen line counts (a fenced tree's "(524 lines)", a
`| File | Lines |` row, "~283 lines"), limits that are no counts ("< 800
lines"), a sibling sharing the version, a stale code comment against the
default, a bug long fixed, a test count in a file no source names, a true
claim only a record backs, and a clean control. Each documentalist case on
it grades both sides: never `checked` over a planted falsehood, and
`checked` on the control (`TestDriftedCasesGradeBoth`). Its cases are played
without a real agent on every `go test`, by a fake `cmd:` agent changing
headers only (`TestDriftedWithFakeAgents`): vouching for every doc must lose
the falsehood checks, recording `judged` on every doc must lose the
control, doing both right must lose neither; the `count-off` findings,
reported by the engine with no agent (ADR-0014, step 2), are graded there
too and must never be lost.

**The reviewer's measure** (#90). A reviewer case adds `review:` to
`given` and `run`: the `defects` planted on the change, each a lens, a
kind, and the places its cause may be quoted at (a file, a range of lines
as they read at the change's head, a `text` within them); the places
where nothing may be found (`not`); and code outside the change
(`outside`), measured and not scored. Built on the `shop` fixture, every
lens asked, each apart (`lenses-together: false`, set by the harness),
event `review`. The run's folder is read with no agent — what
each lens answered, what the engine kept of it, the judge's verdict on
each important finding, the tokens each lens's call and its judges used —
and scored: a point for each defect a finding shown on the change has its
cause in, whichever lens raised it (AI review benchmarks count known bugs
found and false positives, docs/research/code-review.md; the place, from
the quote the engine found again, needs no judge); a point for each
`not` kept clean; a point when nothing important was shown that nothing
planted. The `measure` column of `results.tsv` holds, a lens each, the
defects planted for it and found, those it raised itself, those the judge
refused, what it raised that nothing planted (shown, nits, refused),
outside the change (and verified), its answers, its quotes not found
again, its tokens, and the finder floor asked; the summary sums them by
lens. `TestReviewerCasesPointRight` builds each case — each place holds
its text, the code builds, the fixture's tests pass, a planted defect
they caught measuring nothing — and `TestReviewerWithFakeAgent` plays one
through the engine with a fake agent (testdata/review-agent.sh). The
model and independence matrix of ADR-0005 is a run with
`WORKLINE_EVAL=claude:<model>` and `WORKLINE_JUDGE=claude:<model>`, never
by default.

**Pass rates, not the best.** A measure is read over at least five runs per
model, as the share of runs earning every point, never from the best run.
What a change is designed on is never what it is accepted on: the
**held-out set** — WaterMeter (a solo repository, roles/documentalist/docs/tried.md),
DomoticsCore after its pull request #114, and workline's own nightly
gardening — is not looked at while designing ADR-0014's steps, and is
replayed only to accept them (its step 4).

**The judge.** `judge: <question>` asks a yes-or-no question on what the role
produced — "does the rewritten subject keep the author's meaning?" — of the
agent `WORKLINE_JUDGE` names (an `--ai` value: `claude:sonnet`, a `cmd:`),
with the case, the author's words, the role's and the change. Its facets are
in `roles/judge/`. A yes is a point, a no is lost with its reason.
The judge stands as far from the graded model as it can
([ADR-0005](../adr/0005-independence-takes-the-best-level-available.md)): the
`judge` column names its model and the level reached — `provider`, `model`
(another model of the same provider), `context` (the same model, apart). A
`cmd:` judge is taken to be of another provider: whoever names the command
vouches for it. `WORKLINE_JUDGE_AT_LEAST` sets a floor. Without a judge, or
below the floor, the check is skipped: no point earned or lost, said in the
`failed checks` column. The judge's own tokens are not counted.

**Run often.** `tests/evaluation/schedule/run.sh` runs it on the local `main`,
in a worktree of its own, and appends to this repository's `results.tsv`
(`WORKLINE_EVAL_RESULTS`); the systemd user units next to it run it every
week, or at the next session when the machine was off. It skips the week when
the last commit changing the engine, the roles or the cases already has three
runs (`WORKLINE_EVAL_RUNS`) and no model changed since: the same code on the
same models adds little, and a subscription counts the tokens. The `workline`
column of `results.tsv` names that commit. The reviewer's cases are left
out (`WORKLINE_EVAL_SKIP`, `reviewer` by default): they are measured on
demand, in steps.

*Tried so far* (2026-09-26): the judge through `cmd:`, and `claude:sonnet` at
the `model` level on every judged case — it failed rewrites that dropped
"judged" and "instead of ignoring them", and passed a fair cut that
`keeps-words` fails. The weekly run asks `claude:sonnet` unless
`WORKLINE_JUDGE` says otherwise. *Not yet:* a judge of another provider.
