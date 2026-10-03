# Product owner — tried for real

Each try with a real agent or on a real backlog, newest last.

## 2026-10-03 — planted issues, local forge, Sonnet

A copy of a small project: `WriteRows` once stopped one row short, a
commit fixed it. Four issues planted on the local forge: #1 the bug, fixed
since (obsolete); #2 the same symptom in a user's words, naming no code
(duplicate of #1); #3 the header row never written, on the same function
(a look-alike, still true); #4 a JSON export asked for (a need).

- **First run**: no issue had a state; each got one, the agent not asked.
- **Second run** (2,994 tokens in, 261 out): #2 closed as a duplicate of #1,
  #1's words quoted; #3 and #4 left open, the look-alike told apart.
  The agent said it had not been given the code: `pre` gave none, so #1
  could not be found obsolete. **Fixed**: `pre` now gives each file an
  issue names, whole, with its last commits (case
  `product-owner/code-named-is-given`).
- **Third run** (3,429 in, 682 out), #2 reopened: #1 proposed as obsolete,
  the fixed loop quoted and the fixing commit named, left to a person as
  `close-obsolete` starts at `propose`; #3 and #4 left open. #2 not closed
  this time: "it names no code", said in a note. One run each: whether a
  duplicate naming no code is closed varies; the evaluation's five runs
  will say how often.
- Seen, not fixed: the agent wrote "changed it after this was reported",
  though no issue's date is given; the report's record reads `{}` when
  empty.

## 2026-10-03 — the evaluation, five runs, Sonnet

`product-owner/keeps-a-planted-backlog` (tests/evaluation): five issues
planted on the local forge of a project whose last-row bug a commit fixed —
#1 that bug (obsolete), #2 no header row and #3 its duplicate in other
words, #4 empty rows written as blank lines (still true, on the loop the fix
touched: the trap), #5 a need. Five runs, 11 of 11 each: #1 proposed as
obsolete and left open, #3 closed as a duplicate of #2, #2, #4 and #5 left
alone and not proposed. About 3,800 tokens in and 360 to 570 out a run, one
call, 5 to 6 seconds.

What it does not measure yet: a duplicate by likeness only (proposed, not
closed, by ADR-0018 — the engine does not tell it apart yet); a real
backlog, its issues long and many; a wrong closing in the wild.

## 2026-10-03 — DomoticsCore's roadmap, on a copy, Sonnet

A clone of DomoticsCore (26b2ee0), no remote, the local forge. Its
CODE-ROADMAP.md (7,580 lines, 192 entries) had 38 entries whose heading
says nothing of done; a one-shot script, without AI, opened each as an
issue keeping its id (`BUG-4 — NTP: …`), its sources the files it names,
else the code its backticked names are in, else its component's header.
Ten runs of four issues, about 45 to 62k tokens in and 230 to 1,300 out
each (Claude Code's own prompt is most of it), 7 to 12 seconds.

- **Eight issues proposed as obsolete**, the code quoted: BUG-1, BUG-4,
  BUG-5, BUG-7, BUG-14, BUG-15, BUG-20, BUG-24. Each checked by hand
  against the code: all eight right; the lots table agrees for NTP and
  Storage. None closed: `close-obsolete` starts at `propose`.
- **None proposed wrongly.** Where it could not see enough, it said so in
  a note and closed nothing: BUG-6 (the member's type cut from what it was
  given), BUG-9 (the config path cut), MEM-1 (partly done, re-scope it),
  BUG-48 (a function's body not given), DOC-1 (two parts, one done).
- **No `sources` proposed**, though twice it said the code given was not
  the issue's (SEC-3's route, BUG-56's browser script): it could not tell
  which file was. Only files it is given can be named.
- **Defects found, fixed** (cases in tests/conformance/cases/product-owner):
  - the report was rewritten each run with that run's proposals only: seven
    of the eight were lost. A proposal now stays until a person settles it
    (`proposals-kept-until-settled`);
  - an answer that did not read (prose around YAML) was not asked for
    again — the role had no `promote-after` — and the issues it held were
    still recorded as read. It is asked for again once now, and an issue
    is recorded as read only when the answer reads
    (`unreadable-answer-reads-nothing`, `if-answered` in the role contract);
  - the first try's prompt, 2,000 lines of code, was over the role's budget
    and refused before any call: `code-lines-max` stays at 1,500.
- Not rerun after the fixes: the 570k tokens again to see the report hold
  eight lines, which the cases now prove.

## 2026-10-03 — the import as a command, milestones, Sonnet

A fresh clone of DomoticsCore, no remote. `workline issues import
docs/CODE-ROADMAP.md --apply --forge local`: 188 entries, 150 marked done,
38 issues opened, the same 38 as the script's; its sources, by path or by
backticked name, tests last, left none for 7 entries. Two runs of four
issues (57k and 48k tokens in, 1.5k and 0.7k out):

- **Milestones**, the last release v2.13.0 given: SEC-3 (HIGH) put in
  `v2.14.0`, MEM-7 and MEM-8 (LOW) in `v2.15.0`, each with why. SEC-3's
  code was not given — the import found no file for it — and the lots
  table says it was merged long ago: put in a milestone on its title
  alone. **Fixed** in the instruction: only an issue the code given shows
  still true; not rerun.
- **`sources` used for the first time**: BUG-5, which the import tied to
  no file, named `NTP.h`, a line of it quoted; it is read again with it.
- BUG-1 and BUG-4 proposed as obsolete again, as in the first try.

## 2026-10-03 — workline's own BACKLOG.md imported by the agent, Sonnet

The parser of the first import read DomoticsCore's headings only, and found
0 entries in workline's docs/BACKLOG.md (numbered and bulleted items, no
ids, "Done … Left: …" inside the text). Replaced: the agent reads the file
a share at a time and proposes `open`, its text quoted; the engine checks
the quote and opens once.

On a copy of workline, local forge: one share (238 lines), one call, 14.6k
tokens in, 6.7k out. 30 issues opened; the eight items of the last section,
each "Done", left; an item done in part titled by what is left ("Reusable
action for install without Go in CI"). **Defect found, fixed**: an issue's
body began with the last two lines of the item before — the quote was
located from the first line holding its first word (`-`); it now starts at
the last line that still holds it all (`TestLocate`).
