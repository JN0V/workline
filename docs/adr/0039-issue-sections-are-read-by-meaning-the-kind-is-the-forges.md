# ADR-0039: Issue sections take established names, read by meaning; a bug waits for its steps; the role sets the kind

- **Status:** proposed — built in parts ([#435](https://github.com/JN0V/workline/issues/435)):
  piece 1 and the amendment's repeat rule built; the question at the end
  settled on 2026-10-17
- **Date:** 2026-10-10, revised 2026-10-11, amended 2026-10-11 (below)
- **Builds on:** [ADR-0018](0018-the-product-owner.md) (Need and
  Validation a person's), [ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
  (acts, proposals, a person's undo), [#260](https://github.com/JN0V/workline/pull/260)
  (headings a project names); principles 1, 4, 5, 7, 13
- **Research:** [issue-formats.md](../research/issue-formats.md);
  worked examples: [the proposed headings](../../roles/product-owner/docs/proposed-headings.md)

## Context

- **Step 1 held**: the person's part on top, the builder's below a line,
  rated blind. Its headings are fixed: about twenty places of the engine,
  and each issue's state, name Need, Verification, Validation, Scope.
- **The maintainer decided, on 2026-10-11**:
  1. the headings take the best established names, none invented:
     Validation and Verification read badly;
  2. a bug without steps to reproduce is never readied by the role; a
     person may force it, and the issue then shows what still waits;
  3. the role sets the kind label itself.

## Decision

### (a) Section names, borrowed

| Meaning | Written | Also read | Borrowed from |
|---|---|---|---|
| need | Problem | Need | GitHub's built-in feature template (~49.5k template files), "Problem statement" (~17k), GitLab's "Problem to solve" |
| example | Example | — | TypeScript's "Motivating Example" |
| steps (a bug) | Steps to reproduce, holding **Expected behavior**, **Actual behavior**, **Version** as bold lines | — | ~185k, ~176k, ~66k files; 8 of 21 large bug forms word for word |
| validation | Acceptance criteria | Validation | ~39k files; an Azure DevOps field; Atlassian, GOV.UK; Dan North's story template |
| verification | Test plan | Verification | Kubernetes enhancement proposals, Phabricator |
| scope | Implementation notes | Scope | ~2.4k files; GitLab's "Implementation plan" |

- **Acceptance criteria**: named Given / When / Then scenarios, the
  person who accepts named in the When; a task's, "done when" items.
  Never `- [ ]` boxes: a tick edits the body, read as accepting a draft.
- **Why the old names go**: in trackers, verification is a person
  re-checking a fix, validation is checking a problem is real, and
  "Scope" is what is in or out of the work.
- **Kept**: the order, and the line `---` between the two parts.

### Old names, read forever

One table maps each meaning to its headings: the first is written, all
are read; the ids (`need`, `example`, `steps`, `validation`,
`verification`, `scope`) never change.

- A person's heading is never renamed. An issue keeps the set of names it
  uses: a section missing from an old issue gets the old name, so the
  two sets never mix.
- The names in each issue's state (`wrote`, `deleted`, `sections`) are
  read by meaning: a section a person deleted under an old name is never
  written again under the new one.
- An old "Scope" or "Verification" counts only beside another workline
  section or the role's marker: a project's own "Scope", or GitLab's
  "Verification steps", is not taken for workline's.
- Matched whole, case, spaces, emoji and a final ":" or "?" aside.
- **A project's own template**: `headings` lets its labels ("What
  happened?") count as a meaning, built when a project has one. English
  only: another language needs every engine line, case and measure again.

### (b) A bug is readied by the role only with its steps

- **For a bug, `ready` also needs Steps to reproduce**, a person's or
  accepted by one, as Problem and Acceptance criteria are; the role may
  draft them from the issue's words. Any reproduction counts: steps, a
  command, code, a failing test, a link. The engine checks it is there
  and whose; the agent judges that it reproduces.
- **The reviewer writes its finding's input as the steps**: its issues,
  opened by a bot, have no reporter to ask.
- **The kind must be known**: the role's `ready` states it; with none,
  no `ready`. The engine drops a role's `ready` on a bug without a
  person's steps, at every level.
- **A person forces it** by setting `workline:ready`; `workline:accepted`
  accepts drafts and does not force. The role's line says how.
- **The waiting mark, `workline:needs-repro`**, beside `workline:ready`:
  two independent labels, as Go (`NeedsFix`, `WaitingForInfo`) and
  Gitea (`Reviewed/Confirmed`, `Status/Need More Info`) do. workline's
  prefix: a project's own "needs reproduction" label may drive a bot
  that closes the issue (Next.js after 2 days). Set and cleared by the
  engine alone, no agent nor comment; it closes nothing
  ([its life](../../roles/product-owner/docs/proposed-headings.md#a-bug-with-no-steps)).

### (c) The role sets the kind label

- **A new act, `kind`**: bug, feature or task, only on an issue with no
  type and no kind label, checked again before applying; it never
  removes nor swaps a label. `normal` sets it, 10 a run at most;
  `enterprising`, 20; `cautious` says it in its line only.
- **The project's own label**, the first of `kinds` that exists; GitHub
  issue types read, never set; issues already read get one light read
  ([how, and the settings](../../roles/product-owner/docs/proposed-headings.md#the-kind-the-role-sets)).
- **A person's correction holds**: a kind a person removed or changed is
  never set again on that issue, as Go's bot does; `undone-max` stops the
  act everywhere when it is often wrong.

### What the engine checks, what the agent judges

| The engine, with no agent | The agent |
|---|---|
| which heading holds which meaning, old or new | whether what fills it is enough to build from |
| a meaning filled, and whose | whether the steps reproduce the bug |
| the kind from a type or a label | the kind, when neither gives it |
| a role's `ready` with no kind, or on a bug with no person's steps, dropped | whether an issue is evident for `ready` |
| `workline:needs-repro` set and cleared | what to ask when a bug has no steps |
| a draft that repeats what is above it, dropped (amended below) | whether an Example adds a fact |

### Built after the trial, in order

1. **Sections read by meaning**, no behaviour change. Cases first: old,
   new and mixed names give the same `ready`; a section deleted under an
   old name stays deleted; the 78 existing cases pass unchanged.
2. **The new names**, in one change: the table, the instruction, the
   form templates, the specs.
3. **The kind**: types read, the `kind` act, the light read.
4. **The bug rule** and `workline:needs-repro`.
5. **The instruction per kind**.

### Not now, because…

- Setting GitHub issue types (see (c)); reading template files
  (`headings` covers it); a generic waiting label or "Open questions"
  section (a section to fill invites filler); a timer that closes (a
  person forces a bug); a developer role reading `needs-repro`.

## How it is measured

- **The names, blind, no token**: six frozen issues under the old and new
  names; the maintainer says which section he checks to accept, which
  the builder writes. In use: Acceptance criteria drafts accepted
  unchanged, against Validation's.
- **The kind**: 9 of 10 right, labels hidden; 10% or fewer changed by a
  person. **Bugs**: none readied by the role without steps (a case);
  open `needs-repro` issues and their age.
- **The instruction per kind**: ten issues before and after, rated blind.

## Alternatives rejected

- **Only what a template asks**: `ready` would mean "the form is
  filled", who validates lost (principle 1).
- **A kind set again each run**, as a labeler on every edit: a person's
  correction undone each night.
- **A project's own "needs reproduction" label**, **multilingual
  support**: see (b) and "Old names".

## Consequences

- Without an agent, the names, the forge's kinds, the bug rule and the
  waiting mark hold; nothing is drafted, no kind judged.
- workline's own `.github/ISSUE_TEMPLATE/need.yml` takes the new names
  with piece 2.

## Open question for the maintainer, settled

1. **Implementation notes, or Implementation details?** Settled on
   2026-10-17 by the maintainer: Implementation notes, as recommended;
   details (~3.3k files) invites a design the role must not invent.

## Amendment (2026-10-11): each thing said once

[#103](https://github.com/JN0V/workline/issues/103) said its problem
three times nearly word for word: an import's text on top, rewritten in
plain words ([ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md),
amended 2026-10-10), the Need, and an Example restating it, no real
case known ([worked case](../../roles/product-owner/docs/proposed-headings.md#each-thing-said-once)).

- **A role's text on top becomes the Problem**: rewritten as that draft,
  taken from the top; the origin line and the markers stay.
- **A person's text on top, with no Problem section, is the Problem**,
  a person's: the engine reads it so, no agent, nothing recorded. No
  draft restates it; one is drafted, read instead, only when that text
  states no problem. A person's own `## Problem` always wins. Rejected:
  the agent's word kept in the state, a presence check resting on a
  claim the person cannot see.
- **An Example only with a fact** the Problem lacks: a command, a real
  output, a number, a name. No real case: no Example; a question to the
  reporter only if a case would change what is built.
- **The engine drops a draft that repeats**, no agent: containment, the
  share of its words already said above it, over one-word shingles,
  common words out ([research](../research/issue-formats.md#a-draft-that-repeats)).
  Compared: a Problem draft with a person's text on top; an Example or
  Steps to reproduce with all above; never Acceptance criteria, which
  restate the case by design. Past `repeated-percent-max` (40 by
  default), not written; the role's line: "Not written: an Example that
  said the Problem again." A person's text is only read, never touched.

| Words already said above | #103 | #104 | #111 | Examples kept |
|---|---|---|---|---|
| the Need's, in the text on top | 57% | 64% | 50% | — |
| the Example's, in all above | 62% | 69% | 50% | 3% to 18% |

- **Built after piece 1, before the new names**: it needs the meanings
  piece 1 reads, not the names, and its effect is rated alone.
- **Measured**: #103, #104 and #111 drafted again, rated blind with the
  maintainer; in the trial, drafts accepted unchanged.
