# ADR-0009: A doc far behind sources too large for one task is judged in parts

- **Status:** accepted (2026-09-30, on the measures below); amended 2026-10-14 (on by default), 2026-10-17 (a conflict is on the same words; only when gardening; a partial quote is no proof), 2026-10-18 (a doc judged on its change alone; `checked` after parts, measured, off)
- **Date:** 2026-09-29

## Context

A doc whose sources changed more than a task can show is judged against them
as they are now; when even that does not fit, it goes to a person, saying so.
Catching DomoticsCore up (roles/documentalist/docs/tried.md) left 40 such docs:
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

## Amendment (2026-10-14)

Judging in parts is **on by default** (`judge-in-parts: true`), decided by
the maintainer on [#82](https://github.com/JN0V/workline/issues/82#issuecomment-6163487736)
after six nights, 2026-10-01 to 10-13, on DomoticsCore and workline:

- A doc in parts cost 130k to 250k tokens in, against 10k to 40k judged
  whole; a night is capped by `parts-max-per-run` (16 parts).
- The two faults those nights found are fixed: a doc judged in parts was
  judged again each night ([#356](https://github.com/JN0V/workline/issues/356));
  a fix wrote a wrong code example, now said unverified or checked by the
  project's command ([#121](https://github.com/JN0V/workline/issues/121)).
- A project that will not pay for docs too large turns it off; they then
  go to a person, as before.
- A doc that cannot be judged in parts whatever its sources hold — too
  long itself, or sources in another repository — says so in the round
  that finds it, not "in a later round".

## Amendment (2026-10-17)

A contradiction is a **conflict** only when another part's support quotes
some of the same words of the doc, where they stand — not merely lines
that overlap ([#440](https://github.com/JN0V/workline/issues/440)):

- A sentence spans lines, so a part supporting the words beside a
  contradiction shared its lines. The evaluation kept a false sentence
  in two runs of five; replayed, each could be such a disagreement, which
  the fix call may leave as it is.
- Replayed with no agent: "Each one lasts the configured session TTL"
  (lines 22-23), contradicted by the store, and "24 hours by default"
  (line 23), supported by the configuration, were a conflict; the first
  is now found wrong. Words both parts quote stay a conflict.
- A partial claim is still settled by a support covering its lines.

## Amendment (2026-10-17): only when gardening

Judging in parts runs **only when gardening**, decided by the maintainer
on [#458](https://github.com/JN0V/workline/issues/458#issuecomment-6237581014):

- **What [#82](https://github.com/JN0V/workline/issues/82)'s default did**: parts were asked on every event, so a merge
  request touching a source of a doc too large paid that doc's parts. The
  evaluation case `stays-on-the-task`: 6 calls, about 103k tokens in,
  against 1 call and 13.5k on 2026-10-11. workline's pull requests
  [#436](https://github.com/JN0V/workline/pull/436) and
  [#444](https://github.com/JN0V/workline/pull/444): 7 calls each, about
  129k tokens in, for the same doc 3 minutes apart; neither fix changed
  anything.
- **Now**: on a merge request, at the push or the release, such a doc is
  left for the next gardening, which judges it in parts, and its finding
  says so. One past `parts-max` is said `sources-too-wide` there too.
- **The price**: a fix such a doc needs because of the change waits for
  the next gardening, on that run's own merge request.
- `parts-always`, set only to measure whole against parts, judges in
  parts on any event.
- Nights were not where the cost grew: none on workline asked a part
  from 10-15 to 10-17.

## Amendment (2026-10-17): a partial quote is no proof

A quote a part judged `partial` for a passage, no part finding those
lines wrong, proves nothing wrong there
([#467](https://github.com/JN0V/workline/issues/467)):

- **What happened**: on DomoticsCore, three parts called a line partial.
  The fix cited one of their quotes, a placeholder default, in place of
  the topic the code adopts, and made a true line false. Its own note
  said the excerpts did not settle it.
- **Now**: words the fix takes out on such a quote alone are withheld for
  a person (`citation-unchecked`), the rest applied; the fix's task says
  so. A quote standing elsewhere in the code too still counts.
- **Measured** on every fix after parts kept, DomoticsCore and workline,
  2026-10-01 to 10-17: 13 fix calls; the fix changed 3 places no part
  found wrong. The two that took words out each cited the quote a part
  judged partial, and both took out true words.
- **The price**: a defect true only across two shares, fixed on such a
  quote alone, goes to a person; the fix still settles it with any other
  quote. Words only added still need no claim.
- "The fix is not asked to act on a share that cannot", above, now holds
  for what the fix cites, not only for what it is shown.

## Amendment (2026-10-18): a doc judged on its change alone

A doc whose sources do not fit a task whole, though what changed in them
does, is **read in parts when gardening**, against all its sources as
they are now ([#462](https://github.com/JN0V/workline/issues/462)):

- **What happened**: judged on what changed alone, its `judged` kept it
  from an agent until a source changed. On DomoticsCore, 2026-10-04, five
  claims outside the change were false
  ([ADR-0013, amended](0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-12-what-a-docs-judged-vouches-for)).
- **Now**: a merge request still judges it on its change. Gardening reads
  it in parts: a `judged` its change earned does not hold it, a
  `judged-in-parts` since its sources last changed does. A record kept on
  a merged merge request is not written for it: the parts record when it
  was read.
- **Not in parts**, judged on its change as before: a doc too long
  itself, sources in another repository, past `parts-max`, no code among
  its sources.
- **`checked` does not move**, as for any doc in parts; whether it may is
  [#482](https://github.com/JN0V/workline/issues/482).
- **The night's caps hold**: `parts-max-per-run` a round,
  `ai-max-tokens` the run. The docs read longest ago go first, by the
  later of `checked` and `judged-in-parts`, not the first by name.
- **Measured** with a recording agent, no tokens: CI's night on copies,
  tokens estimated at 711 + 0.82 a character. Uncapped, what each night would ask in
  parts, as if no night before it had read a doc, docs / parts / tokens
  in:

  | Night (main's tip) | Before | After |
  |---|---|---|
  | workline, 10-12 (4950320) | none (the night: 81k) | 5 / 28 / 1.01M |
  | workline, 10-14 (b998ed0) | none (the night: 86k) | 8 / 39 / 1.39M |
  | workline, 10-16 (c5a4294) | none (the night: 56k) | 2 / 10 / 0.33M |
  | workline, 10-18 (11d82b2) | 3 / 13 / 0.48M | 10 / 52 / 1.87M |
  | DomoticsCore, 10-18 (c8c2fea) | 64 / 234 / 7.98M | 66 / 240 / 8.18M |

  Capped as both repositories set it (8 parts a round, 400k tokens a
  run), workline's night of 10-18 cost 405k before, 11 calls, and 426k
  after, 12 calls; DomoticsCore's 420k, 13 calls, both: a run stops past
  its cap, the call crossing it paid.
- **The price**: on workline, a night that asked no part now spends its
  cap, about 400k tokens in where 56k to 86k were spent, and what waits
  outlasts a night; on DomoticsCore, already at its cap with docs far
  behind, two docs more wait. A project lowers `ai-max-tokens` or
  `parts-max-per-run`, or turns `judge-in-parts` off.
- **Past the caps**, a doc waits for a later night, named; one whose
  change no merge request judged is not judged on that change meanwhile.

## Amendment (2026-10-18): `checked` after parts, measured, off

Decided by the maintainer on
[#482](https://github.com/JN0V/workline/issues/482), option 1: measure,
then move `checked` when the engine sees every line stating something
supported — no passage contradicted, partial or uncovered, no claim
dropped, no note — adopted only if five evaluation runs in parts hold
[ADR-0014](0014-checked-is-earned-by-what-was-read.md#4-acceptance-written-before-it-is-measured)
step 4's bars.

- **Built, behind `parts-move-checked`**, off by default. The engine
  moves `checked`, never the fix, when:
  - the parts found nothing wrong, in conflict or to be read together; a
    partial another part's support settles is settled, as the merge
    above reads it;
  - no claim was dropped (a part's note is one), no line is uncovered,
    no source is a doc, which no part reads;
  - each word of every line stating something is quoted, where it
    stands, in a part's support from the code. Headings, fences, derived
    blocks, HTML comments and link definitions state nothing; a code
    example does. A quote of a few words vouches for those words, so a
    part is asked to quote a supported passage whole.
- **Then** `checked` and `judged-in-parts` name that commit, `verified`
  the documentalist; the doc leaves the list for a person. Otherwise the
  finding says what kept it ("no part supports line 14").
- **Measured** on 2026-10-18, engine 38fac2d1, Sonnet 5.5:
  `finds-planted-defects-in-parts` with a clean control,
  `docs/tech/defaults.md` (three lines stating something, one source,
  one part), beside the doc with four defects planted:

  | Run | Score | Tokens in / out | The control | Planted falsehoods |
  |---|---|---|---|---|
  | 1 | 19/19 | 29,074 / 5,668 | `checked` moved | none vouched for |
  | 2 | 17/19 | 28,763 / 4,551 | stays: a claim dropped | none vouched for; one left, the doc unvouched |
  | 3 | 19/19 | 29,542 / 6,038 | `checked` moved | none vouched for |
  | 4 | 18/19 | 30,361 / 5,901 | stays: a fix asked, nothing changed | none vouched for |
  | 5 | 19/19 | 29,700 / 5,297 | `checked` moved | none vouched for |

  147,440 tokens in, the five together: 4.9k a call, as on 10-17 (24.3k
  for five calls, one doc). Run 2's row says `+changes`: docs edited
  while it ran, not the engine.
- **The bars**: no `checked` over a planted falsehood, held in 5 runs of
  5; the control ends `checked` in every run, not held: 3 of 5. So it
  stays off.
- **Why the control stayed**:
  - run 2: a part's claim was dropped, its quote not at the lines it
    cited; the finding did not say which. `claims-dropped` now names the
    part, the quote and where it was looked for.
  - run 4: a part found the control wanting, and the fix changed
    nothing. Its last sentence, "A field the configuration sets is kept
    as it is", is not exact: a field set to zero gets its default. It is
    rewritten for the next measure.
- **Next**: five runs again on the control rewritten, about 150k tokens;
  on by default if the bars hold.
- **The price, once on**: a doc fixed in parts still waits for a person;
  a line no code can support (a link, a word about the project) keeps
  the doc for a person; the weekly sample cannot read such a doc whole,
  so it lists it for a person
  ([ADR-0014, amended](0014-checked-is-earned-by-what-was-read.md#amendment-2026-10-18-checked-earned-in-parts)).
