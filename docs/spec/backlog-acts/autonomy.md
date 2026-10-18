---
type: reference
sources: [internal/backlog/autonomy.go, roles/product-owner/role.yaml, internal/backlog/backlog.go, internal/builtin/productowner/productowner.go]
status: draft
checked: a9762f04
verified: agent:claude-code
---
# Autonomy and caps

Part of [Acts on the backlog](../backlog-acts.md): how far the role goes alone: its level, each kind's mode and cap.

How far the role goes is one setting, `autonomy` (ADR-0038, replacing
ADR-0026's table): `cautious`, `normal` (the default) or `enterprising`,
a preset of each kind's mode and cap. `normal` is the role's defaults:

```yaml
autonomy: normal
issues-per-run: 8
proposals-max: 10                           # issues waiting on a person, then only answers read; 1 to 100
undone-max: 3                               # acts of a kind undone across the issues, then proposed everywhere; 1 to 20
next-max: 5                                 # the ready issues the summary lists, 0 to 20; 0 none
stuck-days: 14                              # an issue waiting on a person longer is said stuck, 1 to 365
changed-needs: false                        # "A changed need"
acts:
  open: {mode: act, max: 30}
  sources: {mode: act, max: 10}
  refine: {mode: act, max: 5}
  ready: {mode: act, max: 5}
  ask: {mode: act, max: 3, rounds: 3}       # rounds: written to a reporter, or revisions, then a person
  close-duplicate: {mode: act, max: 3}      # act | propose | off
  milestone: {mode: off, max: 10}           # off by default, the code kept (ADR-0038)
  order: {mode: off, max: 10}
  close-obsolete: {mode: off, max: 3, days: 7, exempt: [pinned, security]}
  split: {mode: off, max: 2}
  rename: {mode: off, max: 5}
  depend: {mode: off, max: 5}
```

- `act`: done, up to `max` a run; past it, proposed — but a question to
  a reporter (`ask`), the role's to ask at a later run: held, never a
  person's to answer, no label nor line on the issue, its issue read
  again first (`Pending.Held`, #430).
- `propose`: proposed on its issue for a person to accept, not done.
- `off`: dropped; a closing off is said on its issue for a person to close
  ([Closing](closing.md)).

With `milestone`, `order` or `effort` on, a run moves at most a share of
the open issues: `moved-percent-max` (20, a fifth, by default; rounded
up, one at least). Milestones — the engine's slips included —,
priorities and efforts count together, an issue moved twice once; past
the share, a move is proposed (`moved-cap`). Milestones are applied
before priorities.

Closing as obsolete, turned on as `act`, announces first, and closes only
on silence and a second judge's yes (ADR-0024). Its `max` counts
announcements and closings together, the engine's closings first; a
closing past it stays announced and is closed at a later run.

## The levels

The other levels change these (role.yaml, `levels`):

| Setting | cautious | normal | enterprising |
|---|---|---|---|
| sources | act 10 | act 10 | act 10 |
| refine | act 5, Need and Validation drafts proposed (`drafts: propose`) | act 5 | act 10 |
| ask | act 2, 2 rounds | act 3, 3 rounds | act 5, 3 rounds |
| ready (the engine's check: four sections, no draft) | propose | act 5 | act 10 |
| close-duplicate | propose | act 3 | act 5 |
| undepend | propose; a closed blocker's: the engine's, done | the same | the same |
| open (import) | act 30 | act 30 | act 30 |

`cautious` notes and proposes: the code named, Scope and Verification
written, the Need and Validation drafts proposed on the issue
(`drafts-proposed`) — unless a person's already: agreed to in a reply,
accepted by the label, or proposed to an outsider in a comment.
`normal` completes and moves an evident issue to `ready` alone;
`enterprising` does the same, more a run. The level sets what is done with
what is read, not how much is read.

**Precedence**: the level, then a kind the project sets, field by field
(`acts: {rename: {mode: act}}` at `cautious` keeps the role's cap); then
the issue: one set aside gets nothing, one where a person undid that kind
gets a proposal (`undone-here`); then a kind undone `undone-max` times,
proposed everywhere (`demoted`); then a person's yes on one act
(`workline:accepted`), done whatever its mode or cap. The task given to
the agent says what the level means, and lists each kind's mode and where
it comes from — `level`, `setting` (differs from the level's) or
`demoted`. It says whose each section is: `(draft)`, the role's draft;
Need and Validation not marked a draft, `(a person's)`; an issue whose
four sections are there, Need and Validation a person's, not ready and
`ready` not off, is said open to `ready` when evident — the agent judges
it, the engine checks it, and an outsider's is proposed. Its reporter is
said of the project or not. A level the role does
not have, a number out of its bounds, `changed-needs` not true or false,
or `archived` not a list of paths, stops the run; `ignored-runs-max` and
`weekly-sample`, no longer read, are said (`setting-gone`).
