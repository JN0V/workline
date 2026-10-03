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
