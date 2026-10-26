---
sources: [roles/product-owner/role.yaml, internal/backlog/autonomy.go, internal/backlog/ledger.go, internal/sample/acts.go, internal/scale/detect.go]
checked: 7fd48f92
verified: agent:claude-code
---
# Product owner — settings

Part of [the product owner](../README.md). Under
`roles: {product-owner: {settings: …}}` in `.workline/config.yaml`
([config reference](../../../docs/config.md)). A key the project sets wins
over the level, field by field.

## The role's own

| Key | Default | Says |
|---|---|---|
| `autonomy` | `normal` | how far it goes alone: `cautious`, `normal`, `enterprising` (below) |
| `issues-per-run` | `8` | issues read a run |
| `code-lines-max` | `1500` | lines of code given a run, all files together |
| `proposals-max` | `10` (1 to 100) | issues waiting on a person before it reads only those a person commented on, and labels no more |
| `undone-max` | `3` (1 to 20) | acts of a kind a person undoes, across the issues, before that kind is proposed on every issue |
| `next-max` | `5` (0 to 20) | ready issues the summary lists as next |
| `stuck-days` | `14` (1 to 365) | days an issue waits on a person before the summary says it stuck |
| `archived` | `[]` | files no longer a source: a change to their lines flags nothing |
| `changed-needs` | `false` | a changed Need or Scope flags the issues built on it ([ADR-0032](../../../docs/adr/0032-a-changed-need-flags-the-issues-built-on-it.md)) |
| `moved-percent-max` | `20` | with `milestone` or `order` on: the share of the open issues a run moves |
| `repeated-percent-max` | `40` (1 to 100) | an example or steps whose words are past this share, in percent, already said above it is not written ([each thing said once](writing.md#each-thing-said-once)) |
| `priority`, `effort` | unset | where the project keeps each: unset, workline's own labels and no effort; `auto` finds it; `{in: labels, prefix: P}` names a family ([below](#priority-and-effort)) |

No longer read, each said by the run:

- `ignored-runs-max`: the role never pauses
  ([ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).
- `weekly-sample`: the weekly sample of every role's acts draws the
  role's, with no setting
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)).
A project still setting it is told so in every run.

## Each kind of act

`acts.<kind>.mode` is `act` (done, up to `max` a run), `propose` (on the
issue, for a person to accept) or `off` (dropped).

| Kind | `normal` | `cautious` | `enterprising` |
|---|---|---|---|
| `sources` | act, max 10 | | |
| `refine` | act, max 5 | Need and Validation drafts proposed (`drafts: propose`) | max 10 |
| `ask` | act, max 3, `rounds: 3` | max 2, `rounds: 2` | max 5 |
| `ready` | act, max 5 | propose | max 10 |
| `close-duplicate` | act, max 3 | propose | max 5 |
| `open` | act, max 30 (importing) | | |

An empty cell is the `normal` value. `ask.rounds` also bounds the
revisions a person's comments get: the first draft and two more.

## Off by default

The code stays; each comes back with a mode
([ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)):

| Kind | Turned on, does |
|---|---|
| `milestone` | puts an issue in its release's milestone; moves one that slipped |
| `order` | sets an issue's priority, `workline:priority/1` to `/4` unless `priority` says where the project keeps it; a person's kept |
| `effort` | sets an issue's effort, Low to High, once, where `effort` says the project keeps it, on code the run read ([below](#priority-and-effort)) |
| `split` | splits a need too big into 2 to 6 issues ([ADR-0022](../../../docs/adr/0022-a-split-keeps-the-need-a-rename-keeps-a-persons-title.md)) |
| `rename` | renames a vague title; a person's kept |
| `depend` | says what an issue waits on ([ADR-0028](../../../docs/adr/0028-an-issue-names-what-it-waits-on.md)) |
| `close-obsolete` | announces an obsolete issue, closes it `days` later (7) on silence and a second judge's yes ([ADR-0024](../../../docs/adr/0024-obsolete-is-announced-then-closed-on-silence-and-a-second-judge.md)) |

Off, `close-obsolete` still speaks: an issue the code already does is
said so on the issue, the code quoted, for you to close; nothing is
announced or closed.

```yaml
roles:
  product-owner:
    settings:
      autonomy: cautious
      acts: {order: {mode: propose}, close-obsolete: {mode: act}}
```

## Priority and effort

Where the project keeps each issue's priority and effort
([ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md),
[examples](priority-and-effort.md)). Unset by default: workline's own
labels, and no effort read.

| Value | Reads and writes |
|---|---|
| `auto` | a GitHub organisation's own Priority and Effort fields; the effort in GitLab's weight (Premium, Ultimate); else the project's own labels (`P0`…, `P-*`, `priority/*`, `priority::*`, `effort*`), else workline's own, else the forge's: `priority/urgent` to `/low` on GitHub, `priority::1` to `::4` on GitLab, created at the first act |
| `{in: labels, prefix: P}`, `{in: field, name: Severity}`, `{in: weight}` | that family, that field, or GitLab's weight (the effort only); the run stops if the project has none |
| `names: {sev-1: urgent}`, beside either | a label no table knows, read as that level |

- **Said once a run**: where each was found, and a name no table knows.
- **One place each**: an issue holding a priority in another family, or
  in a field beside the labels, gets nothing written.
- **Fields visible to members only**, on a public repository: the values
  are set but never shown — not in the role's comment, its state, nor the
  job's summary — and never proposed. A token not shown to read them, the
  job's own or an App's, writes nothing there
  ([why](../../../docs/research/priority-and-weight.md#checked-live-2026-10-18)).
- **GitLab's weight**: read 0–1 Low, 2–3 Medium, 4 or more High; written
  1, 3 or 5; never a parent's, which GitLab sums from its parts'.
- **Set once, only where empty**: a value a person set stays; one the
  role set and a person changed or cleared is never written again.
- **The order** goes by the priority, then the effort, the lightest
  first; Next shows each issue's cell, "High · Effort Low → quick win".
- **An effort** needs the `effort` act on (`max` 10 a run, 20 at
  `enterprising`), and is estimated only on an issue whose code the run
  read.

```yaml
roles:
  product-owner:
    settings:
      priority: auto
      effort: {in: labels, prefix: "effort/", names: {effort/xl: high}}
      acts: {order: {mode: act}, effort: {mode: propose}}
```
