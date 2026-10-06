---
# The judge of an important finding of this lens: asked whether a test
# exercises the behaviour, not whether the code fails, and shown the tests
# that touch the cause's file (`reads: tests`, the `tests` setting; #223).
judge:
  question: >-
    Is this finding about the tests true: is the behaviour it names left
    without a test that would fail were it wrong? Judge from the tests
    shown, not from whether the code is right. Answer yes when no test
    shown reaches the code the finding quotes, when the one that does
    asserts nothing of that behaviour, or when it would pass either way.
    Answer no, naming the test, if a test exercises it and would fail
    without it; no too if the change needs no test (a rename, a message)
    or if the finding only guesses.
  reads: tests
---
**Tests.** Does a test cover each behaviour the change adds or changes?
Do not run anything: read the tests the change adds or touches, and the
files given. A behaviour changed with no test that would fail without it is
a finding — its cause the changed code, quoted; a test that asserts nothing
of what it names is one too. A change that needs no test (a rename, a
message) is not.
