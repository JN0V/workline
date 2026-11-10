---
sources: [internal/builtin/reviewer/weigh.go, internal/builtin/reviewer/resolve.go, internal/builtin/reviewer/reviewer.go, internal/builtin/reviewer/record.go]
checked: 9b0b2783
verified: agent:claude-code
---
# Reviewer — the tester's "none needed", weighed

Part of [the reviewer](../README.md), beside [the judge](judge.md).

[#1303](https://github.com/JN0V/workline/issues/1303): a finding of the
tests lens goes to [the tester](../../tester/README.md), who adds the
test or answers on the pull request that none is needed, and why.

- **Weighed by a judge**, on the next run, with or without a new
  commit: it is shown the finding, the reason, the code around its
  line and the tests touching its file.
- **The reason holds**: a block goes; a conversation on its line, a
  block's or a warning's, is resolved.
- **It does not hold**: the finding stands; its row says the judge's
  why, run after run.
- **Said**: the summary comment quotes each reason weighed and the
  judge's answer, under **The tester's answers weighed**.
- **Once**: the record keeps each verdict
  (`<!-- workline:weighed … -->`); the same reason is not weighed
  again, a new one is.
- **A test added instead** is a commit like any other: it resolves the
  finding as [a block goes](judge.md#a-block-stands-until-a-commit-answers-it).
- **No agent, or no answer**: nothing is weighed, said; the finding
  stands until a run with an agent.
