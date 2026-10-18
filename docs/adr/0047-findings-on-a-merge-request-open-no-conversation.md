# ADR-0047: Findings on a merge request open no conversation

- **Status:** accepted
- **Date:** 2026-10-15
- **Decided by:** the maintainer, option 1 of four
  ([#412](https://github.com/JN0V/workline/issues/412#issuecomment-6186855318))
- **Builds on:** [ADR-0010](0010-docs-judged-on-the-merge-request-and-by-gardening.md)
  (judge, then apply), [ADR-0044](0044-the-apps-key-stays-with-the-default-branchs-runs.md)
  (the write token with the base's runs); principles 12, 13

## Context

- **Code scanning opens a conversation per new alert** on a line the pull
  request touched, whatever its SARIF level; a base requiring
  conversations resolved holds the merge on them:
  [DomoticsCore#211](https://github.com/JN0V/DomoticsCore/pull/211), 36
  of them, every check green, `BLOCKED`.
- **Nothing in GitHub turns them off**: not the level, not an upload to
  the branch's ref, no setting
  ([tried](../research/ci-and-forge.md#findings-and-conversations-to-resolve-2026-10-15)).
- **A check's annotations open none**: shown beside the line, in the
  changed files; a GitHub App's token with `checks: write` writes them.
- **The reviewer's line findings came twice**: its own review, and a
  code scanning conversation each.

## Decision

- **On a merge request, the SARIF leaves out the documentalist's and the
  reviewer's findings** (`--sarif`, event `merge-request`). Code
  scanning keeps the gates' and the committer's: security findings, a
  person's to resolve.
- **The applying job shows the documentalist's findings** as the
  annotations of the check `workline findings`, on the head judged,
  completed and neutral (`apply --line --from-merge-request`). It places
  them on the head's files; the run's findings are data.
- **A refused check is said** (`findings-not-annotated`), never an error:
  the findings stay in the judge's job summary.
- **`workline doctor` reads the rule**: the default branch's rulesets,
  which any token reads, then its classic protection, which only an
  admin's reads, said when it could not.
- **GitLab is unchanged**: its Code Quality report is a report, not a
  thread.

## Consequences

- The GitHub App needs one more permission, **Checks**, read and write;
  the templates give `apply` `checks: write`.
- These findings leave the Security tab. The templates only ever uploaded
  pull requests there, never the default branch.
- The judge uploads its line's result when the documentalist found
  something, though nothing was proposed, so that `apply` runs.
- **Amended by [ADR-0049](0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md)**
  (8a): the gates' findings leave code scanning on a merge request too.
