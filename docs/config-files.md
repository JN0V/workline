---
sources:
  - internal/forge/local.go#Local
  - internal/forge/local.go#InCI
  - internal/engine/engine.go#keepRuns
  - internal/agent/seen.go#Seen
checked: e81df7b34
verified: agent:claude-code
---
# The files

Part of [the configuration](config.md).

| File | Holds |
|---|---|
| `.workline/config.yaml` | the project's settings, [the project's config](config.md#worklineconfigyaml) |
| `.workline/roles/<role>/prompts/<facet>` | a facet replacing the shipped one (`policy.md`, `instruction.md`…); at the folder's top level for a role with no `prompts/` folder |
| `.workline/work/<id>.md` | a work item, without a forge |
| `.git/workline/issues/<n>.md` | `forge: local`: an issue, never committed; `workline issues` reads them (below) |
| `.git/workline/merge-requests/<n>.md` | `forge: local`: a merge request, its local branch and base named in the front matter; merged once its base holds the branch, closed once the branch is gone |
| `.workline/off` | empty: the global hook skips this repository |
| `.git/workline/reviewer-record` | the commits a review on this machine answered whole, not asked again (roles/reviewer) |
| `.git/workline/inspector-record` | the gates' findings the inspector triaged on this machine, by fingerprint, not asked again (roles/inspector) |
| `~/.config/workline/` (macOS: `~/Library/Application Support/workline/`) | yours: `config.yaml`, `allowed-identities`, `gitleaks.toml`, your facets in `roles/<role>/` ([your own config](config.md#your-own-config)) |
| `~/.cache/workline/models-seen.yaml` (macOS: `~/Library/Caches/workline/`) | the last model that answered each alias on this machine: when another one answers, a run reports `model-changed` once, without blocking ([ADR-0004](adr/0004-follow-model-aliases-and-measure.md)) |

A local issue holds its title, state, labels and milestone in its front
matter, then its body and comments. With no forge (`none`), an issue a role
opens is refused, as every write that needs a forge; in CI (`CI`,
`GITHUB_ACTIONS` or `GITLAB_CI` set), the local forge refuses its writes.

