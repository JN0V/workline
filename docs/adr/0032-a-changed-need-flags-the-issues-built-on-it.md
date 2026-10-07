# ADR-0032: A changed need flags the issues built on it

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 (an issue is read again only when something
  changed), ADR-0022 (a split keeps the need; its children), ADR-0025 (a
  proposal is a box a person ticks), ADR-0028 (what an issue waits on),
  ADR-0030 (an import keys each issue by the text it was opened from);
  principles 1, 4, 5, 6, 9, 12
- **Settles:** #165

## Context

An issue is read again when its own code changed, a person commented,
or a person changed its body. The change stopped there: when a person
rewrote a parent's Need, or a roadmap an issue was imported from
changed, the issues built on it kept their sections and their `ready`
label, and were built against a need that was gone.

Requirements tools settled this long ago with *suspect links*
(docs/research/product-owner.md, "A changed need"): a link remembers
the parent as it was when last reviewed; a change to chosen fields
makes the link suspect; a person clears it once checked. None of them
rewrites the item downstream.

## Decision

### What a change is

- **A person's rewrite of an issue's Need or Scope.** Each issue's state
  keeps those two sections as last read or written by the engine
  (`sections`), the engine's own lines — a draft line, a blocked-by line
  — left out. Another text there, spaces aside, is a change; a section
  written where there was none, an edit elsewhere in the body (a typo
  above), a label are not. A state that kept none yet gets them, once,
  with no agent — so the first change after this is seen.
- **A commit changing the lines of a file an issue was imported from**
  (its `import=` key, the lines its body names): carried from the commit
  the issue was opened at to the one it was last read at, then through
  the commits since; lines changed, removed, or added between them are a
  change; lines that only moved are not.

### What it touches, and what is done

| Touched | Done |
|---|---|
| an open part of the issue changed (ADR-0022, ADR-0029) | read again first, at the run that finds the change, with the text as it was and as it is |
| the issue opened from the lines | read again first, the lines as they were and as they are |
| an open issue waiting on it (ADR-0028) | listed for a person, not read |
| its Scope changed: an open issue whose sources share a file with its own | listed for a person, not read |

Only these direct consequences are followed (principle 9): an issue
read again is not itself a change to what others were built on. A Need
rewritten moves what is built on it — its parts, what waits on it —, not
every issue on the same code: tried on the sandbox, a Need's change
listed nine issues that only named the same file, all noise; the code
shared counts when the Scope, the part of the code the issue holds,
changed.

**Every act on an issue read again for a change is proposed**, never
done (`need-changed`): the change is a person's, and so is what it asks
of the issues built on it. A new act, `unready`, moves a ready issue
back to refine — `workline:ready` off, `workline:to-refine` on, the issue
told why — and is always proposed, whatever the settings, a person's
tick doing it.

### In the report

Under **Changed needs**, each change is a box — the issue, the sections
or the file's lines, the day found — with each issue it touches below:
how, whether it was read again, and the kinds proposed for it. The
record keeps it (`changes`) until a person of the project ticks it
checked, or its issues are all closed. Nothing is written to the issues
touched.

### Flags once

The changed issue's state keeps its new sections at the run that finds
the change — after the agent's answer when it was given a part to read;
an issue read keeps its line's commit as `judged`: the same change is not
found again. A part past `issues-per-run`, or every part without an
agent, is listed for a person instead of read.

## Consequences

- A rewritten need no longer leaves its parts `ready` in silence; a
  person reads, in one place, what each part is and what is proposed.
- Each state comment grows by the Need and Scope it read; each one is
  edited once, with no agent, when it lacks them.
- Not done: following a change further than one step; a change to
  Verification or Validation (what proves and who accepts it, not what is
  built); an imported file renamed (its lines read as gone).

## Amendment (2026-10-07): no box with nothing to decide; an archived file

On workline's own backlog the report asked a person to tick ten changes,
nine of them "read again with the change: nothing proposed", each found
because the backlog file the issues were imported from — archived since,
the issues its source of truth — was edited.

- **Read again, nothing proposed: settled with no person.** A change
  whose every open issue was read again with it and has no proposal
  waiting leaves the record; the report says it once, in a folded line.
  The run that read them decided there was nothing to change; a box would
  ask a person to confirm nothing.
- **Proposals waiting: their boxes are the change's.** The change is said
  beside the issue's title, above its proposals; it has no box of its
  own, and settles once they do.
- **An issue not read** — listed for a person, past `issues-per-run`, no
  agent — keeps the box, under "To check", until a person ticks it.
- **A file archived flags nothing**: `archived`, a list of paths or globs
  in the role's settings, names the files no longer a source; a change to
  their lines is not one, and a change the record held from one leaves
  it. A setting, not a mark in the file: the file need not be one the
  engine can parse, and the project says it where its other settings
  are.

