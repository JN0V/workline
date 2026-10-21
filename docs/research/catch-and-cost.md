# Catch and cost — what the line's checks catch, and what they cost (2026-10-21)

**Question.** workline runs its own line on its own work. Four weeks
in, what does each check catch, what does it cost a product pull request
in tokens, minutes and rounds, and did escapes fall as checks came in?
[ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md)
placed each act by the wait alone.

**Verdict.**

- **Escapes did not fall**: 0.42, 0.52, 0.67, 0.61 defects found after
  merge per product PR (W40 to W43); 0.21, 0.30, 0.28, 0.21 leaving out
  those the line found running on itself.
- **A product PR costs about four times more**: 16.8 M builder tokens
  and 21 machine-minutes (S1), 64.8 M and 92 min (S4); 2.08 extra rounds
  a PR, 1.23 of them caused by the line.
- **Best buy**: Important correctness and edge-cases findings, 94 of 104
  acted on, for 0.30 rounds a PR. **Worst**: the release waiting on
  judged docs, 130 red heads and 61 docs PRs for one truth fix.
- **Upkeep is 42% of merged PRs** in W43, from 9% in W40; three circles
  keep coming back.
- **79% of a product PR's tokens is the build run itself**: better
  placement leaves an issue at about ×3.5 of S1's tokens.

Measured on 2026-10-21, read-only, on this repository from its first
commit (2026-09-23) to `a317e3fa`. What workline decides from it:
[ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md).

## Method and its limits

- **Data**: every PR by GraphQL (403, 383 merged) with commits, check
  rollups, comments, review threads; 205 issues; 6,412 Actions runs;
  main's 654 first-parent commits and 2,132 PR commits; the builder
  agents' transcripts (22 sessions, 450 agents: each call's usage, each
  tool call's time); CI agent tokens from 6 product PRs a period.
- **Periods**: S0 09-23 → 09-29, direct pushes, hooks and tests; S1
  09-30 → 10-04, PRs, docs judged on them, the release held on docs; S2
  10-05 → 10-12, the reviewer; S3 10-13 → 10-18, the merge policy,
  strict up to date, correctness blocking; S4 10-19 → 10-21, ADR-0050,
  gates, the inspector. Catches by ISO week: W40 ≈ S1 … W43 (Mon–Wed) =
  S4.
- **Product PR**: feat, fix, refactor, perf, test. **Upkeep PR**: docs
  brought in line, stamps, counts regenerated, condensing, evaluation
  records, by a person or the App.
- **Processed tokens**: input, cache writes, cache reads (about 98%) and
  output of every call, by the orchestrating session, build and
  follow-up agents, over the product PRs merged.
- **Extra round**: a push after the first; its cause, first match: the
  App's docs commits, a merge of main, stamps, a reviewer block, a push
  touching an Important finding's file, a nit's, else the author's work.
- **Acted on**: a later commit touched the finding's file, or it is
  outdated. Against 142 replies it catches 82% of "fixed", 25% of
  "dismissed": right for Important findings, a quarter high for nits.
- **Escape**: a merged fix PR or a defect issue of merged work, coded by
  hand; its inducer by SZZ (blame of the lines the fix deletes). A floor:
  W43's are not all found, severity unknown.
- **Unseen or not comparable**: catches before a push (local tests, the
  hook's refusals); the work grew (Go 15k → 90k lines, cases 137 →
  1,224); S0 has no PRs (167 feat and fix commits ÷ 4 ≈ 42); S4 is
  three days; lead time holds the person's yes; a defect of the line
  found on its own PRs is both escape and upkeep; product PRs 225 by
  period, 215 by week (`ci`, `build` counted on the cost side).

## Cost per product PR, by period

| Period | Product PRs | Tokens (M) | per 100 code lines (M) | Machine-min | To the checks: tokens · min | Median open → merge (h) | Heads a PR (CI sample) |
|---|---|---|---|---|---|---|---|
| S0, direct | ~42 eq. | ~13.2 | 2.7 | ~16 | – | – | – |
| S1, PRs, docs judged | 54 | 16.8 | 4.0 | 21 | 3% · 9% (understated) | 0.2 | 1.0 |
| S2, reviewer | 68 | 44.5 | 6.8 | 48 | 18% · 33% | 0.8 | 3.0 |
| S3, merge policy, strict | 68 | 52.3 | 8.6 | 63 | 31% · 48% | 1.3 | 2.5 |
| S4, ADR-0050, gates | 35 | 64.8 | 11.4 | 92 | 24% · 44% | 0.7 | 4.5 |

