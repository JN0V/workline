---
sources:
  - cmd/workline/main.go#followCmd
  - internal/engine/follow.go#Follow
  - internal/engine/follow.go#followMay
  - internal/engine/follow.go#maxDraftRebuilds
  - internal/engine/done.go#settleDone
  - internal/forge/local.go#Local.SettleMerges
checked: 3f432ab14
verified: agent:claude-code
---
# workline follow

Part of [the commands](commands.md).

Run when the default branch moves (CI's `follow` job), with no agent.

- **`--gardening`**, run by gardening's applying job: each merge
  request of a role (`workline/<role>/<task>`) whose base moved under it
  is tended
  ([ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed);
  [more](../roles/documentalist/docs/gardening-merge-requests.md#when-the-base-moves)):
  - rebuilt on the base's new tip and force-pushed, the merge request
    kept (`rebuilt`); one on the tip already is left alone; the
    tester's Drafts, 5 a run at most, the oldest first;
  - never rebuilt (`not-rebuilt`, a warning) when it did not leave from
    the base's history (its merge request goes into another branch);
  - closed, one comment saying why (`closed`), when nothing is left to
    merge: the base holds its change, changed the same lines, or changed
    the docs it only records as read (a record whose docs it did not
    change is rebuilt);
  - one a person committed to is left to them, named when it is in
    conflict with the base (`not-rebuilt`), or when git cannot tell
    (`conflict-unread`).
- **People's pull requests** are never written to: a conflict with the
  base, on a doc's `checked` too, is their author's
  ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr)).
- **The [merge policy](merge-policy.md)**, with `merge:`: auto-merge on
  for each merge request a person accepted, each merging once its own
  head is green.
- **What a merge left open**: the tester's Draft of a closed issue is
  closed, and a closed issue loses the states' labels
  ([the hand-over](../roles/tester/docs/hand-over.md#what-the-builder-does)).
- **A local merge request merged with git** (`forge: local`): each open
  issue its body says it closes (`Closes #<n>`) is closed, with a comment
  saying why, and the request is written `merged`, once; the summary
  counts both ([no forge](no-forge.md)).
- Options: `--base` (default: the branch checked out), `--forge`,
  `--repo`, `--json`, `--summary`; `--verdict <file>`, a head's verdict
  `route --verdict` wrote, kept first (`--verdict-only`: nothing else);
  `--merge-group <branch> --head <commit>`, with `merge:`, the yes of the
  head a merge queue's group ends with set on the group's commit, and
  nothing else ([a merge queue](merge-queue.md)).
