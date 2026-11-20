---
sources:
  - internal/kit/commitmsg/commitmsg.go#Check
  - internal/kit/commitmsg/commitmsg.go#letThrough
  - internal/kit/commitmsg/commitmsg.go#possibleCodes
  - internal/kit/commitmsg/commitmsg.go#split
  - internal/kit/commitmsg/commitmsg.go#skipped
  - internal/kit/commitmsg/commitmsg.go#InternalCodes
  - internal/kit/commitmsg/commitmsg.go#internalCodes
  - internal/kit/commitmsg/commitmsg.go#secondLine
  - roles/committer/code/committer.go#Pre
  - roles/committer/code/committer.go#checkRange
  - roles/committer/role.yaml
checked: dd77bf922
verified: agent:claude-code
---
# Committer — the checks

Part of [the committer](../README.md).

## Check (`pre`, no AI)

On `commit-msg` (the git hook), the message being written; on `merge-request`
and `pre-push`, every commit of the range given as `--input range=<base>..<head>`
(the pre-push hook gives the commits being pushed).

| Rule | Refuses |
|---|---|
| `format` | a subject that does not read `type(scope): summary`, with a type from `types` |
| `blank-after-subject` | a second line that is not empty: git would read it as part of the subject |
| `subject-length` | a subject over `subject-max` characters (72) |
| `body-length` | a body over `body-max-lines` lines (12), not counting comments, blank lines and trailers |
| `internal-code` | a reference that means nothing outside the project (`AC-3`) in the subject; it belongs in a trailer (`Refs: AC-3`) |
| `possible-internal-code` | a code with no dash (`F207`), as often a public name (`nRF52`): the agent judges it |

- `internal-codes-allow` lists public names that look like an internal
  code (`SHA-256`, `RFC-1234`); a refusal gives the line that lets one
  through, yours or the project's. The [reviewer](../../reviewer/README.md)
  reads the same lists in the comments a change adds.
- `internal-codes-maybe` holds the patterns of codes with no dash. The
  agent may keep one; without AI it blocks; on commits already made, it is
  only a warning.

Messages git writes itself (`Merge …`, `Revert "…"`, `fixup! …`, `squash! …`,
`amend! …`) are not checked. A message that passes asks no agent.

## The squash title

On a merge request the forge may squash, the subject it lands on the
base is checked by the same rules, room left for GitHub's ` (#N)`
([more](squash-title.md)).
