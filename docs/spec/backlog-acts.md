---
type: reference
sources: [internal/backlog, internal/builtin/productowner, internal/engine, internal/forge, roles/product-owner/role.yaml, roles/product-owner/instruction.md]
status: draft
checked: a64727b
verified: agent:claude-code
---
# Acts on the backlog

What the engine does when a role acts on a project's issues rather than on
its code (ADR-0018). The product owner proposes the acts; the engine checks
each one against the code and the forge, then does it, proposes it, or drops
it. This page is the contract; the acts built are closing, naming an
issue's sources, putting it in a milestone, ordering it, opening one from
a file, refining one to ready, splitting one and renaming it.

## The issue's state

What the engine knows of an issue lives in one comment it keeps on that
issue, marked `<!-- workline:sticky=product-owner/state -->`:

```yaml
sources: [src/export/csv.go#WriteRows]   # what the issue is about, in the code
confirmed: 1a8e5a4                       # the commit it was last found true at
judged: 1a8e5a4                          # the commit the role last read it at
comments: 2                              # people's comments when it was read
body: 3f9a1c0e2b7d                       # a digest of its body, when last read or written
priority: 2                              # the priority the role last set (Ordering)
title: The CSV export drops the last row  # the title the role last set (Renaming)
split: [13, 14]                          # the children it was split into (Splitting)
kept: ['src/export/csv.go:b2a6ba892dee'] # announcements as obsolete kept open, by their quote (Closing)
```

Two carrying the marker, the last is read: on GitLab only a note's author
edits it, so one another token wrote is left, and the state is written
anew after it (ADR-0023).

