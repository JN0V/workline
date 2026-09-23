# Documentalist

What runs, for humans. The AI never reads this file.

## Prepare (`pre`, no AI)

1. **Suspects.** For each doc, list commits since its `checked` commit that
   touched its `sources`. Follow the chain: a suspect technical section makes
   the product docs depending on it suspect too.
2. **Derived blocks.** Regenerate everything between `assembly:derive` markers;
   a difference is a finding.
3. **Hygiene.** Broken links (lychee), style (vale), identifiers named in a doc
   that no longer exist in the code, docs citing a superseded ADR.
4. **Budgets.** Lines per doc, words per section, card size (min and max),
   lines per folder, root agent file.
5. **Duplicates.** Repeated passages across docs (jscpd, then near-duplicates).
6. **Freshness.** Docs whose `checked` is older than the limit.

**Cascades are cut.** A change of code can make a technical doc suspect, which
makes a product doc suspect, and so on. Only the edges marked `now` are handled
inside the change. Every other suspect goes on a pending list — one tracking
issue, updated in place — with the moment it is due: the release gate refuses
to release while product docs due `at release` are still pending, and the
release manager hands them to the documentalist in one batch. Nothing is
forgotten; nothing drags a small fix into a rewrite of the user guide.

**Code never rewrites the authority.** When the code disagrees with a doc marked
as the truth (a spec, an ADR, the architecture), the doc is not updated to
match: the documentalist opens an `issue` for the architect, because the code
may be the one that is wrong.

Each finding is blocking, advisory, or *unknown* — and unknown fails, it is
never read as a pass. Judgement calls become `task.md` entries, at most
`ai-max-calls` per run, and none when `max-open-merge-requests` are waiting.

## Judge (`post`, no AI)

- Every quote in the agent's answer exists at the lines it cites.
- The patch touches only docs, not derived blocks, and does not grow the docs
  (except a real `propagate`).
- Budgets, links and duplicates are checked again on the patched tree.

## Without AI

All checks still run and still block. Each judgement call becomes a checklist
item in the merge-request comment; a person clears it by updating `checked`.
