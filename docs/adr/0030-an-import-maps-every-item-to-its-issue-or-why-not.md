# ADR-0030: An import maps every item to its issue, or why not

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 (the one way issues are opened; the product
  owner keeps the backlog); principles 4, 5, 6, 7, 12, 14
- **Settles:** #163

## Context

`workline issues import <file>` reads a roadmap a share at a time; the
agent proposes an `open` for each item still to do. Its report listed what
it opened and what passed the cap — not what it did not open. An item the
agent judged done, took for an introduction, lost at a share's edge or
simply missed left no trace, and the file then "stays as it is, for its
history": a forgotten requirement was gone. On DomoticsCore's roadmap,
two runs of the same import left out different items (BUG-94, CI-7,
CI-16), and nothing showed it (roles/product-owner/docs/tried.md).

Requirements tools trace declared items — an id, a tag, a row — and list
what is not covered (docs/research/product-owner.md, "An import's
coverage"). A roadmap in prose declares none: which lines make an item is
judgement (principle 4).

## Decision

### The agent answers for every line

On `import`, the agent answers for every line of its share that is not
blank, an item cut by the share's end left to the next share: an `open`,
or a new intention, `skip`, saying why the item is not opened —
`{lines, reason: done, quote}`, the words that say it is done;
`{lines, reason: held, issue}`, the issue that holds it; `{lines, reason:
not-item, why}`, what the text is. A `skip` is never applied: the engine
keeps it in the run folder (`out/skips.yaml`), like a `claim`.

### The engine builds the map, and checks it

After the last share, the command builds the map from the shares it cut,
each share's answer and the forge's issues, open and closed, before and
after the import: each item, by its lines and first words, to the issue
that holds it — opened by this run, already open, closed — or to its
reason, checked: a `done` quote found in the file as written but for
spaces, a `held` issue on the forge, a `not-item` with its why; or past
the cap. Which issue an opening produced is read from the run folder
(`out/openings.yaml`), not from the forge's list, which can lag a moment
behind a new issue (seen on GitHub).

Every line not blank that no entry holds — a reason that does not
check, an `open` whose quote is not found, lines the answer left out — is
listed under **Not covered**, a paragraph an entry; the share that was
to answer for it, from its first line to where the next starts, is
flagged (`items-omitted`).

### Not covered is a person's

A map with anything not covered ends the import `human` (exit 2), its
summary saying how many: a person reads them, then runs the import again
or opens them by hand (principle 12). Without an agent every line is
not covered (principle 5). With `--json`, the map is the result's
`coverage`.

## Consequences

- The agent writes more per share — a `skip` an item done, a heading, an
  introduction —; lines given as ranges keep it short.
- An agent that covers a whole share with one `not-item` passes the
  check; the map shows the reason, for the person who reads it. The
  engine checks that every line is answered, not that the answer is
  right: that stays judgement, the person's at the end.
- The map is printed, not kept on the forge: the import is run by hand
  (docs/triggers.md), and its output is where the person looks.
