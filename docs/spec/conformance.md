---
sources: [tests/conformance/runner_test.go, tests/evaluation]
checked: da339f03
verified: agent:claude-code
judged: af2d6fb
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

What else a case can give, run and expect, and how each is checked, is in
[Conformance — a case in full](conformance/cases.md), [read strictly](conformance/cases.md#read-strictly).

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
  author, insider, bot, created}`); `reactions-by`, the reactions people
  put on its comments, by the comment's id (`{emoji, author, writer}`);
  `edits`, its earlier bodies, newest first, as GitHub's edit history
  keeps them, a body set adding the one it replaces. A comment is its text, or `{body,
  author, insider, bot, id}`, its id its place from 1 when not given, kept
  once a comment before it is deleted; an edit keeps a comment's author;
  `posted` counts the comments posted on an issue, each one a mail on a
  real forge; with
  `scoped-labels: true`, the forge's labels are scoped, as GitLab's. A
  merge request's `threads` are its comments on lines, `{path, line,
  body, resolved}`: a review is posted whole, as GitHub's, on the lines
  of the repository's diff from the merge request's `base` (main) to the
  head reviewed; `review-refused: <why>` refuses every review (403). For
  the merge policy, a merge request is a `draft`, its `auto-merge` on
  (`auto-merge-at`, `auto-merge-by`), its head's `checks`, its `pushes`
  (`{head, at}`, a head a revision of the repository), `clean` (merged at
  once); one merged, `merged-at` and `merged-as` (the commit it became,
  a revision), its head under `refs/pull/<id>/head` of `origin`, as
  the weekly sample lists them; `job-token: true` is a CI job's own token; `unguarded: <why>` a
  base that holds no yes; `statuses` the statuses on commits (`{commit,
  name, state, description, by}`, a commit a revision, `by` workline's
  token's user `workline-bot` or another); `fixtures/verdict.sh` writes a
  head's verdict as `workline route --verdict` does. The
  repository's `setup` may write into it, at `$FORGE_FILE`, what only the
  built repository knows: a commit's id in a record a comment holds.
- **Simulated GitLab.** `forge: gitlab` runs the engine's own GitLab forge
  against a mock of GitLab's REST API (tests/conformance/gitlab_test.go)
  over the same file: its `members` (`{username, access_level}`) are the
  project's, what the engine writes is the token user's, a note another
  user wrote refuses an edit (403), an `is_blocked_by` link is refused for
  its license (403) as on GitLab Free, and GraphQL refuses all. A merge
  request's diffs are the repository's, from its `base` (main) to its
  `branch` (HEAD); a discussion on a line they do not show, or on a kept
  line without its old one, lands on no line, as on gitlab.com;
  `review-refused` refuses every discussion (403). Its merge API sets a
  merge request to merge when its pipeline succeeds, at its head only
  (409 otherwise), says when in a system note, and cancels it; its
  versions are its `pushes`.

## Evaluation

An evaluation case uses the same `given` and `run`, with a real agent, and adds
`grade`: what a good answer contains ("the message names the changed
behaviour", "the product doc keeps every MUST"). Grading is done by checks where
possible, and by a judge model where not, as far from the graded one as can
be (below).

```
tests/evaluation/
  cases/<role>/<case>.yaml   one real situation each
  judge-replay/<item>/       the judge's 40 measured questions, as asked then
  results.tsv                every run's score, appended: the history kept
  schedule/                  run.sh and systemd units: a run every week
  testdata/                  fake agents proving the cases grade as they should
  summary/                   go run ./tests/evaluation/summary: per case and models,
                             the runs, the pass rate (runs earning every point),
                             mean score, range, tokens and seconds; fewer than
                             five runs are marked `few runs`, a model that no
                             longer answers `replaced`; the reviewer's tallies
                             by lens, agent and how the lenses were asked
```

```sh
WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/
```

It runs only when `WORKLINE_EVAL` names the agent — it costs tokens — and says
so when skipped. The name is an `--ai` value, so `WORKLINE_EVAL=claude:haiku@medium`
runs every case on one model and effort, to compare them with the roles' own.
The reviewer's cases ask every lens in one call, as on a machine;
`WORKLINE_EVAL_LENSES` names the lenses asked (`diff-alone`: the facet
alone) and `WORKLINE_EVAL_REVIEWER` sets its settings, a YAML map
(`{lenses-together: false}`: each lens apart; `{ai-max-tokens: 10000}`).
A reviewer case may hold a simulated forge (`given.forge`, as a
conformance case's): the issues its change closes. A case is read as strictly as a conformance case: a key no field names fails to load
it, as does a check `grade` does not know. Besides the fixtures, a case can start from this repository
itself: `given.workline-commit: <sha>` stages that commit's diff on its parent
(a message the hook refused, replayed with the author's words in
`run.message`); `given.workline-at: <sha>` checks it out as it was. Each check
in `grade` is a point: `status`, `agent-calls-max`, `subject-max`,
`keeps-words` (the share of the author's subject words kept), `no-vague-words`,
`file-contains`, `file-lacks`, `checked-is-head`, `judged-is-head` (the
doc's `judged` names the commit judged), `checked-unchanged` (the doc keeps
the `checked` it had), `body-unchanged` (the doc below its header),
`never-confirms` (a claim fixed, or
left unconfirmed, never checked again as it is), `finding` and `no-finding`
(a finding of the run, or none, by `rule`, a part of `where` and of
`message`, each when given), `lines-max` (the lines below a doc's header,
as its `doc-lines` budget counts them), `new-files-min`, and `judge`. A case's score is the points it earned; nothing
passes or fails, the scores are compared from one run to the next. Each line of
`results.tsv` also holds the exact models that answered (`>` for a step up),
the efforts asked, the tokens in and out, and the cost at list price (on a
subscription, the tokens are what counts): a score compares only with the same
models.

Real shapes, both sides graded, the reviewer's measure and the judge's
replay are told in full in [Evaluation — three measures](conformance/evaluation.md).

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
demand, in steps. So is the judge's replay, never run there.

*Tried so far* (2026-09-26): the judge through `cmd:`, and `claude:sonnet` at
the `model` level on every judged case — it failed rewrites that dropped
"judged" and "instead of ignoring them", and passed a fair cut that
`keeps-words` fails. The weekly run asks `claude:sonnet` unless
`WORKLINE_JUDGE` says otherwise. *Not yet:* a judge of another provider.
