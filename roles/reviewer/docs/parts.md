---
sources: [roles/reviewer/role.yaml, internal/builtin/reviewer/parts.go, internal/builtin/reviewer/reviewer.go]
checked: 83eff4a0
verified: agent:claude-code
---
# Reviewer — a change too large for one call

Part of [a run on code](run.md), step 4. Why, and what it was measured
on: [ADR-0020's amendment](../../../docs/adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-17-446-a-change-too-large-read-in-parts).

A change past one call — the role's context budget (100,000 tokens) or
`diff-lines-max` (1500 lines of hunks) — is read in parts, as the
documentalist judges a doc too large
([ADR-0009](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md)).
A change that fits is asked in one call, as before.

## What a part reads

- **What every part holds first**: the lenses, what the author says, the
  issue the change closes, and every file the change touches, by name.
- **Then a share of the files**, in the order of their paths, whole or
  by their hunks, as one call reads them
  ([what the lenses read](run.md#4-the-lenses)):
  - packed to fill the budget, room kept for an answer asked again;
  - a file that fits a part of its own is not cut to fill the one
    before;
  - a larger one is cut to fill what is left: between its hunks, or
    within one by lines, each piece with its file's header and a hunk
    header of its own. Every line is read once.
- **Told it is a part**: "part 2 of 2", and not to report as missing
  what another part may hold — a test, a caller.

## What comes of it

- **Each part a call**, named `1-lenses`, `1.2-lenses`; on a merge
  request, the lens in turn is asked in each part.
- **The findings of every part** are found again in the whole change,
  grouped by line and judged as one review's ([the judge](judge.md)).
- **Said**: `read in 2 parts` in the summary; each part's tokens in
  `out/review.json`.
- **A part with no answer**: the lenses failed on its files, said with
  which part (`lens-failed`); the other parts' findings are shown; the
  commits are not recorded as reviewed.
- **The run's tokens**: parts and judges share `ai-max-tokens`; once
  spent, nothing more is asked, said ([the budget](judge.md#the-budget)).

## Past parts-max

- **`parts-max`**, 2 (from 1 to 8): two parts spend at most two budgets,
  about 200k tokens, leaving the judges a third of `ai-max-tokens`.
- **Past it**, no lens is asked and no token spent: `review-not-whole`
  says how many parts the change would take, and to split it. On a
  merge request, the summary comment says so too.
- **`1`**: never in parts; a change past one call is not reviewed whole,
  said.

## What a part may miss

- A defect seen only across two files read in different parts, as a
  claim true only across two sources for the documentalist.
- Each part knows every file by name; the judge reads the code it needs
  ([ADR-0045](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md)).
- The diff-alone facet, off by default, stays one call, cut past
  `diff-lines-max`.
