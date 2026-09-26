---
type: reference
sources: [internal/engine, internal/intent, internal/role, internal/agent]
verified: agent:documentalist
checked: 0554e32
status: draft
---
# Adapting a role

Part of the [role contract](role-contract.md): how a project tunes a role, and how defects become guards.

## Settings

A role ships its defaults in `role.yaml`. A project overrides them in
`.workline/config.yaml`:

```yaml
roles:
  committer:
    settings:
      body-max-lines: 8
```

If a setting a role depends on cannot be resolved, the role says so and blocks.
A check that silently passes for lack of configuration is still believed in,
which is worse than no check.

## Adapting a role to a project

A project whose rules differ from a role's defaults goes, in order, to the first
level that is enough — each one is heavier than the one before:

| Need | Level | Where |
|---|---|---|
| Other values: tag prefix, versioning scheme, limits, paths | **settings** | `.workline/config.yaml`, `roles.<name>.settings` |
| A rule bites too hard, or is not wanted | **enforcement** | `roles.<name>.enforce`: `warn` or `off` per rule |
| The AI should write differently: language, tone, sections | **facets** | `.workline/roles/<name>/policy.md` (or any facet), replacing the shipped one |
| Extra checks on top of the role's own: release only from `main`, a migration needs a note… | **gates** | the `gates:` section of `.workline/config.yaml`, run by routing before or after the role |
| Different logic: another way to compute versions, another convention | **a role of its own** | a folder of roles given by `--roles` or `WORKLINE_ROLES` today; `roles.<name>.from`, pinned, once built — a fork of the shipped role, or a new one |

Settings, enforcement and facets change how a role behaves without touching its
code. A gate adds checks without replacing anything. Only the last level
replaces the role's scripts, and the project then owns them — the shipped
role's conformance cases are the way to check the fork still keeps the contract.

## Defects become guards

*Not built yet: the ledger and the shared skill. Defects found so far are
guarded by tests, recorded in the role's README ("Tried for real").*

When a role lets a defect through, fixing the output is not enough. The defect
is recorded in the role's `DEFECTS.md` (what happened, why, the countermeasure,
the result), and it is closed only when a mechanical check — a `post` rule, a
gate, a test — makes it hard to repeat.

The "why" is found with the five whys: ask why the defect happened, then why
that happened, until the answer is a cause the project can act on rather than a
symptom. A countermeasure against a symptom only moves the defect elsewhere.
The method ships as a shared skill that any role can load.

A guard is trusted to block only when it is proven: it must fail on the commit
where the defect happened, and pass on the fixed tree. A guard that cannot fail
protects nothing, and a guard that fails on everything gets switched off. A rule written only in `policy.md` is a
wish; the ledger is where wishes become checks.
