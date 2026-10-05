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
| GitLab | [Tasks](https://docs.gitlab.com/user/tasks/), every tier: **an issue's children are tasks only**, another work item type, made through the work items GraphQL API; issues as children of an epic (Premium). [Issue links](https://docs.gitlab.com/api/issue_links/) (`relates_to` on Free) are not a hierarchy | Tasks, but a task is not an issue: the line's acts and reading are on issues |
| Both | A **task list** in the body, `- [ ] #13`: both forges render the issue's title and state on the reference; GitHub's *tasklist blocks* that tracked them were retired for sub-issues | A body, everywhere, the local forge included |

**When to split**, in backlog practice: INVEST's *Small* and *Testable*
(Bill Wake, 2003) — an item a team can finish and prove in a short
iteration; Mike Cohn's [SPIDR](https://www.mountaingoatsoftware.com/blog/five-simple-but-powerful-ways-to-split-user-stories)
(spike, path, interface, data, rules) for where to cut; and, in workline's
words, one Verification an issue (docs/spec/routing.md): a need whose
proof is several checks that pass apart is several issues. The parent
keeps the need it was written for; the children are its parts.

**Titles**: Mozilla's [bug writing guidelines](https://bugzilla.mozilla.org/page.cgi?id=bug-writing.html)
— "approximately 10 words", "quickly and uniquely identify a bug report",
"explain the problem, not your suggested solution". No forge records whose
title an issue has but in its timeline (GitHub's `renamed` event).

**Decision** (ADR-0021): split through sub-issues where the forge has
them, a task list in the parent elsewhere; each child opened through the
one way, with its four sections; a rename sets the title alone, and a
title a person set after the role's is kept.

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
