# ADR-0066: A role mirrors a human role; AI first where code makes it rigid

- **Status:** accepted (2026-11-09)
- **Date:** 2026-11-09
- **Decided by:** the maintainer, on 2026-11-09, after a three-reviewer
  panel on role boundaries
- **Builds on:** [ADR-0018], [ADR-0054], [ADR-0055], [ADR-0065];
  principles 2, 4, 5, 8
- **Amends:** principles 2, 4, 5 and 8 ([PRINCIPLES.md](../PRINCIPLES.md)),
  now; once built, [ADR-0001]'s
  [2026-09-24 amendment](0001-engine-in-go.md#amendment-2026-09-24)
  (a core role needs nothing but git and the engine) and [ADR-0060] §2
  (the builder's part is the product owner's)

## Context

- **Roles took jobs no human in their post holds.** The product owner
  writes an issue's Verification, how the work is proved, and keeps
  it "at every level" ([ADR-0060] §2); in a team, the tester writes the
  test plan. The developer was fenced by a list of files the product
  owner wrote, until [ADR-0065] made that list a hint.
- **"It works without AI" bent roles into rules.** To run with agent
  `none`, judgement became regexes; [ADR-0065] counts the fixes each new
  language opened in them.
- **The panel's reviews agree**: each misplaced act is one a human in
  the post would hand to a colleague.

## Decision

### 1. A role mirrors a human role

- **Each role holds the job, the craft and the scope of a human role**:
  product owner, tester, developer, reviewer…
- **Outside its scope, a role hands the work back** to the role whose
  scope it is, as a colleague would: it neither does it nor drops it.
- **When unsure what a role should do**, ask what a human holding it
  would do. This keeps roles small, fine-grained and well made.
- **First application**: the issue's Verification moves from the product
  owner to the tester; the product owner keeps the Need, the example,
  the Validation, and the Scope as a hint.

### 2. AI first; code replaces it part by part (temporary)

- **Relaxed, for now**: a role need not work with agent `none` where that
  makes it rigid. Its judgement goes to the AI first.
- **Later, each independent part becomes code** to save tokens, once
  its facts converge ([ADR-0065], rule A).
- **What still holds**:
  - the agent stays a choice: none, `claude`, `cmd:<any command>`;
  - a check on converging facts runs and blocks with no agent;
  - with `none`, a role that needs judgement says what it did not do
    (principle 12), never passes silently.
- **Temporary**: this relaxation is revisited once the roles are stable;
  a later ADR restores or drops it.

## Not chosen

- **Keep principle 5 whole**: every new project kept opening fixes in
  rules that stood in for judgement.
- **One wide role per stage**: fewer roles, each holding several crafts;
  rejected for the reason roles exist (principle 3).

## Consequences and risks

- **More tokens per issue for now**: measured as today; parts move to
  code where they converge.
- **A run with no agent does less**: what it skipped is said where a
  person reads it.
- **Each role's page is re-read against its human post**: misplaced acts
  become issues for the product owner to keep.

[ADR-0001]: 0001-engine-in-go.md
[ADR-0018]: 0018-the-product-owner.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
[ADR-0055]: 0055-the-tester-writes-the-acceptance-tests-before-the-code.md
[ADR-0060]: 0060-the-person-is-on-the-loop.md
[ADR-0065]: 0065-code-only-what-converges-language-knowledge-in-packs.md
