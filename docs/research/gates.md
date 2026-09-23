# Gates — review, architecture, security, performance, tests

**Verdict.** Every check exists as a mature binary that outputs SARIF or JSON.
What is missing is a neutral `gates.yml` runner that assembles them and runs
the same on a laptop and in any CI. Deterministic checks decide; AI findings
count only when reproduced.

## Model to follow

The LLM finds, the rules decide: [Clausura](https://github.com/liuyanghejerry/Clausura)
(Rust, SARIF, exit codes, fail-closed on incomplete runs) and
[archfit](https://github.com/alexei-led/archfit) (Go, architecture fitness, AI may
only explain). Three outcomes, not two: pass, finding, or *the check itself
failed* ([Agent_Quality_Kit](https://github.com/arsen-ask-lx/Agent_Quality_Kit)
plants a defect to prove a check can fail).

## Review

- [reviewdog](https://github.com/reviewdog/reviewdog) (Go): posts any linter's findings on changed
  lines, GitHub, GitLab, Gitea, Bitbucket. The neutral reporting layer.
- PR-Agent: advice, not a gate.
- [revmux](https://github.com/umputun/revmux): Claude and Codex reviewers, versioned lenses, audit archive.
- The reviewer sees only the pushed branch, never the implementer's reasoning
  (vercel eve template); the reviewer cannot edit code (superpipelines).

## Architecture guardian

Deterministic, per language: ArchUnit (Java), dependency-cruiser (JS/TS),
import-linter (Python), deptrac (PHP), arch-go and go-arch-lint (Go), archfit.
With AI: Tgenz1213/ArchGuard (ADRs as embeddings, LLM judges a change).
Structurizr documents C4 models; it does not enforce them.

## Security

Deterministic: Semgrep, Trivy, OSV-Scanner, Gitleaks, TruffleHog, OWASP ZAP,
Nuclei — all CLI, SARIF or JSON.
Agent pentest, only against authorized staging:
- [Strix](https://github.com/usestrix/strix) (Apache-2.0): headless, exits non-zero on findings,
  PR-scoped quick mode, GitHub and GitLab.
- [Shannon](https://github.com/KeygraphHQ/shannon) (AGPL-3.0): "no exploit, no report", SARIF.
- pentest-ai: each exploit re-run by an oracle; only reproduced findings count.
- CAI and Cyber-AutoAgent are archived.

## Performance

k6 (thresholds fail the run), Gatling (assertions), Locust (needs a script),
Lighthouse CI (web budgets, slowing), hyperfine (CLI benchmarks),
Bencher (statistical thresholds, any CI).

## Test integrity

- Mutation testing: Stryker, PIT, mutmut, cargo-mutants, Infection, go-mutesting.
- [smixs/code-quality](https://github.com/smixs/code-quality): detects skipped tests, weakened
  assertions, mocks of the code under test.
- Shadow Score: acceptance tests written from the spec by another model family,
  hidden from the coder.
- ai-test-integrity-review: replays modified tests against the old code.

## Verdict as code

conftest / OPA: Rego rules over the JSON outputs, e.g. "no high finding and
p95 under 200 ms". qgate: baseline pinned to a commit so existing debt does not
block adoption. cupcake: OPA policies for agents.
