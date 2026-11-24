---
sources:
  - cmd/workline/main.go#prePushLine
  - cmd/workline/main.go#docsJudgedRef
  - cmd/workline/main.go#docsCmd
  - internal/review/review.go#Docs
  - roles/documentalist/code/documentalist.go#preRun.findSuspects
  - roles/documentalist/code/documentalist.go#rangeFiles
  - roles/documentalist/code/documentalist.go#preRun.checkHygiene
  - roles/documentalist/role.yaml#events
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — at the push

Part of [at the push and the adoption](push.md).

A project that routes `pre-push` (`routing: {events: {pre-push: [committer,
documentalist]}}`) has those docs counted at the push, in one line, without
an agent: a push never waits on one, nor asks
([ADR-0010](../../../docs/adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md)).

- **Only the docs the commits touched are read**: one of their sources,
  or the doc itself; their own budgets and links are counted. The other
  docs — suspect from before, a passage written twice, a link gone — are
  said on main after the merge and by gardening: about 5 s less a push
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)).
- The person has them judged when they choose — `workline docs` — reviews
  each change, and those kept go in one `docs:` commit
  ([ADR-0007](../../../docs/adr/0007-docs-reviewed-after-the-code-before-the-push.md)).
- `workline docs` judges from where the docs were last judged, a ref it
  moves (`refs/workline/docs-judged`), pushed or not: a repository pushed
  to main with no merge request is caught up there.

## On a merge request's commits

- Given the commits of a merge request (`range`), only the docs they
  touched are read, as at a push: those they made suspect — one of their
  own sources touched by those commits, a merge's own change too — are
  judged, whole, against every source that changed
  ([ADR-0007](../../../docs/adr/0007-docs-reviewed-after-the-code-before-the-push.md)).
  A doc suspect from before is gardening's; the checks across every doc
  run on main after the merge, once
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
  [how](../../../docs/main-after-merge.md)).
