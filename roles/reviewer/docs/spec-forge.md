---
sources:
  - defaults/routing.yaml#events.schedule
  - roles/reviewer/role.yaml#settings.specs-per-run
  - roles/reviewer/code/specforge.go#onlyIssue
  - roles/reviewer/code/specforge.go#waitingSpecs
  - roles/reviewer/code/specforge.go#keepersOrder
  - roles/reviewer/code/specforge.go#specOnForge
  - roles/reviewer/code/specforge.go#settleSpecs
  - roles/reviewer/code/specforge.go#waitingQuestions
  - roles/reviewer/code/spec.go#prepareSpecs
  - internal/backlog/backlog.go#Readied
  - internal/backlog/backlog.go#NotReady
  - internal/work/headings.go#Required
  - internal/backlog/spec.go#SpecHold
  - internal/backlog/spec.go#UnreadRecord
  - internal/backlog/spec.go#SpecUnread
checked: cf8d4d9f
verified: human:JN0V
---
# Reviewer — a spec on the forge

Part of [Reviewer — a spec before it is built](spec.md); its comment,
the hold and the rounds: [a spec's hold and rounds](spec-rounds.md).

Opt-in, off by default: the reviewer after the product owner in the
gardening line
([ADR-0020](../../../docs/adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)):

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer]}
```

## Which issues, how many

- **Which issues**: those the product owner keeps (its state comment),
  not `workline:ready` nor `workline:needs-tests`, their Need, Validation and Scope written (drafts too), their
  body not read as it is. A person's text on top stands for the Need,
  as the product owner reads it with its state
  ([#1062](https://github.com/JN0V/workline/issues/1062)). None: no agent asked.
  - On the line's `issue` event, the one issue the product owner just
    ran on, its state comment written yet or not
    ([#840](https://github.com/JN0V/workline/issues/840)).
- **How many**: up to `specs-per-run` a run (5), in the backlog's order
  ([#433](https://github.com/JN0V/workline/issues/433)), its priorities
  read where the product owner's settings say the project keeps them
  ([settings](../../product-owner/docs/settings.md#priority-and-effort)):
  - each read as one alone: its lens call, its judges, its caps
    (`findings-max`, `questions-max`), its comment;
  - the rest wait, named in one finding (`spec-waits`), and come first
    at the next run unless the backlog puts another before them;
  - the run's tokens spent, the specs after are not read: nothing
    written on them, each said (`spec-not-read`), read at the next run.

## In CI, and when it cannot run

- **In CI**: judged, then applied. On the gardening line the reviewer
  reads what an earlier run refined, so a round takes two nights; on the
  line's `issue` event it reads the spec in refine's own run, and its
  review wakes the product owner again at once
  ([the line of states](../../../docs/spec/routing.md#the-line-of-states)). `forge-writes: false` writes no
  comment ([settings](../README.md#settings)): ready then stays held, `spec-not-reviewed`.
- **Without AI**: nothing read, nothing written, each spec said
  (`not-reviewed`); ready held. The sixth round's question
  ([rounds](spec-rounds.md#rounds-and-a-spec-not-settled)) needs
  no agent: written all the same.
- **An agent out of reach**: the run is blocked-external
  ([as for code](judge.md#8-the-verdict-and-what-goes-wrong)) and writes
  nothing, that question included; said, it waits for the next run.
- **Its cost**: a spec 7k to 37k tokens in, judges included; five a night
  stay within `ai-max-tokens`
  ([measured](../../../docs/research/code-review.md#the-reviewers-cost-measured)).

## A record that does not read

- **A record that does not read**
  ([#302](https://github.com/JN0V/workline/issues/302)) is never taken
  for none, which read the spec again at round 1, an agent call a night.
  - The issue is passed over, ready held, the next issue read instead.
  - Said each run (`spec-record-unread`), naming the issue and the line.
  - Released when a person deletes the comment (read anew, round 1) or
    accepts the issue (`workline:accepted`).
