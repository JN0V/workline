# ADR-0039: An issue's sections are read by meaning, in the project's words; its kind is the forge's

- **Status:** proposed — the questions at the end wait on the maintainer;
  built after the trial of [ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
- **Date:** 2026-10-10
- **Builds on:** [ADR-0018](0018-the-product-owner.md) (Need and
  Validation a person's), the layout of
  [ADR-0038's last amendments](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md#amendment-2026-10-09-last-an-issue-a-person-reads),
  [#260](https://github.com/JN0V/workline/pull/260) (headings as a
  setting); principles 1, 4, 5, 13
- **Research:** [issue-formats.md](../research/issue-formats.md)

## Context

- **Step 1 held**: the person's part on top, the builder's below a line,
  one real example — rated blind, understood where the old drafts were
  not. Its headings are English and fixed: about twenty places of the
  engine read `Need`, `Verification`, `Validation`, `Scope` by name.
- **workline runs on any repository**, GitHub, GitLab, Gitea or Forgejo,
  in any language. Left: (a) a project's own issue format; (b) the
  kinds; (c) headings in its language; (d) clearer heading names.
- **The forges** ([research](../research/issue-formats.md)): templates
  are files in the repository, GitHub's YAML forms read by Gitea and
  Forgejo too, Markdown everywhere; a submitted form is Markdown,
  `### <label>` then the answer. The kind is GitHub's issue type in an
  organization, else a label the template sets.
- **Today** a bug form's "What did you expect?" gets a Validation draft
  saying the same, for a person to accept; GitHub's issue type is unread.

## Decision

### (a) A project's own format: used as is, completed by one setting

- **Used as is**: the project's templates and forms, on every forge. The
  reporter's fields keep their words, order and level, never renamed,
  moved or rewritten.
- **Completed**: a field counts as one of workline's meanings when the
  project names it in `headings`. What a template lacks — most bug forms
  have no Verification nor Scope — is added as today, at its level.
  Nothing is replaced.
- **Empty**: `_No response_`, as now, or only an HTML comment, a
  Markdown template's hint.

```yaml
roles:
  product-owner:
    settings:
      # Each meaning, the headings holding it, the first one written; matched
      # whole, case, spaces and punctuation aside; the English ones always read.
      headings: {need: [Need], example: [Example], steps: [Steps to reproduce],
        validation: [Validation], verification: [Verification], scope: [Scope]}
      # The kind, from words in the issue's type or labels, any case.
      kinds: {bug: [bug, defect, incident], need: [feature, enhancement],
        task: [task, chore, maintenance]}
```

### (b) Kinds: the forge's; `ready` the same for all

- **From the forge**: GitHub's issue type, GitLab's `issue_type`
  (`incident` a bug), else the labels by `kinds` (`type::bug`,
  `Kind/Bug` hold `bug`). The type wins; labels of two kinds give none.
- **None found**: the agent judges from the text; the role's line says
  "Read as a bug". A person who disagrees sets the kind label: one
  click, bulk-able, read the next night.
- **`ready` is the same for every kind**: the four sections, Need and
  Validation a person's. The kind changes what fills them, which the
  agent writes and judges:

| Kind | Need | Example | Validation |
|---|---|---|---|
| need | who needs what, and why | one real case, today and wanted | named Given / When / Then scenarios |
| bug | who it hurts | Steps to reproduce: steps, Expected, Actual, version | the steps, the expected result as the Then |
| task | what it costs today | the warning, the minutes lost | "Done when": a checklist a person runs |

### (c) The project's language: headings by setting, words by the agent

- **Headings**: `headings`, the shape
  [#260](https://github.com/JN0V/workline/pull/260) gave the
  documentalist; French: `need: [Besoin]`, `scope: [Périmètre]`…
- **The prose and the scenarios' words** follow the issue's language,
  written by the agent as now (Étant donné / Quand / Alors). The engine
  checks no keyword, and guesses no language.

### (d) Heading names: kept

- Need, Example or Steps to reproduce, Validation, the line,
  Verification, Scope: those step 1 was rated with. The line, not a new
  name, tells the person's part from the builder's. A project names its
  own through `headings`; the defaults are question 2.

### A bug and a need, by default and through a project's form

The default writes `##` headings, in the order of
[writing an issue](../../roles/product-owner/docs/writing.md). The form
writes `###` and the reporter's labels. The bug's form, `bug.yml`, sets
`labels: [bug]`; the project sets `headings: {need: [Need, "What
happened?"], steps: [Steps to reproduce, "How can we reproduce it?"],
validation: [Validation, "What did you expect?"]}`.

| Bug | By default | Through `bug.yml` |
|---|---|---|
| Need | Users who exclude a folder with a trailing slash still upload it: 1.2 GB. | "What happened?" — Excluding `node_modules/` still uploads it: 18,412 files. |
| Steps to reproduce | `snapsync push --exclude node_modules/ ~/site`; Expected: about 40 files; Actual: 18,412; version 2.3.1 | "How can we reproduce it?" — the same command; "Version" — 2.3.1, read as context |
| Validation | Given `node_modules`, When I run that command, Then no file under it is uploaded. | "What did you expect?" — About 40 files: `node_modules` skipped. |
| `---` | the line | added by the role |
| Verification | `TestExcludeTrailingSlash` fails on 2.3.1, passes after the fix. | the same, added by the role |
| Scope | `matchExclude` in `internal/filter/glob.go` | the same, added by the role |

- Through the form, Need and Validation are the reporter's: at `normal`,
  `ready` when evident, nothing to accept. Without `headings`, two
  drafts repeat them, for a person to accept.

The need's form, `feature.yml`, sets `type: Feature`; the project sets
`headings: {need: [Need, "Why is this needed?"]}`.

| Need | By default | Through `feature.yml` |
|---|---|---|
| Need | When I close a month's accounts, I want one export for the month, so I stop merging 30 files. | "Why is this needed?" — Closing September took 30 exports, merged by hand. "What would you like to be added?" — read as context. |
| Example | Closing September took 30 exports, one a day. | — |
| Validation | Given 3 sales on 1 Sep and 2 on 30 Sep, When I export "2026-09", Then I get one file with 5 sales. | the same, drafted by the role: the form has none |
| `---`, Verification, Scope | A test exports a month with sales on two days, reads 5 rows; `ExportRange` in `src/export/csv.go` | the same, added by the role |

### What the engine checks, what the agent judges

| The engine, with no agent | The agent |
|---|---|
| which heading holds which meaning (`headings`) | whether what fills it is enough to build from |
| a meaning filled or empty | the kind, when no type nor label gives it |
| the kind, from a type or a label (`kinds`) | the issue's language, the scenarios' words |
| Need and Validation a person's | whether an issue is evident for `ready` |
| where an added section goes, its heading and level | what to ask when a bug has no steps |

### Built after the trial, in order

1. **Sections read by meaning**, no behaviour change: `ready`, drafts,
   digests, the reviewer's spec read, changed needs, a parent's items.
   The 78 conformance cases holding today's headings pass unchanged.
2. **`headings`**; an HTML comment alone counted empty. Pieces 1 and 2
   only if a project running workline needs them (question 1).
3. **The kind**: GitHub's and GitLab's issue types read; `kinds`; the
   kind in the task, and in the role's line when judged.
4. **The instruction per kind**: a task's "Done when"; Given / When /
   Then where a person gives an input and sees an output.

### Not now, because…

- **Reading template files** (forms, Markdown, GitLab's group templates),
  which one an issue came from, a hint left as written counted empty:
  `headings` covers every forge in one line; no project has asked.
- **A map the agent proposes and the role keeps**: wrong in silence.
- **Guessing the language**: nobody sees the guess nor undoes it; the
  engine's own lines in another language are question 3.
- **A bug's steps required by `ready`**: the kind is often unlabelled,
  and the agent already asks (question 4).
- **The role setting a kind label**: a new act with its undo (question 5).
- **A question or a doc as a kind**: none seen; a doc is a reader's need.
- **A readability score gating `ready`**: reported beside ratings first.

## How it is measured

- **Blind, as step 1**: ten frozen issues — four bugs, three needs,
  three tasks — drafted before and after piece 4; the maintainer says of
  each "understood?", "accepted unchanged?". Kept if the new wins on each kind.
- **The kind**: the agent's, labels hidden, against the label.
- **Pieces 1 and 2**: conformance, then a copy of the project needing
  them: drafts repeating a reporter's field — none after.
- **In use**: the trial's counter — accepted unchanged, edited,
  revised, set aside — the week after against the trial's. No new one.

## Alternatives rejected

- **Only what the template asks**: `ready` would mean "the form is
  filled", with no Scope nor Verification in most bug forms, and who
  validates lost (principle 1).
- **workline's format replacing the project's**: it rewrites a
  reporter's fields, where completing them is enough (principle 13).

## Consequences

- An issue written before reads as it did: no migration.
- Without an agent, headings and kinds are read and `ready` checks the
  meanings; nothing is drafted. No new label nor comment.
- Found: workline's own `.github/ISSUE_TEMPLATE/need.yml` still lists the
  sections in the old order; it takes the layout with piece 4.

## Open questions for the maintainer

1. **Your GitLab project at work: has it description templates, and in
   which language are its issues?** Recommended: pieces 1 and 2 only if
   it has either; else they stay parked, this ADR their plan.
2. **Rename the defaults?** Recommended: keep them; to check, six issues
   under two sets of headings rated blind, no token — the other set
   Need · Example · Accepted when · Tests · Code.
3. **The engine's own lines in French?** Recommended: after a week of
   French headings at work, a `language` setting, as the documentalist's.
4. **A bug `ready` only with its steps?** Recommended: no; count in use
   the bugs readied without steps.
5. **The role setting the kind label it judged?** Recommended: no; it
   says the kind in its line, and a person's label is one click.
