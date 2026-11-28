# ADR-0076: One state comment an item, a new one only when a person must act

- **Status:** proposed, until the maintainer merges it; nothing built.
  Built by [its slices](#slices), one issue each
- **Date:** 2026-11-28
- **Decided by:** the maintainer, on 2026-11-28, before 1.0, with an
  exception to the code budget ([below](#the-budget-exception)); code
  read at `08ad89933` on `main`
- **Builds on:** [ADR-0060](0060-the-person-is-on-the-loop.md),
  [ADR-0013](0013-what-was-judged-is-not-asked-again.md) (whose record
  is read), [ADR-0046](0046-the-auditor-re-checks-every-roles-acts.md#where-each-act-is-read-from);
  principles 13 and 14 ([PRINCIPLES.md](../PRINCIPLES.md))
- **Research:** [what each forge notifies](../research/surfacing-proposals.md#what-each-forge-notifies)
- **Amends, once built:**
  - [ADR-0038, 2026-10-17](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md#amendment-2026-10-17-what-the-trials-first-week-showed):
    the state comment is no longer posted anew; a call is posted (3).
    Its [Notifications](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md#notifications)
    are kept for replies to a person's comment.
  - [ADR-0059 §1](0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue):
    one sticky comment a role becomes one section a role (1).
  - [ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md),
    "each run its own comment": two sections of one comment (1).
  - [ADR-0063 §2, §3](0063-every-open-issue-is-due-waiting-or-done.md#2-one-view-of-an-issue)
    and its Migration: the view reads each section as the comment it
    was; a role's own section, and the line's, stay out of its input.

## Context

- **A mail for each new comment.** The maintainer opens most issues and
  pull requests; GitHub mails him for every new comment on them, never
  for an edit, a label or a reaction. GitLab mails for an edit only a
  user it newly mentions.
- **Counted on 2026-11-28**: issue [#2321](https://github.com/JN0V/workline/issues/2321)
  holds 5 comments of workline's App, one a role; [#2164](https://github.com/JN0V/workline/issues/2164)
  7; pull request [#2337](https://github.com/JN0V/workline/pull/2337)
  10, plus 8 reviews with 17 comments on lines. One asked him anything.
- **The reverse fault**: the spec review's "is the spec good enough to
  build?" is an edit, no mail: [#2087](https://github.com/JN0V/workline/issues/2087)'s
  comment, 7 edits, waited on a person never told.
- **The ecosystem's pattern**: one bot comment, edited in place
  (Codecov, Vercel, Netlify, Renovate's dependency dashboard).

## Decision

### 1. One state comment an issue or a pull request

- **Every role, and the line, write a section** of one comment of
  workline's own, keyed as today (`sticky=<role>/<key>`), replaced whole;
  the others kept as they read. What waits on a person first, the
  history last.
- **In the forge layer**: GitHub's and GitLab's `Sticky` and `KeepOwn`
  (`internal/forge/github.go:561`, `internal/forge/own.go:60`) write a
  section; `Comments` and `Notes` hand each section to readers as the
  comment it was. The local and `cmd:` forges keep a comment a key: they
  mail nobody.
- **Concurrency** (the reviewer's verdict and its lenses apart): a write
  reads the comment just before, and back after; its section missing, it
  writes once more. Still lost, the apply fails as on a forge error, and
  the next run writes it: never in a job's log alone (ADR-0059 §1).
- **Someone else's edit** ([ADR-0013](0013-what-was-judged-is-not-asked-again.md),
  "nor one another edited last"): one edit by a person would make every
  role's record unread. Each section carries the digest of what
  workline wrote; a section matching it is read, one that does not is
  unread (`record-unread`), the others kept. workline's next write makes
  the comment its own again.
- **Past the forge's size** (GitHub: 65,536 characters): the history is
  cut first, then the largest section moves to a second state comment,
  created once.
- **Older comments** a key, workline's own account's, are folded in at
  the next write, then deleted (no mail). One written under a person's
  token (a local run, `internal/engine/engine.go:3127`) is left: it is
  theirs, and GitLab lets only its author delete a note.

### 2. A short dated history at its end

- **One line a write that changed what a person reads**: date, time
  (UTC), role, one sentence. `2026-11-28 08:16 reviewer: no lens blocks
  on 9f07c4d.` The last 10; hidden records stay out.

### 3. A new comment, with a mention, only when a person must act

A **call**: a few lines, naming the person, holding what to answer (the
mail holds what a person answers, ADR-0038 2026-10-27), linking the
state comment. Once a wait, by its marker. On GitLab the mention is in
the call only, never edited into the state comment.

- **When**: a question to answer, a yes awaited, a block only a person
  lifts; and a reply to a person's comment, which names them.
- **Whom**: the issue's author when a person; else whom `CODEOWNERS`
  names for `.workline/`; else nobody named (subscribers still mailed).
- **The tester**: a call only for a frozen test contested, or its tries
  spent; its other blocks are its section.

### Inventory at `08ad89933`

**Calls** (new, with a mention); stickies today unless said:

| Wait | Where |
|---|---|
| a yes: feature, task, findings open, no review yet, after a refused candidate, workline's own kind (new comments today: `:211`, `routine.go:370`) | `internal/engine/code.go:81`, `:92`, `:102`, `:163`, `:180`, `:211`; `internal/engine/routine.go:370` |
| only a person merges; a kind label to set; merge or push again | `internal/engine/code.go:54`, `:70`, `:43` |
| a yes taken off (new comments) | `internal/engine/merge.go:1167`, `:1194`; `internal/engine/candidate.go:579` |
| a candidate to try; conversations to resolve (new) | `internal/engine/candidate.go:359`; `internal/engine/merge.go:1538` |
| files the tester needs | `internal/line/localreview.go:294` |
| the tester's step blocked on two heads in a row (once a head today, new): the first head an edit | `internal/line/testerblock.go:53` |
| a refused merge, a refusal, on a person's own pull request | `internal/engine/refused.go:110`; `cmd/workline/main.go:448` |
| the product owner's question, proposal waiting (posted anew), announcement, reply (new) | `internal/engine/backlog.go:626`, `:639`, `:56`; `internal/backlog/ledger.go:878`, `:902` |
| the spec's rounds ended; the developer's rounds stalled (edits) | `roles/reviewer/code/specforge.go:256`; `roles/developer/code/review.go:1366` |
| a frozen test contested, tries spent | `roles/tester/code/objection.go` |
| a closed issue found again (new) | `internal/backlog/opening.go:147` |

**New comments today that become sections**: workline's own merge alone
(`internal/engine/routine.go:461`), a release shipped and a refused
candidate's note (`internal/engine/candidate.go:593`, `:670`), a revert
(`internal/engine/demote.go:510`: the issue it opens is the call), a run
that did not end (`internal/engine/engine.go:4300`), base red
(`cmd/workline/main.go:2553`), a folded issue (`internal/sample/acts.go:91`),
a set-aside kept (`internal/engine/backlog.go:86`), the report moved
(`internal/backlog/ledger.go:914`), a comment with no key
(`internal/engine/engine.go:3129`).

**Kept as they are** (new comments, no mention):

- **A person's comments** and the assisted-yes traces
  ([ADR-0068 §7](0068-the-line-steps-back-on-its-own-drift.md#7-a-persons-yes-is-assisted-never-delegated)).
- **The weekly sample**, a comment a week whose 👍/👎 are read by its id
  (`internal/sample/apply.go:117`, `internal/sample/count.go:124`;
  [ADR-0015](0015-the-weekly-sample-is-written-on-the-forge.md),
  [ADR-0046](0046-the-auditor-re-checks-every-roles-acts.md#where-each-act-is-read-from)).
- **Comments on lines** (`internal/engine/review.go:48`,
  `internal/engine/engine.go:3088`, `internal/engine/merge.go:1362`):
  the conversations the merge gate reads. A later option: the reviewer's
  findings in the check run's summary, which mails nobody.
- **Comments that close**: an obsolete issue after its announcement and
  a duplicate (`internal/engine/engine.go:3251`, `:3259`; GitHub links
  the original by the text), a pull request closed
  (`internal/engine/follow.go:122`, `internal/engine/done.go:45`). The
  closing mails anyway.
- **An issue opened**, or its comment on one already open under the same
  title (`internal/forge/github.go:661`, `internal/forge/gitlab.go:1206`):
  it stands for an issue, which mails anyway.

## The budget exception

- **Estimate, about +320 net lines of non-test Go**: one state comment,
  digests, folding, size, +260; the call and whom it names, +80; the
  product owner, −45 (`Renew` goes, its deletion kept for folding);
  spec review +10; developer and tester +15.
- **Two hidden markers more** (the comment's, the history's); no setting.
- **Each slice counts its own**; 20 % past its estimate, it stops and asks.

## Traceability

- **The forge keeps every edit**: GitHub's edit history, GitLab's note
  versions; the dated history says what moved, at a glance.
- **The auditor's records are unchanged**, hidden in their sections
  (ADR-0046). A call is never deleted.

## Slices

1. [#2349](https://github.com/JN0V/workline/issues/2349): one state
   comment, sections, dated history (1, 2). No wait loses its mail:
   until #2350 and #2351 land, the keys that wait on a person
   (`merge-waits=*`, `needs-spent`, the product owner's
   `state`) stay comments of their own.
2. [#2350](https://github.com/JN0V/workline/issues/2350): the line's
   waits as calls (3).
3. [#2351](https://github.com/JN0V/workline/issues/2351): the product
   owner's questions, proposals and replies as calls, `Renew` gone.
4. [#2352](https://github.com/JN0V/workline/issues/2352): the spec
   review's question after its rounds.
5. [#2353](https://github.com/JN0V/workline/issues/2353): the developer's
   stall and the tester's two blocks as calls; their notes as sections.

## Consequences

- **An issue built end to end**: from 5–7 mails to its calls, often
  none; a pull request from 10 to its calls, plus one a review on lines.
- **A person asked is mailed**, where an edit hid it (the spec's
  question, a stalled round).

## Not chosen

- **Posting the state comment anew** (ADR-0038's way): a mail an update.
- **A mention edited in**: GitHub mails nobody for an edit.
- **Each role keeps its comment**: what we have, a mail a role an item.
- **The check run's summary for every state**: per commit, not per
  issue; a later option for the reviewer's findings only.
