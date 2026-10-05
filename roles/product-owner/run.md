---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: c592b51
verified: agent:claude-code
---
# Product owner — a run

Part of [the product owner](README.md): what a run does, step by step.
The contract of its acts is
[docs/spec/backlog-acts.md](../../docs/spec/backlog-acts.md).

## A run, when gardening (`schedule`)

1. `pre` gives the agent a share of the open issues (`issues-per-run`):
   those an act was proposed on only for a run's cap first, then those
   never read, then those whose code changed, that someone commented on
   or edited, or reopened after it closed them, since read; each with its
   state comment (its sources, the commit it was last confirmed and read at),
   who opened it — a workline role's issue named as that role's draft to
   refine —, which of its four sections it has, and the code it names —
   by path, by a file's name alone, by a symbol quoted as code, in its body
   or a person's comment — up to `code-lines-max` lines in all; up to six
   others on the same code whole, the rest by title. An issue with no state
   comment gets one, judged from the next run; one whose comment does not
   read is left out (`state-broken`), as is the report issue.
2. The agent proposes acts: a closing — a duplicate, its original's words
   quoted, or an issue the code made obsolete, the code quoted; the code
   an issue is about (`sources`), a line of it quoted; the milestone of
   the release an issue still true belongs to; its priority (`order`, 1
   to 4, a label `workline:priority/N`); refining toward `ready` —
   the sections an issue lacks (`refine`: Scope and Verification from the
   code, Need and Validation as drafts a person makes theirs), the move to
   `ready` (`ready`), or a question to its reporter (`ask`); an issue too
   big to be one need split into 2 to 6, each with its four sections
   (`split`), a child naming the siblings it waits on (`after`); a title
   that says nothing renamed (`rename`); what an issue waits on
   (`depend`, ADR-0028). The task
   shows what was asked or proposed to the reporter and what they
   answered, and the rounds spent.
3. The engine checks each one when it applies it — never as not planned,
   the quote found again, the issue's state readable, no section a person
   wrote rewritten, `ready` only when the four sections are there and none
   a draft, an outsider's issue proposed, a priority or a title a person
   set kept, an issue split once, its children opened through the one way
   and linked to it — a sub-issue, a GitLab task, else a task list in its
   body —,
   what an issue waits on written in the forge's own relation (GitHub's
   dependencies, GitLab's `is_blocked_by` on Premium), else a line
   `Blocked by #12.` in its body, never a cycle, a person's link kept,
   at most `moved-percent-max` of the open issues moved a run (20% at
   `normal`, 10% `cautious`, 30% `enterprising`) — and does it,
   proposes it, or drops it, by the kind's mode and cap (`acts`). One
   report issue lists what was done and proposed. A closing undone, the
   issue reopened, puts that kind back to `propose` (`wrong-closing`).
   The report says each moved issue's priority and milestone before the
   run, to put the order back.

## Obsolete, waiting, slipped

An issue the code made obsolete is announced, not closed (ADR-0024): a
comment to its reporter, the code quoted and the day from which it may
close, and the label `workline:obsolete`. At a run `days` later (7), if
nobody wrote on it, the label is still there, no exempt label (`pinned`,
`security`) was set and the code quoted is still there, a second judge of
another model is asked apart; if it agrees, the engine closes the issue as
completed, saying the judge and its level. Anyone's comment, the label
taken off or the judge's no keeps it open for good on that evidence
(`kept` in its state). Announcements and closings share
`close-obsolete`'s cap.

The backlog's order puts an issue that waits on an open issue after it,
whatever its priority; a cycle is reported, never followed. Every run,
with or without an agent, says the first ready issue waiting on nothing
(`next-ready`) — the one offered to whoever builds next —, each issue
waiting (`waiting`) and each cycle (`dependency-cycle`); the report too,
under "Waiting". A split need is never the one offered: its parts are.

