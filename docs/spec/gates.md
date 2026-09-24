---
sources: [internal/gate]
checked: d30d22a
---
# Gates — v1 (draft)

A gate is a checkpoint the work must pass before going further: before merging,
before releasing. A gate runs tools, reads their output, and gives a verdict by
rules. It never asks a model whether to pass.

## Where gates are declared

In the `gates:` section of `.workline/config.yaml`, one entry per gate, run with
`workline gate <name>`, or as `gate:<name>` in a routing sequence:

```yaml
gates:
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
      - id: versions
        run: python3 tools/check_versions.py   # a project's own script, reused as is
        output: exit
    enforce:
      dependencies: warn                # new rules start in warn
```

- `run` is any command. `{out}` is the gate's output directory.
- `output` says how to read the result: `exit` (the exit code decides) or
  `sarif` (results counted against `max`, by SARIF level; a result without a
  level counts as `warning`). Other structured outputs (k6 or benchmark JSON)
  are for a later version.
- A check with no threshold is refused when the file is loaded. The criteria
  are decided before the gate runs, never while reading the results.
- `enforce` and `warn` before `block` work as in the role contract; baselines
  with expiry dates are not built yet, there or here.

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

A gate gives the same verdict as a role (`--json`), with the findings of every
check, and keeps each check's output under `.git/workline/gates/`. *Not built
yet:* merging the SARIF outputs and posting them on the merge request
(reviewdog-style), on GitHub and GitLab alike.

## Protection

The `gates:` and `routing:` sections of `.workline/config.yaml` and the role
folders can only be changed by people,
never by an agent's patch: they are outside every role's `duties.writes`, and
should be protected on the forge (CODEOWNERS or equivalent). A gate the agent
can edit is not a gate.
