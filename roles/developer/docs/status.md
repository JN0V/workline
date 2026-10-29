# Developer — where it stands (2026-10-29)

**In one line:** `workline develop` picks the first ready issue nobody
took and, with an agent set, claims it with a label and a comment
holding its record, then has the agent write the code, held to its
bounds at apply; with none, it names the issue. Steps 1 and 2 of
[ADR-0054's plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#build-plan);
beta; not routed.

A try is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)):
the forge's assignment of an App and the pick on GitHub, on the pull
request of step 1; the bounds on a copy of a real repository, on the
pull request of step 2.

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open; steps 3 to 8 below.
2. **The main issue validated on real use**: open. A maintainer reads a
   pull request the developer opened for a `ready` issue, its
   Verification proved in it, and merges it or sends it back
   ([#117](https://github.com/JN0V/workline/issues/117)).
3. **Observable**: its acts read by the auditor
   ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row)):
   the pick's record from step 1, its refusals at apply in it from
   step 2; its code, drawn, from step 4.

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
- **The bounds** (step 2): the agent's patches held, file by file, to
  the issue's Scope, the frozen acceptance tests, the files that steer
  agents and a gate's definition; a label or comment of its own
  refused; an objection posted on the test's line of the tester's
  Draft ([the bounds](bounds.md)). Cases `developer/refuses-file-outside-scope`,
  `refuses-acceptance-test-edit`, `refuses-steering-file`,
  `refuses-gate-file`, `has-no-merge-intention`, `objects-to-a-test`.

## Missing

Steps 3 to 8 of
[the plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#build-plan):
the proof and the caps, the pull request, the triggers, the review
loop, the merge, the measure. Also:

- **The claim on GitLab** by assignment, where the token's user is a
  service account that can be assigned: a label for now, on every forge.
- **A claim left behind**: an issue claimed by a run that wrote no code
  keeps its label until a person removes it; the code steps release it.
- **A refused patch asked again**: today it is left out and the rest
  applied; asking the agent again with the refusal would save a round.
- **A new file in a new folder**: the engine writes a `{file, content}`
  patch only where its folder exists; a diff creates it.
- **Starting from the tester's branch**: the code is written on the
  checkout the run is given; working on `workline/tester/issue-<n>` is
  step 4's.
