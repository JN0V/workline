---
sources: [internal/line, internal/routing, internal/work, routing.default.yaml]
checked: 07ca9b9b
verified: agent:claude-code
---
# Routing — v1 (draft)

Who works next is decided by a file, not by a model. Routing reads an event and
the state of the work, and names the roles and gates to run, in order. It is a
pure function: same inputs, same answer, no AI.

## Work starts from a clear need

People own both ends of the line: they state the need and how the result will be
judged, and they accept or reject what comes out. The machine builds in between.
So no development starts without a clear expression of the need. That does not
mean pages of specification; it means the work is no longer research.

A work item — an issue on the forge, or a file in `.workline/work/` without one
— moves through these states:

```
to-refine ──► ready ──► in-progress ──► review ──► validation ──► done
    ▲            │
    └────────────┘  (the need turns out unclear: back to refining)
```

**Ready** means three things are written, even in one line each — Need,
Validation, Scope; the fourth, Verification, is the tester's, written
from its tests in `needs-tests`, before ready, as a tester writes the test plan
([ADR-0066](../adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md)):

| Field | Question |
|---|---|
| Need | Which user need does this answer — who needs what, and why? |
| Verification | How will the machine prove it works: which tests, checks, thresholds? |
| Validation | How will a person accept it: who, looking at what? |
| Scope | What part of the project does it touch? |

