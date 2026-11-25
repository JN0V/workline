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
    asserts nothing of that behaviour, when it would pass without the
    change it is meant to prove, or when it reads the code's text (a
    search in the source) where its own name or scenario asks a
    behaviour, not a text the change writes (a doc, a message).
    Answer yes too when a test file the change adds takes the name of a
    file or type the main code already defines, so that the tests run
    against it in place of the real one. Answer no, naming the test, if a
    test exercises it and would fail without it; no too if the change
    needs no test (a rename, a message), if the test guards a behaviour
    kept as before (it passes on the base by design), or if the finding
    only guesses.
  reads: tests
---
**Tests.** Does a test cover each behaviour the change adds or changes?
Do not run anything: read the tests the change adds or touches, and the
files given. A behaviour changed with no test that would fail without it is
a finding — its cause the changed code, quoted; a test that asserts nothing
of what it names is one too, and so is one that would pass without the
change it is meant to prove, or that reads the code's text (a search in
the source) where its own name or scenario asks a behaviour, not a text
the change writes (a doc, a message). Each is important, never a nit:
the behaviour can break unseen. A change that needs no test (a rename, a
message) is not, nor a test guarding a behaviour kept as before, which
passes on the base by design.

A file the change adds among the tests (a fake, a helper) that has the
name of a file or type the main code already defines is a finding too,
when the project's build would load it in place of the real one while
the tests run (a JVM classpath, a module path): the tests then check the
copy, not the code. Quote the new file's first line.
