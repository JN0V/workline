# ADR-0038: The product owner proposes on the issue; a person answers there, with forge labels

- **Status:** accepted — decided, not built yet; a week's trial first
- **Date:** 2026-10-08
- **Supersedes:** the report issue in
  [ADR-0018](0018-the-product-owner.md) ("one report issue") and
  [ADR-0025](0025-a-persons-tick-is-done-ignored-runs-pause.md) (ticks,
  back-to-act boxes, the pause); the level table of
  [ADR-0026](0026-the-product-owners-autonomy-is-a-level.md); where
  [ADR-0031](0031-the-report-opens-with-what-is-next-and-what-is-stuck.md)
  is shown
- **Builds on:** [ADR-0021](0021-the-product-owner-talks-with-the-reporter.md)
  (rounds with the reporter), [ADR-0023](0023-the-project-bot-keeps-a-gitlab-backlog.md)
  (who is of the project on GitLab),
  [ADR-0035](0035-the-engine-writes-the-jobs-summary.md) (the job's
  summary); principles 1, 4, 13, 14
- **Research:** [surfacing-proposals.md](../research/surfacing-proposals.md)

## Context

- **The report issue was unreadable.** Its reader, the maintainer, could
  not tell what [#110](https://github.com/JN0V/workline/issues/110) was
  nor what to look at: a call to act, a next list, acts done, undo
  instructions, the autonomy table and how it works, on one page. On each
  issue, the role's only comment was a YAML block.
- **The role grew ahead of use.** Eleven decisions, about 7,000 lines of
  Go, about 150 conformance cases, "planted" 39 times in its status and
  trials; real use: one import and workline's own nights. Every objection
  became a mechanism: caps per kind, a moved share, demotion, a pause, a
  second judge, delays.
- **Twelve kinds × three modes × caps × three levels**: nobody could
  predict a run, and the levels did not say the maintainer's intent: how
  readily an evident issue reaches `ready`.
- **The ecosystem's answer** ([research](../research/surfacing-proposals.md)):
  the control on the item, the view as a filter; a dashboard with
  controls is Renovate's alone, and its maintainers want it split
  ([renovate#10924](https://github.com/renovatebot/renovate/issues/10924)).

## Decision

### Refocus

| On by default | Off by default (the code stays) |
|---|---|
| refine (drafts) | milestone, order |
| ask the reporter | split, depend, rename |
| ready | close-obsolete |
| an evident duplicate | changed needs ([ADR-0032](0032-a-changed-need-flags-the-issues-built-on-it.md)) |
| sources | the weekly sample ([ADR-0033](0033-the-weekly-sample-draws-the-product-owners-acts.md)) |
| undo detection | the pause after ignored runs; the report's ticks |

- What is off comes back only when real use asks for it, each by a
  decision of its own.

### One autonomy axis

| Level | What the role does |
|---|---|
| `cautious` | notes and proposes |
| `normal` (default) | completes, and moves evident issues to `ready` alone |
| `enterprising` | completes, and moves issues to `ready` much more often |

- An outsider's issue is never moved to `ready` alone (ADR-0021 stands).
- An act done alone and undone by a person (`ready` taken off…) still
  demotes that kind, as ADR-0026.

### Shown on the issue, through forge labels

- **On each issue it completes or proposes on**: the label
  `workline:proposed` (GitLab: scoped `workline::proposed`) and **one**
  comment of the role, created once, then edited in place: what it did,
  why, what it wants from you, in a few lines.
- **The night's summary** goes in the CI job's summary (ADR-0035): what
  was done, proposed, left to a person, next and stuck. Read-only.
- **What waits on a person** is a saved filter on the label (GitLab: a
  label subscription or a board works too).
- **No report issue.** Migration: the first run reads the old record once,
  moves each issue's part to that issue's state, then closes the report
  with a link to the filter.
- **No global memory left**: what the record held (undone kinds,
  counters) moves to each issue's state, or is recomputed from them.

### How a person answers

| A person wants | They do | The role, next night |
|---|---|---|
| yes | label `workline:accepted` (bulk from the list) | moves it to `ready` |
| revise | a plain comment | puts 👀 on it, rewrites its draft in place, replies in one line |
| correct it themselves | edit the issue's body | the sections edited are the person's, never rewritten |
| not now | take `workline:proposed` off | proposes nothing more on that issue until it changes — that issue only, never a repo-wide demotion |
| it should not exist | close it | never reopens it |

- **Silence is never a yes.** Open proposals are capped: past the cap,
  the role slows down instead of piling up.
- **At most three rounds** (the draft and two revisions), then one line
  "left to a person" in the job summary, nothing more on the issue.
- **Reused, not new**: ADR-0021's bounded loop (`acts.ask.rounds`),
  extended to the project's own issues, with a revision count per issue.
- **GitLab**: scoped labels replace each other; `proposed` gone with
  `accepted` set is a yes, `proposed` gone alone is "not now".

### Who counts

- Only the reporter's and the project members' comments, labels and edits
  (write level, by the forge's own permissions: GitHub's author
  association, GitLab's members as ADR-0023 reads them).
- Bots, strangers and "+1" are ignored: they never bring an issue back
  into a run.
- An act whose only evidence is an outsider's comment is proposed, never
  done alone.

### Notifications

- One mail per proposal (the comment, when created); one per reply to a
  person's comment.
- Everything else is silent: labels, reactions, edits in place.
- At most five issues a run during the trial.

### Not built

- A `revise` label, comment commands, @mentions of the bot, reactions as
  votes, a permission list of workline's own, reminders, a digest.

## Trial, then build

- **One week of real use** on workline and on the maintainer's GitLab
  project, at most five issues a run.
- **Measured**: proposals accepted unchanged, edited, revised, set aside,
  ignored.
- Nothing else is built before it ends.
- **Verified on sandboxes before building**:
  - how GitLab treats an unknown `/word` at a comment's start;
  - whether GitLab creates to-dos for bot users;
  - whom GitHub notifies for a comment by the Actions bot.
- **A known defect, checked first with a conformance case**: a person
  deletes a drafted section, and the next run writes it again. A section
  the role wrote and a person deleted counts as a "no" for it.

## Alternatives rejected

- **A dashboard issue of links only**, labels on the issues: smaller to
  build, the record stays where it is; but still a report in an issue,
  which its reader found illogical, and the job summary gives the
  overview with no object to keep.
- **A GitLab board** as the surface: it is a view over these labels that
  anyone may add, not a design of its own.
- **GitHub Projects**: GraphQL and a personal token or an App
  (`GITHUB_TOKEN` cannot reach user projects), GitHub only.
- **An explicit `workline:rejected` label**: a third label to keep; taking
  `proposed` off already says "not now", closing says "never", and a
  "never" kept repo-wide is memory a person cannot see.

## Consequences

- A person meets the role where the issue is, with the forge's own tools:
  a label, a comment, an edit, a close; bulk from the issue list.
- The role's docs and code still describe and run the report issue until
  this is built; the role's README says so.
- The build starts with conformance cases: the deleted section, a
  thread between two people (no new round), an outsider's comment (no
  read), rounds spent (silence).
- Settings that tune what is now off stay, read only when a kind is
  turned back on.
