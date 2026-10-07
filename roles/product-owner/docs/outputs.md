---
sources: [roles/product-owner/role.yaml, internal/builtin/productowner, internal/backlog, internal/sample/acts.go]
checked: 7545ece
verified: agent:claude-code
---
# Product owner — what it writes

Part of [the product owner](../README.md). What a run does, step by step:
[run.md](run.md).

## The report

One report issue, "Backlog — product owner":

- **Next**: the first `next-max` ready issues in the backlog's order that
  wait on nothing, never a split need, each with its milestone and
  priority.
- **Stuck**: each issue waiting on a person for more than `stuck-days`,
  with since when:
  - ready with no pull request nor commit naming it since;
  - its reporter not answering;
  - a proposal of the report unticked;
  - an announcement as obsolete past its delay with no second judge.
- Then what was done, what is proposed (a box a person of the project
  ticks; done at the next run), what waits on an open issue, and the split
  needs whose parts are all closed, to accept.
- **Changed needs**: each issue whose Need or Scope a person rewrote, or
  imported lines that changed, with the issues built on it — read again,
  or to check — and what is proposed for each; tick its box once checked.

## On the issues

- **On each issue it reads**:
  - a state comment;
  - labels (`workline:priority/N`, `workline:draft`, `workline:obsolete`,
    `workline:ready`); milestones; sections written;
  - a comment to an outsider reporter; sub-issues or tasks;
  - GitHub's dependencies or GitLab's `is_blocked_by` links (Premium),
    else a line `Blocked by #12.` in the body — a link it set taken off
    once its blocker closes.
- **On a split need**: one comment, edited in place, listing its parts —
  open, closed as completed with the pull request or commit that closed
  it, or not delivered — and each item of its Verification, proved where a
  part delivered quotes it and a test it names is in the code, or not
  proved. Accept the need by closing it.
- `--json`, `--sarif`, `--code-quality` like any role.
- A person accepts drafts with the label `workline:accepted`, on one issue
  or many: the next run moves them to `ready`, with no agent.

## The weekly sample

With the docs' sample (`workline sample --apply`, no agent): one in ten of
the acts it did alone that week, on the issue "workline: the weekly sample
of the product owner's acts".

- Each with its day, the level it was done at, and whether a person undid
  it — for you to judge: undo one you find wrong on its issue.
- From the acts undone it may suggest another level — its report says it
  too, beside the one from the proposals you settled; it never changes the
  setting.
