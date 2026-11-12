---
sources: [roles/product-owner/instruction.md, internal/backlog/backlog.go, internal/backlog/repeat.go, internal/work/work.go, internal/backlog/plain.go, internal/backlog/assume.go]
checked: 9ae60ea8
verified: agent:claude-code
---
# Writing an issue a person reads

Part of [the product owner](../README.md). The rules it writes issues by;
the same for a person, or a coding agent, opening one, and for
[the reviewer](../../reviewer/docs/run.md)'s issues: what a person sees
first, the cause folded.

## The layout

| Part | Sections | For |
|---|---|---|
| The person's, on top | Need, Example (a bug: Steps to reproduce), Validation | who decides and accepts |
| A line | `---` | the reader may stop here |
| The builder's, below | Verification (the tester's), Scope | who builds it |

- **Three sections are required** to be ready: Need, Validation, Scope
  ([what ready means](../../../docs/spec/backlog-acts/refine.md)).
- **The Verification is the tester's**: how the machine proves it, which
  test proves which scenario. The tester writes it from its tests once
  the issue is ready, as a tester writes the test plan; one you wrote is
  kept
  ([ADR-0066](../../../docs/adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md)).
  The product owner writes none.
- **Example and Steps to reproduce are not**, nor any other heading you
  add: kept as they are.
- **Your text on top, with no Need heading, is the Need**: the product
  owner drafts no Need saying it again.
- The product owner adds a missing section where this layout puts it,
  never moving yours, and the line between the two parts.

## The rules

- **Need**: one line — who needs what, and why. Not the solution.
- **Example**: one real case — a named person or role, numbers with
  their unit, a real command or output, today next to what is wanted;
  only with a fact the Need lacks.
- **Validation**: named scenarios a person runs, as Given / When / Then,
  each a bullet `- **Name:** Given …, When …, Then …`.
  - **Also read**: a `**Scenario: Name**`, `Scenario: Name` or
    `### Scenario: Name` line, its steps below, bullets or not; the
    same in a fenced `gherkin` block.
  - **Given / When / Then with no name**: no test can cite it, so the
    issue stays in refine, said on it, until the scenarios are named
    ([refining to ready](../../../docs/spec/backlog-acts/refine.md)).
- **A bug**: the Need says who it hurts; Steps to reproduce, then
  Expected, Actual, the version; its Validation repeats the steps, the
  expected result as the Then.
- **Short**: sentences of 25 words at most, one idea a bullet, plain
  words, in English; code and messages as they print.
