# Gates — v1 (draft)

A gate is a checkpoint the work must pass before going further: before merging,
before releasing. A gate runs tools, reads their output, and gives a verdict by
rules. It never asks a model whether to pass.

## `gates.yml`

```yaml
gates: 1

release:
  criteria: docs/nfr.md               # where the thresholds come from, for humans
  checks:                             # commands are examples; flags vary by tool version
    - id: secrets
      run: gitleaks detect --report-format sarif --report-path {out}/secrets.sarif
      output: sarif
      max: {error: 0}
    - id: dependencies
      run: osv-scanner scan --format sarif --output {out}/deps.sarif .
      output: sarif
      max: {error: 0, warning: 5}
    - id: load
      run: k6 run --summary-export {out}/load.json load/checkout.js
      output: exit                    # k6 thresholds already decide pass or fail
    - id: docs-pending
      run: workline docs pending --due release
      output: exit
  enforce:
    dependencies: warn                # new rules start in warn
```

- `run` is any command. `{out}` is the gate's output directory.
- `output` says how to read the result: `exit` (the exit code decides), `sarif`
  or `json` (findings counted against `max`, by severity).
- A check with no threshold is refused when the file is loaded. The criteria
  are decided before the gate runs, never while reading the results.
- `enforce`, baselines with expiry dates, and `warn` before `block` work as in
  the role contract.

## Three outcomes, not two

Each check ends as:

- **pass** — it ran, and the findings are within the thresholds;
- **finding** — it ran, and the findings exceed them;
- **error** — the check itself did not work: missing tool, crash, unreadable
  output, timeout.

An error fails the gate. A check that did not run has proven nothing, and a gate
that passes because a scanner was missing is worse than no gate. A check can be
marked `optional: true`; it still reports that it did not run.

## Where AI fits

AI can run inside a check — an agent-driven pentest, an architecture review —
but its findings count against the thresholds only when reproduced by a
deterministic step (the exploit replays, the rule fires). Otherwise they are
reported as advisory. The verdict comes from the rules either way.

## Verdict and report

A gate writes the same `verdict.yaml` as a role, with one finding list per
check. SARIF outputs are merged and posted on the merge request by the forge
adapter (reviewdog-style), on GitHub and GitLab alike.

## Protection

`gates.yml`, `routing.yml` and the role folders can only be changed by people,
never by an agent's patch: they are outside every role's `duties.writes`, and
should be protected on the forge (CODEOWNERS or equivalent). A gate the agent
can edit is not a gate.
