---
sources: [internal/role/role.go, internal/doctor/levels.go, roles/product-owner/role.yaml, roles/developer/role.yaml]
checked: 477bd024
verified: agent:claude-code
---
# Autonomy

How far each role goes alone before a person: `cautious`, `normal` or
`enterprising`, one level a role, with one default for the project
([ADR-0060](adr/0060-the-person-is-on-the-loop.md#3-autonomy-is-a-level-per-role)).

```yaml
autonomy: enterprising                         # the project's default
roles:
  developer: {settings: {autonomy: cautious}}  # a role's own
merge: {autonomy: normal}                      # the merge keeps its own key
```

## Which level a role runs at

The first that holds:

1. **The role's own**, `roles.<role>.settings.autonomy`: the product
   owner's setting, as it always was.
2. **The project's default**, the top-level `autonomy`, when the role
   has that level.
3. **The role's default**: `normal`.

- **Each task says which**: `role setting`, `project default` or
  `role default`, beside the level.
- **Inside a level**, an act's own setting wins, field by field, and a
  kind a person undid is proposed whatever the level
  ([the product owner's settings](../roles/product-owner/docs/settings.md#each-kind-of-act)).
- **The merge reads only its own key**, `merge: {autonomy: …}`: the
  project's default never turns auto-merge on
  ([the merge policy](merge-policy.md)).
- **A role with no levels** ignores the default; `workline doctor` says
  so (`level-unused`).
- **An unknown level** is refused: at the top, by every command reading
  the config; in a role's settings, by its run.
- **Set it**: `workline init --autonomy <level>` writes the project's
  default; `--human-po yes` sets the product owner's own to `cautious`
  ([commands](commands-setup.md)).

## What each level changes

| Role | `cautious` | `normal` | `enterprising` |
|---|---|---|---|
| Product owner | proposes what sets direction; a question waits for its answer | completes issues, moves an evident one to ready; a question waits 24 h, then an assumption | the same, higher caps; the assumption written at once |
| Developer | 3 review rounds alone, then its issue is blocked for a person | 5 rounds | 8 rounds |
| Merge (its own key) | a person's yes on each pull request | workline's routine work merges alone, the developer's fixes too | features too; a person's yes on the milestone's [release candidate](release-candidate.md) |

- **The product owner's detail**, act by act:
  [its settings](../roles/product-owner/docs/settings.md#each-kind-of-act).
- **The developer's rounds**: `review-rounds`, set on its own, wins
  over the level ([the review loop](../roles/developer/docs/review-loop.md)).

## Roles with no levels

A level is a role's only where it changes what the role does alone
before a person:

- **Tester**: its tests reach the default branch through a pull
  request, which the merge's level governs; an objection is settled by
  the product owner, never a person.
- **Reviewer**: it judges, it decides nothing; whether its findings
  block is `ai-findings`, set by measures, not by trust.
- **Committer, documentalist, inspector**: rules and budgets; nothing
  to trust more or less.
- **Judge, auditor**: they check the others; a level would only weaken
  the check.
