---
sources:
  - internal/gate/gate.go#RunWith
  - internal/gate/gate.go#runner.check
  - internal/gate/gate.go#execute
  - internal/gate/gate.go#rerun
  - internal/gate/gate.go#change.suppressions
  - internal/gate/change.go#changeSince
  - internal/gate/change.go#change.readPatch
  - internal/linters/check.go#suppressions
  - internal/linters/check.go#Suppressed
  - cmd/workline/main.go#gateCmd
checked: e2e165c2d
verified: agent:claude-code
---
# Gates — how the new findings are told

Part of [Gates](gates.md#on-a-merge-request-the-findings-the-change-adds),
beside [the findings a change adds](gates-new.md).

## Told by the tool, suppressions, errors

- **Every check is given the merge base** in `WORKLINE_BASE`, an `exit`
  check too: its command may tell the new findings itself. Where every
  finding counts, `WORKLINE_BASE` is empty.
- **Or the tool tells them**, `new: tool`: the SARIF check runs once, what
  it reports counts. `golangci-lint run
  ${WORKLINE_BASE:+--new-from-merge-base=$WORKLINE_BASE}`, Semgrep's
  `--baseline-commit`. golangci-lint's keeps changed lines only: a
  finding the change causes on a line it left is missed.
- **A suppression the change adds is a finding** (`suppression-added`),
  in a file of its language, when a check runs the tool it silences:
  `//nolint` (golangci-lint), `# noqa` (Ruff, flake8), `eslint-disable`
  (ESLint), `NOLINT` (clang-tidy). One only moved is not.
- **An error, never a pass**: a merge base not known (a shallow clone), a
  check that cannot run there or writes no SARIF, a run past its cap. The
  cap is `base-timeout`; by default twice the head's run, a minute at
  least, the check's `timeout` at most (measured in
  [ADR-0049's plan](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#build-plan)).
- **A tool whose version differs** between the two runs is said
  (`tool-version-differs`, a warning): one the project's tree pins, upgraded
  by the change. A tool's upgrade gets a pull request of its own.
- **A cap on findings hides some**: set the tool's to none
  (golangci-lint's `max-issues-per-linter`, `max-same-issues`).

## With `workline gate`

- **`workline gate <name> --base <rev>`** compares as a merge request
  does, since the merge base of `<rev>` and `HEAD`: a CI job of its own,
  a check by hand. With no `--base`, every finding counts.
- **`--gates-kept <folder>`**, with `--base`: the merge base's findings
  read as main kept them, by the rules
  ([the findings a change adds](gates-new.md)); else run again there.
- **A `<rev>` that is no commit here is an error**, never a run counting
  every finding; an empty one (`--base ""`, a variable unset) is misuse.
- **Each summary says how it counted**: `only the findings the change
  adds since <merge base> count`, or `every finding counts, none
  compared with a base`.
