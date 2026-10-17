# The proposed headings

Part of [the product owner](../README.md). Proposed by
[ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md),
built in parts ([#435](https://github.com/JN0V/workline/issues/435)):
each thing said once is built, under today's names
([writing an issue](writing.md#each-thing-said-once)); the names, a bug's
steps and the kind are not yet. Below, the same issues under the
proposed names, each thing said once.

| Today | Proposed |
|---|---|
| Need | Problem |
| Example | Example |
| Steps to reproduce | Steps to reproduce, with **Expected behavior**, **Actual behavior**, **Version** |
| Validation | Acceptance criteria |
| `---` | `---` |
| Verification | Test plan: the unit, integration and end-to-end tests that prove it |
| Scope | Implementation notes |

## A feature

```markdown
## Problem

Accountants closing a month cannot export it as one file: the export
takes one day at a time.

## Example

Closing September 2026: `shopctl export --day 2026-09-01`, then 29 more
days, 30 CSV files pasted into one sheet, two hours.

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

Users who exclude a folder with a trailing slash still upload all of it.

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

## Each thing said once

[#103](https://github.com/JN0V/workline/issues/103), as a night of the
role left it, said its problem three times:

| Text | Words already said above |
|---|---|
| On top, the import's: "`workline setup` cannot install lychee … It only names the lychee page." | — |
| Need: "A Linux user without brew or cargo needs `workline setup` to install lychee" | 60% |
| Example: "`workline setup` prints the lychee page and stops. The user installs it by hand." | 53% |

Under the amendment, the text on top becomes the Problem, the builder's
part unchanged below the line; no real case is known, so no Example,
and no question: a case would not change what is built.

```markdown
Opened from `docs/BACKLOG.md`, lines 157 to 158 by the product-owner role.

## Problem

On Linux without brew or cargo, `workline setup` cannot install lychee,
the link checker: it names the lychee page and stops.

## Acceptance criteria

- **No brew, no cargo:** Given a Linux machine with neither, When the
  user runs `workline setup`, Then lychee is installed from a pinned
  release binary.

---
```

- **A person's text on top** is the Problem when no Problem section
  follows: the role adds Acceptance criteria and the builder's part,
  and no Problem restating it.
- **An Example kept** adds a fact: the feature's above shares 17% of its
  words with its Problem, the bug's steps 3%. Past 40%
  (`repeated-percent-max`), a draft is not written.

## A bug with no steps

1. The role drafts what it can and does not move the bug to `ready`.
   Its line says: "Ready waits for steps to reproduce; set
   workline:ready to start without them."
2. A person sets `workline:ready`. That night the engine adds
   `workline:needs-repro`, with no comment.
3. Someone writes the steps, or accepts the role's draft of them. The
   engine takes `workline:needs-repro` off; also when the kind stops
   being bug, or `ready` comes off.

- A filter on `workline:needs-repro` lists every bug worked on without
  its steps; past `stuck-days`, the summary lists it as stuck.
- Taken off by a person, it is never set again on that issue.
- One colon on GitLab: `workline::needs-repro` would replace
  `workline::proposed`.

## The kind the role sets

- An issue with no type and no kind label gets the project's own label,
  at `normal`. The role's line says: "Labelled bug. Wrong? Change the
  label; it will not be set again."
- At `cautious`, the line only says "Read as a bug."
- **Never a swap**: on GitLab Premium a scoped label would silently
  replace a person's.
- **Only `task` is created** when the project has no such label. GitLab
  gets labels only, never `issue_type: task`, which makes the issue a
  child work item.
- **GitHub issue types are read, never set**: on a personal account's
  repository the API answers both ways, and a type set without push
  access is silently dropped.
- **What it set is kept in the issue's state**: the Forgejo sample forge
  reports no label events.
- **Issues already read**: one light read for the kind alone, about 1k
  tokens an issue, within `issues-per-run`.
- **Notifications**: none on GitHub; GitLab mails a label's subscribers,
  and workflows a label triggers run: hence the cap.

```yaml
# roles.product-owner.settings: the project's label per kind, the first that exists
kinds: {bug: [bug, "type::bug", Kind/Bug], feature: [enhancement, "type::feature",
  Kind/Feature], task: [task, "type::maintenance"]}
acts: {kind: {mode: act, max: 10}}
```

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
