---
sources:
  - internal/line/localreview.go#localLines
  - internal/line/localreview.go#localRound
  - internal/line/localreview.go#localAnswers
  - internal/line/localreview.go#HeadTree
  - internal/line/localreview.go#sameConfig
  - internal/line/states.go#roundsDue
  - internal/engine/merge.go#SayVerdict
  - internal/forge/local.go#Local.Notes
  - roles/tester/code/objection.go#kinds
checked: c46a93d6c
verified: agent:claude-code
judged: 99d82f5
---
# Review the merge request

Part of [the whole line with no forge](local-line.md).

No forge runs a job on a push here: the reconciler does, from `main`.

```sh
workline route reconcile --ai claude  # reviews !2's new head, runs a round due
workline issues show !2               # what the line found on it
```

- **The merge-request line** runs on each developer's request, not a
  draft, whose head has no verdict yet: the committer, the tester's
  check and the reviewer. Their findings go on `!2`.
- **The line's verdict** on that head is kept in the clone, for
  [the merge](local-line-merge.md). A new head is reviewed at the next
  reconcile.
- **A block is said on `!2`**: which step blocked, its findings, what
  you can do, in one comment edited at each run
  ([#1327](https://github.com/JN0V/workline/issues/1327)).
- **A review round**: a finding, or your comment
  (`workline issues comment '!2' --body "…"`), wakes the developer at
  the next reconcile ([the review loop](../roles/developer/docs/review-loop.md#a-round)).
  Its fix is committed on its branch, unless you
  have that branch checked out: then it says so and waits.
- **A test the review finds missing** goes to the tester, on its Draft
  `!1`: the next reconcile runs its answer, which adds the test or says
  why none is needed
  ([#1265](https://github.com/JN0V/workline/issues/1265)).
- **Each runs in a worktree of the head**, apart: your checkout is
  never read nor written.
- **The config it takes** is your clone's `.workline/config.yaml`, laid
  in that worktree, even an uncommitted change: the
  merge request's branch may hold another, committed one. This is
  intended: the runs take the line you run. In CI the line reads the
  config committed on the head.
- **A comment names you**: a review round names the author of your
  comment as the clone's `git config user.name`.
- **By hand**: [a merge request's step by hand](local-line-steps.md#a-merge-requests-step-by-hand).
