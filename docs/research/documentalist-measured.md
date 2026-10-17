# The documentalist, measured (2026-10-17)

**Question.** Does each of the documentalist's evaluation cases hold
[ADR-0014](../adr/0014-checked-is-earned-by-what-was-read.md#4-acceptance-written-before-it-is-measured)'s
bar — five concordant runs: one engine commit, today's model, every run
earning every point — for criterion 3 of
[ADR-0036](../adr/0036-a-roles-status-is-earned-by-written-criteria.md)?

**Verdict.**

- **Met for all 14 cases**, five concordant full runs on one engine
  each: 12 in the afternoon at be99f444, after the guard of
  [#451](https://github.com/JN0V/workline/pull/451)
  ([below](#after-the-guard-2026-10-17-afternoon)); the last two in the
  evening at 2c532600, after [#460](https://github.com/JN0V/workline/pull/460)
  ([below](#the-last-two-cases-met-2026-10-17-evening)).
- **Not met in the morning: the measure stopped at its first
  confirmation.** `drifted-version-bump` moved `checked` over a planted
  falsehood in one run of three, its own note doubting that claim.
- **Spent**: 979,908 tokens in of the day's 1,000,000. The morning's
  255,597 came from an agent session, at twice the rate; the
  afternoon's 724,311 from outside one ([what it cost](#what-it-cost)).

## The morning's runs

At 760678b9: `main` after the two fixes to confirm,
[#441](https://github.com/JN0V/workline/pull/441) and
[#443](https://github.com/JN0V/workline/pull/443). `WORKLINE_EVAL=claude`,
every call answered by `claude-sonnet-5-5`, one call a run.

| Case | Runs | Scores | Every point, every run | Tokens in |
|---|---|---|---|---|
| drifted-gardening | 3 | 19/19, 19/19, 19/19 | yes, 3 of 3 (5 asked) | 123,331 |
| drifted-version-bump | 3 | 23/23, 23/23, 20/23 | **no** | 132,266 |
| the twelve others | 0 | — | not run | — |

The plan, written before the runs: confirm the two fixes (`drifted`
three runs each, `finds-planted-defects-in-parts` eight at 12/12), and
only then every case to five runs, stopping before the budget.

The command run, and how to read its rows:

```sh
WORKLINE_EVAL=claude go test -count=3 -timeout 60m -v -run 'TestEvaluation/documentalist/drifted' ./tests/evaluation/
awk -F'\t' 'NR==1 || $2 == "760678b9"' tests/evaluation/results.tsv > "$TMPDIR/runs.tsv" && go run ./tests/evaluation/summary "$TMPDIR/runs.tsv"
```

## What failed, and why

The third run of `drifted-version-bump` lost the three checks on
`docs/clock/status.md` together: `never-confirms` (it still says "33
tests"), `judged-is-head` and `checked-unchanged` (`checked` moved to the
head).

- **The ground truth.** The fixture's status.md has one source,
  `Clock.h`. Its "33 tests in Clock/test/test_clock.cpp" names a file no
  source shows: neither fixed nor vouched for, `judged`
  ([real shapes](../spec/conformance/evaluation.md#real-shapes)).
- **The answer**: five patches and a note. The note: "In status.md, the
  'tried on a real ESP32 board' claims and the '33 tests' count could not
  be checked against the sources; the 1.5.0 change only touched the
  version macro, so they are kept." Its patch on status.md moved
  `checked` all the same.
- **The rule it broke.** The documentalist's policy says both: what you
  cannot see backed is kept, not removed; and `checked` moves only when
  each sentence kept was found in the sources, else `judged` and a note.
  The agent kept the words, rightly, and vouched for them, wrongly: it
  took "kept" for "confirmed".
- **No guard in the engine saw it.** A `note` was free text tied to no
  doc (`case "note"` in internal/engine/engine.go): the engine could not
  tell the note doubted the doc it let the agent vouch for.
  `checked-unread` refuses only a doc whose sources were not given whole.
- **Not new.** The same three checks failed together once in five runs
  at d66eeea (2026-10-01, before ADR-0014's step 2); since step 2, 1 run
  of 10 (none at 7427ae8 nor in the weekly runs of 10-04 and 10-11).
- **The patch itself is not kept**: the run's folder goes with the test;
  its note and the checks are in the run's log.

Leads, for [#449](https://github.com/JN0V/workline/issues/449): the
engine keeping `checked` on a doc an answer's own note doubts, a
mechanical guard ([principle 7](../PRINCIPLES.md)), built by
[#451](https://github.com/JN0V/workline/pull/451); one note for the five
docs of one call blurring which one it doubts, unmeasured.

## The runs before this measure

[status.md](../../roles/documentalist/docs/status.md) said the other
cases had one run each: they had 5 to 24 rows each, over several engine
commits and models, so few held five on one engine and one model:

| Case | Rows | Engine commits | Most on one engine and model |
|---|---|---|---|
| condenses-the-role-contract | 24 | 13 | 3 (Sonnet 5, 2 full) |
| confirms-a-doc-still-true | 17 | 11 | 3 (Haiku, 1 full) |
| drifted-gardening | 17 | 5 | 5 (Sonnet 5.5, 4 full at 7427ae8) |
| drifted-version-bump | 17 | 5 | 5 (Sonnet 5.5, 5 full at 7427ae8) |
| finds-planted-defects-in-parts | 19 | 7 | 4 (Sonnet 5.5, 1 full) |
| finds-planted-defects-whole | 20 | 5 | 8 (Opus, 1 full) |
| fixes-a-doc-made-false | 16 | 11 | 3 (Haiku, 3 full) |
| merges-a-card-too-short | 6 | 4 | 3 (Sonnet 5, 3 full) |
| merges-a-repeated-passage | 7 | 5 | 3 (Sonnet 5, 3 full) |
| never-confirms-a-stale-doc-made-false | 10 | 5 | 6 (Sonnet 5, 6 full) |
| opens-an-issue-when-code-disagrees-with-a-spec | 7 | 5 | 3 (Sonnet 5, 3 full) |
| propagates-a-product-doc-at-release | 5 | 4 | 2 (Sonnet 5, 2 full) |
| splits-a-card-too-long | 8 | 4 | 3 (Opus then Sonnet 5, 3 full) |
| stays-on-the-task | 5 | 3 | 3 (Sonnet 5.5, 3 full) |

Rows before 2026-10-17; a commit marked `+changes` counted as its own
engine.

## What a run costs

- **Tokens in** are the whole input, cache included
  (internal/agent/claude.go).
- **Twice the weekly timer's.** The `drifted` cases took 41,109 to 44,090
  a run here, 19,785 to 21,589 in the weekly timer's runs (10-04, 10-11):
  the editor's session variables doubled them
  ([claude-session-env.md](claude-session-env.md); [what it cost](#what-it-cost)).
  At that rate the whole plan would have taken about 1.4M tokens in, not 1M.

## After the guard (2026-10-17, afternoon)

On the engine at be99f444: `main` after
[#451](https://github.com/JN0V/workline/pull/451), which keeps `checked`
on a doc the answer's own note doubts
([#449](https://github.com/JN0V/workline/issues/449)). Every run from
outside any agent session; Sonnet 5.5 answered, Opus 5.5 first on
condense and split (their `frontier` tier).

| Case | Runs | Every point, every run | Tokens in |
|---|---|---|---|
| drifted-version-bump | 5 | yes, 23/23 | 108,505 |
| finds-planted-defects-in-parts | 8 | yes, 12/12 | 195,252 |
| condenses-the-role-contract | 5 | **no**: 3 of 5; judge skipped | 107,890 |
| confirms-a-doc-still-true | 5 | yes, 4/4 | 21,575 |
| drifted-gardening | 5 | yes, 19/19 | 101,055 |
| finds-planted-defects-whole | 5 | yes, 12/12 | 36,700 |
| fixes-a-doc-made-false | 5 | yes, 5/5 | 21,330 |
| merges-a-card-too-short | 5 | yes, 5/5; one run stepped up to Opus | 21,757 |
| merges-a-repeated-passage | 5 | yes, 5/5 | 17,855 |
| never-confirms-a-stale-doc-made-false | 5 | yes, 4/4 | 22,170 |
| opens-an-issue-when-code-disagrees-with-a-spec | 5 | yes, 5/5 | 21,905 |
| propagates-a-product-doc-at-release | 5 | yes, 5/5 | 21,160 |
| splits-a-card-too-long | 5 | yes, 4/4 | 27,157 |
| stays-on-the-task | 0 | not run | — |

How it was run, in a worktree at be99f444, `WORKLINE_EVAL=claude`:

```sh
systemd-run --user go test -count=5 -timeout 60m -run 'TestEvaluation/documentalist/drifted-version-bump$' ./tests/evaluation/
# then one script under systemd-run --user, a case at a time, stopping if in-parts missed its bar or past 630,000 tokens in:
go test -count=8 -timeout 90m -run 'TestEvaluation/documentalist/finds-planted-defects-in-parts$' ./tests/evaluation/
go test -count=5 -timeout 90m -run 'TestEvaluation/documentalist/<case>$' ./tests/evaluation/   # each other case, by name
awk -F'\t' 'NR==1 || $2 == "be99f444"' tests/evaluation/results.tsv > "$TMPDIR/runs.tsv" && go run ./tests/evaluation/summary "$TMPDIR/runs.tsv"
```

### The last two cases, a first try

Left in the afternoon: one graded on its header, its judge unset
([#454](https://github.com/JN0V/workline/issues/454)); the other not run.
Five runs each at 1f81fa73, the judge on Sonnet:

- **`condenses-the-role-contract`: 4 of 5 full**; the fifth made 3 calls
  for a cap of 2, a condense asked again.
- **`stays-on-the-task`: 0 of 5**, the fix right, but 6 calls, about 103k
  tokens in: a merge request judged `README.md` in parts since
  [#381](https://github.com/JN0V/workline/pull/381)
  ([#458](https://github.com/JN0V/workline/issues/458)).

### What it cost

- **724,311 tokens in and 167,942 out**: 68 runs, 107 calls.
- **The doubling explained.** The same one call of
  `drifted-version-bump` took 44,090 tokens in from a session in the
  morning, 21,701 here: about the weekly timer's. A Claude Code session's
  variables reach the agent and double each call
  ([#452](https://github.com/JN0V/workline/issues/452); its fix,
  [#453](https://github.com/JN0V/workline/pull/453)).

## The last two cases, met (2026-10-17, evening)

At 2c532600: `main` after [#460](https://github.com/JN0V/workline/pull/460)
(a merge request leaves a doc too large for the gardening). From outside
any agent session, a run at a time; models as in the afternoon, the judge
on Sonnet.

- **`stays-on-the-task`: 5 of 5 at 4/4**, 1 call a run (6 at 1f81fa73),
  13,647 tokens in each.
- **`condenses-the-role-contract`: 5 of 5 at 5/5**, its judge asked; 2
  calls a run, the condense and the reader's, 21,578 tokens in each.
- **176,125 tokens in** for the 10 runs, under a guard of 200,000.

How it was run, in a worktree at 2c532600, from a clean environment:

```sh
# measure-guarded.sh (not in the repository): one `go test -count=1` per run, stopping before BUDGET tokens in
systemd-run --user --setenv=BUDGET=200000 --setenv=WORKLINE_EVAL=claude --setenv=WORKLINE_JUDGE=claude:sonnet \
  measure-guarded.sh 5 stays-on-the-task condenses-the-role-contract
```
