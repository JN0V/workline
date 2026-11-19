---
sources:
  - cmd/workline/main.go#parseTarget
  - defaults/routing.yaml#states
  - internal/routing/states.go#Config.Wakes
  - internal/tester/write.go#testsDone
  - internal/kit/devmark/devmark.go#Branch
  - internal/engine/engine.go#proposeBranch
  - cmd/workline/main.go#localRange
  - cmd/workline/main.go#roundTree
  - internal/engine/engine.go#applier.patch
checked: c46a93d6c
verified: agent:claude-code
---
# Run each step

Part of [the whole line with no forge](local-line.md).

Each run wakes the role the issue's state names, then moves the issue on:

```sh
workline route issue --target issue:1 --ai claude
workline issues show 1   # what it wrote, and what it asks of you
```

| State label | Wakes | What you do |
|---|---|---|
| `workline:to-refine` | the product owner, then the reviewer on the spec ([after it](spec/routing.md#the-reviewer-opt-in)) | answer it (below) |
| `workline:needs-tests` | the tester: tests on `workline/tester/issue-1`, a Draft `!1` | nothing |
| `workline:ready` | the developer: code on `workline/developer/1`, `!2` | nothing |
| `workline:developing` | nobody: its merge request waits for the review | [review it](local-line-review.md) |

Run the same command after each step until the issue is
`workline:developing` with its merge request open.

- **Your checkout stays as it is**: the tester's tests and the
  developer's code are written and committed on their branch in a
  worktree apart; your branch, and your changes not committed, are
  never touched ([#1372](https://github.com/JN0V/workline/issues/1372)).

## Answer the product owner

At `normal`, its drafts wait for your yes, 24 h at most
([how long](../roles/product-owner/docs/settings.md#how-long-a-question-waits)):

```sh
workline issues label 1 --add workline:accepted   # yes: it moves the issue on at its next run
workline issues comment 1 --body "Refuse inf too."  # it revises what it proposes
```

A comment answers the reviewer's questions on the spec the same way:
the product owner reads it at its next run and writes what it settles
into the Scope.

Its comment on the issue lists every answer
([its run](../roles/product-owner/docs/run.md)).

## A merge request's step by hand

The reconciler runs these for you ([review](local-line-review.md)); by
hand:

- **The line**: `git checkout workline/developer/1`, then
  `workline route merge-request --target merge-request:2 --ai claude`.
  A doc fix it proposes goes in a comment on `!2`, never in your tree.
- **A round**, from another branch: `workline develop --merge-request 2 --ai claude`.
  It runs in a worktree of `!2`'s head.

## When a step stalls

- **A state that wakes nobody**: the run says why;
  `workline route reconcile --ai claude` runs what is due
  ([the reconciler](spec/routing.md#the-reconciler)).
- **A run that changed nothing**: run again after the body or labels
  change, or after you answer with `workline issues comment`; the line
  does not ask twice on the same input
  ([once a state](spec/routing.md#once-a-state-the-record)).
- **More**: [troubleshooting](troubleshooting.md),
  [the product owner's](troubleshooting-product-owner.md).
