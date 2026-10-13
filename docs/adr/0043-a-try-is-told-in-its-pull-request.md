# ADR-0043: A try is told in its pull request; a role's status keeps its criteria and links

- **Status:** accepted
- **Date:** 2026-10-13
- **Decided by:** the maintainer, its direction (2026-10-13)
- **Amends:** [ADR-0036](0036-a-roles-status-is-earned-by-written-criteria.md)
  (where a measure is recorded, what a role's status.md holds)
- **Builds on:** [ADR-0018](0018-the-product-owner.md) (work to do is an
  issue); principles 3, 13, 14

## Context

- **tried.md is a log nobody reads whole.** Three roles keep one, every
  try appended: 896 lines for the documentalist, 1,113 for the product
  owner, 806 for the reviewer (2026-10-13).
- **What it holds is already on the forge.** Each pull request's
  description says what was run and verified
  ([#327](https://github.com/JN0V/workline/pull/327), "How it was
  verified"); its tried.md entry tells it again.
- **Every pull request appends at the same place**, the end of the file.
  Of 150 conflict hunks met by people's pull requests in two weeks, 38
  were doc text, 13 of them two additions at the same place
  ([merge conflicts](../research/merge-conflicts.md#peoples-pull-requests)).
  An agent reads the file before it appends, and pays the tokens.
- **status.md mixes one thing nothing else holds with copies.** The
  criteria of [ADR-0036](0036-a-roles-status-is-earned-by-written-criteria.md)
  and their state live only there. Its "Built" proofs repeat the
  conformance cases and tried.md; its "Missing" repeats work to do, which
  [AGENTS.md](../../AGENTS.md) puts in issues. It ran to 424 lines (the
  documentalist) and 220 (the product owner).
- **The forges keep the evidence on the change**: a pull request
  template fills each description
  ([GitHub](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/creating-a-pull-request-template-for-your-repository),
  `.github/pull_request_template.md`;
  [GitLab](https://docs.gitlab.com/user/project/description_templates/),
  `.gitlab/merge_request_templates/Default.md`).

## Decision

- **tried.md is frozen.** A note at its top says it is an archive, with
  no new entry since 2026-10-13, and links here. Its entries stay as they
  are: code comments, ADRs and docs cite them. No new tried.md anywhere.
- **A try for real is told in the pull request's description**, under a
  heading `## Tried`: what was run, where, the numbers with their units,
  and what was not tried. The pull request is the evidence; status.md
  links to it. workline's own pull request template asks for it.
- **status.md keeps only**:
  - the one-line summary;
  - "To leave beta": each criterion of ADR-0036, its state, its proof as
    links — pull requests, issues, cases, ADRs, research;
  - "Built": a short list of what the role does, each with its case
    names or its cases folder; no proof prose;
  - "Missing": links to issues. An item with no issue yet stays one short
    bullet that says so, until an issue holds it.
- **Where a measure lives**: one that decides something (a default, a
  bar, a criterion) in an ADR or docs/research/; a try's numbers in its
  pull request.
- **[ADR-0036](0036-a-roles-status-is-earned-by-written-criteria.md),
  amended**: its third criterion's result is recorded in an ADR or
  docs/research/, linked from status.md, no longer in status.md or
  tried.md.

## Consequences

- **Fewer conflicts and tokens**: a pull request no longer appends to a
  shared log, and an agent reads a short status.md.
- **The evidence of a try lives on the forge**, as the issues do: moving
  to another forge means carrying the pull requests' descriptions, and a
  clone alone no longer holds them. The tries before 2026-10-13 stay in
  the repository.
- **status.md stays written by hand.** Missing derived from labelled
  issues and Built from the cases is a possible later step, not now.
- **The committer's README** keeps its "Tried for real" section, history
  as tried.md is; a new try goes to the pull request there too.
