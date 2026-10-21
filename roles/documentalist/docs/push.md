---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/adopt.go, internal/builtin/documentalist/byname.go]
checked: 8e3fad75
verified: agent:claude-code
judged-in-parts: c7e1e98
---
# Documentalist — at the push and the adoption

Part of [the documentalist](../README.md). Gardening, named below, is the
scheduled run, nightly or weekly. A doc is *vouched for* when its `checked`
moves to the commit the task gave.

## Before a push

### On a merge request

- **Nothing is pushed to its branch**, not a fix, not a move of
  `checked`: the fix goes in one comment, a diff for its author to apply
  ([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr)).
  A suggestion could only sit on lines the merge request changes.
- **The docs it made suspect are said, never holding the merge**: the
  author fixes the prose the change made wrong, and may move `checked` of
  those read against it. Once merged, gardening's docs pull request moves
  the others' `checked`, nightly until the weekly one: only for a doc whose
  sources fit a task whole (`whole-chars`); the others wait for
  [parts](parts.md), a few a night, or for a person.
- Given the commits of a merge request (`range`), only the docs they
  touched are read, as at a push: those they made suspect — one of their
  own sources touched by those commits, a merge's own change too — are
  judged, whole, against every source that changed
  ([ADR-0007](../../../docs/adr/0007-docs-reviewed-after-the-code-before-the-push.md)).
  A doc suspect from before is gardening's; the checks across every doc
  run on main after the merge, once
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
  [how](../../../docs/main-after-merge.md)).
- One too large to be judged whole is left for the next gardening, which
  judges it [in parts](parts.md), there as with
  `workline docs`
  ([#458](https://github.com/JN0V/workline/issues/458)).

### A doc found with nothing to fix: no commit

On a person's merge request, the agent may find nothing wrong in a doc yet
not vouch for every sentence: its patch only records when (`judged`). That
took a commit on their branch, 41 of 42 in two weeks
([#463](https://github.com/JN0V/workline/issues/463)). Now:

- **Nothing is pushed for it.** Its record is kept, hidden, in the
  documentalist's comment on the merge request: the commit judged, and the
  ids of the doc and its sources there
  ([ADR-0013, amended](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-17-on-a-persons-merge-request-the-record-waits-in-a-comment)).
- **The next push does not judge it again** while the doc and its sources
  stay the same; a change to either has it judged again.
- **Once merged, gardening writes `judged`** on the default branch
  ([records](gardening.md#records-kept-on-merged-merge-requests)); for a
  doc whose sources are too large to go whole, it reads it in parts
  instead ([#462](https://github.com/JN0V/workline/issues/462)).
- **A real fix goes in the comment's diff**, a move of `checked` too.
- **A fork's merge request** keeps its record too.
- **In the comment's diff too**, where the documentalist's comment
  cannot keep the record:
  - a forge that does not say which commit a merge request became
    (`local`, `cmd:`);
  - a comment the job cannot read, said (`record-unread`): on GitLab, give
    the judging job `GITLAB_TOKEN`
    ([ci-gitlab.md](../../../docs/ci-gitlab.md#what-the-template-does));
  - a doc with a source in another repository.
- Only the comment workline's own account wrote is read: the GitHub App,
  else the token's user; on GitLab, `WORKLINE_GITLAB_TOKEN`'s user, or
  one named in `gitlab: {bots}` ([config](../../../docs/config.md)).
  Another's carrying its marker, a person of the project's too, is not:
  its docs are judged again, said once (`record-unread`,
  [ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)).

### At the push

A project that routes `pre-push` (`routing: {events: {pre-push: [committer,
documentalist]}}`) has those docs counted at the push, in one line, without
an agent: a push never waits on one, nor asks
([ADR-0010](../../../docs/adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md)).

- **Only the docs the commits touched are read**: one of their sources,
  or the doc itself; their own budgets and links are counted. The other
  docs — suspect from before, a passage written twice, a link gone — are
  said on main after the merge and by gardening: about 5 s less a push
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)).
- The person has them judged when they choose — `workline docs` — reviews
  each change, and those kept go in one `docs:` commit
  ([ADR-0007](../../../docs/adr/0007-docs-reviewed-after-the-code-before-the-push.md)).
- `workline docs` judges from where the docs were last judged, a ref it
  moves (`refs/workline/docs-judged`), pushed or not: a repository pushed
  to main with no merge request is caught up there.

## At the release

The documentalist does not run at the release: the release never waits
for the docs
([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly)).
A release tool's pull request is judged as any other.

## Adopting a repository

On a repository where no doc says what code it describes, nothing can be
found suspect. `workline init` routes `pre-push` to the committer and the
documentalist, unless the project routes it already, then runs the
documentalist on `init`.

### With an agent

- Each doc the `docs` setting covers whose header has no `sources` is put
  before the agent with the repository's files. The agent proposes the
  code it describes — `sources: []` for a doc describing none, a decision
  or a changelog.
- Its `checked` is the commit that last changed the doc, never HEAD: the
  doc was written against the code as it was then, so a source changed
  since makes it suspect, to be judged (`workline docs`, a merge request)
  rather than vouched for unread.
- The judge refuses a patch touching more than the header, a source the
  repository does not hold, or another `checked`.
- The patches land in the working tree for the person to review and
  commit.

### Without an agent

- Each doc is listed (`no-sources`) with the commit its `checked` would
  name.
- When its name or a folder holding it up to `docs/` is a code folder's or
  file's, those sources are proposed — the shallowest, tests aside,
  `docs/cli.md` → `src/cli/` — for the person to review: never written.
- One finding (`sources-by-name`) says how many docs matched. On six public
  repositories: 0 to 5 of 5 to 30 docs, and 261 of backstage's 452, about
  half of them right (docs named after its plugins and packages; `faq.md`
  under `features/` matched a `features/` folder far away).

## Without AI

All checks still run. Suspect docs are reported for a person, who clears
each one by updating `checked`. On a merge request, with a forge, they are
listed in one comment, a checklist edited on each run rather than a new
comment each push:

- a doc leaves it when its `checked` moves;
- the docs due later are listed apart;
- with none left, the comment says so;
- none is opened on a merge request that never had one.

On a draft, an agent set, the engine asks it nothing
([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)):
the docs the change made suspect are counted, and said judged once it
is marked ready (`ai-waits-for-ready`); no checklist asks a person to.
