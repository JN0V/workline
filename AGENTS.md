# workline — for agents working on this repository

Read [docs/PRINCIPLES.md](docs/PRINCIPLES.md) first; every choice is checked
against it. Then only what the task needs:

| Task | Read |
|---|---|
| a role's behaviour | `roles/<name>/` (README for humans, facets for the AI, docs/ about the role) and docs/spec/role-contract.md |
| where a role stands | `roles/<name>/docs/status.md` (built, tried, missing), then `docs/tried.md` |
| the engine | docs/spec/ (role contract, routing, gates, model grid, conformance, multi-repo) and docs/adr/ |
| what exists elsewhere | docs/research/ — its README says how to search |
| what comes next | the GitHub issues (`gh issue list`); docs/BACKLOG.md is archived |

## How we work

- **Borrow before building**: research what exists, in the ecosystem's words
  (docs/research/README.md), before designing anything new.
- **Write it down in the repository**: decisions in docs/adr/, findings in
  docs/research/, work to do or parked as a GitHub issue (ADR-0018: the
  product owner keeps them). Not in chat, not in memory, not in a file.
- **Docs follow the code**: a doc that describes code names it in `sources`,
  with the commit it was `checked` against. When the documentalist reports it
  `suspect`, check it against the change, fix what is wrong, and move `checked`.
- **Conformance first**: a behaviour starts as a case in tests/conformance/cases/;
  `pending.txt` lists what does not pass yet and must never lie. README's
  case count is a derived block: leave it as it is, gardening regenerates
  it on `main` after the merge (ADR-0027).
- **Try it for real** before calling it done (a copy of a real repository, a real
  agent call), and record what was tried and what was not.
- **Commits**: conventional, atomic, each one builds; workline's own hook checks
  the messages. Explanations go in docs, not in commit bodies.

## Writing user docs

For README.md, docs/ outside spec/, adr/ and research/, and each role's
README and docs/ pages other than status.md and tried.md:

- **Bullets over paragraphs**: short prose is fine, not a novel. A paragraph
  over ~80 words is a defect; a section over 400 words is too long; a doc
  over 200 lines is split into pages linked from it (the documentalist's
  budgets check the last two).
- **An overview stays an overview**: the README, docs/roles.md and other
  front tables hold one short sentence and a link per row; no ADR lists,
  settings or edge cases. Detail goes on the role's page or a page below it.
- **Every ADR, doc or issue cited is a link**: never a bare `ADR-0026`,
  `docs/x.md` or `#128`.
- **Diagrams readable at a glance**: macro blocks first, a few words each;
  detail goes in the text below. Each role's README opens with one.
- **Install covers a person's machine and CI**; the README links to the docs
  early.
- **The agent is a choice, never imposed**: none, `claude`, `cmd:<any
  command>`. An example naming one agent says the others exist.
- **No unexplained internals**: no sandbox issue numbers, token histories
  ("430k before #147"), other projects by name, internal words ("vouched",
  verdict codes) left unexplained to a newcomer. Measures go in status.md,
  tried.md or an ADR; a user doc links to them.

## Writing issues

An issue opened in a session follows the product owner's rules
([writing an issue](roles/product-owner/docs/writing.md), with two
examples):

- **The person's part on top**: Need in one line (who needs what, why),
  one real example (names, numbers with units, a real command or
  output), Validation as named Given / When / Then scenarios.
- **A line `---`, then the builder's part**: Verification, Scope. File
  paths and issue numbers go there, not in Need nor Validation.
- **A bug**: who it hurts, then Steps to reproduce, Expected, Actual,
  the version.
- **Short sentences**, one idea a bullet, no number you did not see.

## Build and test

```sh
go build -o ~/.local/bin/workline ./cmd/workline
go test -count=1 ./...     # -count=1: the conformance suite builds the engine itself
WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/   # real agent, costs tokens
```

The engine is one Go binary (ADR-0001), configured in YAML; core roles need
nothing but git and the engine.

## workline runs on its own commits

Where workline's global hooks are installed, this repository goes through its
own line (.workline/config.yaml):

- **commit-msg** — the committer. A subject over 72 characters, or holding a
  code like `AC-3`, is refused, then rewritten by the agent; the hook prints
  the message it committed. Check it: write subjects that pass.
- **pre-push** — the committer on the commits pushed, then the documentalist,
  with no agent: it lists the docs they made suspect. The review is on the
  pull request (ADR-0011): `main` takes pull requests only, the checks
  green. Push a branch and open the pull request when the person asks;
  never merge it, never turn auto-merge on: the person does.
- **secrets and forbidden terms** — the committer scans each commit with
  gitleaks. A gitignored `.gitleaks.toml` link, when present, points to terms
  that must never reach this public repository, as gitleaks rules. Never
  commit the list, nor copy its terms anywhere; audit with `gitleaks git`
  (the whole history), not `gitleaks dir`, which skips a file it takes for an
  archive.
