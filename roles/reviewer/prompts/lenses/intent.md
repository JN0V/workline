---
# Asked only of a change saying it closes an issue (`Closes #4`), the
# issue's Need, Example, Steps to reproduce, Validation, Verification and
# Scope given to it, and the files the author lists beyond the Scope; its
# judge is shown the issue and the change (#126). Each scenario of the
# Validation is answered: the code doing it and its test quoted (or why
# no test can check it), or a finding; one answered by neither, on a run
# reading the whole change, is judged as shown by nothing, its judge shown
# the whole change: the files `ignore` leaves, and workline's own last
# commits, which the line reviews the range without, too.
needs: issue
judge:
  question: >-
    Is this finding about the change's purpose true, judged against the
    issue the change closes, shown here? Answer yes when the issue asks,
    in its Need, its Example, its Steps to reproduce, a scenario of its
    Validation or its Verification, what the change shown does not do or
    prove — a scenario a test could check and none does, too; a scenario
    of documentation or of the pipeline no test can check is proven by
    the change itself — or when the change does what the issue does not ask: a file
    changed beyond the issue's Scope, listed by the author or not, whose
    change the issue's Need does not need. The Scope is a hint: a file
    beyond it that the Need needs, or that the author's reason ties to it
    truly, is no finding. A finding that the change removes, renames,
    or changes so that a use that worked breaks, a public name (one
    code outside the project or its users reach: a function, type or
    field exported by a library others import, a command or flag, a
    setting key, a file or wire format) is true when the change shown
    does so and the issue neither asks it nor has a Need that needs it.
    For a cause quoted from the issue, the change shown is the whole
    merge request, its commits reviewed before included, and its last
    commits that are workline's own when shown apart: answer no if any
    of it does it, or if the issue does not ask what the finding says
    lacks. Answer no too if the finding only guesses.
  reads: issue
---
**Intent.** The change says it closes the issue under "What the change
is for": compare the two. A part of its Need the change does not do, a
bug's Steps to reproduce it does not fix, or a Verification it does not
prove — no code doing it, no test showing it — is a finding, its cause the issue's words, quoted, its `path` the issue's
number (`"#4"`). A behaviour the issue leaves open, that the change had to
choose, is a `finding` holding `decision`, never an item of its own. Do not
ask more than the issue does; what commits reviewed before may have done
is not missing. How much the change builds for what the issue asks — a
type, an option, a layer more than it needs — is the simplicity lens's,
not yours: you judge whether it does what is asked, not its size.

**A public name.** Does the change remove, rename, or change so that a
use that worked breaks, a name code outside the project, or its users,
can reach — a function, type or field exported by a library others
import, a command or flag, a setting key, a file or wire format — and is
it announced? A name only this project calls is not one. One the issue
neither asks nor needs is an Important finding, its cause the line
removed or changed, quoted. One the issue asks or needs but nothing
marks as breaking the way the project does (a `!` or a `BREAKING
CHANGE` footer in a conventional commit or in the merge request's title,
which a squash merge commits; its changelog) is a `finding` holding
`decision` — whether to mark it — naming what was read; a project with
no such mark is announced by the issue alone.

**Each scenario.** Answer every scenario of the issue's Validation (its
Given / When / Then), one by one, its name as the issue writes it:

- done and proven: a `scenario`, quoting the code that does it, as it
  reads at the head (earlier commits included), and the test that proves
  it; for a scenario no test can check (a document, the pipeline), the
  `reason` in place of the test;
- not done, or no test proves it: a finding, its cause the scenario's
  words quoted from the issue, one line as it reads there.

Each quote is one run of lines as they read, never pieces joined. A
scenario you answer neither way, or with no test nor reason, is judged
as shown by nothing: silence is never "no finding". Its judge sees the
whole change, files you were not given too. The Example says
what the issue means; read the code against it.

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
