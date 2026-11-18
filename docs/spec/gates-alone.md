---
sources:
  - internal/gate/added.go#Added
  - internal/gate/added.go#CheckCount
  - internal/gate/added.go#Finding
  - internal/gate/added.go#stamp
  - internal/line/line.go#runGate
  - internal/line/line.go#Gates
  - internal/line/line.go#Gates.step
  - internal/line/line.go#Gates.sameCommits
  - internal/line/line.go#GatesInput
  - internal/line/line.go#foundFor.read
  - internal/report/report.go#FromLine
checked: e2e165c2d
verified: agent:claude-code
---
# Gates — a gate step's result, and the gates alone

Part of [the new findings shown](gates-shown.md).

## What a gate step holds

- **On an event given a range**, its result in `--json` holds `new`: the
  merge base, each check's count (`new`; `before`, `-1` when not known;
  `failed`, `error`, `fix`; `kept`, the merge base's findings read as
  main kept them), and each new finding: check, tool, rule, SARIF level,
  file, line, message, and a fingerprint. On `main`, `present` names
  every finding the head holds, by its fingerprint
  ([main after a merge](../main-after-merge.md)).
- **The fingerprint** hashes the finding's identity
  ([the comparison](gates-new.md#compared-one-for-one))
  and how many alike come before it: no line number.

## The gates alone

- **`route <event> --gates-only`**: every gate step of the event runs,
  none of its roles. Their tools run the change's code: that job holds no
  write token and no agent's ([gates in CI](../gates-in-ci.md)).
- **`--gates-from <file>`**: the job judging the line reads each gate
  step's verdict there, and runs no gate. One missing from the file, or a
  file for other commits than the line judges, is a `check-error`: the
  gates' result says the commits its range named when they ran
  (`commits`), a branch moved since names others.
- **A gate's findings are reported by the job that ran it**: a step read
  from a file adds none to `--sarif` nor `--code-quality`.
- **The run apart** (`--warn-only --gates-from <file>`) reads no verdict
  there: it hands that result to the inspector, which triages the new
  findings, only for the commits it judges; for others, said
  (`gates-unread`), none given
  ([the inspector](../../roles/inspector/README.md)).
