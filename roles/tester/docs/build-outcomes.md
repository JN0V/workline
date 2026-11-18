---
sources:
  - internal/tester/write.go#Post
  - internal/tester/write.go#draftBody
  - internal/tester/write.go#unprovenOf
  - internal/tester/judged.go#doubt.unproven
  - internal/tester/frozen.go#record
  - internal/tester/objection.go#record.rewritten
  - internal/tester/objection.go#unprovenSaid
  - internal/developer/pullrequest.go#body
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

## A test named by its file

- **A row with no `test-name`** names a test by its file alone: the
  Draft says only the file is proven, and the green proof then wants a
  test of it, green
  ([#1293](https://github.com/JN0V/workline/issues/1293)).

## No error line read

- **`tests-not-built`** warns; with no new test in the report,
  `test write` opens no Draft (`no-new-test`).
