---
sources: [roles/product-owner, internal/builtin/productowner, internal/backlog]
---
# Product owner

What runs, for humans. The AI never reads this file.

It keeps a project's backlog — its open issues — true to the code, between
the need a person states and the result they accept (ADR-0018). The
contract of its acts is [docs/spec/backlog-acts.md](../../docs/spec/backlog-acts.md).

## A run, when gardening (`schedule`)

1. `pre` lists the open issues, with what the engine knows of each (its
   state comment: its sources, the commit it was last confirmed at). An
   issue without one gets it, at the commit the run is on, and is judged
   from the next run; one whose comment does not read is left out
   (`state-broken`), nothing written on it. The report issue is not judged.
2. The agent reads them and proposes closings: a duplicate, its original's
   words quoted, or an issue the code made obsolete, the code quoted.
3. The engine checks each one when it applies it — never as not planned,
   the quote found again, the issue's state readable — and does it,
   proposes it, or drops it, by the kind's mode and cap (`acts` in the
   settings). One report issue lists what was done and proposed.

Without an agent, only the state comments are written.

## Settings

```yaml
roles:
  product-owner:
    settings:
      acts:
        close-duplicate: {mode: act, max: 3}     # act | propose | off
        close-obsolete: {mode: propose, max: 3}
```

A project with a human Product Owner sets its acts to `propose`.

Where the role stands: [status.md](status.md).
