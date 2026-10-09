# ADR-0038: The product owner proposes on the issue; a person answers there, with forge labels

- **Status:** accepted — built (amended below, twice on 2026-10-09); a week's trial running
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
  demotes that kind, as ADR-0026 — amended below: on that issue first,
  everywhere after `undone-max`.

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
  `accepted` set is a yes, `proposed` gone alone is "not now". Amended
  below: on the Free plan they do not, and `accepted` present is the yes.

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

## Amendment (2026-10-08): built, two answers decided by the maintainer

**`workline:accepted` present is the yes, on every plan.** GitLab Free
keeps both scoped labels
([checked](../research/surfacing-proposals.md#checked-on-the-sandboxes)):
the role takes `proposed` off itself; either spelling of `accepted` is a
yes.

**An act undone is a proposal on that issue first.** Undone by a person,
that kind is proposed on that issue only; once undone `undone-max` times
across the open issues (3, from 1 to 20), on every issue. One undo never
switches `normal` off.

**What building it settled**:

- **One comment per issue**: the state comment, what the role did and
  proposes on top, the state folded.
- **The ticks and the pause go with the report**: no global memory;
  `ignored-runs-max` is said no longer read.
- **Off by default**: `changed-needs`, `weekly-sample` (reading every
  issue's state when on), the other kinds by their mode.
- **GitLab labels**: `workline::proposed`, `workline::accepted`.
- **Revise**: one revision a run, however many comments; a comment on an
  issue with no proposal is context.
- **Slowing down**: `proposals-max` (10).
- **The night's summary**, grouped, one line an issue linked, 10 a group
  then "and N more" and the filter: the judging job decides the plan,
  reading only, and says it; the applying job does that plan
  ([backlog-acts.md](../spec/backlog-acts.md#on-the-issue)).
- **`enterprising`** raises the caps; `ready` never has a draft.
- **Reading order** (from [#270](https://github.com/JN0V/workline/pull/270)):
  answers, then a person's new issues newest first, the catch-up last.
- **No quick action**: a line starting with `/` is escaped, every role.
- **Migration**: an old report's proposals, acts, undos and closings move
  to their issues; its pause, ticks, measure and changes are dropped.

## Amendment (2026-10-09): what the first nights showed

The first gardening nights
([run 37895990993](https://github.com/JN0V/workline/actions/runs/37895990993))
and the maintainer's triage of the drafts it had left:

- **Silence on a new issue.** Three issues with a Need and a Validation
  got nothing: the agent drafted their Scope and Verification, named no
  file, and the engine dropped the whole refine. A Scope's files are now
  read from its text when none is given; a Scope naming none is left out,
  the rest written; an issue read and left with nothing is said.
- **Drafts a reader could not use**: a Need rephrasing the solution, a
  Validation as a formula, internal words, a closed issue cited as work
  to come. The agent writes for a reader, as the project's user docs
  are written, and never invents: too thin, several topics, or resting
  on a closed issue, it asks one plain question. The engine links a
  decision cited bare and says a cited issue closed.
- **A closing when `close-obsolete` is off** is said on the issue, its
  evidence quoted, for a person to close; the role closes nothing, and
  `workline:accepted` does not close it. Off stays off: no act.
- **Drafts are marked unseen**: the visible "Draft by the product owner"
  line contradicted the comment's `workline:accepted`; an edited draft
  is a person's. An import's origin line is hidden too.
- **Labels follow the state**: no draft left, no `workline:draft`; set
  aside with no draft, no `workline:to-refine` either.
- **The summary counts by issue** what a person sees — done, proposed,
  set aside —, says each line in plain words, no rule's name.

## Amendment (2026-10-09, evening): a person's word comes first

The night after
([run 37927365638](https://github.com/JN0V/workline/actions/runs/37927365638)):
the maintainer commented on two issues set aside, and the run read five
new issues instead; three issues an earlier engine had failed on were
never read again; five complete issues of the maintainer's got no
`ready`, the agent saying it "did not check who wrote" their sections.

- **A person's comment on any issue is read first**, with the answers:
  set aside, read and left, or never read. Before, only an issue
  waiting on a person was; the others came after the new issues, and a
  set-aside one, once read, still had every act dropped. The agent is
  told their word decides; on an issue set aside, to propose only what
  it asks for or opens again. Set aside, it is so no more once the run
  decides an act on it.
- **Said with the "not now"**: a comment written before the label came
  off is heard with it (the forge's label events); only one after brings
  the issue back.
- **The role's comment is told by its marker**, never by its author: the
  role may run with a person's token.
- **A fix reaches the issues it failed on**: an issue read and left with
  nothing — a section lacking, no code named, no act ever done, nothing
  waiting on anyone — under an older way of reading is read again once,
  after what changed, within `issues-per-run`. The way of reading is a
  number the engine keeps (`backlog.Reading`), raised by a change that
  may give such an issue an act; each issue's state keeps the one it was
  read under (`rules`).
- **The agent is told whose each section is**: a section not marked a
  draft is a person's; an issue whose four sections are there, Need and
  Validation a person's, is said open to `ready` when evident; what the
  level means is said. Whether it is evident stays the agent's judgment;
  the engine checks the sections, and an outsider's `ready` is proposed.
