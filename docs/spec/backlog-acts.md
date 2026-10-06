---
type: reference
sources: [internal/backlog, internal/builtin/productowner, internal/sample/acts.go, internal/engine, internal/forge, roles/product-owner/role.yaml, roles/product-owner/instruction.md]
status: draft
checked: dc0a562
verified: agent:claude-code
---
# Acts on the backlog

What the engine does when a role acts on a project's issues rather than on
its code (ADR-0018). The product owner proposes the acts; the engine checks
each one against the code and the forge, then does it, proposes it, or drops
it. This page is the contract; the acts built are closing, naming an
issue's sources, putting it in a milestone, ordering it, opening one from
a file, refining one to ready — and back —, splitting one — and
reporting on a parent as its parts close —, renaming it, naming what it
waits on, and flagging the issues built on a need that changed.

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
sections:                                # its Need and Scope as last read or written (A changed need)
  Need: Every row exported, so a report counts what was sold.
  Scope: WriteRows in src/export/csv.go.
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
changed its body (`body`), or a person reopened what the role closed —
except the issues read again for a change to what they were built on,
read before all ("A changed need", below). An issue with nothing new is not read again,
however old (ADR-0018). An issue on the same code as one read is given with
its body beside it, up to six, as the original a duplicate would be closed
against; the others are listed by title only. Each issue read gets `judged`
moved to the run's commit; without an agent, or when its answer does not
read, none is.

### A changed need

A person's change to what open issues were built on flags them
(ADR-0032), as a requirements tool marks a link suspect; with no agent,
at every run:

- **An issue's Need or Scope rewritten by a person.** The state keeps
  both as the engine last read or wrote them (`sections`), the engine's
  own lines — a draft line, a blocked-by line — left out
  (`backlog.Basis`). Another text there, spaces aside, is a change
  (`backlog.Rewritten`); a section written where there was none, an edit
  elsewhere in the body, a label are not — the issue is read again for
  its body as before, and flags nothing. A state that kept none gets
  them, once, with no agent.
- **The lines of a file an issue was imported from, changed by a
  commit**: the lines its body names (`Opened from `ROADMAP.md`, lines 7
  to 8`, beside its `import=` key), carried from the commit it was
  opened at (`confirmed`) to the one it was last read at (`judged`),
  then through the commits since (`backlog.LinesChange`): lines changed,
  removed, or added between two of them are a change; lines that only
  moved are not. A commit that cannot be read — gone after a force-push,
  beyond a shallow clone — is said (`lines-unread`, warn), never read
  as no change.

What a change touches (`backlog.Touched`), none twice, in this order:

| Touched | What the run does |
|---|---|
| an open part of the issue (`backlog.Parts`) | read first, before the issues proposed for a cap, the sections as they were and as they are given beside it |
| the issue imported from the lines | read first, the lines as they were and as they are |
| an open issue waiting on it (`backlog.Blockers`) | listed in the report, not read |
| its Scope rewritten: an open issue whose sources share a file with its own | listed in the report, not read |

Every act the agent proposes on an issue read for a change is proposed,
not done (`need-changed`, info), `keep` aside: what the change asks of it
is a person's to decide. A part past `issues-per-run`, or every part
without an agent or while paused, is listed instead of read. Nothing is
written to an issue touched. The changed issue's state keeps its new
sections at that run — after the agent's answer when a part was given
to it —, an issue imported its new `judged`: the same change flags once.
A change that touches no open issue is kept in the state and flags
nothing. Only these direct consequences are followed (principle 9).

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
- Without `--apply`, it says what it would open and writes nothing.
  `open` is capped per share (`acts.open.max`): past it, an item is
  proposed in the report, and the import run again opens it.

### The map: every item to its issue, or why not

A requirement the agent skipped — judged done, taken for an
introduction, lost at a share's edge, or missed — must leave a trace
(ADR-0030). The agent answers for every line of its share that is not
blank, an item cut by the share's end left to the next share: an `open`,
or a `skip` that says why the item is not opened — never applied, kept
in the run folder (`out/skips.yaml`):

```yaml
- skip: {lines: "9", reason: done, quote: {path: ROADMAP.md, text: "Done in 1.2."}}
- skip: {lines: "14", reason: held, issue: 3}
- skip: {lines: "1-5", reason: not-item, why: "the title, the introduction and a heading"}
```

