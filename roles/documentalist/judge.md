---
sources: [internal/builtin/documentalist/judge.go, internal/builtin/documentalist/values.go, internal/builtin/documentalist/words.go, internal/builtin/documentalist/condense.go, internal/builtin/documentalist/dedupe.go, internal/builtin/documentalist/mergecard.go, internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/cite.go, internal/builtin/documentalist/comments.go]
checked: d109d62
verified: agent:claude-code
---
# Documentalist — the judge

Part of [the documentalist](README.md).

## Judge (`post`, no AI)

A patch is refused, and the agent asked again with the reasons, when it
does one of the following — but for the places marked *withheld*: there,
the rest of the fix is applied without the place refused (below, "What
holds is applied"):

- is not a unified diff, or git cannot apply it — the refusal then says
  which line a hunk quotes wrong, a blank line skipped or one that is not
  there, where git says only the hunk's line (DomoticsCore, ADR-0014 step 4).
  A hunk misquoting is first mended where its place is beyond doubt, the
  doc's own lines put back as its context: a hunk whose context differs from
  the doc by blank lines alone is placed where its other lines are, when they
  fit the doc at one place only — a blank line let go in the context, never
  between two lines it removes; else each run of the lines it changes is
  placed by the lines it removes alone, found once in the doc, the context
  dropped — a run that only adds, or removes only blank lines, cannot be.
  Sonnet skipped a blank line, and quoted a context line the doc has not,
  and the right fixes those hunks carried were lost (DomoticsCore, ADR-0014
  step 4). *Withheld* when some hunks quote the doc right and the others
  cannot be mended: those are left out; with none quoting right, the agent
  is asked again — a misquote is a slip it mended at the second answer, 3
  of 4 times on DomoticsCore;
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
- moves `checked` on a doc whose sources did not all go whole into the
  task (`checked-unread`, ADR-0014): judged on diffs, or beside a doc of the
  same task, nobody read the rest. `pre` records the docs whose sources it
  gave whole (`in/read-whole.yaml`; none, when the file is missing); for
  any other, the fix sets `judged` and is kept, and the doc stays suspect
  for a person. *Withheld*: the move of `checked`;
- moves `checked` while the doc, as patched, still states a line count off
  (`checked-over-count-off`, ADR-0014 step 2): the task gave the real
  count; the fix brings it there, with no claim, or leaves `checked`.
  *Withheld*: the move of `checked`;
- takes words out of a doc's body with no claim saying why
  (`removal-uncited`, ADR-0014 step 2): the agent removed a true claim it
  could not see backed (workline d43b3f2). A run of changed lines whose
  removed lines say a word its added lines do not, case aside, needs one
  claim beside the patch whose lines reach it (three lines either way):
  `contradicted`, quoting a file under the doc's sources, or `gone`, naming
  a name the removed lines say, in the code when the doc was last edited and
  gone now, as `identifier-gone` finds it. A claim that does not hold — the
  quote not in the file, the file not a source, the name still in the code
  — is `citation-unchecked`. In an answer patching several docs, each claim
  names its doc (`doc:`, the task says so); one that does not is read for the
  only doc whose patch changes the lines it gives, with its quote found under
  that doc's sources or its name in the lines removed there — a right claim
  for HeapTracker's pitfall named no doc beside a second doc's patch, and the
  right fix was withheld (DomoticsCore, step 4). A claim that could stand for
  no doc, or for several, is read for none and reported
  (`claim-unattributed`); the place it was for is refused saying a claim was
  given without `doc:`. Words only added, lines rewrapped, and a line
  count the engine found off, brought to the engine's number, need none —
  the stated number and the word making it rough ("~", "about",
  "approximately"), in prose, a table's cell or a fenced listing; another
  number in its place needs a claim like any word. Where the run replaces
  line for line, the words said around a count it brings to the engine's
  number need none either — the count changed what the line says ("Watch
  the 800-line limit" became "Over the 800-line hard limit" beside 930
  lines, DomoticsCore, step 4) — but a fact of the line's own still does:
  another number, a version, a name (quoted as code, shaped as one, or
  capitalised past a sentence's start), a negation, a quantifier, a
  conjunction or a tense. Nor does a line reworded taking out only glue
  ("the", "per", "currently"; never such a fact). Words are compared with
  their typography plain — `'` and `’`, quotes, a no-break space, "1 000"
  and "1000" alike — and glue and fact words are the doc's language's,
  English or French (an elided "l’" a word of its own; "a", has, a fact
  in French): the project's `language`, else read from the doc's most
  frequent words. A table's total is a count the engine gives too. The task and every
  refusal say it, the engine's count being the evidence: the agent had left
  the counts it was given, believing a count "cannot be quoted as a source"
  (ADR-0014, step 3). *Withheld*, each run of changed lines refused (as
  `citation-unchecked` and `comment-not-evidence`): asked again, the agent
  had withdrawn a right fix with the refused one beside it (16c660b), and
  rarely brought the refused place back right — 1 of 5 re-asks, on
  DomoticsCore and the evaluation (ADR-0014, steps 3 and 4). Condense, split, merge-card and
  dedupe tasks are judged by their own rules, which check where the text
  went;
- rests a change on a comment alone (`comment-not-evidence`): its claims'
  quotes are found in the file only inside comments, read by the file's
  type — `//` and `/* */` in Go, C, C++, JavaScript and the like; `#` in
  YAML, shell, Python, TOML, a Makefile; a Python docstring (a string
  standing as a statement); `--` in SQL, Lua, Haskell; `<!-- -->` in
  Markdown, HTML, XML; `{# #}` and `{% comment %}` in Django and Jinja
  templates, `{{! }}` in Handlebars; a script without an extension by the
  interpreter its `#!` names; strings are read as strings. A stale comment
  won over the code's setting (workline #29). The comment is reported
  (`comment-disagrees`, at the comment's file and line), kept for the
  run's verdict, so a person or the committer fixes it. *Withheld*. A
  quote from a type whose comments the engine does not know is taken as
  code, and said (`comment-style-unknown`, once a type a run), so a
  comment there passing as evidence is not silent;
- brings a budget, link or duplicate problem the tree did not have, or makes
  one worse — but a size already over budget may grow by what a fix allows,
  and stays reported, for condensing: truth before size.

**What holds is applied** (ADR-0014, step 4). A refusal sent the whole task
again — two docs, 35k tokens — for one place the engine had already named.
A place withheld is not asked for again: the doc's fix is applied without
it, the place left as it was; the doc keeps `checked` where it was and
records `judged` (its header put back, `judged` set), so a person reads it
before moving `checked`, and it is not put before an agent again until a
source changes (ADR-0013). Each place withheld is a finding for a person,
with the refusal's rule, the doc and its lines, and why — in the run's
verdict, the merge request's comment or the gardening pull request, as
other findings — and the doc's `suspect` finding says it was fixed in
part. With every place of a doc withheld, the fix is only its record:
`judged` set, the places reported. `post` leaves the narrowed patches in
`out/intentions.yaml`, and the engine applies those (docs/spec/role-contract.md).

**What the finding says** of a doc judged and not vouched for is what the
run did, counted in its body: "fixed in 2 places", or "nothing found
wrong" when the patch only records `judged` — never "fixed" for a header
alone (workline's first nightly, 2026-10-02, said "what the agent found
wrong is fixed" of a doc it found true). A doc judged in parts says, once
the fix is judged, the places it fixed, or that it changed nothing of what
the parts found. The gardening pull request lists each doc it judged with
the same outcome: fixed and in how many places, still true, or nothing
found wrong; `checked` moved or `judged` recorded.

The answer asked again, for any other refusal, replaces the refused one
whole, so the engine tells the agent to give again, unchanged, every change
not refused. Each answer is
kept in the run folder as it came (`out/agent-answer.txt`; a refused one as
`out/refused-<n>-answer.txt`), and the claims of the accepted one in
`out/claims.yaml`: judged, never applied, kept so that a fix's evidence can
be checked afterwards.

Asked again as often as the role allows and still refused, a doc's fix is
left out (`left-out`, saying why), and the other docs' fixes are judged again
without it and applied; the doc stays as it was, suspect. A block the answer
did not cause — docs due, a release held, a check that could not run — is
never asked again: another answer would change none of it.

Once the fixes pass, two findings are added, never refusing: a `count-off`
a fix brought right says "fixed in this run"; and a version a fix replaces
— a version (three parts, or as the project's `versions.pattern` writes
one: a calendar `26.2`) said fewer times in the lines it adds than in those
it removes, a new one written in its place — still said in the doc fixed,
or in another doc declaring one of its source files that now says the new
version, is `value-left`, with the lines (ADR-0014, step 2). The files of
`versions.files` (a `pyproject.toml`, a `package.json`) are every doc's
sources there. It is not fixed by the engine: an old version may be said on
purpose. History docs are left out, and so is a version given as a range
(`>=1.4.1`, `^1.4.1`), what a dependant accepts, or after a history marker
("First available in v1.9.0", "Added in", "since", "New in version",
"Deprecated in", `versionadded::`), which says when something came.

A condense patch is refused when it touches another existing doc; when a line
that leaves the doc is found in no new doc, unchanged but for a heading's level
— moved, not rewritten; a reference updated in place, keeping most of its
words, is fine (`rewritten`); when a MUST or SHOULD is lost; when a new doc is
not linked from the doc (`not-linked`); or when the budget problem it was for
remains (`still-over-budget`). A condense, split, merge-card or dedupe patch
moving an existing doc's `checked`, or creating a doc that carries one, is
refused (`checked-unread`): moving text gives none of its sources, so a new
doc starts without `checked`, listed unchecked for a person.

The judge reads a diff as its lines read, whatever counts its hunk headers
announce, and compares its reading with git's: the engine applies with
`git apply --recount --unidiff-zero`, and checks with it, so what was judged
is what is applied — a last hunk with no context after it is not taken for
the end of the doc.
