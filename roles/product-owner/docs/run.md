---
sources: [roles/product-owner/role.yaml, roles/product-owner/prompts/instruction.md, roles/product-owner/prompts/policy.md, roles/product-owner/code, internal/backlog]
checked: e81df7b34
verified: agent:claude-code
---
# Product owner — a run

Part of [the product owner](../README.md): what a run does, step by step.
What every run keeps track of: [tracking](tracking.md). What it writes:
[outputs](outputs.md). The contract of its acts:
[backlog acts](../../../docs/spec/backlog-acts.md).

## 1. What it reads first, with no agent

From each open issue's own state
([ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).
On `issue`, from that issue's alone: the other issues' comments are not
read, so a step of the line costs a few API calls, not one per open issue
([#804](https://github.com/JN0V/workline/issues/804)). Of what counts
across the issues, `proposals-max` is read from the labels; `undone-max`
waits for a run on the whole backlog.

- **A yes**: `workline:accepted` on an issue waiting on you. What its
  state proposes is done as recorded, its drafts made yours, the issue
  moved to `ready`; the role takes `workline:proposed` off.
- **A move back to refine still waiting** from before it needed no yes:
  done as recorded, its reason in the role's comment
  ([ADR-0059, amended](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-back-to-refine-needs-no-yes)).
- **Not now**: `workline:proposed` taken off by a person, without a yes.
  Its proposals are dropped; nothing is proposed there until the issue
  changes. A label a bot took off is put back.
- **An act undone**: a title, a priority, a milestone put back,
  `workline:ready` taken off, or put back after the role moved the issue
  back to refine, a split's part closed as not planned, a closing
  reopened. That kind is proposed on that issue from then on.
- **Parked**: an open issue labelled `workline:parked` is not read nor
  offered next, until a person takes the label off
  ([#1752](https://github.com/JN0V/workline/issues/1752)).
- **The first run after a report issue**: its record moves to each
  issue's state, and the report is closed with a link to the filter.

## 2. What the agent is given

On `issue`, its state `workline:to-refine` woken where the project
routes the role there, that issue alone, an import never read too
([the line of states](../../../docs/spec/routing.md#the-line-of-states)).
A `workline:blocked` issue with news, or a week on, too
([blocked, read again](../../../docs/spec/routing.md#blocked-read-again)).
Else a share of the open issues (`issues-per-run`), in this order:

1. an issue you commented on since it was read: waiting on you
   ("revise"), set aside, or neither — your word comes first;
2. an act proposed only for a run's cap;
3. a person's issue never read, newest first — one opened since the last
   run is read in the run that first sees it;
4. one whose code changed, or that a person edited or reopened, or the
   reviewer's spec findings to answer;
5. one read and left with nothing by an older engine, read again once;
6. the catch-up: an import's or a bot's issue never read, oldest first.

Past `proposals-max` issues waiting on you, only those you commented on
are read. The share stops where the prompt would pass the role's context
budget: the issues left out are said in the summary and read first at
the next run.

- A night whose agent was not asked says so, and takes no issue as read
  ([#427](https://github.com/JN0V/workline/issues/427)).
- The role's own comment is told by its hidden mark, never by who posted
  it: it may run with your token.
- One whose hidden record does not read holds its issue, a warning each
  run, never read as none
  ([#302](https://github.com/JN0V/workline/issues/302), `state-broken`):
  mend that comment, or delete it for a state anew.

### What each issue comes with

Each issue comes with:

- what the engine knows of it, who opened it — of the project or not —,
  which sections it has, which are the role's drafts and which yours;
- when its Need, Validation and Scope are there and Need and Validation are yours —
  your text on top, with no Need heading, is its Need —, that it may go
  to `ready` if evident: the agent judges that, the engine checks the
  sections;
- its comments, with who wrote each;
- the issues it cites, open or closed;
- no code: under `claude`, the agent reads the project itself, in a
  read-only checkout at the run's commit (Read, Glob, Grep, 40 turns),
  each file it read logged; a `cmd:` agent reads no code, said as
  `input-cut` ([ADR-0074](../../../docs/adr/0074-agents-read-the-repository-the-engine-stops-preparing-their-input.md));
- for an issue imported from a file: that file's lines around its item,
  and its other lines naming the item's id — facts to refine from, not
  to ask you.

## 3. What it proposes, what the engine checks

What it completes, how a blocked issue is read again, what the engine
refuses: [what it proposes](run-proposals.md).

## 4. A person's comment, and 5. the reporter

A comment brings back an issue set aside, or revises its drafts; an
outsider's issue gets a comment first, never a section written:
[people](run-people.md), which also holds
[importing a file](run-people.md#importing-a-file) and
[opening issues](run-people.md#opening-issues).
