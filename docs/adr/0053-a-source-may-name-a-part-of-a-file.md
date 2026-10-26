# ADR-0053: A source may name a part of a file

- **Status:** accepted; built
- **Date:** 2026-10-26
- **Decided by:** the maintainer, on 2026-10-26: option a after
  [ADR-0051]'s step 7, narrow the sources
- **Amends:** [ADR-0014] (what makes a doc suspect, what a task reads
  whole)
- **Builds on:** [ADR-0012], [ADR-0051]; principles 4, 12, 13

## Context

- **88 of 91 docs name a source over the 20,000 characters a task reads
  whole** ([ADR-0014, amended 2026-10-21][suspect-by-name]): the night
  can move `checked` on almost none.
- **One file makes many docs suspect**: on `main` at a91e3e78, 10 of the
  28 docs suspect were so through `ci/gitlab/workline.gitlab-ci.yml`, one
  hunk in `workline:follow`, which most of them do not describe.
- **A source could already name a part of a doc**:
  `docs/tech/auth.md#token-refresh`, a heading's anchor as the forge gives
  it.

## Decision

- **The same `#` names a part of a code file**:
  - Go: a declaration by its name, `settle.go#settleBranch`; a method as
    `Type.Method`; its doc comment with it. In a grouped `const ( … )`,
    only the spec named.
  - YAML: a key, its path dotted from the top, `ci.yml#jobs.test`; the
    longest key matches first, so `.workline:bot` and `workline:warn:apply`
    are keys. The comment lines right above the key are part of it.
  - Any other file with a `#`: refused loud, never read as the whole file.
- **Suspect only when the part changes**, the rule by name applied to the
  part's lines; a part gone is said ("part #X of f is gone"), as a
  heading gone is.
- **A task reads the part, not the file**: a doc whose parts fit
  `whole-chars` is judged whole, and the night can vouch for it.
- **Parsed with what the engine has**: Go's `go/ast`, the YAML parser
  already in `go.mod`. No new tool.

### Why this syntax

- **One rule for every source**: `#` already names a part of a doc; a
  reader of the header learns one thing.
- **Readable in front matter**: a name a person can grep, no line numbers
  that move at each edit.
- **What else exists**: line ranges (`f.go#L10-L40`), as forges link them,
  break at each edit above them; a symbol is what IDEs and `go doc` name.

## Consequences

- **What escapes**: a change outside the part the doc depends on, a helper
  the named function calls, a YAML anchor (`<<: *bot`) a job merges in.
  Name the helper or the anchor too; staleness and the weekly sample read
  the doc again ([ADR-0014, amended][suspect-by-name]).
- **Judged in parts** ([ADR-0009]) still reads whole files: its parts cite
  lines by number.
- **Measured** on workline, `--ai none`, the engine of this change; the
  narrowed docs listed in its pull request:
  - docs whose sources fit a task whole: 2 of 99 → see the pull request;
  - docs suspect on `main` (gardening's count): 28 → see the pull request.

[ADR-0009]: 0009-docs-far-behind-are-judged-in-parts.md
[ADR-0012]: 0012-a-doc-is-judged-against-its-sources-whole.md
[ADR-0014]: 0014-checked-is-earned-by-what-was-read.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[suspect-by-name]: 0014-checked-is-earned-by-what-was-read.md#amendment-2026-10-21-suspect-only-when-a-line-it-names-changes
