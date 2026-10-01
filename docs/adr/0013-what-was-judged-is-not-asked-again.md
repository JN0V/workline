# ADR-0013: What was judged is not asked again before something changes

- **Status:** accepted
- **Date:** 2026-10-01
- **Amends:** ADR-0012 (a fix that cannot vouch is kept), ADR-0006 (one
  merge request per task)

## Context

DomoticsCore's first gardening with parts on (2026-10-01, roles/documentalist/tried.md)
judged one doc: Opus fixed a version five times and left `checked`, as
ADR-0012 asks of a fix it cannot vouch for. Run again before the person
merged it, the line put the same doc first, judged it again — 26k tokens —
and would have rewritten its pull request to the same end; merged, the doc
was still suspect, and first again the night after. Thirty-four docs waited
to be judged in parts behind it: a run judging docs whole asks no parts, and
a CI run has one round. ADR-0012 said a doc judged once is not judged again
until its sources change; nothing made it so.

Others (docs/research/ci-and-forge.md, "Caps and caches"): Renovate waits
while its pull requests are open; CodeRabbit reviews only what changed since
its last review; agentics' wiki writer keeps the hashes of what it read, but
lets the LLM compute them, and leaves the model out of the key.

## Decision

- **`judged: <commit>`**, in the doc's header, records the commit a doc was
  judged whole at without being vouched for. The agent's patch sets it when
  it leaves `checked`; the judge refuses such a patch without it
  (`judged-not-set`). Like `judged-in-parts` (ADR-0009), the doc is not put
  before an agent again until one of its sources changes after that commit,
  and stays suspect, for a person. A doc whose sources do not tell at all
  gets a patch setting only `judged`, and a note.
- **A task waits for its merge request.** When gardening, the docs a task
  would judge are not put before an agent while that task's merge request
  is open (`workline/documentalist/suspect`, `stale`; `fix` for a doc
  judged in parts). The engine gives the role the open tasks
  (`WORKLINE_OPEN_MERGE_REQUEST_TASKS`). So a doc judged in parts no longer
  waits behind the docs judged whole: their pull request waiting, the
  parts go.
- **A run may cap its tokens** (`ai-max-tokens`, docs/spec/role-contract.md):
  checked against what was spent, never estimated.

The record is the doc's header, in the repository, not a cache beside it:
it holds on the merge request, at night and on a laptop, and a person sees
it in review.

## Consequences

- Gardening goes as fast as its pull requests are reviewed, and spends
  nothing while they wait.
- A doc judged without being vouched for waits for a person, as one judged
  in parts does; one more header line.
- An agent forgetting `judged` is asked again, with the reason: a call more.
- A note alone, with no patch, records nothing: that doc is asked again.