An issue without that comment, or with one that does not read, is never
acted on: an act on it is dropped (`no-state`, `state-broken`) and nothing is
written on the issue. The role's `pre` gives an issue it takes its first
state, `confirmed` at the commit it read, with no sources until an act
names them. An issue another role opens for what it found outside its task
(the reviewer's, ADR-0020) gets its state from the engine as it is opened —
its sources, and the commit it was seen at — and `needs-triage`; it is
never read yet, so the next run reads it first, and its agent is told it
is that role's draft to refine ("Opening issues", below).

## Opening issues

Every role opens an issue through one way (ADR-0018, `backlog.Openings`),
with no AI:

| Step | What the engine does |
|---|---|
| Key | A finding on code quotes its line (`at: {path, text}`); found again in the file, the key is the file and that line, spaces aside (`issue=<path>#<8 hex>`). Otherwise, or the line not found (`issue-quote-not-found`, said), its title (`issue=title#<8 hex>`). A text a backlog file is imported from keys its own (`import=<path>:<hex>`). Hidden in the body. |
| Looked for | In every issue, open and closed, read once a run (`AllIssues`); the key a role wrote before the way was shared (`issue=<role>/<key>`) counts too. |
| Open, holding it | Left as it is: nothing written, whichever role opened it. |
| Closed as not planned or duplicate | A person's no: nothing written (`issue-closed`, said in the run). |
| Closed otherwise | Done, or no reason kept (GitLab, the local forge): one comment, once, "found again" with the commit; left closed (`issue-closed`). |
| New, past `issues-max` | Not opened, counted (`issues-capped`); found again, opened at a later run. Three by default; the product owner's import has its own cap (`open`). |
| New | Opened: the role named in a line; for a finding, `<!-- workline:opened-by=<role> -->` and `needs-triage`; the product owner's state, written last — a run stopped before it, resumed, finds the issue open and writes what it lacks. |

## Reading

A run reads at most `issues-per-run` issues, with at most `code-lines-max`
lines of the code they name: first those an act was proposed on only
because a run's cap was reached — though nothing changed on them, the act
is decided again, and the proposal leaves the report once its issue is
read —, then those never read, then those with
something new since they were read — a commit touched their sources, a
person commented (`comments` counts people's comments read), a person
changed its body (`body`), or a person reopened what the role closed. An issue with nothing new is not read again,
however old (ADR-0018). An issue on the same code as one read is given with
its body beside it, up to six, as the original a duplicate would be closed
against; the others are listed by title only. Each issue read gets `judged`
moved to the run's commit; without an agent, or when its answer does not
read, none is.

## Importing a file

`workline issues import <file>` moves a roadmap, a backlog or notes to the
forge's issues, once. Formats differ from one project to the next, and
telling an item from an introduction, or done from still to do, is
judgement: the product owner reads the file, not a parser (principle 4).

- The command cuts the file by size only (`--lines`, 300 by default), a
  fifth of each share read again in the next, so an item cut at the end of
  one is whole in the other; it runs the role on each share, on the event
  `import`.
- A file often says an item is done far from the item — a table of what
  shipped, a summary at the end. With each share, the agent is given the
  lines of the rest of the file that name an id the share holds (`BUG-6`,
  `LO-12`: capitals, a dash, a number), those of a table or a heading
  first, four an id at most. Which of them says the item is done is its
  judgement; an item without an id is read from its share alone.
- The agent proposes an `open` for each item still to do, or done in part:

  ```yaml
  - open: {title: "Keep the last row", quote: {path: ROADMAP.md, text: "1. **Keep the last row.** WriteRows stops one row short."}}
  ```

- The engine opens it only if the quote is in the file as committed, but
  for spaces (`no-quote`); its body is the file's text at those lines, and
  where it came from — never words of the agent's. An open issue already
  holding the same text is not opened again (`already-open`, a digest of
  the text in a marker), so an import stopped half-way is run again. Each
  issue gets its state, confirmed at the commit, no sources: gardening
  names them (`sources`).
- Without `--apply`, it says what it would open, the quotes not checked
  yet, and writes nothing. `open` is capped per share (`acts.open.max`):
  past it, an item is proposed in the report, and the import run again
  opens it.

The file then stays as it is, for its history.

## Closing

```yaml
- close:
    issue: 12
    reason: duplicate          # duplicate | obsolete
    duplicate-of: 7            # a duplicate's original
    quote: {issue: 7, text: "the export drops the last row"}
    # or, for obsolete: {path: src/export/csv.go, text: "for i := 0; i <= len(rows); i++"}
    why: "#7 reports the same row dropped by WriteRows."
```

- **Never not planned.** Refusing a need is the person's (principle 1): any
  other reason is dropped (`close-reason`), the issue left open.
- **No quote, no act.** The quote must be found again, as written but for
  spaces, in the file at the commit the run is on, or in the issue's title,
  body or one of its comments. A quote missing or not found drops the act
  (`no-quote`).
- A duplicate is closed with the forge's own reason (GitHub's `duplicate`,
  linking the original), with a comment that quotes the evidence and says
  how to undo: reopen it.

### Obsolete: announced, then closed

An obsolete issue is not closed by the agent's act (ADR-0024). In mode
`act`, the act **announces** it — unless the issue bears an exempt label
(`exempt`; `acts.close-obsolete.exempt`, `pinned` and `security` by
default), is announced already (`announced`), or was kept open on the same
quote (`obsolete-kept`, its file and a digest of its text, in the state's
`kept`):

- the label `workline:obsolete` (created when the project has none), then
  a comment naming its reporter: why, the code quoted, the commit, the day
  from which it may be closed (`days` later, 7 by default), and how to keep
  it open; it ends with a block the engine reads back — the quote, why,
  the commit, the day announced, the model that proposed it — and the
  marker `<!-- workline:product-owner/obsolete -->`.

At each later run, `pre` reads the last announcement of each issue not
settled (its quote not in `kept`), with no agent:

| Found | What the engine does |
|---|---|
| A comment after it, not the engine's nor a bot's — whoever wrote it | `keep`: the label off, the quote added to `kept`; nothing written to them |
| The label taken off, or an exempt label set | `keep`, the same |
| The code quoted no longer in the commit | `keep`, and the issue told why |
| Its day not come | nothing |
| Due | a second judge asked (`in/judge/obsolete-<n>/`), apart, at the best independence from the model that announced it (ADR-0005); yes: the engine's `close`, checked again — the announcement there, due, unanswered, the same quote, the judge's yes — then closed as completed (GitLab, the local forge: closed), the label off, a comment with the evidence, the day announced and the judge with its level; no: `keep`, the judge's reason said on the issue |
| Due, no agent or a judge that did not answer | nothing: it waits (`judge-unavailable`) |

Closing as completed, never as not planned: the code did the work, and
not planned is a person's no, which every role's opening reads as such.
`keep` is never held back by a mode nor a cap; the agent may propose it
too, on an issue announced. In `propose`, the agent's act and the
engine's closing are written in the report for a person, nothing is
announced.

## Naming its sources

```yaml
- sources:
    issue: 12
    sources: [src/export/csv.go]
    quote: {path: src/export/csv.go, text: "func WriteRows(rows []string, write func(string)) {"}
    why: "#12's WriteRows is defined here."
```

The files must be in the commit the run is on (1 to 5; `sources-unknown`
otherwise), a quote from a file found in one of them. Done, the issue's
state gets them and loses `judged`: it is read again, with that code, at the
next run.

## Milestones

```yaml
- milestone: {issue: 12, milestone: "v2.14.0", why: "the next release's fix"}
```

Puts an open issue in the milestone of a release, created when none with
that title is open. Ordering says nothing of an issue's truth: no quote,
but its state must read as for any act. The task gives the last release
tag and the milestones open; each issue says its own.

**What slipped** is moved by the engine, with or without an agent: an
open issue whose milestone is named after a tag that exists is put in
the nearest open milestone not released, in version order (`v1.9.0`
before `v1.10.0`); with none left, the move is proposed in the report,
the issue left where it is. The engine's move comes first: an agent's
milestone for the same issue in the same run is dropped.

## Ordering

```yaml
- order: {issue: 12, priority: 1, why: "it loses rows from every report"}
```

Sets an issue's priority: one label, `workline:priority/1` (the most
pressing) to `workline:priority/4`, created when the project has none,
the other three taken off. A level outside 1 to 4 is dropped
(`priority-level`); the priority it has already, too (`priority-same`).
The issue's state records the priority set. A priority label other than
the one recorded — set by a person, or by a person over the role's, or
taken off by one — is a person's: the act is dropped and the label kept
(`priority-kept`); the task shows it as a person's. No quote; the state
must read, as for any act.

