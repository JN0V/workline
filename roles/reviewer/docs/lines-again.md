---
sources:
  - roles/reviewer/code/lines.go#lineKeyOf
  - internal/forge/review.go#LineComment
  - internal/forge/review.go#review
  - internal/forge/review.go#Reviewed.Section
  - roles/reviewer/code/resolve.go#answered
  - roles/reviewer/code/resolve.go#ofThisSide
  - roles/reviewer/code/resolve.go#gone
  - roles/reviewer/code/resolve.go#readAgain
  - roles/reviewer/code/reviewer.go#testFiles
  - roles/reviewer/code/record.go#retested
  - roles/reviewer/code/resolve.go#resolvedLine
  - roles/reviewer/code/resolve.go#fixedRows
  - roles/reviewer/code/resolve.go#fixedWarn
  - internal/engine/review.go#applier.resolveThreads
checked: cf8d4d9f
verified: agent:claude-code
---
# Reviewer — not repeated, found again

Part of [each finding on its line](lines.md).

## Not repeated, found again

- **A key is hidden in each comment**: the lens or rule, the file, the
  line's text, spaces aside, and which of the lines reading so it is; a
  line that cannot be read, its number. Lines added above keep it; a line
  changed gets a new one.
- **Still open on the merge request**: not posted again, counted in the
  summary ("still open from before"). It matters on a later push that
  reads a line again, and when every lens reads the whole change once
  ready ([#129](https://github.com/JN0V/workline/issues/129)).
- **Resolved by a person**, and found again: posted again, in a new review.
- **Resolved by the reviewer once the code answers it**
  ([#1039](https://github.com/JN0V/workline/issues/1039)), as a block
  goes ([judge](judge.md#a-block-stands-until-a-commit-answers-it)):
  - its line is no longer at the head: changed, or removed;
  - or its lens reads the file again and finds it no more, its judge's
    no included; for the tests lens, a test file touching it counts: in
    its folder, named after it in a folder mirroring its own
    (`src/test/…/GridTest.kt` for `src/main/…/Grid.kt`), or naming it
    ([#1417](https://github.com/JN0V/workline/issues/1417)). A push
    changing such a test asks the tests lens again, beside the lens in
    turn, so that it reads it;
  - or, a warning only, the developer's round answered it with a fix
    pushed, its tests green, and the next run does not raise it again
    ([#1050](https://github.com/JN0V/workline/issues/1050)): a warning
    asks nothing, its lens may never read the file again.
  - or the tester answered that no test is needed, and a judge found
    the reason holds
    ([judge](tester-answers.md)).
  - The summary comment counts them, under **Resolved**.
- **Left open**: found again, a block still standing, a lens that read
  elsewhere, a warning nothing answered; one a person opened, or
  answered in, stays theirs.
