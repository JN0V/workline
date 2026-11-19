---
sources:
  - internal/board/board.go#Move
  - internal/board/board.go#Move.Labels
  - internal/board/board.go#Move.Told
  - internal/board/board.go#To
  - internal/board/board.go#Along
  - internal/board/board.go#Move.Fold
  - internal/board/board.go#Move.Write
  - internal/backlog/autonomy.go#Config
  - internal/backlog/backlog.go#describe
checked: c46a93d6c
verified: agent:claude-code
---
# What each state moves

Part of [Boards](boards.md).

## What each state moves

A card follows the issue's state, its label
([states](spec/routing.md#the-line-of-states)). Who moves the issue to
each state is said there: the product owner, the tester, the developer,
or the line itself
([staleness](spec/routing.md#staleness)).

| State | On GitLab, Free and Premium |
|---|---|
| tests | the tests column's label on, beside `workline:needs-tests`; with no tests column, the ready column's |
| ready | the ready column's label on, beside `workline:ready` |
| doing | the doing column's label on |
| blocked | the blocked column's label on, beside `workline:blocked` |
| refine | the refine column's label on |
| done | the Closed list, at the close (`Closes #n`) |

- **One call, both tiers**: on Premium, a scoped label
  (`workflow::Doing`) takes its sibling off by itself; on Free, the
  role takes the other columns' labels off, in the same call: a card
  sits in one column. GitLab's tier is never asked.
- **On GitHub**, a column is an option of the project's Status field
  ([GitHub projects](board-github.md)).
- **Blocked says why** on the issue, in the comment of whoever blocked
  it; the product owner reads it again when there is news, or a week on,
  and moves it back or says why it stays
  ([blocked, read again](spec/routing.md#blocked-read-again)); a person
  may move it back by its label at any time.
- **In the role's record**: the product owner's act says the column;
  the developer's claim comment says it, and its hidden line keeps it
  ([the claim](../roles/developer/docs/pick.md)).
- **A board out of reach**, even in part, moves nothing, said in the run's output;
  no verdict changes.
- **A person's drag is theirs**: it never frees a claim, and no role
  moves a card back.