**The backlog's order** is derived, never stored: the nearest milestone
first (titles in version order; an issue in none after every one in
one), then the priority (an issue with none after 4), then the lowest
number (`backlog.Less`). The task lists the issues not read in that order.

## Refining to ready

An issue is `ready` when four sections are written in its body — `## Need`,
`## Verification`, `## Validation`, `## Scope` (docs/spec/routing.md) — and
its Need and Validation are a person's (ADR-0018). The product owner writes
what it can and asks for the rest:

```yaml
- refine:
    issue: 12
    scope: "WriteRows in src/export/csv.go, and its test."
    sources: [src/export/csv.go]     # the files the scope names, in the commit
    verification: "A test writes three rows and reads three lines back."
    need: "Every row exported, so a report counts what was sold."   # a draft
    validation: "The maintainer opens an export of a known day."    # a draft
    why: "The issue says what is wrong, not how it will be proved."
- ready: {issue: 12, why: "Its four sections are there, Need and Validation the reporter's."}
- ask: {issue: 14, questions: "Which export: CSV or JSON? What should a blank row become?", why: "The need is not clear."}
```

- **`refine`** adds the sections the body does not have, after its text,
  which stays as it is; one there but empty — an issue form's field left
  `_No response_` — is filled in place (an issue form writes its fields as
  `### ` headings, read as `## ` ones). A section with text — a person's,
  or one the role wrote before — is never rewritten (`section-kept`). Need
  and Validation are drafts: each follows a line saying so, holding a
  hidden marker (`<!-- workline:draft -->`), and the issue gets the label
  `workline:draft`. **A person accepts them with one label,
  `workline:accepted`** — on the issue, or on many at once from the list of
  issues; the engine creates that label with the first draft, so it is
  there to pick; only who may triage sets a label, so it is a person of the
  project's. The engine then takes the draft lines out and moves the issue
  to ready, with no agent. Editing a draft and deleting its line makes it
  a person's too. Scope names its files (`sources`, 1 to 5, in
  the commit, as for naming its sources), which the issue's state gets when
  it has none. Nothing to add drops the act (`nothing-to-refine`). The
  issue gets the label `workline:to-refine`.
