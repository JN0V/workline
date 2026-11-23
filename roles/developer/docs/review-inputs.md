---
sources:
  - roles/developer/code/review.go
  - roles/developer/code/gates.go
  - roles/developer/code/checks.go
  - internal/forge/checks.go
checked: 7c650fc50
verified: agent:claude-code
---
# The review loop: what a round reads

Part of [a round](review-loop.md#a-round). Each of these starts a
round, or is read in one, beside the reviewer's verdict and a person's
comment.

- **A gate red**: each new finding of the gates' summary comment is
  asked, at its place, with what [the inspector](../../inspector/README.md)
  said of it: why it matters, its fix. Only a report on its branch's
  head; once a report, its head kept in the record. A check that did
  not run asks nothing.
- **A project check red** ([#917](https://github.com/JN0V/workline/issues/917)):
  a check of the project's own CI failed at its branch's head, its
  tests run whole where the proof ran a part. Asked once every project
  check there is done, the end of its failed steps' log in the task
  (80 lines, 6 kB at most). Once a head: the head kept in the record.
  - **The project's**: workline's own are left out, a job of a
    workflow named `workline …`, a check or status named `workline…`:
    their records say what to fix.
  - **Read on GitHub**; the checks not read are said (`checks-unread`).
- **The issue's body changed** ([#1271](https://github.com/JN0V/workline/issues/1271)):
  a person rewrote the issue after the code was written; the record
  keeps the body it read.
  - **Asked as a person's comment is**: the issue as it reads now is
    in the task; the agent holds its code to it.
  - **Once a body**: the new one kept in the record. A record from
    before has none to compare: nothing asked.
  - **A person's change only**: the engine's own `Blocked by` line,
    added or rewritten, starts no round.
  - **No event wakes it**: the hourly reconciler starts it, the issue
    saying so.
- **The reviewer's rules alone** ([#1040](https://github.com/JN0V/workline/issues/1040)):
  rules that hold the lenses count no review run. A summary read at
  another head is a new verdict all the same, its head kept in the
  record (`review-at`).
- **The reviewer's findings, from its record**
  ([#705](https://github.com/JN0V/workline/issues/705)): read from the
  hidden `workline:findings` line of its comment (each finding's lens,
  place, severity, text), never from its table's words.
  - **No record** beside a table, or one that does not parse: nothing
    asked, the run says why (`findings-record-unread`).
- **A finding on its line**: its row shows only a title; the round
  reads it whole from the reviewer's comment on that line. Only
  workline's own account's comment is read.
