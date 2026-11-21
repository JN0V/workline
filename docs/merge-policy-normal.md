---
sources:
  - internal/steer/codeowners.go#CodeownersPlaces
  - internal/steer/steer.go#Agent
  - internal/engine/code.go#steersAny
  - internal/doctor/guard.go#guardedList
  - internal/engine/merge.go#mergePolicy
checked: bd254cd75
verified: agent:claude-code
reader: efab3a2a0b35
judged-in-parts: 87b34bc
---
# The merge policy at `normal`

Part of [the merge policy](merge-policy.md): at `normal`, workline's own
routine pull requests merge with no person's yes; everything else waits
for one, as at `cautious`
([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md)).

```yaml
merge: {autonomy: normal}   # a project's choice, never a default
```

- **Routine work with no yes**: which of workline's pull requests merge
  alone, their kinds, the record, and the reader's rewordings, which
  wait for a yes ([routine work](merge-policy-routine.md)).
- **A revert demotes its kind**: the App's mark, the demotion issue, and
  the set-up ([reverts](merge-policy-reverts.md)).
- **The developer's code**: a fix alone, a feature with a yes
  ([the developer's code](merge-policy-code.md)).

## What never merges alone

- **The files that define oversight and authority**, listed in the
  project's `CODEOWNERS`: a pull request changing one waits for a
  person's `workline:accepted`, whatever its kind
  ([ADR-0068 §4](adr/0068-the-line-steps-back-on-its-own-drift.md#4-the-guard-is-guarded)).
  workline's own list: [CODEOWNERS](../CODEOWNERS).
- **One list**: the engine holds every path `CODEOWNERS` gives an owner
  (a folder named with or without a trailing `/`, all of it), and a
  `CODEOWNERS` file itself, added or deleted, nothing else. It reads the first found:
  in the order the forge documents, chosen from the forge the engine runs on —
  GitHub: `.github/`, the root, `docs/`; GitLab: the root, `docs/`, `.gitlab/`; on the base, never the
  pull request's own copy.
- **Other role prompts take the normal path**: the reviewer reads them,
  the auditor samples the acts.
- **With no `CODEOWNERS`**: the files that steer agents —
  [AGENTS.md](../AGENTS.md), CLAUDE.md and their likes, the pages they
  link to or import. `workline doctor` says which list it reads.
- **GitHub's required code-owner review stays off** with one owner: an
  author cannot approve their own pull request. A second owner may turn
  it on.
- **CI and workline's config** are held apart, whatever the list says:
  never workline's routine docs, never the developer's
  ([its bounds](../roles/developer/docs/bounds.md)).
- **Never alone** besides: the release tool's, a fork's, and more
  ([routine work](merge-policy-routine.md#routine-work-with-no-yes)).

## What the summary says

As at `cautious` ([each word](merge-policy.md#what-the-summary-says)),
and:

| Said | When |
|---|---|
| `waits-for-a-yes` | the release tool's, or one from a fork; the others: [routine work](merge-policy-routine.md#what-the-summary-says), [the developer's code](merge-policy-code.md#what-the-summary-says) |
| `merge-app-unknown` | the token is no App's: nothing merges alone |
| `merge-demoted` and the other words of a revert | [reverts](merge-policy-reverts.md#what-the-summary-says) |

## Where it is read

- **The policy and its level**: `merge.go` reads the level the project set.
- **The paths `CODEOWNERS` gives an owner**: `codeowners.go` lists them.
- **With no `CODEOWNERS`, the files that steer agents**: `steer.go` lists
  them.
- **A pull request changing one**: `code.go` checks the pull request's files
  against that list.
- **The list `workline doctor` says**: `guard.go` writes it.
