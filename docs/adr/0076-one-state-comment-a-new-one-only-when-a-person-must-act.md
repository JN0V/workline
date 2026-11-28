# ADR-0076: One state comment an item, a new one only when a person must act

- **Status:** proposed, until the maintainer merges it; nothing built.
  Built by [its slices](#slices), one issue each
- **Date:** 2026-11-28
- **Decided by:** the maintainer, on 2026-11-28, before 1.0, with an
  exception to the code budget ([below](#the-budget-exception)); code
  read at `08ad89933` on `main`
- **Builds on:** [ADR-0059 §1](0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)
  (every wait is said on the issue), [ADR-0060](0060-the-person-is-on-the-loop.md)
  (the person is on the loop); principles 13 and 14
  ([PRINCIPLES.md](../PRINCIPLES.md))
- **Research:** [what each forge notifies](../research/surfacing-proposals.md#what-each-forge-notifies)
- **Amends, once built:**
  - [ADR-0038, amendment 2026-10-17](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md#amendment-2026-10-17-what-the-trials-first-week-showed):
    the state comment is no longer posted anew when a proposal starts
    waiting; a short call is posted instead (3). Its "Notifications"
    and the rejected "second, short comment per proposal" read so.
  - [ADR-0059 §1](0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue):
    "one sticky comment a role" becomes one section a role, in the
    item's one comment (1).
  - [ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md),
    "each run its own comment and record": the summary and the lenses
    that warn write two sections of the one comment (1, concurrency).

## Context

- **A mail for each new comment.** The maintainer opens most issues and
  pull requests; GitHub mails him for every new comment on them, never
  for an edit, a label or a reaction. GitLab mails for an edit only a
  user it newly mentions.
- **Counted on 2026-11-28**: issue [#2321](https://github.com/JN0V/workline/issues/2321)
  holds 5 comments of workline's App, one per role
  (`product-owner/state`, `reviewer/spec`, `tester/issue`, `line/state`,
  `developer/claim`); [#2164](https://github.com/JN0V/workline/issues/2164)
  holds 7. Pull request [#2337](https://github.com/JN0V/workline/pull/2337)
  holds 10, plus 8 reviews with 17 comments on lines. One mail each
  when created; none of them asks him anything but one.
- **The reverse fault**: the reviewer's question "is the spec good
  enough to build?" after its rounds (`roles/reviewer/code/specforge.go:256`)
  is an edit of its spec comment: no mail, where a person must act.
- **The ecosystem's pattern**: one bot comment, edited in place
  (Codecov, Vercel, Netlify, Renovate's dependency dashboard).

## Decision

### 1. One state comment an issue or a pull request

- **Every role, and the line, write a section** of one comment of
  workline's own, keyed as today (`sticky=<role>/<key>`). A section is
  replaced whole; the others are kept as they read.
- **Written in the forge layer**: `Sticky` and `KeepOwn`
  (`internal/forge/github.go:561`, `internal/forge/own.go:60`) write the
  section; `Comments` and `Notes` hand each section to readers as the
  comment it was. No role, no record reader changes.
- **The order a person reads**: what waits on a person first, then each
  role's section, the history last.
- **Concurrency**: two jobs may write at once (the reviewer's verdict and
  its lenses apart). A write reads the comment just before it, and reads
  it back after; its section missing, it writes once more. A lost write
  is said in the job's log, as a forge error is.
- **Past the forge's size** (GitHub: 65,536 characters): the history is
  cut first, then the largest section moves to a second state comment,
  created once, edited after.
- **Comments of the earlier engines**: folded into the one comment at
  its next write, then deleted (a deletion mails nobody).

### 2. A short dated history at its end

- **One line a write that changed what a person reads**: date and time
  (UTC), the role, one sentence. `2026-11-28 08:16 reviewer: no lens
  blocks on 9f07c4d.`
- **The last 10 lines**; GitHub keeps every edit of a comment ("edited"
  menu), GitLab its notes' versions: nothing is lost. Hidden records
  stay HTML comments, out of the history.

### 3. A new comment, with a mention, only when a person must act

A **call**: a new comment, a few lines, naming the person, holding what
to answer (the 2026-10-27 lesson of ADR-0038: the mail holds what a
person answers), linking the state comment. Posted once a wait; its
marker keeps it from being posted twice.

- **When**: a question to answer, a yes awaited, a block only a person
  lifts. Nothing else.
- **Whom**: the issue's author when a person; else the people
  `CODEOWNERS` names for `.workline/`; else nobody named (subscribers
  are still mailed).
- **GitLab**: the mention goes only in the call, never in the state
  comment (an edit adding one would mail).

### Inventory at `08ad89933`

**Edited sections already, kept as sections**: the roles' sticky
intentions applied at `internal/engine/engine.go:3127` (product owner's
`state` and evidence, reviewer's `spec`, `summary`, `warn`, tester's
`issue`, `green`, objections, developer's `claim`, `round`, `code`,
`bounds`, `ready`, documentalist's `judged`, `too-large`, `patch`,
inspector's), the engine's `line/state` (`internal/line/record.go:629`),
sizes (`internal/line/size.go:335`), gates (`internal/engine/annotate.go:150`),
merge refusals (`internal/engine/refused.go:110`), "brought up to date"
(`internal/engine/uptodate.go:246`), "held: a person's comment has no
answer" (`internal/engine/merge.go:1059`), the weekly sample
(`internal/sample/apply.go:117`).

**New comments today that become sections**: workline's own merge alone
(`internal/engine/routine.go:461`), a release shipped
(`internal/engine/candidate.go:593`), a refused candidate's note
(`internal/engine/candidate.go:670`), a role's run that did not end
(`internal/engine/engine.go:4300`), base red (`cmd/workline/main.go:2553`),
an issue folded by the sample (`internal/sample/acts.go:91`), the product
owner's reply to a person (`internal/backlog/ledger.go:902`, its 👀
reaction kept), a set-aside kept (`internal/engine/backlog.go:86`), the
report moved (`internal/backlog/ledger.go:914`), a role's comment with no
key (`internal/engine/engine.go:3129`).

**Calls** (new, with a mention):

| Today | Where |
|---|---|
| product owner's question; a proposal starting to wait (now posted anew) | `internal/engine/backlog.go:626`, `:639`; `internal/backlog/ledger.go:878` |
| an obsolete issue announced before it is closed | `internal/engine/backlog.go:56` |
| waiting for a person's yes (feature, task, unreproduced bug, workline's own kind) | `internal/engine/code.go:81`, `:211`; `internal/engine/routine.go:370` |
| a yes taken off, the head moved | `internal/engine/merge.go:1167`, `:1194`; `internal/engine/candidate.go:579` |
| a release candidate to try | `internal/engine/candidate.go:359` |
| held by conversations a person resolves | `internal/engine/merge.go:1538` |
| the tester's step a person decides | `internal/line/testerblock.go:53` |
| the spec's rounds ended: a person decides (today an edit) | `roles/reviewer/code/specforge.go:256` |
| the developer's rounds stalled (today an edit) | `roles/developer/code/review.go:1366` |
| a closed issue found again, for a person to reopen | `internal/backlog/opening.go:147` |

**Kept as they are**:

- **A person's own comments**, and the assisted-yes traces
  ([ADR-0068 §7](0068-the-line-steps-back-on-its-own-drift.md#7-a-persons-yes-is-assisted-never-delegated)):
  a person's, never the line's.
- **Comments on lines** (reviewer's findings, objections, replies,
  `internal/engine/review.go:48`, `internal/engine/engine.go:3088`,
  `internal/engine/merge.go:1362`): the conversations the merge gate
  reads. A later option: the findings as the check run's summary and
  annotations, which mail nobody; out of scope here.
- **A duplicate's closing comment** (`internal/engine/engine.go:3251`):
  GitHub links the original by its text; the closing mails anyway.
- **An issue opened** (a demotion, a refused candidate): an issue is the
  record; the local and `cmd:` forges' comments, which mail nobody.

## The budget exception

- **Estimate**: about +330 lines of non-test Go, −110 → **about +220
  net**. The forge's sections and history, +220; the call and whom it
  names, +50; the call sites, +60; `Renew` and its ledger path removed,
  −90 (`internal/forge/github.go:616`, `gitlab.go:1154`, `fake.go:638`,
  `forge.go:508`, `internal/backlog/ledger.go:877`); a few `Comment`
  sites folded, −20.
- **Two hidden markers more**: the state comment's and the history's.
  No setting.
- **Each slice counts its own** in its pull request; past +260 net, the
  slice stops and asks.

## Traceability

- **The forge keeps every edit**: GitHub's edit history, GitLab's note
  versions; the dated history says what moved, at a glance.
- **The auditor's records are unchanged**: they stay hidden in their
  sections, read back as before
  ([ADR-0046](0046-the-auditor-re-checks-every-roles-acts.md)).
- **A call is never deleted**: what was asked, and when, stays.

## Slices

1. [#2349](https://github.com/JN0V/workline/issues/2349): the forge's one state comment, its sections and
   dated history (1, 2).
2. [#2350](https://github.com/JN0V/workline/issues/2350): the call, for the line's waits (3).
3. [#2351](https://github.com/JN0V/workline/issues/2351): the product owner's questions and proposals as
   calls, `Renew` removed (3).
4. [#2352](https://github.com/JN0V/workline/issues/2352): the spec review's question after its rounds (3).
5. [#2353](https://github.com/JN0V/workline/issues/2353): the developer's and tester's rounds: a stall or a
   block only a person lifts as a call, their notes as sections (3).

Each builds alone after slice 1; slices 3 to 5 after slice 2.

## Consequences

- **An issue built end to end**: from 5–7 mails to the calls it needs,
  often none. A pull request: from 10 to its calls, plus one per review
  with comments on lines.
- **A person asked is mailed**, where an edit hid it before (the spec's
  question, a stalled round).
- **One comment longer to read**: what waits on a person on top.

## Not chosen

- **Deleting and posting the state comment anew** (ADR-0038's way): one
  mail each time, for an update.
- **A mention edited into the state comment**: GitHub mails nobody for
  an edit.
- **Each role keeps its comment, edited**: what we have; one mail a role
  an item.
- **The check run's summary for every state**: a check run is per
  commit, not per issue; a later option for the reviewer's findings only.
