# ADR-0024: An obsolete issue is announced, then closed on silence and a second judge

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 ("closing as obsolete is the costliest act":
  announced first, closed at a later run if nobody answered and a second
  judge agreed), ADR-0005 (independence), ADR-0021 (who wrote a comment),
  ADR-0023 (GitLab keeps no close reason); principles 1, 4, 5, 6, 14

## Context

ADR-0018 decided how an issue the code made obsolete would be closed, and
left closing as obsolete at "propose" until it was measured: so far the
role only wrote it in its report, where a person had to close it by hand.
ADR-0018 is over its length budget; this says how, as a record of its own.
docs/research/product-owner.md ("Closing on silence, announced first")
found every stale bot announcing on the issue first — a label and a
comment — waiting days, cancelled by any activity or the label taken off,
with exempt labels and a cap a run; none asks a second judge.

## Decision

### Announced, on the issue

In mode `act`, an agent's `close` as
obsolete closes nothing: the engine checks it as any closing (the quote
found again in the code, the state readable) and writes, on the issue, a
comment naming its reporter — why it looks obsolete, the code quoted, the
commit — and the date from which it may be closed, and how to keep it
open; and sets the label `workline:obsolete`. The comment ends with a
block the engine reads back (the quote, the reason, the commit, the date,
the model that proposed it).

### Closed at a later run

By the engine, when all hold:

- `days` (7, `acts.close-obsolete.days`) have passed since the
  announcement — days, not runs, as stale bots count; 0 closes at the
  next run;
- **nobody wrote since**: no comment after it but the engine's and bots'
  (ADR-0021: each comment's author is known on every forge) — a
  stranger's "still happens" counts, as a stale bot's any activity;
- the label is still there, and no exempt label is
  (`acts.close-obsolete.exempt`, `pinned` and `security` by default, as
  `exempt-issue-labels` or Kubernetes' `lifecycle/frozen`);
- the code quoted is still there, as written but for spaces;
- **a second judge agrees**: asked apart, at the best independence from
  the model that proposed it (ADR-0005), whether the code as it is now
  solves the issue; its level is written in the closing comment. No agent,
  or a judge that does not answer: it waits, and the run says so
  (principle 5).

It closes as **completed** on GitHub, as **closed** on GitLab and the local
forge, with a comment: the evidence, the announcement's date, the judge
and its level, and "reopen it to undo". Not `not_planned`: the code did
the work, which is what `completed` says; `not_planned` is a person's no
(ADR-0018), and every role's issue opening reads it as one — a subject
found again on an issue the machine closed must still be said.

### Cancelled for good, for that evidence

A person's comment, the label
taken off, an exempt label set, the judge saying no, or the code quoted
gone: the engine keeps the issue open, takes the label off, records the
evidence in the issue's state (`kept`), and says why on the issue when
the person did not (the judge's no, the code gone). The same quote is not
announced again (`obsolete-kept`); another, from code changed since, may
be. While announced, a second announcement is dropped (`announced`). A
`keep` act lets the agent do the same, reading a reply.

### Caps and trust

`acts.close-obsolete.max` (3) counts announcements and
closings together, the engine's closings first; a closing past it waits
for the next run, an announcement past it is proposed in the report, as
any act. A closing leaves the backlog rather than reordering it: not
counted in the moved share. An outsider's issue is announced as any other,
its reporter named.

Trust, as before: a closing reopened is wrong, and puts
`close-obsolete` back to `propose` — the announcement then waits in the
report for a person, and nothing more is announced. The mode's default
becomes **`act`**: in act, nothing closes without a week where the
reporter and every watcher were told, and a second model's yes. The report
lists what was announced (when it may close, how to keep it open), closed
(how to reopen), and kept.

## Consequences

- The role closes what the code solved, a week after saying so where the
  reporter looks, instead of leaving it to a person's report.
- A wrong closing costs a reopen, and stops the closings until the person
  sets the act back.
- Left: what the weekly sample (ADR-0015) finds over these closings; the
  numbers — 7 days, 3 a run — to measure, as ADR-0018's.