The engine builds the map from the shares it cut and these answers, and
checks each reason: a `done` quote found in the file (or the file it
names), as written but for spaces; a `held` issue that is on the forge;
a `not-item` that says why. Each item, by its lines and first words,
goes to:

| Entry | When |
|---|---|
| `#n, opened` | opened by this import — read from the run folder (`out/openings.yaml`), a forge's list lagging behind a new issue |
| `#n, already open` | an open issue held its text before the import (its marker), or the agent named it (`held`) |
| `#n, closed` | a closed issue holds its text: not opened again |
| would be opened | without `--apply` |
| past the cap | proposed in the report; the import run again opens it |
| done | the words that say so, quoted |
| not an item | why |

Every line not blank that no entry holds is listed under **Not
covered**, a paragraph an entry: a quote not found, a reason that does
not check (`no-quote`, `skip-unread`), or lines the answer left out —
the share that was to answer for them flagged (`items-omitted`), each
share answering for its lines up to where the next one starts. Then the
summary says how many, and the import ends `human` (exit 2): a person
reads them, then runs the import again or opens them by hand. Without
an agent (`--ai none`), every line is Not covered. With `--json`, the
map is the result's `coverage`: `items` and `not-covered`, each with
`lines`, `words`, `state`, `issue`, `why`.

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
number (`backlog.Less`) — and an issue that waits on an open issue after
it, whatever its labels ("What an issue waits on", below;
`backlog.Order`). The task lists the issues not read in that order, each
marked with the open issues it waits on.

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
- **`unready`** moves a ready issue back to refine: `workline:ready`
  off, `workline:to-refine` on, a comment telling the issue why and who
  ticked it. It is **always proposed**, whatever the level or the
  settings — a ready issue is moved back by a person only (ADR-0032) —,
  done when a person of the project ticks it; one not `workline:ready`
  is dropped (`unready-not-ready`). No quote; the state must read.

  ```yaml
  - unready: {issue: 13, why: "#12 now asks for JSON: its CSV row count no longer proves it."}
  ```

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

### A parent and its parts

