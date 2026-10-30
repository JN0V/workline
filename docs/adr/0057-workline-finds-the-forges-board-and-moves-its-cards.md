# ADR-0057: workline finds the forge's board and moves its cards where the forge does not

- **Status:** accepted (2026-10-30)
- **Date:** 2026-10-30
- **Decided by:** the maintainer, on 2026-10-30, with the
  [answers](#answers) below
- **Builds on:** [ADR-0018], [ADR-0023], [ADR-0040], [ADR-0054];
  principles 5, 12, 13, 14
- **Amends:** [ADR-0054]'s "never moves an issue": the claim moves its
  card to Doing; [ADR-0018]: setting an issue ready moves it to Ready
- **Research:** [forge-boards.md](../research/forge-boards.md)

## Context

- **People follow the work on a board**: To do, Doing, Done. workline
  marks an issue with its own labels (`workline:ready`,
  `workline:developing`); no column follows them.
- **Boards differ by forge and tier**
  ([research](../research/forge-boards.md)):
  - GitHub Projects: a Status single-select field; built-in workflows
    set Done on close and merge, In progress on a linked pull request.
  - GitLab Premium: a column is a scoped label (`workflow::doing`); the
    forge takes the other `workflow::` label off.
  - GitLab Free: a column is a plain label; nothing comes off alone.
  - None: workline's labels only.
- **The person's answer**: every one of these, found by workline, with
  no setting in the common case.
- **Principle 13**: use the forge's automations first; complete them
  where they stop.

## Decision

### Find the board, each run, read-only

| Forge | What tells the board | Its columns |
|---|---|---|
| GitHub | `Issue.projectItems` (the projects the issue sits in); `Repository.projectsV2` (the projects linked to the repository) | the Status field: `ProjectV2.field(name: "Status")`, a `ProjectV2SingleSelectField`, its `options { id name }` |
| GitHub, its automations | `ProjectV2.workflows { name enabled }` | *Item closed*, *Pull request merged*, *Pull request linked to issue* |
| GitHub, who owns it | `ProjectV2.owner { __typename }`: `Organization` or `User` | — |
| GitLab | GraphQL `project.boards`, and its group's `group.boards` | each board's `lists { listType label { title } }`: a `label` list is a column; `backlog` is Open, `closed` is Closed; `hideBacklogList`, `hideClosedList` |
| None | no project, or no `label` list on any board | workline's labels, as today |

- **Docs**: GitHub's
  [ProjectV2 objects](https://docs.github.com/en/graphql/reference/objects#projectv2)
  and [the Projects API](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects);
  GitLab's [Board](https://docs.gitlab.com/api/graphql/reference/#board)
  and [BoardList](https://docs.gitlab.com/api/graphql/reference/#boardlist)
  types, its [boards API](https://docs.gitlab.com/api/boards/).
- **GitLab's tier is not asked**: the plan (`GET /namespaces/:id`'s
  `plan`, `GET /license`) is not readable by every token, and no step
  needs it. A move puts the new label on and takes the old one off:
  on Premium, the forge has already taken it off; on Free, the role
  does. Same call, both tiers
  ([scoped labels](https://docs.gitlab.com/user/project/labels/#scoped-labels),
  [edit an issue](https://docs.gitlab.com/api/issues/#edit-an-issue)).
- **Several boards or projects**: the one the issue sits in; else the
  only one linked; else none chosen, said once, `board-ambiguous`
  (info).

### Columns matched by their names, loosely

- **Lowercase, letters only**, the scope's key dropped
  (`workflow::In progress` → `inprogress`):

| Slot | Names it takes, first match wins |
|---|---|
| ready | Ready, Todo, To do, Backlog |
| doing | In progress, Doing, WIP, In development, Started |
| done | Done, Closed, Complete, Completed |

- **One optional setting overrides the match**, in `.workline/config.yaml`,
  shared by the roles:

```yaml
board:
  ready: "Sprint backlog"   # a column's name
  doing: "Build"
  project: 3                # GitHub: a project's number, when several
```

- **A slot with no column**: that step writes nothing, and says nothing.

### What each role writes

| Step | Role | GitHub Projects | GitLab Premium | GitLab Free | None |
|---|---|---|---|---|---|
| Ready | the product owner, setting an issue ready ([ADR-0018]) | Status → the ready option (`updateProjectV2ItemFieldValue`) | the ready label on | the ready label on | `workline:ready`, as today |
| Doing | the developer, at its claim ([ADR-0054]) | nothing if *Pull request linked to issue* is on (its pull request opens in the same `apply`); else Status → the doing option | the doing label on | the doing label on, the ready label off | `workline:developing`, as today |
| Done | the forge's close, `Closes #n` at the merge | *Item closed*, *Pull request merged*; nothing written | the Closed list | the Closed list | the close |

- **workline's own labels stay** on every board: they are the roles'
  marks; a person dragging a card never frees a claim.
- **The board is never created nor extended**: a missing column is no
  step; an issue in no project is not added (*Auto-add* is the
  forge's).
- **A Done workflow off** on GitHub: `board-workflow-off` (info), naming
  the project and the workflow to turn on, once a run.
- **A person's drag is theirs**: no role moves a card back.
- **With no agent**: the product owner and developer act only through
  one; nothing moves (principle 5).

### Who may write a GitHub project

- **An organisation's project**: the App's `Organization projects` read
  and write permission. Without it, `board-unwritable` (info), naming
  the permission.
- **A user's project**: GitHub gives an App no Projects permission on a
  user account
  ([App permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)).
  Then:
  - **By default**: the built-in workflows move Doing and Done; Ready
    does not move. Said once, `board-user-project` (info), the summary
    naming the workflows to turn on.
  - **No opt-in token, for now**: a person's classic token with the
    `project` scope, GitHub's advice for user projects
    ([here](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/automating-projects-using-actions)),
    is not taken ([answers](#answers)).
- **Never read as a pass nor a failure**: a board out of reach changes
  no verdict.

## Answers

The maintainer's answers, on 2026-10-30:

- **A user's GitHub project**: no opt-in personal token for now; rely
  on its built-in workflows, and report what does not move.
- **An issue in no project**: unchanged, left to *Auto-add*, as
  decided above.

## Build plan

One pull request a step, its cases first in `tests/conformance/cases/`,
`pending.txt` listing what does not pass. The fake forge holds boards.

1. **Detection and matching**, no write: the engine reads the board and
   prints it in `workline doctor`. Cases: `board-github-status-found`,
   `board-gitlab-lists-found`, `board-none-found`,
   `board-names-matched-loosely`, `board-setting-overrides-match`,
   `board-ambiguous-said`.
2. **GitLab moves**, Free and Premium in one: the product owner's ready
   label, the developer's doing label. Cases:
   `board-gitlab-ready-label-on`, `board-gitlab-doing-label-on`,
   `board-gitlab-free-ready-label-off`, `board-no-slot-moves-nothing`,
   `board-no-agent-moves-nothing`.
3. **GitHub moves**: Status written; the linked-pull-request workflow
   used when on. Cases: `board-github-ready-set`,
   `board-github-doing-left-to-workflow`, `board-github-doing-set`,
   `board-workflow-off-said`.
4. **Who may write**: `board-unwritable-said`,
   `board-user-project-said`.
5. **Docs**: a page under `docs/` (boards, each forge), linked from
   both roles' READMEs and the install guide.
6. **Tried**, told in each pull request: a GitLab Free and a Premium
   board, an organisation's and a user's GitHub project on the
   sandbox; whether *Pull request linked to issue* fires for a pull
   request the App opens.

[ADR-0018]: 0018-the-product-owner.md
[ADR-0023]: 0023-the-project-bot-keeps-a-gitlab-backlog.md
[ADR-0040]: 0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