- **`ready`** is checked by the engine, not taken from the agent: the four
  sections there and not empty, no draft marker in Need or Validation
  unless the issue bears `workline:accepted` — read on the issue, never in
  a proposal (`not-ready` otherwise, naming what is missing). Done, the issue gets
  `workline:ready` and loses `workline:to-refine`, as `workline item ready`
  does, and loses `workline:draft` and `workline:accepted`. An issue opened
  by someone without write access to the project is theirs: moving it to
  ready is proposed in the report, never done (`reporter-outside`) —
  unless a person of the project accepted it with the label. Who is of the
  project: on GitHub, the issue's author association (owner, member,
  collaborator); on GitLab, a member with the Planner role or above, read
  from the project's members (ADR-0023) — a token that may not list them
  fails the run, loud; on a plugged forge, its `insider`. One the forge
  does not say is outside.
- **`ask`** comments on the issue, naming its reporter, with the agent's
  questions. The answer is a person's comment: the issue is read again at
  the next run.
- An issue whose body a person changed since it was read — a draft
  accepted, a section written — is read again (its state's `body`, a
  digest of the body when it was last read or written by the engine).

## Splitting

```yaml
- split:
    issue: 12
    into:                                   # 2 to 6 children
      - title: "Keep the last row in the CSV export"
        need: "Every row exported, so a report counts what was sold."   # a draft
        verification: "A test writes three rows and reads three lines back."
        validation: "The maintainer opens an export of a known day."    # a draft
        scope: "WriteRows in src/export/csv.go."
        sources: [src/export/csv.go]
      - title: "Quote commas in exported cells"
        ...
    why: "Two needs, each proved by its own test."
```

An issue too big to be one need is broken into its parts (ADR-0022):

- **Checked**: 2 to 6 children (`split-size`), each a title of one line,
  120 characters at most, no two alike (`split-title`), its four sections
  written (`split-sections`), its Scope's files 1 to 5, files of the
  commit (`sources-unknown`); the issue's state must read. One whose state lists
  children already is never split again (`already-split`); nor twice in one
  run, nor renamed twice: the first that passes its check is kept
  (`once-a-run`).
- **Each child** is opened through the one way ("Opening issues"), keyed by
  the parent and its title, case and spaces aside
  (`split=<parent>/<8 hex>`): a line `Part of #12.`, its four sections — Need and Validation as drafts, as `refine`
  writes them —, "Opened from #12 by the product-owner role."; the labels
  `workline:to-refine` and `workline:draft`; its own state, its sources,
  confirmed at the commit. Not `needs-triage`, not counted in
  `issues-max`. A child closed already is left closed (`issue-closed`),
  and still counted among the parent's children.
- **Linked to its parent** (`AddSubIssue`): a sub-issue on GitHub; on
  GitLab a task — converted and given its parent through GraphQL, read
  and acted on by REST afterwards as any issue; elsewhere, or when the
  forge refuses, a task list in the parent's body, under `## Sub-issues`
  (`- [ ] #13`), after its text, which stays.
- **The parent** keeps its need, milestone and priority; its state gets
  the children (`split`), last — a run stopped half-way, resumed, finds the
  children it opened — and the digest of its body when the engine listed
  them there, so it is not read again for its own change. A split moves
  nothing: it is not counted in the moved share; the children, never read
  yet, are read and ordered at the next run.

## Renaming

```yaml
- rename: {issue: 12, title: "The CSV export drops the last row", why: "\"bug\" says nothing of the problem."}
```

Sets the title alone, one line, 120 characters at most (`rename-title`);
the title it has already is dropped (`title-same`). The issue's state
records the title set. The title an issue was opened with is its
reporter's and may be renamed; after the role's, a title other than the
one recorded is a person's: the act is dropped (`title-kept`), the task
shows it as a person's. No quote; the state must read.

