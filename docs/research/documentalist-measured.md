# The documentalist, measured (2026-10-17)

**Question.** Does each of the documentalist's evaluation cases hold
[ADR-0014](../adr/0014-checked-is-earned-by-what-was-read.md#4-acceptance-written-before-it-is-measured)'s
bar — five concordant runs: one engine commit, today's model, every run
earning every point — for criterion 3 of
[ADR-0036](../adr/0036-a-roles-status-is-earned-by-written-criteria.md)?

**Verdict.**

- **Met for 12 of 14 cases in the afternoon**, after the guard of
  [#451](https://github.com/JN0V/workline/pull/451) ([below](#after-the-guard-2026-10-17-afternoon)):
  `condenses-the-role-contract` and `stays-on-the-task` not measured.
- **Not met in the morning: the measure stopped at its first
  confirmation.** `drifted-version-bump` moved `checked` over a planted
  falsehood in one run of three, its own note saying it could not check
  that claim. `drifted-gardening`: three of three earning every point.
  The twelve other cases were not run.
- **Spent**: 979,908 tokens in of the day's 1,000,000. The morning's
  255,597 came from an agent session, at twice the rate; the
  afternoon's 724,311 from outside one ([what it cost](#what-it-cost)).

## The morning's runs

On the engine at 760678b9: `main` after
[#441](https://github.com/JN0V/workline/pull/441) (a drifted case's issue
kept on the local forge) and
[#443](https://github.com/JN0V/workline/pull/443) (a part disputes a
defect only on the same words). `WORKLINE_EVAL=claude`, every call
answered by `claude-sonnet-5-5`, one call a run.

| Case | Runs | Scores | Every point, every run | Tokens in |
|---|---|---|---|---|
| drifted-gardening | 3 | 19/19, 19/19, 19/19 | yes, 3 of 3 (5 asked) | 123,331 |
| drifted-version-bump | 3 | 23/23, 23/23, 20/23 | **no** | 132,266 |
| finds-planted-defects-in-parts | 0 | — | not run | — |
| the eleven others | 0 | — | not run | — |

The plan, written before the runs:

1. Confirm the two fixes: the `drifted` cases three runs each, every run
   earning every point; `finds-planted-defects-in-parts` eight runs, 12/12
   each.
2. Only if both held: every documentalist case to five runs on that
   commit, the confirming runs counted, stopping before the budget.

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
- **No guard in the engine sees it.** A `note` is free text for a
  person, tied to no doc (`case "note"` in internal/engine/engine.go): the
  engine cannot tell that the doc it lets the agent vouch for is the one
  the note doubts. `checked-unread` refuses only a doc whose sources were
  not given whole; status.md's one source was.
- **Not new.** The same three checks failed together once in five runs
  at d66eeea (2026-10-01, before ADR-0014's step 2). Since step 2 on this
  case: none in five at 7427ae8, none in the weekly runs of 10-04 and
  10-11 (both lost only the issue the run opened,
  [#439](https://github.com/JN0V/workline/issues/439), fixed by
  [#441](https://github.com/JN0V/workline/pull/441)),
  one in three today: 1 run of 10.
- **The patch itself is not kept**: the run's folder goes with the test;
  its note and the checks are in the run's log.

Leads, not built here (the task was a measure), for
[#449](https://github.com/JN0V/workline/issues/449):

- **A note naming its doc**, as a `claim` does, and the engine keeping
  `checked` on a doc an answer's own note doubts: a mechanical guard
  ([principle 7](../PRINCIPLES.md)).
- **Five docs in one call**: the version bump asks one call for the five
  suspect docs; one note for five docs may blur which one it doubts.
  Unmeasured.

## The runs before this measure

[status.md](../../roles/documentalist/docs/status.md) said the cases other
than `drifted` had one run each. They had 5 to 24 rows each in
`results.tsv`, but over several engine commits and models (Haiku, Opus,
two Sonnet models), so few hold five on one engine and one model:

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
  a run here; the weekly timer's runs of the same cases, 19,785 to 21,589
  (10-04, 10-11); 10-01's runs, launched from an agent session as these
  were, 35,892 to 39,934. `stays-on-the-task` alike: 22.7k on 10-01,
  13.2k to 13.5k weekly. The editor's session variable doubled them
  ([claude-session-env.md](claude-session-env.md); [what it cost](#what-it-cost)).
- **A budget built on the weekly rows** is half what a run from a
  session spends: at today's rate, the whole plan would have taken about
  1.4M tokens in, not 1M.

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

### What is left

- **`condenses-the-role-contract`, not measured.** Two runs lost
  `lines-max` at 202 and 203 lines, for 200. The evaluation counts the
  doc's 4-line header; the engine's budget, which applied both, does
  not: 198 and 199 lines
  ([#454](https://github.com/JN0V/workline/issues/454)).
- **Its judge never asked**: "does every rule the role contract stated
  before still hold?" was skipped in all five runs, `WORKLINE_JUDGE`
  unset; the weekly [run.sh](../../tests/evaluation/schedule/run.sh)
  sets `claude:sonnet`. Five runs again with it.
- **`stays-on-the-task`, not run**: the script was stopped by hand before
  it; its five runs, about 68k tokens in (13.2k to 13.5k a run weekly),
  would have crossed the day's 1,000,000.

### What it cost

- **724,311 tokens in and 167,942 out**: 68 runs, 107 calls.
- **The doubling explained.** The same one call of
  `drifted-version-bump` took 44,090 tokens in from a session in the
  morning, 21,701 here: about the weekly timer's. A Claude Code session's
  variables reach the agent and double each call
  ([#452](https://github.com/JN0V/workline/issues/452); its fix,
  [#453](https://github.com/JN0V/workline/pull/453)).
