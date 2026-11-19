---
sources:
  - internal/report/main.go
  - internal/report/dryrun.go#WithDryRun
  - internal/report/dryrun.go#keptDryRun
  - cmd/workline/main.go#keepMain
  - cmd/workline/main.go#noteBaseRed
  - internal/backlog/opening.go#Tracking
checked: c46a93d6c
verified: agent:claude-code
---
# The default branch's issue

Part of [main after a merge](main-after-merge.md).

## One issue

*workline: findings on the default branch*:

- opened when there is something to list; edited in place when what it
  lists changes; left as it is when nothing is new nor gone; never
  commented on;
- its first line counts the findings by kind; then main red, if so,
  with what main gained since; then, folded, each finding the gates
  found main gained, kept while main holds it, and the whole
  repository's findings. On day one, the old debt is shown once
  ([ADR-0049](adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#6-guards)),
  then only what changes;
- red mains counted a week, each once, on the run it turned red at:
  above one a week, the forge's queue is worth turning on
  ([a merge queue or train](merge-queue.md),
  [ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-2-strict-up-to-date-dropped));
- the product owner leaves it alone, as its other issues;
- the night's release dry run keeps a part of its own there, *The next release*: what would hold
  it ([the release](release.md)). Each writer leaves the other's part
  as it is.

## Base red, not yours

While main is red, one comment on each open merge request whose base
holds the red commit, once a red spell: a check red there for it is the
base's. No merge request waits for another to merge
([the merge policy](merge-policy.md#a-red-main)).

## apply --main

`workline apply --main` reads the result `route main` printed, as data;
red, it comments on the open merge requests whose base, a branch of the
clone, holds the red commit.
