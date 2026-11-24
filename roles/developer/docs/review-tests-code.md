---
sources:
  - roles/developer/code/review.go
  - roles/developer/code/rewrite.go
checked: 7c650fc50
verified: agent:claude-code
---
# The review loop: a test finding on code

Part of [tests in a round](review-tests.md), itself part of
[a round](review-loop.md#a-round).

- **A test finding on code is the tester's**
  ([#1265](https://github.com/JN0V/workline/issues/1265)): the tests
  lens finds a behaviour with no test, on a line of code, and the
  tester's Draft is open.
  - **Sent, not asked**: an objection of kind `missing` on a line of
    the Draft's tests, naming the pull request; the agent is told it,
    not asked to fix it. The tester adds the test or says why none is
    needed; a test added is taken by the next round.
  - **Once**: the record keeps it sent; a review raising it again asks
    no one.
  - **Alone**: no agent, no round counted; the rounds never run out on
    test findings.
  - **Only disputes count**: past round 5 means five rounds that each
    answered a blocking finding of the reviewer; a round on a red check,
    a tester's rewrite or a warning alone is no dispute, and runs on.
  - **None needed, still blocking**: the tester answers on the pull
    request that no test is needed; the reviewer's judge weighs the
    reason, and a reason that holds lifts the block
    ([weighed](../../reviewer/docs/tester-answers.md)).
    The review still blocking on the finding, no round settles it: its issue goes `workline:blocked`,
    the reason on it and on the pull request, for a person who reads
    both.
  - **The developer's still**: one on a test file, one asked again from
    the record when a round resumes, or a row where the reviewer groups
    a finding of another lens under it (code to fix on the same line).
