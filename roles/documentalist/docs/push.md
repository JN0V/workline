---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/adopt.go, internal/builtin/documentalist/byname.go]
checked: 79ecb95
verified: agent:claude-code
---
# Documentalist — at the push, the release and the adoption

Part of [the documentalist](../README.md).

## Before a push

On a merge request, run with `--push-to-merge-request` (the CI templates),
the docs fixed are committed to the merge request's own branch, as
pre-commit.ci does: a suggestion can only sit on lines the merge request
changes, and a doc made wrong by a change of code usually has none. From a
fork, the fix goes in one comment, as a diff to apply.

Given the commits of a push or a merge request (`range`), only the docs they
made suspect — one of their own sources touched by those commits — are
judged, whole, against every source that changed; the other suspect docs
are reported, left for gardening (ADR-0007).

A project that routes `pre-push` (`routing: {events: {pre-push: [committer,
documentalist]}}`) has those docs counted at the push, in one line, without
an agent: a push never waits on one, nor asks (ADR-0010). The person has
them judged when they choose — `workline docs` — reviews each change, and
those kept go in one `docs:` commit (ADR-0007). It judges from where the docs
were last judged, a ref it moves (`refs/workline/docs-judged`), pushed or
not: a repository pushed to main with no merge request is caught up there.

**At the release**, a doc made suspect since the last release holds it until
it is judged (ADR-0010): by the agent, as the release runs the documentalist
first, by `workline docs`, or by a person moving `checked`. One suspect
already at that release was let through then; one judged without being
vouched for, or in parts, waits for a person and does not hold it. The last
release is the highest version tag merged (`release.tags`, `v*` by
default), prereleases left out, never the nearest tag, which a hotfix merged
back would be; a shallow clone, which may lack it or a doc's `checked`,
holds the release (`shallow-clone`).

A release tool's pull request — its branch one of `release.branches`,
release-please's and the like by default — is the release (ADR-0017): on
it the documentalist does what it does at the release, every doc suspect
since the last one and the docs due then, not only what the pull
request's commits touched. Its fix does not go onto that branch, which the
tool rewrites: it goes to a merge request of its own into the release's
base, `workline/documentalist/release`, and the release pull request stays
held until that one is merged and the tool brings it in. That merge request
follows its base, as the tool's own branch does (ADR-0034): on each push to
the base, `workline follow` rebuilds it on the new tip — main's `checked`
kept beside the `judged` the fix wrote, where a rebase would conflict —
and leaves it alone when it is on the tip already, when a person committed
to it, or, asking that person, when its change no longer applies. A new fix
judged on the release pull request is not pushed over a person's commit
either: it goes in a comment there.

## Adopting a repository

On a repository where no doc says what code it describes, nothing can be
found suspect. `workline init` routes `pre-push` to the committer and the
documentalist, unless the project routes it already, then runs the
documentalist on `init`: each doc the `docs` setting covers whose header has
no `sources` is put before the agent with the repository's files, and the
agent proposes the code it describes — `sources: []` for a doc describing
none, a decision or a changelog. Its `checked` is the commit that last
changed the doc, never HEAD: the doc was written against the code as it was
then, so a source changed since makes it suspect, to be judged (`workline
docs`, a merge request) rather than vouched for unread. The judge refuses a
patch touching more than the header, a source the repository does not hold,
or another `checked`. The patches land in the working tree for the person to
review and commit. Without an agent, each doc is listed (`no-sources`) with
the commit its `checked` would name, and, when its name or a folder holding
it up to `docs/` is a code folder's or file's, those sources proposed — the
shallowest, tests aside, `docs/cli.md` → `src/cli/` — for the person to
review: never written. One finding (`sources-by-name`) says how many docs
matched: on six public repositories, 0 to 5 of 5 to 30 docs, and 261 of
backstage's 452, about half of them right (docs named after its plugins and
packages; `faq.md` under `features/` matched a `features/` folder far away).

## Without AI

All checks still run. Suspect docs are reported for a person, who clears each
one by updating `checked`. On a merge request, with a forge, they are listed
in one comment, a checklist edited on each run rather than a new comment each
push: a doc leaves it when its `checked` moves, the docs due later are listed
apart, and with none left the comment says so. None is opened on a merge
request that never had one.
