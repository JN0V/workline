---
name: five-whys
description: Find the cause of a problem that keeps coming back before proposing a fix, a test, a rule or a decision. Use when the task says the five whys are due, such as a disagreement still open after two exchanges, a review finding still there, a role blocked again, or a defect recorded.
---

# The five whys

A problem that comes back is a symptom. Find what lets it come back, then
propose one action against that cause, not against the symptom.

## How

1. **The problem**: one line, what happened and where, as the material
   shows it. Say what happened, never who is to blame.
2. **Why?** Answer from the material only: the thread, the issue, the
   test, the code, the record. Quote the words that show it, in quotes.
   An answer you cannot quote is a guess: say so, and stop there.
3. **Why that?** Ask again about the answer, not about the problem.
   Stop at the first answer the project can change: a scenario left
   vague, a rule missing, a check that never runs, two documents that
   disagree. Five is a ceiling, not a target; two or three is often
   enough.
4. **People are not causes.** "The builder misread it" is not a cause:
   ask why the reading was possible.
5. **Two causes at one why**: when both are needed, follow the one the
   project can act on; name the other in one line, `Also:`.
6. **The root cause**: one line.
7. **One action**, the one that would have stopped the problem, as a
   proposal your role can make. Its kind, one of:
   - `issue`: a need, a rule or a check to add, for the product owner;
   - `test`: a test rewritten or added;
   - `doc`: a document or a scenario's words fixed;
   - `code`: the code changed;
   - `decision`: a choice only a person can make, the question asked.

## The answer

One `note`, apart from your other proposals, that opens with the line
`Five whys`, then one item a line, in this order:

```yaml
- note: |
    Five whys
    Problem: the test and the code disagree on how mul prints its result.
    1. Why do they disagree? The test checks "6\n", the code prints "6" ("printf '%s' $r").
    2. Why does the test check a newline? The scenario says "Then I get 6", silent on printing.
    3. Why is the scenario silent? The issue's Validation names values, never their format.
    Root cause: the Validation leaves the output's format open, so test and code each chose one.
    Also: the builder did not say which format the callers need.
    Action (doc): add to the scenario "Then it prints 6 and a newline", for the product owner to confirm.
```

- `Problem:`, then the whys numbered from 1, at most 5, then
  `Root cause:`, an optional `Also:`, then `Action (<kind>):`.
- 1,500 characters at most, all lines together: short lines, no prose
  around them.
- The analysis is read by the next person or role in the thread: write
  it so they can check each step against the material.
