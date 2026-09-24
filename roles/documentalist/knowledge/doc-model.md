# How the docs are organised

**Every doc says what it depends on**, in its frontmatter:

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

A doc whose frontmatter would show on the forge — a README — carries the same
fields in a comment at its very top instead:

```
<!-- workline
sources: [cmd/workline, ci]
checked: 3f2a91c
-->
```

A source in another repository is prefixed with that repository's name, as
declared in the project's config (`api:src/auth/token.go`), and `checked` then
holds one commit per repository (`checked: {api: 3f2a91c}`).

**A doc is suspect** when a commit after `checked` touched one of its `sources`.
Suspicion follows the chain: code → technical doc → product doc.

**Cards** are short docs on one concept each. Each folder has a generated
`index.md` listing its cards; nobody edits an index by hand.

**Derived facts** — counts, versions, lists taken from the code — live between
markers and are regenerated, never typed:

```
<!-- workline:derive cmd="..." -->
...
<!-- workline:end -->
```
