---
sources: [roles/product-owner/role.yaml, internal/builtin/productowner, internal/backlog]
checked: 8538d558
verified: agent:claude-code
---
# Product owner — what each run tracks

Part of [the product owner](../README.md), beside [a run](run.md): what
the engine follows from run to run, with or without an agent. Nothing of
it is kept outside the issues: each issue's state holds its part
([ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).
The contract: [backlog acts](../../../docs/spec/backlog-acts.md).

## What waits

The backlog's order puts an issue that waits on an open issue after it,
whatever its priority; a cycle is said, never followed. Every run says,
in the job's summary:

- the first `next-max` (5) ready issues, not blocked, waiting on nothing
  (`next-ready`),
  each with its milestone and priority; never a split need;
- each issue waiting (`waiting`) and each cycle (`dependency-cycle`);
- each issue waiting on a person for more than `stuck-days` (14), with
  the day it started (`stuck`):
  - ready, no pull request nor commit naming it since;
  - its reporter not answering;
  - a proposal no person answered;
  - an announcement as obsolete past its delay, no second judge yet.

An issue is said once, in its first list. A day the forge does not say is
said (`stuck-unknown`), never read as nothing stuck.

## Undone acts

Each act done alone a person may undo is kept in its issue's state:

- a closing reopened, a title or priority put back, `workline:ready`
  (or `workline:needs-tests`) taken off, or put back after the role moved
  the issue back to refine, a split's part closed as not planned, a link
  taken off; a kind label changed is never set again, but not counted
  ([the kind](kind.md#a-persons-kind-holds));
- found at the next run: that kind is proposed on that issue from then on;
- never a move of the line: a ready that workline's own account took
  off, as the forge's label events say — the tester finding no test
  possible — is no ready undone, nor one swapped for the developer's
  claim, whoever took it off; one an older engine kept so is taken off
  the state
  ([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md));
- undone `undone-max` (3) times across the open issues: proposed on every
  issue (`demoted`), until fewer are; a run woken for one issue counts
  that issue's alone, reading no other's state
  ([#804](https://github.com/JN0V/workline/issues/804)).

## Its labels

Checked on every issue at every run, with no agent: they say what is true
of it.

- `workline:draft` only while a draft of the role's is left in the body:
  one you edited is yours.
- Set aside — `workline:proposed` taken off — with no draft left: no
  `workline:draft` nor `workline:to-refine` either.
- Set aside, then brought back by your comment and acted on: it waits
  on you again, `workline:proposed` back while it proposes or holds a
  draft.

## A parent and its parts

A split need — a parent — gets one comment, edited in place as its parts
move
([ADR-0029](../../../docs/adr/0029-a-parent-is-accepted-by-a-person-from-what-its-parts-delivered.md)):

- **each part**: open, closed as completed with what closed it, or closed
  without delivering;
- **each item of its Verification**: proved where a part delivered quotes
  it — a test it names found in the code —, or not proved;
- **an item given to a part at the split**: the product owner quotes it
  in the scenario of that part's Validation that delivers it; proved by
  the tests the tester declared for that scenario, found in the code —
  the quote alone proves nothing
  ([split](../../../docs/spec/backlog-acts/split.md#splitting)).

Once all its parts are closed, the comment and a finding
(`parent-to-accept`) ask a person to accept it, and the closing is
proposed: `workline:accepted` closes it. The role never closes it alone.
The line wakes the product owner on a parent as its parts close, even
with no state label
([routing](../../../docs/spec/routing.md#a-parent-told-as-its-parts-close)).

## Off by default

Each comes back with a setting ([settings](settings.md#off-by-default)):

- **Obsolete issues**: announced on the issue, closed `days` later (7) if
  nobody wrote, the label `workline:obsolete` still there and a second
  judge agrees
  ([ADR-0024](../../../docs/adr/0024-obsolete-is-announced-then-closed-on-silence-and-a-second-judge.md)).
  Off, one the code already does is said so on the issue, for you to
  close.
- **Slipped milestones**: an issue in a released milestone moved to the
  nearest open one, or proposed when there is none.
- **A changed need** (`changed-needs`): a Need or Scope a person rewrote,
  or an imported file's lines changed, has the issues built on it read
  again, every act on them proposed
  ([ADR-0032](../../../docs/adr/0032-a-changed-need-flags-the-issues-built-on-it.md));
  an `archived` file flags nothing.
