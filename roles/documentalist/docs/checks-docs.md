---
sources: [internal/builtin/documentalist/hygiene.go, internal/builtin/documentalist/external.go, internal/builtin/documentalist/adr.go, internal/builtin/documentalist/unread.go, internal/builtin/documentalist/reader.go, roles/documentalist/role.yaml]
checked: b9e7900b
verified: agent:claude-code
---
# Documentalist — the checks on the docs alone

The six checks of `pre` that read the docs alone, not the code they
describe; no AI. The other six, and how much each finding weighs:
[the checks](checks.md). Part of [the documentalist](../README.md).

At a push and on a merge request, they read only the docs its commits
touched or made suspect, and passages written twice are not looked for:
main's run after a merge and gardening read every doc
([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
[main after a merge](../../../docs/main-after-merge.md)).

## Budgets

- Lines per doc (its body: the header records who checked it, it says
  nothing), per folder, and of the agents' entry points at the root
  (`AGENTS.md`, `CLAUDE.md`).
- Words per section (its own text, up to the next heading).
- Card size, min and max: too small is fragmentation.

## Duplicates

Paragraphs of at least `min-words` words compared by their three-word
sequences; `similarity` 0.85 catches a passage copied and then lightly
edited.

## Links

- Every link to a file of the repository, and to a heading in it, must
  lead somewhere.
  - A heading's anchor is GitHub's, whatever the forge: its shown text
    lowercased, punctuation dropped (en and em dash too), spaces made
    hyphens, a repeat numbered `-1`, `-2`. GitLab documents the same rule.
- Links to other sites need the network, and break with no change to the
  repository: on `schedule` only, [lychee](https://lychee.cli.rs) checks
  them, with its own `lychee.toml` and `.lycheeignore` if the project has
  them.
  - A link answered by an HTTP error is `external-link-broken`, at its
    line; one not reached (no network, a timeout) is only counted.
  - Otherwise, or without lychee, they are counted and said to be
    unchecked (`links-not-checked`); not on main after a merge, whose
    issue lists problems, not the night's checks.

## Superseded decisions

A doc citing a superseded decision record, by a link to its file or by its
number (`ADR-0003`), without naming what replaced it (`cites-superseded`),
as coherence does.

- **Records**: the numbered files (`0007-…`, `adr001-…`) of a folder named
  `adr`, `adrs`, `adr-…`, `decisions`, `architecture-decisions`,
  `decision-records` or `decision-log`, and the files of the project's
  `decisions` globs.
- **Superseded**: its status says so — the frontmatter's `status`, a
  `Status:` line, or the paragraph under a `Status` heading (adr-tools) —
  or another record's frontmatter `supersedes` it (MADR). What replaced it
  is the record its status names.
- Records themselves are history, and may cite it.

## Docs not read

On `schedule` and `init`, one finding (`docs-not-read`) counts the files
that read as docs and that the role does not read, a few of each kind.
`workline doctor` says the same.

- **Counted**: `.rst`, `.mdx`, `.adoc`, a `.txt` under a `docs` or `doc`
  folder, a README outside the `docs` globs (package READMEs). Left out:
  test folders, fixtures, dependencies, hidden folders, `requirements.txt`.
- **How to include them**: a README is Markdown, read once a glob in `docs`
  takes it (`"**/README.md"`); the other formats are not read yet, no
  header form being settled for them.

## Docs for their reader

On `schedule` only
([#193](https://github.com/JN0V/workline/issues/193)), whether user pages
serve their reader. User pages: the docs matching `reader.pages`, but
`reader.skip` (specs) and the records (decisions, research, history). One
finding a page, with its lines:

- `reference-unlinked`: a decision (`ADR-12`), a doc of the repository or
  an issue (`#128`) named as plain text. Left alone: a name in a code span
  (an example) or an HTML comment; a link's definition
  (`[x]: docs/x.md`) is a link.
- `paragraph-too-long`: a paragraph, or a list item, over
  `reader.paragraph-words` (80).
- `cell-too-long`: a table cell over `reader.cell-words` (30).
- `flow-without-diagram`: a page of `reader.flow`, or with a heading
  holding one of `reader.flow-headings` ("how it works"; as words, any spacing or
  case, in the docs' language), with no diagram (a Mermaid, PlantUML,
  Graphviz or D2 block, or an image).
- `page-unreachable`: no link reaches it from `reader.navigation`
  (`README.md`), through user pages; a link to a folder reaches its
  README, one in an HTML comment nothing. With no such entry,
  `navigation-missing` says so.

Borrowed from Vale and Sphinx, without running them:
[research](../../../docs/research/documentalist.md#docs-for-their-reader-2026-10-07).
With an agent, gardening also reads the pages these checks flag
([pages read for their reader](reader.md)).
