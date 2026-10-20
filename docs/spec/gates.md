---
sources: [internal/gate, internal/line, internal/linters/check.go, internal/report/report.go, internal/report/gates.go, internal/engine/annotate.go]
checked: 6970b95e
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

## On a merge request, the findings the change adds

On an event given a range (`merge-request`, `pre-push`), the thresholds
read only the findings the change adds since the merge base of the
range's start and `HEAD`
([ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md)).
On `schedule` and `release`, and with `workline gate <name>` given no
`--base` ([below](#with-workline-gate)), every finding counts. On `main`
after a merge, every finding is read and kept, and what the push adds
since the branch's tip before it counts.

```yaml
      - id: lint
        run: golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --output.sarif.path={out}/lint.sarif ./...
        output: sarif
        max: {error: 0}
```

- **Run again at the merge base**: a SARIF check runs a second time, on
  the merge base checked out in a tree of its own, outside the repository.
  The same command and tool, and the head's config: the files the change
  touched that its tool reads its config from, by name
  ([the tools' table](../languages.md#the-table-it-reads)), or that the
  check's `config` names (`config: [pyproject.toml]`, globs).
- **Or read as main kept it** ([ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md)):
  with `--gates-kept <folder>`, main's run after a merge writes every
  finding of each SARIF check there, by its commit (`<commit>.json`);
  a change whose merge base it holds reads them instead of running the
  tool there. Only for the same command, `output` and `config`, no file
  the tool reads its config from changed by the change, and the same
  tool: the version its SARIF names, the hash of the binary found on the
  PATH; one naming neither runs again. Else, or none kept for that
  commit, run again, the summary saying why. The folder is filled only
  from main's own runs, never from a store a branch could write
  ([main after a merge](../main-after-merge.md#the-next-base)).
- **Compared one for one, never by line number**: the tool, the rule, the
  file (a rename followed), then the tool's own `fingerprints`, else its
  `partialFingerprints`, else the message and the text of the line, its
  spaces left out. `workline/v1` is not read: it holds a line.
- **Moved is not new**: a finding pushed down its file, or in a file
  renamed. One cut to another file is new, as Semgrep and code scanning
  count it.
- **Day one is quiet**: the old findings are a count in the gate's
  summary, with both runs' times: `lint 1 new, 360 before it (6.5s, then
  7.1s at the merge base)`.
- **Each result marked** for what posts them: the head's SARIF written
  again under `compared/<check>/` in the gate's folder, every result's
  `baselineState` `new` or `unchanged` (SARIF 2.1.0, 3.27.24).

### Told by the tool, suppressions, errors

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
  check that fails there or writes no SARIF, a run past its cap. The cap
  is `base-timeout`; by default twice the head's run, a minute at least,
  the check's `timeout` at most (measured in
  [ADR-0049's plan](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#build-plan)).
- **A tool whose version differs** between the two runs is said
  (`tool-version-differs`, a warning): one the project's tree pins, upgraded
  by the change. A tool's upgrade gets a pull request of its own.
- **A cap on findings hides some**: set the tool's to none
  (golangci-lint's `max-issues-per-linter`, `max-same-issues`).

### With `workline gate`

- **`workline gate <name> --base <rev>`** compares as a merge request
  does, since the merge base of `<rev>` and `HEAD`: a CI job of its own,
  a check by hand. With no `--base`, every finding counts.
- **`--gates-kept <folder>`**, with `--base`: the merge base's findings
  read as main kept them, by the rules above; else run again there.
- **A `<rev>` that is no commit here is an error**, never a run counting
  every finding; an empty one (`--base ""`, a variable unset) is misuse.
- **Each summary says how it counted**: `only the findings the change
  adds since <merge base> count`, or `every finding counts, none
  compared with a base`.

### Shown on the merge request

- **Where the forge shows a finding beside its line**, with no
  conversation to resolve: on GitHub, the annotations of the check
  `workline gates` and one summary comment; on GitLab, the Code Quality
  report. Never code scanning
  ([ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md)).
- **The gates alone, in a job of their own** (`--gates-only`), with no
  write token and no agent's; the judging job reads their verdicts
  (`--gates-from`) and runs no gate.
- **Each new finding's fingerprint** holds no line number.
- **How, and what is said**: [the new findings shown](gates-shown.md), [explained](../../roles/inspector/README.md).

*Not built yet*: a formatter's fixes applied, rather than its command
said; the count's trend; vendored and generated code skipped; CI's build
reused; a monorepo's changed folders only. On GitLab, the agent's and the
bot's tokens are in every job's variables: the gates' job unsets them,
which keeps them from the tools' environment, not from whoever changes
the pipeline.

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
check, and keeps each check's output under `.git/workline/gates/`: its log,
its SARIF, and on a merge request the run at the merge base under `base/`
and the results marked under `compared/`. A check over its thresholds names
its first findings, file and line; no agent is ever asked, so the verdict
and the findings are the same with an agent or none.

## Protection

The `gates:` and `routing:` sections of `.workline/config.yaml` and the role
folders can only be changed by people,
never by an agent's patch: they are outside every role's `duties.writes`, and
should be protected on the forge (CODEOWNERS or equivalent). A gate the agent
can edit is not a gate.
