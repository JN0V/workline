<!-- workline
sources: [tests/evaluation]
-->
# Evaluation in detail

What [evaluation](../spec/conformance.md#evaluation) does beyond its outline.

## Real shapes, both sides graded

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

## Pass rates

**Pass rates, not the best.** A measure is read over at least five runs per
model, as the share of runs earning every point, never from the best run.
What a change is designed on is never what it is accepted on: the
**held-out set** — WaterMeter (a solo repository, roles/documentalist/docs/tried.md),
DomoticsCore after its pull request #114, and workline's own nightly
gardening — is not looked at while designing ADR-0014's steps, and is
replayed only to accept them (its step 4).

## The judge

**The judge.** `judge: <question>` asks a yes-or-no question on what the role
produced — "does the rewritten subject keep the author's meaning?" — of the
agent `WORKLINE_JUDGE` names (an `--ai` value: `claude:sonnet`, a `cmd:`),
with the case, the author's words, the role's and the change. Its facets are
in `roles/judge/`. A yes is a point, a no is lost with its reason.
The judge stands as far from the graded model as it can
([ADR-0005](../adr/0005-independence-takes-the-best-level-available.md)): the
`judge` column names its model and the level reached — `provider`, `model`
(another model of the same provider), `context` (the same model, apart). A
`cmd:` judge is taken to be of another provider: whoever names the command
vouches for it. `WORKLINE_JUDGE_AT_LEAST` sets a floor. Without a judge, or
below the floor, the check is skipped: no point earned or lost, said in the
`failed checks` column. The judge's own tokens are not counted.

## Run often

**Run often.** `tests/evaluation/schedule/run.sh` runs it on the local `main`,
in a worktree of its own, and appends to this repository's `results.tsv`
(`WORKLINE_EVAL_RESULTS`); the systemd user units next to it run it every
week, or at the next session when the machine was off. It skips the week when
the last commit changing the engine, the roles or the cases already has three
runs (`WORKLINE_EVAL_RUNS`) and no model changed since: the same code on the
same models adds little, and a subscription counts the tokens. The `workline`
column of `results.tsv` names that commit. The reviewer's cases are left
out (`WORKLINE_EVAL_SKIP`, `reviewer` by default): they are measured on
demand, in steps.

## Tried so far

*Tried so far* (2026-09-26): the judge through `cmd:`, and `claude:sonnet` at
the `model` level on every judged case — it failed rewrites that dropped
"judged" and "instead of ignoring them", and passed a fair cut that
`keeps-words` fails. The weekly run asks `claude:sonnet` unless
`WORKLINE_JUDGE` says otherwise. *Not yet:* a judge of another provider.
