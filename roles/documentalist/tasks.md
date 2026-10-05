---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/parts.go, internal/builtin/documentalist/freshness.go, internal/builtin/documentalist/dedupe.go, internal/builtin/documentalist/condense.go, internal/builtin/documentalist/mergecard.go, internal/builtin/documentalist/counts.go]
checked: a43fff5
verified: agent:claude-code
---
# Documentalist — the agent's tasks

What the agent is asked: suspect docs, judged whole or in parts, and, when
gardening, stale docs, duplicates, condensing, merging and splitting cards.
Part of [the documentalist](README.md).

## Suspect docs

Suspect docs become the agent's task, each with its lines numbered, what
changed in its sources — 240 lines of diff a doc, whatever the number of
sources — and beside it its sources as they are now, in full, when they fit,
since vouching for every sentence needs them (ADR-0012); a doc far behind,
whose changes do not fit, is judged against its sources as they are now
alone, or in parts when they do not fit either — and the commits its
`checked` must name: at most
`ai-max-calls` docs per call, and no more than fits the role's context budget.
Docs left out are taken in the run's next round, once the first ones are
applied (docs/spec/role-contract.md, "Again"); past the last round, they stay
suspect for a person or a later run. `ai-max-tokens` caps what a run spends, all
calls together: past it, the agent is asked nothing more and the rest waits.

`checked` is earned by what the task gave (ADR-0014). A doc is told it may
move `checked` only when every one of its sources went whole into the task,
as it is now — every text file under a source path, the section of a source
doc, up to the project's `whole-chars` together (20,000 characters by
default; the task grows by as much as it is raised). A doc judged on
diffs alone, or beside a doc of the same task it follows, is told instead
that `checked` cannot move: its fix leaves it and sets `judged`, the judge
refusing a move (`checked-unread`), and the doc stays suspect for a person.
The same holds for docs due at the release and for stale docs. A doc stating
a line count off is told each one, with the real count, counted by the
engine: `checked` cannot move while one stands (`checked-over-count-off`),
so the fix brings it to that count or leaves `checked`. That change needs
no claim, the task says: the engine's count is the evidence, and a count is
not words a file holds, to be quoted.

Every task judging docs — suspect, stale, due at the release, the fix after
the parts — says that every word a patch takes out of a body is cited
(ADR-0014, step 2): beside the patch, a `claim` for each place, with the
doc's lines and either a source's words quoted (`status: contradicted`,
`source: {doc's source file, quote}`) or a name the code no longer has
(`status: gone`, `name`); `doc` names the doc when the answer patches
several. Sources are not numbered in the task, so a claim quotes, and the
engine finds the quote. A comment is not evidence: with nothing else to
cite, the words stay, and a `note` says what could not be confirmed.

## Judged in parts

**Judged in parts** (ADR-0009), when the project sets `judge-in-parts: true`
(off by default, until the evaluation has measured it). A suspect or stale
doc whose sources, as they are now, do not fit a task is judged in parts
instead of going to a person: each part holds the doc whole, then a share of
its source files packed to fill the task; a file larger than a part is cut
into consecutive pieces, each read, so no line goes unread. Sources that are
docs are left out: following a doc is propagation, not evidence. Each part
answers with `claim`s only — a range of the doc's lines `contradicted`,
`partial` or `supported`, quoting the doc and the source — and the engine
asks each in a context of its own (docs/spec/role-contract.md, "In parts").
Then, with no AI, a claim whose quotes are not at the lines it cites is
dropped (`claims-dropped`), for a person, who reads the doc whole: the claim
dropped may have been the one saying what is wrong. The doc is recorded as
judged all the same — asked again each night, its parts cost as much to the
same end (ADR-0014, step 4); contradicted and supported by no other part is
wrong, supported by another a conflict, partial to be read together —
unless another part supports the same lines, which settles them; a line
naming a name from the code that no part speaks of is `uncovered`, for a
person. One last call (`fix`) gets the doc and only the source lines the
parts cited, and patches what is wrong; the judge refuses a fix that moves
`checked` (`checked-moved-in-parts`). The header records the commit it was
judged in parts at, `judged-in-parts`, found wrong or not, and the doc stays
suspect, saying so: nobody read it whole. It is not put before an agent
again until one of its sources changes after that commit; a person clears
it by reading it whole and moving `checked`.

Not judged in parts, and left to a person, saying why: a doc leaving less
than a quarter of a task for its sources (condense or split it first); one
with sources in another repository; one needing more than `parts-max` parts
(8), reported `sources-too-wide` with how many it would take — narrow its
sources or split the doc; one whose parts did not all answer (the engine says
`part-unanswered`: nothing they found is kept, it is judged again later).
At most `parts-max-per-run` parts (16) are asked in a run; a doc past it
waits for the next round. A run asking parts asks nothing else.

## When gardening

**Stale docs, when gardening.** On `schedule`, with no suspect doc to judge,
the stale docs become the task, each with its sources as they are now, in
full: the agent confirms it, fixes it, or says with a `note` that the sources
do not tell. A doc whose sources are too large to be given in full stays for
a person: confirming it unread would only fake its freshness.

**Duplicates, when gardening.** On `schedule`, with no suspect or stale doc,
the first passage written in two docs becomes the task: keep it in the doc it
belongs to, and in the other a sentence and a link to it. It comes before
condensing: two copies drift apart. The judge checks that what leaves a doc
is found in the other, that it links to the one keeping it, that no MUST or
SHOULD is lost, and that the passage is no longer repeated.

**Condensing, when gardening.** On `schedule`, with no suspect or stale doc
and no repeated passage,
the doc most over its budget (a doc too long first, then an agent's entry
point, a section) becomes the task: bring it within budget by moving
whole parts into a new doc, and linking to it. One doc a run. A history doc —
a changelog, a decision record, a tried or research record — is never the
task: moving its parts away rewrites the record; its budget stays reported,
for a person. A merge request or a push never turns into a rewrite of the
docs.

**Merging cards, when gardening.** Last, with nothing else to do, a card too
short to stand alone goes into the card it belongs with, chosen by the agent
among the others, those of its folder first: its text added as it is, under a
heading, the card deleted, and every link to it pointed there. The judge
checks that the text is found in one card, which loses nothing, that the docs
linking to it change those links only, and that no link is left dangling.

**Splitting a card, when gardening.** Next, a card too long holds more than
one concept: it keeps its first, and each other one moves, as written, into a
card of its own that it links to. The judge checks what condensing checks, and
that each new doc is a card; then a judge model — not the one that split it —
is asked whether each card holds one concept, the one its title names. A no
goes back to the agent with its reason.

## One merge request per task

Run with `--open-merge-request` (the CI
templates' scheduled jobs), a gardening task's patch goes on a branch of its
own, `workline/documentalist/<task>`, with a merge request titled after the
task; running the task again updates it (ADR-0006). While
`max-open-merge-requests` of them wait for review, gardening proposes nothing
(`gardening-paused`), and what it finds is still reported. While a task's
own merge request waits, the docs it would judge wait too (ADR-0013): a doc
judged in parts, whose task is `fix`, is then asked before the docs judged
whole. A task proposed by an earlier round of the same run counts as
waiting: a CI night judged with `--no-apply` that proposes the docs judged
whole goes round again, and the docs put off to parts are judged the same
night, on the `fix` merge request (ADR-0013, amended) — workline's first
nightly had put thirteen off "to a later round" a CI run never had. The
merge request lists each doc it judged with what became of it: fixed and in
how many places, still true, or nothing found wrong; `checked` moved or
`judged` recorded.
