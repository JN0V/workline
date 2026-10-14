---
sources: [tests/evaluation]
checked: f28701f9
judged: 62abc22
verified: agent:claude-code
---
# Evaluation — three measures

Three measures of [Evaluation](../conformance.md#evaluation), told in full.

## Real shapes

**Real shapes, both sides graded** (ADR-0014, step 1). The `drifted`
fixture (tests/conformance/fixtures/repos/drifted.sh) holds docs drifted as
DomoticsCore's and workline's did, every one `checked`, its sources fitting
whole in a task: frozen line counts (a fenced tree's "(524 lines)", a
`| File | Lines |` row, "~283 lines"), limits that are no counts ("< 800
lines"), a sibling sharing the version, a stale code comment against the
default, a bug long fixed, a test count in a file no source names, a true
claim only a record backs, and a clean control. Each documentalist case on
it grades both sides: never `checked` over a planted falsehood, and
`checked` on the control (`TestDriftedCasesGradeBoth`). Its cases are played
without a real agent on every `go test`, by a fake `cmd:` agent changing
headers only (`TestDriftedWithFakeAgents`): vouching for every doc must lose
the falsehood checks, recording `judged` on every doc must lose the
control, doing both right must lose neither; the `count-off` findings,
reported by the engine with no agent (ADR-0014, step 2), are graded there
too and must never be lost.

To measure the `drifted` cases again, five runs, as ADR-0014's steps 1
and 3 ran them (`-v` for the notes):

```sh
for i in 1 2 3 4 5; do WORKLINE_EVAL=claude:sonnet go test -count=1 -timeout 60m -run 'TestEvaluation/documentalist/drifted' ./tests/evaluation/; done
# the summary pools every run of a case: keep one engine's rows apart (the workline column)
awk -F'\t' 'NR==1 || $2 == "<commit>"' tests/evaluation/results.tsv > "$TMPDIR/runs.tsv" && go run ./tests/evaluation/summary "$TMPDIR/runs.tsv" | grep drifted
```

## The reviewer's measure

**The reviewer's measure** (#90). A reviewer case adds `review:` to
`given` and `run`: the `defects` planted on the change, each a lens, a
kind, and the places its cause may be quoted at (a file, a range of lines
as they read at the change's head, a `text` within them); the places
where nothing may be found (`not`); and code outside the change
(`outside`), measured and not scored. Built on the `shop` fixture, every
lens asked, each apart (`lenses-together: false`, set by the harness),
event `review`. The run's folder is read with no agent — what
each lens answered, what the engine kept of it, the judge's verdict on
each important finding, the tokens each lens's call and its judges used —
and scored: a point for each defect a finding shown on the change has its
cause in, whichever lens raised it (AI review benchmarks count known bugs
found and false positives, docs/research/code-review.md; the place, from
the quote the engine found again, needs no judge); a point for each
`not` kept clean; a point when nothing important was shown that nothing
planted. The `measure` column of `results.tsv` holds, a lens each, the
defects planted for it and found, those it raised itself, those the judge
refused, what it raised that nothing planted (shown, nits, refused),
outside the change (and verified), its answers, its quotes not found
again, its tokens, and the finder floor asked; the summary sums them by
lens. `TestReviewerCasesPointRight` builds each case — each place holds
its text, the code builds, the fixture's tests pass, a planted defect
they caught measuring nothing — and `TestReviewerWithFakeAgent` plays one
through the engine with a fake agent (testdata/review-agent.sh). The
model and independence matrix of ADR-0005 is a run with
`WORKLINE_EVAL=claude:<model>` and `WORKLINE_JUDGE=claude:<model>`, never
by default.

## The judge's replay

**The judge's replay**
([ADR-0045](../../adr/0045-the-judge-asks-for-the-code-it-needs.md),
[#362](https://github.com/JN0V/workline/issues/362)). The 40 verdicts of
the reviewer's judge measured on 2026-10-14
([the judge, measured](../../research/judge-measured.md)), kept in
`tests/evaluation/judge-replay/`, a folder each:

- `question.yaml`, `answer.yaml`: what the judge was asked and answered,
  copied byte for byte from the run's `workline-runs` artifact
  (`in/judge/<key>/`);
- `item.yaml`: the pull request, lens, finding, `where` and `commit`; the
  run and folder it came from (`origin: artifact`; `rebuilt` would mark
  one remade from git, its artifact gone); what the judge said then and
  its tokens; the truth the measure settled (`truth`, `settled-by`:
  agreement or panel, `matters` when the panel was asked, `evidence`).

`TestJudgeReplay` asks each question again, in a context of its own, of
the judge `WORKLINE_EVAL` names, with today's `roles/judge`, in one of
two arms
([ADR-0045](../../adr/0045-the-judge-asks-for-the-code-it-needs.md#measured-again-before-it-is-on-by-default)):

- **today's judge**: a no drops the finding; case `judge/<item>`;
- **the judge that reads** (`WORKLINE_EVAL_REPLAY_ROUNDS`, 2;
  `WORKLINE_EVAL_REPLAY_LINES`, 400): asked as the reviewer asks it,
  its reads served from this repository's git objects at the item's
  commit (`reviewer.JudgeReading`); a no drops the finding only on its
  quote, found again; case `judge-reads/<item>`.

A line of `results.tsv` an item, a point for a true finding verified or
a false one dropped; the run's tally is logged: right of the 20 it said
yes to then and of the 20 it said no to, the true findings dropped then
and again, the false ones let through, its yes right, unsure, the reads
and the tokens a judge. Past `WORKLINE_EVAL_TOKENS_MAX` tokens in
(2100000), counting `WORKLINE_EVAL_TOKENS_BEFORE` (what earlier runs of
the same replay spent), no call starts, those in flight finish (`-parallel`
at most), and it fails, said. Never in the weekly run:

```sh
WORKLINE_EVAL=claude:opus go test -count=1 -timeout 60m -parallel 4 -run 'TestJudgeReplay$' -v ./tests/evaluation/
WORKLINE_EVAL=claude:opus WORKLINE_EVAL_REPLAY_ROUNDS=2 WORKLINE_EVAL_TOKENS_BEFORE=<spent> go test -count=1 -timeout 90m -parallel 4 -run 'TestJudgeReplay$' -v ./tests/evaluation/
```

With no agent, on every `go test`: `TestJudgeReplaySet` reads the set and
scores what the judge said then — 31 of 40, its yes right 19 of 20, its
no 12 of 20, as measured; `TestJudgeReplayWithFakeJudge` asks the 40 of a
`cmd:` judge, and stops at its tokens;
`TestJudgeReplayReadingWithFakeJudge` asks one as the judge that reads.
Replayed on 2026-10-14:
[results](../../research/judge-measured.md#replayed-2026-10-14).
