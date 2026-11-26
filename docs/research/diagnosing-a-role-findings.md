# Diagnosing a role — findings per role and strategies (2026-11-26)

> **INTERIM — to be completed when the benches end.** Numbers marked
> **INTERIM** come from 3 runs or fewer: inside the noise of
> [the method](diagnosing-a-role.md#4-enough-runs). The method, the
> pitfalls and the causes used below are on
> [diagnosing-a-role.md](diagnosing-a-role.md).

## Findings per role

### Reviewer

**INTERIM**: every number in this section comes from 3 runs an arm.

Two scorers are used below, both blind to the arm:

- **Readers and judge**: one independent reader per case classified each
  miss by cause; a blind Opus judge, one call a case run, said which
  shown findings state each point (97 % agreement with the readers).
  Used for the levers table and the clean counts.
- **Meaning scorer** ([#2282](https://github.com/JN0V/workline/pull/2282)):
  one yes-or-no question per point, Opus, each judgement twice. Used for
  the profiles and kept catches.

Levers on the 15 miss cases (34 points, 102 point-runs a 3-run set),
readers and judge:

| Lever | Size | Evidence |
|---|---|---|
| Model (Sonnet to Opus) | about +10 of 102, in every structure | saw but did not say falls from 22 to 8 (lenses), 33 to 24 (free agent) |
| Severity and decision rules | 2 to 13 | a true defect filed as a decision becomes a question for a person; the free agent on Opus calls weak tests true but not blocking |
| Repository reading | 8 to 11, only on the 4 out-of-diff points the lenses never had | did not look: 10 to 11 point-runs without reads, 0 to 3 with |
| The reviewer's own judge | 0 to 7 | it is not shown the author's Left out nor the title; its word search misses a prefix |
| Answer format | 0 to 4; 2 to 4 in read arms only | prose before the YAML; the re-ask is a new draw, not a repair |

- **Only the model's lever stands out of the noise**: 3 runs see a gap of
  11 to 14 of 102 at best, so the other ranks are inside it. The model's
  +10 is itself at the edge, and comes with twice the clean false alarms
  (4 to 8 of 12) and a commit effect (19, 19 against 22, 21, 22).
- **Out-of-diff points**: 6 of the 34 points have their evidence outside
  the diff; the lenses without reads lost 4 of them to not looking. The
  bench audit's re-score (regular expressions) gives the lenses 0 % of
  the 6; the meaning scorer gives 0 % on Sonnet and 11 to 17 % on Opus.
- **False alarms on clean changes**: 23 of 36 come from one intent-lens
  rule (a behaviour kept as before has no test), which the tests lens
  exempts; most fall on `clean-tg-11`, where they are mostly true (the
  case is not clean). The free agent shows none in 6 runs.
- **Profiles differ** under level totals (meaning scorer): the free
  agent finds out-of-diff points (67 to 83 %) and finds fewer weak tests
  (36 to 45 %); the lenses without reads the opposite (0 to 17 %, 64 to
  85 %).
- **Kept catches fall with reads, for the lenses**: 5 to 7 of 15 with
  reads against 8 to 12 without (meaning scorer). The free agent, which
  reads, keeps 11 to 12.
- **INTERIM**: the 6-run arms (production baseline with its Opus judge,
  lenses on Opus, a verifier agent after the lenses on Opus or Sonnet,
  the free agent with merge standards, a prompt variant, effort high)
  and the judge question below are running.
- **INTERIM, the judge question**: must the judge be another model than
  the lenses? Four arms, 6 runs each, Sonnet or Opus lenses × Sonnet or
  Opus judge, plus the same calls re-scored with no judge. Self-preference
  would show as a same-model judge refusing fewer of its own lens's empty
  findings.

### Developer

From the records of 78 pull requests, 2026-11-23 to 26, and 133 review
rounds:

- **19 of 43 rounds with no fix** were woken by a label-trace comment;
  v0.91.0 stops this: a comment within 2 minutes of the same account's
  accepted label asks no round
  ([#2220](https://github.com/JN0V/workline/pull/2220)). Not yet seen in
  production on every repository.
- **A person's send-back is answered**: 15 of 18 fixed green in one
  round. The 3 others needed a workflow file, a board measure, or hit a
  merge skew.
- **Of 21 send-backs**: 8 the agent's craft (did not do what the issue
  says 4, regression or side effect 2, code it calls not opened 2), 6
  partial work presented as done, 3 round mechanics, 4 the issue itself.
- **`Closes #n … proven` is the engine's**: in 6 of 6 partial-work
  send-backs, the agent's own text said what was missing.
- **Reviewer Importants**: 0 of 14 declined findings were true and
  fixable but argued away; 1 got an empty round.
- **INTERIM**: 4 arms queued at the same commit (a checklist before the
  last message, effort high, a second pass by a fresh call, Opus),
  scored by meaning per send-back point.

### Tester

From 24 replay cases (69 scenarios), one per issue whose test plan held
a bad scenario, 2026-11-23 to 25:

- **9 weak tests**: 5 find words in a source, 3 take another or easier
  path, 1 greps where the scenario asks about text.
- **24 scenarios kept as today with no test**: all 24 are true of the
  code; but in 10 (and partly 3 more) an existing test already guarded
  them, where the Draft said none did. The engine wrote that sentence
  after a grep for a test name; the agent was not asked to look.
- The one false kept claim of the 11-25 evaluation was a degenerate
  Validation (its numbers already hold), a product owner point.
- **INTERIM**: causes at v0.91.0 and strategies not yet measured.

### Product owner

- **INTERIM** (3 runs each, one case). The case scores 6 points a run:
  4 issues whose kind is graded, and 2 checks on the run itself. Counted
  as kinds right of the 12 graded over 3 runs, if the run checks passed:
  - full read of five issues the maintainer re-kinded, on `main`: Opus
    9, Sonnet 4; with repository reading
    ([#2277](https://github.com/JN0V/workline/pull/2277)): Sonnet 7,
    Opus 6;
  - kinds alone: 8 to 10 in every arm, level.
- One issue cannot be kinded from its body nor its repository: the bug
  needs a platform's status values not in the code. Unreachable.
- **INTERIM**: 5 strategies, 6 runs each, queued: Opus, effort high,
  repository reading, instructions on the line's bounds and own product
  questions, and reading plus instructions. Three held-out cases say
  whether the instruction generalises.

### Documentalist

**INTERIM**: 3 runs an arm or fewer, one case (7 wrong claims).

- **It holds back removals it cannot cite.** In the reading arm
  ([#2276](https://github.com/JN0V/workline/pull/2276)), every run found
  that nothing publishes two documented events. 2 runs of 3 left both
  for a person, once writing "I had no source line to cite for removing
  the claim"; one run fixed the first. The citation rule (every word
  removed is cited from a source) holds it.
- **Reading arm against `main`**, Sonnet: #2276 fixed 2 of 21 claims
  with 0 runs with a false edit, against 2 of 21 and 2 false-edit runs of
  3 for its own baseline, at 2.2× the tokens. A same-day bench of `main`
  on Sonnet fixed 6 of 21, with a false edit in 3 of 3 runs: the claims
  count is inside the noise. Opus on `main`: 6 of 14 claims, a false
  edit in 2 of 2 runs.
- **INTERIM**: 6 arms queued on 6 cases (9 targets a run): `main`,
  reading, a prompt on what to cite when nothing does what the doc says,
  the citation rule off for removals, effort high, then the best with
  Opus.

## Strategies tried or queued

| Strategy | Role | Status |
|---|---|---|
| Agents read the repository, pickers deleted ([ADR-0074](../adr/0074-agents-read-the-repository-the-engine-stops-preparing-their-input.md) step 2) | reviewer | closed ([#2273](https://github.com/JN0V/workline/pull/2273)): equal within noise, +55 % tokens |
| Same | documentalist | held ([#2276](https://github.com/JN0V/workline/pull/2276)): 0 false edits, same claims, ×2.2 tokens |
| Same | product owner | held ([#2277](https://github.com/JN0V/workline/pull/2277)): equal |
| ADR-0074's bar amended to not worse | all | open ([#2274](https://github.com/JN0V/workline/pull/2274)) |
| One free agent with read tools, no lens, no judge | reviewer | **INTERIM**, 3 runs ([#2279](https://github.com/JN0V/workline/pull/2279)): level by meaning at the same model; no false alarm on clean; finds fewer weak tests |
| Lenses on Opus | reviewer | most consistent signal, not proven; a 6-run set to confirm, then the maintainer's label |
| Verifier agent after the lenses (Opus or Sonnet) | reviewer | **INTERIM**, running |
| Prompt: decision rule, unannounced break, a test nothing runs, no test owed for kept behaviour | reviewer | **INTERIM**, running |
| Judge model: same as lenses or the other | reviewer | **INTERIM**, running |
| Effort high | reviewer, product owner, documentalist, developer | **INTERIM**, queued |
| Only an explicit request wakes a round | developer | merged in v0.91.0 ([#2220](https://github.com/JN0V/workline/pull/2220)) |
| Checklist, second pass, Opus | developer | **INTERIM**, queued |
| Instructions on the line's bounds and product questions | product owner | **INTERIM**, queued |
| Citation rule off for removals; prompt on what to cite | documentalist | **INTERIM**, queued |
| Score by meaning | bench | open ([#2282](https://github.com/JN0V/workline/pull/2282)) |
| Count a case once in the summary | bench | open ([#2280](https://github.com/JN0V/workline/pull/2280)) |
| Narrow what the bench's commit covers | bench | not built (the bench audit's fix 8) |
| Baseline rows of the replay set | bench | open ([#2271](https://github.com/JN0V/workline/pull/2271)) |
| Dead code removal | engine | merged ([#2272](https://github.com/JN0V/workline/pull/2272)): 122 lines |

## To be completed when the benches end

- The 6-run tables per role, with each arm's commit, judge and runs.
- Which predictions held.
- The shadow check of each winner on new real changes.
- The tester's causes at v0.91.0 and its strategies.
