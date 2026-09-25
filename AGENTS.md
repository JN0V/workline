# workline — for agents working on this repository

Read [docs/PRINCIPLES.md](docs/PRINCIPLES.md) first; every choice is checked
against it. Then only what the task needs:

| Task | Read |
|---|---|
| a role's behaviour | `roles/<name>/` (README for humans, facets for the AI) and docs/spec/role-contract.md |
| the engine | docs/spec/ (role contract, routing, gates, model grid, conformance, multi-repo) and docs/adr/ |
| what exists elsewhere | docs/research/ — its README says how to search |
| what comes next | docs/BACKLOG.md |

## How we work

- **Borrow before building**: research what exists, in the ecosystem's words
  (docs/research/README.md), before designing anything new.
- **Write it down in the repository**: decisions in docs/adr/, findings in
  docs/research/, parked topics in docs/BACKLOG.md. Not in chat, not in memory.
- **Docs follow the code**: a doc that describes code names it in `sources`,
  with the commit it was `checked` against. When the documentalist reports it
  `suspect`, check it against the change, fix what is wrong, and move `checked`.
- **Conformance first**: a behaviour starts as a case in tests/conformance/cases/;
  `pending.txt` lists what does not pass yet and must never lie.
- **Try it for real** before calling it done (a copy of a real repository, a real
  agent call), and record what was tried and what was not.
- **Commits**: conventional, atomic, each one builds; workline's own hook checks
  the messages. Explanations go in docs, not in commit bodies.

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
- **pre-push** — the committer on the commits pushed, then the documentalist:
  docs made suspect are judged, patched in the working tree, and the push stops
  for them to be reviewed and committed.
- **forbidden-terms** — a gitignored `.forbidden-terms` link, when present,
  points to terms that must never reach this public repository. Never commit
  the list, nor copy its terms anywhere.
