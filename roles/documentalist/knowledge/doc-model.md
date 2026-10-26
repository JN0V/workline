# How the docs are organised

**Every doc says what it depends on**, in a comment at its very top, which
no preview, forge or PDF shows:

```
<!-- workline
sources: [src/auth/token.go, docs/tech/auth.md#token-refresh]
checked: 3f2a91c
-->
```

A doc with a frontmatter of its own keeps these fields there, beside its
others:

```yaml
---
type: concept | task | reference | decision | card
sources: [src/auth/token.go, docs/tech/auth.md#token-refresh]
owner: team-or-person
verified: human:alice | agent:documentalist   # who last confirmed it is true
checked: 3f2a91c                               # commit it was last confirmed against
status: draft | stable | deprecated
---
```

A source in another repository is prefixed with that repository's name, as
declared in the project's config (`api:src/auth/token.go`), and `checked` then
holds one commit per repository (`checked: {api: 3f2a91c}`).

A doc describing no code — a decision record, a backlog, a changelog — says
so with `sources: []`, and has no `checked`.

**A doc is suspect** when a commit after `checked` touched one of its `sources`.
Suspicion follows the chain: code → technical doc → product doc.

**Cards** are short docs on one concept each. Each folder has a generated
`index.md` listing its cards; nobody edits an index by hand.

**Derived facts** — counts, versions, lists taken from the code — live between
markers and are regenerated on the default branch, never typed — not even on
a branch whose change makes them behind. A marker names a command the project
declares; the text between the markers is its output, inline or on lines of
their own:

```
<!-- workline:derive conformance-cases -->56<!-- workline:end --> cases pass.
```
