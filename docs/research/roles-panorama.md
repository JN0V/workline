# Roles panorama — which roles a line needs, for any project (2026-10-06)

**Question.** Beyond the committer, documentalist, reviewer and product
owner, which roles does a software line need — for every kind of project,
not workline's own — which of them are tools run as gates, and in what
order to build them?

**Verdict.** Most specialist roles of a real team are **tools first**:
static analysis, security, accessibility, performance, licences,
infrastructure and migration linters exist as pinned binaries with an exit
code or SARIF, and belong in gates (docs/spec/gates.md). A role is needed
only where judgement sits between the tools and a person: inspector,
security, auditor, process engineer, tester, architect, UX, PM, and the
developer last. The auditor and the process engineer stay two roles (a
quality audit is not industrial engineering); a team's SonarQube verdict
is read, never rerun.

Already covered elsewhere, not repeated here: the tools per kind of check
(gates.md), how AI reviewers find and verify (code-review.md), how
automation is measured in production (self-evaluation.md), the factories
with roles (landscape.md).

## Method

- Words searched: *agent roster*, *SDLC agents*, *quality gate*, *new code
  period*, *SARIF baseline*, *diff-aware analysis*, *DAST baseline*, *AI
  pentest*, *test architect*, *accessibility CI*, *migration linter*.
- Every repository below checked through the GitHub API on 2026-10-06
  (stars, licence, not archived, pushed in 2026); every claim quoted from
  its README or documentation that day.
- SonarQube's Web API read from SonarSource's own instance
  (next.sonarqube.com `api/webservices/list`), since the public docs point
  to each instance's `/web_api` page rather than list the endpoints.
- Dropped, not confirmed: deptrac (its GitHub repository now redirects to
  an archived fork); Strix on GitLab (only a GitHub Actions example found).

## Roles in real teams and in agent methods