- **Paths and numbers below the line**: file paths, issue numbers and
  decisions go in Scope, not in Need nor Validation. A Scope is written
  from the code it names, read first; one still a guess says so on a
  line `Guessed:`, and holds no other issue back
  ([#970](https://github.com/JN0V/workline/issues/970)).
- **A Scope is paths**: each file a path from the root, or a glob
  (`core/session/**`), in backticks; never a class or module name
  alone. A hint, not a fence: the developer lists each file it changes
  beyond them, with why
  ([ADR-0065](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence)).
  - A file to create: on a line `New:`, its full path.
  - A path the repository lacks, not on a `New:` line: the Scope is
    left out, said. A path its `.gitignore` matches, a file the build
    writes, is kept as a value.
- **Fold only what is long** — a log, a list of more than ten lines — in
  `<details>`.
- **Never invent**: no real case known, ask the reporter for one.

## A need

```markdown
## Need

When I open a merge request on a repository with old lint debt, I want
to see only the findings my change adds, so old code does not block me.

## Example

`main` has 140 lint findings. Alice adds one unchecked error in
`report.go`. The gate counts 141 against a limit of 100 and blocks her;
nothing says which one is hers.

## Validation

- **Old debt:** Given `main` with 140 findings, When a merge request
  adds one unchecked error, Then only that one blocks, shown on its line.
- **Moved code:** Given a change that moves an old finding 10 lines
  down, When the gate runs, Then it is not counted as new.

---

## Scope

`internal/gate`: the runs on the base and the head.
<details><summary>Tools a project may declare</summary>golangci-lint,
govulncheck, Semgrep, CodeQL, osv-scanner, Trivy, ESLint, Ruff</details>
```

## A bug

```markdown
## Need

Users who exclude a folder with a trailing slash still upload it:
1.2 GB, 9 minutes.

## Steps to reproduce

1. `snapsync push --exclude node_modules/ ~/projects/site`
2. Output: `uploaded 18412 files (1.2 GB) in 9m02s`
- **Expected:** about 40 files. `--exclude node_modules` (no slash) works.
- **Actual:** `node_modules` uploaded. Version 2.3.1, Linux, every time.

## Validation

- **Trailing slash:** Given a folder `node_modules`, When I run
  `snapsync push --exclude node_modules/`, Then no file under it is
  uploaded.

---

## Scope

`matchExclude` in `internal/filter/glob.go`.
```

## Each thing said once

A problem said three times reads as three problems
([ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md),
amended; [worked case](proposed-headings.md#each-thing-said-once)):

- **A role's text on top becomes the Need draft**: what an import or
  another role opened the issue with, rewritten in plain words, then
  taken from the top. The line saying where it comes from, and the
  hidden markers, stay. What it folded there, a cause and its fix,
  moves whole to the end of the Scope written; with none written, it
  stays on top
  ([#841](https://github.com/JN0V/workline/issues/841)).
- **Your text on top is the Need**, when no Need heading follows: the
  product owner adds Validation and the builder's part, never a Need
  restating it.
- **A draft that says again what is above it is not written**: past
  `repeated-percent-max` ([settings](settings.md)), 40% of its words
  already said. Validation is never measured: it restates the case.
- **What you said yes to is written whole**: a proposal you accepted, a
  text you agreed to.
- **Who decides whose text it is**: the engine, from the digest it kept
  when the issue was opened, never the agent. Edited by you, the text
  is yours: never taken.
- **Your level decides how**: written at `normal`, proposed at
  `cautious`, like the drafts.

## What the product owner says it changed

- **One line on the issue**, in its comment: "Added: Scope. Changed: —.
  See the issue's edit history."
- **Changed** lists the sections it rewrote:
  - its own drafts, or a text as it wrote it;
  - the Scope, even yours, to answer the reviewer
    ([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md)):
    it is its to keep, at every level; never the Verification, the
    tester's, a finding there answered by the tester;
  - never a sentence of your Need, Example or Validation.
- **Rewrote: the text on top as the Need, in plain words** when it took
  a role's text on top into the Need.
- **Not written: the Example, which said again what is above it** when
  it left a draft out.
- **The text before** is in the forge's edit history: "edited" above the
  description on GitHub, Gitea and Forgejo; the "changed the
  description" note on GitLab, compared with the version before on its
  Premium plan.

## Assumptions

A question nobody answers stops nothing
([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md#1-questions-time-out-as-lazy-consensus)):
past the level's delay ([settings](settings.md#how-long-a-question-waits)),
the product owner goes on with its best answer, written plainly.

```markdown
## Assumptions

- Assumption taken on 2026-11-02, the question of 2026-11-01 unanswered
  after 24 h: `--verbose=false` prints the plain line, as other flags do.
```

- **Its own section, at the end of the body**: your Need, Example and
  Validation are never rewritten; the assumption sits beside them.
- **Said in its comment** too: "Assumption taken on #844: …. Answer
  here, or edit its line under Assumptions in the issue, to change it."
- **A draft unanswered** is taken as written, its line saying so.
- **To correct one**: answer on the issue, or edit its line. Your answer
  marks the line "Answered by @you", and the product owner reworks what
  it built on it, back to refine if it was ready.

## Proposed next

New names for these sections, and a bug readied only with its steps,
are proposed, not built:
[the proposed headings](proposed-headings.md),
[#435](https://github.com/JN0V/workline/issues/435). The kind set by
the role is built: [the kind it sets](kind.md).
