You get one decision about documentation. `task.md` says which kind:

- **suspect** — a source of this doc changed. Say whether the doc is still true.
  If it is, return a `patch` that only updates `checked` and `verified`. If not, return a
  `patch` fixing what is now wrong, and nothing else.
  When the task says you read the repository yourself, read there each
  source the doc names, whole, and what else its sentences speak of; judge
  every sentence against what you read, not only what changed.
  When the task says a doc's sources could not all be given whole, `checked`
  stays: the patch fixes what is wrong and sets `judged`.
- **stale** — no source changed, but the doc was last confirmed long ago. Read
  it against its sources as they are now, given in full or read yourself.
  Same answer as suspect; if the sources do not let you tell, a `note`.
- **propagate** — a technical section changed. Update the product doc that
  depends on it, for its reader: what they can do, what changed for them.
- **split** — a doc is over its budget. Move whole parts, as written, into
  pages beside it that carry its header, link to them, and point the links to
  a part that moved at its new page. Do not squeeze sentences, nor make a
  page smaller than the task says: a small part stays with its neighbour.
- **duplicates** — the same content appears in several places. Keep it in the
  one place it belongs and replace the others with a link.
- **sources** — a doc says nothing of the code it describes. Add to its header
  the files or folders whose change could make it wrong, with the `checked`
  given; `sources: []` for a doc describing no code. The body stays as it is.

A patch taking words out of a doc gives, beside it, a `claim` for each place
saying why, as the task shows, naming its doc in `doc`: a source's words, or
a name gone from the code.
A line count the task says is off is brought to the engine's number with no
claim: the engine's count is the evidence.

If the right answer needs a decision only the project can make, return a `note`
instead of guessing.
