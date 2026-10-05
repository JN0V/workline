---
sources: [roles/product-owner/role.yaml, roles/product-owner/instruction.md, roles/product-owner/policy.md, internal/builtin/productowner, internal/backlog]
checked: 472daca
verified: agent:claude-code
---
# Product owner

What runs, for humans. The AI never reads this file.

It keeps a project's backlog — its open issues — true to the code, between
the need a person states and the result they accept (ADR-0018). The
contract of its acts is [docs/spec/backlog-acts.md](../../docs/spec/backlog-acts.md).

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
under "Waiting".

An issue in a milestone whose release is tagged slipped: the engine moves
it to the nearest open milestone not released, with or without an agent,
or proposes it in the report when there is none.

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

Every role opens an issue through one way the engine keeps for the
product owner (ADR-0018; docs/spec/backlog-acts.md, "Opening issues"): a
key per subject, an issue open or closed holding it never opened again,
the role named, `needs-triage`, a cap a run (`issues-max`); the product
owner reads it from its next run.

## Settings

```yaml
roles:
  product-owner:
    settings:
      issues-per-run: 8
      code-lines-max: 1500
      autonomy: normal                           # cautious | normal | enterprising
      ignored-runs-max: 3                        # 1 to 20; 0 never pauses
      moved-percent-max: 20                      # milestones and priorities, together
      acts:
        open: {mode: act, max: 30}               # when importing
        sources: {mode: act, max: 10}
        milestone: {mode: act, max: 10}
        order: {mode: act, max: 10}
        close-duplicate: {mode: act, max: 3}     # act | propose | off
        close-obsolete: {mode: act, max: 3, days: 7, exempt: [pinned, security]}  # announced, then closed
        refine: {mode: act, max: 5}
        ready: {mode: act, max: 5}
        ask: {mode: act, max: 3, rounds: 3}    # rounds: times a reporter is asked
        split: {mode: act, max: 2}
        rename: {mode: act, max: 5}
        depend: {mode: act, max: 5}              # propose at cautious, 10 at enterprising
```

These are `normal`. `autonomy` changes them in one word (ADR-0026;
role.yaml's `levels`, docs/spec/backlog-acts.md's table): `cautious`, for a
project whose Product Owner is a person — `workline init` asks —, keeps
the acts that check facts (sources, asking the reporter, announcing what
is obsolete, Scope and Verification) and proposes those that set
direction (Need and Validation drafts, splits, renames, milestones,
priorities, duplicates, what an issue waits on), for a person to tick in the report;
`enterprising` raises the caps. A kind set here wins over the level,
field by field; a kind demoted stays proposed whatever the level. The
task and the report say each kind's mode and where it comes from; the
report may suggest another level from what people did with the
proposals, and never changes it.

Where the role stands: [status.md](status.md).
