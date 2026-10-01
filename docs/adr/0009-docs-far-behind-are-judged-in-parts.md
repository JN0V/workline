# ADR-0009: A doc far behind sources too large for one task is judged in parts

- **Status:** accepted (2026-09-30, on the measures below)
- **Date:** 2026-09-29

## Context

A doc whose sources changed more than a task can show is judged against them
as they are now; when even that does not fit, it goes to a person, saying so.
Catching DomoticsCore up (roles/documentalist/tried.md) left 40 such docs:
30 whose sources do not fit a task, 10 the agent itself would not vouch for
— it found what was wrong, a version 1.4.1 where the code says 1.11.0, but
could not confirm the rest, and a fix that leaves `checked` alone is refused.
The note it left brings the same doc back first on the next run.

Measured on those 40 docs (docs/research/documentalist.md): 305 source files,
806 pairs of a doc and a file. One agent per source file is out of reach.
Packed into parts that fill a task, 393 calls, about 3.2 million tokens in;
most of them for a few docs naming whole libraries (`docs/index.md`: 168
files, 2.7 MB). At most 8 parts a doc, 22 docs in 81 calls. Four docs are too
long themselves to leave room for any source.

What exists: verification splits by claim and checks each against chunks of
evidence (FActScore, SAFE, MiniCheck); split by source, the known failures
are a claim true only across two sources, a claim about deleted code that
every part calls "not here", and "not here" read as "false". Strict
aggregation (RefChecker): one contradiction makes it wrong. Parts return a
structured answer merged in a last pass (LLM×MapReduce). Extracting claims
costs more than it gives; line ranges serve. Whether parts beat the whole is
not settled for our case, nor whether a light model checks as well.

## Decision

- **Only for docs that do not fit whole.** A doc whose sources as they are
  now fit a task is judged whole, as today. Parts replace "a person judges
  it" for the others, not the whole judgement.
- **A part is the doc whole, then a share of its source files**, packed to
  fill the task: not one call per file. A file larger than a part is cut in
  consecutive pieces, every piece read, so no line is vouched for unread.
  The doc comes first, the same in every part.
- **Not in parts**:
  - a doc leaving less than a quarter of the task for sources: it is over
    its budget, condensed or split first (the existing tasks);
  - sources that are docs: a doc following another is propagation, judged
    along its edge, not as evidence;
  - a doc needing more than `parts-max` parts (8 by default): its sources
    are too wide to be judged; a person narrows them or splits the doc, and
    the finding says so, with how many parts it would take.
- **Each part answers in a closed shape**, a new intention of the catalogue:
  for each passage of the doc it has something to say about — a line range,
  no claim extraction — `contradicted` (the doc's words, the source's words
  with their lines, why), `partial` (what this source leaves out) or
  `supported`. A passage not named is not covered by that source. The engine
  checks every quote against the lines it cites and drops those that are
  not there. A part writes no patch.
- **The engine merges, without AI**, by line range:
  - contradicted, and supported by no other part: **wrong**;
  - contradicted in one part, supported in another: **conflict**;
  - partial in one part or more: **to be read together** — unless another
    part supports those lines: it held the source that settles them, and
    the fix is not asked to act on a share that cannot (workline #29);
  - a passage naming an identifier of the code that no part supports:
    **uncovered** — often code renamed or gone, or a source missing.
- **One last call fixes it**: the doc, with only the excerpts the parts
  cited for what is wrong, in conflict or to be read together. It patches
  what it finds wrong, and nothing else.
- **`checked` does not move.** The fix is accepted with the doc still
  suspect: nobody read it whole against its sources. The header records the
  commit it was judged in parts at (`judged-in-parts`); the doc is not put
  before an agent again until a source changes after it. A doc the agent
  answered with a note is recorded the same way, and not asked again either.
  Whether a run with no finding may one day move `checked` is decided on the
  measures, not here.
- **Parts run on `light`, the last call on `standard`**, if the evaluation
  says so; until then both on `standard`.
- **Measured before adopted.** An evaluation case: a long doc with three to
  six sources, defects planted through it — wrong in one source, true only
  across two, about code renamed or deleted, a count or a default — and a
  clean control doc. Judged whole, then in parts forced by a lower budget,
  on the same doc; recall by kind of defect, false alarms, a defective doc
  called clean, tokens; three runs each, parts on light and on standard.
  Parts are adopted if they miss no kind of defect the whole catches,
  besides the one true across two sources, and never call a defective doc
  clean.
- **Then tried for real**, on a copy of DomoticsCore, on the docs within
  `parts-max`; the tokens counted.

## Consequences

- A doc no single task could hold gets its errors fixed, at the cost of
  many calls: tens of thousands of tokens a doc on DomoticsCore, where one
  call was before. Gardening spreads it over runs (`ai-max-calls`).
- A doc stays suspect after it is fixed; the finding says it was judged in
  parts, and when. A person clears it by moving `checked` after reading it.
- A defect true only across two sources is the known blind spot: to be read
  together catches some of it, not all.
- The engine runs several agent calls in one round, each in its own
  context: a new step of the role contract (docs/spec/role-contract.md),
  written with its conformance cases before the code.
- Docs naming whole libraries are pointed out, instead of costing a hundred
  calls each.

## Measured

2026-09-30, the evaluation cases `finds-planted-defects-whole` and
`-in-parts` (fixture `service`): one commit makes four passages of a doc
false — a default early, a count in the middle, a sentence the store and
the configuration both speak to, a name renamed late — and changes code
the doc does not describe; twelve checks, the four true sentences kept
among them. Three runs each:

| | score | tokens in / out | time |
|---|---|---|---|
| whole, Sonnet 5.5 | 12, 12, 12 | 8k / 1.5k | 12 s |
| in parts, Sonnet 5.5 (4 parts and the fix) | 12, 12, 12 | 40k / 4k | 35 s |
| parts on Haiku 4.5, fix on Sonnet | 12, 11, 12 | 37k / 36–48k | 5–6 min |
| Haiku alone, whole or in parts | 5 | — | — |
| whole, Opus 5.5 (2026-10-01) | 5, 5, 5, 12, 5, 5, 5 | 8k / 1.5k | 15 s |
| in parts, Opus 5.5 | 12, 11, 12 | 41k / 7k | 75 s |

Opus judged whole wrote a note, not a fix, six times in seven: it saw what
was wrong, but given only the diffs it would not move `checked` over
sentences it could not see — where Sonnet did. In parts, given the sources
whole, it fixed them. Parts missed nothing the whole caught, the sentence across two sources
included, and called no true sentence false: adopted, for docs too large to
be judged whole, at five times the tokens. Parts stay on `standard`: Haiku
wrote ten times more, took ten times longer, and missed once; alone, its
fix named a wrong path. Before these runs, two of three in parts had failed
on the format: one brace too many in a part's one-line YAML spoiled its
whole answer, and the parts found were all thrown away. A part's answer is
now read claim by claim, and the task asks for block style.
