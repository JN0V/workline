---
sources: [roles/product-owner, internal/builtin/productowner, internal/backlog]
---
# Product owner

What runs, for humans. The AI never reads this file.

It keeps a project's backlog — its open issues — true to the code, between
the need a person states and the result they accept (ADR-0018). The
contract of its acts is [docs/spec/backlog-acts.md](../../docs/spec/backlog-acts.md).

## A run, when gardening (`schedule`)

1. `pre` gives the agent a share of the open issues (`issues-per-run`):
   those never read first, then those whose code changed since they were
   read; each with what the engine knows of it (its state comment: its
   sources, the commit it was last confirmed and read at) and the code it
   names, up to `code-lines-max` lines in all. The others are listed by
   title only. An
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
      issues-per-run: 8
      code-lines-max: 1500
      acts:
        close-duplicate: {mode: act, max: 3}     # act | propose | off
        close-obsolete: {mode: propose, max: 3}
```

A project with a human Product Owner sets its acts to `propose`.

Where the role stands: [status.md](status.md).
