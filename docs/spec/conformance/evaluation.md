---
sources: [tests/evaluation]
status: draft
---
# Evaluation — two measures

Two measures of [Evaluation](../conformance.md#evaluation), told in full.

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
