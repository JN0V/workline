---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/adopt.go, internal/builtin/documentalist/byname.go, internal/builtin/documentalist/releasekept.go, internal/engine/night.go]
checked: c29e9973
verified: agent:claude-code
judged-in-parts: c7e1e98
---
# Documentalist — at the push, the release and the adoption

Part of [the documentalist](../README.md). Gardening, named below, is the
scheduled run, nightly or weekly. A doc is *vouched for* when its `checked`
moves to the commit the task gave.

## Before a push

### On a merge request

- Run with `--push-to-merge-request` (the CI templates), the docs fixed are
  committed to the merge request's own branch, as pre-commit.ci does: a
  suggestion can only sit on lines the merge request changes, and a doc
  made wrong by a change of code usually has none.
- From a fork, the fix goes in one comment, as a diff to apply.
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
  `workline docs` and at the release
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
- **A real fix is one commit**, naming the doc: "docs: bring
  docs/tech/auth.md in line with the code". A move of `checked` too.
- **A fork's merge request** keeps its record too.
- **Committed, as before**, where the comment cannot keep it:
  - a forge that does not say which commit a merge request became
    (`local`, `cmd:`);
  - a comment the job cannot read, said (`record-unread`): on GitLab, give
    the judging job `GITLAB_TOKEN`
    ([ci-gitlab.md](../../../docs/ci-gitlab.md#what-the-template-does));
  - a doc with a source in another repository.
- Only a comment a bot or a person of the project wrote is read.

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

A doc made suspect since the last release holds it until it is judged
([ADR-0010](../../../docs/adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md)):
by the agent, as the release runs the documentalist first, by `workline
docs`, or by a person moving `checked`.

- One suspect already at that release was let through then.
- One judged without being vouched for, or in parts, waits for a person
  and does not hold it.
- One too large to be judged whole holds it until the next gardening
  judges it in parts, or a person moves `checked`.
- The last release is the highest version tag merged (`release.tags`, `v*`
  by default), prereleases left out — never the nearest tag, which a
  hotfix merged back would be.
- A shallow clone, which may lack it or a doc's `checked`, holds the
  release (`shallow-clone`).

### A release tool's pull request

Its branch one of `release.branches` (release-please's and the like by
default), it is the release
([ADR-0017](../../../docs/adr/0017-the-release-manager.md)): on it the
documentalist does what it does at the release — every doc suspect since
the last one and the docs due then, not only what the pull request's
commits touched.

- **No agent on its moves**
  ([ADR-0050, §7](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md#7-decided-by-the-maintainer-on-2026-10-19)):
  on each push to it, the docs due are counted with no agent and hold the
  release; one finding says so (`ai-waits-for-night`). A doc fix would
  otherwise change its head after a person's yes.
- **The agent judges them every night**, on that pull request's head,
  checked out for the run, then put back:
  - the other nights, the templates' release docs run:
    `workline route release --target merge-request:release` finds the
    release pull request open into the branch. None open, or no doc due:
    no agent is asked. GitHub: `workline-release-docs.yml`; GitLab: a
    schedule with `WORKLINE_TASK=release-docs`;
  - the gardening's night, before its own tasks. Gardening leaves the
    docs that fix holds;
  - with no agent that night, it is said (`release-not-judged`), and the
    release waits.
- **Sooner, by hand**: run the release docs workflow (GitHub) or play its
  schedule (GitLab); or, where an agent may judge,
  `workline route release --target merge-request:<n> --push-to-merge-request`.
- **Its fix does not go onto that branch**, which the tool rewrites: it
  goes to a merge request of its own into the release's base,
  `workline/documentalist/release`. The release pull request stays held
  until that one is merged and the tool brings it in.
- **Its record stands for the heads the tool rebuilds**: the
  documentalist's comment on the release pull request keeps the head each
  fix set `checked` or `judged` to, and the ids of the doc's sources
  there. Once the fix is merged and the tool rebuilt its branch, a doc
  whose sources are the same is not judged again, the same version bump
  included; one whose source changed is
  ([ADR-0034, amended](../../../docs/adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-19-the-fixs-record-stands-for-the-rebuilt-heads)).
- **That merge request follows its base**, as the tool's own branch does
  ([ADR-0034](../../../docs/adr/0034-the-release-fix-follows-its-base.md)):
  on each push to the base, `workline follow` rebuilds it on the new tip —
  main's `checked` kept beside the `judged` the fix wrote, where a rebase
  would conflict.
- **`workline follow` leaves it alone** when it is on the tip already, when
  a person committed to it, or, asking that person, when its change no
  longer applies.
- **A new fix judged on the release pull request** is not pushed over a
  person's commit either: it goes in a comment there; nor applied once the
  tool moved its branch since the night's run (`moved-on`): the next run
  judges the new head.

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
Nor on a release tool's pull request, whose docs due are judged at
night (`ai-waits-for-night`, [above](#a-release-tools-pull-request)).
