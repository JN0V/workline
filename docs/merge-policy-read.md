---
sources:
  - internal/engine/merge.go#mergePolicy
  - internal/engine/queue.go#queueGroup
  - internal/engine/uptodate.go#upToDateAs
  - internal/forge/github_merge.go#github.AutoMerge
  - internal/forge/gitlab_merge.go#gitlab.AutoMerge
checked: e81df7b34
verified: agent:claude-code
---
# The merge policy: where it is read, and the try

Part of [the merge policy](merge-policy.md).

## Where it is read

- **The policy, the level, the yes, the labels**: `merge.go` reads the
  level and finds the person's yes on the label events, set aside the
  labels not taken from a person.
- **What holds it, a release tool's rebuild, a fork**: `merge.go`, which
  also finds a pull request the forge has left stalled.
- **The queue**: `queue.go`; **up to date**: `uptodate.go`; **auto-merge
  on**: `github_merge.go`, `gitlab_merge.go`.

## Tried

Tried live: [tried.md](../roles/documentalist/docs/tried.md#2026-10-12--the-merge-policy-cautious-live);
left: [status.md](../roles/documentalist/docs/status.md).
