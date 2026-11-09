---
sources: [internal/tester/write.go, internal/backlog/verification.go, internal/backlog/spec.go, internal/engine/engine.go]
checked: b8e17f20
verified: agent:claude-code
---
# The issue's Verification: the tester's test plan

Part of [the tester](../README.md). How an issue's work is proved is
written by the tester, as a human tester writes the test plan
([ADR-0066](../../../docs/adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md)).
The product owner writes none: it keeps the Need, the Validation and
the Scope ([writing an issue](../../product-owner/docs/writing.md)).

## When it is written

- **Once its tests are proved**: red on the base, linted, the Draft
  ready to open; the issue in `needs-tests`, before ready.
- **Only where the tester is routed**: with none, the product owner's
  ready moves the issue straight to `workline:ready`, and no
  Verification is written.
- **By the engine, from the tests**: never the agent's text. Each
  scenario, the test that proves it, its file; or why no test can check
  it.
- **Below the line**, beside the Scope, marked by a line a reader does
  not see (`<!-- workline:verification=tester -->`).
- **Said on the issue**, in the tester's one comment there: "the tester
  wrote its Verification from them".

```markdown
## Verification

- **Two by three:** `tests.test_mul_two_by_three`, in `tests/test_mul_two_by_three.sh`.
- **By zero:** `tests.test_mul_by_zero`, in `tests/test_mul_by_zero.sh`.

Each test fails on the code before the change and passes once it is
built; they are on `workline/tester/issue-3`. Written by the tester from
its tests.
```

## A Verification someone else wrote

- **Kept word for word**: a person's, or one a person edited.
- **Rewritten**: one the product owner wrote before the tester took it
  over, as its state records it and unedited since.
- **Its tests prove it another way**: when that text names not every
  test, the tester's comment says so and names each test, with its
  scenario.
- **Its own** is rewritten at its next tests.

## A reviewer's finding on it

- **The tester's**: the reviewer's comment says so; such a finding
  alone holds nothing from ready, spends no round of the review, and is
  never put to a person
  ([the spec review](../../reviewer/docs/spec.md)).
- **Given to its agent**: the finding is quoted in its task, to answer
  in the tests.
