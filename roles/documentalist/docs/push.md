---
sources:
  - internal/engine/engine.go#pushToMergeRequest
  - roles/documentalist/code/freshness.go#wholeCharsDefault
  - roles/documentalist/code/documentalist.go#docNames
  - roles/documentalist/code/documentalist.go#touchesNames
  - roles/documentalist/code/documentalist.go#holdsName
  - roles/documentalist/code/part.go#partChanged
  - roles/documentalist/code/parts.go#leftForGardening
  - roles/documentalist/code/parts.go#partsLater
  - roles/documentalist/code/personsaid.go#personSaid.line
  - roles/documentalist/code/personsaid.go#personSaidBody
  - roles/documentalist/code/documentalist.go#checklist
  - roles/documentalist/code/documentalist.go#preRun.leaveForNextRound
  - internal/engine/engine.go#AIWaitsReady
  - internal/engine/engine.go#ReadyInput
  - internal/engine/engine.go#AIJudgedAtReady
  - internal/engine/engine.go#warnsOnMergeRequest
  - internal/kit/verdict/verdict.go#WaitsForReady
  - roles/documentalist/role.yaml#events
  - roles/documentalist/role.yaml#merge-request-ai
  - defaults/ci/github/workline.yml#concurrency
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — at the push and the adoption

Part of [the documentalist](../README.md). Gardening, named below, is the
scheduled run, nightly or weekly. A doc is *vouched for* when its `checked`
moves to the commit the task gave.

- [A doc found with nothing to fix](push-records.md): no commit on a
  person's merge request, its record kept in a comment.
- [At the push](push-local.md): the docs counted, judged with
  `workline docs`; on a merge request, only the docs its commits touched.
- [Adopting a repository](adopt.md): `workline init`, each doc's sources
  proposed.

## On a merge request

- **Nothing is pushed to its branch**, not a fix, not a move of
  `checked`: the fix goes in one comment, a diff for its author to apply
  ([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr)).
  A suggestion could only sit on lines the merge request changes.
- **The docs it made suspect are said, never holding the merge**
  ([cut 3](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr)):
  the author fixes the prose the change made wrong, and may move `checked` of
  those read against it. Once merged, the weekly docs pull request moves
  the others' `checked`: only for a doc whose sources fit a task whole
  (`whole-chars`); the others wait for [parts](parts.md), a few a run,
  or for a person.
- **A user page it takes over its budget blocks**: within it at the
  merge base (under its old name if moved), over it on the head; the
  author condenses or splits it. One
  already over on the base only warns
  ([amended](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#amendment-2026-11-25-a-budget-a-pull-request-adds-blocks)).
- **A doc writing names in backticks is suspect only when a line it
  names changes**: a file it names, or a line removed or changed holding
  a name or a value it writes; a line added alone leaves it
  ([ADR-0014, amended](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md#amendment-2026-10-21-suspect-only-when-a-line-it-names-changes)).
- **A source naming a part of a file** — `f.go#Name`, `f.go#Type.Method`,
  `ci.yml#jobs.test` — makes the doc suspect only when that part changes;
  a task reads that part, not the file
  ([ADR-0053](../../../docs/adr/0053-a-source-may-name-a-part-of-a-file.md)).
- A doc too large to be judged whole is left for the next gardening,
  which judges it [in parts](parts.md), there as with `workline docs`
  ([#458](https://github.com/JN0V/workline/issues/458)).
- **A doc left to a person is said once** on a merge request, kept in a
  comment of its own ("Docs left to a person"); a later push says it
  again only when the doc or its sources changed
  ([#1361](https://github.com/JN0V/workline/issues/1361)).

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

The agent judges a merge request once, on the run at ready
([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#2-the-moments)):
a push after it asks nothing, and the docs it made suspect are listed
(`judged-at-ready`), for the author and the docs pull request. A run at
ready cancelled by a quick push leaves its docs to the weekly docs pull
request.

- **A patch the judge refuses** on a merge request is said as a warning,
  nothing of it applied, never holding the merge; no stronger model is
  asked again
  ([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#build-order)).

## At the release

The documentalist does not run at the release: the release never waits
for the docs
([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly)).
A release tool's pull request is judged as any other.

