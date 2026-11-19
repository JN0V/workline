---
sources:
  - cmd/workline/main.go#followCmd
  - internal/forge/local_merge.go#Local.AutoMerge
  - internal/forge/local.go#Local.StateOf
  - internal/forge/local.go#Local.SettleMerges
  - internal/engine/uptodate.go#provenUpToDate
  - internal/engine/refused.go#sayRefused
checked: c46a93d6c
verified: agent:claude-code
---
# The merge

Part of [the whole line with no forge](local-line.md).

```sh
git checkout main
workline follow          # merges what the policy says yes to; closes its issue
workline issues          # #1 closed, !1 and !2 merged
```

`workline follow` does what a forge's auto-merge would
([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-11-10-1322-the-local-forge-merges-itself)):

- **A bug's fix merges alone**: its tests red on `main` first, every
  gate passed, the reviewer's verdict on its head, no Important finding
  ([what each gets](../roles/developer/docs/merge.md#what-each-gets-at-normal)).
- **A feature or a task waits for your yes**, said on `!2`:
  `workline issues label !2 --add workline:accepted`, then `follow`.
- **Only the head that passed**: a commit since waits for the next
  reconcile's review.
- **Into `main` in this clone**: a merge commit naming `!2`; nothing is
  pushed. `main` checked out with uncommitted changes, or a conflict:
  nothing merges, the reason said on `!2` and on its issue.
- **Behind `main`, which changed one of its files**: no run checks
  `main` here, so `follow` merges `main` into its branch and runs your
  `test-command` on that merge. Green: the branch moves there, the next
  reconcile reviews it, the next `follow` merges it. Red: nothing moves,
  the test named on `!2` and its issue
  ([#1370](https://github.com/JN0V/workline/issues/1370)).
- **A conflict with `main`** names its files: the next reconcile's
  developer round merges `main` in, never a rebase
  ([behind main](../roles/developer/docs/conflict.md)). One that fails
  blocks the issue: you merge `main` into its branch, or close `!2`
  ([#1354](https://github.com/JN0V/workline/issues/1354)).
- **No `merge:` setting**: you merge with git, then `follow` closes the
  issue:

```sh
git checkout main && git merge --ff-only workline/developer/1
workline follow
```
