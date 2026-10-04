---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: 5173ead
verified: agent:claude-code
---
# Product owner

What runs, for humans. The AI never reads this file.

It keeps a project's backlog — its open issues — true to the code, between
the need a person states and the result they accept (ADR-0018). The
contract of its acts is [docs/spec/backlog-acts.md](../../docs/spec/backlog-acts.md).

## A run, when gardening (`schedule`)

1. `pre` gives the agent a share of the open issues (`issues-per-run`):
   those never read first, then those whose code changed, that someone
   commented on or edited, or reopened after it closed them, since read; each with its
   state comment (its sources, the commit it was last confirmed and read at),
   who opened it, which of its four sections it has, and the code it names —
   by path, by a file's name alone, by a symbol quoted as code, in its body
   or a person's comment — up to `code-lines-max` lines in all; up to six
   others on the same code whole, the rest by title. An issue with no state
   comment gets one, judged from the next run; one whose comment does not
   read is left out (`state-broken`), as is the report issue.
2. The agent proposes acts: a closing — a duplicate, its original's words
   quoted, or an issue the code made obsolete, the code quoted; the code
   an issue is about (`sources`), a line of it quoted; the milestone of
   the release an issue still true belongs to; refining toward `ready` —
   the sections an issue lacks (`refine`: Scope and Verification from the
   code, Need and Validation as drafts a person makes theirs), the move to
   `ready` (`ready`), or a question to its reporter (`ask`).
3. The engine checks each one when it applies it — never as not planned,
   the quote found again, the issue's state readable, no section a person
   wrote rewritten, `ready` only when the four sections are there and none
   a draft, an outsider's issue proposed — and does it,
   proposes it, or drops it, by the kind's mode and cap (`acts`). One
   report issue lists what was done and proposed. A closing undone, the
   issue reopened, puts that kind back to `propose` (`wrong-closing`).

Without an agent, only the state comments are written. On `import`,
`workline issues import <file>` has the agent read a committed file a
share at a time — with the lines elsewhere in the file that name its
items' ids, where a file often says what is done — and opens each item
still to do as an issue quoting it, never twice (`open`); without `--apply`, it only says what it would open.

## Settings

```yaml
roles:
  product-owner:
    settings:
      issues-per-run: 8
      code-lines-max: 1500
      acts:
        open: {mode: act, max: 30}               # when importing
        sources: {mode: act, max: 10}
        milestone: {mode: act, max: 10}
        close-duplicate: {mode: act, max: 3}     # act | propose | off
        close-obsolete: {mode: propose, max: 3}
        refine: {mode: act, max: 5}
        ready: {mode: act, max: 5}
        ask: {mode: act, max: 3}
```

A project with a human Product Owner sets its acts to `propose`.

Where the role stands: [status.md](status.md).
