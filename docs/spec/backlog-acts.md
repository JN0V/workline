---
type: reference
sources: [internal/backlog, internal/builtin/productowner, internal/engine, internal/forge, roles/product-owner/role.yaml, roles/product-owner/instruction.md]
status: draft
checked: 964a9ab
verified: agent:claude-code
---
# Acts on the backlog

What the engine does when a role acts on a project's issues rather than on
its code (ADR-0018). The product owner proposes the acts; the engine checks
each one against the code and the forge, then does it, proposes it, or drops
it. This page is the contract; the acts built are closing, naming an
issue's sources, putting it in a milestone, and opening one from a file.

## The issue's state

What the engine knows of an issue lives in one comment it keeps on that
issue, marked `<!-- workline:sticky=product-owner/state -->`:

```yaml
sources: [src/export/csv.go#WriteRows]   # what the issue is about, in the code
confirmed: 1a8e5a4                       # the commit it was last found true at
judged: 1a8e5a4                          # the commit the role last read it at
comments: 2                              # people's comments when it was read
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
person commented (`comments` counts people's comments read), or a person
reopened what the role closed. An issue with nothing new is not read again,
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

## Autonomy and caps

Each kind of act has a mode and a cap per run, set in the role's settings:

```yaml
acts:
  open: {mode: act, max: 30}
  sources: {mode: act, max: 10}
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
