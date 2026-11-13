---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: 3f424301
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
the next run. A night whose agent was not asked says so, and takes no
issue as read
([#427](https://github.com/JN0V/workline/issues/427)). The role's own comment is told by its hidden mark, never by
who posted it: it may run with your token. One whose hidden record does
not read holds its issue, a warning each run, never read as none
([#302](https://github.com/JN0V/workline/issues/302), `state-broken`):
mend that comment, or delete it for a state anew.

Each issue comes with:

- what the engine knows of it, who opened it — of the project or not —,
  which sections it has, which are the role's drafts and which yours;
- when its Need, Validation and Scope are there and Need and Validation are yours —
  your text on top, with no Need heading, is its Need —, that it may go
  to `ready` if evident: the agent judges that, the engine checks the
  sections;
- its comments, with who wrote each;
- the issues it cites, open or closed;
- the code it names, found by [the code
  picker](../../../docs/spec/code-picker.md) the tester and the reviewer
  share — a file linked on the forge counts; each definition of a name it
  writes as code, the lines it names, a file whole when it fits —, or
  else the repository's folders, to name the code to read from;
- for an issue imported from a file: that file's lines around its item,
  and its other lines naming the item's id — facts to refine from, not
  to ask you.

## 3. What it proposes, what the engine checks

- **Completing**: the sections an issue lacks (`refine`), a question to
  its reporter (`ask`), the move to `ready` — to `workline:needs-tests`
  first where the project routes the tester and its Validation has
  scenarios —, the code it is about
  (`sources`), an evident duplicate (`close`), the kind of an issue no
  label tells (`kind`, [the kind](kind.md)).
- **A blocked issue**: `unblock` to its state or to refine, or
  `blocked` afresh; blocked on its own read in refine, its text
  rewritten with neither, back to refine. The developer's ask for files
  outside the Scope:
  the Scope widened — a new file its pull request adds counts —, or
  why not; a widening left out keeps the issue blocked. A run too big
  for its role, or one predicted past its normal at ready: the Scope
  narrowed, split, granted `workline:size/large` or left to a person ([how](../../../docs/spec/backlog-acts/refine.md#its-size-at-ready)).
- **A question unanswered** past `acts.ask.wait`: its assumption
  written (`assume`)
  ([how long a question waits](settings.md#how-long-a-question-waits)).
- **Written for a reader**: a Need in one line, who needs what and why;
  one real example; a Validation as scenarios a person runs; the
  builder's part below a line; plain words in English, a link for each
  decision or doc cited ([writing an issue](writing.md)).
- **Never invented**: an issue too thin, holding several topics, or
  resting on a closed issue gets one plain question; one the code
  already does is said on it for you to close.
- **Off by default**: milestones, priorities, splits, renames, links,
  obsolete issues ([settings](settings.md#off-by-default)).
- The engine checks each act when it applies it: never a section a person
  wrote, edited or deleted; `ready` only with Need, Validation and Scope, and no draft;
  a Scope's files read from its text, each path found or on a line
  `New:`, or the Scope left out, the rest written
  ([writing](writing.md)); an outsider's issue proposed; never closed as not planned.
- An issue read and left with nothing that refines it — a kind or an
  order alone counts for nothing — is said in the summary, and on the
  issue: what it lacks, each act dropped and why, the agent's note, and
  when it is read again.
- In refine, the line wakes the product owner on it again, twice at
  most on the same text, told to refine it now; then it blocks the
  issue for you
  ([ADR-0059, amended](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-an-issue-left-unrefined-is-woken-again)).
  Given the same code as its last read, it is not woken again: blocked
  at once, the code it got said
  ([amended again](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-read-again-is-given-new-code-or-none-is-made)).
- Its sources named from the repository's folders, or only code the
  run did not give, need no line quoted: the issue is read again with
  that code; never for files its own text names, given ([#1440](https://github.com/JN0V/workline/issues/1440)).
- A Scope written with no code given for the issue is left out, its
  paths kept as the code to read: it is written from that code at the
  next run, never guessed
  ([#970](https://github.com/JN0V/workline/issues/970)). Beside sources
  whose code it was given, it is written at once.
- Then it does it, proposes it on its issue, or drops it, by the kind's
  mode and cap, and by the issue: one set aside gets nothing; one where a
  person undid that kind gets a proposal.

## 4. A person's comment

On an issue with nothing waiting on you — set aside, or read and left —,
a comment of yours or of its reporter is read first, the agent told your
word decides.

- **Set aside, then commented on**: it is brought back, your comment
  quoted to the agent; it completes, proposes or moves it to `ready` as
  for any issue, unless you say otherwise ("not now", "leave it").
- **Said with the "not now"**: a comment written before you took
  `workline:proposed` off brings nothing back.

On an issue waiting on you, it is "revise":

- the issue is read first, the agent told to revise;
- it rewrites its own sections in place — a draft, or a text as it wrote
  it — never yours;
- 👀 on each comment read, then one line in reply;
- one revision a run, two at most; past them, one line "left to a person"
  in the summary, and said in its own comment on the issue, edited in
  place, never a new one
  ([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)).

## 5. The reporter

- **Never asked what the code answers**: an ask on an issue whose code
  the same run names (`sources`) is not asked; the issue is read again
  with that code at the next run
  ([#548](https://github.com/JN0V/workline/issues/548)).
- **Asked again** only after they answered, never the same question
  twice, `acts.ask.rounds` times at most
  ([ADR-0021](../../../docs/adr/0021-the-product-owner-talks-with-the-reporter.md)):
  a question asked before is left out, the rest of the proposal kept.
- **An outsider's issue** gets no section in its body: the engine comments
  to its reporter what it understood and the sections it would write. A
  reply `agreed`, or `workline:accepted`, has them written; with the
  label, an answer since from the reporter or the project is read by the
  agent first, and its `refine` is written instead. If that
  comment's hidden block does not read, nothing is written, said each run
  (`proposal-unread`): copy the sections in by hand.

## Importing a file

On `import`, `workline issues import <file>` has the agent read a committed
file a share at a time
([the command](../../../docs/commands-work.md#workline-issues-import)):

- each item still to do opened as an issue quoting it, never twice;
- every other line answered with why (`skip`);
- every item mapped to its issue or reason; a line with neither listed for
  a person (exit 2).

## Opening issues

Every role opens an issue through one way the engine keeps for the
product owner
([backlog acts, "Opening issues"](../../../docs/spec/backlog-acts/opening.md)):
a key per subject, never opened twice; the role named, `needs-triage`; a
cap a run (`issues-max`).
