---
sources:
  - internal/developer/developer.go
  - internal/tester/proof.go
checked: 07ca9b9b
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

- **Ready**: labelled `workline:ready`, not `workline:blocked`, waiting
  on no open issue, with no parts.
- **Assigned to nobody**: a person's assignment is their claim; the
  developer never unassigns anyone.
- **Not taken**: no claim label on it.
- **No open merge request closing it**: `Closes #n`, `Fixes #n` in its
  title or body. `Refs #n`, as the tester's Draft says, closes nothing.
- **Clear of the issues the developer holds with code**: no path its
  Scope names ([read as the bounds read it](bounds.md): in backticks,
  or in plain words the repository holds) under or over one theirs
  names. A claim holds its Scope while an open pull request closes it;
  one whose agent wrote no code holds none
  ([amended](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-claim-with-no-code-holds-no-scope)).
- **Its tests written first**, where the project routes the tester: an
  issue whose Validation names scenarios waits until the tester's Draft
  of its tests is open ([the triggers](triggers.md#on-a-schedule)), or
  the tester recorded another kind of proof in its place
  ([proof kinds](../../tester/docs/hand-over.md#a-proof-kind-not-always-a-new-test)):
  taken then, the kind and its reason in its agent's task.

`--issue <n>` looks at that one alone. Each ready issue passed over is
said in the run's verdict, with why.

## What waits is said on the issue

An issue the pick holds back for its own reason gets the developer's
comment, the claim's, edited in place on each run, never a second one
([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)):

- **Its tests not written yet**: why, and that the tester's Draft comes
  next.
- **Its Scope naming a file an issue the developer holds changes**:
  which one, the file, and that it is looked at again once that one is
  merged or let go. The files held are its pull request's changed
  files, its diff from the merge base, not its Scope; read from
  `origin`, else its Scope and the files listed beyond it
  ([#1256](https://github.com/JN0V/workline/issues/1256)).
  An issue that is not a bug merges on a person's yes: the comment
  names its pull request and that yes
  ([#998](https://github.com/JN0V/workline/issues/998)).
  A Scope marked as a guess, a line starting `Guessed:`, waits on
  nothing: its paths were named before their code was read
  ([#970](https://github.com/JN0V/workline/issues/970)). Its words
  alone saying no code was given are no mark: prose is no fact
  ([#1228](https://github.com/JN0V/workline/issues/1228)).
- A person's hold (an assignee), a claim, or an open pull request
  closing it say themselves: nothing is written for them.

## The claim

GitHub does not assign an issue to an App: tried on 2026-10-29, the REST
API answered 403 for `workline-jn0v[bot]`, and GraphQL said "Assigning
agents is not supported with GitHub App installation tokens". So the
claim is a label, on every forge:

- **`claim-label`**, `workline:developing` by default, put on the issue,
  `workline:ready` taken off in the same call: one state at a time
  ([#1081](https://github.com/JN0V/workline/issues/1081)).
- **A comment** saying it took the issue and what it passed over, with
  a hidden line, `workline:developer-act`: the commit the run read, the
  issue, the issues passed and why. The auditor reads it (below).
- **A person takes it over**: assign themselves. To give it back to
  the line, swap the label for `workline:ready`: the label taken off
  alone leaves the issue in no state, off the line.
- **Its card to Doing**, on a board with a doing column
  ([boards](../../../docs/boards.md)). The comment and its hidden line
  say the move, or why it is left:
  - GitLab: that column's label on, the other columns' off, in one call.
  - GitHub: left to the project's *Pull request linked to issue* when
    on; else the card's Status set, on an organisation's project only.
- **No code, no pull request**: the comment says so, and when the issue
  is blocked for a person, its body unchanged
  ([letting a claim go](#letting-a-claim-go)). Meanwhile the claim holds no Scope: an issue overlapping it
  may be taken.
  - **A Draft already green**: a person may push to the tester's Draft
    a change only they may make. Its agent writing nothing, the Draft's
    head is proven as it is; green, the pull request opens from it, no
    commit added ([#1434](https://github.com/JN0V/workline/issues/1434)).
- **Its pull request does not open**: the claim given back at once,
  what failed in the comment; the second time, the issue blocked
  ([when it does not open](pull-request.md#when-it-does-not-open)).

## Letting a claim go

A claim no pull request followed is let go by the developer's next run,
on any event, with or without an agent:

- **When**: `claim-hours` (24 by default) since the forge says the label
  was put on (GitHub's issue timeline, GitLab's label events), and no
  open pull request says `Closes #n`.
- **Sooner, a new body**: a claim whose agent wrote no code is let go
  at once when the issue's body changed since the claim read it
  ([#1084](https://github.com/JN0V/workline/issues/1084)).
- **Woken for it**: Doing wakes no role, so the line's reconciler wakes
  the developer on such a claim, once, past `claim-hours` or on its new
  body. Until then it says the day the claim is let go.
- **No code, the same body**: past `claim-hours`, the line does not wake
  the developer. Its agent would get the same input again. The line
  takes the claim's label off and blocks the issue for a person, the
  reason written (`no-code-spent`,
  [#1120](https://github.com/JN0V/workline/issues/1120)).
  - A developer's run started for another issue keeps that claim too
    (`claim-left-to-line`): the line alone decides the block.
- **What**: `claim-label` taken off, `workline:ready` put back on; the
  claim's comment edited to say why and what comes next. The issue is
  not taken again in the same run.
- **A day the forge does not say**: the claim is kept, said in the
  verdict (`claim-age-unknown`).
- **Taken again**: the line's reconciler wakes the developer on the
  issue let go, and on one it passed over for that claim
  ([the record](../../../docs/spec/routing.md#once-a-state-the-record)),
  and on one whose run ended with no verdict, said on the issue
  ([#953](https://github.com/JN0V/workline/issues/953)).
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
