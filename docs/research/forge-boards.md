# Forge boards — focused research (2026-10-30)

**Question.** A person follows the work on a Kanban board (To do, Doing,
Done). workline marks an issue with its own labels (`workline:ready`,
`workline:developing`) and the developer "never moves an issue"
([ADR-0054](../adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md)),
so the board never advances. What do the forges move on their own, what
do coding agents and trackers do when work starts, and what is left for
a role to write?

**Verdict.**

- **Done is the forge's on both**: GitHub Projects sets Status to Done
  when an issue closes or a pull request merges, on by default; a GitLab
  board shows a closed issue in its Closed list. `Closes #n` is enough.
- **Doing is the forge's on GitHub Projects too**: since 2025-11 a new
  project's default workflow *Pull request linked to issue* sets the
  issue's Status to In progress. The developer opens its pull request,
  with `Closes #n`, in the same `apply` as its claim.
- **GitLab moves nothing on its own** until the close: a column is a
  label, and a role that starts work must put that label on.
- **A GitHub App cannot write a user-owned project**: the Projects
  permission is an organisation's. A user's project moves only by its
  built-in workflows, or by a person's token.
- **Every tool that starts work says it on the issue**: Copilot reacts
  👀 and links a draft pull request; Linear moves the issue to In
  Progress when its pull request opens, Done when it merges; Jira
  transitions on a branch or commit naming the issue.

## Method

- In the ecosystem's words: *Status field*, *built-in workflows*,
  *auto-add*, *linked pull request*, *Development sidebar*, *issue
  board*, *board lists*, *scoped labels*, *workflow labels*, *start
  progress*, *transition*.
- Read GitHub's Projects and Apps docs, its changelog, GitHub's GraphQL
  schema by introspection (2026-10-30), GitLab's board and label docs,
  Linear's and Atlassian's integration docs.
- Not asked: JN0V's own projects. The session's token lacks
  `read:project` (`gh project list --owner JN0V` refused it).

## GitHub Projects

| Mechanism | What it does | Source |
|---|---|---|
| Status field | A single-select field, options Todo, In progress, Done in a new project; a board's columns are its options | [About Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/learning-about-projects/about-projects) |
| *Item closed* | Status → Done; on by default | [built-in automations](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-built-in-automations) |
| *Pull request merged* | Status → Done; on by default | same |
| *Item added to project* | Status → Todo; off by default | same |
| *Pull request linked to issue* | "automatically sets the status of an issue to 'In progress'"; default in new projects since 2025-11-06 | [changelog](https://github.blog/changelog/2025-11-06-improved-onboarding-flow-for-github-projects/) |
| Auto-close on status | Closes the issue when its Status changes to a chosen value; off by default | built-in automations |
| *Auto-add to project* | Adds new or updated items matching a filter; "existing items matching your criteria will not be added"; 1 such workflow on Free, 5 on Pro and Team, 20 on Enterprise | [adding items automatically](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/adding-items-automatically) |

### Linking a pull request

- **Closing keywords** in the pull request's description link it, but
  only when it "targets the repository's default branch"; a keyword in a
  commit links nothing.
- **The Development sidebar** links by hand, up to ten issues a pull
  request.
- Source: [linking a pull request](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/linking-a-pull-request-to-an-issue).
- **For workline**: the developer's pull request says `Closes #n` and
  targets `main`, so it links; the tester's Draft says `Refs #n`, so it
  does not move the board.

### Writing a project from a role

- **GraphQL only**: `updateProjectV2ItemFieldValue(input: {projectId,
  itemId, fieldId, value: {singleSelectOptionId}})`; an item found by
  `Issue.projectItems`. Each `ProjectV2` lists its `workflows`, each
  with `name` and `enabled`, readable by the same token (schema
  introspected 2026-10-30).
- **Tokens**, from
  [automating Projects using Actions](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/automating-projects-using-actions):
  - `GITHUB_TOKEN` "is scoped to the repository level and cannot access
    projects";
  - a GitHub App, for an organisation's project: "permission to read
    and write organization projects"; "repository projects will not be
    sufficient";
  - a personal access token, "recommended for user projects" (classic,
    `project` scope).
- **A user's project**: GitHub's
  [App permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)
  list "Projects" under the organisation's permissions only. An
  installation token on a user account reaches no project; only a
  user's token does
  ([a project hitting it](https://github.com/JSONbored/loopover/issues/3184)).
  workline's App is installed on a user account (JN0V).

## GitLab issue boards

| Mechanism | What it does | Tier | Source |
|---|---|---|---|
| Label list | "all open issues for a label"; a column is a label | Free | [issue boards](https://docs.gitlab.com/user/project/issue_board/) |
| Open, Closed lists | Open: open issues in no other list, leftmost; Closed: all closed issues, rightmost | Free | same |
| Moving a card | Removes the label of the list it left, adds the one of the list it reached; to Closed closes the issue | Free | same |
| Several boards | A project's: every tier; a group's: one on Free | Free | same |
| Assignee, milestone, iteration lists | A column per person, milestone, iteration | Premium | same |
| Scoped labels | `key::value`, one per key: adding `workflow::review` removes `workflow::development` | Premium | [labels](https://docs.gitlab.com/user/project/labels/) |

- **Moving by API is relabelling**: `PUT /projects/:id/issues/:iid` with
  `add_labels` and `remove_labels`
  ([issues API](https://docs.gitlab.com/api/issues/#edit-an-issue)); the
  [boards API](https://docs.gitlab.com/api/boards/) lists a board's lists
  and their labels.
- **Free**: plain labels. Putting Doing on leaves To do there: the card
  sits in both lists until a role takes To do off.
- **Premium**: with `workflow::todo` and `workflow::doing`, putting the
  second takes the first off, by the forge.
- **The merge request moves nothing** until it closes the issue; no
  built-in "linked merge request → Doing" was found.

## What starts work elsewhere

| Tool | When work starts | When it ends | Source |
|---|---|---|---|
| Copilot coding agent | Assigned the issue; reacts 👀, opens a draft pull request linked to it, an event in the issue's timeline | Asks the person for a review | [using Copilot on an issue](https://docs.github.com/en/copilot/using-github-copilot/coding-agent/using-copilot-to-work-on-an-issue) |
| Linear's GitHub integration | "move linked issues to 'In Progress' when PRs are open"; copying the branch name can move it to a started status, and assign on start | Merged → Done; draft, review and ready-to-merge states configurable | [Linear GitHub docs](https://linear.app/docs/github) |
| Jira's development integration | A branch or commit naming the key triggers Start progress, To Do → In Progress (branch created not supported for GitHub, DCON-432) | Pull request merged as a trigger | [workflow triggers](https://confluence.atlassian.com/display/JIRA/Configuring+workflow+triggers) |

- **None of them writes a board's column from the agent**: the tracker
  reads the pull request and moves its own issue.
- **The issue is linked on start**, so the person sees who works on it
  where they look.

## What is left for workline

- **GitHub, any project**: nothing to write for Doing nor Done if
  *Pull request linked to issue* and *Item closed* are on. What is left:
  say when a project the issue sits in has them off, where the token can
  read the project.
- **GitHub, a claim with no pull request yet**: none today — claim and
  pull request land in one `apply`.
- **Back to To do**: no forge moves an issue back when its pull request
  closes unmerged (none found). The developer never takes its claim off
  today; a person does, and drags the card.
- **GitLab**: the Doing label at the claim, the To do label off on Free.
  The close does the rest.
- **To try**: whether *Pull request linked to issue* fires for a draft
  pull request, for one an App opens, and in a project made before
  2025-11.