- **Throughput held**: product PRs an active day about 6, 10.8, 8.5,
  13.6, 11.7. Builders in parallel absorbed the cost.
- **To the checks**: fix rounds, merge series, docs upkeep, CI waits and
  reads inside a build, the hooks (about 1.5 min a build, flat).
- **Biggest steps**: the reviewer (S1 → S2, follow-ups 0.07 → 0.9 a PR);
  strict up to date (merge series 0.18 → 0.89 a PR, S3 → S4; CI waits in
  a build 6.9 → 17 min, S2 → S4).
- **The line's own agents in CI**: 50–161k tokens in a PR, under 0.3% of
  the builder's. Docs upkeep: 41% of S4's merged PRs, 1.9 M tokens a PR.
- Sources: `analyze.py`, `cost_split.py`, `check_share.py`,
  `done_time.py`, `ci_sample.py`, `ci_tokens.py`.

## Escapes, rounds and upkeep, by week

| Week | Product PRs | Escapes a PR | without the line's own finds | Important acted a PR | Extra rounds a PR | Important · nit · docs · main merges | Upkeep PRs | + releases | + self fixes |
|---|---|---|---|---|---|---|---|---|---|
| W40 | 53 | 0.42 | 0.21 | 0.02 | 0.77 | 0.02 · 0.02 · 0.38 · 0.13 | 9% | 19% | 20% |
| W41 | 56 | 0.52 | 0.30 | 0.79 | 1.84 | 0.48 · 0.62 · 0.14 · 0.02 | 15% | 29% | 30% |
| W42 | 72 | 0.67 | 0.28 | 0.61 | 2.97 | 0.32 · 0.47 · 0.49 · 0.97 | 18% | 28% | 36% |
| W43 | 33 | 0.61 | 0.21 | 0.61 | 2.70 | 0.42 · 0.70 · 0.18 · 0.76 | 42% | 47% | 61% |

- **119 defects escaped**: found by the line on workline itself 64, the
  reviewer outside a change 14, trials 13, measures and evaluations 13,
  while developing 10, real use 5.
- **SZZ**: an inducer for 67 of 79 fix PRs, 36 of them past the
  reviewer; median age 5.8 days (52 dated).
- **Shipped before fixed**: of 61 fixes whose inducer was dated from the
  merge list, 21 (34%) merged before the Monday after it, 3 (5%) before
  the next release at today's cadence.
- **Tests**: 1 failure in 1,745 `test` runs; 427 of 427 green on main.
- **All weeks**: 447 extra rounds on 215 product PRs, 2.08 a PR: 265
  (1.23) caused by the line, 65 (0.30) answering an Important finding,
  117 the author's own.
- **Line-only commits on PR branches**: 10%, 17%, 22%, 18%; docs a
  product PR touches only to move `checked`: 7.8, 8.9, 13.8 (W41–W43).
- Sources: `escapes.py`, `szz.py`, `analyze_prs.py`, `touched.py`,
  `weekly.py`, `classify_lines.py`.

## Catch and cost per check, best first

