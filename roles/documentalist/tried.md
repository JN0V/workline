# Documentalist — tried for real

What each try on a real repository or with a real agent showed, newest last.
The role's README says what the role does.

On 2026-09-24, with Claude (sonnet, then opus when asked again), on copies of
the `documented` fixture:

- tokens moved from one hour to two, three runs: the agent's answer was right
  each time, in one call; the first run lost the fix to the defect below, the
  two after it applied it;
- a function added, the doc still true: the first answer was refused for
  growing the doc's body; the second, one tier up, moved `checked` and nothing
  else, and opened an issue for a comment the new code contradicts;
- two defects found this way, now guarded: a hunk header counting fewer lines
  than it holds made git drop the fix itself (hence `--recount`), and a commit
  like `11180e1` was read by YAML as a number, so the doc passed for unchecked.

On workline's own repository, without AI: no false positive; one real finding
(docs/spec/role-contract.md is over its line budget).

On 2026-09-24, before a push of workline itself (a copy, pushed to a local
remote): the line found three docs whose code had changed since they were
checked — two of them real, left behind by the day's own commits. Claude judged
all three still true and said why, line by line. Its first patches set
`checked` to the commit that had changed each source, not to the one the task
gave: the task now says "your patch sets `checked: …`", and a refusal repeats
the commit. The next run was right at the first attempt; the push stopped for
review, the docs were committed, and the push went through.

On 2026-09-24, gardening workline itself (a copy): the documentalist chose
docs/spec/role-contract.md, 330 lines for a budget of 200. Three defects came
out before it worked, each now guarded: the agent's time was capped at three
minutes (a role now sets `model.timeout`); an answer holding a code block of
its own was cut at that block (the answer is now read whole first); and a new
file after another in one diff, without `diff --git` lines, was read by
`git apply --recount` as lines of the first (those lines are now added). Then
Claude moved 142 lines, unchanged, into role-outcome.md and role-adapting.md,
each tracking the contract's sources, linked both ways — at the second
attempt, the first citing a wrong line. That split is the one in this
repository.

On 2026-09-27, on workline's own repository:

- links to other sites, with lychee: 75 checked, none broken; a 404 added on a
  copy was reported at its line; behind a proxy that does not answer, each doc
  got a count of links not reached, and no link was called broken;
- a superseded decision, on a copy: the one doc citing it without its
  successor was reported, not the two naming both;
- stale docs, on the `documented` fixture with Claude (Sonnet): a doc edited
  into a claim its source does not back was confirmed once in four runs; the
  policy now says moving `checked` vouches for every sentence kept, and a case
  guards it (tests/evaluation);
- a spec the code disagrees with (`truth.doc`): the doc kept as it was, and an
  issue quoting both sides, in three runs out of three;
