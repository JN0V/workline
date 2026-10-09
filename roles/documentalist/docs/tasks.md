---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/parts.go, internal/builtin/documentalist/freshness.go, internal/builtin/documentalist/dedupe.go, internal/builtin/documentalist/condense.go, internal/builtin/documentalist/mergecard.go, internal/builtin/documentalist/counts.go]
checked: 2522e53
verified: agent:claude-code
---
# Documentalist — the agent's tasks

What the agent is asked of the suspect docs, judged whole or in parts. A
doc is *vouched for* when its `checked` moves to the commit the task gave.
When gardening (the scheduled run, nightly or weekly), the agent also gets
stale docs, duplicates, condensing, merging and splitting cards:
[gardening](gardening.md). Part of [the documentalist](../README.md).

## Suspect docs

### What a task holds

- Each doc, its lines numbered.
- What changed in its sources: 240 lines of diff a doc, whatever the
  number of sources.
- Beside it, its sources as they are now, in full, when they fit, since
  vouching for every sentence needs them
  ([ADR-0012](../../../docs/adr/0012-a-doc-is-judged-against-its-sources-whole.md)).
- A doc far behind, whose changes do not fit: its sources as they are now
  alone, or [in parts](#judged-in-parts) when they do not fit either.
- The commits its `checked` must name.

### How many

- At most `ai-max-calls` docs per call, and no more than fits the role's
  context budget.
- Docs left out are taken in the run's next round, once the first ones are
  applied ([role contract](../../../docs/spec/role-contract.md#one-run),
  "Again"); past the last round, they stay suspect for a person or a later
  run.
- `ai-max-tokens` caps what a run spends, all calls together: past it, the
  agent is asked nothing more and the rest waits.

### When `checked` may move

`checked` is earned by what the task gave
([ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md)):

- **Every source whole**: a doc is told it may move `checked` only when
  every one of its sources went whole into the task, as it is now — every
  text file under a source path, the section of a source doc — up to the
  project's `whole-chars` together (20,000 characters by default, 25,000
  at most; the task grows by as much as it is raised).
- **Otherwise**: a doc judged on diffs alone, or beside a doc of the same
  task it follows, is told that `checked` cannot move. Its fix leaves it
  and sets `judged`, the judge refusing a move (`checked-unread`), and the
  doc stays suspect for a person. The same holds for docs due at the
  release and for stale docs.
- **A line count off**: the doc is told each one, with the real count,
  counted by the engine. `checked` cannot move while one stands
  (`checked-over-count-off`), so the fix brings it to that count or leaves
  `checked`. That change needs no claim, the task says: the engine's count
  is the evidence, and a count is not words a file holds, to be quoted.

### Every word removed is cited

Every task judging docs — suspect, stale, due at the release, the fix after
the parts — says that every word a patch takes out of a body is cited
([ADR-0014, step 2](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md#2-the-mechanical-checks-no-agent);
[how the judge reads claims](judge-claims.md)):

- beside the patch, a `claim` for each place, with the doc's lines and
  either a source's words quoted (`status: contradicted`, `source: {doc's
  source file, quote}`) or a name the code no longer has (`status: gone`,
  `name`);
- `doc` names the doc when the answer patches several;
- sources are not numbered in the task, so a claim quotes, and the engine
  finds the quote;
- a comment is not evidence: with nothing else to cite, the words stay,
  and a `note` says what could not be confirmed.

## Judged in parts

[ADR-0009](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md),
when the project sets `judge-in-parts: true` (off by default, until the
evaluation has measured it): a suspect or stale doc whose sources, as they
are now, do not fit a task is judged in parts instead of going to a person.

### The parts

- Each part holds the doc whole, then a share of its source files packed
  to fill the task.
- A file larger than a part is cut into consecutive pieces, each read, so
  no line goes unread.
- Sources that are docs are left out: following a doc is propagation, not
  evidence.
- Each part answers with `claim`s only — a range of the doc's lines
  `contradicted`, `partial` or `supported`, quoting the doc and the source
  — and the engine asks each in a context of its own
  ([role contract](../../../docs/spec/role-contract.md#one-run), "In
  parts").

### Reading the claims, no AI

- A claim whose quotes are not at the lines it cites is dropped
  (`claims-dropped`), for a person, who reads the doc whole: the claim
  dropped may have been the one saying what is wrong. The doc is recorded
  as judged all the same — asked again each night, its parts cost as much
  to the same end
  ([ADR-0014, step 4](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md#4-acceptance-written-before-it-is-measured)).
- Contradicted and supported by no other part: wrong.
- Contradicted and supported by another: a conflict.
- Partial: to be read together — unless another part supports the same
  lines, which settles them.
- A line naming a name from the code that no part speaks of: `uncovered`,
  for a person.

### The fix

- One last call (`fix`) gets the doc and only the source lines the parts
  cited, and patches what is wrong.
- The judge refuses a fix that moves `checked` (`checked-moved-in-parts`).
- The header records the commit it was judged in parts at,
  `judged-in-parts`, found wrong or not, and the doc stays suspect, saying
  so: nobody read it whole.
- It is not put before an agent again until one of its sources changes
  after that commit; a person clears it by reading it whole and moving
  `checked`.

### Not judged in parts

Left to a person, saying why:

- a doc leaving less than a quarter of a task for its sources: condense or
  split it first;
- one with sources in another repository;
- one needing more than `parts-max` parts (8), reported `sources-too-wide`
  with how many it would take: narrow its sources or split the doc;
- one whose parts did not all answer (`part-unanswered`): nothing they
  found is kept, it is judged again later.

At most `parts-max-per-run` parts (16) are asked in a run; a doc past it
waits for the next round. A run asking parts asks nothing else.

