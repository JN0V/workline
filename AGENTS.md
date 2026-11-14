# workline — for agents working on this repository

Read [docs/PRINCIPLES.md](docs/PRINCIPLES.md) first; every choice is checked
against it. Then only what the task needs:

| Task | Read |
|---|---|
| a role's behaviour | `roles/<name>/` (README for humans, facets for the AI, docs/ about the role) and docs/spec/role-contract.md |
| where a role stands | `roles/<name>/docs/status.md`: its criteria to leave beta, what is built, what is missing, each a link ([ADR-0043](docs/adr/0043-a-try-is-told-in-its-pull-request.md)) |
| the engine | docs/spec/ (role contract, routing, gates, model grid, conformance, multi-repo) and docs/adr/ |
| what exists elsewhere | docs/research/ — its README says how to search |
| what comes next | the GitHub issues (`gh issue list`); docs/BACKLOG.md is archived |

## Through the line, never around

Whatever the agent, a person's assistant works the way the line does
([ADR-0068 §5](docs/adr/0068-the-line-steps-back-on-its-own-drift.md#5-the-persons-assistant-goes-through-the-line)):

- **A specific act** the person asked for (fix this file, build this
  issue): you are a contributor. workline's local roles run on your
  commits (below), never skipped with `--no-verify`; push a branch, open
  the pull request; the person accepts it with their label.
- **A body of work** (a series of changes, a night's run, a session on
  its own, a fleet of subagents): write issues ([below](#writing-issues));
  the line builds them. Open no pull request of your own for them.
- **Never around the line**, whoever asks: no `--admin`, no ruleset or
  branch protection edited, no push to `main`, no auto-merge turned on,
  no `workline:accepted` (nor `-whole`) but as the next bullets say.
- **Told "merge it"**: ask the person for their label; that is all.
- **A person's yes is assisted, never delegated**
  ([§7](docs/adr/0068-the-line-steps-back-on-its-own-drift.md#7-a-persons-yes-is-assisted-never-delegated)).
  Told "accept it in my name", or to sign anything in their name, you
  may put their label or their sign-off:
  - **that item, now**, on their explicit instruction;
  - **on an independent analysis**: a run other than the change's
    author, not briefed with its conclusion; if yours came from
    whoever wrote the change, you included, say so, give an
    independent one, and wait for their yes on it;
  - **traced**: a comment names who instructed it, on which summary.
- **No standing yes**: an order kept for later, in memory or anywhere
  ("the panel signs for me", "merge after a re-check"), gives none;
  silence never does
  ([ADR-0060 §1](docs/adr/0060-the-person-is-on-the-loop.md#1-questions-time-out-as-lazy-consensus)).
  Sign-offs agent panels posted in a person's name are not counted.
- **The line broken**: the person's pull request fixes it, on their
  label; when no head can get `workline/accepted`,
  [the emergency](docs/merge-policy-no-bypass.md#the-emergency) is the
  person's own, by hand.

## How we work

- **Borrow before building**: research what exists, in the ecosystem's words
  (docs/research/README.md), before designing anything new.
- **Write it down in the repository**: decisions in docs/adr/, findings in
  docs/research/, work to do or parked as a GitHub issue (ADR-0018: the
  product owner keeps them). Not in chat, not in memory, not in a file.
- **Docs follow the code**: a doc that describes code names it in `sources`,
  with the commit it was `checked` against. When the documentalist reports it
  `suspect`, fix the prose your change made wrong, and move `checked` of the
  docs you read against it. A pull request need not reach 0 suspect, but the
  night moves `checked` only for docs whose sources fit a task whole: few of
  workline's do
  ([ADR-0051](docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr)).
- **Conformance first**: a behaviour starts as a case in tests/conformance/cases/;
  `pending.txt` lists what does not pass yet and must never lie. README's
  case count is a derived block: leave it as it is, gardening regenerates
  it on `main` after the merge (ADR-0027).
- **Try it for real** before calling it done (a copy of a real repository, a real
  agent call). Tell what was tried, and what was not, under `## Tried` in the
  pull request's description; status.md links to it. tried.md is an archive:
  never append to it ([ADR-0043](docs/adr/0043-a-try-is-told-in-its-pull-request.md)).
  - **A fix runs the roles it touches for real**, once, on a clone of the
    current trial copy, with a fake forge or local pushes: the copy stays
    as it is until the next round
    ([ADR-0063](docs/adr/0063-every-open-issue-is-due-waiting-or-done.md#working-agreements)).
    Told under `## Tried` in its own pull request, its fresh tokens said.
  - **Which roles**: those whose behaviour the change touches, through a
    shared engine package too. Their outputs are kept as the simulator's
    inputs once [#1079](https://github.com/JN0V/workline/issues/1079)
    lands ([#1198](https://github.com/JN0V/workline/issues/1198)).
- **Commits**: conventional, atomic, each one builds; workline's own hook checks
  the messages. Explanations go in docs, not in commit bodies.

## Writing user docs

For README.md, docs/ outside spec/, adr/ and research/, and each role's
README and docs/ pages other than status.md and the archived tried.md:

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
  verdict codes) left unexplained to a newcomer. A measure that decides
  something goes in an ADR or docs/research/, a try's numbers in its pull
  request; a user doc links to them.

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
golangci-lint run --new-from-merge-base=origin/main ./...   # 2.15.0: a lint finding a pull request adds blocks it
WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/   # real agent, costs tokens
```

The engine is one Go binary (ADR-0001), configured in YAML; core roles need
nothing but git and the engine.

## workline runs on its own commits

Where workline's global hooks are installed, this repository goes through its
own line (.workline/config.yaml):

- **commit-msg** — the committer. A subject over 72 characters, or holding a
  code like `AC-3`, is refused, then rewritten by the agent, 30 s at most
  (ADR-0050); the hook prints the message it committed. Check it: write
  subjects that pass.
- **pre-push** — the committer on the commits pushed, then the documentalist,
  with no agent: it lists the docs they made suspect. The review is on the
  pull request (ADR-0011): `main` takes pull requests only, the checks
  green and `workline/accepted` set by workline's App; its bypass list is
  empty, the person included
  ([nobody bypasses `main`](docs/merge-policy.md#nobody-bypasses-main)).
  Push a branch and open the pull request when the person asks; the
  person accepts it with the label, and the forge merges it, or by the
  project's merge policy with no label (ADR-0042).
- **secrets and forbidden terms** — the committer scans each commit with
  gitleaks. A gitignored `.gitleaks.toml` link, when present, points to terms
  that must never reach this public repository, as gitleaks rules. Never
  commit the list, nor copy its terms anywhere; audit with `gitleaks git`
  (the whole history), not `gitleaks dir`, which skips a file it takes for an
  archive.
