---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/adopt.go]
checked: 6013a50
verified: agent:documentalist
---
# Documentalist — at the push, the release and the adoption

Part of [the documentalist](README.md).

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

**At the release**, a doc made suspect since the last tag holds it until it
is judged (ADR-0010): by the agent, as the release runs the documentalist
first, by `workline docs`, or by a person moving `checked`. One suspect
already at the tag was let through then; one judged without being vouched
for, or in parts, waits for a person and does not hold it.

## Adopting a repository

On a repository where no doc says what code it describes, nothing can be
found suspect. `workline init` routes `pre-push` to the committer and the
documentalist, unless the project routes it already, then runs the
documentalist on `init`: each doc under `docs` whose header has no `sources`
is put before the agent with the repository's files, and the agent proposes
the code it describes — `sources: []` for a doc describing none, a decision
or a changelog. Its `checked` is the commit that last changed the doc, never
HEAD: the doc was written against the code as it was then, so a source
changed since makes it suspect, to be judged on the next push, rather than
vouched for unread. The judge refuses a patch touching more than the header,
a source the repository does not hold, or another `checked`. The patches land
in the working tree for the person to review and commit. Without an agent,
each doc is listed (`no-sources`) with the commit its `checked` would name.

## Without AI

All checks still run. Suspect docs are reported for a person, who clears each
one by updating `checked`. On a merge request, with a forge, they are listed
in one comment, a checklist edited on each run rather than a new comment each
push: a doc leaves it when its `checked` moves, the docs due later are listed
apart, and with none left the comment says so. None is opened on a merge
request that never had one.
