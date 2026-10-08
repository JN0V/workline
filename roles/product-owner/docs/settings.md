---
sources: [roles/product-owner/role.yaml, internal/backlog/autonomy.go, internal/backlog/ledger.go, internal/sample/acts.go]
checked: f8fffea
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
| `proposals-max` | `10` (1 to 100) | issues waiting on a person before it reads only those answered |
| `undone-max` | `3` (1 to 20) | acts of a kind a person undoes, across the issues, before that kind is proposed on every issue |
| `next-max` | `5` (0 to 20) | ready issues the summary lists as next |
| `stuck-days` | `14` (1 to 365) | days an issue waits on a person before the summary says it stuck |
| `archived` | `[]` | files no longer a source: a change to their lines flags nothing |
| `changed-needs` | `false` | a changed Need or Scope flags the issues built on it ([ADR-0032](../../../docs/adr/0032-a-changed-need-flags-the-issues-built-on-it.md)) |
| `weekly-sample` | `false` | the weekly sample draws a week's acts done alone ([ADR-0033](../../../docs/adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)) |
| `moved-percent-max` | `20` | with `milestone` or `order` on: the share of the open issues a run moves |

`ignored-runs-max` is no longer read: the role never pauses
([ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).
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
| `order` | sets `workline:priority/1` to `/4`; a person's kept |
| `split` | splits a need too big into 2 to 6 issues ([ADR-0022](../../../docs/adr/0022-a-split-keeps-the-need-a-rename-keeps-a-persons-title.md)) |
| `rename` | renames a vague title; a person's kept |
| `depend` | says what an issue waits on ([ADR-0028](../../../docs/adr/0028-an-issue-names-what-it-waits-on.md)) |
| `close-obsolete` | announces an obsolete issue, closes it `days` later (7) on silence and a second judge's yes ([ADR-0024](../../../docs/adr/0024-obsolete-is-announced-then-closed-on-silence-and-a-second-judge.md)) |

```yaml
roles:
  product-owner:
    settings:
      autonomy: cautious
      acts: {order: {mode: propose}, close-obsolete: {mode: act}}
```
