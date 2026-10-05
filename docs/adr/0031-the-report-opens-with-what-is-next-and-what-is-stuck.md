# ADR-0031: The report opens with what is next and what is stuck

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 (the report issue; the order, derived, never
  stored), ADR-0021 (rounds written to a reporter), ADR-0024 (an
  announcement, then a second judge), ADR-0025 (a proposal is a box a
  person ticks), ADR-0028 (what waits on an open issue is never offered
  first); principles 4, 5, 12, 14
- **Settles:** #164

## Context

The product owner's report says what the last run did and what it
proposes. Which ready issue to take next, a person derived by hand; an
issue `ready` for weeks with nothing started, a question its reporter
never answered, a box nobody ticked, an announcement nobody judged, all
stayed out of sight. Kanban calls it *work item age*; no-response bots
count the days since a question; every tool reads the day from the
tracker's own history (docs/research/product-owner.md, "What is next, and
what is stuck").

## Decision

### Next

The report opens with the first `next-max` (5) issues of the backlog's
order (`backlog.Order`) bearing `workline:ready`, waiting on no open issue
and with no parts — the issue offered first (`next-ready`) at their head
—, each with its milestone and priority. The "Waiting" part keeps the
issues waiting and the cycles.

### Stuck

Then each issue waiting on a person for more than `stuck-days` (14) days,
with the day it started waiting and how long:

| Waits | Since | From |
|---|---|---|
| `ready`, nothing started: no pull or merge request, no commit links to it since | the day it last got the label | the forge (`Trail`): GitHub's timeline, GitLab's label events and system notes |
| an answer from its reporter: the last round written to them (a question or a text), no comment of a person after it | that round's day | the forge (`Note.Created`) |
| a person's tick: a proposal in the report, unticked and unsettled | the day it was first proposed | the record (`since`, kept while the proposal stays) |
| a second judge: announced obsolete, its delay past, not closed nor kept | the day its delay ended | the announcement's own day, and `close-obsolete.days` |

The last waits on no `stuck-days`: past its delay, it is already late. A
link older than the label started nothing — an earlier reference, a
pull request that only moved the backlog — and is not counted.

An issue appears once, in its first list: one in Next is not said stuck;
one stuck for two reasons is said for the first in the table's order.

### Rebuilt each run, nothing stored

Both parts are computed by the engine at every run, with or without an
agent, from the forge as it is; only a proposal's first day is kept, in
the record, since no forge dates a line of a report. A run that would
otherwise write nothing goes on to rewrite the report when its Next or
Stuck no longer reads as the report says — how long changes with the
day, so a report with an issue stuck is rewritten once a run.

### Where the forge does not say

The local forge keeps no dates, a plugged one may refuse `trail`, a
comment may come without its day: an issue whose day is not known is
not said stuck — it cannot be measured —, and the run says so once
(`stuck-unknown`, info), never read as "nothing stuck" (principle 12).

## Consequences

- A person, or #117's developer role, reads in one place where to start
  and what waits on them, and how long.
- One GraphQL query a run on GitHub, two listings on GitLab, for each
  ready issue not in Next; the comments the run reads already carry
  their day.
- The report is rewritten once a run while something is stuck: one more
  edit of its body a run, which the person's ticks are read before.
- `next-max` 0 to 20 (0: no Next list), `stuck-days` 1 to 365; another
  value stops the run, as `ignored-runs-max`.
- Not decided now: closing or pinging on what is stuck — the report says,
  a person acts.
