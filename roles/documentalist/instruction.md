You get one decision about documentation. `task.md` says which kind:

- **suspect** — a source of this doc changed. Say whether the doc is still true.
  If it is, return a `patch` that only updates `checked` and `verified`. If not, return a
  `patch` fixing what is now wrong, and nothing else.
- **propagate** — a technical section changed. Update the product doc that
  depends on it, for its reader: what they can do, what changed for them.
- **condense** — a doc is over its budget. Move whole parts into cards or a
  companion doc, and link to them. Do not squeeze sentences.
- **duplicates** — the same content appears in several places. Keep it in the
  one place it belongs and replace the others with a link.
- **split** — a doc covers several concepts. Propose cards, one concept each.

If the right answer needs a decision only the project can make, return a `note`
instead of guessing.
