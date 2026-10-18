---
type: reference
sources: [internal/backlog/depend.go, internal/backlog/order.go, internal/backlog/board.go, internal/builtin/productowner/productowner.go, internal/forge/forge.go]
status: draft
checked: fbf66fcf
verified: agent:claude-code
---
# The order, and what is next

Part of [Acts on the backlog](../backlog-acts.md): the backlog's order, the issues offered next, and those stuck.

## The order, and what waits

**The backlog's order** is derived, never stored: the nearest milestone
first (by its due date, GitHub's and GitLab's alike, a milestone with one
before a milestone without; then by title, in version order, `v1.9.0`
before `v1.10.0`; an issue in none after every one in one), then the
priority (an issue with none after 4), then the lowest
number (`backlog.Less`) — and an issue that waits on an open issue after
it, whatever its labels ([What an issue waits on](depend.md);
`backlog.Order`). The task lists the issues not read in that order, those
given by title marked with the open issues they wait on.

`backlog.Order` is Kahn's: the next issue is the first,
by `Less`, whose blockers among those ordered are all placed; a blocker
closed, or not an open issue, holds nothing back, so a blocker closed
puts the issue back where its labels say. When none can be placed, those
left hold a cycle: it is reported, the cycle's first issue by `Less` placed — never one that only waits on it —, and
the order goes on — the cycle never followed. `ready` is allowed on a blocked issue,
its sections say it is understood, not that it can start; but **the
first ready issue offered** is the first in the order bearing
`workline:ready` that waits on no open issue and has no parts
(`backlog.Offered`, `backlog.NextReady`) — to whoever builds next, person
or developer role (#117); the job's summary lists the first `next-max`
of them ("What is next, what is stuck", below).

Every run, with or without an agent, says it in its findings:
`next-ready` (info) the issues offered first, `waiting` (info) each issue
waiting on an open one, `dependency-cycle` (warn) each cycle, "#12 waits
on #14, #14 waits on #12".

## What is next, what is stuck

The job's summary says two lists (ADR-0031, ADR-0038), computed by `pre` at every
run, with or without an agent, from the forge as it is — nothing of them
stored:

- **Next** (`next-ready`): the first `next-max` (5) issues of the order
  (`backlog.Order`): bearing `workline:ready`, waiting on no open issue,
  no parts; each with its milestone and priority. At 0, none.
- **Stuck** (`stuck`): each issue waiting on a person for more than
  `stuck-days` (14) days, with the day it started and how long:

| Waits on | Since | Read from |
|---|---|---|
| someone to start it: `workline:ready`, no pull or merge request nor commit naming it since | the day it last got the label | the forge's `Trail`: GitHub's timeline, GitLab's label events and "mentioned in" notes, a plugged forge's `trail` |
| its reporter: the last round written to them, no person's comment after it ([The conversation with the reporter](conversation.md)) | that round's day | the comment's `created`, as the forge gives it |
| a person's answer: a proposal on the issue | the day first proposed | the state's `since`, carried to the same act decided again |
| a second judge: announced obsolete, due, neither closed nor kept | the day its delay ended | the announcement's `announced` and `close-obsolete.days` — listed once due, whatever `stuck-days` |

A link older than the label started nothing. An issue appears once, in
its first list: one in Next is not stuck; one waiting for two reasons is
said for the first in the table's order. An issue whose day the forge does
not say is not said stuck, and the run says so (`stuck-unknown`, info;
warn when the forge failed to answer).
