---
sources: [roles/product-owner/instruction.md, internal/backlog/backlog.go, internal/work/work.go]
checked: 60d5f8c
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
| The builder's, below | Verification, Scope | who builds it |

- **Four sections are required**: Need, Verification, Validation, Scope
  ([what ready means](../../../docs/spec/backlog-acts.md#refining-to-ready)).
- **Example and Steps to reproduce are not**, nor any other heading you
  add: kept as they are.
- The product owner adds a missing section where this layout puts it,
  never moving yours, and the line between the two parts.

## The rules

- **Need**: one line — who needs what, and why. Not the solution.
- **Example**: one real case — a named person or role, numbers with
  their unit, a real command or output, today next to what is wanted.
- **Validation**: named scenarios a person runs, as Given / When / Then,
  in the issue's language (French: Étant donné / Quand / Alors).
- **A bug**: the Need says who it hurts; Steps to reproduce, then
  Expected, Actual, the version; its Validation repeats the steps, the
  expected result as the Then.
- **Short**: sentences of 25 words at most, one idea a bullet, plain
  words, the issue's language; code and messages as they print.
- **Paths and numbers below the line**: file paths, issue numbers and
  decisions go in Scope, not in Need nor Validation.
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

## Verification

- A finding on both the base and the head is not counted; one only on
  the head is.
- A tool that did not run is an error, never a pass.

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

## Verification

`TestExcludeTrailingSlash` fails on 2.3.1, passes after the fix.

## Scope

`matchExclude` in `internal/filter/glob.go`.
```

## Text a role wrote

What an import or another role opened an issue with, above its sections,
is the role's until a person edits it:

- **Rewritten in plain words**, the problem first, one real case, at most
  100 words; what only a builder needs — the code, a path, the fix —
  folded in `<details>` at its end.
- **Kept**: the line saying where it comes from, the hidden markers.
- **Who decides**: the engine, from the digest it kept when the issue was
  opened, never the agent. Edited by you, the text is yours: never
  rewritten. Rewritten once, again only when your comment asks.
- **Your level decides how**: written at `normal`, proposed at
  `cautious`, like the drafts.

## What the product owner says it changed

- **One line on the issue**, in its comment: "Added: Verification, Scope.
  Changed: —. See the issue's edit history."
- **Changed** lists only its own drafts it rewrote; it never changes a
  sentence of yours.
- **Rewrote: the description, in plain words** when it rewrote a role's
  text on top.
- **The text before** is in the forge's edit history: "edited" above the
  description on GitHub, Gitea and Forgejo; the "changed the
  description" note on GitLab, compared with the version before on its
  Premium plan.
