---
type: reference
sources: [internal/backlog/depend.go, internal/backlog/backlog.go, internal/forge/forge.go, internal/engine/backlog.go]
status: draft
checked: 434f7e39
verified: agent:claude-code
---
# What an issue waits on

Part of [Acts on the backlog](../backlog-acts.md): naming the issues an issue waits on, and taking a link off.

```yaml
- depend: {issue: 14, blocked-by: [13], why: "its test needs #13's fixed row count"}
```

An issue that cannot start before another is done names it (ADR-0028):

- **Kept** in the forge's own relation where it has one — GitHub's issue
  dependencies, GitLab's `is_blocked_by` link (Premium and up) — through
  `AddBlocker`; elsewhere — GitLab Free, which refuses the link for its
  license, the local forge, a plugged forge answering `{native: false}` —
  a line in the body, after its text: `Blocked by #13, #15.
  <!-- workline:blocked-by -->`, the one line the engine rewrites as it
  adds a blocker; the issue's state gets the body's digest, so the
  engine's own line is not read as a person's change.
- **Read** from both, on every forge: the relation, as the open issues are
  listed (`Issue.BlockedBy`; GitHub asks only the issues its listing says
  are blocked, GitLab one GraphQL query — an instance that answers it with
  errors has none), and every line of a body starting with "Blocked by"
  (case aside, a colon allowed) followed by issue references, a person's
  own line included (`backlog.Blockers`).
- **Checked**: the issue open (`no-state`); each blocker open, not an old
  report, not the issue itself, 1 to 5 (`depend-issue`); only the blockers not there
  already are written, none left drops it (`depend-same`); one that would
  close a cycle with the relations there and those this run set is
  dropped (`depend-cycle`); its state must read; no quote. It does not
  count in the moved share.
- **A split's child** names the siblings it waits on by their place in
  the split, from 1 (`after: [1]`); checked with the split (`split-after`:
  a place out of range, itself, or a cycle among them), written once the
  children are opened, as `depend` writes it.
- **The role takes off only its own.** A relation a person set — a link,
  their own line — is read and kept, never removed; one the role set and
  a person took off, its blocker still open, is an act undone ([Trust](trust.md)).
  The role's own are told from a person's by what it keeps: the blockers
  of each `depend` in the issue's state (`done`), those a split set among its
  children in the parent's state (`after: {12: [11]}`), and the engine's
  marked line in a body (`Ledger.OwnBlockers`). A parent closed first
  keeps its state read (#393): found through a child that waits and
  names it — the marker it was opened with, or its first line `Part of
  #9.` —, or an old report that recorded its split; read while a child
  that waits is open, kept watched when it does not read, let go once the
  forge no longer gives the parent. The task marks them "set by the role".

```yaml
- undepend: {issue: 14, blocked-by: [13], why: "#14 no longer reads #13's row count"}
```

A link the role set whose reason is gone is taken off (ADR-0028,
amended):

- **Its blocker closed** — completed, not planned, a duplicate, or gone
  from the forge: the wait is over. At each run, with no agent, `pre`
  proposes the engine's own `undepend` for each such link
  (`Ledger.Stale`); it is done whatever the level, never capped nor read as
  a sample's act. A blocker closed without delivering — not planned, a
  duplicate, gone — is said (`blocker-not-delivered`, warn): whether the
  issue still stands without it is a person's to say.
- **Its blocker still open**, the agent judging it no longer holds: an
  `undepend` always proposed on the issue, done when a person accepts it.
- **Checked**: the issue open, 1 to 5 blockers, each still one of its
  blockers — one gone already left out, none left drops it
  (`undepend-gone`) — and each the role's own; a person's drops it
  (`undepend-theirs`). No quote.
- **Taken off** through `RemoveBlocker` for those in the forge's relation
  (GitHub's `DELETE …/dependencies/blocked_by/{id}`, GitLab's link
  deleted; a plugged forge's `remove-blocker`, asked only for a link it
  keeps), and out of the engine's line for the rest — the line taken out
  when none is left, the state given the body's digest. The state's
  `done` loses them: a blocker opened again is not read as a person
  taking the link off.

Not done: a relation across projects; a split's links set before the
parent's state kept them (`after`), and a `depend` the state dropped
once all its blockers closed, both unknown to the role and left.
