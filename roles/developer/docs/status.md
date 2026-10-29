# Developer — where it stands (2026-10-29)

**In one line:** `workline develop` picks the first ready issue nobody
took and, with an agent set, claims it with a label and a comment
holding its record; with none, it names it. Step 1 of
[ADR-0054's plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#build-plan);
beta; not routed.

A try is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)):
the forge's assignment of an App and the pick on GitHub, on the pull
request of step 1.

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open; steps 2 to 8 below.
2. **The main issue validated on real use**: open. A maintainer reads a
   pull request the developer opened for a `ready` issue, its
   Verification proved in it, and merges it or sends it back
   ([#117](https://github.com/JN0V/workline/issues/117)).
3. **Observable**: its acts read by the auditor
   ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row)):
   the pick's record from step 1; its code, drawn, from step 4.

## Built

- **The pick** (step 1): the first ready issue in the backlog's order,
  assigned to nobody, not taken, no open merge request closing it, its
  Scope clear of the issues held; claimed by a label, since GitHub
  assigns no issue to an App
  ([the pick](pick.md#the-claim)); its record hidden in its comment.
  Cases `developer/picks-first-ready-unassigned`,
  `skips-assigned-issue`, `skips-issue-with-open-pr`,
  `skips-overlapping-scope`, `none-ready-says-so`,
  `no-agent-opens-nothing`.

## Missing

Steps 2 to 8 of
[the plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#build-plan):
the bounds at `apply`, the proof and the caps, the pull request, the
triggers, the review loop, the merge, the measure. Also:

- **The claim on GitLab** by assignment, where the token's user is a
  service account that can be assigned: a label for now, on every forge.
- **A claim left behind**: an issue claimed by a run that wrote no code
  keeps its label until a person removes it; the code steps release it.