A parent is accepted by a person, from what its parts delivered
(ADR-0029). Its parts are the forge's own relation (`Issue.Children`:
GitHub's sub-issues of the same repository, listed for the parents its
listing counts; GitLab's tasks, one GraphQL query), the task list under
`## Sub-issues` in its body (`- [ ] #13`, a person's included) — one rule
for the comment, the report and the order (`backlog.Parts`); a part a
person unlinked is no longer one. An issue whose state lists a split
(`backlog.SplitInto`) is still never closed by the role.

- **Ready** means what it means on any issue; a parent is refined as any.
  It is never the first ready issue offered (`next-ready`): its parts are
  what is built.
- **Its comment**, at every run, with or without an agent, paused or not,
  on every open parent whose state reads: one comment marked
  `<!-- workline:sticky=<role>/parts -->`, edited in place, and not edited
  at all when its text is the same (`Sticky` compares first). A table of
  its parts — open (ready or not), closed as completed with what closed it
  (`Closers`: GitHub's `ClosedEvent.closer`, a pull request or a commit;
  GitLab's state events, the last closing's `source_commit` or merge
  request, found among `closed_by`'s; none: "by hand"; a forge that
  refuses to say: "not read", and a finding `closers-unread` (warn) —
  one that does not answer stops the run), closed as not planned or as a duplicate, or
  gone from the forge: those three not delivered. GitLab and the local
  forge keep no reason: a closed part there is taken as done. Then each
  item of its Verification — each list item, or the section whole — proved
  when a part delivered quotes it, in its own Verification or in the text
  of what closed it, case, spaces, `` ` ``, `*`, `_` and the final
  punctuation aside, said with where; "not proved" otherwise ("not proved yet" while a part is open). A parent
  without a Verification is said to have none.
- **All its parts closed**: the comment asks a person to accept the need
  by closing the parent, or to reopen a part or open one for what is
  missing, naming first the parts not delivered and the items not proved;
  a finding `parent-to-accept` (info) says it, and the report lists it
  under "To accept". The role never closes a parent nor labels it: a
  `close` on one, a duplicate or an announcement as obsolete, is dropped
  (`parent-accepted-by-a-person`) unless a person ticked it. The agent's
  task marks a parent's parts, open or closed.

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

## What an issue waits on

```yaml
- depend: {issue: 14, blocked-by: [13], why: "its test needs #13's fixed row count"}
```

An issue that cannot start before another is done names it (ADR-0028):

- **Kept** in the forge's own relation where it has one — GitHub's issue
  dependencies, GitLab's `is_blocked_by` link (Premium and up) — through
  `AddBlocker`; elsewhere — GitLab Free, which refuses the link for its
  license, the local forge, a plugged forge answering `{native: false}` —
  a line in the body, after its text: `Blocked by #13, #15.
  <!-- workline:blocked-by -->`, the one line the engine rewrites as it
  adds a blocker; the issue's state gets the body's digest, so the
  engine's own line is not read as a person's change.
- **Read** from both, on every forge: the relation, as the open issues are
  listed (`Issue.BlockedBy`; GitHub asks only the issues its listing says
  are blocked, GitLab one GraphQL query — an instance that answers it with
  errors has none), and every line of a body starting with "Blocked by"
  (case aside, a colon allowed) followed by issue references, a person's
  own line included (`backlog.Blockers`).
- **Checked**: the issue and each blocker open, not the report, not the
  issue itself, 1 to 5 (`depend-issue`); only the blockers not there
  already are written, none left drops it (`depend-same`); one that would
  close a cycle with the relations there and those this run set is
  dropped (`depend-cycle`); its state must read; no quote. It does not
  count in the moved share.
- **A split's child** names the siblings it waits on by their place in
  the split, from 1 (`after: [1]`); checked with the split (`split-after`:
  a place out of range, itself, or a cycle among them), written once the
  children are opened, as `depend` writes it.
- **The role only adds.** A relation a person set — a link, their own
  line — is read and kept, never removed; one the role set and a person
  took off, its blocker still open, is an act undone ("Trust").

### The order, and what waits

`backlog.Order` is Kahn's: the next issue is the first,
by `Less`, whose blockers among those ordered are all placed; a blocker
closed, or not an open issue, holds nothing back, so a blocker closed
puts the issue back where its labels say. When none can be placed, those
left hold a cycle: it is reported, the cycle's first issue by `Less` placed — never one that only waits on it —, and
the order ends — never followed. `ready` is allowed on a blocked issue,
its sections say it is understood, not that it can start; but **the
first ready issue offered** is the first in the order bearing
`workline:ready` that waits on no open issue and has no parts
(`backlog.Offered`, `backlog.NextReady`) — to whoever builds next, person
or developer role (#117); the report lists the first `next-max` of them
under "Next" ("The report").

Every run, with or without an agent, says it in its findings:
`next-ready` (info) the issue offered first, `waiting` (info) each issue
waiting on an open one, `dependency-cycle` (warn) each cycle, "#12 waits
on #14, #14 waits on #12". The report says the issues waiting and the
cycles too, under "Waiting".

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

How far the role goes is one setting, `autonomy` (ADR-0026): `cautious`,
`normal` (the default) or `enterprising`, a preset of each kind's mode and
cap. `normal` is the role's defaults:

```yaml
autonomy: normal
ignored-runs-max: 3                         # runs nobody answered, then a pause; 0 never
next-max: 5                                 # the ready issues the report opens with, 0 to 20; 0 none
stuck-days: 14                              # an issue waiting on a person longer is said stuck, 1 to 365
acts:
  open: {mode: act, max: 30}
  sources: {mode: act, max: 10}
  refine: {mode: act, max: 5}
  ready: {mode: act, max: 5}
  ask: {mode: act, max: 3, rounds: 3}       # rounds: written to a reporter, then a person
  split: {mode: act, max: 2}
  depend: {mode: act, max: 5}
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

### The levels

The other levels change these (role.yaml, `levels`):

| Setting | cautious | normal | enterprising |
|---|---|---|---|
| sources | act 10 | act 10 | act 10 |
| close-duplicate | propose | act 3 | act 5 |
| close-obsolete (announced first) | act 3, 14 days | act 3, 7 days | act 5, 7 days |
| milestone | propose | act 10 | act 10 |
| order | propose | act 10 | act 15 |
| moved-percent-max | 10 | 20 | 30 |
| refine | act 5, Need and Validation drafts proposed (`drafts: propose`) | act 5 | act 10 |
| ready (the engine's check, never a draft) | act 5 | act 5 | act 10 |
| ask | act 2, 2 rounds | act 3, 3 rounds | act 5, 3 rounds |
| split | propose | act 2 | act 4 |
| depend | propose | act 5 | act 10 |
| rename | propose | act 5 | act 10 |
| open (import) | act 30 | act 30 | act 30 |

`cautious` is for a project whose Product Owner is a person (`workline
init` asks): the acts that check facts are done, those that set direction
proposed. A refine there writes Scope and Verification and proposes the
Need and Validation drafts in the report (`drafts-proposed`) — unless a
person's already: agreed to in a reply, accepted by the label, or
proposed to an outsider in a comment. The level sets what is done with
what is read, not how much is read: `issues-per-run`, `code-lines-max`
and other roles' `issues-max` are their own.

**Precedence**: the level, then a kind the project sets, field by field
(`acts: {rename: {mode: act}}` at `cautious` keeps the level's cap); then
a kind demoted ("Trust"), proposed whatever the level until a person's
tick; then a person's tick on one act ("The person's hand"). The task
given to the agent lists each kind's mode and where it comes from —
`level`, `setting` (differs from the level's) or `demoted` —, and so does
the report, on its `Autonomy:` line. A level the role does not have, or
`ignored-runs-max` outside 0 to 20, `next-max` outside 0 to 20 or
`stuck-days` outside 1 to 365, stops the run.

## The report

One issue, kept in place (`KeepIssue`, title "Backlog — product owner"),
opens with what is next and what is stuck ("What is next, what is
stuck", below), then lists what the last run did and what is proposed, each proposal a box a
person of the project ticks to have it done ("The person's hand"), each closing with its
quote and how to undo it — a rename with the title it had, a split with
its children's titles, to close; an announcement with the day it may close
and how to keep it open; an issue kept open, and why. Under "Waiting",
each issue waiting on an open one, and each cycle. Under "To accept", each open parent whose parts are all closed,
recorded (`to-accept`) so the report is rewritten when that list
changes ("A parent and its parts"). Under "Changed needs", each change
to what open issues were built on ("A changed need"), a box — the issue
and the sections a person rewrote, or the file's lines, the day it was
found —, and under it each open issue it touches: how, read again or
not, and the kinds of act proposed for it; it stays until a person of
the project ticks it checked, or none of its issues is open, kept in the
record (`changes`). Under "Before this run", each issue the run
moved is listed with its priority and milestone as they were, to put the
order back. A proposal stays there from run to run until a person
settles it — its issue closed — or a run decides it again. An issue to
open past the cap stays there until an open issue holds its text: an
import run again opens it. Its own engine
comment (`<!-- workline:sticky=product-owner/acts -->`) records the
closings done, the kinds dropped back to `propose`, the proposals — each
with its line, the act as decided (`proposal`) and the day it was first
proposed (`since`) —, the runs nobody
answered (`ignored`), the comments of people of the project on the
report (`comments`), the role's own acts a person may undo (`done`, each
with the value before and the one set, and the level it was done at;
closings in `closed`, with their level) and those undone (`undone`, with
their evidence), what people did with the proposals at the level in
force (`measure`: ticked, settled otherwise), and the acts it did alone
(`did`: each act a run decided to do alone, with its kind, issue, level,
day and a line; for 35 days, at most 200; a person's tick and a slip
moved by the engine left out), for the weekly sample. From it the report may
suggest another level — at `cautious`, more than 80% of 10 proposals
settled or more ticked as proposed suggests `normal` —; it never changes
the setting.

### The weekly sample of its acts

`workline sample --apply`, after the docs' sample, reads the record of a
project with a report (ADR-0033): of the acts in `did` whose day falls in
the week sampled, one in ten, rounded up, drawn the same on a rerun, goes
to the tracking issue "workline: the weekly sample of the product owner's
acts", a comment a week, for a person to judge — each with its issue,
kind, day, level, and, when a person undid it since, what shows it (a
closing reopened, the record's `undone`). An act found wrong is undone on
its issue, which demotes its kind at the next run ("Trust"). Over the acts
`did` keeps at the level in force, up to the week's end, 10 or more: more
than one in ten undone suggests the level below; none undone at `normal`
suggests `enterprising`; at `cautious`, the report's suggestion stands.
The sample never changes the setting; a record that does not read is said
(`acts-not-read`), the docs' sample written still.

### What is next, what is stuck

The report opens with two lists (ADR-0031), computed by the engine at
every run, with or without an agent, from the forge as it is — nothing of
them stored:

- **Next**: the first `next-max` (5) issues of the order
  (`backlog.Order`) the first ready issue offered would be taken from
  (`backlog.Next`): bearing `workline:ready`, waiting on no open issue, no
  parts; each with its milestone and priority. At 0, no Next.
- **Stuck**: each issue waiting on a person for more than `stuck-days`
  (14) days, with the day it started and how long:

| Waits on | Since | Read from |
|---|---|---|
| someone to start it: `workline:ready`, no pull or merge request nor commit naming it since | the day it last got the label | the forge's `Trail`: GitHub's timeline, GitLab's label events and "mentioned in" notes, a plugged forge's `trail` |
| its reporter: the last round written to them, no person's comment after it ("The conversation with the reporter") | that round's day | the comment's `created`, as the forge gives it |
| a person's tick: a proposal of this report, unticked, unsettled | the day first proposed | the record's `since`, carried to the same act decided again; one recorded before, the day first read |
| a second judge: announced obsolete, due, neither closed nor kept | the day its delay ended | the announcement's `announced` and `close-obsolete.days` — listed once due, whatever `stuck-days` |

A link older than the label started nothing. An issue appears once, in
its first list: one in Next is not stuck; one waiting for two reasons is
said for the first in the table's order. Only the ready issues offered,
those in Next aside, are asked their trail. An issue whose day the forge does not say — the local
forge keeps none, a plugged one may refuse `trail`, a comment without
its day — is not said stuck, and the run says so (`stuck-unknown`, info;
warn when the forge failed to answer).

A run that writes nothing else still rewrites the report when its
opening no longer reads as the report's body says, a proposal has no day
yet, or, with no report open, when Next or Stuck lists an issue; how long
an issue has waited changes with the day, so while one is stuck the
report is rewritten once a run.

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
| A person of the project, the box of a changed need (`changed/<issue>`; `changed-lines/<issue>` for the lines it was imported from, a change of its own) | Checked: it leaves the report and the record, said under "Boxes ticked"; nothing done to its issues |
| A person of the project, an issue to open, rounds spent, a slip with nowhere to go | Not done: the report says to do it by hand; the proposal leaves it |
| Outside the project, a bot, or nobody the forge names | Nothing done (`tick-ignored`, the reason said in the report, under "Boxes ticked"); the box unticked when the report is rewritten |

A proposal ticked, done or dropped, leaves the record: a box still ticked
whose proposal the record no longer holds is nothing.

**Paused.** A run that read issues with an agent while the report held a
proposal is ignored when, since the last run, no person of the project
ticked a box or wrote on the report, no closing was undone and no
proposal's issue was closed. After `ignored-runs-max` in a row (3 by
default, 1 to 20), the report says **Paused**, with a box to resume; the
next runs ask no agent (`paused`: nothing read, no second judge) until a
person does one of those. At 0 the role never pauses: every run's
findings (`never-paused`) and the report say so, with the runs nobody
answered.

## Trust

A closing is wrong when its issue is open again. At the next run of the
role, acts or not, the engine reads the issues it closed; one open again
puts that kind of act back to `propose`, whatever the settings say, with a
finding `wrong-closing`, and is read again; the report says so, with a
box to set it back to `act`, its closings still closed and those reopened
beside it. Only a person of the project's tick sets it back ("The
person's hand").

Any other act of the role's a person undoes demotes its kind the same
way (ADR-0026), found at the next run — with or without an agent — from
the record's `done` and the forge, the evidence in the finding `undone`
and in the report:

| Act | Undone when |
|---|---|
| rename | the issue's title is the one it had before |
| order | its priority is the one it had before (none included) |
| milestone | its milestone is the one it had before (none included) |
| ready | `workline:ready` is no longer on the open issue |
| split | a child listed in the parent's state is closed as not planned (GitHub; GitLab keeps no reason) |
| depend | a blocker it added, still open, is no longer among the issue's blockers: the link or the line taken off |

A title, priority or milestone a person set to a third value is theirs:
the act is no longer watched, and demotes nothing. An act whose issue is
closed is no longer watched; a split, once each child is closed or gone
from the forge (deleted, moved); a depend, once each blocker it added is closed. A move
the engine made itself (a slip) and an act a person ticked are not the
role's choice, and are not watched. The record keeps the newest 200.
Changing the level never lifts a demotion.