The [BMAD method](https://docs.bmad-method.org/reference/skills-and-agents/)
ships five agents today: analyst, product manager, architect, developer,
UX designer; its test architect and tech writer are listed as on hiatus.
Real teams add the roles below. "Here" is what workline does with each.

| Role in a team | Its job | Here |
|---|---|---|
| Scrum master | runs the process, removes blockers | ≈ the product owner (ADR-0018): order, refine, ask, report |
| Product owner | the backlog: refined, ordered, accepted | built, beta |
| Product manager | the product as a whole: what exists, what neighbours do, what to add or reorganise | **PM**, later and wider than the product owner: a functional map of the existing product, a watch of similar and adjacent products, functional suggestions and reorganisation; to design with the maintainer |
| Analyst | research, product brief | part of the PM; research before building is already a rule (principle 13) |
| Architect | structure, dependencies, decisions | **architect**: spec review (#128) plus architecture rules as gates |
| UX designer | flows, screens, accessibility | **UX**, for projects with a user interface: accessibility tools as gates, a role for what tools cannot judge |
| Developer | builds | **developer**, last (#117) |
| QA, test architect | tests from the need, test strategy | **tester**: tests written from an issue's Verification, red first, hidden from the developer |
| Code reviewer | reads the change | **reviewer**, beta (ADR-0020) |
| Static analysis owner | linters, quality gate | **inspector** |
| Security engineer, pentester | SAST, dependencies, pentest | **security**: gates now, a reviewer lens, DAST later |
| Tech writer | docs | **documentalist**, built |
| Release manager | versions, notes | the release manager's step (ADR-0017) |
| DBA | schema, migrations | a reviewer lens plus migration linters as gates |
| Performance, SRE | budgets, load | gates (k6, Lighthouse CI, hyperfine) |
| Legal, licences | licence compliance | gates (go-licenses, licensee, Trivy's licence scan) |
| Platform, IaC | infrastructure code | gates (Checkov, Trivy config, hadolint) |
| Quality auditor | re-checks that the work was done as claimed | **auditor**, widened to every role |
| Industrial / process engineer | studies the line, proposes changes | **process engineer** (ADR-0019, #89) |

## Auditor and process engineer: two roles

In a factory, the **quality audit** re-checks a sample of finished work
against the standard, independently of whoever did it, and gives a verdict
per item; **industrial engineering** reads the line's measures and
proposes how to change the line. Kept apart, because an evaluator that
also changes what it evaluates can game its own measure (self-evaluation.md:
the Darwin Gödel Machine erasing its markers).

| | Auditor | Process engineer |
|---|---|---|
| Reads | a sample of each role's acts: docs vouched for, the product owner's closings, splits and milestones, the reviewer's findings kept and dropped, the committer's rewrites | the measures: the auditor's verdicts, the evaluation's results, the engine's run records, what people did after each act |
| Against | the code at that act's commit | counts over a window |
| Gives | a verdict per act, with its proof | proposals, each an issue (ADR-0019) |
| Independence | the best level available from the model that acted (ADR-0005) | the best level available from the models that ran |
| Never | proposes a change to the line | changes the line itself |

Today the auditor reads only docs (ADR-0015); the product owner's acts are
drawn for a person, with no agent (ADR-0033).

## Inspector: static analysis on a merge request

Gates already run any tool and read `exit` or `sarif` against thresholds
set in advance (docs/spec/gates.md). What a team expects of static
analysis on a merge request, and is missing:

- **Only new findings.** A finding the base branch already had is not the
  change's. [reviewdog](https://github.com/reviewdog/reviewdog) (9.6k★,
  MIT) filters to "added/modified lines" by default and reads SARIF 2.1.0;
  GitHub code scanning matches results across commits by SARIF
  `partialFingerprints` and shows on a pull request only alerts on lines
  it adds or edits
  ([docs](https://docs.github.com/en/code-security/code-scanning/integrating-with-code-scanning/sarif-support-for-code-scanning)).
  To borrow: fingerprints compared between base and head, not line numbers.
- **Posted where people read**: reviewdog posts on GitHub and in GitLab
  merge request discussions; workline's reviewer already writes SARIF and
  GitLab Code Quality.
- **An AI that explains and triages, never changes the verdict**: why a
  finding matters here, which are likely false positives, in what order to
  fix; the gate's numbers decide (principle 4; Clausura and archfit in
  gates.md).
- **Pinned tools per language**:

| Language or file | Tools (all verified, active in 2026) |
|---|---|
| Go | [golangci-lint](https://github.com/golangci/golangci-lint), [govulncheck](https://github.com/golang/vuln) |
| JavaScript, TypeScript | [ESLint](https://github.com/eslint/eslint) |
| Python | [Ruff](https://github.com/astral-sh/ruff) |
| Many languages | [Semgrep](https://github.com/semgrep/semgrep), [CodeQL](https://github.com/github/codeql-action) |
| Dependencies | [osv-scanner](https://github.com/google/osv-scanner), [Trivy](https://github.com/aquasecurity/trivy) |
| GitHub Actions | [actionlint](https://github.com/rhysd/actionlint), [zizmor](https://github.com/zizmorcore/zizmor) |
| SQL | [SQLFluff](https://github.com/sqlfluff/sqlfluff) |

### Teams with SonarQube: read it, never rerun it

A team that runs SonarQube already has its quality gate and its issues,
computed on its own new-code definition. Rerunning the analysis would
duplicate cost and give a second, diverging verdict. SonarQube's Web API
(as listed by [SonarSource's own instance](https://next.sonarqube.com/sonarqube/web_api/api/qualitygates/project_status))
gives both, scoped to a merge request:

- `api/qualitygates/project_status` takes `projectKey`, `branch` or
  `pullRequest`, `analysisId`: the gate's status and its conditions;
- `api/issues/search` takes `components`, `branch` or `pullRequest`,
  `inNewCodePeriod`, `issueStatuses`, `impactSeverities`: the issues
  themselves;
- authentication by a bearer token
  ([Web API](https://docs.sonarsource.com/sonarqube-server/extension-guide/web-api.md)).

To borrow: a `sonar` output kind beside `exit` and `sarif`, whose
verdict is SonarQube's own gate; a missing analysis for that commit is an
error, never a pass (principle 12).

## Security

- **Now, as gates**: secrets ([gitleaks](https://github.com/gitleaks/gitleaks),
  already in the committer), dependencies (osv-scanner, govulncheck), SAST
  (Semgrep, CodeQL), SBOM and its vulnerabilities
  ([Syft](https://github.com/anchore/syft) then
  [Grype](https://github.com/anchore/grype), or Trivy), CI workflows
  (zizmor). All are single binaries with SARIF or JSON.
- **A `security` lens of the reviewer**: what scanners miss in the change
  itself (an authorisation check removed, input reaching a query), through
  the reviewer's quotes and judge (ADR-0020).
- **Pentest, later**: DAST on a release or a schedule, **only against an
  authorised staging environment**, never production.
  [OWASP ZAP's baseline scan](https://www.zaproxy.org/docs/docker/baseline-scan/)
  spiders for a minute and scans passively, "doesn't perform any actual
  attacks", exits 1 on a FAIL; [Nuclei](https://github.com/projectdiscovery/nuclei)
  runs templates. AI pentest tools, verified:
  [Strix](https://github.com/usestrix/strix) (66.7k★, Apache-2.0)
  "exits with non-zero code when vulnerabilities are found", scopes pull
  requests to changed files, a proof of concept per finding;
  [Shannon](https://github.com/KeygraphHQ/shannon) (48.6k★, AGPL-3.0):
  white-box, "No exploit, no report", SARIF, not for production.
- **An agent's finding counts only once its exploit replays** in a
  deterministic step (docs/spec/gates.md, "Where AI fits").

## UX and accessibility

For projects with a user interface. Tools as gates:
[axe-core](https://github.com/dequelabs/axe-core) (MPL-2.0),
[pa11y](https://github.com/pa11y/pa11y),
[Lighthouse CI](https://github.com/GoogleChrome/lighthouse-ci), whose
`lhci assert` exits non-zero on an `error` assertion and can assert the
accessibility score
([configuration](https://github.com/GoogleChrome/lighthouse-ci/blob/main/docs/configuration.md)).
The role judges what tools cannot: a flow against the need, consistency
between screens, wording.

## Architect, tester, the rest as gates

| Need | What to borrow |
|---|---|
| Architect | spec review before building (#128, the reviewer's spec subject); rules as gates: [ArchUnit](https://github.com/TNG/ArchUnit) (Java), [dependency-cruiser](https://github.com/sverweij/dependency-cruiser) (JS, TS), [import-linter](https://github.com/seddonym/import-linter) (Python), [go-arch-lint](https://github.com/fe3dback/go-arch-lint) (Go) |
| Tester | tests from the issue's Verification before the code, red first, kept from the developer (Shadow Score in gates.md); mutation testing as a gate |
| Performance | [k6](https://github.com/grafana/k6) thresholds, Lighthouse CI budgets |
| Licences | [go-licenses](https://github.com/google/go-licenses), [licensee](https://github.com/licensee/licensee), Trivy |
| Infrastructure as code | [Checkov](https://github.com/bridgecrewio/checkov), Trivy's config scan (tfsec is now part of Trivy), [hadolint](https://github.com/hadolint/hadolint) |
| Database | a reviewer lens; migration linters as gates: [squawk](https://github.com/sbdchd/squawk) (Postgres), [strong_migrations](https://github.com/ankane/strong_migrations) (Rails), [Atlas](https://github.com/ariga/atlas) lint |
| Verdict over several outputs | [conftest](https://github.com/open-policy-agent/conftest) |

## Roadmap, decided by the maintainer

1. Reviewer to block-ready ([#90](https://github.com/JN0V/workline/issues/90), [#126](https://github.com/JN0V/workline/issues/126)).
2. Inspector ([#202](https://github.com/JN0V/workline/issues/202)); SonarQube read as a gate ([#203](https://github.com/JN0V/workline/issues/203)).
3. Security ([#204](https://github.com/JN0V/workline/issues/204)).
4. Auditor widened to every role ([#205](https://github.com/JN0V/workline/issues/205)).
5. Process engineer ([#89](https://github.com/JN0V/workline/issues/89)).
6. Tester ([#206](https://github.com/JN0V/workline/issues/206)).
7. Architect ([#207](https://github.com/JN0V/workline/issues/207)).
8. UX ([#208](https://github.com/JN0V/workline/issues/208)).
9. PM ([#209](https://github.com/JN0V/workline/issues/209)), a draft to design with the maintainer.
10. Developer ([#117](https://github.com/JN0V/workline/issues/117)), **last**: its pull requests need every check above.
