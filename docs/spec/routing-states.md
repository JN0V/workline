---
sources: [internal/line/budget.go#BudgetWaitMax, internal/line/crossed.go#unanswered, internal/line/freed.go#freed, internal/line/freed.go#woken, internal/line/payload.go#payload, internal/line/seconds.go#round, internal/line/states.go#never, internal/line/states.go#reconcile, internal/line/wip.go#waits, internal/line/wip.go#waiting, internal/routing/adopt.go#config, internal/routing/adopt.go#config.write, internal/routing/routing.go#Config, internal/routing/states.go#State, internal/routing/states.go#Config.State, internal/routing/states.go#Config.Wakes, internal/work/headings.go#Scope, internal/work/work.go#check, defaults/routing.yaml]
checked: a19e8cb0
verified: agent:claude-code
---
# Routing — the line of states

Part of [Routing](routing.md): the line of states, and the reconciler that reads it again.

## The line of states

[ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md):
an issue's state is its label, one at a time; each state wakes one role,
whose result moves the issue to the next. The table is the routing's
`states`, a project's list replacing the shipped one:

| State | Label | Wakes | Its result moves it to |
|---|---|---|---|
| blocked | `workline:blocked` | nobody: its reason on the issue; the product owner reads it again on news ([below](routing-blocked.md#blocked-read-again)), at once for a developer's Scope ask or a run too big for its role | — |
| doing | `workline:developing` | nobody: the developer's claim; one with no pull request wakes the developer to let it go, past `claim-hours` or on a change to its body, the day said until then ([#1084](https://github.com/JN0V/workline/issues/1084)); one whose agent wrote no code on the body it still has is blocked for a person past `claim-hours`, never run again on that input (`no-code-spent`, [#1120](https://github.com/JN0V/workline/issues/1120)); one with its pull request open waits on it, said, naming who acts next: a Draft the developer's, its round woken by its checks' end or a red one, a person's only past a cap, its agent's failure or its last round; a change a person made to its body since the code (the engine's own `Blocked by` line aside) is the developer's review round on it ([#1271](https://github.com/JN0V/workline/issues/1271)) | — |
| ready | `workline:ready` | the developer | doing |
| tests | `workline:needs-tests` | the tester | ready |
| refine | `workline:to-refine` | the product owner | tests |

- **Read by a pure function** of the issue's labels
  (`routing.Config.Wakes`); an issue bearing two takes the first row
  listed.
- **The issue line**: `workline route issue --target issue:<n>` runs the
  role the issue's state wakes, alone, when the project routes it on
  `issue`; none, said (`state-wakes-none`): a state that wakes nobody,
  or a role not routed there. Without a target, the event's steps run
  as listed. On a forge, a role's work is handed over in a merge
  request, as `--open-merge-request` does, unless the run says
  `--open-merge-request=false`; then nothing of a role that proposes
  one is applied (`merge-request-not-opened`,
  [#1280](https://github.com/JN0V/workline/issues/1280)).
- **GitLab's webhook**: on `issue` with no target, a pipeline its
  trigger started (`$TRIGGER_PAYLOAD`, the event or its file) names the
  issue an issue event put a state's label on, or a note event's
  reviewer comment on a spec in refine (below); any other event — an
  edit, another label, another comment — runs nothing, said at once. An
  event that does not read blocks the run (`routing-error`, at `payload`).
- **The product owner on `issue`** reads the issue named alone, at
  once, not a night's share: one with no state of its own yet too. On
  `issue` with no issue named, it reads none. Only that issue's comments
  and label events are listed, when it judges and when its acts are
  decided and written; the open issues' listing stays, for the order
  and `proposals-max` ([#804](https://github.com/JN0V/workline/issues/804)). The developer on `issue` likewise reads
  the tester's recorded proof on that issue alone.
- **The API budget first**: on a forge that says it at no cost
  (GitHub's `rate_limit`), the issue line and the reconciler read what
  the token has left before they read an issue. Under a tenth, they wait
  for a reset within `BudgetWaitMax` (5 min; `api-budget-low`); further
  off, the run is deferred, passing, nothing read nor run, the step left
  due for the reconciler (`api-budget-deferred`). A call refused for the
  rate half-way, theirs or a role's, waits for a reset as close
  ([the API budget](../github/calls.md#the-api-budget)).
- **The spec review at once**: where the project routes the reviewer
  on `issue`, refine's step runs it after the product owner, on that
  issue's spec alone ([#128](https://github.com/JN0V/workline/issues/128)). Judged with no write token, the reviewer
  reads it though the product owner's state comment is not written
  yet: the apply writes both
  ([#840](https://github.com/JN0V/workline/issues/840)). Its review,
  written since, makes the step due again: the product owner reads it
  then, and readies the issue. Its mark counts on a line of its own; a
  person's comment quoting it (`> `) is no review. On GitHub the apply that wrote it starts
  that step; on GitLab the webhook's note event does, the reviewer's
  comment on the spec of an issue in refine being the one comment that
  names an issue
  ([#846](https://github.com/JN0V/workline/issues/846)).
- **The wait said**: the product owner holding the issue for that
  review (`spec-not-reviewed`), the line's comment on it says what it
  waits for, and what comes next; the next record drops it.
- **Moves are exclusive**: the next state's label on, every other
  state's off, in one call.
- **Its card follows**: on a board with a column for the state, named
  as the state is (refine, tests, ready, doing, blocked), the card moves
  with the label ([boards](../board-moves.md#what-each-state-moves)).
- **Who moves it**: the product owner's ready sends an issue with named
  scenarios to `tests` where the tester is routed, else to `ready`
  ([refine](backlog-acts/refine.md)); the tester's proven, linted Draft
  moves it to `ready`
  ([the hand-over](../../roles/tester/docs/written.md)),
  as its proof kind does; its "no test possible" sends it back to
  `refine`, where the product owner's ready with a proof kind moves it
  to `ready`
  ([refine](backlog-acts/refine.md#no-test-possible-a-proof-kind-picked)).

### The reconciler

`workline route reconcile` walks every open issue, in the backlog's
order, and runs each step due that no event started: hourly in CI, a
command on a person's machine, its next loop the next step. On GitHub,
each reconcile starts the next one, past an environment's wait timer
([the heartbeat](../github/heartbeat.md)): GitHub's schedule runs late.

- **No agent of its own**: it reads labels, records and label events;
  the steps it starts ask theirs.
- **`reconcile-max`** (3) steps a run at most; the issues due past it
  are said (`reconcile-left`) and wait for the next run.
- **The one waiting longest first**, not the backlog's first
  ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-the-one-waiting-longest-first-room-wakes-the-waiting)):
  - a first step in its state waits since its state's label was put on;
  - a step due again in the same state, since its last run (`ran` in
    the record; none known, after the others);
  - a write with no agent, a block, first.
  A step started waits anew from then: each due step starts within a
  bounded number of runs, whatever its age.
- **`--input room=true`** with `reconcile=list`: only the steps due
  whose next state has a limit and room now, for the job that freed it
  (`state-room`); the summary says them.
- **`--input reconcile=list`**: it says the issues due (`due` in its
  result), starts none and writes nothing, for a CI job that starts each
  one's own run. `--input skip` leaves a role's issues out.
- **`workline tick --issues`** reads the same listing, each issue's step,
  review round and answer due as its own job beside the line's timed
  jobs and the backlog run ([the tick](../tick.md)).
- **The developer's review rounds**: where the project routes the
  developer, it reads its open pull requests too. A round due that no
  event started (the reviewer's verdict, the gates' report, a project
  check red at a head or a person's comment its last round did not
  read) is counted in
  `reconcile-max`; listed in `reviews`, number and branch, for the CI
  job, else said with its command (`review-round-due`;
  [#891](https://github.com/JN0V/workline/issues/891)).
- **The local forge's pull requests**
  ([#1324](https://github.com/JN0V/workline/issues/1324)): no job runs
  on a push nor a comment there, so the reconciler, on a person's
  machine, runs them itself. Each open developer's request, not a
  draft, whose head has no verdict kept: its merge-request line, its
  verdict kept (`line-ran`), once a head; a machine error keeps none, the
  next reconcile runs it again. Each round due: run, not said
  (`round-ran`), unless its branch is checked out (`round-not-run`).
  Each tester's Draft an objection waits on, unanswered — the review's
  test findings the developer sent there, or its own objection — its
  answer run (`answer-ran`, `workline test answer`), where the tester
  is routed ([#1265](https://github.com/JN0V/workline/issues/1265)).
  One that wrote no answer is said on the Draft (`answer-none`), not run
  again until a comment is added there (`answer-waits`).
  A comment on a line is kept as a comment of its own on the merge
  request, its place hidden in it; a reply names it. All in a worktree
  of the head, detached, the clone's config laid in, even an uncommitted one (intended)
  it, removed after; counted in `reconcile-max`; none with `--no-apply`.
  A person's comment is one with no workline mark.
- **An issue whose step is its round**: in the developer's state, its
  own pull request open from `workline/developer/<n>`, an issue is no
  new claim (`state-round`), as one back to ready from a Scope ask.
  `route issue` on it lists that round in `reviews` when due, or says
  its command; nothing written. The reconciler leaves the issue, and
  lists the round with the others
  ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-04-a-widened-scope-resumes-its-round-at-once)).
- **The stalled pull requests**
  ([ADR-0068 §8](../adr/0068-the-line-steps-back-on-its-own-drift.md#8-the-wake-script-print-only)):
  each developer's open pull request, a Draft too, whose head has not
  moved for `routing.stalled-hours` (12; `0`, never) and whose verdict
  kept is not a pass is said in the summary, its age and the verdict's
  step; read with no agent, as the wake reads it. A verdict the forge
  does not give is said "a stall not read", with why: on GitHub, the
  job names the App that keeps the verdicts (`WORKLINE_APP`).
- **An issue that cannot be read** is said (`state-unread`) and the
  others walked; the forge out of reach stops the run.
- **Run twice in a row**, the second does nothing more: the first left
  its records.
- **`--input last=<RFC 3339>`**: when it last ran, as the CI keeps it.
  Over two hours ago, its summary says how late it is
  (`reconcile-late`;
  [#789](https://github.com/JN0V/workline/issues/789)).
