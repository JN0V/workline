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
