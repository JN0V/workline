---
sources: [internal/builtin/documentalist/judge.go, internal/builtin/documentalist/values.go, internal/builtin/documentalist/condense.go, internal/builtin/documentalist/dedupe.go, internal/builtin/documentalist/mergecard.go, internal/builtin/documentalist/documentalist.go]
checked: 6013a50
verified: agent:documentalist
---
# Documentalist — the judge

Part of [the documentalist](README.md).

## Judge (`post`, no AI)

A patch is refused, and the agent asked again with the reasons, when it:

- is not a unified diff, or git cannot apply it — the refusal then says
  which line a hunk quotes wrong, a blank line skipped or one that is not
  there, where git says only the hunk's line (DomoticsCore, ADR-0014 step 4);
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
  for a person;
- moves `checked` while the doc, as patched, still states a line count off
  (`checked-over-count-off`, ADR-0014 step 2): the task gave the real
  count; the fix brings it there, with no claim, or leaves `checked`;
- takes words out of a doc's body with no claim saying why
  (`removal-uncited`, ADR-0014 step 2): the agent removed a true claim it
  could not see backed (workline d43b3f2). A run of changed lines whose
  removed lines say a word its added lines do not, case aside, needs one
  claim beside the patch whose lines reach it (three lines either way):
  `contradicted`, quoting a file under the doc's sources, or `gone`, naming
  a name the removed lines say, in the code when the doc was last edited and
  gone now, as `identifier-gone` finds it. A claim that does not hold — the
  quote not in the file, the file not a source, the name still in the code
  — is `citation-unchecked`. Words only added, lines rewrapped, and a line
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
  ("the", "per", "currently"; never such a fact). A table's total is a count the engine gives too. The task and every
  refusal say it, the engine's count being the evidence: the agent had left
  the counts it was given, believing a count "cannot be quoted as a source"
  (ADR-0014, step 3). A refusal names the places refused only, and the rest
  of the doc's patch that holds, to be sent again unchanged: the agent had
  withdrawn a right fix with the refused one beside it (16c660b). Condense, split, merge-card and
  dedupe tasks are judged by their own rules, which check where the text
  went;
- rests a change on a comment alone (`comment-not-evidence`): its claims'
  quotes are found in the file only inside comments, read by the file's
  type — `//` and `/* */` in Go, C, C++, JavaScript and the like; `#` in
  YAML, shell, Python, TOML, a Makefile; `<!-- -->` in Markdown, HTML,
  XML; strings are read as strings. A stale comment won over the code's
  setting (workline #29). The comment is reported (`comment-disagrees`, at
  the comment's file and line), kept for the run's verdict whatever the
  agent answers when asked again, so a person or the committer fixes it;
- brings a budget, link or duplicate problem the tree did not have, or makes
  one worse — but a size already over budget may grow by what a fix allows,
  and stays reported, for condensing: truth before size.

The answer asked again replaces the refused one whole, so the engine tells
the agent to give again, unchanged, every change not refused. Each answer is
kept in the run folder as it came (`out/agent-answer.txt`; a refused one as
`out/refused-<n>-answer.txt`), and the claims of the accepted one in
`out/claims.yaml`: judged, never applied, kept so that a fix's evidence can
be checked afterwards.

Asked again as often as the role allows and still refused, a doc's fix is
left out (`left-out`, saying why), and the other docs' fixes are judged again
without it and applied; the doc stays as it was, suspect.

Once the fixes pass, two findings are added, never refusing: a `count-off`
a fix brought right says "fixed in this run"; and a version a fix replaces
— a three-part version said fewer times in the lines it adds than in those
it removes, a new one written in its place — still said in the doc fixed,
or in another doc declaring one of its source files that now says the new
version, is `value-left`, with the lines (ADR-0014, step 2). It is not
fixed by the engine: an old version may be said on purpose. History docs
are left out, and so is a version given as a range (`>=1.4.1`, `^1.4.1`),
what a dependant accepts.

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
