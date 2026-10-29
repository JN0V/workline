---
sources:
  - internal/developer/developer.go
checked: 170e0fba
verified: agent:claude-code
---
# The pick

How the developer chooses the issue it takes
([ADR-0054](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#it-takes-work-by-the-forges-assignment)).
No agent is asked: it is mechanical, and the conformance cases prove it.

## Which issue

The first open issue, in the
[backlog's order](../../../docs/spec/backlog-acts/next.md#the-order-and-what-waits),
that is:

- **Ready**: labelled `workline:ready`, waiting on no open issue, with
  no parts.
- **Assigned to nobody**: a person's assignment is their claim; the
  developer never unassigns anyone.
- **Not taken**: no claim label on it.
- **No open merge request closing it**: `Closes #n`, `Fixes #n` in its
  title or body. `Refs #n`, as the tester's Draft says, closes nothing.
- **Clear of the issues the developer holds**: no path its Scope names
  (in backticks, with a `/` or a `.`) under or over one theirs names.
- **Its tests written first**, where the project routes the tester: an
  issue whose Validation names scenarios waits until the tester's Draft
  of its tests is open ([the triggers](triggers.md#on-a-schedule)).

`--issue <n>` looks at that one alone. Each ready issue passed over is
said, with why.

## The claim

GitHub does not assign an issue to an App: tried on 2026-10-29, the REST
API answered 403 for `workline-jn0v[bot]`, and GraphQL said "Assigning
agents is not supported with GitHub App installation tokens". So the
claim is a label, on every forge:

- **`claim-label`**, `workline:developing` by default, put on the issue.
- **A comment** saying it took the issue and what it passed over, with
  a hidden line, `workline:developer-act`: the commit the run read, the
  issue, the issues passed and why. The auditor reads it (below).
- **A person takes it back**: remove the label, or assign themselves.

## With no agent

The issue is named in the run's summary; no label, no comment, no
branch (principle 5).

## What the auditor reads

The pick is the engine's own move, so the weekly sample does not ask a
judge to re-check it
([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row)).
Its record stays on the issue, read with the code once that is built.
