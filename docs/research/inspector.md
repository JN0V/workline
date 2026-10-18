# Inspector — how static analysis tells new findings (2026-10-18)

**Question.** The inspector ([#202](https://github.com/JN0V/workline/issues/202))
keeps, on a merge request, only the findings the change adds. How do
SARIF, the forges and the linters themselves tell a new finding from an
old one, where do the base's findings come from, and what has AI triage
of findings been measured to do?

**Verdict.**

- **Two families.** *Line filters* keep what sits on changed lines
  (golangci-lint, reviewdog, code scanning's pull request view).
  *Result matching* runs base and head and compares identities
  (Semgrep, Trunk, GitLab, the SARIF SDK). Only matching sees a finding
  the change causes on a line it did not touch.
- **SARIF names the states** (`baselineState`: new, unchanged, updated,
  absent) and leaves the identity to whoever compares: `fingerprints`
  are the comparer's, `partialFingerprints` the tool's hints, never a
  line number.
- **GitHub compares for you only with code scanning**: public
  repositories, or an organisation paying for GitHub Code Security.
  Check-run annotations work everywhere, but compare nothing.
- **GitLab compares Code Quality by `fingerprint` on every tier**,
  against the target branch's pipeline at the merge base; the findings
  beside the diff's lines are Ultimate's.
- **No vendor lets AI change the verdict**: it suggests, holds a
  comment, gives a score. The one study counting real bugs lost found
  the best method dismissing 12 to 25% of them.

Already covered elsewhere, not repeated here: reviewdog's filter modes
([code-review.md](code-review.md#outputs-caps-who-merges)), code
scanning's upload, `partialFingerprints` and limits, GitLab Code
Quality's format ([server-side.md](server-side.md#reporting-to-the-forge-2026-09)),
code scanning's conversations and the Checks API's 50 annotations a call
([ci-and-forge.md](ci-and-forge.md#findings-and-conversations-to-resolve-2026-10-15)),
SonarQube's Web API ([roles-panorama.md](roles-panorama.md#teams-with-sonarqube-read-it-never-rerun-it)).

Searched in the ecosystem's words: *baselineState*, *result matching*,
*diff-aware scan*, *hold the line*, *new from merge base*, *baseline
pipeline*, *bulk suppressions*, *noise filtering*, *false positive
detection*, *incremental analysis*. Pages, API descriptions and source
code read on 2026-10-18; "inferred" marks what no page states.

## SARIF: one finding across runs

- **`baselineState`** ([SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/errata01/os/sarif-v2.1.0-errata01-os-complete.html),
  §3.27.24): `new` "detected in the current run but not … in the
  baseline run"; `unchanged`; `updated` "changed … in a way that the tool
  considers significant"; `absent` "detected in the baseline run but …
  not in the current run". Present on one result, it "SHALL be present
  on every" result of the run: a full comparison, nothing left
  undetermined.
- **`fingerprints`** (§3.27.16): a stable identity; "A direct SARIF
  producer SHOULD NOT populate this property", a result management
  system does. **`partialFingerprints`** (§3.27.17): what the tool adds
  to it; the comparer combines them as it likes.
- **Appendix B**: the comparer uses the tool, the rule and the path, and
  "SHOULD NOT include an absolute line number"; a fingerprint "only needs
  to be stable enough".
- **GitHub reads one key**: "Code scanning only uses the
  `primaryLocationLineHash`" ([SARIF support](https://docs.github.com/en/code-security/reference/code-scanning/sarif-files/sarif-support-for-code-scanning));
  `baselineState` is not among the properties it reads. Without
  `upload-sarif`, "users may see duplicate alerts".
- **How `upload-sarif` fills it** ([fingerprints.ts](https://github.com/github/codeql-action/blob/048d0ea295b6784a80010b29fd3af3ee29461dcd/src/fingerprints.ts)):
  a rolling hash of the 100 characters, spaces and tabs skipped, from the
  start of the result's first line, line endings made `\n`; `:n`
  numbers identical hashes within the file; a result with no line takes
  line 1. Inferred: a moved line keeps its alert; an edit to that line,
  or to the next lines within those 100 characters, makes a new one.

| Writes | Key | Source |
|---|---|---|
| CodeQL | `partialFingerprints.primaryLocationLineHash` | fingerprints.ts above |
| Semgrep | `fingerprints.matchBasedId/v1`: path, rule, pattern with its metavariables, hashed, an index appended | [Sarif_output.ml](https://github.com/semgrep/semgrep/blob/c712f3f57613dce1ea85403c1d39f53501b57646/src/osemgrep/reporting/Sarif_output.ml), [duplicates](https://docs.semgrep.dev/semgrep-code/remove-duplicates) |
| gitleaks | `partialFingerprints`: commit, email, author, date, message | [sarif.go](https://github.com/gitleaks/gitleaks/blob/83e99bacf166f92c562c8da0ca8ef12a5eccd49e/report/sarif.go) |
| golangci-lint, Ruff | none | [sarif.go](https://github.com/golangci/golangci-lint/blob/f8514979dbc8f9e5192530f88f9a24351eec4614/pkg/printers/sarif.go), [sarif.rs](https://github.com/astral-sh/ruff/blob/aad0e909ef1390f4b2a3ba8aa0a67fb8ea5cbacd/crates/ruff_linter/src/message/sarif.rs) |
| workline | `partialFingerprints.workline/v1`: the rule and `where`, which may hold `:<line>` | [report.go](../../internal/report/report.go) |

Comparers that write the states: the SARIF SDK's `match-results-forward`
("New, Absent, and Unchanged"; matchers by full fingerprint, identical
result, partial fingerprint, context region:
[usage](https://github.com/microsoft/sarif-sdk/blob/main/docs/multitool-usage.md),
[matchers](https://github.com/microsoft/sarif-sdk/tree/main/src/Sarif/Baseline/ResultMatching));
Qodana's baseline, a SARIF file giving new, unchanged and absent
([baseline](https://www.jetbrains.com/help/qodana/baseline.html)).

## GitHub: code scanning, its rule, annotations

| What | Found | Source |
|---|---|---|
| On a pull request | an alert shows only if "all the lines of code identified by the alert exist in the pull request diff", added or edited, not deleted; by default for pull requests targeting the default branch | [SARIF support](https://docs.github.com/en/code-security/reference/code-scanning/sarif-files/sarif-support-for-code-scanning), [alerts](https://docs.github.com/en/code-security/concepts/code-scanning/code-scanning-alerts) |
| The base | `upload-sarif` sends `base_ref` and `base_sha`: the merge base when it uploads for the merge commit, else the pull request's base | [upload-lib.ts](https://github.com/github/codeql-action/blob/96501a33aca65626acc8abecc8828308ea02e5d0/src/upload-lib.ts) |
| The check | "Code scanning results" fails on `error`, `critical` or `high`, settable | [triage in pull requests](https://docs.github.com/en/code-security/how-tos/manage-security-alerts/manage-code-scanning-alerts/triaging-code-scanning-alerts-in-pull-requests) |
| The rule | "Require code scanning results": a required tool's alert at the threshold (Alerts: None, Errors, Errors and Warnings, All; Security alerts: None to All), its analysis in progress, or the tool not configured; not merge queues; "not related to status checks" | [merge protection](https://docs.github.com/en/code-security/concepts/code-scanning/merge-protection), [rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) |
| Who has it | public repositories; organisation repositories on Team or Enterprise with GitHub Code Security, "$30 per month per active committer"; inferred: none on a user's private repository | [availability](https://github.com/github/docs/blob/main/data/reusables/gated-features/code-scanning.md), [2025-04-01](https://github.blog/changelog/2025-04-01-github-advanced-security-is-here-for-github-team-organizations) |
| CodeQL on pull requests | "only report new alerts found within the changed code (the diff range)"; the base's database kept in the Actions cache (`codeql-overlay-base-database`, 7.5 GB uncompressed at most) | [2025-05-28](https://github.blog/changelog/2025-05-28-incremental-security-analysis-makes-codeql-up-to-20-faster-in-pull-requests/), [2026-03-24](https://github.blog/changelog/2026-03-24-faster-incremental-analysis-with-codeql-in-pull-requests/), [caching.ts](https://github.com/github/codeql-action/blob/2a3e7563aa8643435135b77a9521565f20a0625b/src/overlay/caching.ts) |
| GitHub Code Quality | GA 2026-07-20, Team and Enterprise Cloud, $10 per active committer; CodeQL rules and AI; findings as inline comments; "Require code quality results" in rulesets; other tools' SARIF not mentioned | [GA](https://github.blog/changelog/2026-07-20-github-code-quality-is-now-generally-available/), [thresholds](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/set-pr-thresholds) |
| Annotations | any repository and plan, a GitHub App's token; shown in Checks and Files changed; title 255 characters, details 64 KB; a suite keeps 1,000 check runs of a name | [check runs](https://docs.github.com/en/rest/checks/runs#create-a-check-run) |

## GitLab: Code Quality, SAST, approval policies

| What | Found | Source |
|---|---|---|
| Code Quality by tier | merge request report: Free; pipeline report: Premium; diff annotations, project summary: Ultimate; CodeClimate-based scanning deprecated in 17.3, removal planned in 19.0 | [Code Quality](https://docs.gitlab.com/ci/testing/code_quality/) |
| Its comparison | new = the head's fingerprints the base lacks, resolved the reverse; no base report, none computed | [comparer](https://gitlab.com/gitlab-org/gitlab/-/blob/c6845f9d96001023abaf024429c80ebccf7cad50/lib/gitlab/ci/reports/codequality_reports_comparer.rb) |
| The baseline | the target's tip (merged results), else the merge base, else the diff's start; that SHA's latest pipeline, "in any state"; "Does not check whether the baseline pipeline has report artifacts" | [baseline selection](https://docs.gitlab.com/ci/testing/#baseline-pipeline-selection) |
| SAST | the JSON report on every tier; "New findings in merge request reports" and in the changes view: Ultimate | [SAST](https://docs.gitlab.com/user/application_security/sast/) |
| Approval policies | Ultimate; "newly detected" against the common ancestor, pre-existing against the default branch's tip; no results on the target: "any detected vulnerabilities are reported as new"; a rule that cannot be read requires approval by default | [policies](https://docs.gitlab.com/user/application_security/policies/merge_request_approval_policies/) |

## Linters' own "new only" modes

| Tool | How it tells | Caveat | Source |
|---|---|---|---|
| golangci-lint | `--new-from-merge-base`, `--new-from-rev`, `--new-from-patch` "work by comparing `git diff` output and issues"; `--whole-files` widens to changed files | "If an issue is not reported as the same line as the changes then the issue will be skipped" | [FAQ](https://golangci-lint.run/docs/welcome/faq/), [options](https://golangci-lint.run/docs/configuration/file/) |
| Semgrep | `--baseline-commit`, and `semgrep ci` on a pull request: runs again on the baseline, in a git worktree, only the files and rules the head matched; drops head matches equal in rule, path (renames followed) and matched lines, counted one for one | "Cross-file analysis is not supported for diff-aware scans" | [CLI](https://docs.semgrep.dev/cli-reference), [CI jobs](https://docs.semgrep.dev/deployment/customize-ci-jobs), [Diff_scan.ml](https://github.com/semgrep/semgrep/blob/31729a1719c6e76ac8d41ec8fb9ab191621ee10f/src/osemgrep/cli_scan/Diff_scan.ml) |
| ESLint | bulk suppressions (v9.24.0, 2025-04-04): `eslint-suppressions.json` holds "the number of violations for each rule and file", `error` rules only; `--prune-suppressions` | "if more violations are found for the same rule in the same file, ESLint will report **all** of them" | [suppressions](https://eslint.org/docs/latest/use/suppressions), [blog](https://eslint.org/blog/2025/04/introducing-bulk-suppressions/) |
| Ruff | none: `--add-noqa` or `--add-ignore` writes a comment on each violating line | baselines asked in [ruff#1149](https://github.com/astral-sh/ruff/issues/1149), open | [linter](https://github.com/astral-sh/ruff/blob/main/docs/linter.md) |
| Trunk | hold-the-line: each modified file linted twice, the upstream in a "shadow" tree, cached, both "using your current linter config"; else a filter on modified lines | the post is of 2022-03-07; today's docs say it "only lints your git diffs" | [blog](https://trunk.io/blog/trunk-check-hold-the-line), [docs](https://docs.trunk.io/code-quality/overview/deal-with-existing-issues) |

## AI triage, measured

| Tool or study | What it does, measured | Source |
|---|---|---|
| Copilot Autofix | fixes, not triage; beta May to July 2024, median 28 minutes against 1.5 hours by hand; a harness of "over 2,300 alerts" whose fixes are merged unedited, then rescanned and tested | [blog](https://github.blog/news-insights/product-news/secure-code-more-than-three-times-faster-with-copilot-autofix/), [responsible use](https://github.com/github/docs/blob/main/content/code-security/responsible-use/security-and-quality-ai-features.md) |
| Semgrep Multimodal (was Assistant) | suggests "whether a finding can safely be ignored"; noise filtering holds a likely false positive's comment, a person agrees or reopens. To 2025-08-21: 3,500+ customers, 6.5M+ findings, 96% human-agree, 60% of findings filtered on average; internally 96% right when it says false positive, which "does not mean that Multimodal catches all false positives" | [overview](https://docs.semgrep.dev/semgrep-assistant/overview), [metrics](https://docs.semgrep.dev/semgrep-multimodal/metrics) |
| SonarQube AI CodeFix | a fix for one issue on demand; Server's Enterprise and Data Center; no triage, no number | [AI CodeFix](https://docs.sonarsource.com/agent-centric-development-cycle/in-your-long-living-branches-the-code-maintenance-loop/ai-codefix.md) |
| GitLab false positive detection | Ultimate with Duo; Critical and High SAST on the default branch; a score (80–100 likely false positive, 60–79 possible, under 60 likely not); a person dismisses; no accuracy published | [detection](https://docs.gitlab.com/user/application_security/vulnerabilities/false_positive_detection/) |
| Du et al., 2026-01 (Tencent) | 433 alarms, 328 false; LLM plus static analysis removed 94–98% of false ones, recall on real bugs 0.75–0.88 (about 13 to 26 of 105 lost); plain prompting removed 26–52%; recall "below the enterprise-expected threshold of 90%"; 2.1 to 109.5 s an alarm against 10 to 20 minutes by hand | [arXiv:2601.18844](https://arxiv.org/abs/2601.18844) |
| Xiong and Zhang, 2026-01 | agents on SAST alarms: OWASP Benchmark's false rate from over 92% to 6.3% at best; post-cutoff C/C++, 95.5% of false ones found at 95.5% precision, 36.4% by plain prompting; aggressive filtering hides true vulnerabilities | [arXiv:2601.22952](https://arxiv.org/abs/2601.22952) |

## Where the base's findings come from

| Source | Cost | Staleness, failure |
|---|---|---|
| Rerun at the merge base (Semgrep, Trunk) | a second run, limited to changed files; a checkout of the merge base; a build for compiled analyses | none: same tool, config and advisory data both sides; fails if the base does not build |
| Code scanning's stored analysis | free on public repositories, else Code Security; `GET …/code-scanning/analyses/{id}` as `application/sarif+json` gives "a subset of the analysis data that was uploaded" ([REST](https://docs.github.com/en/rest/code-scanning/code-scanning)) | only if the base branch's own runs upload |
| Actions artifact | 90 days kept by default; another run's needs a token with `actions: read` and its `run-id` ([artifacts](https://docs.github.com/en/actions/concepts/workflows-and-actions/workflow-artifacts), [download-artifact](https://github.com/actions/download-artifact)) | the run's commit, tool and data |
| Actions cache | 10 GB a repository, dropped after 7 days unused; a pull request restores the base branch's caches, writes only its own ([caching](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching)) | as artifacts, plus eviction |
| GitLab artifacts | 30 days on GitLab.com unless set; each ref's latest and latest successful pipeline kept ([artifacts](https://docs.gitlab.com/ci/jobs/job_artifacts/), [GitLab.com](https://docs.gitlab.com/user/gitlab_com/)) | picked at the merge base, its artifacts not checked |

Inferred: a stored report made before a new advisory makes osv-scanner
count an unchanged dependency against the change; a linter upgraded in
the change makes every finding of a new rule "new", where a rerun with
the head's config counts none on old code.

## What this means for workline

Findings, not decisions: the maintainer took each in
[ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md).
Which tools a project needs, and AI in their place:
[inspector-coverage.md](inspector-coverage.md).

**(a) Annotations or code scanning alerts, on GitHub.**

- [#202](https://github.com/JN0V/workline/issues/202) asks for findings
  "where the reviewer's go"; since [ADR-0047](../adr/0047-findings-on-a-merge-request-open-no-conversation.md)
  that is no longer code scanning on a merge request.
- *Annotations* of a `workline inspector` check: any plan, no
  conversation, the check's conclusion as the required status. workline computes
  "new" itself, and nothing outlives the run: no dismissal, no history.
- *Code scanning alerts*: GitHub compares base and head, keeps the
  history, has a merge rule; but only on public repositories or with
  Code Security, one conversation per new alert, only its own line hash
  read, and a base analysis needed from the base branch's runs, which
  the templates do not upload today.
- *Both*: annotations decide, alerts kept for security findings, the
  split ADR-0047 makes today.

**(b) Rerun at the merge base, or the base's stored report.**

- *Rerun*: right by construction, any forge, nothing stored; one more
  run and a second checkout on every merge request.
- *Stored report*: no second run; it needs a run on each base push, a
  store with its own expiry, and a "no report" that is loud (principle
  12), where GitLab's Code Quality compares nothing and its policies
  call everything new.
- *Either way*, matching needs an identity without a line: the tools'
  own when they write one (Semgrep), else one computed like
  `primaryLocationLineHash`. workline's `workline/v1` holds a line when
  `where` does, which GitLab's comparison reads.

**(c) The boundary with the security role ([#204](https://github.com/JN0V/workline/issues/204)).**

- Semgrep, CodeQL, osv-scanner and zizmor sit in both issues.
- *By mechanism*: the inspector owns new-only matching, posting and
  triage for every SARIF check; security owns which checks, its reviewer
  lens and DAST. One comparison, no duplicate.
- *By tool*: lint to the inspector, scanners to security; two owners
  of the same machinery.
- *By kind of finding*: a dependency's advisory can appear with no
  change, and GitLab compares pre-existing ones to the default branch's
  tip: security may need the whole tree on a schedule, where the
  inspector needs only what the change adds.
