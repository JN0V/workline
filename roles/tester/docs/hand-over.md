---
sources: [internal/tester/moment.go, internal/tester/write.go, internal/tester/said.go, internal/tester/frozen.go, internal/tester/objection.go, internal/tester/green.go]
checked: 99a14dc0
verified: agent:claude-code
---
# The hand-over: what whoever builds finds

Part of [the tester](../README.md). The contract the developer
([ADR-0054](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md))
or a person reads before building a ready issue.

```mermaid
flowchart LR
  ready["<b>Ready</b><br/>issue #n"]
  tester(["Tester"])
  draft["<b>Draft</b><br/>red tests,<br/>Refs #n"]
  builder(["Builder"])
  pr["<b>Same pull request</b><br/>green, Closes #n"]
  ready --> tester --> draft --> builder --> pr
```

## What is there

- **The branch**: `workline/tester/issue-<n>`, the base plus the
  tester's commit of the tests. The tests are red there.
- **The Draft**: a pull request from that branch, opened by workline's
  account, titled `test: the acceptance tests of #<n>`.
  - It says `Refs #<n>`, never `Closes`: it does not count as a pull
    request closing the issue.
  - `## Tests`: a table, each scenario and the test that checks it;
    a scenario no test can check is listed as not tested, with why.
  - What the proof said: each test red on the base, by an assertion or
    a missing name. A test whose package did not build is named
    `file:name` ([how](missing-name.md)).
- **The record**: workline's own comment on the Draft, marked
  `workline:tester-record`. It holds the issue, the base, each test
  file with its git blob, and the scenarios as read. Only the tester's
  runs write it.

## When the tester refuses an issue

`workline test write --issue <n>` on an issue not ready, or whose
Validation names no scenario, blocks before any agent is asked. The
refusal is written on the issue too
([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)):

- **One comment of the tester's**, edited on each refusal: why, and
  what comes next (ready it, or name its scenarios).
- **Not a job's log alone**: a person on the issue sees why no Draft
  came.

## What the builder does

- **Claims the issue**: a person by the forge's assignment, the
  developer by its label ([the claim](../../developer/docs/pick.md#the-claim));
  the tester assigns nobody.
- **Starts from the branch**: checks it out and pushes its commits on
  top. Never a force push, never a branch of its own for the issue.
  Where the project routes the tester, a ready issue whose Draft is not
  open yet waits for it.
- **The developer** opens a pull request of its own instead,
  `workline/developer/<n>`, on top of the Draft's commits; merged, it
  brings the tests ([its pull request](../../developer/docs/pull-request.md)).
- **Leaves the tests as they are**: a change to a recorded file, or its
  deletion, blocks the merge request (`tester-test-changed`). Its own
  new tests go in other files.
- **Objects** to a test that does not fit, on the test's line
  ([the form](../README.md#frozen-tests-and-objections)); the tester
  keeps it or rewrites it, on top of the branch.
- **Closes**: once the tests are green, the description says
  `Closes #<n>`, and the Draft is marked ready. It then merges by the
  project's policy
  ([ADR-0042](../../../docs/adr/0042-the-engine-merges-by-the-projects-policy.md)).
- **Green three times**: once marked ready, each new test passes on the
  head three runs out of three; one red is flaky and blocks
  (`test-flaky`, [the proof](green-proof.md)).

## Not yet

- A Draft a person closed unmerged is not remembered as refused: a
  later scheduled run may take that issue again
  ([status](status.md#missing)).
