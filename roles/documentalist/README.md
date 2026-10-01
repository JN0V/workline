---
sources: [roles/documentalist/role.yaml]
checked: d30d22a
judged: 6013a50
verified: agent:documentalist
---
# Documentalist

What runs, for humans. The AI never reads this file.

It keeps the docs true to the code they describe, in four pages:

- [The checks](checks.md): what `pre` finds with no AI — suspect docs,
  budgets, duplicates, links, identifiers gone, derived blocks, superseded
  decisions, code no doc describes, freshness, line counts off — their
  levels, and how cascades are cut.
- [The agent's tasks](tasks.md): suspect docs judged whole or in parts, and,
  when gardening, stale docs, duplicates, condensing, merging and splitting
  cards, one merge request per task.
- [The judge](judge.md): what `post` refuses of the agent's patches, and
  a replaced version left beside a fix.
- [At the push, the release and the adoption](push.md): the merge request,
  `workline docs`, the release held by suspect docs, `workline init`, and
  what runs without AI.

Where the role stands: [status.md](status.md).

## Tried for real

Each try on a real repository or with a real agent, what it showed, and the
defects it found, now guarded, are in [tried.md](tried.md), newest last.
