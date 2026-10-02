# ADR-0012: A doc is judged against its sources whole, and a fix that cannot vouch is kept

- **Status:** accepted; amended 2026-10-02 (the bound's name)
- **Date:** 2026-10-01
- **Amends:** ADR-0007 on what a suspect doc is judged against; the
  documentalist's policy on `checked`

## Context

A suspect doc was put before the agent with what changed in its sources
since `checked` — the diffs — and the sources whole only when the diffs did
not fit. Moving `checked` vouches for every sentence the doc keeps; the
policy says a sentence the agent cannot find is fixed, or the answer is a
note; and a fix that leaves `checked` alone is refused (`still-suspect`).

Measured on the evaluation (ADR-0009, "Measured"): with four defects planted
in one doc, Opus 5.5 saw them all, but given only the diffs, six times in
seven it would not move `checked` over sentences it could not see, and
answered a note: nothing fixed. Sonnet 5.5 fixed them and moved `checked`,
vouching for what it had not seen. On DomoticsCore (2026-09-29), ten docs
went the same way: the agent found a version 1.4.1 where the code said
1.11.0, could not vouch for the rest, and the fix was lost.

The more careful model was right, and the line punished it.

## Decision

- **A suspect doc is judged against its sources as they are now**, with
  what changed beside them, whenever the sources fit the task
  (`staleSourceChars`, the same bound as a stale doc's; *now the setting
  `whole-chars`, see Amendment*). Only when they do
  not, the diffs alone, as before; and past that, in parts (ADR-0009).
- **A fix that does not move `checked` is kept.** It is applied, and the
  doc stays suspect, its finding saying it was fixed but not vouched for;
  the agent's note says what it could not confirm. A fix that moves
  `checked` must still name the commit given (`still-suspect` otherwise).
- The policy says so: fix what is wrong; move `checked` only when every
  sentence is found in the sources given; otherwise leave it, and say in a
  note what could not be confirmed.

## Consequences

- More tokens a suspect doc: its sources whole, up to 20,000 characters,
  where a diff was a few hundred. A doc judged once against its sources is
  not judged again until they change.
- No model is pushed to vouch for what it did not read; what it finds wrong
  is fixed either way.
- A doc fixed but not vouched for is a person's to confirm, as one judged in
  parts is.

## Measured

2026-10-01, `finds-planted-defects-whole` (four planted defects, twelve
checks), three runs each, after this decision:

| | before: diffs only | after: sources whole beside them |
|---|---|---|
| Opus 5.5 | 5/12 six runs in seven: a note, no fix | 12, 12, 12 (twice in two calls) |
| Sonnet 5.5 | 12, 12, 12 | 12, 12, 12 |
| tokens in a judgement | 8k | 11k |

## Amendment (2026-10-02)

The bound named `staleSourceChars` above is now the documentalist's setting
`whole-chars` (`wholeChars` in internal/builtin/documentalist/freshness.go),
20,000 characters by default, still shared by suspect and stale docs: a
project whose sources are larger raises it (docs/research/documentalist-genericity.md,
fixes 4 to 8). The decision is unchanged.
