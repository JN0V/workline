---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: c67e16a
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
are read. The role's own comment is told by its hidden mark, never by
who posted it: it may run with your token.

Each issue comes with:

- what the engine knows of it, who opened it — of the project or not —,
  which sections it has, which are the role's drafts and which yours;
- when its four sections are there and Need and Validation are yours,
  that it may go to `ready` if evident: the agent judges that, the
  engine checks the sections;
- its comments, with who wrote each;
- the issues it cites, open or closed;
- the code it names — a file linked on the forge counts —, or else the
  repository's folders, to name a Scope's files from.

## 3. What it proposes, what the engine checks

- **Completing**: the sections an issue lacks (`refine`), a question to
  its reporter (`ask`), the move to `ready`, the code it is about
  (`sources`), an evident duplicate (`close`).
- **Written for a reader**: a Need says who needs what and why, from
  their side; a Validation, what a person sees once it is done; plain
  words, a link for each decision or doc cited.
- **Never invented**: an issue too thin, holding several topics, or
  resting on a closed issue gets one plain question; one the code
  already does is said on it for you to close.
- **Off by default**: milestones, priorities, splits, renames, links,
  obsolete issues ([settings](settings.md#off-by-default)).
- The engine checks each act when it applies it: never a section a person
  wrote, edited or deleted; `ready` only with four sections and no draft;
  a Scope's files read from its text, or the Scope left out, the rest
  written; an outsider's issue proposed; never closed as not planned.
- An issue read and left with nothing is said in the summary.
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
  in the summary, nothing on the issue.

## 5. The reporter

- **Asked again** only after they answered, never the same question
  twice, `acts.ask.rounds` times at most
  ([ADR-0021](../../../docs/adr/0021-the-product-owner-talks-with-the-reporter.md)):
  a question asked before is left out, the rest of the proposal kept.
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
