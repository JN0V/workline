# ADR-0057: A board moves by the forge's own workflows, and by a label at the claim

- **Status:** proposed
- **Date:** 2026-10-30
- **Builds on:** [ADR-0018], [ADR-0023], [ADR-0040], [ADR-0054];
  principles 5, 12, 13, 14
- **Amends:** [ADR-0054]'s "never moves an issue": it moves its card at
  the claim, by a label, where the project names one
- **Research:** [forge-boards.md](../research/forge-boards.md)

## Context

- **People follow the work on a board**: To do, Doing, Done. workline
  marks an issue with its own labels (`workline:ready`,
  `workline:developing`); no column follows them, so the board never
  advances.
- **The forges already move most of it**
  ([research](../research/forge-boards.md)):
  - GitHub Projects: an issue closed or a pull request merged sets
    Status to Done, on by default; *Pull request linked to issue* sets
    In progress, a new project's default since 2025-11.
  - GitLab: a column is a label; a closed issue shows in the Closed
    list. Nothing moves to Doing on its own.
- **The developer claims and opens its pull request in one `apply`**
  ([ADR-0054]), the pull request saying `Closes #n` against `main`: on
  GitHub, the link is the forge's signal for In progress.
- **workline's App cannot write a user's project**: GitHub's Projects
  permission is an organisation's. JN0V's repositories are a user's.
- **Principle 13**: use the forge's mechanism; complete it where it
  stops; replace it only where it cannot be completed.

## Decision

### Done is the forge's close, everywhere

- **No role writes Done.** `Closes #n` closes the issue at the merge;
  GitHub's *Item closed* and *Pull request merged* set Done, GitLab's
  Closed list shows it.

### On GitHub: the project's own workflows

- **workline writes no project.** In progress comes from *Pull request
  linked to issue*, Done from *Item closed*: one place for the rule, the
  project's, whoever opens the pull request — a person or the developer.
- **It says when they are off** (principle 14): at the claim, the
  developer reads the issue's projects (`Issue.projectItems`) and each
  one's `workflows`; one of the two off is a finding,
  `board-workflow-off` (info), naming the project and the workflow to
  turn on, once a run.
- **A project it cannot read is said**, `board-unread` (info): a user's
  project, read by the App, is never read; the summary says the board
  moves by its workflows alone. Never read as a pass nor a failure.
- **An issue in no project**: nothing said.

### On GitLab, and any board of labels: the project names its columns

- **Two settings of the developer**, empty by default:

| Setting | What it is | Example, Free | Example, Premium |
|---|---|---|---|
| `board-doing` | the label the claim puts on, beside `claim-label` | `Doing` | `workflow::doing` |
| `board-to-do` | the label the claim takes off | `To do` | (none: the scope takes it off) |

- **The claim's one `label` intention** adds `board-doing` and removes
  `board-to-do`: one act, on the forge's own words; `claim-label` stays
  the developer's mark, so a person dragging the card elsewhere never
  frees the issue for another pick.
- **A label the project lacks** is said, `board-label-missing` (warn),
  and nothing is created: a typo would make a column nobody sees.
- **Never `workline::` as a scope**, as [ADR-0040] says: on Premium it
  would knock `workline::proposed` off.

### Who moves what

| Column | Moved by | When |
|---|---|---|
| To do | the forge: GitHub's *Item added to project*, GitLab's Open list | the issue is opened or added |
| Ready (if the board has one) | the product owner's `workline:ready`, already a label: a GitLab list for it, nothing written | the issue is set ready ([ADR-0018]) |
| Doing | GitHub: *Pull request linked to issue*; GitLab: the developer's `board-doing` | the claim, its pull request in the same `apply` |
| Done | the forge's close | the merge |

- **Back to To do**: the developer never takes its claim off today; a
  person does, and drags the card. A release act, if one is built, puts
  `board-to-do` back and takes `board-doing` off.
- **A person's drag is theirs**: no role moves a card back.
- **With no agent**: nothing claimed, nothing moved (principle 5).
- **With no board**: no setting, no project; nothing changes.

### Not done

- **The App writing an organisation's project** (`Organization projects`
  read and write, `updateProjectV2ItemFieldValue`): only if a project
  needs Doing before the pull request opens, or a Ready column; asked of
  the person, below.
- **Adding issues to a project**: *Auto-add* is the forge's.

## For the person

- **Which board do you use?** A GitHub project (yours or an
  organisation's), a GitLab board on Free (plain labels) or Premium
  (`workflow::` scoped labels)? It sets which step comes first.
- **A Ready column on GitHub Projects** needs a token writing the
  project: an organisation's App permission, or none on a user's.
  Wanted?

## Build plan

One pull request a step, its cases first in
`tests/conformance/cases/developer/`, `pending.txt` listing what does
not pass.

1. **GitLab's labels at the claim**: the two settings, the claim's
   intention, the missing label said. Cases:
   `board-claim-puts-doing-label`, `board-claim-takes-to-do-off`,
   `board-unset-moves-nothing`, `board-no-agent-moves-nothing`,
   `board-label-missing-said`; on GitLab and a `cmd:` forge.
2. **GitHub's workflows said**: the forge reads an issue's project
   items and their workflows; the fake forge holds them. Cases:
   `board-workflow-off-said`, `board-project-unread-said`,
   `board-no-project-says-nothing`.
3. **Docs**: a page under `roles/developer/docs/` (the board, both
   forges), the settings in the role's README, a line in the install
   guide: turn on the two workflows, add a list for each label.
4. **Tried**: a GitLab Free board on the sandbox, and a GitHub project
   on the sandbox — whether *Pull request linked to issue* fires for a
   draft and for a pull request the App opens; told in the pull
   request.

[ADR-0018]: 0018-the-product-owner.md
[ADR-0023]: 0023-the-project-bot-keeps-a-gitlab-backlog.md
[ADR-0040]: 0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
