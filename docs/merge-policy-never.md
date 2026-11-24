---
sources:
  - internal/engine/merge.go#mergePolicy
  - internal/engine/merge.go#forkGuarded
  - internal/engine/merge.go#releaseRebuild
checked: e81df7b34
verified: agent:claude-code
---
# Never without a person's yes: a release, a fork

Part of [the merge policy](merge-policy.md#never-without-a-persons-yes).

- **A release pull request** merges only labelled; it never waits for
  the docs ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly)).
  An issue labelled to hold it, or main not green, turns its verdict red
  until they are gone ([what holds a release](release-holds.md)).
- **A fork's**: its run is the workflow its head holds, so the line
  keeps no verdict of it and reads no review of it:
  - labelled, it merges once every check on its head passed: the
    person's yes carries it;
  - one changing CI, workline's config, a gate's file or what steers
    agents is never merged by the line (`fork-gate-files`): a person
    merges it by hand;
  - unlabelled, `follow` says it waits for a yes
    ([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-11-04-1043-a-fork-goes-by-its-yes-and-its-checks)).
