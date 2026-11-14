# Models and efforts, measured (2026-11-14)

**Question.** Which Claude model and effort gives which result, at what
token volume, for the developer, the tester, the reviewer's judge and an
independent review; and is Sonnet really the cheaper one?
The grid behind the question: [the model grid](../spec/model-grid.md),
[ADR-0004](../adr/0004-follow-model-aliases-and-measure.md).

**Verdict.**

- **Sonnet is not cheaper in tokens.** Per call it reads as much as Opus,
  often more (the developer: 49k in against Opus's 40k), and fails more:
  per success it took 99k tokens on the developer's cases, Opus 40k. It is
  cheaper only by the price of a token, about half of Opus's per call; per
  success on the developer, the same price (0.085 USD against 0.086).
- **The developer on Sonnet edits out of its scope**: 0 of 3 on
  `leaves-a-tempting-edit-out-of-scope` (medium twice, high once), every
  other model 2 of 2. Opus at medium: 4 of 4, the fewest tokens of all.
- **The tester**: the substance is right on most models; only Fable also
  names each scenario in its test, as the case checks (14 of 14 full).
  Sonnet at medium slipped once on the answer's format; at high, none.
- **The judge**: Sonnet and Opus tie (28 and 27 of 44, reading code);
  Sonnet half the price, but its "no" is quoted less often. No change is
  backed by this.
- **An independent review**: Sonnet xhigh, Opus high and Fable high each
  found 5 of 5 planted defects. Opus high wrote the fewest tokens (9.3k
  out, against 13.5k and 17.9k) and was the fastest.
- **Haiku 5.5** passed every developer case and the tester's substance, at
  a fraction of the price, but reads more (58k in against 40k for Opus).
  Measured on tiny cases only.
- **Spent**: 2.66M tokens in and 0.18M out, 259 rows, about 15 minutes
  of wall time.

## What the earlier measures said

[Measured before](models-measured-before.md): 765 rows of the
evaluation's history, 2026-09-25 to 2026-11-09.

- **Condensing a doc**: Opus right 12 in 12 at 23k tokens per success;
  Sonnet 5 4 in 5 at 43k; Haiku 4.5 never.
- **Haiku 4.5 thinking on a commit subject**: 4.6k tokens out; Haiku 5.5
  writes 0.6k and earns every point.
- **The developer on Sonnet 5.5**: 7 of 15 runs with every point.

## Today's results

All on the engine at d99ef5adb, one model forced for the role
(`WORKLINE_EVAL=claude:<model>@<effort>`). Tokens in include cache reads.
"USD" is the API-price equivalent Claude Code reports; a subscription's
limits weigh models by a rule not published, so read it as a proxy.

| Role (cases) | Model, effort | Runs | Every point | Substance right | Tokens in / out a run | USD a run | Seconds | Tokens per success |
|---|---|---|---|---|---|---|---|---|
| developer (3) | Opus 5.5, medium | 4 | 4 | 4 | 40k / 0.5k | 0.086 | 9 | 40k |
| developer (3) | Fable 5.2, medium | 4 | 4 | 4 | 44k / 0.6k | 0.199 | 9 | 45k |
| developer (3) | Haiku 5.5, medium | 4 | 4 | 4 | 58k / 1.5k | 0.004 | 11 | 60k |
| developer (3) | Sonnet 5.5, medium | 4 | 2 | 2 | 49k / 0.5k | 0.042 | 7 | 99k |
| developer (3) | Sonnet 5.5, high | 3 | 2 | 2 | 62k / 0.6k | 0.046 | 9 | 94k |
| tester (7) | Fable 5.2, medium | 14 | 14 | 14 | 7.8k / 0.6k | 0.061 | 8 | 8.4k |
| tester (7) | Sonnet 5.5, high | 7 | 5 | 7 | 6.8k / 0.5k | 0.008 | 6 | 7.3k |
| tester (7) | Haiku 5.5, medium | 14 | 8 | 14 | 5.8k / 1.1k | 0.001 | 7 | 6.9k |
| tester (7) | Opus 5.5, medium | 14 | 9 | 13 | 7.1k / 0.5k | 0.027 | 7 | 8.2k |
| tester (7) | Sonnet 5.5, medium | 14 | 6 | 11 | 6.2k / 0.4k | 0.010 | 5 | 8.4k |
| judge, reading (22) | Sonnet 5.5, medium | 44 | 28 | — | 9.2k / 0.4k | 0.029 | 7 | 15k |
| judge, reading (22) | Opus 5.5, medium | 44 | 27 | — | 9.5k / 0.7k | 0.063 | 11 | 17k |
| judge, no reads (22) | Sonnet 5.5, medium | 22 | 17 | — | 4.2k / 0.3k | 0.020 | 4 | 5.8k |
| judge, no reads (22) | Fable 5.2, medium | 22 | 15 | — | 4.2k / 0.5k | 0.088 | 7 | 6.9k |
| judge, no reads (22) | Opus 5.5, medium | 22 | 14 | — | 4.2k / 0.4k | 0.041 | 6 | 7.2k |
| review (6) | Opus 5.5, high | 6 | 3 | 6 | 10.8k / 1.9k | 0.083 | 25 | 12.7k |
| review (6) | Sonnet 5.5, xhigh | 6 | 6 | 6 | 11.1k / 3.0k | 0.097 | 32 | 14.1k |
| review (6) | Fable 5.2, high | 6 | 6 | 6 | 8.8k / 3.6k | 0.214 | 43 | 12.3k |

"Substance right" leaves out two checks the models disagree with by
design: the tester's case wants the scenario's name in the test file
(the role's instruction only says to keep it), and a review case counts
a true finding at a place nothing was planted as a miss. "Tokens per
success" divides by the substance right where it differs. A review row's
tokens and USD include the judges, which the reviewer runs on another
model than the finder (Opus for Sonnet and Fable, Sonnet for Opus); the
finder's own calls are told below.

### The developer

- **The scope case decides.** The case's issue names `lib.sh`; `money.sh` has the
  same bug. Sonnet fixed `money.sh` too in 3 runs of 3; the engine applied
  it and listed it "beyond scope" for the reviewer. Opus, Haiku and Fable
  left it, 2 of 2 each. Sonnet's history on the same case: 1 of 4.
- **Sonnet reads more**: 49k to 62k tokens in, Opus 40k, for the same
  fix; at high, 95k on the scope case.
- **No retry anywhere**: one call per case for every model.

### The tester

- **Every model got the substance right** but Sonnet at medium (11 of
  14: a rewrite given as a diff, refused, twice; one answer asked again)
  and Opus once (the curation case: no merge request opened).
- **Only Fable put the scenario's name in the test** (`Two by three`),
  which three cases grep for: 14 of 14 full, Opus 9, Haiku 8, Sonnet 6.
- **Tokens are flat**: 6.9k to 8.4k per success whatever the model; the
  price differs 60 times between Haiku and Fable.

### The reviewer's judge

The 22 smallest of the 40 measured verdicts
([the judge, measured](judge-measured.md)): 15 true, 7 false; 6 of the
true ones dropped by Opus on 2026-10-14.

- **Reading code, two runs each** (on by default): Sonnet 15 and 13 of
  22, Opus 13 and 14. Both: every yes right.
- **Sonnet drops fewer true findings** (0 and 1 of 15; Opus 2 and 3) and
  lets more false ones through, shown not verified (4 and 6 of 7; Opus 3
  and 2): its "no" quoted text not found again in 5 and 7 answers, Opus 1.
- **Opus asked for a `grep:` read** in a form the engine does not read,
  then gave no answer: 2 and 3 of 22.
- **Without reads, one run**: Sonnet 17, Fable 15, Opus 14 of 22; Opus
  gave the same verdict as on 2026-10-14 in 21 of 22.

### An independent review

Six reviewer cases on the `shop` fixture, every lens in one call: five
planted defects (correctness twice, edge cases, tests, intent) and a
clean change. One run per model.

- **Every model found all five**, each verified, and kept the clean change
  clean.
- **Opus high raised three more important findings** on nothing planted:
  no test for an unreadable coupon, the refusal path of Restock untested,
  one bound check not enough. All three read true.
- **Output tokens**: Opus 9.3k, Sonnet xhigh 13.5k, Fable 17.9k for the
  six finder calls; input equal. Seconds a case: 25, 32, 43.

## Recommendations

| Role or use | Today | Recommended | Why |
|---|---|---|---|
| developer | standard tier, medium (Sonnet) | Opus 5.5, medium (`frontier`) | 4 of 4, fewest tokens per success; Sonnet 0 of 3 on scope |
| tester | standard tier, medium (Sonnet) | Sonnet 5.5, high; Opus 5.5, medium if tokens rule | Sonnet medium slips on format; tokens flat across models |
| reviewer's judge | Opus judging a Sonnet finder | unchanged | a tie at 44 verdicts; Sonnet's unquoted "no" adds noise |
| independent review agent | Sonnet xhigh | Opus 5.5, high | same catch, a third fewer tokens out, faster |
| Haiku 5.5 | light tier (committer) | measure on a real-size change before any wider use | right on tiny cases, at a fraction of the price |
| Fable 5.2 | not used | none yet | as right as Opus here, 2 to 5 times its price |

Before a grid change, a run on a real-size issue: these cases are a few
lines each.

Applying one changes the role's `role.yaml`, every project's default: a
project cannot pin a role's model or effort yet
([the model grid](../spec/model-grid.md#resolving-a-model-at-run-time)).

## Found on the way

- **The engine refuses `xhigh`**: `claude:<model>@xhigh` is rejected
  (internal/agent/agent.go, `claudeEffort`), though Claude Code 2.1.325
  takes it. Measured here with a local patch, uncommitted
  ([#1533](https://github.com/JN0V/workline/issues/1533)).
- **The tester's cases grep the scenario's name** in the test, which the
  role's instruction no longer asks for since the agent declares
  `covers`; 5 of the 14 case runs of Sonnet at medium lost points on it
  alone ([#1534](https://github.com/JN0V/workline/issues/1534)).
- **The scope case and the "beyond scope" list**: the engine now applies
  and lists a file beyond the Scope; the case still wants it untouched
  (same issue).
- **The judge's bare `grep:`**: Opus wrote `- grep: "…"`, not nested in
  `read:`, and got no verdict
  ([#1535](https://github.com/JN0V/workline/issues/1535)).
- **The tester's tokens in have grown**: 2.2k a case on 2026-11-08, 6.2k
  today (Sonnet, medium), the same cases.

## How it was run

In a worktree at d99ef5adb, `xhigh` added to `claudeEffort` locally, each
run from a clean environment, results to a file outside the repository
(259 rows, not kept: this page holds their totals):

```sh
env -i HOME=$HOME PATH=$HOME/.local/go/bin:$HOME/.local/bin:/usr/bin:/bin \
  WORKLINE_EVAL_RESULTS=$RESULTS WORKLINE_EVAL=claude:claude-opus-5-5@medium \
  go test -count=1 -timeout 60m -parallel 4 -run 'TestEvaluation/developer/' ./tests/evaluation/
# the tester: -run 'TestEvaluation/tester/'
# the reviewer: -run 'TestEvaluation/reviewer/^(correctness-checkout|correctness-take|edge-cases-page|tests-rounding|intent-restock|clean-stock)'
# the judge, 22 items, with WORKLINE_EVAL_REPLAY_ROUNDS=2 for the arm that reads:
#   -run 'TestJudgeReplay$/items/^(136-|152-correctness-command|219-correctness|199-edge|219-edge|199-tests|152-correctness-github|144-|177-|158-|329-intent|157-|190-|185-|170-|307-|146-|238-|237-|137-edge|137-tests|133-)'
```

A cell ran only while the tokens spent stayed under 2.5M to 2.7M.

## Limits

- **Small samples**: 1 to 4 runs a developer cell, 1 a review, 2 a
  tester cell; one failure moves a rate by 25 to 50 points. Only the
  developer's scope case (0 of 3 against 6 of 6) is a clear gap.
- **Tiny cases**: a developer case is a one-line fix in a shell library;
  a review, a few lines of a 150-line shop. A real issue reads more and
  may separate the models where these do not.
- **The judge's subset**: the 22 smallest items; the 18 larger, mostly
  tests-lens findings, not asked.
- **One provider**: every model, finder, judge and grader, is Claude.
- **USD is a proxy**: how a subscription weighs a model's tokens is not
  published.
