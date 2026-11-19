---
type: reference
sources: [internal/backlog/backlog.go#moves, internal/backlog/backlog.go#builder, internal/backlog/backlog.go#questions, internal/backlog/backlog.go#question, internal/backlog/backlog.go#holds, internal/backlog/assume.go, internal/backlog/proof.go, internal/backlog/plain.go, internal/backlog/repeat.go, internal/backlog/write.go, internal/work/headings.go, internal/work/work.go, internal/work/scenarios.go, internal/work/scope.go, internal/backlog/scenarios.go, internal/backlog/verification.go]
status: draft
checked: a19e8cb0
verified: agent:claude-code
---
# Refining to ready — a question, a proof kind, a blocked issue, a size

Part of [Refining to ready](refine.md): what refining does past writing the sections.

## A question nobody answers

[ADR-0060](../../adr/0060-the-person-is-on-the-loop.md#1-questions-time-out-as-lazy-consensus):
a question unanswered for the level's delay is taken as the role's
assumption, written on the issue, and the issue goes on.

```yaml
- assume: {issue: 14, assumption: "CSV, as the other exports.", why: "No answer in 24 h."}
```

- **The delay**, `acts.ask.wait` (`backlog.Delay`): `never` at
  `cautious`, `24h` at `normal`, `"0"` at `enterprising`
  ([settings](../../../roles/product-owner/docs/settings.md#how-long-a-question-waits)).
  Counted from the question's comment as the forge dates it
  (`created_at`, `backlog.Unanswered`), never from the issue's last
  update; a comment after it, anyone's but the engine's, is an answer
  and cancels it. A forge that dates no comment: never timed out
  (`assume-undated`).
- **Told once**: past the delay, the task says so for that issue
  (`told` in the state); the line wakes the role on it once
  ([routing](../routing.md#once-a-state-the-record)). At `"0"`, the task
  asks an `assume` beside each `ask`.
- **`assume`** is done alone, at every level the delay allows, checked:
  a question waiting past the delay, or asked in this run at `"0"`; not
  taken already (`assumed` in the state, the question's round). Dropped
  otherwise: `assume-early`, `assume-waits` (`never`),
  `assume-nothing-asked`, `assumed-already`, `assume-empty`.
- **Written in the body**, under `## Assumptions` at its end, above a
  footer (`backlog.WithAssumption`): "- Assumption taken on <day>, the
  question of <day> unanswered after 24 h: <assumption>." Need, Example,
  Validation and the builder's part are left as they are. Said in the
  role's one comment: "Assumption taken on #14: …. Answer here, or edit
  its line under Assumptions in the issue, to change it."
- **A draft is a question too**: a Need or Validation draft no person
  accepted, all that keeps the issue from ready (`backlog.OnlyDrafts`),
  waits the same delay, from when the forge says the issue got
  `workline:draft`, or a person's later comment. Past it, the engine
  moves the issue to ready with no agent, its draft marks out, a line
  under Assumptions saying the draft was taken as written; an agent's
  `ready` passes so too. At `cautious`, `ready` still needs a person.
- **A late answer**: a counted comment after the assumption
  (`assumed-at`) marks each line "Answered by @x on <day>: their answer
  stands over it", with no agent, and is read as a person's comment on
  what the role proposed: its own sections rewritten, `unready` if ready
  and no longer true ([answers](answers.md)).

## No test possible: a proof kind picked

The tester found no test possible and sent the issue back to refine
([ADR-0059](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#4-a-proof-kind-not-always-a-new-test)):

```yaml
- ready: {issue: 12, why: "Only the README's words change.", proof: {kind: docs-checks, reason: "The documentalist's checks prove it."}}
```

- **Read once**: the tester's reason, recorded with `none`, is given in
  the task until read (`no-test` in the state, its digest).
- **`ready` with `proof`** picks the kind in place of new tests:
  `existing-suite`, `docs-checks`, or `flaky-n-of-n` with its `test` (and
  `runs`). The issue goes to ready, not to its tests; the kind is kept
  in its state, `answers` naming the tester's word. The developer takes
  it while that word is the tester's last (`backlog.ReadProof`).
- **Refused**: a `proof` where the tester did not find none
  (`proof-not-asked`), with no reason, an unknown kind, a flaky test
  unnamed (`proof-reason-missing`, `proof-kind-unknown`,
  `proof-test-missing`).
- **Or back to its tests**: the role rewrites what keeps a test from
  being written — its Scope, its own Validation draft
  ([#1053](https://github.com/JN0V/workline/issues/1053)) — or asks; a
  `ready` with no proof beside that rewrite goes to
  `workline:needs-tests`.
- **Or back unchanged, once**
  ([#1053](https://github.com/JN0V/workline/issues/1053)): the reason
  is the tester's own — code missing from its task, a run that failed —
  not the issue's. `ready` with `retest: true` sends it to its tests as
  it stands; the state keeps the word it answers, with the body the
  tester read (`retest`, `backlog.RetestKey`), and the tester asks its
  agent again (`backlog.Retested`). The tester's next
  "no test possible" is recorded `retried`: a second retest is refused
  (`retest-spent`), as is one beside a `proof` (`retest-with-proof`) or
  with no "no test possible" to answer (`retest-not-asked`). On a text
  rewritten since the tester read it, or by a refine of the same run,
  it is a plain `ready`.

```yaml
- ready: {issue: 12, retest: true, why: "The tester lacked the counter's code, not a scenario."}
```

- **Never back unchanged otherwise**
  ([#815](https://github.com/JN0V/workline/issues/815)): the tester's
  record holds the digest of the body it read. While the body is that
  one, no kind picked and no retest sent (`backlog.NoTestStands`), a
  `ready` with no proof is refused (`no-test-unanswered`) unless a refine of the same
  run rewrites it; neither the spec review nor `workline:accepted`
  moves it. Left with nothing, the issue is said on it
  (`no-test-waits`), and the line moves it to `workline:blocked`, the
  tester's reason said (`no-test-unsettled`,
  [routing](../routing.md#the-line-of-states)). An older record with no digest holds nothing back:
  the tester's next word records one.
- **With no agent**, a person of the project comments `proof: <kind> —
  why` after the tester's comment (`backlog.PersonsPick`): the engine
  does that `ready`, whatever the role's mode, as their yes.

## A blocked issue read again

The line wakes the role on a blocked issue with news on it, or a week on
([routing](../routing.md#blocked-read-again),
[ADR-0059, amended](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-blocked-is-read-again)).
The task says why it was blocked, what is new, and the state it came
from.

```yaml
- unblock: {issue: 12, to: tests, why: "#9, which it waited on, is merged."}
- blocked: {issue: 12, why: "The objection needs the maintainer's call.", person: true}
```

- **`unblock`** moves it to `to`: refine, or the state it came from
  (`revisit-from`; Ready for one a developer had claimed). Every other
  state's label off, its card moved; the role's comment says why, and
  that putting `workline:blocked` back undoes it. Another `to` is
  dropped (`unblock-to`).
- **`blocked`** keeps it there, its reason said afresh in the role's
  comment; `person: true` says only a person can lift it. Nothing moved.
- **Done alone**, at every level, as `unready` is: cheap, undone by a
  label. Either on an issue not blocked is dropped (`not-blocked`); a
  `ready` on a blocked one too (`blocked-first`).
- **Or a `close`**, by its own rules.
- **A `refine` alone**, on an issue the line blocked from refine for
  the product owner's own read left lacking, its text rewritten and no
  `unblock`, `blocked`, `close` nor `split` on it: the engine moves it
  back to refine in the same run, said as an `unblock`. The read it was
  blocked on is gone, so it is read again in refine as it stands, not
  left blocked a week more
  ([#1206](https://github.com/JN0V/workline/issues/1206)). A refine
  proposed or dropped moves nothing; nor does any refine on a block a
  person or another role made, or one on a run's size: a rewrite does
  not lift those.
- **The developer's ask for its Scope**, from before
  [#1228](https://github.com/JN0V/workline/issues/1228) made the Scope a hint
  ([ADR-0059, amended](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-fix-outside-the-scope-asks-the-product-owner)):
  the task names the files. To widen it, a `refine` rewrites the Scope
  whole, the files in it, then `unblock` to ready: the developer
  resumes on its pull request. Else `blocked`, why not.
- **A run too big for its role**
  ([ADR-0064, step 3](../../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)):
  woken at once (`size`, the role), told what crossed with the run's
  numbers, and the issue's fresh tokens so far against its normal and
  its backstop ([#1193](https://github.com/JN0V/workline/issues/1193)).
  To narrow it, a `refine` rewrites the Scope whole, fewer files, a
  person's Scope too, then `unblock` to its state. A narrowing not
  written holds the unblock beside it (`scope-not-narrowed`); sent back
  as it is, the next crossing waits for a person. Its need atomic — no part
  passes its own scenarios apart — at the
  normal size: `size` with `tier: large`, then `unblock`
  ([ADR-0064, step 4](../../adr/0064-size-is-a-target-not-a-wall.md#3-the-product-owner-decides-once-a-state-an-issue)).
  Parts that pass apart, the issue no part itself: a `split`,
  `too-big: true` ([split on size](split.md#split-on-size)), then
  `blocked`, `person: true`: a person accepts it from its parts. Else
  `blocked`, `person: true`, why it cannot be narrowed.

## Its size at ready

On the line's refine step, the task asks the product owner to predict
the issue's size before `ready`
([ADR-0064, §2](../../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)),
in the same call:

- **Against its normal**: the fresh tokens an issue of its kind spends
  here, the project's 85th percentile kept by `workline size`, else the
  seed (400k, doubled at the highest effort); times its tier.
- **From** its Scope's files, its scenarios and its effort.
- **Well past it**: a `split`, `too-big: true`, and no `ready`; or, its
  need atomic, `size` with `tier: large`, then `ready`. A part of a
  split is offered no split, an issue with a tier no grant.
- **A bug, its tests first**: in place of the split, `ready` with
  `too-big: true`; once its tests are written, the line asks its split
  ([split on size](split.md#split-on-size)).

## Granting a size

```yaml
- size: {issue: 12, tier: large, why: "one parser: no part of it passes its tests alone"}
```

- **What it does**: puts `workline:size/large` on the issue; every size
  target of the runs on it doubles (its context budget, token cap, code
  and change given at once). Said on the issue, kept in its state
  (`size: large`), counted and sampled by the auditor.
- **Checked**: `large` alone; `x-large` is a person's (`size-person`);
  an issue with a tier already is never granted (`size-kept`); one the
  role granted once, never again: a person who took the label off
  decided (`size-once`).
- **Never the backstop**: the issue's runs in all still stop at 4× its
  normal, for a person
  ([routing](../routing.md#the-line-of-states)).
- **Mode**: `act` at `normal` and `enterprising`, `propose` at
  `cautious` ([autonomy](autonomy.md)).

## The tester's Verification

How the work is proved is the tester's, as a human tester writes the
test plan ([ADR-0066](../../adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md),
[#1261](https://github.com/JN0V/workline/issues/1261)):

- **The product owner writes none**: a `verification` in its `refine` or
  a `split` child is left out (`work.Testers`); ready never waits for one.
- **The tester writes it** once its tests are proved red and linted, by
  the engine, not its agent (intention `verification`, `tester.testPlan`):
  each scenario, the test that proves it, or why none can. It is
  marked `<!-- workline:verification=tester -->`, below the line, and its
  comment on the issue says it wrote it.
- **Its own is rewritten** at its next tests, and so is one the product
  owner wrote before, as its state's `wrote` records it, unedited
  (`backlog.TestersToWrite`). **A person's is kept** word for word
  (`backlog.WithVerification`). When that text names not every test,
  the tester's comment says the tests prove it another way, naming each.
- **A reviewer's finding on it is the tester's**: it holds nothing from
  ready (`SpecReview.Testers`), it clears the spec as no finding would
  (`backlog.SpecCleared`: the engine's ready with no agent), it spends no
  round of the review, its rounds spent it is no person's, the product
  owner is not given it, and the tester's task quotes it
  (`backlog.SpecForTester`).
- **No tester routed, no Verification**: the product owner's ready puts
  `workline:ready` straight on, with no `needs-tests` step; the issue is
  built without one. Route the tester to have it written.

## A body's footer

A body's footer belongs to no section (`work.Footer`): what a tool or a
person appends after the last section is not the Scope's text. It is,
outside fenced code (CommonMark's fences: three or more backticks or
tildes open one, a backtick fence's info string holding no backtick; a
line of the same character alone, as long or longer, closes it):

- from a **thematic break** (`---`, `***`, `___`, CommonMark's, a blank
  line above it, so not a heading's underline) after the last heading, to
  the end — one with headings after it, as between the person's part and
  the builder's, is no footer, and is not the text of the section it
  ends (`work.Sections`): a draft's digest is the same with it or without;
- or else the body's **last line, when it is a tool's signature**: alone
  in its paragraph, reading "Generated with …" or "Generated by …", after
  an emoji or a mark if any — `🤖 Generated with [a tool](https://…)`.

Every reading of the sections leaves it out — `ready`'s check, the
state's `sections`, `wrote` and `deleted`, a finding's section —, and a
section `refine` adds, or a parent's `## Sub-issues`, goes above it,
the footer left last as it was.
