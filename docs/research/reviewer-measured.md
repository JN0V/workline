# The reviewer, measured (2026-10-15)

**Question.** Before a developer agent receives the reviewer's findings:
how often does each lens find a planted defect, how often is it wrong,
what does the judge refuse, what does a review cost, and is a cheaper
push as good ([#360](https://github.com/JN0V/workline/issues/360))?

**Verdict.**

- **Every lens together found every planted defect**, five runs a case:
  55 of 55, an important finding verified on 49. Lenses asked apart were
  not measured: together missed nothing.
- **Correctness verified its own 15 of 15**, held no clean change, and
  no review on a false finding: the one finding it verified that nothing
  planted is true, by both raters.
- **No false finding verified** by any lens together: the two false ones
  were shown not verified, which only warns.
- **Tests finds all, verifies 9 of 15**: in 6 runs it raised the defect
  as a nit (a test passing either way, an untested function), and a nit
  is never judged. Its 12 findings on nothing planted are true: the
  planted defect seen from its test.
- **Intent and claims**: 5 of 5 each, verified.
- **A review: 21.4k tokens in**, 2.3k out, on these small changes.
- **The diff alone does not win** on a push: a third cheaper, but 8 of
  18 defects verified against 11 of 11 for the lens of the defect.

## Every lens together, five runs

The role's way on a machine and when a merge request becomes ready: one
call for every lens, each important finding judged. 14 cases, 70
reviews.

| Lens | Found / planted | By itself | Its own verified | A finding at the defect refused | Important on nothing planted (verified) | Nits on nothing planted | Refused on nothing planted |
|---|---|---|---|---|---|---|---|
| correctness | 15/15 | 15 | 15 | 7 runs | 2 (1) | 0 | 4 |
| edge cases | 15/15 | 14 | 13 | 1 | 1 (0) | 4 | 1 |
| tests | 15/15 | 15 | 9 | 1 | 12 (12) | 11 | 0 |
| intent | 5/5 | 5 | 5 | 3 | 0 | 0 | 0 |
| claims | 5/5 | 5 | 5 | 0 | 1 (0) | 7 | 0 |

- **Found**: a finding shown on the change at the defect's place, by any
  lens, a nit included
  ([#90](https://github.com/JN0V/workline/issues/90)'s point). **Its own verified**: the lens of
  the defect's kind raised it important, and the judge said yes: what
  `ai-findings: block` on that lens holds a merge on.
- **Refused at the defect**: another finding at the same place, the
  defect verified anyway. Of correctness's 7, 5 are the claims lens on
  the take case, the judge answering that the code does what its
  message says: "never take a product below zero".
- **Clean changes**: the rename, one important finding in five runs
  (above); `Has` and `Clear`, none.
- **Outside the change**: two findings, both refused: no issue opened.
  `Snapshot`, untested on `main`, raised in no run.
- **Decisions for a person**: none asked.
- **The judge**: 141 findings judged, 111 yes, 20 no, 10 shown not
  verified; 160 calls, 19 of them a judge asked again, after its reads.
- **Tokens**: 1,499,972 in and 159,316 out, 239 calls; 21.4k in a
  review, 6.5k to 46.4k. The lenses' call 8.1k; a finding judged
  6.6k, its rounds of reads included.

## On nothing planted

16 important findings, read against the code by two raters (method,
below), who agree on each one's truth:

- **correctness**: "Return can put back more than was taken", verified:
  true (Return checks nothing of `n`), worth acting on for one rater of
  two. "Mean ignores quantities", refused four times, shown not verified
  once: false (an item is a line; doc and test agree).
- **edge cases**: on the clean rename, "a negative quantity lowers the
  total", shown not verified: true of the code, but on `main` before the
  rename.
- **tests**: 12, verified, all true: no test of a bad coupon (3), of an
  empty cart (4), of a page below 1 (1; the test file predates the
  change), of an order with a zero line (4).
- **claims**: MeanPrice's doc against its code, shown not verified:
  false.
- **A true finding refused**: "zero total quantity not handled", one
  line above the empty cart's place; the defect was verified anyway.

## The correctness choice

`ai-findings` for correctness, on these 70 reviews:

| | block | warn |
|---|---|---|
| Reviews held | 29: 28 on a planted defect (15 correctness, 13 of another kind), 1 on a true finding nothing planted | none |
| Clean changes held | 0 of 15 | none |
| Held on a false finding | 0 | — |
| Correctness defects let through | 0 of 15 | 15 of 15, each shown verified, merged unless the person reads it |

- Under `block`, a verified finding holds the merge until a commit
  answers it ([ADR-0020](../adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md));
  a finding shown not verified only warns, as the one false correctness
  finding shown did.
- On a push, one lens asked alone: correctness verified its 3 defects,
  nothing on the 3 clean changes.

## Intent and claims

- **Intent**: a change closing the case's issue (`#8`, in its simulated
  forge), leaving out its Need's refusal of a zero line. Found 5 of 5,
  quoting the issue, verified; in four runs the other lenses
  also found it in Restock's code. Asked only when a change closes an
  issue.
- **Claims**: a message saying "no change in behaviour" over a Total
  that drops non-positive lines. Found 5 of 5, verified. On the other 13
  cases, 65 reviews: one important finding shown not verified, 7 nits,
  12 claims dropped as not found in what the author said.

## A cheaper push

A push to a merge request asks one lens, in turn
([ADR-0020](../adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md));
[#243](https://github.com/JN0V/workline/issues/243) asks whether the diff
alone would do instead. Both measured on the same cases; the budget left
one run of each, two of the diff alone on 9 cases.

| A push | Reviews | Shown at the defect | Important, verified | Important on nothing planted | Tokens in a push |
|---|---|---|---|---|---|
| One lens, the defect's | 14 | 11/11 | 11/11 | 1, verified, false: "mean ignores quantities" (edge cases) | 15.2k |
| The diff alone | 23 | 14/18 | 8/18 | 0 | 10.3k |
| Every lens together (above) | 70 | 55/55 | 49/55 | 16: 14 true, 2 false not verified | 21.4k |

- **One lens** was asked the lens of the defect planted: the push whose
  lens in turn is the right one, its best case; correctness on a clean
  change. Whichever lens is in turn, about 62% of pushes show the
  defect, estimated from the together runs (the share of the lenses in
  turn showing it there), not measured.
- **The diff alone** verifies the correctness defects (6 of 6) and the
  page guard (2 of 2); it misses the empty cart twice and the untested
  `Shipping`, and verifies none of the tests, intent and claims defects
  nor the negative `Return` (nits, or refused): it holds no test file,
  no issue, no message.
- **Does it win?** No. It is a third cheaper, and shows a defect more
  often than the average lens in turn (14 of 18, against about 62%), but
  it verifies 8 of 18, all on code a lens in turn verifies too:
  correctness and the page guard. #243's setting is not built.

## Method

- **The cases**: the fourteen of
  [tests/evaluation/cases/reviewer](../../tests/evaluation/cases/reviewer),
  on the `shop` fixture: three planted defects for each code lens, one
  intent, one claims, two clean changes, one with untested code outside
  the change.
- **The engine**: b62ca106, its roles as on `main` (fe0b3a7a). From
  d89c0fa8 the intent case also places its defect in Restock's code; its
  first run, scored before, is read here with that place.
- **The models**: Sonnet (claude-sonnet-5-5) finding, Opus
  (claude-opus-5-5) judging, at the `model` level; the judge reads
  (`judge-read-rounds: 2`).
- **A case a process**, five at a time; the tokens read from
  `results.tsv` after each; a case not started when it could cross 2.0M
  tokens in, all runs together. Stopped at 1,949,571.
- **The raters**: the 9 distinct findings on nothing planted, given to
  two agents (Fable, Opus) with each case's repository, never the
  judge's verdict: true or not, caused by the change or not, worth
  acting on before a merge. Agreed on every truth; on "worth acting on"
  for 8 of 9.

```sh
# every lens together, five runs
for i in 1 2 3 4 5; do WORKLINE_EVAL=claude WORKLINE_EVAL_REVIEWER='{ai-max-tokens: 50000}' go test -count=1 -timeout 60m -run 'TestEvaluation/reviewer/' ./tests/evaluation/; done
# the diff alone
WORKLINE_EVAL=claude WORKLINE_EVAL_LENSES=diff-alone WORKLINE_EVAL_REVIEWER='{ai-max-tokens: 40000}' go test -count=1 -timeout 60m -run 'TestEvaluation/reviewer/' ./tests/evaluation/
# one lens: the defect's, correctness on a clean change
for l in correctness edge-cases tests intent claims; do
  cases=$l; [ $l = correctness ] && cases='correctness|clean|tests-untested'; [ $l = tests ] && cases='tests-(restock|rounding|shipping)'
  WORKLINE_EVAL=claude WORKLINE_EVAL_LENSES=$l WORKLINE_EVAL_REVIEWER='{ai-max-tokens: 40000}' go test -count=1 -timeout 60m -run "TestEvaluation/reviewer/($cases)" ./tests/evaluation/
done
# the sums: a row per lens and way of asking
awk -F'\t' 'NR==1 || $2 == "b62ca106" || $2 == "d89c0fa8"' tests/evaluation/results.tsv > "$TMPDIR/runs.tsv"
go run ./tests/evaluation/summary "$TMPDIR/runs.tsv"
```

## Limits

- **Small changes**: one commit, a few lines, on a 150-line shop. A
  review of a real pull request reads more and costs more
  ([the reviewer's cost](code-review.md#the-reviewers-cost-measured)).
- **Planted defects, one a case**: a finding on another true defect
  counts as "nothing planted" until read; the raters read those.
- **"Found" is lenient**: any finding at the defect's lines counts, a
  nit or another angle included. "Its own verified" is the strict count.
- **The cheaper push**: one or two runs, short of the five of
  [ADR-0014](../adr/0014-checked-is-earned-by-what-was-read.md); one
  lens measured only when it is the defect's.
- **One provider**: finder, judge and raters are Claude models.
