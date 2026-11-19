---
sources: [internal/line/backstop.go#passed, internal/line/freed.go#freed, internal/line/freed.go#woken, internal/line/freed.go#later, internal/line/repro.go#repro, internal/line/revisit.go#revisit, internal/line/size.go#Past.said, internal/line/stale.go#stale, internal/line/states.go#never, internal/line/states.go#blocking, internal/line/states.go#reconcile, internal/line/wip.go#untested, internal/line/wip.go#apart, internal/line/wip.go#waits, internal/line/wip.go#waiting, internal/line/wip.go#retest, internal/routing/adopt.go#config, internal/routing/adopt.go#config.write, internal/work/headings.go#Scope, internal/work/scenarios.go#apart, internal/work/scope.go#Guessed, internal/work/work.go#parts, internal/work/work.go#check, defaults/routing.yaml]
checked: a19e8cb0
verified: agent:claude-code
---
# Routing — blocked, parents, room and staleness

Part of [Routing](routing.md): what the line does with a blocked issue, a parent, the room in the next state, a stale issue and a bug not reproduced, after [the line of states](routing-states.md#the-line-of-states).

### Blocked, read again

[ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-blocked-is-read-again):
a blocked issue wakes no role by its state; the reconciler, with no
agent, tells whether there is news on it, and wakes the product owner
only then. A person's comment or body edit on it asks the same at once,
by the event: the template's `Due` step on GitHub, the note or issue
webhook on GitLab ([#1414](https://github.com/JN0V/workline/issues/1414)).

- **`revisit`** (7): the days a blocked issue waits with nothing new
  before the product owner reads it again; `0`: never, a person moves
  it out. Shipped in `defaults/routing.yaml`, a project's
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
  ([the line of states](routing-states.md#the-line-of-states)).
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
  Shipped: `ready: 5`, in `defaults/routing.yaml`; a project's
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
([the hand-over](../../roles/tester/docs/proof-kinds.md#a-bug-not-reproduced)).
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
