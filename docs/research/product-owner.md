# Product owner — focused research (2026-10-03)

**Verdict.** The forges already hold everything a backlog needs — issue
forms, milestones, sub-issues, close reasons, closing keywords, labels — and
bots already keep one report issue in place (Renovate's dashboard) and cap
what an agent may write (GitHub's agentic workflows, *safe outputs*). What
nobody does is ask whether an issue is still true **because the code it is
about changed**: every "obsolete" signal in use is inactivity, which the
literature and the communities reject, or the reporter's silence taken for
consent. That is the documentalist's `sources`/`checked` mechanism applied to
issues, and the place a product owner role adds something: it proposes, with the
code quoted; a person orders, accepts, groups and closes, as the Scrum Guide
keeps the Product Owner accountable even when the work is delegated.

Stars and last push from the GitHub API that day; behaviour from each
tool's docs, linked. "Unverified" marks what was only read in a summary.

## What the forges and bots already do

| Mechanism | What it does | Left to a person |
|---|---|---|
| GitHub [issue forms](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms); GitLab [description templates](https://docs.gitlab.com/user/project/description_templates/) | Fields with predictable headings; `required` only in the UI, only on public repositories; none on GitLab | Whether a field says anything |
| [Milestones](https://docs.github.com/en/rest/issues/issues#update-an-issue), GitLab [iterations](https://docs.gitlab.com/user/group/iterations/) and epics | Containers with progress; GitLab moves open issues to the next iteration | What goes in which |
| [Sub-issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/adding-sub-issues) | Parent and children, across repositories | The breakdown |
| [Close reasons](https://docs.github.com/en/rest/issues/issues#update-an-issue) | `completed`, `not_planned`, `duplicate` with `duplicate_issue_id` | The reason |
| [Closing keywords](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/linking-a-pull-request-to-an-issue) | `Fixes #n` closes on merge — into the default branch only, on both forges | Writing it |
| [Code scanning](https://docs.github.com/en/code-security/concepts/code-scanning/alert-tracking-with-issues) | An alert closes when a later analysis no longer finds it (SARIF fingerprints); "alert and issue statuses are not automatically synchronized" | The linked issue |
| [Renovate Dependency Dashboard](https://docs.renovatebot.com/key-concepts/dashboard/) (22.7k★) | One issue edited in place; a ticked checkbox is the approval the next run applies | Ticking |
| [gitlab-triage](https://gitlab.com/gitlab-org/ruby/gems/gitlab-triage) (224★), GitLab's [triage-ops](https://gitlab.com/gitlab-org/quality/triage-ops) | YAML policies: conditions (date, labels, milestone…), actions, `limits`, `summarize` into one report issue; triage-ops already calls an AI for one policy | The policies |
| [actions/stale](https://github.com/actions/stale) (1.7k★) | Stale label after 60 days, closed 7 days later | Reopening |
| [@semantic-release/github](https://github.com/semantic-release/github) (536★); Sentry, Linear | "Resolved in version X" on the issues a release shipped; Sentry reopens a regression | — |

## What AI triage tools decide alone

| Tool | Alone | Proposed | Evidence |
|---|---|---|---|
| [claude-code's own workflows](https://github.com/anthropics/claude-code/blob/main/.github/workflows/claude-dedupe-issues.yml) | Labels, one duplicates comment, **closes as duplicate after 3 days** unless the author objects (a script, not the model) | Up to 3 duplicates | Links only |
| [gh-aw](https://github.com/github/gh-aw) (5.3k★), [githubnext/agentics](https://github.com/githubnext/agentics) | Read-only agent; each write a declared *safe output* with a cap per run | A triage report | The issue's text |
| [elastic/ai-github-actions](https://github.com/elastic/ai-github-actions/tree/main/gh-agent-workflows/stale-issues-investigator) (11★) | Labels issues it finds **already resolved**; another job closes after 30 days unless someone objects | One report, at most 10 candidates, "when in doubt, skip" | **Reads the code**, may run tests; a `fixes #N` merged elsewhere, code, or the thread |
| [Linear Triage Intelligence](https://linear.app/docs/triage-intelligence) | Per property and per value: apply, suggest or hide | Duplicates, always | On demand |
| [Atlassian Rovo](https://support.atlassian.com/rovo/docs/atlassian-agents/) | Organizer moves and may delete items (unverified) | **Readiness Checker** scores an item against the team's definition of ready | A score, the gaps |
| [Dosu](https://dosu.dev/blog/automating-github-issue-triage), [CodeRabbit](https://docs.coderabbit.ai/issues/enrichment), [GitHub Models labelers](https://github.com/actions/ai-inference) | Labels | Drafted replies, similar issues | The thread, not the code |

Their faults: duplicates closed wrongly (claude-code #14325), bare links
without a reason, similarity taken for duplication, silence taken for
consent, each tool tied to one tracker; Sweep pivoted, VS Code's triage
actions are archived.

## What projects do

- **Kubernetes** ([triage](https://github.com/kubernetes/community/blob/master/contributors/guide/issue-triage.md)): `needs-triage` on every new issue until a person sets `triage/accepted`, `needs-information`, `not-reproducible` or `duplicate`; a robot stales, rots and closes after 90+30+30 days, `/lifecycle frozen` to opt out.
- **Rust** ([triagebot](https://github.com/rust-lang/triagebot), 232★): anyone may set the labels the config allows, decision labels stay with the team; *nominations* put an issue on a team's agenda; no stale bot.
- **CPython**: a triage team closes as duplicate or not planned; its stale workflow never marks an issue. Its move from bugs.python.org kept each old id in a table at the top of the issue.
- **VS Code** ([wiki](https://github.com/microsoft/vscode/wiki/Issues-Triaging)): triaged means it has a milestone; closed bugs are *verified*, and the reporter is asked once the build with the fix ships.
- **Godot** ([guidelines](https://docs.godotengine.org/en/4.4/contributing/workflow/bug_triage_guidelines.html)): `confirmed` by someone other than the reporter, "may not be relevant anymore" a year later; a milestone is "a goal, not a guarantee".
- **Home Assistant**, **Node.js**: actions/stale; Home Assistant asks to retest on the latest version, then closes on silence.

The **Scrum Guide 2020** gives the Product Owner the backlog — creating,
ordering, keeping it understood — and keeps them accountable when they
delegate. It has no "Definition of Ready"; its critics
([Cohn](https://mountaingoatsoftware.com/blog/the-dangers-of-a-definition-of-ready))
warn that one used as a gate becomes a stage gate. Closing by inactivity:
[DeVault](https://drewdevault.com/2021/10/26/stalebot.html) against it;
[Khatoonabadi et al. 2023](https://arxiv.org/abs/2305.18150) measured fewer
active contributors after a stale bot.

## Patterns worth borrowing

1. **Write through the forge's own fields**: milestones for lots, close
   reasons, sub-issues, labels; GitLab scoped labels where the tier allows.
2. **One report issue kept in place**, checkboxes as the person's yes
   (Renovate, `summarize`), rather than a comment on every issue.
3. **Typed writes with a cap per run** (*safe outputs*): workline's
   propose and apply, in the ecosystem's words.
4. **Evidence kinds and "when in doubt, skip"** (Elastic): a merged
   `fixes #N` the forge did not act on, code quoted, the thread.
5. **"What is missing" labels** over a generic stale: `needs-repro`,
   `needs-info` (Kubernetes, Rust).
6. **Readiness as a score or a hint** (Rovo), never a gate (Cohn).
7. **Proposing apart from deciding**: Rust's open labels and protected
   decision labels; Kubernetes' `needs-triage` until a person accepts.
8. **Migrating a backlog**: one cut-over, the old id kept in each issue,
   the old file kept read-only (CPython, Apache Lucene).

## Priority and ranking (2026-10-04)

Where projects keep an issue's place in the backlog, in their words:
*priority*, *rank*, *triage*.

| Where | How | What a role can write |
|---|---|---|
| Kubernetes | [`priority/*` labels](https://github.com/kubernetes/community/blob/master/contributors/guide/issue-triage.md): `critical-urgent`, `important-soon`, `important-longterm`, `backlog` | A label, by any triager |
| Rust | [`P-critical`, `P-high`, `P-medium`, `P-low`](https://forge.rust-lang.org/compiler/prioritization.html) labels, set at a weekly triage | A label |
| Linear | [Four levels](https://linear.app/docs/priority) — Urgent, High, Medium, Low — and "No priority" | A field |
| GitLab | [Scoped labels](https://docs.gitlab.com/user/project/labels/#scoped-labels) (`priority::1`, one of a scope at a time; Premium) and an [issue reorder API](https://docs.gitlab.com/api/issues/#reorder-an-issue) for a board's rank | A label; a rank with the API |
| GitHub | Issue fields with a Priority — for organisations only; a [Projects v2](https://docs.github.com/en/issues/planning-and-tracking-with-projects) item's position, with a token scoped to the project | Neither, on a user account (JN0V's) without a project token |
| GitLab, iterations | [Open issues roll over](https://docs.gitlab.com/user/group/iterations/) to the next iteration when one ends | — the engine's slip, borrowed |

**Decision** (ADR-0018, "Ordering"): priority as labels, four levels
(`workline:priority/1` to `/4`, 1 the most pressing), the way Kubernetes,
Rust and Linear count; one at a time, as GitLab's scoped labels; the order
itself derived — nearest milestone, priority, number — not stored. A
forge's native rank (GitLab's reorder, a GitHub project's position) is
deferred: it is not on every forge workline speaks, nor writable with the
token a role holds.

## A subject found again (2026-10-04)

How tools that open an item for what they find keep from opening it twice,
and what they do when the item was closed — in their words: *fingerprint*,
*grouping*, *regression*, *ignore*.

| Tool | Same subject | Closed, found again |
|---|---|---|
| [Sentry](https://docs.sentry.io/concepts/data-management/event-grouping/) | Events with the same *fingerprint* are one issue; a fingerprint can be set by rules | Resolved: reopened as *regressed*, and alerted. Archived: silent, forever or until it escalates |
| [SonarQube](https://docs.sonarsource.com/sonarqube-server/10.5/user-guide/issues) | An issue is tracked across analyses by its rule and location | Fixed: reopened when an analysis finds it again. *Accepted* or *false positive*, set by a person: kept, never reopened |
| [GitHub code scanning](https://docs.github.com/en/code-security/how-tos/manage-security-alerts/manage-code-scanning-alerts/resolving-code-scanning-alerts) | SARIF `partialFingerprints` follow an alert across commits (server-side.md) | Dismissed: dismissed on every branch, a person may reopen it |
| [Renovate](https://docs.renovatebot.com/key-concepts/pull-requests/) | One pull request per update; the dashboard issue kept in place | Closed unmerged: that update is ignored, never recreated; a newer version opens a new one ("immortal" pull requests, for groups, are the exception it warns about) |

Two shared rules: **the key is the tool's, computed, not a person's or a
model's guess**; and **a person's "no" is never undone by the tool**, while
"fixed" seen again is a signal. Where the tools reopen a fixed item, a
workline role cannot: no role but the product owner closes or reopens.

**Decision** (ADR-0018, "Opening issues, for every role"): the key is the
engine's, from the line of code a finding quotes; issues open and closed
are looked in; closed as not planned or duplicate, nothing; closed
otherwise, said once on the issue, left closed; the line changed, a new
subject.

## Splitting and renaming (2026-10-05)

How forges hold a need broken into parts, and how backlog practice decides
to break one or to retitle it — in their words: *sub-issues*, *child
items*, *task list*, *story splitting*, *summary*.

| Where | Parent and children | What a role can write |
|---|---|---|
| GitHub | [Sub-issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/adding-sub-issues): up to 100 a parent, 8 levels, across repositories; one parent a child. [REST](https://docs.github.com/en/rest/issues/sub-issues): `POST /repos/{o}/{r}/issues/{n}/sub_issues` with `sub_issue_id` — the issue's **id**, not its number —, `GET …/sub_issues`, `GET …/parent`, `replace_parent` to move a child | Both, with the issues token a role already holds |
| GitLab | [Tasks](https://docs.gitlab.com/user/tasks/), every tier: **an issue's children are tasks only** — asked of gitlab.com's GraphQL for JN0V/workline-sandbox (a user namespace, Free), the type Issue allows the children `[Task]` and the parents `[]`. The hierarchy is in the work items **GraphQL API only** (`workItemConvert`, `workItemUpdate` with `hierarchyWidget.parentId`); REST's `issue_type=task` on an existing issue is refused. But the REST issues API **lists a task** (`issue_type: task`), and comments, labels and renames it as an issue: tried by hand, 2026-10-05. [Issue links](https://docs.gitlab.com/api/issue_links/) (`relates_to` on Free) are not a hierarchy. [Epics](https://docs.gitlab.com/user/group/epics/) are a group's, Premium and up, absent from a user namespace on Free: a portfolio of needs across projects, a roadmap's theme — closer to a milestone than to the parts of one need | A task: opened as an issue, converted, given its parent, two GraphQL calls; then read and acted on by REST as any issue |
| Both | A **task list** in the body, `- [ ] #13`: both forges render the issue's title and state on the reference; GitHub's *tasklist blocks* that tracked them were retired for sub-issues | A body, everywhere, the local forge included |

### When to split, and titles

**When to split**, in backlog practice: INVEST's *Small* and *Testable*
(Bill Wake, 2003) — an item a team can finish and prove in a short
iteration; Mike Cohn's [SPIDR](https://www.mountaingoatsoftware.com/blog/five-simple-but-powerful-ways-to-split-user-stories)
(spike, path, interface, data, rules) for where to cut; and, in workline's
words, one Verification an issue (docs/spec/routing.md): a need whose
proof is several checks that pass apart is several issues. The parent
keeps the need it was written for; the children are its parts.

**Titles**: Mozilla's [bug writing guidelines](https://bugzilla.mozilla.org/page.cgi?id=bug-writing.html)
— "approximately 10 words", "quickly and uniquely identify a bug report",
"explain the problem, not your suggested solution". Who set a title is kept
only in a forge's history (GitHub's `renamed` timeline event, a GitLab system note).

### Decision

(ADR-0022): split through the forge's own children —
GitHub's sub-issues, GitLab's tasks — and a task list in the parent where
there are none (the local forge, an instance without work items); each
child opened through the one way, with its four sections. Epics are not
used to split: a need's parts are not a portfolio, and they are not on
every tier; grouping themes across projects, perhaps later, the product
manager's. A rename sets the title alone, and a title a person set after
the role's is kept.

## What an issue waits on (2026-10-05)

How forges and trackers hold "this cannot start before that is done", and
how they order and show it — in their words: *issue dependencies*,
*blocked by* / *blocking*, *linked items*, *issue links*, *topological
order*.

| Where | The relation | What a role can write, and read |
|---|---|---|
| GitHub | [Issue dependencies](https://github.blog/changelog/2025-08-21-dependencies-on-issues/), generally available since 2025-08: *blocked by* and *blocking*, up to 50 of each an issue; shown on the issue and in lists and projects (a "Blocked" mark), searched with `is:blocked`, `blocked-by:`. [REST](https://docs.github.com/en/rest/issues/issue-dependencies): `GET`/`POST /repos/{o}/{r}/issues/{n}/dependencies/blocked_by`, the blocker by its **id** (`issue_id`), not its number; `DELETE …/blocked_by/{issue_id}`. Every issue of the REST listing carries `issue_dependencies_summary` (`blocked_by` open, `total_blocked_by`): asked of JN0V/workline and JN0V/workline-sandbox (user repositories), 2026-10-05 | Both, with the issues token a role holds; one listing tells which issues to ask further |
| GitLab | [Linked items](https://docs.gitlab.com/user/project/issues/related_issues/): `relates to` on every tier; `blocks` / `is blocked by` **Premium and Ultimate only**, an icon beside a blocked issue's title in lists and boards, gone when its blocker closes; closing a blocked issue asks to confirm. [Issue links API](https://docs.gitlab.com/api/issue_links/) `POST /projects/:id/issues/:iid/links` with `link_type`. Asked of gitlab.com, JN0V/workline-sandbox (Free), 2026-10-05: `is_blocked_by` answers **403 "Blocked issues not available for current license"**; GraphQL's `blockedByIssues` answers, empty, on Free | Premium: the link, and every open issue's blockers in one GraphQL query. Free: nothing native but `relates to`, which says no direction |
| Jira | [Issue linking](https://confluence.atlassian.com/adminjiraserver/configuring-issue-linking-938847862.html): `blocks` / `is blocked by`, one of four default link types. Jira [does not enforce it](https://community.atlassian.com/forums/Jira-questions/Automatically-add-blocks-and-is-blocked-by-based-on-rank/qaq-p/1000810): a blocked issue can be ranked first or put in a sprint; teams write automation from the rank, and Advanced Roadmaps schedules from the links | — |
| Text | "Blocked by #12" written in a body or a comment, by hand: both GitHub and GitLab render the reference with its title and state; nothing orders by it | A body, everywhere |

**Ordering with dependencies** is a topological sort; Kahn's algorithm
takes the next item whose prerequisites are placed, and a tie between
those ready is broken by any other order — here, the backlog's. A cycle
is a person's mistake to report, not to follow: GNU make says "Circular
X <- Y dependency dropped." and goes on. None of the trackers above keeps
a blocked issue out of first place by itself; the forges only show it.

### Decision

ADR-0028: the forge's own relation where it has one (GitHub's
dependencies, GitLab's `is_blocked_by` on Premium), a marked line in the
body elsewhere (`Blocked by #12.`), read back the same way, as a person's
own "Blocked by" line; the backlog's order puts a blocked issue after its
open blockers, reports a cycle and never follows it.

## A parent and its parts (2026-10-05)

What happens to a need split into parts once they close — in the
ecosystem's words: *sub-issues*, *child items*, *tasks*, *closing
keywords*, *closed by*, *epic done*.

| Where | What a parent shows | What closed a part |
|---|---|---|
| GitHub | A progress bar from its sub-issues; the REST listing carries `sub_issues_summary` (`total`, `completed`), so one listing tells which issues are parents; `GET …/issues/{n}/sub_issues` lists them, open or closed, a sub-issue of another repository included (`repository_url`). Never closed by GitHub when its sub-issues are | GraphQL's `ClosedEvent.closer`, a `PullRequest` (merged with "Closes #n") or a `Commit` (pushed with one), and `closedByPullRequestsReferences`. Asked of JN0V/workline #161 (closed by pull request #170) and JN0V/workline-sandbox #10, 2026-10-05 |
| GitLab | Tasks under an issue, a progress count; the work items' GraphQL `workItems { widgets { … on WorkItemWidgetHierarchy { children } } }` lists each open item's children, one query a page — asked of gitlab.com JN0V/workline-sandbox (Free), 2026-10-05 | `GET /projects/:id/issues/:iid/resource_state_events`: each closing with its `source_commit` or `source_merge_request_id` (a global id, found among `closed_by`'s merge requests); no note is written for either. Asked of JN0V/workline-sandbox #26 (a commit) and #27 (merge request !4), 2026-10-05 |
| Jira | An epic's children and a progress bar; its automation's common rule moves the parent to Done once every child is | — |

Nobody checks the parent's own acceptance criteria against what its
parts delivered: the forges count, Jira's rule closes on the count — the
part of a need lost between children is what principle 1 leaves to a
person.

### Decision

ADR-0029: the engine writes on the parent, with no agent, what each part
became and what closed it, and which items of its Verification a part
delivered quotes; a person accepts it by closing it; the role never does.

## Asking, and the answer (2026-10-05)

How bots that ask a reporter for something carry the conversation on — in
their words: *needs more info*, *response required*, *waiting on author*,
*pending*, *bump*.

| Tool | Asking | The answer | Rounds, then |
|---|---|---|---|
| [no-response](https://github.com/lee-dohm/no-response) | A person sets `responseRequiredLabel` | Only a comment **by the issue's author** counts: the label goes, an issue it closed is reopened | One; `daysUntilClose` of silence, closed |
| [label-actions](https://github.com/marketplace/actions/label-actions) | A label (`needs more info`) posts a canned comment naming `{issue-author}` | Not read: a person takes the label off | None |
| Kubernetes `triage/needs-information` | A triager's label | A person reads it | The lifecycle robot on silence (above) |
| Rust's triagebot `S-waiting-on-author` | Set on a review asking changes | The author says `@rustbot ready`: the labels swap back | As many as the review needs |
| [Zendesk *pending*](https://support.zendesk.com/hc/en-us/articles/4408832749210) | The agent's reply sets the ticket waiting on the customer | Any customer reply sets it back to open | *Bump, bump, solve*: two reminders, then solved |
| [Dependabot](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-pull-request-comment-commands), [Copilot coding agent](https://docs.github.com/en/copilot/responsible-use/copilot-coding-agent) | — | Comment commands obeyed **only from people with write access** | — |

Three shared rules. **An answer is a comment after the question**, and it
moves the item back to whoever acts next. **Who wrote it matters** when the
answer is a decision: no-response counts only the author's, Dependabot
and Copilot only a writer's. **Rounds are few** — one, or two reminders —
and none of these closes on silence without a person having set it up.
None asks a *narrower* question after an answer: the follow-up is a
person's, or a canned text.

On GitHub, [an issue's body can be edited only by its author, or by
someone with write access](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/editing-an-issue);
a label only by who may triage. Either is a change only the reporter or
the project's people make — a comment is anyone's.

**Decision** (ADR-0021): the product owner asks again only after an
answer, at most three times, then the report; an outsider's issue gets
the refined text proposed in a comment, applied when an insider sets
`workline:accepted`, or written by the reporter into their own body.

## Closing on silence, announced first (2026-10-05)

How bots close what they find dead, in their words: *stale*, *rotten*,
*lifecycle*, *days-before-close*, *exempt*, *remove-stale-when-updated*,
*objection*, *close reason*.

| Tool | Announced by | Waits | Cancelled by | Closed as |
|---|---|---|---|---|
| [actions/stale](https://github.com/actions/stale) | the `Stale` label and a comment, after `days-before-stale` (60) of inactivity | `days-before-close` (7) | any update or comment (`remove-stale-when-updated`, on by default); `exempt-issue-labels`; `operations-per-run` (30) caps the API calls | `close-issue-reason`, `not_planned` by default, with `close-issue-message` |
| Kubernetes' [triage robot](https://github.com/kubernetes/community/blob/master/contributors/guide/issue-triage.md) | `lifecycle/stale` after 90 days, `lifecycle/rotten` 30 days later | 30 more days | any activity resets the clock; `/remove-lifecycle stale`; `lifecycle/frozen` exempts | closed, `/reopen` to undo |
| GitLab's [triage-ops](https://gitlab.com/gitlab-org/quality/triage-ops) | a policy's label and comment | the policy's dates | the policy's conditions | closed: GitLab keeps no reason |
| Elastic's [stale-issues investigator](https://github.com/elastic/ai-github-actions/tree/main/gh-agent-workflows/stale-issues-investigator) and [remediator](https://github.com/elastic/ai-github-actions/tree/main/gh-agent-workflows/stale-issues-remediator) | an agent finds it **resolved** (a linked pull request, the code, the thread), labels it `stale`, one report | 30 days | an *objection* — a comment saying it is still relevant, read by an agent — takes the label off | closed with a comment |

Shared: **an announcement a person sees** (a label, a comment) before any
closing; **a delay in days**, not in runs; **any activity or the label
taken off cancels**; **exempt labels**; a cap per run. GitHub's close
reasons: `completed` (work was done), `not_planned` (none was: won't fix,
stale, can't reproduce), `duplicate`; stale bots use `not_planned`,
because nobody did anything. Only Elastic's closes because the work was
*done*, and it asks an agent, not a check, whether a comment objects.
None asks a second, independent judge before closing.

**Decision** (ADR-0024): an issue the code solved is announced — a comment
quoting the code, the label `workline:obsolete` — and closed after `days`
(7) if nobody wrote and the label is still there, once a second judge of
another model agrees; any person's comment or the label taken off cancels,
for good for that evidence. Closed as `completed`, not `not_planned`: the
code did the work, and `not_planned` is the person's no that every role's
issue opening reads as such (ADR-0018).

## A tick and who ticked it; a bot nobody answers (2026-10-05)

In the ecosystem's words: *dependency dashboard*, *checkbox*, *task list*,
*approval*, *edit history*, *auto-pause*, *inactive repository*.

| Where | A tick read | Who ticked it | Nobody answering |
|---|---|---|---|
| [Renovate's dashboard](https://docs.renovatebot.com/key-concepts/dashboard/) | at its next run, the box applied (`dependencyDashboardApproval`) | not checked: its docs say nothing of who may tick; the forge's edit rights are the only guard | — |
| GitHub | no event for a task box; each version of a body kept with its editor (GraphQL `userContentEdits`, the whole body each, newest first — read live on JN0V/workline-sandbox) | the editor; only the author and [write access and above](https://docs.github.com/en/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/repository-roles-for-an-organization) edit another's issue; `collaborators/<login>/permission` says which | — |
| GitLab | a system note per box, "marked the checklist item **…** as completed" (or "incomplete"), written for a tick on the page and for a description changed through the API alike, the item's markdown escaped (`\#`, `\=`) and a hidden comment's text kept without its `<!--`/`-->` — read live on gitlab.com; a box added already ticked writes none | the note's author | — |
| [Dependabot](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/troubleshoot-dependency-security/dependabot-updates-stopped) | — | — | pauses a repository where a pull request of its stayed open 90 days and no person merged or closed one, changed its config or ran it; no pull request opened while paused; any of those resumes it |

**Decision** (ADR-0025): a box in the report carries a hidden key; a
person of the project's tick — by the forge's own record of who ticked it
— is done at the next run, as the record keeps the act; a bot's, an
outsider's or an unknown author's is said and not done, stricter than
Renovate since a run's token may tick what it wrote. Three runs nobody
answered pause the agent, as Dependabot, resumed by a person's act.

## An import's coverage (2026-10-05)

In the ecosystem's words: *requirements traceability*, *traceability
matrix*, *coverage*, *uncovered requirement*, *import report*, *skipped
rows*.

| Where | What it maps | What is left out, and how it says so |
|---|---|---|
| A requirements traceability matrix (systems engineering's practice; ISO/IEC/IEEE 29148 asks requirements be traceable) | each requirement, by its id, to what designs, implements and tests it | a row with an empty cell: read by a person |
| [OpenFastTrace](https://github.com/itsallcode/openfasttrace) | specification items, tagged in any text file, to the items that cover them | each item not covered listed as a defect; the trace ends "not ok", a non-zero exit |
| [Doorstop](https://github.com/doorstop-dev/doorstop) | requirements as files under git, each linked to its parent | `doorstop` validates the tree and warns on an item no child links to |
| [sphinx-needs](https://github.com/useblocks/sphinx-needs) | needs in the docs, linked by id | a table filtered on needs with no link: a person writes the filter |
| Trackers' CSV importers (Jira's, GitLab's) | one row a new issue | a row refused is in the import's log; a row the importer never read as one is not anywhere |

All map **declared** items — an id, a tag, a row. A roadmap in prose has
none: which lines are an item is the agent's judgement, so the
partition comes from it, and the engine checks only what it can — every
line answered, each reason's evidence there.

**Decision** (ADR-0030): as a traceability matrix, every item of the
file to the issue that holds it, or to a reason checked by the engine;
as OpenFastTrace, what is left with neither listed, and the run not
passing (`human`, exit 2) — never a silent pass.

## Gaps no tool covers

1. **An issue re-checked because the code it names changed** since it was
   written or last confirmed — commit-scoped, not age-scoped. Only
   Elastic's investigator reads code, by age, GitHub only, closing on
   silence.
2. **Refining to ready with code quoted**: Need, Verification, Validation,
   Scope proposed from the code, not only scored.
3. **Lots proposed** for a release, for a person to take or leave.
4. **One contract across forges** (GitHub, GitLab, local).
5. **No closing on silence**: a proposal stays one until a person says yes.

## Questions for the decision record

- Does the assistant close anything itself — the duplicate a person
  confirmed, the issue whose `Fixes #n` was merged elsewhere — or only
  label and report?
- Where does an issue keep its `sources` and the commit it was confirmed
  at: a block in its body (as the documentalist's header), a hidden
  marker, a label?
- One report issue (Renovate) or a comment on each issue, or both?
- Which writes need an explicit yes (close, milestone), which may it do
  (labels such as `needs-triage`, `maybe-obsolete`)?
- Which roles open issues through the shared mechanism, and what it
  guarantees them (no duplicate, `needs-triage`, sources, the cap).
- The first try: DomoticsCore's CODE-ROADMAP.md (7,500 lines, ids such as
  BUG-15, lots of one pull request each) moved to issues, milestones for
  lots.
