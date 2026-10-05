# ADR-0027: Derived blocks are regenerated on the default branch, after the merge

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0006 (gardening opens one merge request per task),
  ADR-0010 (docs judged on the merge request and by gardening), ADR-0017
  (the release pull request, its fixes on a pull request of their own);
  principles 4, 12, 14
- **Settles:** #149 (derived blocks make parallel pull requests conflict)
  and #97 (a derived block on a release tool's pull request)

## Context

A derived block (`<!-- workline:derive name -->…<!-- workline:end -->`) is
the output of a command the project names; the documentalist regenerated a
stale one on every run. On a push the regenerated block stopped the push
until it was committed; on a merge request the apply job committed it to
the branch. Every branch adding a conformance case therefore changed
README's case count, and two branches open together always conflicted on
that one line: #137 and #139, #144 and #146, all rebased on 2026-10-05 only
for it. A block is a function of the whole default branch, which no branch
holds until it is merged.

On a release tool's pull request (#97) the same regeneration computed the
block from the release branch and sent it, as ADR-0017 sends every fix, to
a pull request of its own on the default branch: a block quoting the
version would say the next one there before the release.

docs/research/documentalist.md ("Generated sections and parallel branches")
found that a merge driver (`merge=union`, a custom one) is never run by a
forge's merge button; that towncrier and changesets avoid changelog
conflicts by keeping nothing shared on a branch and compiling at the
release; that the all-contributors bot writes its README table on the
default branch, by a pull request of its own; and that tools checking a
generated file equal on each pull request (mdox `--check`, Kubernetes'
`hack/verify-*`) accept the rebases this issue is about.

## Decision

**A branch's derived block may lag; it is regenerated on the default
branch.** On a push, a merge request and a release (tool's pull request or
`workline route release`), the documentalist still runs the commands and
reports a stale block as `derived-behind`, which passes, writes nothing and
is only counted on a push. Gardening, which runs on the default branch,
regenerates it (`derived-stale`) and opens or updates its one merge request,
`docs: regenerate the derived blocks` (ADR-0006); adoption (`init`) too. A
block that cannot be regenerated (`derive-unknown`, `derive-failed`) still
blocks everywhere: that is the branch's fault.

A person never writes a derived block by hand, on a branch or elsewhere;
nobody needs to.

**How long the default branch stays wrong**: until the next gardening run
merges, one night here; the run reports it meanwhile, and a project wanting
less runs gardening more often. Backpressure (`max-open-merge-requests`)
still holds the merge request back, the finding still said.

### Not chosen

- **Checking "not behind" rather than "equal"** — the branch would still
  have to write its own count, and two branches still conflict on it.
- **Refreshing in the release pull request** (#97's first option) — the
  tool rewrites its branch from the default one, and a version derived
  from the release branch is wrong on the default branch before the
  release.
- **A job on each merge to the default branch** — one more workflow
  holding a write token, for a line gardening refreshes anyway.

## Consequences

- Contributors adding a conformance case no longer touch README's count;
  parallel pull requests merge one after the other with no rebase for it.
- The default branch can be wrong for up to a gardening period, and is
  reported wrong each run until the merge request is merged.
- #97's second point already held: a version said only in a derived block
  is never reported as a value left behind (only the lines outside derived
  blocks are read for it).
- Conformance: documentalist/derived-block-behind-on-a-merge-request,
  derived-block-behind-on-a-push, derived-blocks-after-two-merges.
