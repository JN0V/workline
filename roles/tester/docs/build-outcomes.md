---
sources:
  - roles/tester/code/write.go#Post
  - roles/tester/code/write.go#draftBody
  - roles/tester/code/write.go#unprovenOf
  - roles/tester/code/judged.go#doubt.unproven
  - roles/tester/code/frozen.go#record
  - roles/tester/code/objection.go#record.rewritten
  - roles/tester/code/objection.go#unprovenSaid
  - roles/developer/code/pullrequest.go#body
checked: c46a93d6c
verified: agent:claude-code
---
# What follows a broken build's proof

Part of [A broken build, red for the right reason](missing-name.md):
what the fast path or [the judge](build-judge.md) does to the run.

## Red

- **Red, by the fast path or the judge's yes**: the Draft opens, and
  its body quotes the proof's findings: `red-judged` says why.

## Red unproven

- **Red unproven** is written on the Draft, its body and its record,
  with the compiler's lines and the judge's reason; the issue moves to
  ready, and the developer starts. Its pull request says it too. The
  green proof on the code and the
  [objection chain](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md)
  settle it: never a pass unsaid, never a stuck run.
- **A test rewritten on an objection** is proven again: red unproven,
  its reply and the record say why.

## Asked once more

- **The judge's no** (`build-broken`): the agent is asked once more,
  with the build's error.
- **No report and no error a row reads** (`report-missing`): on the
  tests' run, the agent is asked once more; on the base alone, final.
  None on the base with exit 0, a module with no test yet, is an empty
  run; none with the tests either is final, the base's finding
  ([#2091](https://github.com/JN0V/workline/issues/2091)).

## A test named by its file

- **A row with no `test-name`** names a test by its file alone: the
  Draft says only the file is proven, and the green proof then wants a
  test of it, green
  ([#1293](https://github.com/JN0V/workline/issues/1293)).

## No error line read

- **`tests-not-built`** warns; with no new test in the report,
  `test write` opens no Draft (`no-new-test`).
