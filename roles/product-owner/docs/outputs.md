---
sources: [roles/product-owner/role.yaml, internal/builtin/productowner, internal/backlog, internal/sample/acts.go]
checked: 47ed51e
verified: agent:claude-code
---
# Product owner — what it writes

Part of [the product owner](../README.md). What a run does, step by step:
[run.md](run.md).

## On each issue

- **One comment**, created once, then edited in place
  ([ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)):
  - what it did there last, and when;
  - what it proposes, in plain words; drafts it proposes, folded;
  - what it wants from you, while the issue waits on you;
  - set aside, when you took its label off;
  - what it knows of the issue, folded: what it read, wrote, did,
    proposed, and what you undid.
- **Labels**: `workline:proposed` (GitLab: `workline::proposed`) while it
  waits on you; `workline:draft`, `workline:to-refine`, `workline:ready`.
- **Sections** written in the body, Need and Validation as drafts; to an
  outsider, a comment proposing them instead.
- **A reply in one line** to your comments, 👀 on each.
- **On a split need**: one comment listing its parts and what they
  proved ([tracking](tracking.md#a-parent-and-its-parts)).

No line of its own text starts with `/`: GitLab would run it as a quick
action, on a comment or an edit.

## In the CI job's summary

Each night ([ADR-0035](../../../docs/adr/0035-the-engine-writes-the-jobs-summary.md)),
read-only, one line an issue:

- `done`, `done-as-accepted`: what it did alone, and on your yes;
- `proposed`: what waits on you;
- `set-aside`, `left-to-a-person`: your "not now", the rounds spent;
- `next-ready`: the first ready issues of the order;
- `stuck`: what waits on a person past `stuck-days`.

`--json`, `--sarif`, `--code-quality` like any role.

## Where to look

- **What waits on you**: a saved filter on `workline:proposed` (GitLab: a
  label subscription or a board works too).
- **What it did**: the job's summary, and the comment on each issue.

## The weekly sample

Off by default (`weekly-sample`). Turned on, with the docs' sample
(`workline sample --apply`, no agent): one in ten of the acts it did alone
that week, read from each issue's state, on the issue "workline: the
weekly sample of the product owner's acts", each with its day, level and
whether a person undid it; a level suggested, never set.
