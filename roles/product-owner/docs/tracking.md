---
sources: [roles/product-owner/role.yaml, internal/builtin/productowner, internal/backlog]
checked: c24603b
verified: agent:claude-code
---
# Product owner — what each run tracks

Part of [the product owner](../README.md), beside [a run](run.md): what
the engine follows from run to run, most of it with or without an agent.
The contract: [backlog acts](../../../docs/spec/backlog-acts.md).

## Obsolete: announced, then closed

An issue the code made obsolete is announced, not closed
([ADR-0024](../../../docs/adr/0024-obsolete-is-announced-then-closed-on-silence-and-a-second-judge.md)):
a comment to its reporter, the code quoted and the day from which it may
close, and the label `workline:obsolete`.

- **At a run `days` later (7)**, if nobody wrote on it, the label is still
  there, no exempt label (`pinned`, `security`) was set and the code quoted
  is still there, a second judge of another model is asked apart.
- **If it agrees**, the engine closes the issue as completed, saying the
  judge and its level.
- **Kept open for good** on that evidence (`kept` in its state): anyone's
  comment, the label taken off, or the judge's no.
- Announcements and closings share `close-obsolete`'s cap.

## What waits

The backlog's order puts an issue that waits on an open issue after it,
whatever its priority; a cycle is reported, never followed. Every run, with
or without an agent, says:

- the first ready issue waiting on nothing (`next-ready`) — the one offered
  to whoever builds next;
- each issue waiting (`waiting`);
- each cycle (`dependency-cycle`).

The report says them too, folded under "issues waiting on another open
issue". A split need is never the one offered: its parts are.

## Next and Stuck

After what a person has to do, the report says **Next** and **Stuck**
([ADR-0031](../../../docs/adr/0031-the-report-opens-with-what-is-next-and-what-is-stuck.md);
changing: [ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)),
rebuilt at every run with or without an agent, nothing of them stored.

- **Next**: the first `next-max` (5) ready issues of the order waiting on
  nothing, with their milestone and priority.
- **Stuck**: each issue waiting on a person for more than `stuck-days`
  (14), with the day it started:
  - ready since the forge says it got the label, no pull request nor
    commit naming it since;
  - its reporter last written to, no answer since;
  - a proposal of the report unticked, since the record says it was first
    proposed;
  - an announcement as obsolete past its delay, no second judge yet.
- An issue appears once, in its first list.
- A day the forge does not say is said (`stuck-unknown`), never read as
  nothing stuck.
- A run with nothing else to write rewrites the report when its opening
  changed.

## A changed need

A changed need flags the issues built on it
([ADR-0032](../../../docs/adr/0032-a-changed-need-flags-the-issues-built-on-it.md)),
found with or without an agent: an issue's Need or Scope a person rewrote
since its state kept them, or the lines of a file an issue was imported
from, changed by a commit since it was read.

- Its open parts, and the issue imported, are read first, with the text as
  it was and as it is.
- Every act the agent proposes on them goes to the report (`need-changed`)
  — `unready`, back to refine, included, which only a person's tick does.
- The issues waiting on it and those on the same code are listed, not
  read.
- Nothing is written to them. In the report:
  - every issue read again, nothing proposed: no box, one folded line;
  - proposals made: under each issue, with the change it was read for;
  - an issue not read: a box under "To check", until a person ticks it.
- A file the project archived (`archived`) flags nothing: a roadmap whose
  items became issues, edited since.

## A parent and its parts

A split need — a parent — gets one comment, with or without an agent,
edited in place as its parts move
([ADR-0029](../../../docs/adr/0029-a-parent-is-accepted-by-a-person-from-what-its-parts-delivered.md)):

- **each part**: open, closed as completed with the pull request or commit
  that closed it, or closed without delivering (not planned, a duplicate,
  gone);
- **each item of its Verification**: proved where a part delivered quotes
  it, in its own Verification or in what closed it — and, when it names a
  test, that test found in the code at the run's commit —, or "not proved"
  (`proof-test-missing` when the test is not there, `proof-test-unread`
  when git could not say).

Once all its parts are closed, the comment, a finding (`parent-to-accept`)
and the report's "To accept" ask a person to accept it by closing it. The
role never does, and an agent's closing of a parent is dropped.

## Slipped milestones

An issue in a milestone whose release is tagged slipped: the engine moves
it to the nearest open milestone not released, with or without an agent,
or proposes it in the report when there is none.