Split and rename on an issue opened by someone without write access are
proposed, not done (`reporter-outside`), as moving it to ready — to the
project, in the report: the text of a need is the reporter's to agree to
("The conversation with the reporter"), how the backlog cuts and names it
the project's. The label `workline:accepted` lifts all three.

## The conversation with the reporter

Each comment the engine writes to a reporter is a round (ADR-0021): an
`ask`, or a `refine` proposed to an outsider. Its marker holds its round —
`<!-- workline:product-owner/ask -->` for the first question,
`ask=2`, `ask=3` after; `proposal=1`, `proposal=2` for a text proposed. The
issue's comments are the record; the state does not copy them.

| Check | Otherwise |
|---|---|
| A person commented after the last round (a comment not ending with an engine marker: one quoting the engine's is a person's) | Dropped: `already-asked`, or `already-proposed` for a text; nothing written |
| No question the act holds — each from the sentence before it (". " or "! " then a capital; "e.g. Which" cuts too) or its list item's line (one starting with a capital) to its `?`, the bullet, spaces and case aside — is one an earlier round asked, its lead ("to refine this issue:", "What it still needs:") aside | Dropped: `asked-before` |
| Rounds before it under `acts.ask.rounds` (three) | Proposed in the report: "Settle #N with its reporter, written to 3 times already", the questions or the text it would write (`asks-spent`) |

A later round thanks the reporter ("thank you; to refine this issue,
still: …"). The agent is given the conversation in order — the engine's
rounds among the people's comments — and the rounds spent.

**An outsider's issue** — its reporter without write access, not opened by
a role, not bearing `workline:accepted` — is theirs: a `refine` writes
nothing in its body. The engine comments to the reporter, naming them:
what the role understood (its `why`), the sections it would add (Need and
Validation as drafts from their words), what it still needs (`questions`),
and how to agree; the sections also in a YAML block, under "As the engine
reads it" — the block read back, the agent's fences turned to `'''` so none ends or replaces it. The issue gets `workline:to-refine`; the act counts in
`refine`'s cap. Agreement is read where only the right people write:

- **`workline:accepted`**, set by a person of the project: the next run
  writes the sections of the last proposal the body still lacks, with no
  agent, then moves the issue to ready in the same run, the drafts
  accepted with it — unless the reporter answered it since: the agent
  reads the answer first, and refines it itself, the issue accepted;
- **the reporter's own edit** of the body (only its author or a writer
  can edit it): read again as any body changed; moving it to ready stays
  proposed (`reporter-outside`) until a person accepts;
- **a reply `agreed`** (ADR-0021, amended): after the last round, that
  round a proposal, the last comment of the reporter or of a person of
  the project (not a bot, its author named) has `agreed` as its first line
  (case, spaces, a final "." or "!" aside); a stranger's or a bot's
  comment after it changes nothing, a later word of theirs replaces it.
  With `workline:accepted` set too, the label's path applies at once: the
  text written and the issue moved to ready. The next run writes the sections of the last
  proposal the body still lacks, with no agent, Need and Validation as
  drafts, and does not read the issue again for that reply; ready, a
  split, a rename stay the label's. The plan checks the agreement again
  on the forge, and that the sections are those proposed (`not-agreed`).

Any other reply is read as an answer — the agent may propose a revised
text, a round. Each comment is given to the agent with who wrote it: the
reporter, of the project, outside it, a bot.

## Autonomy and caps

Each kind of act has a mode and a cap per run, set in the role's settings:

```yaml
acts:
  open: {mode: act, max: 30}
  sources: {mode: act, max: 10}
  refine: {mode: act, max: 5}
  ready: {mode: act, max: 5}
  ask: {mode: act, max: 3, rounds: 3}       # rounds: written to a reporter, then a person
  split: {mode: act, max: 2}
  rename: {mode: act, max: 5}
  milestone: {mode: act, max: 10}
  order: {mode: act, max: 10}
  close-duplicate: {mode: act, max: 3}      # act | propose | off
  close-obsolete:  {mode: act, max: 3, days: 7, exempt: [pinned, security]}
```

