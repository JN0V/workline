# ADR-0049: The inspector judges the new findings of the project's own tools

- **Status:** accepted; steps 1 to 5 built ([the build plan](#build-plan))
- **Date:** 2026-10-18
- **Decided by:** the maintainer, on 2026-10-18: the direction a panel of
  four argued (users, principles, prior art, skeptic), and the three
  choices [inspector.md](../research/inspector.md#what-this-means-for-workline)
  left open
- **Builds on:** [ADR-0013] (what was judged is not asked again),
  [ADR-0020] (the reviewer finds, the engine verifies), [ADR-0036] (a
  status earned by written criteria), [ADR-0042] (what never merges
  alone), [ADR-0044] (the write token stays with the base's runs),
  [ADR-0046] (the auditor samples every role's acts), [ADR-0047] (findings
  open no conversation); principles 2, 4, 5, 6, 7, 11, 12, 13
- **Research:** [inspector.md](../research/inspector.md),
  [inspector-coverage.md](../research/inspector-coverage.md)
- **Settles:** the inspector's design ([#202]); the order of [#244]

## Context

- **The maintainer's question**: should the inspector make sure every
  language has a linter and a static analyser, suggest the right tool,
  install it, and lint with the AI where no tool exists?
- **Old debt hides the new finding**: on workline's `main`, golangci-lint
  2.15.0 with its defaults finds 356 issues (325 `errcheck`) in 9 s. One
  unchecked error planted, `--new-from-merge-base` reports only it.
- **Detection, suggestion and pinned installs exist** as data and tools,
  some under licences that limit them (CodeQL on open-source code only).
- **No product lints with an LLM on a pull request** where no rule
  exists: GitHub Code Quality's "AI findings" come after the merge and
  "cannot be set as a threshold". The others put the model beside the
  tools.
- **A gate decides by rules** ([gates.md](../spec/gates.md#where-ai-fits)).
- **The forges compare base and head** only where paid or public.

## Decision

### 1. Coverage per language, by kind

- **By kind, per language**: lint, types, dependencies, security; not
  "one linter plus one analyser". Languages from tracked files:
  extensions, shebangs, [Linguist](https://github.com/github-linguist/linguist)'s table.
- **An engine act, not the role's**: `workline doctor`, `workline init`
  and gardening report it. No judgement is needed (principle 4).
- **Reported by default; blocking is the project's choice**: the
  `linted` rule of [#244].
- **What the project runs counts**, SonarQube included ([#203]).

### 2. What is suggested, in this order

1. What the project already runs: its CI steps, pre-commit hooks, a
   Makefile `lint`, `npm run lint`.
2. The ecosystem's own tool: `pio check`, `cargo clippy`, `go vet`.
3. A table borrowed from
   [trunk-io/plugins](https://github.com/trunk-io/plugins) (MIT,
   credited), regenerated from it, never kept by hand (principle 13).

- **A project-dependent linter** (ESLint, clang-tidy) only when its
  config is present: trunk's `config_present`.
- **A licence that limits a tool is said with it**: CodeQL only on
  open-source code, else a paid licence; Semgrep's registry rules for
  the project's own use.

### 3. Installation

- **In CI**: pinned with a checksum (mise's aqua backend, or aqua), in
  user space, never `sudo`; a bumper keeps it current (Renovate,
  `pre-commit autoupdate`). Principle 11.
- **On a person's machine**: the engine proposes, the person says yes
  (`workline setup`).
- **A project file changed** (a linter's config, `package.json`,
  `gates:`) goes through a pull request a person accepts.
- **The project's own linter config is never overridden.**

### 4. No AI linting as a check

- **An AI reading code in place of a missing linter is not a check**: it
  is not deterministic, so the base-versus-head comparison breaks; it
  costs tokens on every push; it duplicates the reviewer.
  [gates.md](../spec/gates.md#where-ai-fits) settles it already.
- **Instead, a rule written once**: the inspector's AI may write a
  Semgrep or ast-grep rule, proven to fail on the defect and pass on the
  fix (`semgrep --test`, `ast-grep test`). A person accepts it; a tool
  runs it (principles 6, 7).
- **With no agent**, a language with no tool is listed "unchecked",
  never counted as a pass. A tool that did not run is an error, never a
  pass (principles 5, 12).

### 5. The inspector's judgement

- **Triage the new findings**: why one matters here, which look like
  false positives, in what order to fix. Never the count, never the
  verdict.
- **A recurring defect becomes a rule** (above).
- **A rule's level**: it proposes moving a rule between warn and block,
  or changing it, from how often people dismiss or suppress it.
- **Asked once**: a finding already triaged is not asked again
  ([ADR-0013]). The auditor samples the triage ([ADR-0046]).

### 6. Guards

- **New findings by a rerun at the merge base**, with the head's config
  and the same tool version on both sides. A tool's upgrade gets a pull
  request of its own.
- **Day one is quiet**: old debt never blocks. It is shown once as a
  count, then as a trend.
- **A suppression the change adds counts as a finding**: `nolint`,
  `noqa`, `eslint-disable`, `NOLINT`.
- **Vendored and generated code is skipped**: `.gitattributes`
  `linguist-vendored` and `linguist-generated`, `.pio/`,
  `node_modules/`, `vendor/`.
- **A tool needing a build reuses CI's build**; otherwise the tools that
  need none run, and what is lost is said.
- **Linters run in a job with no write token and no AI key**; the
  applying job posts ([ADR-0044], principle 6). A fork's pull request
  above all.
- **A formatter's fixes are applied**, not commented on. Many findings of
  one rule make one comment, within the reviewer's caps.
- **Monorepos**: each directory's own config, only the changed paths.
- **The reviewer is told what the linters cover**, so its AI stops
  spending there ([ADR-0020]).
- **The second run has a cap** on CI minutes.

### 7. Order

- **[#244]'s detection and doctor warning come first**, or with the
  inspector: with no linter declared, the inspector has nothing to read.
  Its blocking rule comes later.
- **SonarQube ([#203]) counts as coverage** where a project runs it.

### 8. The three open choices

- **(a) On GitHub, a check's annotations plus one summary comment**, not
  code scanning alerts: those are paid on private repositories and open
  a conversation each ([ADR-0047]). On GitLab, Code Quality.
- **(b) The base's findings from a rerun at the merge base**, through a
  tool's own "new only" mode when it has one (Semgrep's
  `--baseline-commit`; golangci-lint's, which reads changed lines only).
- **(c) The boundary with security ([#204])**: the inspector owns the
  SARIF comparison and the posting, for every tool. Security owns its
  scanners' recipes, its review lens, the whole-tree dependency schedule
  and DAST.
- **[ADR-0042]'s "never alone" list gains any pull request changing
  `gates:`.** It holds today: only the documentalist's docs merge with no
  yes (`notAllowed`, internal/engine/routine.go). ADR-0042 is to be
  amended when an agent's pull requests can.

## Build plan

1. [#244]'s detection and doctor warning. Built: the languages are told
   by trunk-io/plugins' file types, the table its tools are keyed by,
   with Linguist's `.gitattributes` overrides
   ([languages](../languages.md)).
2. The engine's base-versus-head comparison on a merge request;
   thresholds on new findings only. Built
   ([gates](../spec/gates.md#on-a-merge-request-the-findings-the-change-adds)):
   - **Rerun by default**; a tool's own mode is the project's choice
     (`new: tool`), golangci-lint's reading changed lines only.
   - **A finding's identity holds no line**: the tool's `fingerprints`,
     else its `partialFingerprints`, else the message and the line's
     text. `workline/v1` holds a line: not read.
   - **The cap, measured** on a copy of workline with golangci-lint
     2.15.0, one unchecked error planted: 1 new, 360 before it. With
     empty caches, 18.4 s at the head, 5.8 s at the merge base; with warm
     ones, 6.5 s and 7.1 s. The default, twice the head's run and a
     minute at least, leaves room for both.
3. Posting: annotations plus a summary on GitHub, Code Quality on GitLab.
   Built ([gates](../spec/gates.md#shown-on-the-merge-request),
   [gates in CI](../gates-in-ci.md)):
   - **A job of their own**: the gates run alone (`--gates-only`), with a
     read token and no secret; the judge reads their verdicts
     (`--gates-from`), the applying job shows them (`apply --gates`).
   - **A fingerprint with no line**: the identity of step 2, hashed with
     how many alike come before it, for GitLab's comparison.
   - **Many of one rule, one line** of the summary comment, as many rules
     as the reviewer's `findings-max`. Nothing is filtered by what the
     reviewer posted: what is shown stays the same with and without an
     agent; telling the reviewer what the linters cover is its own step.
   - **A formatter's fixes**: its command is said (`fix:`); applying them
     is left, a patch the gates' job would propose.
   - **On GitLab**, every job holds the project's variables: the gates'
     job unsets the agent's and the bot's tokens, which keeps them from
     the tools' environment only.
4. `roles/inspector`: the AI's triage, and runs with no agent. Built
   ([#574](https://github.com/JN0V/workline/pull/574),
   [the inspector](../../roles/inspector/README.md)): one call a push
   with new findings, each kept by its fingerprint, in a comment of its
   own, in the run apart given the gates' job's result. Measured on a
   copy of workline, an unchecked `os.Rename` and a deferred `os.Remove`
   planted: 2,275 tokens in, the defect first, the removal a likely false
   positive, the gate's block unchanged.
5. A golangci-lint gate on workline itself: [#202]'s Validation. Built
   ([#584](https://github.com/JN0V/workline/pull/584)), a new finding
   blocking.
6. [#203]: SonarQube read as a gate.

## Leaving beta

By [ADR-0036], once released:

- **Built**: steps 2 to 4 with their conformance cases, `pending.txt`
  honest.
- **Validated**: the maintainer runs [#202]'s Validation on workline and
  on a copy of a real repository with lint debt.
- **Measured**: the verdict and the findings posted are the same with
  and without an agent. The triage runs five times on a bench of planted
  findings, true defects and known false positives
  ([ADR-0014](0014-checked-is-earned-by-what-was-read.md)'s five runs):
  its false-positive calls are counted against the truth, recorded in
  docs/research/.

## Consequences

- The inspector is a role with judgement; coverage, suggestion and
  installation are engine acts (principle 2).
- [#244] moves ahead: its detection before the inspector.
- [gates.md](../spec/gates.md) says what is to be built on a merge
  request.
- Projects keep their tools and configs; workline adds none of its own.

[#202]: https://github.com/JN0V/workline/issues/202
[#203]: https://github.com/JN0V/workline/issues/203
[#204]: https://github.com/JN0V/workline/issues/204
[#244]: https://github.com/JN0V/workline/issues/244
[ADR-0013]: 0013-what-was-judged-is-not-asked-again.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0046]: 0046-the-auditor-re-checks-every-roles-acts.md
[ADR-0047]: 0047-findings-on-a-merge-request-open-no-conversation.md