- the checklist, on a throwaway pull request of this repository on GitHub
  (#2): created, then edited in place on the next run, then emptied once the
  doc's `checked` moved; one comment throughout.
- a repeated passage, on the `untidy` fixture with Claude (Sonnet): kept in
  the sessions doc, the FAQ linking to its heading, at the first attempt, and
  in three evaluation runs out of three.
- a card too short, on the `documented` fixture with two cards, with Claude
  (Sonnet): merged into the other card, deleted, its link pointed there, at
  the first attempt — under a second top-level title, which the task now
  forbids; then three evaluation runs out of three.
- a merge request per gardening task, with Claude, a local bare remote and
  the simulated forge: the first attempt was refused by the machine's own
  commit hook, the fixture's identity not being allowed, and left the patch
  staged — it is now left unstaged, as the patches wrote it; with an allowed
  identity, the branch was pushed with one commit, the merge request opened,
  the tree back on main and clean, and a second run updated the same merge
  request.
- the same on GitHub (2026-09-28), from a clone of this repository, without
  AI, a derived block made stale: pull request #3 opened from
  `workline/documentalist/derived`, one commit, the README's count; a second
  run updated it; with `max-open-merge-requests: 1`, the next run paused —
  but still regenerated the block, which could have opened one merge request
  too many: a paused run now proposes no patch at all.
- splitting a card, on the `documented` fixture, Opus splitting and Sonnet
  judging: right at the first attempt by hand; measured, the judge said no
  twice in two runs, taking the sentence on how long a token lasts, in the
  sessions card, for a second concept, and Opus's second try was then refused
  for rewording a line. The question now says a sentence explaining the
  concept by what it depends on stays within it: three runs out of three
  since, the judge naming why each card holds one concept.
- condensing on Sonnet, four more runs: three right, one moving nothing;
  with the two of 2026-09-25, four in six, writing more tokens than Opus,
  which is right ten times in ten: condensing stays on `frontier`.
- a doc fixed on a merge request, on GitHub (2026-09-28): a throwaway pull
  request (#4) added a conformance case, so the README's count was stale; run
  as the CI's apply job would, on the detached merge checkout, the
  documentalist committed the count to the pull request's own branch, on top
  of the author's commit, and left the tree clean.
- propagating, on the `documented` fixture with the technical doc saying two
  hours: pending on a merge request, blocking the release without an agent,
  and at the release, with Claude (Sonnet), the product doc brought to two
  hours and checked again, at the first attempt and in two evaluation runs
  out of two.
- adopting a repository (`workline init`, 2026-09-28), with Claude, on a copy
  of this repository (18 docs saying nothing of their sources) and of a small
  firmware project (12 docs, none declaring any): each doc got its sources, or
  `sources: []` for a decision, a changelog, a research card, in three or four
  rounds of 22 to 38k tokens in. Sonnet's answer was refused in six rounds of
  seven, then Opus's taken: folders written `roles/`, with the slash, taken
  for missing; a blank line between the header and the title, taken for a
  change of the doc (three rounds); a diff git could not apply (two). The
  first two are now taken, and guarded. And a second note of one answer
  overwrote the first, and no note reached the person — yet the notes had
  found a doc behind the changelog, and one possibly replaced by another:
  every note is now in the result, and shown. Sonnet alone, on this
  repository, after the slash was taken: right at each first attempt.
- adopting DomoticsCore (2026-09-28), 64 docs: the run failed before any
  call, the task over the role's budget — its changelog, 134 kB, was put
  before the agent whole. A doc now shows its first lines only, enough to
  tell what it describes.
  Run again: 14 docs adopted in five rounds, then the rounds ran out; three
  answers were refused for a hunk numbered from line 0 (`@@ -0,0 +1,4 @@`,
  the doc's first line as context), which git reads from line 1 and the
  judge took for a line that does not exist: now read as git does.
  After two more runs, 63 of 64: the last, cut to its first lines, was left
  by the agent, which said, rightly, that the sections cut were unknown to
  it. A doc cut now lists the headings of what is not shown.
- catching DomoticsCore up (2026-09-29), its 53 suspect docs judged in one
  go: the first call failed before a token was spent, its first doc, with
  what changed in its folder sources, over the role's budget. A doc too large
  for a task alone is now left for a person, saying so, and the others are
  judged; cutting it would have its `checked` vouch for lines never read.
  Stopped after four runs, about a million tokens in, 46 docs still to
  judge. Their sources named whole folders; narrowed to the files each doc
  names (2663 files behind them down to 911), they cost more, not less: the
  diff shown was capped at 80 lines a source. It is now 240 lines a doc:
  estimated 517k tokens to 338k for the 42 left, and 12 docs too large for a
  task down to 2.
  Run again, on Sonnet 5.5 once the alias moved: five calls on one doc, the
  agent rightly refusing to vouch for it on diffs cut short, and the engine
  taking its note for progress. A round leaving only notes now ends the run;
  a doc whose changes do not fit is judged against its sources as they are
  now, like a stale one — 26 of the 42 left had more changes than a task
  shows, up to 17,099 lines of diff.
