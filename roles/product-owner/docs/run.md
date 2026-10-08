---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: c24603b
verified: agent:claude-code
---
# Product owner — a run

Part of [the product owner](../README.md): what a run does, step by step.
It runs when gardening — the scheduled run, nightly or weekly
(`schedule`). What every run keeps track of — obsolete issues, what waits,
the report's opening, changed needs, parents, slipped milestones:
[tracking](tracking.md). The contract of its acts:
[backlog acts](../../../docs/spec/backlog-acts.md).

## A run, when gardening (`schedule`)

### 1. What the agent is given

`pre` gives the agent a share of the open issues (`issues-per-run`), in
this order:

- those an act was proposed on only for a run's cap;
- then those never read;
- then those whose code changed, that someone commented on or edited, or
  reopened after it closed them, or on whose spec the reviewer left
  findings open, not given yet
  ([#128](https://github.com/JN0V/workline/issues/128)), since read.

Each issue comes with:

- its state comment: its sources, the commit it was last confirmed and
  read at;
- who opened it — a workline role's issue named as that role's draft to
  refine;
- which of its four sections it has (Need, Scope, Verification,
  Validation);
- the code it names — by path, by a file's name alone, by a symbol quoted
  as code, in its body or a person's comment — up to `code-lines-max`
  lines in all; up to six others on the same code whole, the rest by
  title.

An issue with no state comment gets one, judged from the next run; one
whose comment does not read is left out (`state-broken`), as is the report
issue.

### 2. The acts it proposes

- **Closing**: a duplicate, its original's words quoted, or an issue the
  code made obsolete, the code quoted.
- **`sources`**: the code an issue is about, a line of it quoted.
- **Milestone**: the release an issue still true belongs to.
- **`order`**: its priority, 1 to 4, a label `workline:priority/N`.
- **Refining toward `ready`**: the sections an issue lacks (`refine`:
  Scope and Verification from the code, Need and Validation as drafts a
  person makes theirs), the move to `ready` (`ready`), or a question to its
  reporter (`ask`).
- **`split`**: an issue too big to be one need split into 2 to 6, each
  with its four sections, a child naming the siblings it waits on
  (`after`).
- **`rename`**: a title that says nothing.
- **`depend`**: what an issue waits on
  ([ADR-0028](../../../docs/adr/0028-an-issue-names-what-it-waits-on.md));
  `undepend`, a link it set that no longer holds, always proposed to a
  person.

The task shows what was asked or proposed to the reporter and what they
answered, and the rounds spent.

### 3. What the engine checks

The engine checks each act when it applies it:

- never closed as not planned; the quote found again; the issue's state
  readable;
- no section a person wrote rewritten;
- `ready` only when the four sections are there and none a draft — and,
  with the reviewer after it in the line, once the reviewer read the spec
  as it is with no important finding open, which then moves it with no
  agent ([#128](https://github.com/JN0V/workline/issues/128));
- an outsider's issue proposed; a priority or a title a person set kept;
- an issue split once, its children opened through the one way and linked
  to it — a sub-issue, a GitLab task, else a task list in its body;
- what an issue waits on written in the forge's own relation (GitHub's
  dependencies, GitLab's `is_blocked_by` on Premium), else a line
  `Blocked by #12.` in its body; never a cycle; a person's link kept;
- a link the role set whose blocker closed taken off by the engine, with
  no agent — the role's own told from a person's by its record and the
  engine's line;
- at most `moved-percent-max` of the open issues moved a run (20% at
  `normal`, 10% `cautious`, 30% `enterprising`).

Then it does each act, proposes it, or drops it, by the kind's mode and
cap (`acts`).

### 4. What a run leaves

- **One report issue**, what a person has to do first: what is
  proposed, what to check and to accept; then what was done, folded, with
  each moved issue's priority and milestone before the run, to put the
  order back ([outputs.md](outputs.md#the-report)).
- **A closing undone**, the issue reopened, puts that kind back to
  `propose` (`wrong-closing`).
- **Each act done alone** — not a person's tick, nor a slip the engine
  moved — is kept in the record (`did`) with its day and level for 35
  days: the weekly sample draws a week's for a person
  ([ADR-0033](../../../docs/adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)).

## The reporter and the person

- **Asked again** only after they answered, never the same question twice,
  three times at most (`acts.ask.rounds`); then the report asks a person to
  settle it with them
  ([ADR-0021](../../../docs/adr/0021-the-product-owner-talks-with-the-reporter.md)).
- **An outsider's issue** — its reporter not of the project: on GitHub
  without write access, on GitLab not a member from the Planner role up
  ([ADR-0023](../../../docs/adr/0023-the-project-bot-keeps-a-gitlab-backlog.md))
  — gets no section in its body. The engine comments to the reporter what
  the role understood, the sections it would write and what it still
  needs.
- **A reply `agreed`** from the reporter or a person of the project has
  those sections written at the next run, with no agent; ready stays the
  label's.
- The agent is told who wrote each comment.

## Accepting, importing

### Accepting the drafts

A person accepts the drafts with one label, `workline:accepted`, on one
issue or many from the list of issues:

- the next run takes the draft lines out and moves each to `ready`, with or
  without an agent;
- on an outsider's issue, it first writes the sections last proposed to
  its reporter, unless they answered since: the agent reads the answer
  first;
- without an agent, otherwise, only the state comments are written.

### Importing a file

On `import`, `workline issues import <file>` has the agent read a committed
file a share at a time — with the lines elsewhere in the file that name its
items' ids, where a file often says what is done, and the closed issues an
import opened from the share's lines.

- Each item still to do is opened as an issue quoting it, never twice
  (`open`).
- Without `--apply`, it only says what it would open, and lists its runs
  for `workline apply`, in CI's job that holds the write token.
- The agent says why of every other line (`skip`: done, the words quoted;
  held by an issue; not an item).
- The engine checks each reason and maps every item of the file to its
  issue or its reason. The lines with neither are listed as not covered,
  the share that left them out flagged, and the import ends for a person
  (exit 2).

## The report's boxes, and the pause

Each proposal in the report is a box
([ADR-0025](../../../docs/adr/0025-a-persons-tick-is-done-ignored-runs-pause.md);
changing: [ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)):

- **Ticked by a person of the project** — the forge says who ticked it:
  GitHub's edit history, GitLab's system notes — it is done at the next
  run, as the record keeps it, with no agent.
- **Recorded without its act** by an older engine: its issue is read
  again first; ticked, the act the agent drafts then is done as that
  person's yes — never handed back to them.
- **Ticked by an outsider, a bot or nobody the forge names**, it is not,
  and the report says why.
- **A kind back to `propose`** after a person undid one of its acts gets a
  box too, to set it back to `act`: a closing reopened; a title, a
  priority or a milestone put back; `workline:ready` taken off; a split's
  part closed as not planned
  ([ADR-0026](../../../docs/adr/0026-the-product-owners-autonomy-is-a-level.md));
  a link it set between issues taken off.
- **The pause**: three runs in a row (`ignored-runs-max`; 0 never pauses,
  said in every report) read with an agent and nobody answering — no box
  ticked, no comment on the report, no act undone, no proposal settled —
  pause the role: no agent asked until a person does one of those. The
  report's top says it a run ahead: "answer before the next run, or the
  role pauses".

## Opening issues

Every role opens an issue through one way the engine keeps for the
product owner
([ADR-0018](../../../docs/adr/0018-the-product-owner.md);
[backlog acts, "Opening issues"](../../../docs/spec/backlog-acts.md#opening-issues)):

- a key per subject: an issue open or closed holding it is never opened
  again;
- the role named, `needs-triage`;
- a cap a run (`issues-max`);
- the product owner reads it from its next run.
