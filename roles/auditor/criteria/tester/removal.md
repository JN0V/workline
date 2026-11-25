The tester's curator removed these tests, proven with no
agent: green before and after, no covered line nor killed mutant lost.
Is each test removed truly redundant: does a test kept check the same
behaviour? A regression's test, or a behaviour the measures cannot see
(an order, a message, an edge the lines do not tell), is not redundant.

Claims: the removed test's words checking what no test kept checks
(`source.path`: `diff`), and the files kept or the pull request's words
showing it (`pull request`, a test file).

The note: `true` (each removal is redundant) or `false`.
