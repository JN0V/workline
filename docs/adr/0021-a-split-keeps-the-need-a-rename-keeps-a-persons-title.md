# ADR-0021: A split keeps the need in its parent; a rename keeps a person's title

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 (the product owner; "opens issues, and splits a
  need into technical tasks"; one way to open issues), docs/spec/routing.md
  (four sections, one Verification an issue), principles 1, 4, 6

## Context

ADR-0018 lets the product owner split a need and rename an issue alone,
and says nothing of how. It is over its length already, and these are two
acts with their own guards, so they get their own record rather than a
third amendment. docs/research/product-owner.md ("Splitting and
renaming") found GitHub's sub-issues writable with the token a role holds,
GitLab's children of an issue to be tasks — another work item type, not an
issue —, a task list in the body rendered on both, and backlog practice
splitting what is not *Small* or *Testable* (INVEST, SPIDR).

## Decision

**Split** (`split`): an issue too big to be one need — several things,
each proved by its own Verification — is broken into 2 to 6 children.

- **The parent keeps the need** it was written for: its body is not
  rewritten, nor closed, nor taken out of the order. Its state records its
  children (`split: [13, 14]`); one with children is not split again
  (`already-split`), whatever the agent proposes at a later run.
- **Each child is a need of its own**: a title and the four sections of
  the issue form (ci/github/issue-form-need.yml), all four required;
  Verification and Scope the role's, Need and Validation drafts a person
  accepts, as refining writes them (`workline:draft`, `workline:accepted`).
  Its first line says it is part of the parent.
- **Opened through the one way** (`backlog.Openings`), keyed by the parent
  and the child's title: a run stopped half-way, run again, finds the
  children it opened and does not open them twice. They are the role's
  own breakdown, not a finding: no `needs-triage`, not counted in
  `issues-max`; the cap is the act's, splits a run.
- **Linked where the forge links**: a native sub-issue on GitHub; a task
  list under `## Sub-issues` in the parent's body elsewhere — GitLab,
  whose children of an issue are tasks the line does not read, the local
  forge, a forge plugged by a command that says it has none.
- **It moves nothing**: the parent keeps its milestone and priority; the
  children are new issues, never read yet, read and ordered at the next
  run within its moved share. A split is not counted in that share.

**Rename** (`rename`): the title alone, one line, 120 characters at most —
Mozilla's rule of thumb: about ten words that tell this issue from any
other, the problem, not the fix. The body is never touched.

- **A person's title is kept**: the state records the title the role set
  (`title`). The title an issue was opened with is its reporter's words
  and may be renamed once; after the role's, a title other than the one
  recorded is a person's, and the act is dropped (`title-kept`), as
  `priority-kept` and `section-kept` do. The forge's history of renames is
  not read: no forge but GitHub keeps one.

**For both**, as for the other acts: a mode and a cap per run
(`split: {mode: act, max: 2}`, `rename: {mode: act, max: 5}`); the issue's
state must read; no quote — they say nothing of an issue's truth. An
issue opened by someone without write access is theirs: its split or
rename is proposed, not done, as moving it to ready (ADR-0018). The
report says each act done and how to undo it: a rename, the title before,
to set back; a split, the children's titles, to close — the parent was
left as it was, but for its task list.

## Consequences

- A need too big for one change reaches the developer as parts, each
  ready on its own; the parent shows the whole and its progress where the
  forge counts sub-issues.
- On GitLab, the children are issues listed in the parent, not tasks: a
  GitLab project sees no progress bar. Writing tasks waits for the line to
  read work items.
- Not built: a child moved to another parent, or a split undone by the
  role; a parent's own readiness while its children are open.
