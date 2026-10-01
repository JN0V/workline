---
sources: [internal/builtin/documentalist/judge.go, internal/builtin/documentalist/condense.go, internal/builtin/documentalist/dedupe.go, internal/builtin/documentalist/mergecard.go, internal/builtin/documentalist/documentalist.go]
checked: 6013a50
verified: agent:documentalist
---
# Documentalist — the judge

Part of [the documentalist](README.md).

## Judge (`post`, no AI)

A patch is refused, and the agent asked again with the reasons, when it:

- is not a unified diff, or git cannot apply it;
- touches a doc that was not put before the agent;
- quotes lines that are not at the numbers its hunks cite — git apply alone
  would find them elsewhere and apply anyway. A hunk a few lines off (3 at
  most) whose quoted lines the doc holds only there is placed there:
  agents miscount lines in long docs, not their content;
- replaces a version on a line that still says the old one elsewhere — a
  badge's text bumped, its link left (`replaced-in-part`);
- changes lines between `workline:derive` markers;
- makes a doc's body longer by more than a tenth of it (the header does not
  count): a fix may say what the code now does, never pad;
- moves `checked` to another commit than the one given — a fix that leaves
  `checked` alone is kept, and the doc stays suspect, fixed but not vouched
  for (ADR-0012), if it sets `judged` to that commit (`judged-not-set`): the
  doc is not put before an agent again until a source changes (ADR-0013);
- brings a budget, link or duplicate problem the tree did not have, or makes
  one worse — but a size already over budget may grow by what a fix allows,
  and stays reported, for condensing: truth before size.

Asked again as often as the role allows and still refused, a doc's fix is
left out (`left-out`, saying why), and the other docs' fixes are judged again
without it and applied; the doc stays as it was, suspect.

A condense patch is refused when it touches another existing doc; when a line
that leaves the doc is found in no new doc, unchanged but for a heading's level
— moved, not rewritten; a reference updated in place, keeping most of its
words, is fine (`rewritten`); when a MUST or SHOULD is lost; when a new doc is
not linked from the doc (`not-linked`); or when the budget problem it was for
remains (`still-over-budget`).

The judge reads a diff as its lines read, whatever counts its hunk headers
announce, and compares its reading with git's: the engine applies with
`git apply --recount --unidiff-zero`, and checks with it, so what was judged
is what is applied — a last hunk with no context after it is not taken for
the end of the doc.