Verification is what the line checks on its own; validation is what a person
decides at the end. Writing both before starting is what keeps the machine
from grading its own work. The engine checks that the three are present and
not empty, the Verification once the tester wrote it; it does not judge their quality. A section is the text under its
`## ` or `### ` heading (`work.Sections`): a heading written twice holds
both texts, one naming a Gherkin scenario (`### Scenario: …`) is no
section but a scenario of the one it lies in, and a thematic break ending a section — the line between the
person's part and the builder's — is not its text. Each is read by its
meaning, through one table (`work.Headings`,
[ADR-0039](../adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)):
a meaning's first heading is the one written, every one is read; today
each has one, the names above. Other headings, an
example or the steps to reproduce, are kept and never required
([writing an issue](../../roles/product-owner/docs/writing.md)). A person moves an item to
`ready`, or the product owner does once Need and Validation are a person's
([refining to ready](backlog-acts/refine.md)), through the tester's state
where the project routes the tester ([the line of states](#the-line-of-states)).
Development roles only take `ready` items, and the item's **scope** guides
the run ([stay on the task](role-outcome.md#stay-on-the-task)). *Built for an issue: the developer
takes a `ready` one, its Scope a hint, each file beyond it listed with why
([bounds](../../roles/developer/docs/bounds.md#beyond-the-scope)); a file in
`.workline/work/` is taken by no role yet.*

`review` is the machine's part: reviewers and gates run the verification.
`validation` is the person's part, by the issue's kind
([ADR-0060, decision 4](../adr/0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind)):
a reproduced bug's fix merges by the project's policy once every gate
passes; a feature or a task waits for a person's yes, on its pull
request, or at `enterprising` on the release candidate
([the release candidate](../release-candidate.md)). A rejection sends it
back to `in-progress` with the reason, or to `to-refine` if the need
itself was wrong.

An item that needs exploration stays in `to-refine`. Research, spikes and
prototypes happen there, and their outcome is a refined item, not merged code.

## The line

workline ships a default line (`routing.default.yaml`); a project changes it in
the `routing:` section of `.workline/config.yaml`. An event it lists replaces
the default one; a `handoffs` list it gives replaces the default list — an
empty list forbids every handoff. `workline route <event>` runs an event's
steps; the git hooks run `commit-msg` and `pre-push` through the line too.
`pre-push` is not routed by default: the global hook reaches every repository
on the machine, and a project opts in.

```yaml
routing:
  events:                                 # event -> what runs, in order
    commit-msg:    [committer]
    merge-request: [committer, documentalist, gate:merge]
    schedule:      [documentalist]
    release:       [gate:release]       # before the release tool tags; never the docs
    main:          [documentalist]      # after a merge: merge-request's gates first

  fail-fast:                            # per event; true unless set
    schedule: false                     # every step runs, the worst verdict stands
    main: false

  handoffs:                             # the only handoffs a role may ask for
    - {from: my-role, to: documentalist}  # a project's own role; none shipped asks
```

workline cuts no releases: a release tool does (release-please,
semantic-release…), and `release` runs the project's release gates before
it; none by default, never the docs
([ADR-0017](../adr/0017-the-release-manager.md),
[ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)). A release tool's
pull request is a merge request as any other; each role is told its
branch (`release.branches`). A role routed on `release` that does not run
there, the documentalist in a routing written before
[ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md), is
skipped, said (`step-skipped`, a warning). A routing that still
names the retired `release-manager` is refused, saying which tool to use.

On the release — `release`, or `merge-request` from a release tool's
branch — the line also reads what holds it on the forge, once, beside its
steps, stopping none of them; its verdict takes the worst
([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md),
[what holds a release](../release-holds.md)): each open issue labelled
`workline:hold-release` (`release-held`, by its number); the checks on the
tip of the branch the merge request goes into, the default branch on
`release` — a red one is `release-base-red`, one not done
`release-base-pending`, both blocks; none at all, `release-base-unchecked`,
a warning. Given `--release-wait`, checks not done and nothing else
holding it, the line reads them again once its steps ran, every 30 s
(a twentieth of the wait if shorter) till none is pending or the wait
ends: the job runs on meanwhile, pending on the forge, and ends with
`release-base-red` or none; still not done, `release-base-pending`
says the time waited. A read that fails while waiting is a hiccup: the
wait goes on from the last read, and only the last read's failure holds. On GitHub only the runs a push started count, with other apps'
checks and the commit statuses, workline's own left out, and a workflow that
could not start counts red; on GitLab, the
newest pipeline of the tip no schedule started. What could not be read is
`release-holds-unread`, a block; with no forge, or one that says nothing of
releases, a warning, as is a merge request with no `--branch` the forge does
not say is a release tool's. The line's summary ends with `the release held: …`.
Not read by `--gates-only`, `--warn-only`, a queue's run, nor the night's
release judging.

- Steps run in the listed order. The first step that ends in `block`,
  `human`, `blocked-external` or an error stops the sequence. It fails closed.
- But on an event the routing sets `fail-fast: false` for, every step runs,
  and the line takes the worst verdict: `block`, then `human`, then
  `blocked-external` ([ADR-0037](../adr/0037-the-schedule-runs-every-step.md)).
  The shipped line sets it for `schedule`, whose roles are upkeep, each on
  its own: a documentalist that blocks does not keep the product owner
  from the backlog. A role that does not pass runs no handoff either way.
- There, an agent's proposal still refused once asked again is upkeep left
  for later, not a failure: nothing of it applied, each refusal said as a
  warning, the step `pass`; the next run asks again
  ([ADR-0037, amended](../adr/0037-the-schedule-runs-every-step.md#amendment-2026-10-09-a-refused-proposal-is-said-not-failed)).
  Anything else that does not pass still counts.
- A `handoff` intention is applied only if its edge is declared here. Anything
  else is refused, like any invalid intention.
- A chain of handoffs is capped (`max-handoffs`, default 3). A loop stops with a
  verdict, not with a timeout.
- Events caused by the engine's own writes (its comments, labels, commits) do
  not trigger routing, so two roles cannot wake each other forever. Its
  commits carry a `Workline-Role: <role>` trailer, and a `Workline-Model:
  <agent>:<model>` for each agent and model whose answer they hold, so a
  later reader can stand apart from them
  ([ADR-0005](../adr/0005-independence-takes-the-best-level-available.md)).
  A line given a range
  whose last commits carry one judges the range as it stood before them,
  each left for its parent, a merge of the base (`workline follow`'s,
  [ADR-0041](../adr/0041-workline-settles-its-own-lines-when-the-base-moves.md),
  which [ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)
  removed: one still on a branch) for its first: what the engine wrote never wakes the line on
  itself, and a person's commit that blocked still blocks, never turned
  green unjudged. With nothing of a person's left in the range, only the
  committer runs; it checks every commit, the engine's too. A trailer is not
  taken on trust: a merge is the engine's only when it merges the range's
  base, or a commit of its history, and its tree is exactly what
  workline's own merge of its two parents gives; another commit only when
  it changes doc pages alone — Markdown, never a role's facet (in
  `roles/<name>/` only its README.md and `docs/`) nor under a tool's
  folder, never a file steering agents (AGENTS.md, CLAUDE.md…); else it is
  judged as a person's
  ([ADR-0042](../adr/0042-the-engine-merges-by-the-projects-policy.md)).
  `Workline-Role: committer` marks a person's commit whose message the
  committer rewrote
  ([ADR-0046](../adr/0046-the-auditor-re-checks-every-roles-acts.md)): never
  the engine's.

### On a forge: judge, then apply

`workline route <event> --no-apply` runs the whole line in a job that holds no
write token, with the forge and target of the merge request (`--forge`,
`--target`; `--input` and `--scope` go to every step). Each step judges the tree
as it is — no step sees what an earlier one proposed — and nothing is applied.
The result lists the runs to apply, in the line's order (`pending`) — a
step that went round, proposing a merge request a round, gives one run a
round ([the role contract](role-contract.md#one-run), "Again");
`workline apply --line <result>` applies them in the job that holds the token
and no AI key, and stops at the first that does not pass, saying the runs
left after it (`runs-not-applied`) and the command that resumes them. A role
keeping a backlog decides its acts in the judging job, reading the forge
only, and keeps that plan for the apply
([on the issue](backlog-acts/on-the-issue.md)).

The gates go before, in a job of their own, with no write token and no
agent's: `--gates-only` runs the event's gate steps alone, each one, and
the judging job's line, given that result (`--gates-from`), reads each
gate step's verdict there and runs none; one not there is a `check-error`
([gates](gates.md#shown-on-the-merge-request)). A role step after a gate
step is given what the gates found: the input `gates`, a file holding
their result as `--gates-only --json` prints it, the gates' job's when
read from it, else written in the git directory until the line ends
(`gates-not-written` when it cannot be). The inspector triages
the new findings there ([ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md));
another role ignores it.

What only warns goes beside it, off the verdict's path
([ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#2-an-act-over-its-budget)): with
`--warn-apart`, a step whose role lists the event in role.yaml's `apart`
(the reviewer and the inspector, on a merge request) is given the input
`warn=apart` and leaves what only warns — the reviewer's lenses that
warn, the inspector's whole triage; any other step runs whole,
never given it. With `--warn-only`, the line runs only those steps, given
`warn=only`, no gate and no other role, and writes no verdict
(`--verdict` refused); a flag and an `--input warn=` naming the other run
are refused. Given `--gates-from`, it hands the gates' job's result to
those steps (`gates`), only for the commits it judges, else said
(`gates-unread`), and reads no verdict there. Its
proposals are applied as the judging job's, by a job of their own: the
head's verdict is never theirs
([apart](../../roles/reviewer/docs/apart.md)).

Where the merge request's code can change its judging job (GitHub), the
job that applies runs from the base and reads only the run's proposals
(`--from-merge-request <n> --head <sha>`): refused whole, a role not on
merge requests or a proposal its role never makes, an act on the
backlog, a comment or a label on an issue other than the one the role's
branch was opened for — but on a round of the role's own branch, a
comment on the issue it closes and a move of that issue to Blocked or
Doing ([#924](https://github.com/JN0V/workline/pull/924))
(`run-refused`);
once the branch moved on (`moved-on`), only what the role names in
`on-moved-on`, the reviewer's review or the inspector's triage of the head
judged, never a patch nor
a label ([ADR-0044, amended](../adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md#amendment-2026-10-15-the-review-of-a-head-the-branch-moved-on-from)).

Applying runs no role, so a handoff proposed under `--no-apply` is recorded
and not run: both steps say so (`handoff-deferred`), with the command that runs
it where an agent may judge.

### A draft asks no agent

[ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#4-what-the-table-holds-to):
given `--input draft=true`, every step, a handoff too, runs as with no
agent, told it waits for ready (the role contract's
`WORKLINE_AI_WAITS`). The line's summary says the AI review waits for
the merge request to be marked ready; a role whose agent would have
judged says so too (`ai-waits-for-ready`, a warning), and the head's
verdict is then a review to come, never a pass
([merge policy](../merge-policy.md)). The rules and gates run as on any
push. With no agent set, nothing waits; `draft=false` changes nothing.

### Steps left out

`--input skip=<step>,…` leaves those steps out of the line: one schedule
runs some roles weekly, others nightly, as workline's own does (the docs
on Sundays, the backlog each night).

### Once, at ready

[ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md): a role whose
`merge-request-ai` is `ready` (the documentalist) asks its agent on the
run at ready alone. The templates pass `--input ready=true` on that run,
`ready=` on a push after it; there the role runs as with no agent, told
so (`WORKLINE_AI_WAITS=judged-at-ready`), and lists what the push made
suspect (`judged-at-ready`, a warning). Unsaid, a run is at ready.

### A release's dry run

[ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#3-the-moment-table):
`route release --dry-run`, each night, says what would hold the
release if it were cut now ([how](../release.md)):

- **On what**: the branch's tip. A tip that is a release already runs no
  step: nothing to cut.
- **Every step** of `release`, none stopping the others, with no agent
  and nothing applied. What a step that does not pass says blocks, else
  what it leaves to its verdict, else the step, is listed in the
  result's `dry-run` (`holds`, `head`, `since`); then
  lowered to a warning. The line passes.
- **Refused** (misuse, 64): another event, an agent named, `--target`,
  `--open-merge-request`, `--verdict`, the
  gates alone or read, `--warn-*`.
- **Posted** by `apply --release-dry-run`: a part of main's issue,
  edited when what it lists changes, taken out when nothing holds.

### A merge queue's or train's run: the gates alone

[ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#4-what-the-table-holds-to):
given `--input queue=true`, the line runs the event's gate steps alone,
on the merged result its range ends with, whoever made its last commit;
no role runs, so no agent is asked. With `--gates-from`, it reads their
verdicts, as on a merge request. The line's summary says it is a queue's
or train's run, and names the steps it left, their verdict the head's;
with no gate, that nothing was checked. A `--verdict` is refused: the
head's verdict and the person's yes stay the merge request's
([ADR-0042](../adr/0042-the-engine-merges-by-the-projects-policy.md));
so is `--warn-only`, and any value but `true` or `false`
([a merge queue or train](../merge-queue.md)).

### Main after a merge

[ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#3-the-moment-table):
`route main`, on the default branch's new tip, given the push as
`range` ([how](../main-after-merge.md)):

- **Its steps**: the gate steps of `merge-request` first, each once,
  then those `main` lists (the documentalist, by default); every step
  runs (`fail-fast: false`).
- **The gates**: what the push adds since main's tip before it counts,
  the range's start, else HEAD's first parent; with none, every finding
  is said, none counted. Every finding is kept by HEAD's commit in the
  folder `--gates-kept` names, the next base.
- **No agent asked** by the shipped roles: the documentalist checks the
  whole repository there, judging no doc. The engine's own commits are
  checked as any other.
- **Its result** names the event (`event`) and what it compared
  (`main`: `before`, `head`, the commits between); `apply --main` keeps
  it in one issue, edited only when what it lists changes, and, red,
  says so once on each open merge request whose base holds it.

### Each step's seconds, a moment's budget

[ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#1-a-wait-budget-per-moment):
a line's result says each step's `seconds`, its own, and the `moment`
it is part of: `name`, `budget`, `seconds`, `over`, and then the `step`
that took the most. Its summary says them in one line after the steps,
`66 s in all — committer 0.5 s, documentalist 0.8 s, reviewer 64 s
(lenses 40 s, judges 20 s)`: the agents' share of a step, the
reviewer's lenses, the inspector's triage, the committer's rewrite, the
judges. A moment over
its budget says so once more, naming that step; it never blocks.

| Moment | The run | Budget |
|---|---|---|
| `commit` | the `commit-msg` line | 1 s |
| `commit-rewrite` | the same, an agent asked: the committer's rewrite | 30 s |
| `push` | the `pre-push` hook, every range pushed together | 5 s |
| `first-result` | `merge-request --gates-only` | 1 min |
| `verdict` | `merge-request`; the gates' job's seconds added, their verdicts read from it (`--gates-from`) | 3 min |
| `ready-review` | `merge-request --warn-only`: the lenses that warn, the inspector's triage, posted when done | 10 min |
| `queue` | `merge-request --input queue=true`, a merge queue's or train's run: what workline adds to it | 1 min |

- Any other event has none: main, the night and the release, its dry
  run too, wait for nobody.
- The line measures itself: a runner's setup and queue, and the job
  that applies, are not in it.
- A project sets its own in `routing.budgets`, in seconds, finite and
  above 0; a moment not in this table is refused (`config-invalid`).
- At a commit, the hook says the seconds with what it says already, a
  refusal or a rewrite, and a commit over its budget even when it says
  nothing else; at a push, each time it checks commits, a stop
  included (a branch deleted checks none). Each agent call's seconds
  stay in its run's `out/calls.jsonl`.

## The line of states

[ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md):
an issue's state is its label, one at a time; each state wakes one role,
whose result moves the issue to the next. The table is the routing's
`states`, a project's list replacing the shipped one:

| State | Label | Wakes | Its result moves it to |
|---|---|---|---|
| blocked | `workline:blocked` | nobody: its reason on the issue; the product owner reads it again on news ([below](#blocked-read-again)), at once for a developer's Scope ask or a run too big for its role | — |
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
  with the label ([boards](../boards.md#what-each-state-moves)).
- **Who moves it**: the product owner's ready sends an issue with named
  scenarios to `tests` where the tester is routed, else to `ready`
  ([refine](backlog-acts/refine.md)); the tester's proven, linted Draft
  moves it to `ready`
  ([the hand-over](../../roles/tester/docs/hand-over.md#when-the-tests-are-written)),
  as its proof kind does; its "no test possible" sends it back to
  `refine`, where the product owner's ready with a proof kind moves it
  to `ready`
  ([refine](backlog-acts/refine.md#no-test-possible-a-proof-kind-picked)).

### Once a state: the record

- **Idempotent**: the line keeps one comment of its own on each issue
  it ran a step for (`workline:sticky=line/state`), holding the state
  acted on, the body's hash, when the state's label was put on (the
  forge's label events), and the hops so far. The same three again run
  nothing (`state-acted`); labelled anew, or its body changed, the step
  is due again.
- **A person's labels are input, the role's own writes are not**
  ([ADR-0063, §3](../adr/0063-every-open-issue-is-due-waiting-or-done.md#3-the-input-of-a-step)):
  the record also keeps `workline:accepted` and the size tiers
  (`labels`). After a step's run, the body and the size tiers are
  recorded as the run left them, `workline:accepted` as the step
  started on it: the role's own edit does not wake it again; a label a
  person puts on afterwards does. A record from before takes its size
  tiers as seen, and `workline:accepted` as new only when put on after
  its last run (`ran`).
- **A clean review takes no agent**: refine's step woken by the
  reviewer's review alone, nothing important open, every section and
  scenario there, runs the product owner with no agent: its move to
  ready is a rule (`state-ready-clean`). Left undone, the line says so
  (`state-ready-held`) and the agent reads the review at the next run.
- **A refusal is not acted**: an issue in the tester's state whose last
  tester run wrote no tests, said in the tester's comment, is never
  left waiting there: the line moves it to `workline:blocked`, the
  tester's reason said (`tests-refused`;
  [#794](https://github.com/JN0V/workline/issues/794)). Not for a
  machine error: see below.
- **A machine error moves nothing**
  ([ADR-0063, §6](../adr/0063-every-open-issue-is-due-waiting-or-done.md#6-typed-outcomes)):
  each run's outcome is typed from its findings (`verdict.OutcomeOf`).
  A run blocked by one of a few codes is a machine error, by kind:
  `engine-error` (run: the run failed, any shape, a new one too),
  `agent-invalid-output` and `no-test-written` (answer),
  `patch-unreadable` (patch), `build-broken` (build: the tester's test
  or the developer's code), `no-new-test` and `report-missing` (report), `proof-over-time`
  (time: a test command stopped at its deadline, [#1197](https://github.com/JN0V/workline/issues/1197)). No label, no state, no hop: the record
  keeps the count by kind (`errors`), and its comment says the error and
  when the role runs again. The step is due again on the same input
  after 10 minutes (`state-retry`; before, `state-waits` with the time),
  given what failed at the end of its task; 2 tries per kind within
  24 h of that kind's first error (`line.ErrorBudget`). The next error
  of the kind, or its 24 h passed, moves the issue to
  `workline:blocked`, the error said (`machine-error-spent`). A new
  input — the state entered anew, the body or labelling changed —
  starts the count again. The developer's code that does not build is
  asked again once in the run, with the error, as the tester's test is
  ([#1083](https://github.com/JN0V/workline/issues/1083)); not when its
  agent objects to a frozen test beside it: the tester answers that.
  The tester's comment then says the line runs it again, never that it
  blocks it.
- **A run too big for its role asks the product owner first**
  ([ADR-0064, step 3](../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints);
  [#1168](https://github.com/JN0V/workline/issues/1168)): its outcome is
  `needs-input(product-owner: size)`, never a machine error, never run
  again on the same input, which crosses the same
  ([#1160](https://github.com/JN0V/workline/issues/1160)).
  - **When**: its agent not asked for its prompt's size
    (`prompt-over-budget`, passed or not); blocked at its token cap
    (`ai-max-tokens`). A cut of what it read (`input-cut`) is said, never
    a crossing
    ([amended](../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)).
  - **Then**: the step moves the issue to `workline:blocked`, the size
    said, and that the product owner reads it at once (`size-crossed`).
    The record keeps the role (`size-of`) and the state (`sized`).
  - **The product owner reads it** at the next run, no news needed
    (`state-revisit`), told the facts: what crossed, with the run's
    numbers, and the issue's fresh tokens so far against its normal and
    its backstop
    ([#1193](https://github.com/JN0V/workline/issues/1193)). It narrows
    the Scope and sends it back; grants it `workline:size/large` and sends it back; splits it,
    `split` on for it, and keeps it blocked for a person to accept from
    its parts
    ([ADR-0064, step 4](../adr/0064-size-is-a-target-not-a-wall.md#3-the-product-owner-decides-once-a-state-an-issue);
    [#1171](https://github.com/JN0V/workline/issues/1171),
    [#1172](https://github.com/JN0V/workline/issues/1172)); or keeps it
    blocked, `person: true`. Its run failing, or its answer not read,
    keeps the question, asked again within the machine-error budget.
  - **Once a state, per issue**: a second crossing from a state in
    `sized`, the product owner's own crossing, or a project whose
    product owner reads no blocked issue, waits for a person at once
    (`size-crossed-again`, `size-crossed-own`); so does a crossing at
    `workline:size/x-large`, the top tier (`size-crossed-top`).
  - **The work is kept**: only the state's label moves; a run whose role
    moved the issue keeps that move.
  - A record from before, its machine error of kind `size` or `budget`,
    is spent at once, for a person (`machine-error-spent`).
  - **A bug predicted past its size**, readied for its tests first
    (`split-after-tests` in the product owner's state): due to its
    developer, it is blocked the same way, no run crossed
    (`size-predicted`, `size-of: prediction`), so the product owner
    splits or grants it once its tests are written
    ([split on size](backlog-acts/split.md#split-on-size);
    [#1212](https://github.com/JN0V/workline/issues/1212)).
- **An issue's size tier** ([ADR-0064, §3](../adr/0064-size-is-a-target-not-a-wall.md#3-the-product-owner-decides-once-a-state-an-issue)):
  `workline:size/large` (×2), the product owner's grant, or
  `workline:size/x-large` (×4), a person's label only. Every step on the
  issue is given it (input `size-tier`, `size-tier` said), and every size
  target of its run multiplied: `context.budget`, `ai-max-tokens`,
  `code-lines-max`, `code-bytes-max`, `diff-lines-max`; `workline size`
  measures the issue against its normal times the tier.
- **The per-issue backstop**
  ([ADR-0064, §5](../adr/0064-size-is-a-target-not-a-wall.md#5-the-hard-backstop);
  [#1174](https://github.com/JN0V/workline/issues/1174)): the fresh
  tokens of every run on the issue (`spent`), all roles, its merge
  requests' runs included, against 4× its normal, the tier left out.
  - **Its normal**: its kind's, kept by `workline size`, else the seed
    (400k, 800k at effort High): a backstop of 1.6M, or 3.2M.
  - **`large` never raises it**; a person's `workline:size/x-large`
    lifts it to 16× the normal (6.4M at the seed). Nothing above it.
  - **Before a step**: reached, the step is not run; the issue moves to
    `workline:blocked`, the total and the backstop said
    (`backstop-crossed`; `backstop` in the record).
  - **After a run**: one that takes it there blocks it in the same step,
    wherever its role moved it on the line, over a size question to the
    product owner; its work, a pull request, is kept.
  - **A person's, with no time-out**: no product owner reads it again,
    news or a week (`backstop-waits`). A person puts `x-large` and moves
    it back by its label, or splits it: each new issue counts its own.
  - **A review round stops there too** (the panel of 2026-11-08): due
    on an issue at its backstop, the round is not run, by the line nor
    by `workline develop --merge-request`; the issue is blocked, the
    pull request kept open (`backstop-crossed` on the pull request).
  - **The liveness caps stay**: `max-hops`, `ai-max-turns`,
    `ai-max-seconds`, the machine-error budget, the rounds' `rounds`.
  - **At `x-large`'s 16×, nothing lifts it** (the same panel): a
    person splits the issue or finishes it by hand.
- **The code a role lacks is given**: the product owner naming the
  code an issue is about (`sources`) that its task did not give
  ([the code picker](code-picker.md)), its Scope left for that code, or
  the tester answering `needs: <paths> — why` is a needs-input of the
  engine (`needs-code`), never "no test possible". Due at once: the
  role runs again in the same step, given it (`state-given`; its task
  says so, the tester's and the product owner's get those files). Given
  them, it may name other files, none given yet (a folder's files led
  it to one): it runs once more, given those too
  ([#1410](https://github.com/JN0V/workline/issues/1410)). Said
  lacking again, the step itself moves the issue to
  `workline:blocked` for a person to name its files (`needs-spent`).
- **No code on the same input, a person's**: the developer's claim
  whose agent wrote no code, past `claim-hours` with the issue's body
  unchanged, is not let go to be taken again on that input: the line
  moves the issue to `workline:blocked`, the claim's label off, the
  reason written (`no-code-spent`,
  [#1120](https://github.com/JN0V/workline/issues/1120)). A changed
  body still lets the claim go at once.
- **Left free by the developer, due again**
  ([ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-02-a-ready-issue-left-free-wakes-the-developer-again)):
  a wake of the developer records when (`woke`) and the issues it held
  then (`held`): its claims an open pull request closes, a claim with no
  code holding nothing ([#952](https://github.com/JN0V/workline/issues/952)).
  A ready issue, assigned to nobody, is due again
  (`state-freed`) once its claim was put on since and let go, a hop
  more, so a take-and-let-go cycle ends at `max-hops`; or once an issue
  in `held` is held no more; or, passed over for an overlap, once its
  Scope names no file a claim held changes, once for the comment that said it
  (`unheld`)
  ([#998](https://github.com/JN0V/workline/issues/998)). A record from before `woke` is due once
  where the developer wrote its comment. Never on a clock: no loop.
- **A developer run lost is a machine error**: a run the line woke that
  ends with no verdict read from its post (`engine-error`, as when
  post dies; the run's result says `no-verdict`) reached nothing
  ([#953](https://github.com/JN0V/workline/issues/953)): of kind run,
  as above. A record from before typed outcomes keeps when it was lost
  (`lost`): read as one such error, run again, never written again.
- **Refine's record says the reviewer was asked** on the body (`asked`),
  where the reviewer is routed on `issue`. Acted on with no such mark,
  its spec still waiting on a review, the step is due again, once: a
  record from before the reviewer read it in refine's own run
  ([#840](https://github.com/JN0V/workline/issues/840)). Marked, no
  review come, the step is due once more, for the reviewer: it may have
  skipped the spec then
  ([#1070](https://github.com/JN0V/workline/issues/1070),
  `state-review-missing`; `review-again` in the record). After that, it
  stays acted: no loop. A spec waits on a review by the product owner's
  rule: its sections read with its state, text on top its Need.
- **A review answered by asking for a person's Scope** under the
  rule before
  [ADR-0060](../adr/0060-the-person-is-on-the-loop.md): the product
  owner's state shows a section it may rewrite now and was not given
  then (`backlog.Unrevised`). The step is due again, once a review
  (`revise`, the review's hash): the product owner rewrites it, the
  person no longer waited on.
- **A question past its delay**
  ([ADR-0060](../adr/0060-the-person-is-on-the-loop.md#1-questions-time-out-as-lazy-consensus)):
  in refine, acted on, the product owner's question to the reporter
  unanswered past `acts.ask.wait` — counted from the question's comment
  as the forge dates it (`created_at`), never the issue's last update —,
  or its drafts, all it lacks, unaccepted past it (from the forge's time
  of `workline:draft`). The step is due again, once
  (`state-timeout`; `timeout` in the record, `ask=N` or
  `drafts=<body>`), no agent until then: the product owner writes its
  assumption, or takes the drafts as written. An answer before the
  delay cancels it; `wait: never` (cautious) wakes none
  ([settings](../../roles/product-owner/docs/settings.md#how-long-a-question-waits)).
  Until then the line says it waits (`state-waits`): on its reporter,
  or a person's yes on its drafts, and the day the product owner takes
  its assumption, never silent
  ([#1082](https://github.com/JN0V/workline/issues/1082)). Its comment
  on the issue, written after the product owner's run, says the same:
  what waits, why, and what of a person ends it — a yes on its drafts
  or on what it proposes, or the reporter's answer
  ([#1195](https://github.com/JN0V/workline/issues/1195)).
  Its drafts' own body — the one its state's digest names —, or the
  reviewer's review of it with nothing important open, make the step
  due no sooner: the product owner could write nothing. A person's edit
  since (even one a night review read), a comment it has not read, a
  label put on again, or an important finding does
  ([#1127](https://github.com/JN0V/workline/issues/1127)).
  Its drafts with no review yet on that body, the reviewer on the line:
  the step is the reviewer's alone, the product owner not called
  (`state-review-pending`; `alone` in the record). In CI, the reviewer
  of the judge job read the body before the apply job wrote the drafts;
  its review lands, then the rule above holds it. A retry of that step
  after a machine error, and the once more of a review that did not
  come (`state-review-missing`), run the reviewer alone too; its errors
  and its block name the reviewer
  ([#1150](https://github.com/JN0V/workline/issues/1150)).
- **A person's comment, unread, due again**
  ([#1331](https://github.com/JN0V/workline/issues/1331)): in refine,
  acted on, a person's comment (`backlog.Counted`) the product owner's
  state says it has not read — an answer to the reviewer's question,
  or to its own. The step is due again, once for those comments
  (`state-comment`; `heard` in the record, their count when the line
  woke it): a run that leaves them unread is no loop. On the local
  forge, which names no author, a comment counts as a person of the
  project's. Its run's `ready` held for them, nothing written into the
  spec (`answer-held` in its state), the step is due once more
  (`state-comment-held`; `answer-held` in the record), never again on
  the same comments.
- **Left unrefined, due again**
  ([ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-an-issue-left-unrefined-is-woken-again)):
  in refine, acted on, the product owner's state shows it read the body
  as it is and left a required section missing, nothing waiting on a
  person — no question unanswered, no drafts, no proposal, not split,
  left nor set aside. The step is due again on the same body
  (`state-unrefined`; `retry` in the record), twice at most
  (`line.RefineRetries`); the third time the issue moves to
  `workline:blocked`, what it lacks said (`refine-capped`). A body
  changed starts the count again. Never on the same code
  ([#980](https://github.com/JN0V/workline/issues/980)): the record
  keeps the sources each wake gave (`given`); the state naming the same
  when its read left it unrefined, the issue is blocked at once, the
  code it was given said (`refine-capped`).
- **Never read, due again**
  ([#1041](https://github.com/JN0V/workline/issues/1041)): in refine,
  woken on this body, the product owner's state shows it never read the
  issue (no state, or nothing judged). The run read nothing, so the step
  is due again once a body (`state-unread-body`; `unread` in the record).
- **No test possible, unsettled**
  ([#1053](https://github.com/JN0V/workline/issues/1053)): in refine,
  acted on, the tester's "no test possible" still stands on the body and
  the product owner's state says it picked no proof, sent no retest, nor
  rewrote the text. The issue moves to `workline:blocked`, the tester's reason said
  (`no-test-unsettled`): never left in refine with nothing due.
- **Not for an outside failure**: a step that ended
  `blocked-external`, its agent out of reach, keeps no record: the next
  event or reconciler runs it again.
- **Each run's fresh tokens are counted**
  ([ADR-0064, step 2](../adr/0064-size-is-a-target-not-a-wall.md#1-normal-size-is-measured-per-project)):
  the record keeps, by role, the fresh tokens of each run its steps made
  that called an agent (`spent`), through every state; its comment says
  the issue's total so far. A step ended `blocked-external` keeps none.
  A run on a merge request (the reviewer, the inspector, the tester's
  guard and answers, the developer's review rounds) is counted on the
  issue it closes, when applied: once a run, its id kept (`counted`); an
  issue with no record counts none. `workline size` derives the project's normals from them
  ([commands](../commands.md#workline-size)). The issue's total is held
  to its backstop (below).
- **Judged with no write token** (`--no-apply`), the records are left in
  the line's result (`states`) and written by `workline apply --line`,
  after its runs applied.
- **`max-hops`** (8): each state step on an issue is a hop, counted in
  its record. The one past the cap is not run: the issue goes
  `workline:blocked`, the other states' labels off, the count said in
  that comment (`hops-capped`). A person moving it back to a state by
  its label starts its count again. A hop is a new run, not a handoff
  (`max-handoffs` caps those inside one run).
- **Progress starts the count again**
  ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-04-progress-starts-the-hop-count-again)):
  the record keeps the furthest state the issue reached (`reached`,
  along each state's `next`). A step into a state further still counts
  as hop 1: the cap stops a loop, never the step that ends it. Back in
  a state reached before, the count goes on.

### Blocked, read again

[ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-blocked-is-read-again):
a blocked issue wakes no role by its state; the reconciler, with no
agent, tells whether there is news on it, and wakes the product owner
only then. A person's comment or body edit on it asks the same at once,
by the event: the template's `Due` step on GitHub, the note or issue
webhook on GitLab ([#1414](https://github.com/JN0V/workline/issues/1414)).

- **`revisit`** (7): the days a blocked issue waits with nothing new
  before the product owner reads it again; `0`: never, a person moves
  it out. Shipped in `routing.default.yaml`, a project's
  `routing: revisit:` replacing it; a negative one is refused
  (`config-invalid`). Read only where the project routes
  the product owner on `issue`.
- **First seen blocked**: the line writes its record once
  (`state-blocked-seen`), no agent: the state it came from, when it saw
  it, its body's hash and its comments' — the base news is told
  against. A block the line makes (`hops-capped`, `tests-refused`)
  writes it at once.
- **Blocked by the machine, before typed outcomes**: a record with no
  version (`v`) whose reason is `tests-refused`'s or
  `no-test-unsettled`'s may hold a machine error read as a verdict. It
  is news: the product owner reads it again at once, not in a week;
  the record then written has a version, so once
  ([ADR-0063, migration](../adr/0063-every-open-issue-is-due-waiting-or-done.md#migration)).
- **The developer's ask for its Scope**, from before
  [#1228](https://github.com/JN0V/workline/issues/1228) made the Scope a hint: first seen blocked, the
  comment blocking it holding the developer's `workline:scope-ask`
  mark, the product owner is woken at once (`state-revisit`), told the
  files ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-fix-outside-the-scope-asks-the-product-owner)).
- **A run too big for its role**: the record's `size-of` names it; the
  product owner is woken at once (`state-revisit`), told the role, the
  record then written without it, so once
  ([the line of states](#the-line-of-states)).
- **The developer's code beyond the Scope** moves nothing: applied,
  listed on its pull request, the Scope a hint
  ([ADR-0065, change 2](../adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence)). An issue sent back to
  refine by an ask from before is read as before: the product owner is
  due on its own state, told the files.
- **The developer's red check with no fix**: blocked for a person, no mark; read again as any blocked issue,
  on news or after `revisit` days
  ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-red-check-with-no-fix-blocks-for-a-person)).
- **News**, each against that record: its body edited; a comment
  written or edited by anyone but the engine (one holding no
  `<!-- workline:` mark); an issue it waits on closed — the forge's
  relation or a "Blocked by" line ([depend](backlog-acts/depend.md)),
  each closing told once. None, and fewer than `revisit` days since it
  was seen or last read again: nothing run, no agent
  (`state-blocked-quiet`).
- **Then the product owner**, on that issue alone (`state-revisit`),
  told what is new and the state it came from (inputs `revisit`,
  `revisit-from`). It decides one, said in its one comment: `unblock`
  to that state or to refine, or `blocked`, the reason afresh, for a
  person when only one can lift it
  ([refine](backlog-acts/refine.md#a-blocked-issue-read-again)). Its
  text rewritten with neither, on a block the line made of its own
  read in refine, it goes back to refine in the same run
  ([#1206](https://github.com/JN0V/workline/issues/1206)).
- **Unblocked, never on the same input**
  ([#1157](https://github.com/JN0V/workline/issues/1157)): the record
  keeps the input of a role's last run in its state (`last-run`: body,
  labels, review, the code given, the comments the engine did not
  write) when its agent said it lacked code there, or the product owner
  read refine. Back in that state with nothing changed:
  - **Code said lacking once**: the role runs, given it
    (`state-given`); said lacking again, `workline:blocked` at once.
  - **Said lacking twice, or refine read and blocked there**:
    `workline:blocked` at once, no agent (`needs-spent`,
    `state-same-input`). A person's comment, a new body or label makes
    it a new input: the role runs. A block from a later state is news
    refine's read never saw: the product owner reads it again.
- **Counted like a step**: within `reconcile-max`; not a hop, its state
  the same. With no agent, nothing is judged; the record still moves
  the next reading `revisit` days on.
- **Cost**: the issue's comments, one call a blocked issue a reconcile;
  the open issues' listing, already the reconciler's, one call a hundred
  issues more on an issue's own run. No agent call without news or the
  week gone by.

### A parent, told as its parts close

[ADR-0029, amended](../adr/0029-a-parent-is-accepted-by-a-person-from-what-its-parts-delivered.md#amendment-2026-11-08-the-line-tells-a-parent-a-person-accepts-it-with-one-label):
a split's parent bears no state once a person lifts its block, or stays
blocked; its parts close in their own runs.

- **Read where its state wakes nobody**: no state label, or Blocked;
  in refine too, when its own step is not due. Only where the project
  routes the product owner on `issue`.
- **Its parts**: the forge's relation, read from the open issues'
  listing, and its `## Sub-issues` task list
  ([a parent and its parts](backlog-acts/parent.md)). A part not open
  is closed.
- **New input**: a part closed, or `workline:accepted` put on, since the
  line last read it (the record's `parts`). The product owner is woken
  on it with no agent (`state-parent`): it writes what the parts
  delivered and, all closed, proposes the closing.
- **A bug's, all its parts closed**, no person's yes on it: the tester
  runs first, with no agent, given `symptom`: its Draft's tests over the
  default branch (`state-parent-symptom`). Green, the product owner closes it alone
  past `cautious`
  ([a bug's parent](backlog-acts/parent.md); [#1212](https://github.com/JN0V/workline/issues/1212)).
- **Nothing new**: it waits, said in the job's summary
  (`state-parent-waits`): on its open parts, or on a person to accept
  it. A blocked parent is then read again as any blocked issue.
- **In CI**, a part closing starts no run on its parent: it is told at
  the next hourly reconcile.

### Room in the next state

[ADR-0059, decision 9](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#9-pull-and-room-in-the-next-state):
a step starts only when the state it moves the issue to has room.

- **`wip`**: a limit per state, the open issues it holds at most.
  Shipped: `ready: 5`, in `routing.default.yaml`; a project's
  `routing: wip:` is laid over it state by state, `0` lifting one. A
  limit on no state of the table is refused, and so are limits on every
  state of a ring of `next`: each would wait on the next, for good.
- **Full**: the step of a state whose `next` is at its limit does not
  start, nor asks an agent. The line says it once on the issue, in its
  comment (`state-waits`), the record kept as it was, with the state it
  waits on (`waits`). A second event while still full writes nothing.
- **Room again**: the next event on the issue, or the reconciler, finds
  the step due and starts it; its record, written then, drops the wait.
- **Room freed wakes the waiting**: the CI job that moved an issue asks
  the line with `--input room=true` (`route issue` or `route
  reconcile`, with `reconcile=list`). It lists the issues whose step
  waits on a state with a limit and room now, the one waiting longest
  first, and starts each one's run
  ([#960](https://github.com/JN0V/workline/issues/960)). A state still
  full costs no read of its waiting issues.
- **Counted** from the open issues' labels, as `StateOf` reads them.
  The reconciler counts a step it starts in its next state already, so
  the steps after it in the same run see it there.
- **What the developer can take**, alone, in its state ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-02-a-limit-counts-what-can-be-pulled)):
  an issue there that waits for its tests (named scenarios, no tester's
  Draft open, no proof kind recorded, where the project routes the
  tester), or on another open issue, or has parts, or whose Scope
  names a file a claim holding code changes (an open pull request
  closes it,
  [amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-claim-with-no-code-holds-no-scope);
  its files are that pull request's diff from the merge base, read
  with git,
  [amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-09-a-claim-holds-the-files-its-pull-request-changes);
  a diff git cannot read holds the claim's Scope and the files its pull
  request lists beyond it,
  [#1228](https://github.com/JN0V/workline/issues/1228); a Scope marked
  `Guessed:`, or whose words say no code was given when it was written,
  overlaps nothing,
  [#970](https://github.com/JN0V/workline/issues/970),
  [#998](https://github.com/JN0V/workline/issues/998)),
  is counted apart, and said in the wait's reason. Read only at the
  limit: the tester's open Drafts once a run, the comments of a ready
  issue with none, the open pull requests once when a claim is there.
- **To its tests**: a ready issue still waiting for them goes to the
  tester's state, by rule, no agent (`state-untested`), whether the
  developer was woken on it or not. Each move wakes the tester: the
  reconciler counts it in `reconcile-max`, after the steps due. How to
  undo it is said in the line's
  comment, the move kept in its record (`retest`). Put back by a person,
  it is said once and left (`state-untested-kept`, `retest-kept`); its
  developer step is not run then, which would only refuse it.
- **`--input reconcile=list`**: an issue whose wait is not said yet is
  listed due, its own run saying it; so is a ready issue to move to its
  tests, or to say left.

### Staleness

[ADR-0059, decision 7](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#7-staleness-per-state):
an issue left too long in a state, whose Scope names code gone, goes
back to refine.

- **`stale`**: an age limit per state, in days. Shipped: `ready: 30`,
  `tests: 14`; a project's `routing: stale:` is laid over them, `0`
  lifting one. Refine has none: it is where a stale issue goes.
- **The Scope first, no API call**: each word in backticks under
  `## Scope` that names a path (a `/` or a file's extension) or code (a
  capital, `_`, a dot or `()`) is looked up in the repository: the
  path in `git ls-files`, the name, its last part, as a word in any
  file but Markdown (decision records keep old names). Labels, settings,
  flags and plain words are not checked.
- **Then the age**, only for an issue whose Scope names something gone:
  from when the forge says the state's label was put on (GitHub's
  timeline, GitLab's label events), never the last update.
- **Past the limit**: the issue goes back to refine, its other states'
  labels off, its card to the refine column, the reason said in the
  line's comment (`state-stale`), which wakes the product owner. No
  agent asked; outside `reconcile-max`.
- **Undone by a person** (the label put back): the next time it is
  stale, the line says so once (`state-stale-kept`) and leaves it
  ([ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-back-to-refine-needs-no-yes)).
  The record keeps the move (`stale`) while the issue stays in that
  state; a step of another state drops it.
- **Checked** by the reconciler for every open issue in a state with a
  limit, whether a role the project routes wakes it or not, and by an
  issue's own run before its step. `--input reconcile=list` lists it
  due, its own run moving it.
- **Cost**: no API call for an issue whose Scope resolves; for one
  naming something gone, its label events and comments, a call or two
  each.

### A bug not reproduced

[ADR-0060, decision 4](../adr/0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind):
a bug whose tests passed on the base waits in its tests' state, marked
`workline:needs-repro`, its reporter asked by the tester
([the hand-over](../../roles/tester/docs/hand-over.md#a-bug-not-reproduced)).
The line reads it with no agent:

- **Its body edited** since the tester's run: the tester runs again.
- **A person's answer** after the mark (its reporter, or the
  project's): back to refine, the mark off, said (`repro-answered`);
  the product owner reads it.
- **No answer** within the product owner's delay (`acts.ask.wait`),
  counted from when the forge says the mark was put on: Blocked, its
  reason "not reproduced", the mark off (`not-reproduced`). Its kind
  stays a bug.
- **Else** it waits, said with the time it will be blocked
  (`repro-waits`); at `cautious` (`never`), until it is answered.

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
  of the head, detached, the clone's config laid in
  it, removed after; counted in `reconcile-max`; none with `--no-apply`.
  A person's comment is one with no workline mark.
- **An issue whose step is its round**: in the developer's state, its
  own pull request open from `workline/developer/<n>`, an issue is no
  new claim (`state-round`), as one back to ready from a Scope ask.
  `route issue` on it lists that round in `reviews` when due, or says
  its command; nothing written. The reconciler leaves the issue, and
  lists the round with the others
  ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-04-a-widened-scope-resumes-its-round-at-once)).
- **An issue that cannot be read** is said (`state-unread`) and the
  others walked; the forge out of reach stops the run.
- **Run twice in a row**, the second does nothing more: the first left
  its records.
- **`--input last=<RFC 3339>`**: when it last ran, as the CI keeps it.
  Over two hours ago, its summary says how late it is
  (`reconcile-late`;
  [#789](https://github.com/JN0V/workline/issues/789)).

## Where state lives

On a forge, the state of a work item is a label (`workline:ready`,
`workline:developing`…, [the line of states](#the-line-of-states)),
readable by humans and by any CI. Without a forge, it
is the `state:` line of the item's file in `.workline/work/<id>.md`, whose
`## Need`, `## Validation` and `## Scope` sections
`workline item ready <id>` checks before moving it — `### ` headings too,
as an issue form writes them, a field left `_No response_` empty, a
footer after the last section left out
([a body's footer](backlog-acts/refine.md#a-bodys-footer)). On a forge,
a person moves an issue by its label too; the engine's moves are one
transition at a time, each said in the line's comment on the issue. In
a file, *built so far: the move to `ready`, and its check; not logged.*

## The reviewer, opt-in

The reviewer
([ADR-0020](../adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)) is in no shipped line: a project adds it to its
`merge-request` line, by hand or with `workline init --review`, and runs
it on a machine before a push with `workline review`. On a merge request it
reviews the commits not reviewed yet, one lens a push, none on a draft;
with `--input lenses=all`, every lens on the whole change, commits
reviewed before too.
The templates pass it once, when the merge request becomes ready
([#129](https://github.com/JN0V/workline/issues/129)): the event says so on GitHub (`opened` not a draft,
`ready_for_review`); on GitLab, where marking ready starts no pipeline,
the next pipeline tells it by the merge request's draft state and its
"marked this merge request as **ready**" note, newer than the
reviewer's comment. It never moves an item: `review` stays the
machine's part, `validation` the person's. Its `spec` event reads a spec
before it is built — a file, or an issue by its number (`--input
issue=<n>`); on a machine, `workline review --spec` or `--issue`
([#128](https://github.com/JN0V/workline/issues/128)).

After the product owner in a line, the reviewer reads the specs it
refined ([#128](https://github.com/JN0V/workline/issues/128)): on `schedule`, up to `specs-per-run` issues
a run ([#433](https://github.com/JN0V/workline/issues/433)),
the findings on each in a comment on it; the product owner's `ready`
waits until none important is open
([a spec read before ready](backlog-acts/spec-review.md)). Opt-in, as on
merge requests:

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer]}
```

The tester writes the acceptance tests of an issue waiting for them
(`needs-tests`, or `ready` with none) on `schedule`, one issue a run,
and on `issue` the one its state names: on `schedule`, the first in the
backlog's order whose Validation names scenarios and whose Draft is not
open
([ADR-0055](../adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md#amendment-2026-10-28-step-4-as-built)).
It needs `--open-merge-request`; opt-in, after the product owner and the
reviewer:

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer, tester]}
```

The developer takes a `ready` issue to a pull request
([ADR-0054](../adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md)):
on `issue`, the one CI names (`--target issue:<n>`) when it is labelled
ready; on `schedule`, the first nobody took, one a run. It needs
`--open-merge-request`; opt-in
([the triggers](../../roles/developer/docs/triggers.md)):

```yaml
routing:
  events: {issue: [developer], schedule: [documentalist, developer]}
```

## Not in this version

Which
act runs at which moment, decided: [ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md).
