---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/opening.go, internal/forge/forge.go, internal/engine/backlog.go]
status: draft
checked: 18109269
verified: agent:claude-code
---
# Splitting and renaming

Part of [Acts on the backlog](../backlog-acts.md): breaking an issue into its parts, and setting its title.

## Splitting

```yaml
- split:
    issue: 12
    into:                                   # 2 to 6 children
      - title: "Keep the last row in the CSV export"
        need: "Every row exported, so a report counts what was sold."   # a draft
        verification: "A test writes three rows and reads three lines back."
        validation: "The maintainer opens an export of a known day."    # a draft
        scope: "WriteRows in src/export/csv.go."
        sources: [src/export/csv.go]
      - title: "Quote commas in exported cells"
        ...
    why: "Two needs, each proved by its own test."
```

An issue too big to be one need is broken into its parts (ADR-0022):

- **Checked**: 2 to 6 children (`split-size`), each a title of one line,
  120 characters at most, no two alike (`split-title`), its four sections
  written (`split-sections`), its Scope's files 1 to 5, files of the
  commit (`sources-unknown`); the issue's state must read. One whose state lists
  children already is never split again (`already-split`); nor twice in one
  run, nor renamed twice: the first that passes its check is kept
  (`once-a-run`).
- **Each child** is opened through the one way ([Opening issues](opening.md)), keyed by
  the parent and its title, case and spaces aside
  (`split=<parent>/<8 hex>`): a line `Part of #12.`, its four sections — Need and Validation as drafts, as `refine`
  writes them —, "Opened from #12 by the product-owner role."; the labels
  `workline:to-refine` and `workline:draft`; its own state, its sources,
  confirmed at the commit. Not `needs-triage`, not counted in
  `issues-max`. A child closed already is left closed (`issue-closed`),
  and still counted among the parent's children.
- **Linked to its parent** (`AddSubIssue`): a sub-issue on GitHub; on
  GitLab a task — converted and given its parent through GraphQL, read
  and acted on by REST afterwards as any issue; elsewhere, or when the
  forge refuses, a task list in the parent's body, under `## Sub-issues`
  (`- [ ] #13`), after its text, which stays.
- **The parent** keeps its need, milestone and priority; its state gets
  the children (`split`), last — a run stopped half-way, resumed, finds the
  children it opened — and the digest of its body when the engine listed
  them there, so it is not read again for its own change. A split moves
  nothing: it is not counted in the moved share; the children, never read
  yet, are read and ordered at the next run.

## Renaming

```yaml
- rename: {issue: 12, title: "The CSV export drops the last row", why: "\"bug\" says nothing of the problem."}
```

Sets the title alone, one line, 120 characters at most (`rename-title`);
the title it has already is dropped (`title-same`). The issue's state
records the title set. The title an issue was opened with is its
reporter's and may be renamed; after the role's, a title other than the
one recorded is a person's: the act is dropped (`title-kept`), the task
shows it as a person's. No quote; the state must read.

Split and rename on an issue opened by someone without write access are
proposed, not done (`reporter-outside`), as moving it to ready — to the
project, on the issue: the text of a need is the reporter's to agree to
([The conversation with the reporter](conversation.md)), how the backlog cuts and names it
the project's. The label `workline:accepted` lifts all three.
