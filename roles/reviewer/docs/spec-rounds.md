---
sources:
  - roles/reviewer/role.yaml#settings.spec-rounds
  - roles/reviewer/code/specforge.go#specComment
  - roles/reviewer/code/specforge.go#whoAnswers
  - roles/reviewer/code/specforge.go#specRecord
  - roles/reviewer/code/specforge.go#round
  - roles/reviewer/code/specforge.go#specOnForge
  - roles/reviewer/code/specforge.go#stopComment
  - roles/reviewer/code/specforge.go#askComment
  - internal/engine/engine.go#applier.apply
  - roles/reviewer/code/spec.go#answeredInSpec
  - roles/reviewer/code/spec.go#answersScenario
  - internal/backlog/spec.go#SpecReview.String
  - internal/backlog/spec.go#SpecHold
  - internal/backlog/spec.go#SpecCleared
  - internal/backlog/spec.go#LeftOpen
checked: cf8d4d9f
verified: human:JN0V
---
# Reviewer — a spec's hold and rounds

Part of [Reviewer — a spec before it is built](spec.md): what the
reviewer writes on an issue it read [on the forge](spec-forge.md), and
how its reviews end.

## Its comment, and the hold

- **What it writes**: one comment on the issue, edited each review:
  - what holds it from ready, and who answers each section a finding
    lies in: the product owner, the Scope whoever wrote it, and a Need
    or a Validation that is its own; the person, a Need or a Validation
    they wrote
    ([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md));
    the tester, the Verification, answered as it writes the tests: such
    findings alone hold nothing, spend no round, and the engine moves
    the issue on with no agent
    ([ADR-0066](../../../docs/adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md));
  - the findings, the questions for a person;
  - a hidden record (`workline:spec-review`): the body's digest, the
    round, the important findings open and the sections they lie in.

  A nit or a decision holds nothing; a finding its judge refused is
  dropped ([the judge](judge.md)).
- **The hold** is the product owner's: its `ready` waits until the
  reviewer read the body as it is and no important finding is open, then
  the engine moves it, with no agent
  ([backlog acts](../../../docs/spec/backlog-acts/refine.md)). Not while
  the product owner's own question waits, nor after it moved the issue
  back to refine on that body
  ([a spec read before ready](../../../docs/spec/backlog-acts/spec-review.md)).
- **The answer**: the product owner's next refine rewrites the sections
  the findings lie in that are its own; a person's, it asks the reporter.
  The body changed, the reviewer reads it again: one round more.
  A finding the last review held on a scenario is settled when a line
  outside the Need, Verification and Validation opens with that
  scenario's name, as a person answers it
  ([#1356](https://github.com/JN0V/workline/issues/1356)).

## Rounds, and a spec not settled

How a spec's reviews on the forge end, or wait:

- **Five rounds** (`spec-rounds`): reviews in a row that leave an important
  finding open. The sixth asks no agent, nor takes a place of `specs-per-run`:
  the comment puts a question to a person, and the reviewer stops
  reading it. Each review edits the comment, and a forge mails nobody
  for an edit: so the question is also asked in a new comment, once,
  naming the issue's author and linking the findings
  ([#2352](https://github.com/JN0V/workline/issues/2352)). The person can:
  - accept it as it reads (`workline:accepted`);
  - settle it and set `workline:ready`;
  - or delete the comment, for five rounds more.
- **A review not whole** (a lens failed, the tokens spent) keeps the last
  record, its body unread: read again at the next run, the rounds counted
  as before, ready still held.
- **A record that does not read**: passed over, ready held
  ([on the forge](spec-forge.md#a-record-that-does-not-read)).
