# The proposed headings

Part of [the product owner](../README.md). Proposed by
[ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md),
not built: issues are written today as
[writing an issue](writing.md) says. Below, the same issues under the
proposed names.

| Today | Proposed |
|---|---|
| Need | Problem |
| Example | Example |
| Steps to reproduce | Steps to reproduce, with **Expected behavior**, **Actual behavior**, **Version** |
| Validation | Acceptance criteria |
| `---` | `---` |
| Verification | Test plan |
| Scope | Implementation notes |

## A feature

```markdown
## Problem

Accountants closing a month run one export per day and merge 30 files
by hand.

## Example

Closing September took 30 exports, one a day, merged by hand.

## Acceptance criteria

- **One month:** Given 3 sales on 1 Sep and 2 on 30 Sep, When the
  accountant exports "2026-09", Then they get one file with 5 sales.

---

## Test plan

A unit test exports a month with sales on two days and reads 5 rows.

## Implementation notes

`ExportRange` in `src/export/csv.go`.
```

## A bug

```markdown
## Problem

Users who exclude a folder with a trailing slash still upload it: 1.2 GB.

## Steps to reproduce

1. Run `snapsync push --exclude node_modules/ ~/projects/site`.
2. Read the last line of output.

- **Expected behavior:** about 40 files; `node_modules` skipped.
- **Actual behavior:** `uploaded 18412 files (1.2 GB) in 9m02s`.
- **Version:** 2.3.1, Linux, every time.

## Acceptance criteria

- **Trailing slash:** Given a folder `node_modules`, When the user runs
  `snapsync push --exclude node_modules/`, Then no file under it is
  uploaded.

---

## Test plan

`TestExcludeTrailingSlash` fails on 2.3.1 and passes after the fix.

## Implementation notes

`matchExclude` in `internal/filter/glob.go`.
```

- **The person who accepts is named in the When**: the accountant, the
  user.
- **Plain bullets, never `- [ ]` boxes**: a tick edits the issue, which
  the role would read as accepting its draft.

## A bug with no steps

1. The role drafts what it can and does not move the bug to `ready`.
   Its line says: "Ready waits for steps to reproduce; set
   workline:ready to start without them."
2. A person sets `workline:ready`. That night the engine adds
   `workline:needs-repro`, with no comment.
3. Someone writes the steps, or accepts the role's draft of them. The
   engine takes `workline:needs-repro` off.

- A filter on `workline:needs-repro` lists every bug worked on without
  its steps.

## The kind the role sets

- An issue with no type and no kind label gets the project's own label,
  at `normal`. The role's line says: "Labelled bug. Wrong? Change the
  label; it will not be set again."
- At `cautious`, the line only says "Read as a bug."

## An issue under the old names

- An issue with `## Need` and `## Validation` gets `## Verification` and
  `## Scope`, not the new names: an issue keeps the names it uses.
- A person's heading is never renamed.

## Through a project's own form

A bug form asks "What happened?", "How can we reproduce it?", "What did
you expect?" and "Version". The project adds its labels to the names
read:

```yaml
roles:
  product-owner:
    settings:
      headings: {need: ["What happened?"], steps: ["How can we reproduce it?"],
        validation: ["What did you expect?"]}
```

- The reporter's fields stay as they are, under `###`.
- The role adds the line `---`, then `### Test plan` and
  `### Implementation notes`.
- Problem, Steps to reproduce and Acceptance criteria are the
  reporter's, so the bug may be readied with nothing to accept.
