# Models and efforts, measured before (2026-09-25 to 2026-11-09)

Part of [models and efforts, measured](models-measured.md): what the
evaluation's history said before the grid of 2026-11-14.

From tests/evaluation/results.tsv (765 rows, 2026-09-25 to 2026-11-09;
its history only reshaped columns, no row lost) and the measured pages.
Models then: Sonnet 5 and Haiku 4.5 until 2026-09-28; Sonnet 5.5 from
2026-09-30, Haiku 5.5 from 2026-10-11; Opus 5.5 throughout; Fable never.

| Role, task | Model, effort | Runs | Every point | Tokens in / out a run | Seconds |
|---|---|---|---|---|---|
| documentalist, condense | Opus 5.5, medium | 12 | 12 | 14k / 10k | 85 |
| documentalist, condense | Sonnet 5, medium | 5 | 4 | 17k / 15k | 116 |
| documentalist, condense | Sonnet 5 stepped up to Opus | 1 | 0 | 33k / 25k | 189 |
| documentalist, condense | Haiku 4.5, medium | 3 | 0 | 25k / 31k | 282 |
| committer, rewrite | Haiku 4.5, low | 35 | 24 | 7.4k / 4.6k | 45 |
| committer, rewrite | Haiku 4.5, medium | 12 | 11 | 8.3k / 5.7k | 47 |
| committer, rewrite | Haiku 5.5, low | 14 | 14 | 5.1k / 0.6k | 15 |
| reviewer, every lens | Sonnet 5.5 finding, Opus 5.5 judging | 94 | 75 | 22k / 2.2k | 28 |
| tester, 7 cases | Sonnet 5.5, medium | 52 | 47 | 1.9k / 0.5k | 7 |
| developer, 3 cases | Sonnet 5.5, medium | 15 | 7 | 34k / 0.4k | 8 |

The surprises, where the smaller model cost more:

- **Condensing a doc**: Sonnet 5 wrote 50% more tokens than Opus and was
  right 4 times in 5, Opus 12 in 12: 43k tokens per success against 23k.
  A Sonnet try stepped up to Opus cost 58k and still failed. Haiku 4.5
  wrote three times Opus's tokens, took three times as long, and never
  succeeded. Hence condensing on `frontier`
  ([the documentalist's tries](../../roles/documentalist/docs/tried.md)).
- **Adopting a repository** (2026-09-28): Sonnet's answer refused in six
  rounds of seven, then Opus's taken (same page).
- **Haiku 4.5 thinking on a commit subject**: 4.6k tokens out for a
  subject line, 69% full at low. Haiku 5.5 writes 0.6k and earns every
  point: the generation, not the tier, made it cheap.
- **A side call**: from an editor's session, Claude Code had Haiku re-read
  the whole question beside the model asked, doubling tokens in
  ([claude-session-env.md](claude-session-env.md)); fixed in the adapter.