- `act`: done, up to `max` a run; past it, proposed.
- `propose`: written in the report issue for a person, not done.
- `off`: dropped.

A run moves at most a share of the open issues, the report issue left
out: `moved-percent-max` (20, a fifth, by default; rounded up, one at
least). Milestones — the engine's slips included — and priorities count
together, an issue moved twice once; past the share, a move is proposed
(`moved-cap`). Milestones are applied before priorities.

Closing as obsolete is `act` by default since it announces first, and
closes only on silence and a second judge's yes (ADR-0024; ADR-0018: a
quote proves the text is there, not that the issue is solved). Its `max`
counts announcements and closings together, the engine's closings first;
a closing past it stays announced and is closed at a later run. Neither
counts in the moved share: a closing takes an issue out of the backlog,
it does not reorder it.

## The report

One issue, kept in place (`KeepIssue`, title "Backlog — product owner"),
lists what the last run did and what is proposed, each proposal a box a
person of the project ticks to have it done ("The person's hand"), each closing with its
quote and how to undo it — a rename with the title it had, a split with
its children's titles, to close; an announcement with the day it may close
and how to keep it open; an issue kept open, and why. Under "Before this run", each issue the run
moved is listed with its priority and milestone as they were, to put the
order back. A proposal stays there from run to run until a person
settles it — its issue closed — or a run decides it again. An issue to
open past the cap stays there until an open issue holds its text: an
import run again opens it. Its own engine
comment (`<!-- workline:sticky=product-owner/acts -->`) records the
closings done, the kinds dropped back to `propose`, the proposals — each
with its line and the act as decided (`proposal`) —, the runs nobody
answered (`ignored`) and the comments of people of the project on the
report (`comments`).

## The person's hand

Each proposal is a box, its line ending with a hidden key,
`<!-- workline:proposal=<issue>/<kind> -->` (an issue to open: its text's
key). At each run, acts or not, the engine reads the boxes ticked in the
report's body and who ticked each (ADR-0025):

| Forge | Who ticked it |
|---|---|
| GitHub | the editor of the body's version that ticked it (`userContentEdits`, the hundred newest; after a gap — older versions, one deleted — the next version ticks nothing); of the project when GitHub gives them write, maintain or admin — a token that may not read that: nobody known; a user of type Bot is a bot |
| GitLab | the author of the system note "marked the checklist item … as completed"; of the project from the Planner role (ADR-0023) |
| local | a person of the project, unnamed: whoever works in the clone |
| plugged | its `ticks` operation (docs/spec/forge-command.md); one it refuses: nobody known |

| Ticked by | What the engine does |
|---|---|
| A person of the project, a proposal it can do | Done as the record holds it, never as an intention says (`not-ticked`), whatever its mode or cap; checked again as any act, dropped and said when it no longer holds. A closing as obsolete is closed at once, no announcement nor second judge, naming who ticked it |
| A person of the project, the box of a kind back to propose | That kind back to `act` (`back-to-act`); the settings untouched |
| A person of the project, an issue to open, rounds spent, a slip with nowhere to go | Not done: the report says to do it by hand; the proposal leaves it |
| Outside the project, a bot, or nobody the forge names | Nothing done (`tick-ignored`, the reason said in the report, under "Boxes ticked"); the box unticked when the report is rewritten |

A proposal ticked, done or dropped, leaves the record: a box still ticked
whose proposal the record no longer holds is nothing.

**Paused.** A run that read issues with an agent while the report held a
proposal is ignored when, since the last run, no person of the project
ticked a box or wrote on the report, no closing was undone and no
proposal's issue was closed. After three in a row (`ignored: 3`), the
report says **Paused**, with a box to resume; the next runs ask no
agent (`paused`: nothing read, no second judge) until a person does
one of those.

## Trust

A closing is wrong when its issue is open again. At the next run of the
role, acts or not, the engine reads the issues it closed; one open again
puts that kind of act back to `propose`, whatever the settings say, with a
finding `wrong-closing`, and is read again; the report says so, with a
box to set it back to `act`, its closings still closed and those reopened
beside it. Only a person of the project's tick sets it back ("The
person's hand").
