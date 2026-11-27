# Diagnosing the reviewer — measured (2026-11-26/27)

**Question.** The evaluation of 2026-11-25 gave the reviewer 5 of 10.
Which change makes it state more of the points a maintainer sent pull
requests back for, without more false alarms on clean changes nor lost
catches? The method is on [diagnosing-a-role.md](diagnosing-a-role.md).

**Verdict.**

- **Two structures lead, both reading the repository on Opus**: Sonnet
  lenses then an Opus verifier (`hso`, 24.2 points of 34 a run, 6 runs)
  and one free Opus agent with merge standards (`free2`, 25.0, 6 runs),
  against production's 17.3 (3 runs). Both beyond the noise (p = 0.012);
  level with each other (p = 0.39).
- **The gain is mostly repository reading**: about 5 more a run on the 6
  points whose evidence lies outside the diff. On the 6 ambiguous points
  the merge standards aim at, every arm says 2.0 to 2.5 a run.
- **`free2` is cheaper and faster, and level on weak tests**: 73 % of
  those points against `hso`'s 71 % and production's 70 %. The 36 to
  45 % of round 1 was the free agent without merge standards. The bench
  does not settle the choice between the two leaders.
- **Shipped**: two fixes in v0.92.0
  ([#2287](https://github.com/JN0V/workline/pull/2287),
  [#2288](https://github.com/JN0V/workline/pull/2288)); `hso`'s verifier
  in shadow, posting nothing, in v0.93.0, as the skeptic's review
  advised from round 1's weak-test gap, which `free2` does not show
  ([#2312](https://github.com/JN0V/workline/pull/2312), turned on for
  workline by [#2319](https://github.com/JN0V/workline/pull/2319)).
- **Unproven**: no held-out pull request; every merge standard was
  written from these cases; the shadow's results are not read yet.

## The set and the scorers

- **24 replay cases**
  ([tests/evaluation/replay/](../../tests/evaluation/replay/)): 15 misses
  holding 34 planted points, 5 kept catches, 4 clean changes. One clean
  case, `clean-tg-11`, holds a real defect (an untested scenario).
- **By place** (the official score): any Important on a planted line.
  Over round 1, 61 of 310 points were earned by place only and 79 true
  statements scored nothing.
- **Readers and a blind judge** (`semscore`, Opus, one call a case run):
  97 % agreement with 24 independent readers on 699 point-runs.
- **The meaning scorer**
  ([#2282](https://github.com/JN0V/workline/pull/2282), Opus medium,
  blind, each item twice): 30 of 30 hand-checked items; agrees with itself
  on 99 % of 1 202 items. Round 1 counts 28 reachable points; round 2
  counts said points of 34.
- **Its independence**: Sonnet as the scorer on 2 runs each of `hso` and
  `free2` gave the same verdict on 153 of 156 point-runs (98 %).

## Round 1: structure and model, 3 runs an arm

Six arms already run on 2026-11-26, scored by meaning with
[#2282](https://github.com/JN0V/workline/pull/2282) (of 28). Two
values in a cell are two commits or two prompt variants of the arm.

| Arm | Said a run | Out of diff | Weak test | Kept of 5 a run | Tokens a run |
|---|---|---|---|---|---|
| Lenses, Sonnet | 17, 16, 15 · 15, 17, 15 | 0 % | 73 · 64 % | 4, 3, 3 · 3, 3, 2 | 1 070k · 1 105k |
| Lenses, Opus | 18, 19, 21 · 19, 17, 18 | 17 · 11 % | 85 · 73 % | 4, 4, 4 · 3, 5, 4 | 1 553k · 1 295k |
| Lenses reading, Sonnet ([#2273](https://github.com/JN0V/workline/pull/2273)) | 16, 14, 16 · 16, 18, 14 | 28 % | 55 · 58 % | 2, 3, 2 · 3, 1, 3 | 1 706k · 1 769k |
| Lenses reading, Opus | 16, 21, 20 | 56 % | 61 % | 2, 2, 1 | 4 272k |
| Free agent, Sonnet ([#2279](https://github.com/JN0V/workline/pull/2279)) | 16, 18, 16 | 67 % | 36 % | 4, 3, 4 | 1 173k |
| Free agent, Opus | 18, 21, 18 | 83 % | 45 % | 4, 4, 4 | 1 957k |

- At the same model the structures are level; Opus says 2 to 3.7 more a
  run in each. Reading costs kept catches to the lenses.
- **Causes**, by the readers, of 102 point-runs an arm: had the code and
  did not say it, Sonnet lenses 22, Opus lenses 8 (free agent 33 and
  24); never had the code, 10 to 11 for the lenses without reads, all on
  4 out-of-diff points; a severity rule 2 to 13; the role's own judge 0
  to 7; format 0 to 4, read arms only; unreachable 0.
- **Clean**: 23 of the 36 false Importants come from one intent-lens rule
  (a behaviour kept as before has no test), 21 of them on `clean-tg-11`.
  The free arms show none in 6 runs.

## Round 2: preregistered, at one commit

Written at 14:27Z, before any strategy ran
(commit 7a18dbca4 evaluated). Seven arms, so a chance winner was
expected; 6 runs an arm planned.

| Arm | What it changed | Prediction, of 102 a triple |
|---|---|---|
| `base` | Sonnet lenses, Sonnet judge | ~53 |
| `opus` | Opus lenses (judge Sonnet, as production picks) | ~63, more clean alarms |
| `hso` | Sonnet lenses, then one Opus verifier reading the checkout, with merge standards; no engine judge | 68–72, clean 0–1 of 12 |
| `hss` | the same, verifier on Sonnet | 58–62 |
| `free2` | the free Opus agent with merge standards | 63–68 |
| `prompt` | lenses with 5 prompt changes | 56–58 |
| `high` | Sonnet lenses at effort high | 53–57 |

Amendments, each before the runs it touches:

- **14:44Z**: production judges Sonnet lenses with **Opus**
  (`judgemark.Pick`); every bench had forced Sonnet. New arm `prod`, the
  production baseline every strategy is compared with.
- **14:46Z**, the owner's question, must the judge be another model than
  the lenses: a 2 × 2 of lens and judge model (`opusopus` added), and
  "no judge" read from the same calls.
- **19:47Z and 20:15Z**: cuts (below); new arm `fv`, the free Opus agent
  then a Sonnet verifier, 3 runs; predicted as `free2` or +1 a run.

## Round 2 results

Points said of 34, per run. Kept of 5. Clean: cases with a false
Important, of 4.

| Arm | Scorer | Said per run | Mean | Kept | Clean |
|---|---|---|---|---|---|
| `base` | semscore | 19, 17, 17, 17, 19, 17 | 17.7 | 3, 3, 3, 3, 3, 2 | 1, 1, 2, 2, 1, 2 (place) |
| `opus` | semscore | 20, 19, 20, 19, 18 | 19.2 | 3, 5, 4, 4, 4 | 3, 2, 2, 2, 2 (place) |
| `opusopus` | semscore | 18, 18 (third unscored) | — | 5, 4 | 1, 1 (place) |
| `prod` | #2282 | 16, 17, 19 | 17.3 | 3, 3, 3 | 0 by meaning |
| `hso` | #2282 | 26, 23, 24, 23, 25, 24 | 24.2 | 3, 2, 3, 3, 3, 4 | 0 |
| `free2` | #2282 | 24, 27, 25, 25, 23, 26 | 25.0 | 4, 3, 3, 4, 3, 4 | 0 |
| `fv` | #2282 | 22, 20, 24 | 22.0 | 3, 4, 4 | 0 |

- **Tests** (Welch on per-run totals, exact permutation p): `hso` vs
  `prod` +6.8 (p 0.012); `free2` vs `prod` +7.7 (p 0.012); `free2` vs
  `hso` +0.8 (p 0.39); `fv` vs `free2` −3.0 (p 0.06).
- **Predictions**: `base` 53 a triple, as said; `hso` 73 and 72, at the
  top of its range; `free2` 76 and 74, above it; `opus` 59, under its ~63.
- **Paired stages**: `hso`'s lenses alone, same calls, said 18, 17, 15
  (semscore) against 26, 22, 24 after the verifier; by place the verifier
  adds 4 to 7 in 6 runs of 6. `fv`'s Sonnet verifier lowers its own first
  stage by place (21, 19, 21 against 22, 23, 23): a verifier can drop
  true findings.
- **Weak or missing test points said**: `prod` 70 %, `hso` 71 %,
  `free2` 73 %, `fv` 70 %.
- **Clean**: `prod` shows 1, 2, 1 Importants a run, all on `clean-tg-11`,
  judged true defects or a nit. `hso` and `free2` show none.
- **Tokens a case run**, fresh / cache included, from the kept calls
  after 3 runs: `prod` 44k / 47k, `opus` 37k / 53k, `opusopus` 55k / 59k,
  `hso` 47k / 144k, `free2` 16k / 93k. The skeptic's count on other runs:
  `prod` 31k and `hso` 53k to 65k fresh. Fresh depends on cache warmth.
- **Wall a case**: `prod` 45 to 50 s, `hso` 68 to 80 s, `free2` 22 to 28 s.

## The judge question

Lens-written Importants on the change that state no point, share the
judge refused: Sonnet lenses judged by Sonnet (6 runs) 14 %, Opus lenses
by Sonnet (4 runs) 14 %, Sonnet lenses by Opus 1 of 8 (one run).

- No self-preference seen in those shares. Only `prod`'s first run and
  no `opusopus` run went through this analysis: the question stays open.
- **A same-model side effect, found**: Sonnet judging Sonnet dropped 16 of
  16 Importants outside the change, 3 of them true (a build break in
  dc-273). Fixed by [#2287](https://github.com/JN0V/workline/pull/2287).

## Cuts, and why

- **19:47Z, the owner: stop what is surely worst (tokens).** `opus`
  stopped at 5 runs: 18 to 20, no overlap with the leaders, 2 to 3 false
  clean alarms a run against 0. `prod` stopped at 3 (each `hso` run holds
  a production engine stage on its own lens calls); `opusopus` at 3.
- **20:15Z, the owner: two leaders to 6 runs.** `hss`, `prompt` and
  `high` were dropped **untested**: not shown worse. They would have said
  whether a Sonnet verifier gives the gain at less cost, whether the
  lens prompt mends the clean alarms with no new code, and the model
  lever at Sonnet's price.

## What shipped, measured after

- **[#2288](https://github.com/JN0V/workline/pull/2288)**, the prompt
  part for clean alarms: a kept behaviour left as it was owes no test. 3
  runs each against `prod` at the same commit, meaning scorer:

  | | `prod` | #2288 |
  |---|---|---|
  | Clean Importants a run | 1, 2, 1 | 0, 1, 0 |
  | of them false alarms, by meaning | 0, 0, 0 | 0, 1, 0 |
  | Said of 28, per run (mean) | 14, 15, 17 (15.3) | 14, 15, 14 (14.3) |
  | Weak or missing test points said | 70 % | 58 % |
  | Kept of 5, per run | 3, 3, 3 | 4, 4, 3 |

  The Importants it removes, the scorer calls true defects or a nit (the
  owner calls them a norm the issue does not ask); by meaning it adds one
  false alarm and says fewer weak-test points. Within the noise, 3 runs.

- **[#2287](https://github.com/JN0V/workline/pull/2287)**: conformance
  only. Production's judge is the other model, so the bench cannot show
  it; the forced same-model arms dropped 16 outside findings in 6 runs.
- **[#2312](https://github.com/JN0V/workline/pull/2312)**: `hso`'s
  verifier after the post, recorded, never blocking, behind a setting
  (Part of [#2308](https://github.com/JN0V/workline/issues/2308)); the
  skeptic's verdict was "ship in shadow only". Another build,
  [#2305](https://github.com/JN0V/workline/pull/2305), is open.

## Held out, and unproven

- **No held-out case**: the merge standards map case by case to the
  misses they were written from. Every gain here is an upper bound.
- **The shadow is the check**: on real pull requests, the verifier should
  state the owner's send-back points production missed, and not block
  what the owner merges as it is.
- **Kept catches**: `kept-wl-2110` is said in 1 run of 18 in round 1,
  whatever the arm.
- **The clean set is 4 cases**, one of them defective.
- **The scorer's hand check**, 30 of 30, was one reader of the judge's
  model family.
