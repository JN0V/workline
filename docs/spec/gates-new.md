---
sources:
  - internal/line/line.go#runGate
  - internal/gate/gate.go#runner.check
  - internal/gate/gate.go#rerun
  - internal/gate/gate.go#Check.configNames
  - internal/gate/change.go#changeSince
  - internal/gate/change.go#change.checkout
  - internal/gate/change.go#configFor
  - internal/gate/kept.go#keptPath
  - internal/gate/kept.go#commandOf
  - internal/gate/kept.go#binaries
  - internal/gate/kept.go#keptBase
  - internal/gate/sarif.go#parse
  - internal/gate/sarif.go#fingerprint
  - internal/gate/sarif.go#lines.at
  - internal/gate/sarif.go#result.rest
  - internal/gate/sarif.go#identity
  - internal/gate/sarif.go#compareWith
  - internal/gate/sarif.go#writeCompared
checked: e2e165c2d
verified: agent:claude-code
---
# Gates — the findings a change adds

Part of [Gates](gates.md#on-a-merge-request-the-findings-the-change-adds).

On a merge request, a SARIF check's thresholds read only the findings the
change adds since the merge base
([ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md)):

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
  finding of each SARIF check there (but one `new: tool`), by its commit
  (`<commit>.json`); a change whose merge base it holds reads them
  instead of running the tool there. Only for the same command, `output`
  and `config`, no file the tool reads its config from changed by the
  change, and the same tool: the version its SARIF names, the hash of
  the binary found on the PATH; one naming neither runs again. Else, or
  none kept for that commit, run again, the summary saying why. The
  folder is filled only from main's own runs, never from a store a
  branch could write
  ([the next base](../next-base.md)).

## Compared one for one

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

How a tool tells them itself, suppressions, errors, and `workline gate`
by hand: [how the new findings are told](gates-told.md).

*Not built yet*: a formatter's fixes applied, rather than its command
said; the count's trend; vendored and generated code skipped; CI's build
reused; a monorepo's changed folders only.
