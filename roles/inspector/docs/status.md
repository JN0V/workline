# Inspector — where it stands (2026-10-20)

**In one line:** on a merge request, triages the findings the project's
gates report the change adds: why each matters, a likely false positive
said with why, the order to fix them, in one comment, never the gates'
verdict. Off the verdict's path, in the run apart. Built as step 4 of
[ADR-0049's plan](../../../docs/adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#build-plan);
beta, not released yet.

A try is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)):
the first, on a copy of workline with golangci-lint and one real call,
on [#574](https://github.com/JN0V/workline/pull/574); the first on
workline's own pull requests, a finding planted, on
[#584](https://github.com/JN0V/workline/pull/584).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold, as
[ADR-0049](../../../docs/adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#leaving-beta)
sets them:

1. **Nothing buildable in Missing**: open. Steps 2 to 5 are built with
   their cases; Missing below holds what is left.
2. **The main issue validated on real use**: open.
   [#202](https://github.com/JN0V/workline/issues/202)'s Validation, on
   workline and on a copy of a real repository with lint debt. On
   workline, the golangci-lint gate is built (ADR-0049's step 5), tried
   once on a planted finding; the maintainer's run of the Validation
   there, and the copy, are left.
3. **AI verdicts measured**: open. The triage run five times on a bench
   of planted findings, true defects and known false positives, its
   false-positive calls counted against the truth, recorded in
   docs/research/. Not built yet.

## Built

Cases in
[tests/conformance/cases/inspector](../../../tests/conformance/cases/inspector).

- **Each new finding triaged, posted in its own comment**, the record
  hidden in it: `new-findings-triaged-and-posted`.
- **A likely false positive leaves the gates' verdict**, the finding
  still counting: `false-positive-leaves-the-gates-verdict`.
- **Triaged once, never asked again**, by fingerprint
  ([ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md)):
  `triaged-once-not-asked-again`; after a push, only the new finding
  asked: `new-finding-after-a-push-asked-alone`; on a machine, the
  record kept there: `on-a-machine-after-the-gates`; past
  `findings-max`, the rest asked at the next push:
  `findings-past-the-cap-asked-at-the-next-push`.
- **No agent, no call, said**: `no-agent-no-call-said`; a draft waits for
  ready: `draft-waits-for-ready`; no new finding, no call:
  `no-new-finding-no-call`.
- **Off the verdict's path**
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)):
  `verdicts-line-leaves-the-triage`; the gates' result read only for the
  commits judged: `gates-of-another-head-not-triaged`.
- **An answer that does not read**, asked again once, then said:
  `answer-not-read-said`.
- **In CI**, beside the lenses that warn: workline-warn.yml waits for
  workline.yml's gates job, `workline:warn` for `workline:gates`
  (`TestWarnApartSaysWhatIsMissing`, `TestWarnApartNeverHoldsTheVerdict`).
- **A golangci-lint gate on workline itself**, ADR-0049's step 5: its
  findings triaged on workline's own pull requests, the tool installed
  pinned, its checksum in the workflow (`TestOwnGatesToolsArePinned`).

## Missing

- **The weekly sample of the triage**: the auditor draws its acts
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md));
  each triage keeps its head and model for it, no kind drawn yet:
  [#575](https://github.com/JN0V/workline/issues/575).
- **A recurring defect proposed as a rule**, Semgrep or ast-grep, proven
  to fail on the defect and pass on the fix
  ([ADR-0049, section 4](../../../docs/adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#4-no-ai-linting-as-a-check)):
  [#576](https://github.com/JN0V/workline/issues/576).
- **A rule's level proposed** between warn and block, from how often
  people dismiss or suppress it
  ([ADR-0049, section 5](../../../docs/adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#5-the-inspectors-judgement)):
  [#577](https://github.com/JN0V/workline/issues/577).
- **The reviewer told what the linters cover**
  ([ADR-0049, section 6](../../../docs/adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#6-guards)):
  [#578](https://github.com/JN0V/workline/issues/578).
- **Two runs judged before either is applied**: the second does not see
  the first's record, and asks its findings again; the reviewer's apply
  merges its record for that ([#375](https://github.com/JN0V/workline/issues/375)):
  [#580](https://github.com/JN0V/workline/issues/580).
- **SonarQube read as a gate**, ADR-0049's step 6, its findings triaged
  as the gates' are: [#203](https://github.com/JN0V/workline/issues/203).
- **Gates per changed folder** in a repository of several projects, each
  folder's own config
  ([ADR-0049, section 6](../../../docs/adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#6-guards)):
  [#553](https://github.com/JN0V/workline/issues/553), its part 2.
