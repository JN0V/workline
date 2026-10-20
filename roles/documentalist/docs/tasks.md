---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/parts.go, internal/builtin/documentalist/counts.go]
checked: 7eadd606
verified: agent:claude-code
---
# Documentalist — the agent's tasks

What the agent is asked of the suspect docs, judged whole or in parts. A
doc is *vouched for* when its `checked` moves to the commit the task gave.
When gardening (the scheduled run, nightly or weekly), the agent also gets
stale docs, duplicates, condensing, merging and splitting cards, and the
pages the reader checks flag: [gardening](gardening.md). Part of [the documentalist](../README.md).

## Suspect docs

### What a task holds

- Each doc, its lines numbered.
- What changed in its sources: 240 lines of diff a doc, whatever the
  number of sources.
- Beside it, its sources as they are now, in full, when they fit, since
  vouching for every sentence needs them
  ([ADR-0012](../../../docs/adr/0012-a-doc-is-judged-against-its-sources-whole.md)).
- A doc far behind, whose changes do not fit: its sources as they are now
  alone, or [in parts](parts.md) when they do not fit either.
- Sources that do not fit, though what changed in them does: what changed
  alone on a merge request; when gardening, [in parts](parts.md)
  against them all.
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
  release and for stale docs. What the run says of it names what was read
  and what was not ([the judge](judge.md#what-the-finding-says)).
- **A source that cannot be read** — a file gone, or a section whose
  heading is renamed or removed — is never given as empty: the doc is
  judged on what changed, so `checked` cannot move. With nothing else to
  judge it on (far behind, or stale), it is not put before the agent: the
  finding says which source, and a person judges it.
- **A line count off**: the doc is told each one, with the real count,
  counted by the engine. `checked` cannot move while one stands
  (`checked-over-count-off`), so the fix brings it to that count or leaves
  `checked`. That change needs no claim, the task says: the engine's count
  is the evidence, and a count is not words a file holds, to be quoted.
- **In parts**: a fix never moves it; the engine does when the parts
  support every line stating something, unless the project turns
  `parts-move-checked` off
  ([judged in parts](parts.md#when-checked-moves-after-parts)).

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
  and a `note` says what could not be confirmed, naming its doc in `doc`:
  that doc's `checked` does not move (`checked-over-note`,
  [the judge](judge.md#moving-checked)).

## Judged in parts

A suspect or stale doc whose sources, as they are now, do not fit a task
is judged in parts when gardening
([ADR-0009](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md)):
each part holds the doc whole and a share of its sources, and the engine
puts their claims together. How: [judged in parts](parts.md).
