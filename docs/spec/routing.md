---
sources: [internal/line, internal/routing, internal/work, routing.default.yaml]
checked: f34d3dbe
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

**Ready** means four things are written, even in one line each:

| Field | Question |
|---|---|
| Need | Which user need does this answer — who needs what, and why? |
| Verification | How will the machine prove it works: which tests, checks, thresholds? |
| Validation | How will a person accept it: who, looking at what? |
| Scope | What part of the project does it touch? |

Verification is what the line checks on its own; validation is what a person
decides at the end. Writing both before starting is what keeps the machine
from grading its own work. The engine checks that the four are present and
not empty; it does not judge their quality. A section is the text under its
`## ` or `### ` heading (`work.Sections`): a heading written twice holds
both texts, and a thematic break ending a section — the line between the
person's part and the builder's — is not its text. Each is read by its
meaning, through one table (`work.Headings`,
[ADR-0039](../adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)):
a meaning's first heading is the one written, every one is read; today
each has one, the names above. Other headings, an
example or the steps to reproduce, are kept and never required
([writing an issue](../../roles/product-owner/docs/writing.md)). A person moves an item to
`ready`, or the product owner does once Need and Validation are a person's
(docs/spec/backlog-acts.md, "Refining to ready"). Development roles only take `ready` items, and
the item's **scope** becomes the run's scope (see "Stay on the task" in the
role contract). *Not built yet: the scope is given with `--scope`; no role
takes items yet.*

`review` is the machine's part: reviewers and gates run the verification.
`validation` is the person's part: the item reaches `done` only when the person
named in *Validation* accepts it. A rejection sends it back to `in-progress`
with the reason, or to `to-refine` if the need itself was wrong.

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
it; none by default, never the docs (ADR-0017, ADR-0051). A release tool's
pull request is a merge request as any other; each role is told its
branch (`release.branches`). A role routed on `release` that does not run
there, the documentalist in a routing written before ADR-0051, is
skipped, said (`step-skipped`, a warning). A routing that still
names the retired `release-manager` is refused, saying which tool to use.

On the release — `release`, or `merge-request` from a release tool's
branch — the line also reads what holds it on the forge, once, beside its
steps, stopping none of them; its verdict takes the worst (ADR-0051,
[what holds a release](../release-holds.md)): each open issue labelled
`workline:hold-release` (`release-held`, by its number); the checks on the
tip of the branch the merge request goes into, the default branch on
`release` — a red one is `release-base-red`, one not done
`release-base-pending`, both blocks; none at all, `release-base-unchecked`,
a warning. On GitHub only the runs a push started count, with other apps'
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
  later reader can stand apart from them (ADR-0005). A line given a range
  whose last commits carry one judges the range as it stood before them,
  each left for its parent, a merge of the base (`workline follow`,
  ADR-0041) for its first: what the engine wrote never wakes the line on
  itself, and a person's commit that blocked still blocks, never turned
  green unjudged. With nothing of a person's left in the range, only the
  committer runs; it checks every commit, the engine's too. A trailer is not
  taken on trust: a merge is the engine's only when it merges the range's
  base, or a commit of its history, and its tree is exactly what
  workline's own merge of its two parents gives; another commit only when
  it changes doc pages alone — Markdown, never a role's facet (in
  `roles/<name>/` only its README.md and `docs/`) nor under a tool's
  folder, never a file steering agents (AGENTS.md, CLAUDE.md…); else it is
  judged as a person's (ADR-0042). `Workline-Role: committer` marks a
  person's commit whose message the committer rewrote (ADR-0046): never
  the engine's.

### On a forge: judge, then apply

`workline route <event> --no-apply` runs the whole line in a job that holds no
write token, with the forge and target of the merge request (`--forge`,
`--target`; `--input` and `--scope` go to every step). Each step judges the tree
as it is — no step sees what an earlier one proposed — and nothing is applied.
The result lists the runs to apply, in the line's order (`pending`) — a
step that went round, proposing a merge request a round, gives one run a
round (docs/spec/role-contract.md, "Again");
`workline apply --line <result>` applies them in the job that holds the token
and no AI key, and stops at the first that does not pass, saying the runs
left after it (`runs-not-applied`) and the command that resumes them. A role
keeping a backlog decides its acts in the judging job, reading the forge
only, and keeps that plan for the apply (docs/spec/backlog-acts.md, "On the
issue").

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
merge requests or a proposal its role never makes (`run-refused`);
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
  `--open-merge-request`, `--push-to-merge-request`, `--verdict`, the
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

## Where state lives

On a forge, the state of a work item is a label (`workline:ready`,
`workline:in-progress`…), readable by humans and by any CI. Without a forge, it
is the `state:` line of the item's file in `.workline/work/<id>.md`, whose
`## Need`, `## Verification`, `## Validation` and `## Scope` sections
`workline item ready <id>` checks before moving it — `### ` headings too,
as an issue form writes them, a field left `_No response_` empty, a
footer after the last section left out (docs/spec/backlog-acts.md, "A
body's footer"). Either way, a state changes only through the
engine, one transition at a time, and each transition is logged. *Built so far:
the move to `ready`, and its check; transitions are not logged yet.*

## The reviewer, opt-in

The reviewer (ADR-0020) is in no shipped line: a project adds it to its
`merge-request` line, by hand or with `workline init --review`, and runs
it on a machine before a push with `workline review`. On a merge request it
reviews the commits not reviewed yet, one lens a push, none on a draft;
with `--input lenses=all`, every lens on the whole change, commits
reviewed before too.
The templates pass it once, when the merge request becomes ready
(#129): the event says so on GitHub (`opened` not a draft,
`ready_for_review`); on GitLab, where marking ready starts no pipeline,
the next pipeline tells it by the merge request's draft state and its
"marked this merge request as **ready**" note, newer than the
reviewer's comment. It never moves an item: `review` stays the
machine's part, `validation` the person's. Its `spec` event reads a spec
before it is built — a file, or an issue by its number (`--input
issue=<n>`); on a machine, `workline review --spec` or `--issue` (#128).

After the product owner in a line, the reviewer reads the specs it
refined (#128): on `schedule`, up to `specs-per-run` issues a run (#433),
the findings on each in a comment on it; the product owner's `ready`
waits until none important is open
(docs/spec/backlog-acts.md, "A spec read before ready"). Opt-in, as on
merge requests:

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer]}
```

## Not in this version

The developer and tester roles, which will consume `ready` items. Which
act runs at which moment, decided: [ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md).
