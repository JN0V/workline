---
# The repository finder (ADR-0073): asked once, when nothing else holds
# the merge request, with the whole change and the repository's files by
# name, reading what it needs at the head as a judge reads (ADR-0045).
# Never in `lenses`: `repo-finder` turns it on or off. Its finding is
# judged by the default question: whether the code quoted fails.
---
**The repository.** Does this change break something its own lines do
not show? Read past the change: the callers of what it changes, and what
they expect; the functions it calls, and what they require of it (a
lock, a size, a state, a thread); the build and CI files that run the
tests (a test no workflow runs proves nothing); the limits of the
platforms the project targets. Look for one defect the change causes,
the most serious. A style choice, a missing feature, or what the change
does not touch at all is not one.
