# ADR-0014: `checked` is earned by what the engine can show was read

- **Status:** accepted (2026-10-01), after a critical review by seven
  independent reviewers (BMAD analyst, architect, product manager,
  developer; adversarial, verification-gap and edge-case lenses);
  amended 2026-10-01 (who confirms a verdict), 2026-10-02 (where the
  weekly sample is written: ADR-0015)
- **Date:** 2026-10-01
- **Amends:** ADR-0012 (a doc judged against its sources whole), ADR-0013
  (`judged`)

## Context

The documentalist's fixes, reviewed against the code
([documentalist-fixes-reviewed.md](../research/documentalist-fixes-reviewed.md)),
fail in two distinct ways:

- **An unearned `checked`.** Five DomoticsCore docs (six when accepted:
  counted again in step 0, five distinct) had `checked` moved over
  false numbers or dates. Every one had sources far past what a task shows
  whole (20,000 characters; 28 to 165 KB): the agent saw diffs alone, and
  the judge accepted the move (documentalist.go, `sourcesNow` appended only
  when it fits; judge.go, no refusal). The engine let the agent vouch for
  what it never read — a missing check, not only a careless agent.
- **A wrong edit.** On workline, three fixes were wrong in content, none
  moved `checked`: a stale code comment or template header taken as truth
  (d38a0e4, #29), a true claim removed because the sources given did not
  show it (d43b3f2); on DomoticsCore a bug long fixed was rewritten instead
  of removed (92feb08). Quoting would not have stopped #29: it was judged in
  parts, with quote-checked claims, and still followed the comment.

The measures did not see either: no evaluation case grades whether
`checked` moved over what stayed false, results are read at their best, and
the review itself is one agent's reading, not yet confirmed by a person. A
person reading with Claude made the same mistake (9e20ce2 kept "~283 lines"
and moved `checked`).

## Decision

The order matters: each step is measured, with the bar written below, before
the next is built.

### 0. The engine refuses an unearned `checked` (now)

- **`checked` moves only when every source was given whole in the task.**
  Otherwise the fix must set `judged` (ADR-0013); a move is refused
  (`checked-unread`). On every path: merge request, gardening, `workline
  docs`, the release. Fixes keep being committed: what the agent writes is
  right four times in five, and each lands on a merge request a person
  merges (ADR-0011); the doc it fixes stays suspect for a person.
- **The damage already done is undone.** Every `checked` moved since
  adoption — by the bot or by a person — on a doc whose sources could not
  have been read whole at that commit is put back to `judged` at that
  commit, in one reviewed commit per repository: the doc shows as suspect
  again, for a person.
- Shipped in a release; projects take it by moving their pinned version.

### 1. The ground truth and the evaluation, before any new mechanism

- **A person confirms the reviewed verdicts**, each at its own commit, and
  the research file says which are confirmed. Unconfirmed verdicts are not
  cases. *(Amended 2026-10-01: a second, independent check confirms, not the
  person; see Amendment.)*
- **A fixture of real shapes** (`drifted`): a fenced tree listing
  "(524 lines)", a `| File | Lines |` table, "~283 lines", "< 800 lines", a
  sibling doc sharing a version, a stale comment against a config default, a
  line "returns 1.4.0 instead of 1.4.1" about a bug fixed, a true claim
  backed only by a record, and a clean control doc that must end `checked`.
- **New grades**: `finding`, `no-finding`, `judged-is-head`,
  `checked-unchanged`. Every documentalist case grades both: never `checked`
  over a planted falsehood, and `checked` on the clean control.
- **Pass rates, not the best**: at least five runs per model; a held-out set
  (WaterMeter, DomoticsCore after #114, workline's nightly gardening) never
  used while designing. The baseline is measured on the documentalist as it
  is, then after step 0.

### 2. The mechanical checks, no agent

Each with conformance cases, negative ones first, then measured on copies of
DomoticsCore and workline for misses and false alarms.

- **A line count off** (`count-off`): a count next to one file the doc names
  among its sources (by path suffix), in prose, a table row or a listing;
  not next to a limit word (`<`, under, limit, max, target) nor in a doc
  with `sources: []`, a changelog, a decision record or a derived block.
  Exact within one line, or a tenth for `~`. While one stands, `checked`
  cannot move. Tests, fields and other counts: only through a project's
  `derive` command, never guessed.
- **A replaced value left** (`value-left`): a version a fix replaces, still
  in the same doc (a badge's link) or in a doc declaring the same source
  file; three-part versions only; reported, never fixed by the engine;
  history docs excluded.
- **The removal rule**: a fix that removes a body line must cite why — a
  source contradicting it, or a name the engine finds gone from the code
  (as `identifier-gone` does). A removal without it is refused. Condense,
  dedupe, split and merge tasks are exempt: they move text, the judge checks
  where it went.
- **A comment is not evidence**: a fix whose only source for a change is a
  comment line (by file type: `//`, `#`, `/* */`, `<!-- -->`) is refused,
  and the disagreement between comment and code is reported instead.

### 3. A gate before any claims

After step 2, the reviewed commits and the held-out set are replayed, and
what still goes wrong is counted by kind. Only what remains is designed for,
in an ADR of its own. If quote-checked claims are still needed, that ADR
must answer what the review found the first draft did not: the engine, not
the agent, lists the lines stating a fact and sets `checked`; claims are
checked against the doc as patched; the sources are numbered; a quote must
hold the doc's numbers, versions and names; a claim may cite several lines,
or an absence the engine checks; support is checked by a model independent
of the one judged (ADR-0005), since a quote can exist without proving the
sentence; a person can confirm one passage without rereading the doc; and why
claims now pay, where docs/research/documentalist.md found claim extraction
costs more than it gives.

### 4. Acceptance, written before it is measured

Step 0 to 2 are done when, on the held-out set, over five runs:

- no `checked` moved over a passage a person finds false;
- no true claim removed; no comment winning over the code;
- every line count off and every replaced value left that the rules cover
  is reported, with fewer false alarms than real ones;
- the clean control ends `checked` in every run;
- the right fixes (versions, flags) are still made, at least as often as at
  the baseline;
- tokens per judged doc no more than at the baseline plus a tenth.

Then every week a sample of the docs vouched for since — one in ten — is
read against the code by a model of another provider or a person, written to
the measures; one false `checked` sets the documentalist back to step 0.
*(Amended 2026-10-01: who reads, as for step 1; see Amendment.)*

**Measured (2026-10-02): held**, at the third run, on the engine at
d890be5, five runs on Sonnet over the held-out set: nothing false
vouched for, nothing true removed, every count off reported, the right
fixes made, +5.7% tokens a judged doc on the first night. Steps 0 to 2
are accepted; the weekly sample starts with the release
(roles/documentalist/docs/tried.md, 2026-10-02).

## Amendment (2026-10-01)

- **A verdict is confirmed by a second check, not by the person
  rereading commits.** A check independent of the first review reads the
  verdict against the code at that commit (`git show <commit>:<path>`) and
  agrees; the record says who checked and at what independence level
  (ADR-0005), honestly. Only a real disagreement goes to the person, with
  the change, the proof and both readings beside it.
- **Step 4's weekly sample** is read the same way: a model of another
  provider when one is available, else the best level ADR-0005 reaches,
  said in the measures; a disagreement, or a false `checked` found, goes to
  the person.
- **Done for the sixteen reviewed verdicts** (2026-10-01) at ADR-0005
  level 3, not another provider: an agent of the same provider, sharing no
  context with the first review, confirmed all sixteen, five with a
  correction; none needed the person. d38a0e4 was wrong in fact, but
  nothing in its tree contradicted it: a race between branches, not a
  misreading, and not an evaluation case
  (docs/research/documentalist-fixes-reviewed.md).

## Amendment (2026-10-02)

- **Step 4's weekly sample is built** (`workline sample`), and its
  "written to the measures" is decided in ADR-0015: the result goes to one
  tracking issue on the forge, a comment a week, not to a commit. A false
  `checked` — or one whose sources could not be read whole — labels that
  issue `documentalist-step-0` and opens a merge request putting the doc's
  `checked` back, for the person, who confirms by merging.
- **The reader** is the judge the project sets, never the model the
  vouching commit names (`Workline-Model`); on workline, Opus reading what
  Sonnet vouched for, ADR-0005's level 2, no other provider being set up.

## Not decided here

New code no doc describes (today silently ignored unless `documented` is
set) is its own decision, in docs/BACKLOG.md: deriving `documented` from the
declared sources is circular, and it costs an agent call per file added; it
starts without AI, by folder.

## Consequences

- Fewer docs vouched for, at first: most fixed docs end `judged`, listed
  for a person. That is what was checked.
- A doc with sources too large to read whole is never vouched for by an
  agent: it is split, its sources narrowed, or a person reads it.
- The evaluation grows real shapes and costs more runs per measure.
