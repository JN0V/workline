---
# Asked only of a change saying it closes an issue (`Closes #4`), the
# issue's Need, Verification and Scope given to it; its judge is shown
# the issue and the change (#126).
needs: issue
judge:
  question: >-
    Is this finding about the change's purpose true, judged against the
    issue the change closes, shown here? Answer yes when the issue asks,
    in its Need or its Verification, what the change shown does not do or
    prove, or when the change does what the issue's Scope leaves out.
    Answer no if the change does it, if the issue does not ask it, or if
    the finding only guesses.
  reads: issue
---
**Intent.** The change says it closes the issue under "What the change
is for": compare the two. A part of its Need the change does not do, or a
Verification it does not prove — no code doing it, no test showing it —
is a finding, its cause the issue's words, quoted, its `path` the issue's
number (`"#4"`). Something the change does that the issue's Scope leaves
out is one too, its cause the changed code. Do not ask more than the
issue does; what commits reviewed before may have done is not missing.
