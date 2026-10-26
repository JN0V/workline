---
type: reference
sources: [internal/backlog/change.go, internal/builtin/productowner/changes.go, internal/builtin/productowner/productowner.go, internal/backlog/autonomy.go, internal/backlog/parent.go, internal/backlog/depend.go]
status: draft
checked: edc4ffc2
verified: agent:claude-code
---
# A changed need

Part of [Acts on the backlog](../backlog-acts.md): the issues flagged when what they were built on changes.

Off by default; `changed-needs: true` turns it on (ADR-0038). A
person's change to what open issues were built on flags them (ADR-0032),
as a requirements tool marks a link suspect; with no agent, at every run:

- **An issue's Need or Scope rewritten by a person.** The state keeps
  both as the engine last read or wrote them (`sections`), the engine's
  own lines — a draft line, a blocked-by line — left out
  (`backlog.Basis`). Another text there, spaces aside, is a change
  (`backlog.Rewritten`); a section written where there was none, an edit
  elsewhere in the body, a label are not — the issue is read again for
  its body as before, and flags nothing. A state that kept none gets
  them, once, with no agent.
- **The lines of a file an issue was imported from, changed by a
  commit**: the lines its body names (`Opened from `ROADMAP.md`, lines 7
  to 8`, hidden in a marker since 2026-10-09, beside its `import=` key), carried from the commit it was
  opened at (`confirmed`) to the one it was last read at (`judged`),
  then through the commits since (`backlog.LinesChange`): lines changed,
  removed, or added between two of them are a change; lines that only
  moved are not. A commit that cannot be read — gone after a force-push,
  beyond a shallow clone — is said (`lines-unread`, warn), never read
  as no change.
- **Not a file archived**: `archived`, a list of paths or globs, names
  the files no longer a source — a roadmap whose items became issues,
  kept for its history. A change to their lines flags nothing. A value
  that is not a list of paths stops the run (principle 12).

What a change touches (`backlog.Touched`), none twice, in this order:

| Touched | What the run does |
|---|---|
| an open part of the issue (`backlog.Parts`) | read first, before the issues proposed for a cap, the sections as they were and as they are given beside it |
| the issue imported from the lines | read first, the lines as they were and as they are |
| an open issue waiting on it (`backlog.Blockers`) | listed in the job's summary, not read |
| its Scope rewritten: an open issue whose sources share a file with its own | listed in the job's summary, not read |

Every act the agent proposes on an issue read for a change is proposed,
not done (`need-changed`, info), `keep` and an `undepend` of closed blockers aside: what the change asks of it
is a person's to decide. A part past `issues-per-run`, or every part
without an agent, is listed instead of read. Nothing is
written to an issue touched. The changed issue's state keeps its new
sections at that run — after the agent's answer when an issue a change
touches was given to it —, an issue imported its new `judged` once read:
the same change flags once; an issue imported and not read flags again
at the next run.
A change that touches no open issue is kept in the state and flags
nothing. Only these direct consequences are followed (principle 9).
