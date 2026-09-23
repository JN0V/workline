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

**A doc is suspect** when a commit after `checked` touched one of its `sources`.
Suspicion follows the chain: code → technical doc → product doc.

**Cards** are short docs on one concept each. Each folder has a generated
`index.md` listing its cards; nobody edits an index by hand.

**Derived facts** — counts, versions, lists taken from the code — live between
markers and are regenerated, never typed:

```
<!-- assembly:derive cmd="..." -->
...
<!-- assembly:end -->
```
