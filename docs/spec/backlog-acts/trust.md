---
type: reference
sources: [internal/backlog/undo.go, internal/backlog/backlog.go, internal/backlog/acts.go, internal/sample/acts.go, internal/sample/draw.go, internal/backlog/autonomy.go, internal/backlog/ledger.go]
status: draft
checked: 434f7e39
verified: agent:claude-code
---
# Trust

Part of [Acts on the backlog](../backlog-acts.md): the acts a person undoes, found and counted.

A closing is wrong when its issue is open again: the closed issue's state
keeps the kind it was closed with (`closed`); found open at a later run,
that is an act undone (`wrong-closing`), the issue read again.

Any other act of the role's a person undoes is found the same way, at the
next run — with or without an agent — from its issue's `done` and the
forge, the evidence in the finding `undone`:

| Act | Undone when |
|---|---|
| rename | the issue's title is the one it had before |
| order | its priority is the one it had before (none included) |
| milestone | its milestone is the one it had before (none included) |
| ready | `workline:ready` is no longer on the open issue |
| split | a child listed in the parent's state is closed as not planned (GitHub; GitLab keeps no reason) |
| depend | a blocker it added, still open, is no longer among the issue's blockers: the link or the line taken off |

An act undone is kept in its issue's state (`undone`): that kind is
proposed on that issue from then on (`undone-here`, ADR-0038, amended).
Undone `undone-max` times (3) across the open issues' states, closings
reopened included, the kind is proposed on every issue (`demoted`) until
fewer are. A title, priority or milestone a person set to a third value
is theirs: the act is no longer watched. An act whose issue is closed is
no longer watched; a split, once each child is closed or gone from the
forge; a depend, once each blocker it added is closed — the link then
taken off by the engine ([What an issue waits on](depend.md)). A move the engine
made itself (a slip) and an act a person accepted are not the role's
choice, and are not watched.

## The weekly sample of its acts

Part of the weekly sample of every role's acts, with no setting
(ADR-0046, folding in ADR-0033's issue of its own; `weekly-sample` is no
longer read, `setting-gone`). `workline sample` reads every issue's
state, open and closed — one forge call an issue — for the acts done
alone (`did`, a person's yes left out): of those whose day falls in the
week sampled, one in ten of each kind, rounded up, drawn the same on a
rerun, each re-checked by the auditor on what its record says it read
(`at`, `model`, `body`; [The issue's state](state.md)). Each says, when
a person undid it since, what shows it (a closing reopened, the state's
`undone`); an undo counts against one act, the newest of its issue and
kind done by the day it was found.

Over the acts done alone at the level of the last one, up to the week's
end, 10 or more: more than one in ten undone suggests the level below;
none undone at `normal` suggests `enterprising` (`acts-suggest`), said
in the week's comment. The sample never changes the setting. The
role's former tracking issue, "workline: the weekly sample of the product
owner's acts", still open, is told once where the weeks went, and closed
(`sample-folded`).
