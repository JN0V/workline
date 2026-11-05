---
type: reference
sources: [internal/codepick, internal/codenav, internal/builtin/productowner/productowner.go]
status: draft
checked: b729cc4f
verified: agent:claude-code
---
# The code picker

How the product owner, the tester and the reviewer get the code an issue
names: one picker, any language, no build
([ADR-0063 §8](../adr/0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them)).
Each role used to cut code its own way; round 3 of the C++ trial put 8 of
its 11 defects there.

```mermaid
flowchart LR
  text["<b>The issue</b><br/>title, Need, Steps, Scope"] --> names["<b>Names and paths</b><br/>read as Markdown"]
  names --> grep["<b>git grep -w</b><br/>each name"]
  grep --> plan["<b>The plan</b><br/>under the role's budget"]
  plan --> task["<b>The task</b><br/>the cut said"]
```

## What it reads

- **The text**: the title, the body (Need, Steps, Validation, Scope),
  a person's comments; for the product owner, the lines of the file an
  issue was imported from that name its item, never that file.
- **Paths named outright**: the Scope's words in backticks, the product
  owner's `sources`. A folder gives its own files, a pattern the files it
  matches, the tests aside when code matches too.
- **Paths in the text**: a file the commit holds, or the one file with
  that name; for the product owner, a link to a file on the forge too.
- **Names**: the words written as code, as the reviewer's judge reads
  them: in backticks, called (`f(`), or shaped as a name (`load_all`,
  `answerItems`). Not an acronym (`CSV`, `MQTT`): a format as often as
  code. Not a keyword nor a language's own type.

## What it gives, in order

Under the role's budget, each piece once a task:

1. **Lines named after a file**: `a.go:12`, `a.go:12-30`, "docs/x.md,
   lines 6353 to 6353". The lines, the function around each end, else
   15 lines either side.
2. **Each definition of a name**, whole, its comment up to 20 lines
   above ([code navigation](../research/code-navigation.md): Go by its
   parser, other languages by a pattern and braces or indentation; C++
   methods defined out of their class too).
   - Those in a file named first. A name defined in a file named is that
     one: one of the same name elsewhere is counted, not given.
   - Then the project's own, 12 a name at most.
   - Copies in tests, examples and tools last, 2 a name, the rest counted.
3. **The lines naming a name in a file named**: the function around
   each, 40 lines at most, else 3 lines either side.
4. **A file named, whole** when what it adds fits; else, nothing of it
   given yet, its first lines.
5. **The other places naming a name**, the rarer name first, code before
   docs, 6 a name at most, the rest counted.
6. **The tester's test**: the nearest file matching `test-paths` (its
   folder's, named after the code, else its module's, else the project's
   first). A fifth of the budget is kept for it while the code is
   planned, said when the code wanted it.

## What the task says

- **Each file**: whole, its first lines (`cut: the first N lines`), or
  its pieces, each with why it is given.
- **The cut**: a piece the budget ran out in, where it goes on to; what
  is named past the budget; files named with nothing given, by size.
- **Counted, not shown**: definitions past the caps, other places
  naming a name.
- **One step away, as paths**: the files the code given includes or
  imports (C and C++ `#include`, Python and JavaScript imports), and
  those defining the types it derives from.
- **Given above**: a file given for an earlier issue of the same task.

## Read again only for code not given

- **The product owner's pre** lists the files it gave, by issue
  (`code-given` in the run's folder).
- **A `sources` naming only those, beside a Scope**: the Scope written
  in the same run stands, no `needs-code`, nothing read again
  ([refining](backlog-acts/refine.md)). Without a Scope, or beside an
  `ask`, the issue is read again as before.
- **Naming code not given** is still read again, with that code
  (`needs-code`, the "name the code, be read again" protocol of agents
  with no tools).
- **Its Scope names every file the change edits**, each copy of a
  function it fixes; `sources` holds up to 5 of them.

## Each role

| Role | Budget | Shown |
|---|---|---|
| [Product owner](backlog-acts/reading.md) | `code-lines-max` (1500), shared among the issues read | numbered lines, each file's last commits |
| [Tester](../../roles/tester/README.md) | `code-lines-max` (1500) | lines as they are, the tests beside after |
| [Reviewer, a spec](../../roles/reviewer/docs/spec.md) | `code-lines-max` (600); the judge's `judge-lines-max` (200) | numbered lines |

- **Same everywhere**: the same picker on a person's machine and in CI,
  with any agent or none.
- **Not here**: the reviewer's judge on a change's finding reads from the
  finding's cause ([judge](../../roles/reviewer/docs/judge.md)); agents
  with read tools come later (step 9 of ADR-0063).
