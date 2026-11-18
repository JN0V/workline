---
sources:
  - internal/builtin/documentalist/kept.go#keptRecord
  - internal/builtin/documentalist/kept.go#keptBody
  - internal/builtin/documentalist/kept.go#readKept
  - internal/builtin/documentalist/kept.go#holdKept
  - internal/builtin/documentalist/kept.go#keepRecords
  - internal/builtin/documentalist/kept.go#keptOn
  - internal/builtin/documentalist/kept.go#stickyOn
  - internal/builtin/documentalist/kept.go#othersKept.findings
  - internal/builtin/documentalist/kept.go#recordsMerged
  - internal/forge/own.go#Kept.Last
  - internal/forge/own.go#github.own
  - internal/forge/own.go#gitlab.own
  - internal/engine/engine.go#pushToMergeRequest
checked: e2e165c2d
verified: agent:claude-code
---
# Documentalist — a doc found with nothing to fix: no commit

Part of [at the push and the adoption](push.md).

On a person's merge request, the agent may find nothing wrong in a doc yet
not vouch for every sentence: its patch only records when (`judged`). That
took a commit on their branch, 41 of 42 in two weeks
([#463](https://github.com/JN0V/workline/issues/463)). Now:

- **Nothing is pushed for it.** Its record is kept, hidden, in the
  documentalist's comment on the merge request: the commit judged, and the
  ids of the doc and its sources there
  ([ADR-0013, amended](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-17-on-a-persons-merge-request-the-record-waits-in-a-comment)).
- **The next push does not judge it again** while the doc and its sources
  stay the same; a change to either has it judged again.
- **Once merged, gardening writes `judged`** on the default branch
  ([records](gardening-merge-requests.md#records-kept-on-merged-merge-requests)); for a
  doc whose sources are too large to go whole, it reads it in parts
  instead ([#462](https://github.com/JN0V/workline/issues/462)).
- **A real fix goes in the comment's diff**, a move of `checked` too.
- **A fork's merge request** keeps its record too.
- **In the comment's diff too**, where the documentalist's comment
  cannot keep the record:
  - a forge that does not say which commit a merge request became
    (`local`, `cmd:`);
  - a comment the job cannot read, said (`record-unread`): on GitLab, give
    the judging job `GITLAB_TOKEN`
    ([ci-gitlab.md](../../../docs/ci-gitlab.md#what-the-template-does));
  - a doc with a source in another repository.
- Only the comment workline's own account wrote is read: the GitHub App,
  else the token's user; on GitLab, `WORKLINE_GITLAB_TOKEN`'s user, or
  one named in `gitlab: {bots}` ([config](../../../docs/config.md)).
  Another's carrying its marker, a person of the project's too, is not:
  its docs are judged again, said once (`record-unread`,
  [ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)).
