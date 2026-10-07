---
type: reference
sources: [internal/engine, internal/intent, internal/role, internal/agent]
verified: agent:claude-code
checked: 13d7bca
status: draft
judged-in-parts: fa1d682
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

The project's settings are laid over the role's as Helm lays a chart's values
over its defaults:

- **A map is merged key by key, at every depth.** `acts: {refine: {mode:
  propose}}` changes refine's mode only: refine keeps its `max`, and every
  other kind of act keeps its defaults. `budgets: {doc-lines: 300}` keeps the
  other budgets.
- **A list or a scalar replaces the default whole.** `exempt: [wontfix]`
  drops `pinned` and `security`; `types: [feat, fix]` is the whole list.
  To add to a default list, write it out with the addition.
- **`null` removes the key**, as if the role had no default: `acts: {ask:
  null}`. The role then reads the setting as unset — a product owner's act
  kind is proposed, a documentalist's budget says `setting-missing`. To turn
  something off, prefer its own value (`mode: off`, `[]`) or `enforce: off`.

**Levels.** A role may ship named sets of its settings, `levels` in its
`role.yaml`, one setting picking one: the product owner's `autonomy`
(`cautious`, `normal`, `enterprising`; ADR-0026). The level picked is laid
over the role's defaults first, the same way, then the project's
settings over both: a project picks a level in one word and still sets
any field on its own. The role gets beside its settings `by-level`, what
they are with the project's left out, to say which the project changed.

If a setting a role depends on cannot be resolved, the role says so and blocks.
A check that silently passes for lack of configuration is still believed in,
which is worse than no check.

## Adapting a role to a project

A project whose rules differ from a role's defaults goes, in order, to the first
level that is enough — each one is heavier than the one before:

| Need | Level | Where |
|---|---|---|
| Other values: limits, patterns, paths | **settings** | `.workline/config.yaml`, `roles.<name>.settings` |
| A rule bites too hard, or is not wanted | **enforcement** | `roles.<name>.enforce`: `warn` or `off` per rule |
| The AI should write differently: language, tone, sections | **facets** | `.workline/roles/<name>/policy.md` (or any facet, or a reviewer's lens, `lenses/<lens>.md`), replacing the shipped one; a user copy in `~/.config/workline/roles/<name>/` is read after the project's and before the shipped one |
| Extra checks on top of the role's own: release only from `main`, a migration needs a note… | **gates** | the `gates:` section of `.workline/config.yaml`, run by routing before or after the role |
| Different logic: another convention, another way to check it | **a role of its own** | a folder of roles given by `--roles` or `WORKLINE_ROLES` today; `roles.<name>.from`, pinned, once built — a fork of the shipped role, or a new one |

Settings, enforcement and facets change how a role behaves without touching its
code. A gate adds checks without replacing anything. Only the last level
replaces the role's scripts, and the project then owns them — the shipped
role's conformance cases are the way to check the fork still keeps the contract.

## Defects become guards

*Not built yet: the ledger and the shared skill. Defects found so far are
guarded by tests, recorded in the role's README ("Tried for real"), or in the
page it links to.*

When a role lets a defect through, fixing the output is not enough. The
defect is recorded in the role's `DEFECTS.md` (what happened, why, the
countermeasure, the result), and it is closed only when a mechanical check —
a rule the role checks after its agent, a gate, a test — makes it hard to
repeat.

The "why" is found with the five whys: ask why the defect happened, then why
that happened, until the answer is a cause the project can act on rather than a
symptom. A countermeasure against a symptom only moves the defect elsewhere.
The method ships as a shared skill that any role can load.

A guard is trusted to block only when it is proven: it must fail on the commit
where the defect happened, and pass on the fixed tree. A guard that cannot fail
protects nothing, and a guard that fails on everything gets switched off. A rule written only in `policy.md` is a
wish; the ledger is where wishes become checks.
