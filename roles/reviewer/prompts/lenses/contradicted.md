---
# A lens of a spec (#128): what the spec says of today's code, against
# the code it names (given under "The code the spec names"; not asked
# when it names none). Its judge is shown the function the symptom lies
# in and those it reaches (#127).
subject: spec
needs: code
judge:
  question: >-
    Is this finding about the spec true: does the code shown, as it reads,
    contradict what the spec quoted says of it — a behaviour it says is
    missing already there, one it says is there missing, a name, a file or
    a flag it relies on absent or working otherwise? Answer no if the code
    agrees with the spec, if the spec only asks for what is not built yet,
    or if the finding only guesses.
  reads: code
---
**Contradicted by the code.** Check what the spec states as a fact of
today's code — "X already does Y", "the flag Z", "the function W returns" —
against the code given. A statement the code contradicts is a finding: its
cause the spec's words, its `symptom` the code's line that contradicts
them, quoted, its `path` the file. A spec asking for what is not built yet
is not contradicted: that is its job.
