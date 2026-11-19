---
# Each finding quotes the author's claim, found again in what they said
# (the commit messages, the merge request), or is dropped (#126).
cites: claim
judge:
  question: >-
    Is this finding about the author's claim true: does the code quoted,
    as it reads, contradict the claim quoted, as the author made it?
    Answer no if the code does what the claim says, if the claim says
    nothing the code could contradict, or if the finding only guesses.
---
**Claims.** What the author says of the change, under "What the author
says" — "no change in behaviour", "covered by the tests", "only a
rename", "handles an empty list" — is testimony: check each claim against
the change. A claim the change contradicts is a finding: `claim` the
author's words, quoted as they read, its cause the line that contradicts
them. A claim the change bears out, or one it cannot be checked against,
is not.
