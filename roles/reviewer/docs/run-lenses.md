---
sources: [roles/reviewer/role.yaml, roles/reviewer/prompts/instruction.md, roles/reviewer/prompts/policy.md, roles/reviewer/prompts/lenses, roles/reviewer/code]
checked: e81df7b34
verified: agent:claude-code
---
# Reviewer — the lenses

Part of [a run on code](run.md): step 4, what each lens reads and answers.

## 4. The lenses

- The lenses (`lenses/<lens>.md`, a project's own in
  `.workline/roles/reviewer/prompts/lenses/`): correctness, edge cases,
  tests, intent, claims ([#126](https://github.com/JN0V/workline/issues/126)),
  simplicity.
- Comments, asked only when `story-words` flags a comment, beside the
  lens in turn, never in its place.
- On a machine: every one.
- On a merge request: one a push, in turn (`lenses-per-push`).
  - Add a block's lens when the push changes code
    ([more](verdict.md#a-block-stands-until-a-commit-answers-it)).
  - Add the tests lens when it changes the test of a file it found
    untested ([more](lines.md)).
- Every one with `--input lenses=all`, once when it becomes ready
  ([at ready](run.md#every-lens-once-when-it-becomes-ready)).
- None on a draft (`--input draft=true`, [ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)).

### Asking them

- **Together** (`lenses-together`,
  [#147](https://github.com/JN0V/workline/issues/147)): the lenses of a run
  in one call, the change given once; each finding names its lens. One
  naming none is read as the first lens's, said (`finding-lens-unnamed`);
  one naming a lens not asked is dropped, said. `false`: a call each.
- **What they read**:
  - what the author says, as testimony: the messages of the commits not
    reviewed yet, whole, their trailers left, and the merge request's
    title and body (`testimony-lines-max`);
  - each file they change once: whole, the change marked in it, while the
    files fit `code-lines-max`; the others by the change's hunks, each
    widened to the block around it, found by indentation, up to a quarter
    of `code-lines-max` ([#1334](https://github.com/JN0V/workline/issues/1334)). A change
    past one call (the budget, `diff-lines-max`): [in parts](parts.md).
  - each cut past one of these settings is said in the run, with the
    setting that raises it (`input-cut`).
- **Tests, and the scenarios no test proves**: the `## Unproven` list
  of the developer's pull request (its own branch, never a fork's; cut past
  `testimony-lines-max`) is given to the tests lens with the tester's
  reasons, once: its record says so, and no later run asks again. A
  scenario the change makes testable with no test is its Important
  finding, settled the usual way: the tester adds the test or says
  none is needed, the judge weighs
  ([the Draft](../../tester/docs/hand-over.md#what-is-there)). Cases
  `reviewer/tests-lens-given-unproven-scenarios`,
  `tests-lens-unproven-not-asked-again`,
  `tests-lens-unproven-not-from-a-fork`.

### Intent and simplicity

- **Intent** (`needs: issue`): a change saying `Closes #4` (or `fixes`,
  `resolves`, `implements`), in a commit or the merge request, has `#4` read
  from the forge, its Need, Example, Steps to reproduce, Validation,
  Verification and Scope given (`issue-lines-max`), and the files the
  author lists under "Beyond the Scope", with their reasons, from the
  merge request and its commits.
  - The lens says what the issue asks and the change does not do or
    prove, its cause quoted from the issue (`path: "#4"`), and what the
    change does past the Scope that the Need does not need: the Scope is
    a hint, not a fence.
  - A public name the change removes or renames (a function, type,
    field, setting, format others call) is an important finding unless
    the issue asks it and the change marks it as breaking (`!`, a
    `BREAKING CHANGE` footer).
  - Each Validation scenario is answered: the code doing it and its test
    quoted (`scenario`), or why no test can check it (`reason`, said
    `scenario-no-test`), or a finding.
  - On a run reading the whole change in one part, a scenario answered
    neither way, or whose code or test is not found at the head,
    becomes an important finding its judge keeps or drops: silence is
    never "no finding". On a later push or a part, it is said
    (`scenario-unasked`), not judged.
  - Its judge sees what the lenses never do: the files `ignore` leaves,
    and workline's own last commits the line leaves out of the range (a
    developer's fix to the CHANGELOG, `--input own`). A no quoting them
    is found again as those commits leave them.
  - No issue closed: not asked, said in the summary; one that cannot be
    read: not asked, said (`issue-unread`).
- **Simplicity** (`needs: issue`, [ADR-0068](../../../docs/adr/0068-the-line-steps-back-on-its-own-drift.md#next-each-decided-when-reached)): read with
  the same issue as intent; is the change more complicated than its Need
  requires?
  - A finding quotes what the change adds that the Need and the
    Verification do not ask — a public function, an overload, an
    option, a layer — and names the smaller change that closes the issue
    as well; its judge is shown the issue.
  - It judges how much, never whether: what the issue does not ask at
    all, and a part of the Need left undone, stay intent's.
  - Advisory: `severity: nit` in its front matter shows each finding as
    a nit, never judged, never asked of the developer, until its false
    alarms are counted on past pull requests
    ([#1497](https://github.com/JN0V/workline/issues/1497)).
### Claims, the answers, the diff alone

- **Claims** (`cites: claim`): each finding quotes the author's claim ("no
  change in behaviour"), found again in what they said, or is dropped
  (`finding-unfounded`), and its cause the line contradicting it.
- **What they answer**: `finding`s — lens, severity, title, why, its cause
  quoted, its symptom when elsewhere, a fix; the claims lens's, the claim;
  a finding holding `decision`, the question for a person (step 6),
  never an item of its own.
- **The diff alone** (`diff-alone`, off,
  [#126](https://github.com/JN0V/workline/issues/126)): a call of its own,
  given the change only (`git diff -U3`, up to `diff-lines-max`), not the
  files, the messages nor the issue: what the change says by itself.
  - Its findings are quoted, judged and routed as a lens's.
  - `--lenses diff-alone` asks it alone, whatever the setting.
  - Off by measure: it found what the lenses found, nothing more
    ([tried](tried.md)).
- **The floor** (`finder-floor`): each lens is asked to look for a number
  of candidates first, from the change's size. A floor on candidates,
  never on what is judged or shown: a lens answering nothing leaves the
  review clean (`finder-floor-nothing-found-is-clean`).
