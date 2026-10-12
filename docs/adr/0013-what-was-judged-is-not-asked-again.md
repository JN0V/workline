# ADR-0013: What was judged is not asked again before something changes

- **Status:** accepted; amended 2026-10-02 (a task proposed earlier in the
  run waits too); amended 2026-10-12 (a doc's `judged` says what was read)
- **Date:** 2026-10-01
- **Amends:** ADR-0012 (a fix that cannot vouch is kept), ADR-0006 (one
  merge request per task)

## Context

DomoticsCore's first gardening with parts on (2026-10-01, roles/documentalist/docs/tried.md)
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

## Amendment (2026-10-02)

workline's first nightly gardening (roles/documentalist/docs/tried.md,
2026-10-02) judged one doc whole and put thirteen off "to be judged in parts
in a later round": a CI run judged with `--no-apply` has one round, and a
round judging docs whole asks no parts. The waiting above lets the parts go
only on a night when the `suspect` merge request is still open; a person
who merges it each morning, while the code keeps making docs suspect, starves
the parts for good (seen on DomoticsCore too).

- **A task proposed earlier in the run waits too.** A run judged with
  `--no-apply --open-merge-request` goes round again when its round
  proposed a merge request for a task no earlier round proposed, and it
  deferred work (`in/more`). The next round is told that task waits, as an
  open one (`WORKLINE_OPEN_MERGE_REQUEST_TASKS`, and alone in
  `WORKLINE_PROPOSED_TASKS`), and counts it against
  `max-open-merge-requests`. Each round is a run folder to apply, a merge
  request of its own (ADR-0006). A doc put off to parts is so judged the
  same night as the docs judged whole, unless the night's caps say
  otherwise — `max-open-merge-requests`, `ai-max-tokens`,
  `parts-max-per-run` — each of which says what waits.

Chosen over alternating whole and parts across nights: alternating needs a
memory of the last night's task, which the repository does not hold — and
ADR-0013 keeps the record in the repository, not beside it; it would also
leave a doc judged whole waiting a night it need not. Going round is what
the engine already does on a machine that applies (the merge request opened,
the next round sees it waiting); the change makes the CI run, which applies
in another job, do the same. Its cost is a second task a night, within the
caps above.

## Amendment (2026-10-12)

About a doc's `judged` header only; the product owner's own `judged`, on
an issue, is unchanged.

DomoticsCore's gardening judged a doc whose sources were too large to go
whole ([ADR-0014](0014-checked-is-earned-by-what-was-read.md)): the agent
read only what changed in them, found nothing wrong there, and the run said
"nothing found wrong, `judged` recorded" of the doc
([#120](https://github.com/JN0V/workline/issues/120)). Five claims outside
every diff were false. `judged` keeps such a doc from an agent until a
source changes, and then only the new diff is read: those claims are never
read against the code, while the finding and the pull request read as all
clear.

- **`judged` records when, not what was read.** "Judged whole" above means
  judged as one doc, not in parts; it does not mean every source was read.
- **A doc not given every source whole says what was read.** Its finding,
  the run's summary and the gardening merge request name what the task
  gave — what changed in a source since `checked`, the doc it follows — and
  mark the rest "not read". "Nothing found wrong" is said of what was read
  ("nothing found wrong in what was read"), never of the rest. `pre`
  records it in `in/read-not-whole.yaml`, beside `in/read-whole.yaml`.

Not chosen, for now: the agent reading the doc's claims against the code
as it is now, not only the diff, before saying nothing is wrong. It would
vouch for more, but would cost a read of each such doc against its sources
every night one changes. Left as a follow-up; the doc still waits for a
person, as before.
