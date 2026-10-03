---
type: reference
sources: [internal/backlog, internal/engine, internal/forge]
status: draft
---
# Acts on the backlog

What the engine does when a role acts on a project's issues rather than on
its code (ADR-0018). The product owner proposes the acts; the engine checks
each one against the code and the forge, then does it, proposes it, or drops
it. This page is the contract; the first act built is closing.

## The issue's state

What the engine knows of an issue lives in one comment it keeps on that
issue, marked `<!-- workline:sticky=product-owner/state -->`:

```yaml
sources: [src/export/csv.go#WriteRows]   # what the issue is about, in the code
confirmed: 1a8e5a4                       # the commit it was last found true at
judged: 1a8e5a4                          # the commit the role last read it at
```

An issue without that comment, or with one that does not read, is never
acted on: an act on it is dropped (`no-state`, `state-broken`) and nothing is
written on the issue. The role's `pre` gives an issue it takes its first
state, `confirmed` at the commit it read, with no sources until it is
refined.

## Reading

A run reads at most `issues-per-run` issues, with at most `code-lines-max`
lines of the code they name: those never read first, then those whose
sources a commit touched since they were read. An issue whose code did not
change is not read again, however old (ADR-0018); the others are listed by
title only, so a duplicate can still be named. Each issue read gets
`judged` moved to the run's commit; without an agent, none is.

## Closing

```yaml
- close:
    issue: 12
    reason: duplicate          # duplicate | obsolete
    duplicate-of: 7            # a duplicate's original
    quote: {issue: 7, text: "the export drops the last row"}
    # or, for obsolete: {path: src/export/csv.go, text: "for i := 0; i <= len(rows); i++"}
    why: "#7 reports the same row dropped by WriteRows."
```

- **Never not planned.** Refusing a need is the person's (principle 1): any
  other reason is dropped (`close-reason`), the issue left open.
- **No quote, no act.** The quote must be found again, as written but for
  spaces, in the file at the commit the run is on, or in the issue's body
  or one of its comments. A quote missing or not found drops the act
  (`no-quote`).
- A duplicate is closed with the forge's own reason (GitHub's `duplicate`,
  linking the original), an obsolete issue as completed; both with a
  comment that quotes the evidence and says how to undo: reopen it.

## Autonomy and caps

Each kind of act has a mode and a cap per run, set in the role's settings:

```yaml
acts:
  close-duplicate: {mode: act, max: 3}      # act | propose | off
  close-obsolete:  {mode: propose, max: 3}
```

- `act`: done, up to `max` a run; past it, proposed.
- `propose`: written in the report issue for a person, not done.
- `off`: dropped.

Closing as obsolete starts at `propose` (ADR-0018: a quote proves the text
is there, not that the issue is solved).

## The report

One issue, kept in place (`KeepIssue`, title "Backlog — product owner"),
lists what the last run did and what it proposes, each with its quote and
how to undo it. Its own engine comment
(`<!-- workline:sticky=product-owner/acts -->`) records the closings done
and the kinds dropped back to `propose`.

## Trust

A closing is wrong when its issue is open again. At the next run the engine
reads the issues it closed; one open again puts that kind of act back to
`propose`, whatever the settings say, with a finding `wrong-closing`; the
report says so. Only the person sets it to `act` again. *Not built yet:
how the person does so, reading a tick with its author.*
