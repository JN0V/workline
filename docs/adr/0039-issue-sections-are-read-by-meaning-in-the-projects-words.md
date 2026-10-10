# ADR-0039: An issue's sections are read by meaning, in the project's words; its kind is the forge's

- **Status:** proposed — the questions at the end wait on the maintainer;
  nothing is built before
  [ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)'s
  trial ends
- **Date:** 2026-10-10
- **Builds on:** [ADR-0018](0018-the-product-owner.md) (Need and
  Validation a person's), [ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
  (its trial and counter; the layout of its
  [last amendments](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md#amendment-2026-10-09-last-an-issue-a-person-reads)),
  [#260](https://github.com/JN0V/workline/pull/260) (headings as a
  setting, any language); principles 1, 4, 5, 13
- **Research:** [issue-formats.md](../research/issue-formats.md)

## Context

- **Step 1 held**: the person's part on top, the builder's below a line,
  one real example — rated blind, understood where the old drafts were
  not. Its headings are English and fixed: about twenty places of the
  engine read `Need`, `Verification`, `Validation`, `Scope` by name.
- **workline runs on any repository** — GitHub, GitLab Premium at the
  maintainer's work, Gitea, Forgejo — in any language. Left: (a) a
  project's own issue format; (b) the kinds; (c) headings in the
  project's language; (d) clearer heading names.
- **The forges** ([research](../research/issue-formats.md)): templates
  are files in the repository — GitHub's YAML forms, read by Gitea and
  Forgejo too, and Markdown everywhere. A submitted form is Markdown,
  `### <label>` then the answer. The kind is GitHub's issue type in an
  organization, else a label the template sets. No tool maps a
  template's fields to fixed meanings.
- **Today**: a bug form's "What did you expect?" gets a Validation draft
  beside it, saying the same, for a person to accept. The agent tells a
  bug from its text; GitHub's issue type is never read.

## Decision

### (a) A project's own format: used as is, completed by one setting

- **Used as is**: the project's templates and forms, on every forge. The
  reporter's fields keep their words, order and level, never renamed,
  moved or rewritten; `required`, labels and type stay the forge's.
- **Completed**: a field counts as one of workline's meanings when the
  project names it in `headings`. What a template lacks — most bug forms
  have no Verification nor Scope — is added as today, by the layout, at
  the template's heading level.
- **Replaced**: nothing.
- **Empty**: `_No response_`, as now, and a section holding only an HTML
  comment, a Markdown template's hint.

```yaml
roles:
  product-owner:
    settings:
      # Each meaning, the headings that hold it; the first is the one the
      # role writes. Matched whole, case, spaces and punctuation aside;
      # workline's English headings are always read too.
      headings:
        need: [Need]
        example: [Example]
        steps: [Steps to reproduce]
        validation: [Validation]
        verification: [Verification]
        scope: [Scope]
      # The kind, from words in the issue's type or labels, any case.
      kinds:
        bug: [bug, defect, incident]
        need: [feature, enhancement]
        task: [task, chore, maintenance]
```

### (b) Kinds: the forge's; `ready` the same for all

- **From the forge**: GitHub's issue type, GitLab's `issue_type`
  (`incident` is a bug), else the labels by `kinds` (`type::bug`,
  `Kind/Bug` hold `bug`). The type wins; labels of two kinds give none.
  The engine reads it and gives it to the agent as a fact.
- **None found**: the agent judges from the text, and the role's line
  says "Read as a bug". A person who disagrees sets the project's kind
  label: one click, bulk-able, read the next night.
- **`ready` is the same for every kind**: Need, Validation, Verification,
  Scope; Need and Validation a person's. The kind changes what fills
  them, which the agent writes and judges:

| Kind | Need | Example | Validation |
|---|---|---|---|
| need | who needs what, and why | one real case, today and wanted | named Given / When / Then scenarios |
| bug | who it hurts | Steps to reproduce: steps, Expected, Actual, version | the steps, the expected result as the Then |
| task | what it costs today | the warning, the minutes lost | "Done when": a checklist a person runs |

### (c) The project's language: headings by setting, words by the agent

- **Headings**: `headings`, the shape [#260](https://github.com/JN0V/workline/pull/260)
  gave the documentalist. French: `{need: [Besoin], example: [Exemple],
  steps: [Étapes pour reproduire], verification: [Vérification], scope:
  [Périmètre]}`.
- **The prose and the scenarios' words** follow the issue's language,
  written by the agent as now (Étant donné / Quand / Alors). The engine
  checks no keyword, and guesses no language.

### (d) Heading names: kept

- Need, Example or Steps to reproduce, Validation, the line,
  Verification, Scope: those step 1 was rated with. The line, not a new
  name, tells the person's part from the builder's.
- A project names its own through `headings`; the defaults: question 2.

### A bug and a need, by default and through a project's form

By default ([writing an issue](../../roles/product-owner/docs/writing.md)):

```markdown
## Need
Users who exclude a folder with a trailing slash still upload it: 1.2 GB.
## Steps to reproduce
1. `snapsync push --exclude node_modules/ ~/site`
- Expected: about 40 files. Actual: 18,412 files. Version 2.3.1.
## Validation
- Given `node_modules`, When I run `snapsync push --exclude node_modules/`,
  Then no file under it is uploaded.

---

## Verification
`TestExcludeTrailingSlash` fails on 2.3.1, passes after the fix.
## Scope
`matchExclude` in `internal/filter/glob.go`.
```

Through `.github/ISSUE_TEMPLATE/bug.yml`, `labels: [bug]`, with
`headings: {need: [Need, "What happened?"], steps: [Steps to reproduce,
"How can we reproduce it?"], validation: [Validation, "What did you
expect?"]}`:

```markdown
### What happened?
Excluding `node_modules/` still uploads it: 18,412 files.
### How can we reproduce it?
`snapsync push --exclude node_modules/ ~/site`
### What did you expect?
About 40 files: `node_modules` skipped.
### Version
2.3.1

---

### Verification
…
### Scope
…
```

- The role added the line, Verification and Scope, at the form's level.
  A bug, by the form's label. Need and Validation are the reporter's: at
  `normal`, `ready` when evident, nothing to accept. Without the setting,
  it adds a Need and a Validation repeating the reporter, as drafts.

A need by default, then through `feature.yml`, `type: Feature`, with
`headings: {need: [Need, "Why is this needed?"]}`:

```markdown
## Need
When I close a month's accounts, I want one export for the month,
so I stop merging 30 files by hand.
## Example
Closing September took 30 exports, one a day.
## Validation
- Given 3 sales on 1 Sep and 2 on 30 Sep, When I export "2026-09",
  Then I get one file with 5 sales.

---

## Verification
A test exports a month with sales on two days, reads 5 rows back.
## Scope
`ExportRange` in `src/export/csv.go`.
```

```markdown
### What would you like to be added?
One export for a whole month.
### Why is this needed?
Closing September took 30 exports, one a day, merged by hand.
### Validation
- Given 3 sales on 1 Sep and 2 on 30 Sep, When I export "2026-09",
  Then I get one file with 5 sales.

---

### Verification
…
### Scope
…
```

- The form has no Validation: the role drafts it, a person accepts it
  with `workline:accepted` or edits it. "What would you like to be
  added?" holds no meaning: kept, read by the agent as context.

### What the engine checks, what the agent judges

| The engine, with no agent | The agent |
|---|---|
| which heading holds which meaning (`headings`) | whether what fills it is enough to build from |
| a meaning filled or empty | the kind, when no type nor label gives it |
| the kind, from a type or a label (`kinds`) | the issue's language, the scenarios' words |
| Need and Validation a person's | whether an issue is evident for `ready` |
| where an added section goes, its heading and level | what to ask when a bug has no steps |

- **Without an agent**, headings and kinds are read and `ready` checks
  the meanings; nothing is drafted. No new label nor comment either way.

### Built after the trial, in order

1. **Sections read by meaning**, no behaviour change: `ready`, drafts,
   the role's digests, the reviewer's spec read, changed needs, a
   parent's items; records keyed by heading read as before. The 78
   conformance cases holding today's headings pass unchanged.
2. **`headings`**; an HTML comment alone counted empty.
3. **The kind**: GitHub's issue type and GitLab's `issue_type` read;
   `kinds`; the kind in the task, and in the role's line when judged.
4. **The instruction per kind**: a task's "Done when"; Given / When /
   Then where a person gives an input and sees an output.

Pieces 1 and 2 only if a project running workline needs them
(question 1); 3 and 4 in any case.

### Not now, because…

- **Reading template files** (forms' fields, Markdown templates, GitLab
  Premium's group templates by the API), which one an issue came from,
  a hint left as written counted empty: `headings` covers every forge
  in one line, and no project running workline has its own template yet.
- **A map the agent proposes and the role keeps**, per template and
  language: a new memory, wrong in silence — the grid ADR-0038 cut.
- **Guessing the language**: nobody sees the guess nor undoes it in one
  click.
- **The engine's own lines in another language**: a catalogue per
  language (question 3).
- **A bug's steps required by `ready`**: the kind is often unlabelled,
  and the agent already asks (question 4).
- **The role setting a kind label or type**: a new act with its undo
  (question 5).
- **A question or a doc as a kind**: none in the trial; a doc is a need
  whose person is a reader.
- **A readability score gating `ready`**: reported beside the ratings
  first.

## How it is measured

- **Blind, as step 1**: ten frozen issues — four bugs, three needs,
  three tasks — drafted before and after piece 4, shuffled; the
  maintainer says of each "understood?" and "accepted unchanged?". Kept
  if the new wins on each kind.
- **The kind**: the agent's on labelled issues, labels hidden, against
  the label; no person needed.
- **Pieces 1 and 2**: conformance (French headings; a form's fields
  readied with no draft), then a copy of the project that needs them:
  drafts repeating a reporter's field — none after.
- **In use**: ADR-0038's counter — accepted unchanged, edited, revised,
  set aside — the week after against the trial's. No new counter.

## Alternatives rejected

- **Only what the template asks**: `ready` would mean "the form is
  filled", with no Scope nor Verification in most bug forms, and who
  validates lost (principle 1).
- **workline's format replacing the project's**: it rewrites a
  reporter's fields, where completing them is enough (principle 13).

## Consequences

- An issue written before reads as it did: the English headings are
  always read, no migration.
- A project with a form and `headings` gets no draft repeating its
  reporter.
- Found: workline's own `.github/ISSUE_TEMPLATE/need.yml` still lists the
  sections in the old order; it takes the layout with piece 4.

## Open questions for the maintainer

1. **Your GitLab project at work: has it description templates, and in
   which language are its issues?** Recommended: pieces 1 and 2 only if
   it has either; else they stay parked, this ADR their plan.
2. **Rename the defaults?** Recommended: keep them; to check, six issues
   under two sets of headings rated blind, no token — the other set Need ·
   Example · Accepted when · Tests · Code. The old names are read either way.
3. **The engine's own lines in French?** Recommended: not before the
   work project has run a week with French headings; then a `language`
   setting, English and French, as the documentalist's.
4. **A bug `ready` only with its steps?** Recommended: no; count in use
   the bugs readied without steps.
5. **The role setting the kind label it judged?** Recommended: no; it
   says the kind in its line, and a person's label is one click.
