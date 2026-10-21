---
sources: [roles/reviewer/role.yaml, internal/builtin/reviewer/once.go, internal/builtin/reviewer/reviewer.go, internal/builtin/reviewer/record.go]
checked: ae1b860d
verified: agent:claude-code
---
# Reviewer — a finding judged once

Part of [the reviewer](../README.md), on a merge request: step 7 of
[a run](run.md), [the judge](judge.md), when a finding comes back.

[ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
step 6, applies [ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md)
to the judges: the same finding on the same patch is not judged again.

## What is kept

- **Its verdict**, read from its line in the comment
  ([every finding judged](judge.md#every-finding-judged-kept-for-the-weekly-sample)):
  no judge asked.
- **Said** on the finding: "judged reviewing `abc1234`: the same finding
  on the same patch, not judged again". The summary counts them.
- **A yes** keeps its independence. When `judge-at-least` now asks more,
  it is judged again.
- **A no** keeps its reason; the finding is dropped, said.
- **No new line**: a verdict kept is not a second act for the weekly
  sample.

## What is the same

- **The same finding**: its lens, its cause and claim as quoted, spaces
  aside, its symptom. A title worded otherwise is the same finding.
- **The same patch**: the patch id of the merge request's change to the
  cause's file, base to head.
  - A rebase, or a merge of the base, keeps it.
  - A commit changing that file again makes a new one: judged again.
  - A file the change leaves: its blob at the head.
  - A cause in an issue: the whole change's patch id, and the issue's
    text.
- **What else the judge reads** keys it too: the symptom's file, the tests
  a tests judge reads, the issues an intent judge reads.
- **Not seen**: a commit changing another file the judge read, a caller.
  The verdict stands until the cause's file changes.

## When it happens

After ready, a push's lenses read only the new commits
([after ready](ready.md#after-ready-only-what-is-new)), so a finding
comes back when commits are read again:

- the run on ready again, after a draft;
- every lens again, after a failed run on ready;
- a review not whole, read again at the next push;
- a record that does not read;
- commits a conflict altered in a rebase.

On a machine, no line is kept: every finding is judged.
