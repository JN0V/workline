You get one decision about documentation. `task.md` says which kind:

- **suspect** — a source of this doc changed. Say whether the doc is still true.
  If it is, return a `patch` that only updates `checked` and `verified`. If not, return a
  `patch` fixing what is now wrong, and nothing else.
  When the task says a doc's sources could not all be given whole, `checked`
  stays: the patch fixes what is wrong and sets `judged`.
- **stale** — no source changed, but the doc was last confirmed long ago. Read
  it against its sources as they are now, given in full. Same answer as
  suspect; if the sources given do not let you tell, a `note`.
- **propagate** — a technical section changed. Update the product doc that
  depends on it, for its reader: what they can do, what changed for them.
- **condense** — a doc is over its budget. Move whole parts into cards or a
  companion doc, and link to them. Do not squeeze sentences.
- **duplicates** — the same content appears in several places. Keep it in the
  one place it belongs and replace the others with a link.
- **merge-card** — a card is too short to stand alone. Add its text, as it is,
  to the card it belongs with, delete it, and point its links there.
- **split** — a card too long covers several concepts. Keep the first, move
  each other one, as written, into a card of its own, and link to it; a second
  model then checks each card holds one concept.
- **part** — a doc too large to be judged whole, with a share of its sources.
  For each passage this share speaks to, a `claim`: contradicted, partial or
  supported, quoting both at the lines shown. Nothing else.
- **fix** — what the parts found wrong in a doc judged in parts. Patch that,
  and nothing else; the header stays as it is.
- **sources** — a doc says nothing of the code it describes. Add to its header
  the files or folders whose change could make it wrong, with the `checked`
  given; `sources: []` for a doc describing no code. The body stays as it is.

A patch taking words out of a doc gives, beside it, a `claim` for each place
saying why, as the task shows: a source's words, or a name gone from the code.

If the right answer needs a decision only the project can make, return a `note`
instead of guessing.
