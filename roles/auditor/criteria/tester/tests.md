The tester wrote these tests from the issue's scenarios, before any
code, and proved them red on the base. Does each test check its named
scenario's Then, as the person wrote it? Would it fail on the base
because that Then does not hold yet — not for a reason of its own (a
typo, a missing setup, a wrong path)? A guard, the test of a scenario
kept as today the Draft lists under Guards, is green on the base on
purpose: does it check that scenario's Then as it holds today?

Claims: the test's words that check something else, or fail for its own
reason (`source.path`: a test file), and the scenario's words they miss
(`scenarios`), or the base code showing it (a file `(base)`).

The note: `true` (each test checks its scenario, red for the right
reason, a guard green as its scenario holds today) or `false`.