| Check | Catch | Cost |
|---|---|---|
| Reviewer, Important correctness and edge-cases | 94 of 104 acted on; 29 of 29 replies "fixed" | about 65 rounds, 0.30 a PR |
| Gates, lint, no agent | 13 new errors on 3 PRs, fixed before merge | 4 red heads, 1 race defect |
| Committer hook | no subject over 72 since 09-23 but squash titles; refusals unrecorded | 0.52–0.58 s a commit; 1 defect |
| Reviewer nits | 619 shown, about half acted on (tests 30%, intent 1 of 7) | 93 rounds, 0.43 a PR |
| Docs suspect on PRs | 9 prose fixes on 8 PRs, 7 in W40 | 69 rounds, 54 stamp-only commits, 12 defects |
| Up to date with main, strict and follow | not measurable: no test red after a merge of main | 103 rounds (0.97 a PR in W42), 5 defects |
| The reviewer's machinery | none | 17 red heads with no finding, 12 record defects |
| Release waiting on judged docs | 1 truth fix ([#472]) in 39 releases | 130 red heads, 61 docs PRs (47 of 57 parsed stamps only), 12 PRs to unblock, 10 defects |

- **Ignored findings and later escapes**: an unacted tests nit sits on
  12 of 27 inducing product PRs (44%) and 66 of 107 others (62%), no
  signal; an unacted correctness Important on 3 of 27 (11%) against 3
  of 107 (3%), small numbers.
- **Tests lens, Important**: 12 of 30 acted on. **Edge-cases**: 55 of 57
  acted on, with no block.
- **Evaluations and measures found 13 of 119 escapes**: judgement
  defects no deterministic check sees.
- **Squash titles**: 81 of main's 654 subjects (12%) are over 72
  characters, all for GitHub's " (#N)"; no check reads a PR's title.
- **The documentalist turns a head red almost only on release PRs**:
  129 of the 135 runs it stopped.
- Sources: `analyze_prs.py`, `docs_catches.py`, `product_docs.py`,
  `parse_fail_logs.py`, `weekly.py`.

## The recurring circles

1. **The release held by docs**: 32 items, 12 issues and 20 PRs, in
   three weeks; version-pin parsing fixed 3 times in 2 days; 14 App docs
   PRs in 90 minutes for one release, all stamps or pins. Cause: 22
   releases in 14 days, each a docs gate.
2. **What the reviewer read**: 22 items, a new defect every two to three
   days; a push landing before the apply 6 times in 8 days, in the
   reviewer and the inspector. Each role keeps a hidden record its own
   apply job writes. The reviewer reads the first comment holding its
   marker, from any author, and writes the last
   ([record.go](../../internal/builtin/reviewer/record.go), `loadRecord`).
3. **The documentalist's own PRs and `checked` lines**: 30 items; one
   night's gardening opened 4 bugs ([#605] to [#608]); [#606], a false
   `verified:`, merged itself to main; the night starts 4.5 to 7 h late,
   racing the day's merges ([#607]).

- Merge, follow and accepted: 23 items, 5 defects.
- Source: `themes.py`.

## The build run, out of placement's reach

- **51.5 of S4's 64.8 M tokens a product PR (79%)** is the build run,
  its own CI waits and reads included. The rest: fix-round resumes
  7.3 M, merge series and docs upkeep 2.9 M, the orchestrator 3.1 M.
- **It made 72% of the growth** from S2 to S4 (36.8 → 51.5 M): 215 →
  287 tool calls a build, mostly reading and editing. A build's minutes,
  39.0 → 83.8: model 19.5 → 37.9, CI waits 6.9 → 17.0, `go test` 6.2 →
  8.8, scratch runs and tries 2.4 → 14.6, hooks 1.6 → 1.4. Its model
  calls grew 152 → 207 as the code grew six-fold.
- **Placement removes about 7 M and 23 min a PR**: docs upkeep, most
  merge series, stamp and nit rounds, the CI waits strict added. About
  58 M and 69 min stay: ×3.5 of S1's tokens, ×3.3 of its minutes.
- Sources: `cost_split.py`, `tool_time.py`, `agg_tool_time.py`,
  `tool_calls_share.py`, `bash_other.py`.

## Scripts

Kept with their data outside the repository; Python run with `-I`,
shell with `gh` and `jq`, read-only. In order: `fetch_prs.sh`,
`fetch_first_commit.sh`, `git_lines.py`, `parse_runs.py`,
`parse_usage.py`, `agent_prs.py`, `main_prs.py`, `agent_active.py`,
`tool_time.py`, `analyze.py`, `ci_sample.py`, `ci_tokens.py`,
`cost_split.py`, `agg_tool_time.py`, `tool_calls_share.py`,
`lines_split.py`, `done_time.py`, `check_share.py`, `tables.py`; then
`fetch_prs.py`, `fetch_fail_logs.sh`, `parse_fail_logs.py`,
`classify_lines.py`, `analyze_prs.py`, `touched.py`, `docs_catches.py`,
`product_docs.py`, `szz.py`, `escapes.py`, `weekly.py`, `themes.py`.
The shipped-before-fixed share, main's runs and the squash titles were
counted inline over the same data by a reviewing panel.

[#472]: https://github.com/JN0V/workline/pull/472
[#605]: https://github.com/JN0V/workline/issues/605
[#606]: https://github.com/JN0V/workline/issues/606
[#607]: https://github.com/JN0V/workline/issues/607
[#608]: https://github.com/JN0V/workline/issues/608
