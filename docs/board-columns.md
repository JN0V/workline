---
sources:
  - internal/board/board.go#names
  - internal/board/board.go#Order
  - internal/board/board.go#Key
  - internal/board/board.go#setting
  - internal/board/board.go#Match
  - internal/board/board.go#Choose
  - internal/role/role.go#Board
  - internal/doctor/board.go#boards
  - internal/forge/board.go#gitlab.boards
checked: c46a93d6c
verified: agent:claude-code
---
# Board columns

Part of [Boards](boards.md).

## Columns matched by their names

A name is matched lowercase, letters only, a scope's key dropped
(`workflow::In progress` reads `inprogress`). In each slot, the first
name of its list wins:

| Slot | Names it takes |
|---|---|
| ready | Ready, Todo, To do, Backlog |
| doing | In progress, Doing, WIP, In development, Started |
| done | Done, Closed, Complete, Completed |
| refine | Refine, To refine, Refinement, Triage, New, Backlog, Icebox |
| tests | Tests, Test, Needs tests, To test, Acceptance tests |
| blocked | Blocked, On hold, Impeded |

- **One column, one slot**: the slots take their columns in the
  table's order, a column taken by one never another's. Backlog is
  ready's on a board with no Ready nor To do column, refine's beside one.
- **A slot with no column** moves nothing. *Testing* and *QA* are not
  taken: on many boards they come after the code, where workline's
  tests come before it; name them in `board:` if they are yours.

## The `board:` setting

Only when the names above do not fit, in `.workline/config.yaml`:

```yaml
board:
  ready: "Sprint backlog"   # a column's name, matched as loosely
  doing: "Build"
  tests: "QA"               # refine, tests and blocked too
  project: 3                # GitHub: a project's number, when several are linked
```

- **A name no column holds**: a warning, `board-setting-unmatched`; that
  step moves nothing.
- **Several projects, no `project:`**: none is chosen, a note,
  `board-ambiguous`. GitLab boards listing the same labels count as
  one: a label is the same move on each. `project:` applies on GitHub
  only.
- **GitLab, a group's boards unread**: a warning; the project's own
  boards still count.
