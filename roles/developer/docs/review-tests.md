---
sources:
  - roles/developer/code/review.go
  - roles/developer/code/rewrite.go
checked: 7c650fc50
verified: agent:claude-code
---
# The review loop: tests in a round

Part of [a round](review-loop.md#a-round). The tests are the
tester's: the developer never edits a frozen one
([the bounds](bounds.md)).

- **A test the tester rewrote** ([#939](https://github.com/JN0V/workline/issues/939)):
  after an objection, the tester's Draft holds an acceptance test
  otherwise than this branch.
  - **Taken as the tester's record names it**: the round lays it on
    its branch, the agent's worktree holding it, proves it with the
    rest and pushes it.
  - **The record follows**: its start commit moves to the Draft's
    head, so [the bounds](bounds.md) hold the new test.
  - **Not in the record**: never taken, and no round due for it.
- **A finding in a frozen test**: the developer may not change that
  file. The round asks its agent to object to the test at that line,
  the finding as its why; [the tester](../../tester/README.md)
  rewrites it, and the next round takes the rewrite. Its Draft closed:
  nothing to object to, the rounds run out to a person.
- **A test finding on code is the tester's**
  ([#1265](https://github.com/JN0V/workline/issues/1265)): the tests
  lens finds a behaviour with no test, on a line of code: sent to the
  tester, no agent asked, no round counted
  ([in full](review-tests-code.md)).
