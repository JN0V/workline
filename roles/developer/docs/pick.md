---
sources:
  - internal/developer/developer.go
checked: 37da4914
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
said in the run's verdict, with why.

## What waits is said on the issue

An issue the pick holds back for its own reason gets the developer's
comment, the claim's, edited in place on each run, never a second one
([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)):

- **Its tests not written yet**: why, and that the tester's Draft comes
  next.
- **Its Scope overlapping an issue the developer holds**: which one, and
  that it is looked at again once that one is merged or let go.
- A person's hold (an assignee), a claim, or an open pull request
  closing it say themselves: nothing is written for them.

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
- **Its card to Doing**, on a board with a doing column
  ([boards](../../../docs/boards.md)). The comment and its hidden line
  say the move, or why it is left:
  - GitLab: that column's label on, the other columns' off, in one call.
  - GitHub: left to the project's *Pull request linked to issue* when
    on; else the card's Status set, on an organisation's project only.
- **No code, no pull request**: the comment says so, and when the claim
  is let go.

## Letting a claim go

A claim no pull request followed is let go by the developer's next run,
on any event, with or without an agent:

- **When**: `claim-hours` (24 by default) since the forge says the label
  was put on (GitHub's issue timeline, GitLab's label events), and no
  open pull request says `Closes #n`.
- **What**: `claim-label` taken off; the claim's comment edited to say
  why and what comes next. The issue is not taken again in the same run.
- **A day the forge does not say**: the claim is kept, said in the
  verdict (`claim-age-unknown`).
- **Not yet**: its card stays in Doing; the line's states
  ([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#build-plan),
  step 2) move it.

## With no agent

The issue is named in the run's summary; no claim, no branch, no card
moved (principle 5). What waits is still said on its issue, and a stale
claim let go: neither needs judgement.

## What the auditor reads

The pick is the engine's own move, so the weekly sample does not ask a
judge to re-check it
([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row)).
Its record stays on the issue, read with the code once that is built.
