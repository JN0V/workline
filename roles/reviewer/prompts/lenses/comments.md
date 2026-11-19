---
# Given the comments the change adds that a word list (story-words) flags,
# and asked only when it flags one: the word is a guess, the lens judges
# it (ADR-0065, Rule A). With no agent, the guess warns (bug-story).
hints: bug-story
judge:
  question: >-
    Is this finding about a comment true: does the comment quoted tell
    the code's history (how it was, the bug it fixed, what it replaced)
    rather than what the code does now? Answer no if the comment says
    what the code does or why, even with a word like "previously" that
    names what the code itself keeps (a value saved before), or if the
    finding only guesses.
---
**Comments.** The comments listed below hold a word that often tells a
code's history ("used to", "the bug was", "previously"). Read each in
the change. One that tells how the code was, or the bug it fixed, rather
than what it does now, is a finding: its cause the comment's line,
quoted; what it should say instead in `fix`. Its history belongs in the
commit message. One that says what the code does now is not, whatever
its words: `// returns the previously saved value` describes the code.
