---
sources:
  - internal/board/board.go#Move
  - internal/board/board.go#LinkedWorkflow
  - internal/board/board.go#ClosedWorkflow
  - internal/board/board.go#Move.Findings
  - internal/board/board.go#Move.Labels
  - internal/board/board.go#Move.Writes
  - internal/board/board.go#Move.Told
  - internal/board/board.go#To
  - internal/board/board.go#userSaid
  - internal/board/board.go#Move.Write
  - internal/forge/board.go#SetStatus
  - internal/forge/board.go#notInProject
  - internal/forge/board.go#github.setCardStatus
  - internal/forge/board.go#projectUnwritable
  - internal/engine/backlog.go#applier.moveCard
checked: c46a93d6c
verified: agent:claude-code
---
# GitHub projects

Part of [Boards](boards.md).

## On GitHub

A column is an option of the project's Status field: a move sets that
one value, and puts no label for it.

| Step | An organisation's project | A user's project |
|---|---|---|
| tests, ready, blocked, refine | the role that moves the issue sets Status to that option | no move: GitHub lets no App write it |
| doing | left to *Pull request linked to issue* when on, the pull request opening; else the developer sets Status | left to *Pull request linked to issue*; off, no move |
| done | *Item closed*, at the close | the same |

- **An issue in no project** is not added: that is the project's
  *Auto-add* workflow.
- **In the role's record**: the move, or why it is left (the
  workflow, a user's project).

## Who may write a GitHub project

- **An organisation's project**: the App needs *Organization projects:
  read and write*; a person's token, the `project` scope. Refused, the
  card stays and the run says `board-unwritable` once.
- **A user's project is never written**: GitHub offers an App no right
  on a user's projects
  ([App permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)).
  A person's token could, but workline takes none for it
  ([ADR-0057](adr/0057-workline-finds-the-forges-board-and-moves-its-cards.md#who-may-write-a-github-project)):
  - the ready moves no card;
  - the project's workflows move Doing and Done;
  - the run says `board-user-project` once, naming the workflows off.

## What a run says

The role that moves a card says each once a run, an info: no verdict
changes.

| Rule | Means |
|---|---|
| `board-workflow-off` | an organisation's project with *Item closed* off: no card reaches Done |
| `board-user-project` | a user's project: the ready moves no card; the workflows to turn on |
| `board-unwritable` | an organisation's project the token may not write; the right to give |
