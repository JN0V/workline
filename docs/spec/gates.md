---
sources:
  - internal/gate/gate.go#Check
  - internal/gate/gate.go#Gate
  - internal/gate/gate.go#defaultTimeout
  - internal/gate/gate.go#Load
  - internal/gate/gate.go#Gate.Validate
  - internal/gate/gate.go#RunWith
  - internal/gate/gate.go#enforce
  - internal/gate/gate.go#Run
  - internal/gate/gate.go#fixWith
  - internal/gate/gate.go#execute
  - internal/gate/gate.go#maskLog
  - internal/gate/gate.go#rerun
  - internal/gate/gate.go#overThreshold
  - internal/gate/gate.go#first
  - internal/gate/gate.go#outDir
  - internal/gate/sarif.go#parse
  - internal/line/line.go#runGate
  - cmd/workline/main.go#gateCmd
  - internal/developer/bounds.go#ciFiles
  - internal/developer/bounds.go#checker.rule
  - roles/committer/role.yaml#duties.writes
  - roles/developer/role.yaml#duties.writes
  - roles/documentalist/role.yaml#duties.writes
  - roles/documentalist/role.yaml#settings.docs
  - roles/inspector/role.yaml#duties.writes
  - roles/product-owner/role.yaml#duties.writes
  - roles/reviewer/role.yaml#duties.writes
  - roles/tester/role.yaml#duties.writes
  - roles/tester/role.yaml#settings.test-paths
checked: e2e165c2d
verified: agent:claude-code
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
  are decided before the gate runs, never while reading the results. So is
  a setting the engine cannot read: a `timeout` or `base-timeout` that is
  not a duration past zero, a `new` it does not know, an `id` given twice
  or holding a `/` (it names the check's log and results).
- `timeout` caps a check's run (`10m` unless set); `new`, `base-timeout`
  and `config` say how a merge request's new findings are told (below).
- `fix` is the command that fixes what the check finds, a formatter's
  (`gofmt -w .`): said with its findings, which are not shown line by
  line on a merge request (below).
- `enforce` names a check by its id, or a rule (`suppression-added`); `warn`
  before `block` works as in the role contract. A check `off` that did not
  run is still said, as a warning. Baselines with expiry dates are not
  built yet, there or here.
- `workline doctor` counts the tool a check runs as reading its languages,
  and names the check to add for a language none reads
  ([languages](../languages.md)).
- A name no gate holds (`workline gate <name>`, `gate:<name>`) is an
  error naming the gates declared, or saying none is.

## On a merge request, the findings the change adds

On an event given a range (`merge-request`, `pre-push`), the thresholds
read only the findings the change adds since the merge base of the
range's start and `HEAD`
([ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md)).
On `schedule` and `release`, and with `workline gate <name>` given no
`--base` ([by hand](gates-told.md#with-workline-gate)), every finding
counts. On `main` after a merge, every finding is read and kept, and
what the push adds since the branch's tip before it counts.

- **How they are told**: run again at the merge base, or read as main
  kept it; compared one for one, never by line number; a finding moved
  is not new ([the findings a change adds](gates-new.md)).
- **Told by the tool, suppressions, errors**, and `workline gate --base`
  by hand: [how the new findings are told](gates-told.md).

### Shown on the merge request

- **Where the forge shows a finding beside its line**, with no
  conversation to resolve, never code scanning; the gates alone
  (`--gates-only`), in a job of their own with no write token, the
  judging job reading their verdicts (`--gates-from`):
  [the new findings shown](gates-shown.md),
  [explained](../../roles/inspector/README.md).
- **On GitLab**, the tokens scoped to the environment `workline` never
  reach the gates' job, which declares none; they still reach whoever
  changes the pipeline ([gates in CI](../gates-findings.md#on-gitlab),
  [the environment](../gitlab/environment.md)).

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
check, and keeps each check's output under `.git/workline/gates/`: its log
(a secret's value masked, [#1215](https://github.com/JN0V/workline/issues/1215)),
its SARIF, and on a merge request the run at the merge base under `base/`
and the results marked under `compared/` ([how](gates-new.md)). A check
over its thresholds names its first findings, file and line; no agent is ever asked, so the verdict
and the findings are the same with an agent or none.

## Protection

The `gates:` and `routing:` sections of `.workline/config.yaml` and the role
folders can only be changed by people,
never by an agent's patch: they are outside every role's `duties.writes`
but the developer's, whose bounds refuse them, and should be protected on the forge (CODEOWNERS or equivalent). A gate the agent
can edit is not a gate.
