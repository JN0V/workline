---
sources:
  - internal/engine/code.go#weighCode
  - internal/engine/code.go#unreproduced
  - internal/kit/devmark/merge.go#Merge.WaitsForAYes
  - internal/backlog/assume.go#DefaultWait
checked: bd254cd75
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
- **Told by the issue it closes**, never by the agent: the rules, the
  labels and the record on [the developer's merge](../roles/developer/docs/merge.md).

## What the summary says

As at `cautious` ([each word](merge-policy.md#what-the-summary-says)),
and:

| Said | When |
|---|---|
| `review-to-come` | the developer's code, the reviewer in the line, its verdict not on the head yet: it waits for it, 24 h at most |
| `waits-for-a-yes` | the developer's feature or task, a fix whose bug no test reproduced, a fix no review read at its head, or a finding open |

## Where it is read

- **The kind and its gates**: `weighCode`, in `code.go`.
- **A fix whose bug no test reproduced**: `unreproduced`, in `code.go`.
- **A feature or a task, a fix with no yes**: `WaitsForAYes`, in `merge.go`
  of `devmark`.
- **The 24 h**: `DefaultWait`, in `assume.go`, unless `acts.ask.wait`
  says another.
