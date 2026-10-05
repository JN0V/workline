---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: 01bf620
verified: agent:claude-code
---
# Product owner

What runs, for humans. The AI never reads this file.

It keeps a project's backlog — its open issues — true to the code, between
the need a person states and the result they accept (ADR-0018). The
contract of its acts is [docs/spec/backlog-acts.md](../../docs/spec/backlog-acts.md).

## A run, when gardening (`schedule`)

1. `pre` gives the agent a share of the open issues (`issues-per-run`):
   those an act was proposed on only for a run's cap first, then those
   never read, then those whose code changed, that someone commented on
   or edited, or reopened after it closed them, since read; each with its
   state comment (its sources, the commit it was last confirmed and read at),
   who opened it — a workline role's issue named as that role's draft to
   refine —, which of its four sections it has, and the code it names —
   by path, by a file's name alone, by a symbol quoted as code, in its body
   or a person's comment — up to `code-lines-max` lines in all; up to six
   others on the same code whole, the rest by title. An issue with no state
   comment gets one, judged from the next run; one whose comment does not
   read is left out (`state-broken`), as is the report issue.
2. The agent proposes acts: a closing — a duplicate, its original's words
   quoted, or an issue the code made obsolete, the code quoted; the code
   an issue is about (`sources`), a line of it quoted; the milestone of
   the release an issue still true belongs to; its priority (`order`, 1
   to 4, a label `workline:priority/N`); refining toward `ready` —
   the sections an issue lacks (`refine`: Scope and Verification from the
   code, Need and Validation as drafts a person makes theirs), the move to
   `ready` (`ready`), or a question to its reporter (`ask`).
3. The engine checks each one when it applies it — never as not planned,
   the quote found again, the issue's state readable, no section a person
   wrote rewritten, `ready` only when the four sections are there and none
   a draft, an outsider's issue proposed, a priority a person set kept,
   at most a fifth of the open issues moved a run — and does it,
   proposes it, or drops it, by the kind's mode and cap (`acts`). One
   report issue lists what was done and proposed. A closing undone, the
   issue reopened, puts that kind back to `propose` (`wrong-closing`).
   The report says each moved issue's priority and milestone before the
   run, to put the order back.

An issue in a milestone whose release is tagged slipped: the engine moves
it to the nearest open milestone not released, with or without an agent,
or proposes it in the report when there is none.

A person accepts the drafts with one label, `workline:accepted`, on one
issue or many from the list of issues: the next run takes the draft lines
out and moves each to `ready`, with or without an agent. Without an
agent, otherwise, only the state comments are written. On `import`,
`workline issues import <file>` has the agent read a committed file a
share at a time — with the lines elsewhere in the file that name its
items' ids, where a file often says what is done — and opens each item
still to do as an issue quoting it, never twice (`open`); without `--apply`, it only says what it would open.

Every role opens an issue through one way the engine keeps for the
product owner (ADR-0018; docs/spec/backlog-acts.md, "Opening issues"): a
key per subject, an issue open or closed holding it never opened again,
the role named, `needs-triage`, a cap a run (`issues-max`); the product
owner reads it from its next run.

## Settings

```yaml
roles:
  product-owner:
    settings:
      issues-per-run: 8
      code-lines-max: 1500
      moved-percent-max: 20                      # milestones and priorities, together
      acts:
        open: {mode: act, max: 30}               # when importing
        sources: {mode: act, max: 10}
        milestone: {mode: act, max: 10}
        order: {mode: act, max: 10}
        close-duplicate: {mode: act, max: 3}     # act | propose | off
        close-obsolete: {mode: propose, max: 3}
        refine: {mode: act, max: 5}
        ready: {mode: act, max: 5}
        ask: {mode: act, max: 3}
```

A project with a human Product Owner sets its acts to `propose`.

Where the role stands: [status.md](status.md).
