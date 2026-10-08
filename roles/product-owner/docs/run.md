---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: 47ed51e
verified: agent:claude-code
---
# Product owner — a run

Part of [the product owner](../README.md): what a run does, step by step.
What every run keeps track of: [tracking](tracking.md). What it writes:
[outputs](outputs.md). The contract of its acts:
[backlog acts](../../../docs/spec/backlog-acts.md).

## 1. What it reads first, with no agent

From each open issue's own state
([ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)):

- **A yes**: `workline:accepted` on an issue waiting on you. What its
  state proposes is done as recorded, its drafts made yours, the issue
  moved to `ready`; the role takes `workline:proposed` off.
- **Not now**: `workline:proposed` taken off by a person, without a yes.
  Its proposals are dropped; nothing is proposed there until the issue
  changes. A label a bot took off is put back.
- **An act undone**: a title, a priority, a milestone put back,
  `workline:ready` taken off, a split's part closed as not planned, a
  closing reopened. That kind is proposed on that issue from then on.
- **The first run after a report issue**: its record moves to each
  issue's state, and the report is closed with a link to the filter.

## 2. What the agent is given

A share of the open issues (`issues-per-run`), in this order:

1. an issue waiting on you that you commented on: "revise";
2. an act proposed only for a run's cap;
3. a person's issue never read, newest first — one opened since the last
   run is read in the run that first sees it;
4. one whose code changed, or that a person edited or reopened, or the
   reviewer's spec findings to answer;
5. the catch-up: an import's or a bot's issue never read, oldest first.

Past `proposals-max` issues waiting on you, only the first are read.

Each issue comes with what the engine knows of it, who opened it, which
sections it has and which are the role's, its comments with who wrote
each, and the code it names.

## 3. What it proposes, what the engine checks

- **Completing**: the sections an issue lacks (`refine`), a question to
  its reporter (`ask`), the move to `ready`, the code it is about
  (`sources`), an evident duplicate (`close`).
- **Off by default**: milestones, priorities, splits, renames, links,
  obsolete issues ([settings](settings.md#off-by-default)).
- The engine checks each act when it applies it: never a section a person
  wrote, edited or deleted; `ready` only with four sections and no draft;
  an outsider's issue proposed; never closed as not planned.
- Then it does it, proposes it on its issue, or drops it, by the kind's
  mode and cap, and by the issue: one set aside gets nothing; one where a
  person undid that kind gets a proposal.

## 4. A person's comment: revise

On an issue waiting on you, a comment of yours, or of its reporter:

- the issue is read first, the agent told to revise;
- it rewrites its own sections in place — a draft, or a text as it wrote
  it — never yours;
- 👀 on each comment read, then one line in reply;
- one revision a run, two at most; past them, one line "left to a person"
  in the summary, nothing on the issue.

## 5. The reporter

- **Asked again** only after they answered, never the same question
  twice, `acts.ask.rounds` times at most
  ([ADR-0021](../../../docs/adr/0021-the-product-owner-talks-with-the-reporter.md)).
- **An outsider's issue** gets no section in its body: the engine comments
  to its reporter what it understood and the sections it would write. A
  reply `agreed`, or `workline:accepted`, has them written.

## Importing a file

On `import`, `workline issues import <file>` has the agent read a committed
file a share at a time
([the command](../../../docs/commands.md#workline-issues-import)):

- each item still to do opened as an issue quoting it, never twice;
- every other line answered with why (`skip`);
- every item mapped to its issue or reason; a line with neither listed for
  a person (exit 2).

## Opening issues

Every role opens an issue through one way the engine keeps for the
product owner
([backlog acts, "Opening issues"](../../../docs/spec/backlog-acts.md#opening-issues)):
a key per subject, never opened twice; the role named, `needs-triage`; a
cap a run (`issues-max`).
