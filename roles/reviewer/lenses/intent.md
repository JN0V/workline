---
# Asked only of a change saying it closes an issue (`Closes #4`), the
# issue's Need, Verification and Scope given to it, and the files the
# author lists beyond the Scope; its judge is shown the issue and the
# change (#126).
needs: issue
judge:
  question: >-
    Is this finding about the change's purpose true, judged against the
    issue the change closes, shown here? Answer yes when the issue asks,
    in its Need or its Verification, what the change shown does not do or
    prove, or when the change does what the issue does not ask: a file
    changed beyond the issue's Scope, listed by the author or not, whose
    change the issue's Need does not need. The Scope is a hint: a file
    beyond it that the Need needs, or that the author's reason ties to it
    truly, is no finding. The change shown is the whole merge request,
    its commits reviewed before included: answer no if any of it does it,
    if the issue does not ask it, or if the finding only guesses.
  reads: issue
---
**Intent.** The change says it closes the issue under "What the change
is for": compare the two. A part of its Need the change does not do, or a
Verification it does not prove — no code doing it, no test showing it —
is a finding, its cause the issue's words, quoted, its `path` the issue's
number (`"#4"`). A behaviour the issue leaves open, that the change had to
choose, is a `finding` holding `decision`, never an item of its own. Do not
ask more than the issue does; what commits reviewed before may have done
is not missing. How much the change builds for what the issue asks — a
type, an option, a layer more than it needs — is the simplicity lens's,
not yours: you judge whether it does what is asked, not its size.

**Beyond the Scope.** The Scope is the product owner's hint of the files,
not a fence: the developer may change a file beyond it, and lists it under
"Beyond the Scope" with its reason. Judge each file listed there, and any
file the change touches beyond the Scope unlisted: a change the Need needs
(a build file, a test fake, a helper the code calls) is no finding; one the
issue does not ask — an unrelated fix, a refactor on the way, a feature
creeping in — is an Important finding, its cause the changed code, its
title naming the file as off the task: the Scope no longer stops it, so
this is where it is held. A reason that does not hold is said in the
finding.
