---
type: reference
sources: [internal/backlog/coverage.go, internal/builtin/productowner/productowner.go, cmd/workline/main.go, roles/product-owner/instruction.md]
status: draft
checked: fac02675
verified: agent:claude-code
---
# Importing a file

Part of [Acts on the backlog](../backlog-acts.md): moving a roadmap or a backlog file to the forge's issues, once.

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
- The agent is given the open issues by title, and the closed issues an
  import opened from the share's lines — found by their text in the file
  as it reads, else by the lines their body says they came from — each
  with how it was closed: an item one holds is answered `held`, not
  opened again. Reopening one is a person's.
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
  Its runs are listed as pending, as `workline route --no-apply` lists
  a line's (`to apply:`; `--json`: `pending`): `workline apply --line
  <result>`, in a job with the write token and no AI key, opens them,
  no agent called, each opening checked against the forge's issues
  then, as with `--apply`. The map stays the judge's,
  "would be opened"; the import run again maps them open.
  `open` is capped per share (`acts.open.max`): past it, an item is not
  opened (`act-cap`), and the import run again opens it.

## The map: every item to its issue, or why not

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
| past the cap | not opened; the import run again opens it |
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
