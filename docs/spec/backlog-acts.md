---
type: reference
sources: [internal/backlog, internal/builtin/productowner, internal/engine, internal/forge, roles/product-owner/role.yaml, roles/product-owner/instruction.md]
status: draft
checked: 5173ead
verified: agent:claude-code
---
# Acts on the backlog

What the engine does when a role acts on a project's issues rather than on
its code (ADR-0018). The product owner proposes the acts; the engine checks
each one against the code and the forge, then does it, proposes it, or drops
it. This page is the contract; the acts built are closing, naming an
issue's sources, putting it in a milestone, opening one from a file, and
refining one to ready.

## The issue's state

What the engine knows of an issue lives in one comment it keeps on that
issue, marked `<!-- workline:sticky=product-owner/state -->`:

```yaml
sources: [src/export/csv.go#WriteRows]   # what the issue is about, in the code
confirmed: 1a8e5a4                       # the commit it was last found true at
judged: 1a8e5a4                          # the commit the role last read it at
comments: 2                              # people's comments when it was read
body: 3f9a1c0e2b7d                       # a digest of its body, when last read or written
```

An issue without that comment, or with one that does not read, is never
acted on: an act on it is dropped (`no-state`, `state-broken`) and nothing is
written on the issue. The role's `pre` gives an issue it takes its first
state, `confirmed` at the commit it read, with no sources until an act
names them.

## Reading

A run reads at most `issues-per-run` issues, with at most `code-lines-max`
lines of the code they name: those never read first, then those with
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
  linking the original), an obsolete issue as completed; both with a
  comment that quotes the evidence and says how to undo: reopen it.

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
  issues; only who may triage sets a label, so it is a person of the
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
  unless a person of the project accepted it with the label. A forge that does not say who has write access
  (GitLab, for now) counts every reporter as outside.
- **`ask`** comments on the issue, naming its reporter, with the agent's
  questions — once an issue (`already-asked`). The answer is a person's
  comment: the issue is read again at the next run.
- An issue whose body a person changed since it was read — a draft
  accepted, a section written — is read again (its state's `body`, a
  digest of the body when it was last read or written by the engine).

Not built yet: splitting a need into sub-issues; renaming; asking again
after an answer; the reporter asked, rather than the report, for an
outsider's issue.

## Autonomy and caps

Each kind of act has a mode and a cap per run, set in the role's settings:

```yaml
acts:
  open: {mode: act, max: 30}
  sources: {mode: act, max: 10}
  refine: {mode: act, max: 5}
  ready: {mode: act, max: 5}
  ask: {mode: act, max: 3}
  milestone: {mode: act, max: 10}
  close-duplicate: {mode: act, max: 3}      # act | propose | off
  close-obsolete:  {mode: propose, max: 3}
```

- `act`: done, up to `max` a run; past it, proposed.
- `propose`: written in the report issue for a person, not done.
- `off`: dropped.

Closing as obsolete starts at `propose` (ADR-0018: a quote proves the text
is there, not that the issue is solved).

## The report

One issue, kept in place (`KeepIssue`, title "Backlog — product owner"),
lists what the last run did and what is proposed, each closing with its
quote and how to undo it. A proposal stays there from run to run until a person
settles it — its issue closed — or a run decides it again. An issue to
open past the cap stays there until an open issue holds its text: an
import run again opens it. Its own engine
comment (`<!-- workline:sticky=product-owner/acts -->`) records the
closings done, the kinds dropped back to `propose`, and the proposals.

## Trust

A closing is wrong when its issue is open again. At the next run of the
role, acts or not, the engine reads the issues it closed; one open again
puts that kind of act back to `propose`, whatever the settings say, with a
finding `wrong-closing`, and is read again; the report says so. Only the person sets it to `act` again. *Not built yet:
how the person does so, reading a tick with its author.*
