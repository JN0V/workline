<!-- workline
sources: [cmd/workline, tests/conformance/pending.txt]
checked: cdd117f
verified: agent:claude-code
-->
# Contributing

Thank you for helping. workline is built by its own line: the rules an
agent working here follows are in [AGENTS.md](AGENTS.md), and they hold
for people too. Every choice is checked against
[the principles](docs/PRINCIPLES.md).

## Build and test

```sh
git clone https://github.com/JN0V/workline && cd workline
go build -o ~/.local/bin/workline ./cmd/workline   # or anywhere on your PATH
go test -count=1 ./...    # unit tests and the conformance suite
```

`-count=1` matters: the conformance suite builds the engine itself, which Go's
test cache does not see. The evaluation grades the roles with a real agent on
real cases, and costs tokens, so it runs only when asked:
`WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/`
([conformance](docs/spec/conformance.md#evaluation)).

## Before you build

- **Look for an issue**, or open one: the work to do is in
  [GitHub's issues](https://github.com/JN0V/workline/issues), kept in order
  by the product owner role.
- **Borrow before building**: look for what exists, in the ecosystem's
  words ([docs/research/](docs/research/)), before designing something new.
- **A behaviour starts as a conformance case** in
  `tests/conformance/cases/`; `tests/conformance/pending.txt` lists what
  does not pass yet, and must never lie.

## Changes

- **Commits**: conventional (`type(scope): summary`), atomic, each one
  building; a subject of 72 characters at most, with no internal code in it
  (`Refs:` in a trailer). With workline's hooks installed, the committer
  checks them as you write.
- **Docs follow the code**: a doc that describes code names it in
  `sources`, with the commit it was `checked` against. A push says how many
  docs your commits made suspect: fix them in the same pull request.
- **Decisions** go in [docs/adr/](docs/adr/), findings in docs/research/.
- **Pull requests** only reach `main`, the checks green; a person reviews
  and merges. The CI runs the committer, the documentalist and the
  reviewer on each one.
- Never commit a secret nor a forbidden term: the committer scans each
  commit with gitleaks.
