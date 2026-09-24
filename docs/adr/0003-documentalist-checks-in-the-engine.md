# ADR-0003: The documentalist's hygiene checks live in the engine

- **Status:** accepted
- **Date:** 2026-09-24

## Context

The research (docs/research/documentalist.md) points to tools for each check:
lychee for links, jscpd for repeated passages, Vale for per-section word
counts. The role planned to call them, listed in `uses`. Each is a separate
binary to install and pin in every CI job and on every laptop (principles 10
and 11), and none covers the whole need:

- jscpd finds exact copies only; near-duplicates need our own code anyway.
- lychee's value is links to other sites, which need the network (principle 5).
  Links inside the repository — files and headings — are plain lookups.
- Vale counts words per section, but not lines per doc or folder, card sizes,
  or the agents' entry points.

Agents' patches raise a second question. Real runs showed Claude writing hunk
headers that count fewer lines than the hunk holds; plain `git apply` stops at
the count and silently drops the rest — in one run, the fix itself.

## Decision

- Budgets, duplicates (three-word sequences, Jaccard similarity, every pair of
  paragraphs), links inside the repository and identifiers gone from the code
  (DOCER, with `git grep`) are written in Go, in the role's built-in steps:
  about 550 lines, no dependency beyond git.
- Links to other sites are counted and reported as unchecked until lychee is
  wired in, as an optional `uses`.
- A `patch` is applied as its lines read: `git apply --recount`. The judge
  reads the diff the same way and compares its result with git's before
  anything is applied; each hunk must quote the doc at the lines it cites.

## Consequences

- The checks run wherever the engine runs, offline, with nothing to install.
- Comparing every pair of paragraphs is quadratic. It is instant at the size of
  a project's docs; a MinHash index replaces it if a project ever needs one.
- Style (Vale) and links to other sites (lychee) stay to be wired, and will be
  optional: when missing, the verdict says which check did not run.
- A wrong hunk count no longer loses lines; a wrong line number is still
  refused, because it means the agent did not read the doc it quotes.
