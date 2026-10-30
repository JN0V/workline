---
sources: [internal/board/board.go, internal/forge/board.go, internal/doctor/board.go]
checked: 16acf2af
---
# Boards

```mermaid
flowchart LR
  forge["<b>The forge</b><br/>its board"] --> find["<b>workline</b><br/>finds it, read-only"]
  find --> match["<b>Columns matched</b><br/>ready, doing, done"]
  match --> doctor["<b>workline doctor</b><br/>says what it found"]
  match --> roles["<b>The roles</b><br/>move the cards"]
```

People follow the work on a board: To do, Doing, Done. workline finds
the forge's board each run and creates none
([ADR-0057](adr/0057-workline-finds-the-forges-board-and-moves-its-cards.md)).

- **`workline doctor`** says the board found and its columns matched.
- **The roles move its cards**: the product owner to the ready column,
  the developer to the doing column (below); GitHub's own workflows
  do what they can first.
- **workline's labels stay** (`workline:ready`, `workline:developing`):
  they are the roles' marks, board or not.
- **Only when it matters**: the doctor looks for a board when the line
  routes the product owner or the developer, the two roles that move
  cards.

## What is found, by forge

| Forge | The board | Its columns |
|---|---|---|
| GitHub | the open projects linked to the repository; their owner, an organization or a user | the options of the project's Status field |
| GitLab | the project's boards, and its group's | the labels of each board's label lists, scoped (`workflow::Doing`) or plain |
| `local`, `cmd:`, none | none | none: workline's labels only |

- **GitHub's built-in workflows** are read too: *Item closed*, *Pull
  request merged*, *Pull request linked to issue*, each on or off.
- **GitLab's tier is not asked**: Free and Premium boards read the same.
- **Done on GitLab** is the Closed list: the issue's close moves it.

## Columns matched by their names

A name is matched lowercase, letters only, a scope's key dropped
(`workflow::In progress` reads `inprogress`). In each slot, the first
name of its list wins:

| Slot | Names it takes |
|---|---|
| ready | Ready, Todo, To do, Backlog |
| doing | In progress, Doing, WIP, In development, Started |
| done | Done, Closed, Complete, Completed |

A slot with no column moves nothing.

## The `board:` setting

Only when the names above do not fit, in `.workline/config.yaml`:

```yaml
board:
  ready: "Sprint backlog"   # a column's name, matched as loosely
  doing: "Build"
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

## What each role moves

From
[ADR-0057](adr/0057-workline-finds-the-forges-board-and-moves-its-cards.md#what-each-role-writes):

| Step | Who | On GitLab, Free and Premium |
|---|---|---|
| ready | the product owner, setting an issue ready | the ready column's label on, beside `workline:ready`; the other columns' off, in one call |
| doing | the developer, at its claim | the doing column's label on, the other columns' off, in one call |
| done | the forge, closing the issue at the merge (`Closes #n`) | the Closed list |

- **One call, both tiers**: on Premium, a scoped label
  (`workflow::Doing`) takes its sibling off by itself; on Free, the
  role takes the other columns' labels off: a card sits in one
  column. GitLab's tier is never asked.
- **In the role's record**: the product owner's act says the column;
  the developer's claim comment says it, and its hidden line keeps it.
- **With no agent**, neither role acts: no card moves.
- **A board out of reach**, even in part, moves nothing, said in the run's output;
  no verdict changes.
- **A person's drag is theirs**: it never frees a claim, and no role
  moves a card back.

### On GitHub

A column is an option of the project's Status field: a move sets that
one value, and puts no label for it.

| Step | An organisation's project | A user's project |
|---|---|---|
| ready | the product owner sets Status to the ready option | no move: GitHub lets no App write it |
| doing | left to *Pull request linked to issue* when on, the pull request opening; else the developer sets Status | left to *Pull request linked to issue*; off, no move |
| done | *Item closed*, at the close | the same |

- **The App's right**: on an organisation's project, *Organization
  projects: read and write*. GitHub offers no such right on a user's
  project
  ([ADR-0057](adr/0057-workline-finds-the-forges-board-and-moves-its-cards.md#who-may-write-a-github-project)).
- **An issue in no project** is not added: that is the project's
  *Auto-add* workflow.
- ***Item closed* off**: no card reaches Done. The role that moves a
  card says it once a run, `board-workflow-off` (info), naming the
  project.
- **In the role's record**: the move, or why it is left (the
  workflow, a user's project). A write the forge refuses is said in
  the run's output; no verdict changes.
- **Turning a workflow on**: the project's menu, *Workflows*.

## What the doctor says

| Rule | Level | Means |
|---|---|---|
| `board-found` | ok | the board, each slot's column, GitHub's workflows on and off |
| `board-none` | note | no board, or none with a column workline can set |
| `board-ambiguous` | note | several boards; the fix names the setting |
| `board-setting-unmatched` | warn | `board:` names a column or project that is not there |
| `board-unread` | warn | the forge did not answer; the reason, never read as no board |

- **GitHub, `board-unread`**: a person's token needs the `read:project`
  scope: `gh auth refresh -s read:project`. A GitHub App needs
  *Organization projects: read*; it reads no project a user owns.
