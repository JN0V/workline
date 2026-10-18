# ADR-0040: Priority and effort are the forge's; the product owner estimates both, a person checks

- **Status:** proposed — built in the order below; the review's two
  questions answered by the maintainer on 2026-10-11 (Urgent, the default)
- **Date:** 2026-10-11
- **Supersedes, once built:** the labels `workline:priority/1` to `/4` of
  [ADR-0018](0018-the-product-owner.md) ("Ordering"); its order rule
  stands, effort added
- **Builds on:** [ADR-0018](0018-the-product-owner.md) (the order derived,
  never stored), [ADR-0026](0026-the-product-owners-autonomy-is-a-level.md)
  (levels), [ADR-0031](0031-the-report-opens-with-what-is-next-and-what-is-stuck.md)
  (Next), [ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
  (proposals on the issue, a person's undo, `order` off by default);
  principles 1, 4, 12, 13
- **Research:** [priority-and-weight.md](../research/priority-and-weight.md);
  worked examples: [priority and effort](../../roles/product-owner/docs/priority-and-effort.md)

## Context

- **Today** the `order` act, off by default, sets workline's own labels
  `workline:priority/1` to `/4`. On workline, 18 open issues carry one, all
  set by the role; 8 of 9 `ready` issues have none: Next goes by number.
- **The forges have their own**: a GitHub organisation's Priority and
  Effort issue fields, GitLab's ranked `priority::` labels and Premium's
  `weight`, projects' own label families. Principle 13: use them.
- **The maintainer decided, on 2026-10-11**: the forges' mechanisms, never
  workline's labels; an effort beside the priority, the development
  complexity; the role estimates both with the agent, as developers
  already ask an agent to; a person only checks.

## Decision

### (a) Two axes, the forge's levels, read by name

| Axis | Levels, in the order's direction | None |
|---|---|---|
| Priority | Urgent, High, Medium, Low | after Low |
| Effort | Low, Medium, High | read as Medium |

- **By name, never by an option's position** (jn0v-lab's Effort lists High
  first): GitHub's default options; a project's `P0`–`P3` or `priority::1`
  to `::4`, the lowest number Urgent; words such as critical, important,
  small, large; `effort1` to `3` (hours, days, weeks)
  ([the names read](../../roles/product-owner/docs/priority-and-effort.md#other-scales-read-by-name)).
  A name no table knows is no value, said once; a setting maps it.
- **GitLab's `weight`**, an integer: read 0–1 as Low, 2–3 Medium, 4 or more
  High; written 1, 3 or 5. These sit on both common point scales,
  Fibonacci (1, 2, 3, 5, 8) and 1 to 5, so a person's later estimate lands
  beside them, the bands cut both in three, and a milestone's sum still
  reads as complexity. Another scale sets its bands. Never on a parent:
  GitLab sums its children.

### (b) Where the values live: one source per axis, detected

The engine detects, each run and with no agent, the first that holds:

1. **The setting**, per axis, `auto` by default: a field, GitLab's
   `weight`, or a label family. One the forge lacks stops the run, said.
2. **A native field**: on GitHub, a single-select Priority or Effort that
   GraphQL's `repository.issueFields` lists (null on a user's repository);
   on GitLab, effort only, an issue's `weight` key (Premium, Ultimate).
3. **The project's own label family**: `P0`…, `P-*`, `priority/*`,
   `Priority/*`, `priority::*`, `effort*`; two values or more, one on an
   open issue. Never `size/*`: Kubernetes' counts lines changed. Then
   workline's own `workline:priority/*`, one on an open issue enough,
   until renamed (f).
4. **Else the forge's words**, created at the first act:

| Forge | Priority | Effort |
|---|---|---|
| GitHub, a user's repository | `priority/urgent`, `/high`, `/medium`, `/low` | `effort/low`, `/medium`, `/high` |
| GitLab Free | `priority::1` to `::4`, ranked 1 to 4 as prioritized labels | `effort::low`, `::medium`, `::high` |
| GitLab Premium, Ultimate | the same | `weight` |

- **GitHub's names are the fields' options**: a repository moved to an
  organisation maps 1:1. **GitLab's ranking** makes its own "Label
  priority" sort follow the role's, on Free. **Never** `time_estimate`
  (time, not complexity; it feeds time reports), nor a label in the
  `workline::` scope (on Premium it would knock `workline::proposed` off).
- **Never mirrored.** A field and a label that disagree on an issue:
  nothing written there, said once. The summary says where each axis was
  found.

### (c) The order: the value-vs-effort grid

- **Within a milestone**: an issue's blockers first, as today
  (`backlog.Order`); then the priority, Urgent first; then the effort, Low
  first; then the lowest number (`backlog.Less`).
- **No score**: a sum or a ratio lets a noisy effort carry an issue across
  priority levels.

| Priority | Effort Low | Effort Medium or none | Effort High |
|---|---|---|---|
| Urgent | 1 | 2 | 3 |
| High | 4, quick win | 5 | 6, big bet |
| Medium | 7 | 8 | 9 |
| Low | 10, fill-in | 11 | 12, time sink |
| none | 13 | 14 | 15 |

- **No effort reads as Medium**: an issue not estimated neither jumps a
  quick win nor falls behind a big bet. No agent: the order still holds.
- **Next shows each issue's cell**, "High · Effort Low → quick win", says a
  value missing, and counts the issues missing one
  ([a night's Next](../../roles/product-owner/docs/priority-and-effort.md#next)).

### (d) The product owner estimates both; a person checks

- **One intention**, `order: {issue, priority, effort, why}`; **two kinds**
  for mode, cap and undo: `order` (the priority) and `effort`.
- **Priority** from the issue and its code: data lost, a regression, others
  blocked first; what can wait last.
- **Effort only for an issue whose code this run read**, its Scope's files
  given; else dropped (`effort-unread`). Anchors: Low, hours; Medium,
  days; High, weeks.

| Level | Priority and effort |
|---|---|
| `cautious` | proposed on the issue ([ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)); `workline:accepted` sets them |
| `normal`, the default level | set where empty, Urgent included, 10 a run per axis |
| `enterprising` | set where empty, Urgent included, 20 a run per axis |

- **Off by default**, as `order` is today: on for workline once built; on
  at `normal` for every project by an amendment, once the measures below
  pass on workline and the jn0v-lab repository.
- **Urgent is set like any level**, with one more guard, no ban: the
  night's summary lists each Urgent the role set, for a person to see at
  once.

- **Never overwritten**: only an empty value is written; one the role did
  not set is a person's or another tool's, kept. One a person changed or
  cleared after the role set it is never written again on that issue:
  [ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)'s
  undo, that axis proposed there, everywhere past `undone-max`.
- **Set once**: never re-estimated, so Next does not move on a night its
  issues did not; `moved-percent-max` bounds a first backfill too.
- **Stuck estimates flagged**: past 10 values set on an axis, one value
  over 60% of them (all "Medium", too many "Urgent" or "High") is said.

### (e) Writes, and what a public page must not show

- **Read again before writing**: the applying job drops an act whose value
  changed since the plan (today `order` does not). Only real changes, each
  read back; caps per run and per axis.
- **No notification of workline's own**: field and label writes notify
  nobody on GitHub; GitLab mails a person subscribed to a label the role
  adds, the forge's rule, said in the role's docs. The role's comment is
  edited in place.
- **GitHub's fields may be "organization members only"**: on a public
  repository, no value in the role's comment, state or CI summary; a
  visibility not read counts as hidden. The state keeps the day the role
  set an axis: a later change in the issue's timeline is a person's.
- **Never read as empty**: until the live try shows a token reads
  members-only values, that token writes nothing on that axis, said.

### (f) workline's labels, and the live try

- **Renamed in place**: `workline:priority/2`, `/3`, `/4` (`/1` was never
  created) become `priority/high`, `/medium`, `/low`, three silent calls;
  the state's `priority: 1` to `4` reads as Urgent to Low: still the
  role's own.
- **Live, on a new public, non-fork repository of jn0v-lab**, created when
  building: jn0v-lab/workline-sandbox is a fork, and `gh` in a fork's clone
  resolves to the parent. Tried with an App's token, `GITHUB_TOKEN` and a
  person's: members-only values, writes read back, notifications.

## Parked, and why

- **GitHub's `suggest`** (a value held for approval): a preview, still
  rolling out; the proposal on the issue works on every forge.
- **Gitea and Forgejo**: the product owner cannot run there until
  [#65](https://github.com/JN0V/workline/issues/65).
- **A ratio score** ([research](../research/priority-and-weight.md#combining-priority-and-effort)):
  a noisy effort would jump priority levels.
- **GitHub Projects fields**: a project's, not the issue's; a user's
  project needs a classic token.
- **Writing the forge's manual rank**: [#292](https://github.com/JN0V/workline/issues/292)
  stays parked; GitLab's "Label priority" sort already follows.

## Built in small pull requests, conformance first

| Step | What | Proved by |
|---|---|---|
| 1 | **Levels by name**: one scheme reads `workline:priority/*`, a field, a family; no behaviour change | cases: Effort listing High first sorts High last; `P0`, `priority::1`, weight 4 read; existing cases unchanged |
| 2 | **Detection and labels**: settings, families, words created; a disagreement writes nothing. Built behind the settings `priority` and `effort`: unset, workline's own labels as before | fake forge cases; the GitLab sandbox (Free) |
| 3 | **Effort and the grid**: the `effort` kind; `Less`; Next's cell; the 60% flag; read again before applying. The kind off by default; set once and Next's cell with the settings of step 2 only | cases: a person's value kept per axis; cleared, never refilled; changed since the plan, dropped; no code, no effort |
| 4 | **GitHub fields**, members-only values never shown | fake forge cases; live on jn0v-lab, each token, the API version header |
| 5 | **GitLab's `weight`**: the bands, never a parent | cases; live once a Premium project is at hand |
| 6 | **workline's labels renamed**, the acts on for workline; `auto` the settings' default | the measures below, four weeks; then the amendment turning them on |

## How it is measured

- **Corrected**: the role's values a person changed or cleared, per axis
  and direction (raised each time: it underestimates); 20% or fewer past
  30 values. **Spread**: none over 60%.
- **Next**: the share of issues a person started that were in the night's
  Next, from the forge's trail.
- **Blind**: the maintainer ranks 10 ready issues, values hidden; at least
  2 of the engine's first 3 among his first 3.
