---
sources:
  - internal/wake/wake.go#readForge
  - internal/forge/wake.go#closesChunk
  - internal/forge/wake.go#github.Closes
  - internal/forge/wake.go#gitlab.Closes
  - internal/forge/github.go#github.MergedSince
  - internal/forge/gitlab.go#gitlab.MergedSince
  - internal/forge/wake.go#github.Labelled
  - internal/forge/wake.go#github.AppStatus
  - internal/forge/github_merge.go#github.Bypass
checked: e2e165c2d
verified: agent:claude-code
---
# The wake's cost

Part of [The wake](wake.md).

- **No agent, no token** of a model
  ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)).
- **Forge reads**: the merges of 14 days (a page of 100 a call), their
  closing issues (50 merge requests a call on GitHub, one each on
  GitLab), the label, the rulesets, the statuses of the merged heads.
  About 15 s on workline's history.
