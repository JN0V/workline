---
sources: [cmd/workline, internal/engine/engine.go, internal/hooks, internal/line/states.go]
checked: 523d0359
verified: agent:claude-code
judged: e343388
---
# The commands that do the work

The tester's, the developer's, the reviewer's and the backlog's commands,
with every option and case; part of [the commands](commands.md).

## workline test

`workline test check --base <ref>` is the tester's red-first check, with
no agent ([tester](../roles/tester/README.md#check-a-change)); it returns 0 or 1:

- **The new tests**: those the head's JUnit report holds and the
  base's does not, the test command run in a checkout apart for each.
- **Red on the base**: each fails there with the change's test files
  laid on it; one that passes is refused.
- **Green on the head**, when the change holds code too.
- **`--base ""`** is misuse, 64: a CI variable left unset.

`workline test write --issue <n>` has the tester write a ready issue's
tests, one a scenario, through the agent
([tester](../roles/tester/README.md#write-an-issues-tests)):

- **`--ai`**: none, `claude`, `cmd:<any command>`; with none, nothing is
  written, and it says so.
- **Proven first**: the check above runs on them, the base with the tests
  alone; red, they are committed on `workline/tester/issue-<n>`, opened
  as a Draft `Refs #<n>`. Not red, nothing is opened.
- **`--forge`**: where the issue is read and the Draft opened; the
  project's `forge` setting by default.

`workline test answer --merge-request <n>` has the tester answer the
first objection waiting on its Draft
([tester](../roles/tester/README.md#frozen-tests-and-objections)):

- **Kept** or **rewritten**, in the objection's thread; a rewrite is its
  own commit on the branch, red on the base, pushed without force.
- **Checked out** at the branch's tip; another branch is never written.
- **`--ai`**: none says so and answers nothing; the need itself, an
  unreadable form and a person's turn need no agent.

`workline test measure --history <file>` is the curator's measures of
the corpus on `HEAD`, no agent
([curator](../roles/tester/docs/curator.md)); 0, or 1 when it did not
run:

- **What**: each test's time and result, the slowest, red and flaky
  ones, coverage where the command writes it.
- **`--history`**: read before, written after; missing, the first
  measure. **`--summary <file>`**: the measures, for a CI job's page.

`workline test curate --history <file>` has the curator remove a few
redundant tests of one folder ([removals](../roles/tester/docs/removals.md)):

- **The agent proposes**, a reason each; `--ai` none removes nothing.
- **The proof, no agent**: the folder's tests before and after, green,
  no covered line and no killed mutant lost; else nothing is opened.
- **Proven**: a pull request on `workline/tester/curate-<folder>`, not
  a Draft, on `--forge`.

## workline develop

`workline develop` has the developer take a ready issue
([the pick](../roles/developer/docs/pick.md)) and, with an agent, write
its code within [its bounds](../roles/developer/docs/bounds.md):

- **The first ready issue** in the backlog's order, assigned to nobody,
  not taken, no open merge request closing it, its Scope clear of the
  issues the developer holds; `--issue <n>` looks at that one alone.
- **`--ai`**: none, `claude`, `cmd:<any command>`; with none, the issue
  is named, nothing claimed. With one, the claim label and a comment
  holding the record go on the issue; the agent's patches within the
  bounds are proven on the project's tests and pushed on
  `workline/developer/<n>`, on top of the tester's Draft when one is
  open ([the pull request](../roles/developer/docs/pull-request.md)):
  ready when green with no cap reached, else a Draft
  ([the proof](../roles/developer/docs/proof.md)); the others are refused.
- **`--no-apply`**: judged, nothing written; `workline apply <run>` claims.
- **None to take**: it says so, and passes.
- **`--merge-request <n>`**: a review round on its own pull request, not
  an issue: the review's findings fixed, proven and pushed on its
  branch ([the review loop](../roles/developer/docs/review-loop.md)).

## workline route reconcile

The line of states' safety net
([the reconciler](spec/routing.md#the-reconciler)): every open issue's
step due that no event started, run.

- **Due**: the role its state wakes is routed on `issue`, and the line's
  record on it does not hold that state, body and labelling yet.
- **`--ai`**, `--open-merge-request`, `--forge`: as for each step.
- **At most `reconcile-max`** steps a run (3); `--input reconcile=list`
  names them and starts none.
- **Run twice in a row**, the second does nothing more.

## workline review

The reviewer reviews this branch before it is pushed
([the reviewer](../roles/reviewer/README.md),
[ADR-0020](adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)).

- **The range**: `<base>..HEAD` (`--base`, default the role's `base`
  setting, `main`), every lens (`--lenses` names some).
- **What it does**: the rules on the comments the change adds, then each
  lens, each finding's quote found again, the change's findings told from
  those outside it, each important one judged.
- **What it prints**: the findings, the tokens each call used against the
  reviewer's `ai-max-tokens`, and the path of the run's `out/review.json`,
  for the author's agent to fix them before pushing.
- **Options**: `--json`, `--sarif`, `--code-quality`, `--ai`, `--forge`
  (where an issue for what lies outside the change goes).
- **A spec instead**: `--spec <file>` or `--issue <n>`
  ([#128](https://github.com/JN0V/workline/issues/128)), on the
  reviewer's `spec` event: the file as it reads in the working tree, or the
  issue's body from the forge, through the spec lenses (ambiguous,
  unverifiable, out of scope, contradicted by the code); each finding
  quoted from the spec, the important ones judged
  ([a spec](../roles/reviewer/docs/spec.md)).

## The backlog

### workline item ready

`workline item ready <id>` moves a work item to `ready`, once its Need,
Verification, Validation and Scope are written (`--forge` reads it from
the forge).

### workline issues

`workline issues`, `workline issues show <n>` or `show !<n>` reads the
local forge (`forge: local`,
[ADR-0016](adr/0016-writes-go-where-the-project-lives.md)): lists the
issues (`#<n>`) and merge requests (`!<n>`) kept in the clone, with their
state and labels, or shows one whole, its comments after its body. It
writes nothing.

### workline issues import

`workline issues import <file> [--apply]` moves a roadmap or backlog file,
whatever its form, to the forge's issues
([backlog acts](spec/backlog-acts/import.md)).

- The product owner reads it a share at a time; each item still to do is
  opened once, its text quoted from the file.
- **Without `--apply`**: only says what it would open, writes nothing, and
  lists its runs as pending (`to apply:`; `--json`: `pending`).
  `workline apply --line <result>` opens them with no agent, in the job
  that holds the write token.
- **The map**
  ([ADR-0030](adr/0030-an-import-maps-every-item-to-its-issue-or-why-not.md)):
  each item of the file, by its lines and first words, to its issue —
  opened, already open, closed — or why it has none: done, the words
  quoted; not an item, why; past the cap.
  - The lines no answer holds are listed under "Not covered", which exits
    2 (`--json`: the result's `coverage`).
  - The map stays the judge's: run the import again to see the issues
    open.
- **Options**: `--ai`, `--forge`, `--lines`; `--summary` as for `route`,
  the map with it.