The report opens with **Next** and **Stuck** (ADR-0031), rebuilt at every
run with or without an agent, nothing of them stored: the first
`next-max` (5) ready issues of the order waiting on nothing, with their
milestone and priority; then each issue waiting on a person for more
than `stuck-days` (14), with the day it started — ready since the forge
says it got the label, no pull request nor commit naming it since; its
reporter last written to, no answer since; a proposal of the report
unticked, since the record says it was first proposed; an announcement
as obsolete past its delay, no second judge yet. An issue appears once,
in its first list. A day the forge does not say is said
(`stuck-unknown`), never read as nothing stuck. A run with nothing else
to write rewrites the report when its opening changed.

A changed need flags the issues built on it (ADR-0032), found with or
without an agent: an issue's Need or Scope a person rewrote since its
state kept them, or the lines of a file an issue was imported from,
changed by a commit since it was read. Its open parts, and the issue
imported, are read first, with the text as it was and as it is; every
act the agent proposes on them goes to the report (`need-changed`) —
`unready`, back to refine, included, which only a person's tick does.
The issues waiting on it and those on the same code are listed, not
read. The report's "Changed needs" says each change and its issues,
until a person ticks it checked; nothing is written to them.

A split need — a parent — gets one comment, with or without an agent,
edited in place as its parts move (ADR-0029): each part open, closed as
completed with the pull request or commit that closed it, or closed
without delivering (not planned, a duplicate, gone); each item of its
Verification proved where a part delivered quotes it, in its own
Verification or in what closed it, or "not proved". Once all its parts
are closed, the comment, a finding (`parent-to-accept`) and the report's
"To accept" ask a person to accept it by closing it; the role never
does, and an agent's closing of a parent is dropped.

An issue in a milestone whose release is tagged slipped: the engine moves
it to the nearest open milestone not released, with or without an agent,
or proposes it in the report when there is none.

## The reporter and the person

The reporter is written to again only after they answered, never the same
question twice, three times at most (`acts.ask.rounds`); then the report
asks a person to settle it with them (ADR-0021). An outsider's issue — its
reporter not of the project: on GitHub without write access, on GitLab
not a member from the Planner role up (ADR-0023) — gets no section in its
body: the engine comments to the reporter what the role understood, the
sections it would write and what it still needs. A reply `agreed` from
the reporter or a person of the project has those sections written at the
next run, with no agent; ready stays the label's. The agent is told who
wrote each comment.

## Accepting, importing

A person accepts the drafts with one label, `workline:accepted`, on one
issue or many from the list of issues: the next run takes the draft lines
out and moves each to `ready`, with or without an agent; on an outsider's
issue, it first writes the sections last proposed to its reporter, unless
they answered since: the agent reads the answer first. Without an
agent, otherwise, only the state comments are written. On `import`,
`workline issues import <file>` has the agent read a committed file a
share at a time — with the lines elsewhere in the file that name its
items' ids, where a file often says what is done — and opens each item
still to do as an issue quoting it, never twice (`open`); without `--apply`, it only says what it would open.
The agent says why of every other line (`skip`: done, the words quoted;
held by an issue; not an item); the engine checks each reason and maps
every item of the file to its issue or its reason, the lines with
neither listed as not covered, the share that left them out flagged,
and the import ending for a person (exit 2).

## The report's boxes, and the pause

Each proposal in the report is a box. Ticked by a person of the project —
the forge says who ticked it: GitHub's edit history, GitLab's system
notes — it is done at the next run, as the record keeps it, with no
agent; ticked by an outsider, a bot or nobody the forge names, it is not,
and the report says why (ADR-0025). A kind back to `propose` after a
person undid one of its acts — a closing reopened; a title, a priority or
a milestone put back; `workline:ready` taken off; a split's part closed
as not planned (ADR-0026); a link it set between issues taken off —
gets a box too, to set it back to `act`.
Three runs in a row (`ignored-runs-max`; 0 never pauses, said in every
report) read with an agent and nobody answering — no box ticked, no
comment on the report, no act undone, no proposal settled — pause the
role: no agent asked until a person does one of those.

## Opening issues

Every role opens an issue through one way the engine keeps for the
product owner (ADR-0018; docs/spec/backlog-acts.md, "Opening issues"): a
key per subject, an issue open or closed holding it never opened again,
the role named, `needs-triage`, a cap a run (`issues-max`); the product
owner reads it from its next run.
