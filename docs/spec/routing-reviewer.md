---
sources: [internal/line/states.go#never, internal/line/wip.go#waits, internal/line/wip.go#waiting, internal/work/headings.go#Validation, internal/work/headings.go#Scope, internal/work/work.go#check, defaults/routing.yaml]
checked: a19e8cb0
verified: agent:claude-code
---
# Routing — where state lives, the reviewer, what is not built

Part of [Routing](routing.md): where the state of a work item lives, the opt-in reviewer, and what this version leaves out.

## Where state lives

On a forge, the state of a work item is a label (`workline:ready`,
`workline:developing`…, [the line of states](routing-states.md#the-line-of-states)),
readable by humans and by any CI. Without a forge, it
is the `state:` line of the item's file in `.workline/work/<id>.md`, whose
`## Need`, `## Validation` and `## Scope` sections
`workline item ready <id>` checks before moving it — `### ` headings too,
as an issue form writes them, a field left `_No response_` empty, a
footer after the last section left out
([a body's footer](backlog-acts/refine.md#a-bodys-footer)). On a forge,
a person moves an issue by its label too; the engine's moves are one
transition at a time, each said in the line's comment on the issue. In
a file, *built so far: the move to `ready`, and its check; not logged.*

## The reviewer, opt-in

The reviewer
([ADR-0020](../adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)) is in no shipped line: a project adds it to its
`merge-request` line, by hand or with `workline init --review`, and runs
it on a machine before a push with `workline review`. On a merge request it
reviews the commits not reviewed yet, one lens a push, none on a draft;
with `--input lenses=all`, every lens on the whole change, commits
reviewed before too.
The templates pass it once, when the merge request becomes ready
([#129](https://github.com/JN0V/workline/issues/129)): the event says so on GitHub (`opened` not a draft,
`ready_for_review`); on GitLab, where marking ready starts no pipeline,
the next pipeline tells it by the merge request's draft state and its
"marked this merge request as **ready**" note, newer than the
reviewer's comment. It never moves an item: `review` stays the
machine's part, `validation` the person's. Its `spec` event reads a spec
before it is built — a file, or an issue by its number (`--input
issue=<n>`); on a machine, `workline review --spec` or `--issue`
([#128](https://github.com/JN0V/workline/issues/128)).

After the product owner in a line, the reviewer reads the specs it
refined ([#128](https://github.com/JN0V/workline/issues/128)): on `schedule`, up to `specs-per-run` issues
a run ([#433](https://github.com/JN0V/workline/issues/433)),
the findings on each in a comment on it; the product owner's `ready`
waits until none important is open
([a spec read before ready](backlog-acts/spec-review.md)). Opt-in, as on
merge requests:

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer]}
```

The tester writes the acceptance tests of an issue waiting for them
(`needs-tests`, or `ready` with none) on `schedule`, one issue a run,
and on `issue` the one its state names: on `schedule`, the first in the
backlog's order whose Validation names scenarios and whose Draft is not
open
([ADR-0055](../adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md#amendment-2026-10-28-step-4-as-built)).
It needs `--open-merge-request`; opt-in, after the product owner and the
reviewer:

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer, tester]}
```

The developer takes a `ready` issue to a pull request
([ADR-0054](../adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md)):
on `issue`, the one CI names (`--target issue:<n>`) when it is labelled
ready; on `schedule`, the first nobody took, one a run. It needs
`--open-merge-request`; opt-in
([the triggers](../../roles/developer/docs/triggers.md)):

```yaml
routing:
  events: {issue: [developer], schedule: [documentalist, developer]}
```

## Not in this version

Which
act runs at which moment, decided: [ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md).
