---
sources:
  - internal/engine/code.go#weighCode
  - internal/engine/code.go#unreproduced
  - internal/kit/devmark/merge.go#Merge.WaitsForAYes
  - internal/backlog/assume.go#DefaultWait
checked: e81df7b34
verified: agent:claude-code
---
# The developer's code at `normal`

Part of [the merge policy at `normal`](merge-policy-normal.md).

## The developer's code: a fix alone, a feature with a yes

- **A fifth kind, `code`**: the developer's pull request for an issue
  its labels call a fix, every gate passed and no finding of
  the reviewer's open that blocks or is Important; findings it cannot
  read from the reviewer's record hold it.
- **A fix, once the reviewer read its head**: the reviewer's summary
  names the pull request's head, read by its lenses, or its record
  covers a head rebased since, or the engine's own clean merge of the
  base since; else it waits for the reviewer, 24 h
  after the push where the line runs it, then for a person's yes, said
  on it
  ([ADR-0065](adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence),
  [the detail](../roles/developer/docs/merge.md#what-each-gets-at-normal)).
- **A fix, once its bug was reproduced**: its code starts from the
  tester's Draft, whose tests failed on the base; else it waits for a
  person's yes, said once on it
  ([ADR-0060](adr/0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind)).
- **A feature or a task waits for a person's yes**, said once on it: a
  `task` is a label an agent may put
  ([#937](https://github.com/JN0V/workline/issues/937)).
- **The comment that asks for the yes says what the label alone will
  not mend**: a red verdict on the head, or a conflict with main. The
  pull request merges once the line passes on a new head, or once it is
  brought up to date with its base. The comment for open findings, an
  unread head or an unreproduced bug says it too.
- **A wait said at an earlier head is edited in place**, not left to
  ask for a yes: on a head the line blocks, the comment says the pull
  request merges once the line passes on a new head, a person's yes
  neither.
- **Told by the issue it closes**, never by the agent: the rules, the
  labels and the record on [the developer's merge](../roles/developer/docs/merge.md).

## What the summary says

As at `cautious` ([each word](merge-policy-summary.md)),
and:

| Said | When |
|---|---|
| `review-to-come` | the developer's code, the reviewer in the line, its verdict not on the head yet: it waits for it, 24 h at most |
| `waits-for-a-yes` | the developer's feature or task, a fix whose bug no test reproduced, a fix no review read at its head, or a finding open |

## Where it is read

- **The kind and its gates**: `code.go` weighs the developer's pull request
  against the gates above.
- **A fix whose bug no test reproduced**: `code.go` also says whether a test
  reproduced it.
- **A feature or a task, a fix with no yes**: `merge.go`, in the developer's
  marks, says whether the pull request waits for a person's yes.
- **The 24 h**: `assume.go` fixes the wait at 24 h. No setting changes it.
