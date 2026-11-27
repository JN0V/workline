# ADR-0075: Urgent work first, and a merge queue of our own

- **Status:** proposed, until the maintainer merges it; slice 2 built.
  Built by [its slices](#slices), one issue each. [The tick's
  contract](#open-the-ticks-contract) is not decided
- **Date:** 2026-11-27
- **Decided by:** the maintainer, on 2026-11-27, on an analysis of the
  code and of three repositories' forges (workline, DomoticsCore,
  TraceGlass); code read at `4b8383b25` on `main`
- **Builds on:** [ADR-0050] §4, [ADR-0069]; principles 10, 12, 13, 14
  ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends, once built:**
  - [ADR-0040 §(c)](0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md#c-the-order-the-value-vs-effort-grid)
    (`internal/backlog/order.go:240-244`): Urgent goes before the
    milestone (6); in the product owner's reading, an effort Low issue
    crosses a priority (2). There "quick win" is High·Low; here, any
    priority below 1 with effort Low.
  - [ADR-0059] decision 9: a full Ready stops every step; one Urgent
    passes (4). Its [2026-11-03 amendment](0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-the-one-waiting-longest-first-room-wakes-the-waiting),
    the one waiting longest first: priority 1 before it, in the
    reconcile and the room's listing alike (5).
  - [ADR-0042]'s [2026-10-21 amendment](0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-10-21-adr-0051-no-up-to-date-no-one-at-a-time):
    "Gone: the line's order, `fork-behind`, the first in line's update";
    decision 7 brings an ordered queue back, where the forge holds a
    request behind.
  - [ADR-0067]: the base merged into every accepted request behind
    becomes into the head alone (7).
- **Refines:** [ADR-0051] cut 2: strict stays off where it is; where a
  project keeps it on, its rounds are N, not about N²/2 (7).
- **Not in conflict:**
  - [ADR-0069]: decision 3 adds a reason the tick lists a backlog run;
    the tick still decides, nothing moves from an event.
  - [ADR-0050] §4: a native queue's run is untouched, gates only; our
    queue chooses which head is updated, and that head's run is the
    line's usual one.
- **Research:** [merging.md](../research/merging.md#who-merges-elsewhere);
  Kanban's *expedite class of service*, GitHub's *merge queue*

## Context

### Priorities are read late, or not at all

- **The levels**: `workline:priority/1` to `/4`, Urgent to Low, or a
  project's [own names](../../roles/product-owner/docs/priority-and-effort.md#other-scales-read-by-name);
  a person sets them (`order` off, `roles/product-owner/role.yaml:126`).
- **Where an Urgent issue waits today**, step by step:

| Step | Reads the priority? | Where |
|---|---|---|
| A new issue, no state label | nothing wakes it: the reconcile lists every open issue, then skips those no state wakes | `internal/line/states.go:981-995` |
| A backlog run due | no: Ready below its target **and** Ready moved | `internal/line/tick.go:18-30`, `internal/line/backlog_due.go:24-68` |
| The product owner's reading order | no: answered, then changed, then new, … | `roles/product-owner/code/productowner.go:779-796` |
| A step whose next state is full | no: it waits | `internal/line/wip.go:197-238` |
| The reconcile past `reconcile-max` | no: longest waiting first | `internal/line/room.go:56-63` |
| The developer's pick | yes, after the milestone | `internal/backlog/order.go:245-265`, `roles/developer/code/developer.go:353` |
| A claim on overlapping files | yes: a lower priority holds no Urgent | `roles/developer/code/developer.go:316-331` |
| The merge (`follow`) | no: the forge's listing order | `internal/engine/merge.go:935-945` |

- **The nightly is a backlog run** (`internal/tick/tick.go:70`), at
  02:17 on workline (`.workline/config.yaml:195`); Ready holds 10 there
  (`:187`), 5 shipped.

### Strict rulesets update every pull request at once

- **`strict`**: off on workline ([ADR-0051] cut 2), on for DomoticsCore
  and TraceGlass (the maintainer, 2026-11-25). All three are a user's:
  no [GitHub merge queue](../github/merge-queue.md).
- **`follow` walks every open pull request**
  (`internal/engine/merge.go:935-945`) and merges the base into each
  accepted one the forge holds behind, its checks green, in the same
  run (`internal/engine/merge.go:1446-1456` →
  `internal/engine/uptodate.go:37-81`). Only `workline/developer/`
  branches are updated (`internal/engine/uptodate.go:38`).
- **On DomoticsCore since 11-20**: 7 of 40 merged had the base merged
  in, 15 times in all, one 5 times in a day: about N²/2 CI rounds.
- **The yes survives workline's own merge of the base**: it is no push
  ([one head](../merge-policy-one-head.md#worklines-own-commit-is-no-push)).

## Decision

**Priority 1 is the line's expedite class: it is read first at every
step, and one at a time may pass a full Ready. Where the forge holds an
accepted pull request behind, the engine updates only the head of its own
queue, in priority order.**

### 1. Urgent is priority 1 only

- **Urgent**: `workline:priority/1`, or a project's value read as Urgent.
  Fixed in code: no setting. Priority 2 and below: the normal order.

### 2. The product owner reads by priority, quick wins beside

- **Woken by the tick**, its backlog run reads priority 1 first, then 2,
  3, 4, then none.
- **A quick win** may come before or beside a long issue of a higher
  priority: value over size.
  - A quick win is an issue whose effort reads as the smallest value,
    Low ([the effort scale](../../roles/product-owner/docs/priority-and-effort.md#other-scales-read-by-name)).
  - A long issue is one whose effort reads High.
  - Never before a priority-1 issue.
- **Today's groups** (answered, changed, new, …) break ties.

### 3. The tick wakes the product owner for an Urgent issue

- **A backlog run is due** when a priority-1 issue has no state label, or
  sits in `to-refine`, and no backlog run started since its priority
  label was put on. Ready full or not.

### 4. One Urgent may pass a full Ready

- **A step of a priority-1 issue into Ready starts** though Ready is
  full, when Ready holds no other priority-1 issue.
- **Other states' limits** hold as today.
- **Said** as today's wait is: a person sees the limit passed.

### 5. The reconcile starts priority 1 first

- Past `reconcile-max`, priority-1 steps go first, then the longest
  waiting, as today; the room's listing the same (`startDue`, shared).

### 6. Urgent before the milestone

- **The developer's order**: priority 1 first; then the milestone, the
  priority, the effort, the number, as today.

### 7. A merge queue of our own

- **Acts only** when the forge holds an accepted pull request behind:
  strict on, no native queue. No setting.
- **Each `follow` run**:
  1. orders the accepted pull requests by the priority of the issue each
     closes, else its own label, then by when auto-merge was turned on;
  2. takes as head the first one held by nothing else: red checks, a
     conflict, an open conversation, a person's comment unanswered;
  3. merges the base into the head alone, with today's merge
     (`internal/engine/uptodate.go`); the others are said "queued,
     position k" in the run's summary: no comment, no marker;
  4. the forge merges the head; the push to `main` runs `follow`; the
     next one becomes the head.
- **A head that fails** moves on: red checks and a conflict take
  today's rounds; checks pending past the `verdict` budget
  (`defaults/routing.yaml:79`) are said and skipped.
- **A person's branch** is never updated, as today: passed over, said.
- **A native queue or train**: left to it. No organisation for now.

### 8. The code budget's exception

- **Until 1.0, a fix is net ≤ 0 lines of non-test Go** (2026-11-25).
- **Granted for decisions 1 to 7 alone**: about 80 to 120 lines in all.
- **Nothing else rides on it**: no new setting, rule code nor marker;
  any other change keeps net ≤ 0, or asks.

## Open: the tick's contract

Not decided; the maintainer settles it after 2026-11-28.

- **Built**: `workline tick` lists what is due (`internal/tick/tick.go:64-76`),
  reads past runs on GitHub alone ([tick.md](../tick.md)).
- **The questions**:
  - dispatch on for DomoticsCore and TraceGlass;
  - on GitHub, the loop as the beat, its cron a restart only, the
    per-task crons dropped from the template;
  - on GitLab and a person's machine, how runs are remembered and what
    beats ([ADR-0069] §5, §7, not built);
  - a check that says when the beat stopped.
- **Until then**, decision 3 runs on today's tick.

## Slices

Each buildable alone, in this order:

1. [#2325](https://github.com/JN0V/workline/issues/2325): the merge queue (7).
2. [#2322](https://github.com/JN0V/workline/issues/2322): the tick (3); built (`internal/line/tick.go`, `BacklogDue`).
3. [#2323](https://github.com/JN0V/workline/issues/2323): a full Ready and the reconcile (4, 5).
4. [#2321](https://github.com/JN0V/workline/issues/2321): the reading order (2).
5. [#2324](https://github.com/JN0V/workline/issues/2324): the developer's order (6).

## Not chosen

- **A person runs the step by hand**: the person becomes the fast lane ([principle 14](../PRINCIPLES.md)).
- **React to `issues: opened`.** Against [ADR-0069]'s 2026-11-17 amendment.
- **Urgent and High expedited.** The expedite lane would not be small.
- **Strict off on DomoticsCore and TraceGlass.** Against 2026-11-25.
- **Native queues only.** An organisation or Premium; first in, first out.

## Not verified

- A real Urgent issue's wait from end to end.
- CI time per round on DomoticsCore: medians 0.7 to 4.7 min per
  workflow, the longest 17.8 min; a "25 minutes" figure is not measured.
- GitHub's `enqueuePullRequest(jump)`, GitLab's "merge immediately".
- Every line count above is an estimate.

[ADR-0040]: 0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[ADR-0059]: 0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md
[ADR-0067]: 0067-a-role-request-behind-main-merges-main-in.md
[ADR-0069]: 0069-one-wake-every-five-minutes-starts-what-is-due.md
