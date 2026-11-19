---
sources:
  - .github/workflows/workline.yml#on
  - .github/workflows/workline.yml#env
  - .github/workflows/workline.yml#jobs.judge.permissions
  - .github/workflows/workline.yml#jobs.pinned
  - .github/workflows/workline.yml#jobs.head-gates.name
  - .github/workflows/workline.yml#jobs.head-judge.name
  - .github/workflows/workline-warn.yml#name
  - .github/actions/engine/action.yml#name
  - .github/actions/tests/action.yml
  - .github/workflows/workline-apply.yml#env
  - .github/workflows/workline-develop.yml#env
  - .github/workflows/workline-answer.yml#env
  - .github/workflows/workline-bounds.yml#env
  - .github/workflows/workline-merge.yml#env
  - .github/workflows/workline-gardening.yml#env
  - .github/workflows/workline-sample.yml#env
  - .github/workflows/workline-loop.yml#env
  - .github/workflows/workline-develop.yml#on
  - .github/workflows/workline-answer.yml#on
  - .github/workflows/workline-merge.yml#on
  - .github/workflows/workline-gardening.yml#on
  - .github/workflows/workline-sample.yml#on
  - .github/workflows/test.yml#name
  - .github/workflows/test.yml#jobs.rest
  - .workline/config.yaml#routing.timed
  - .github/workflows/release-please.yml#name
  - release-please-config.json
  - .github/workflows/release.yml#name
  - .goreleaser.yaml#release
checked: 3bab32400
verified: agent:claude-code
judged: 220b6dc
---
# workline's own workflows

Part of [workline on GitHub Actions](../ci-github.md): how workline's own .github/workflows/ differ from the templates.

workline's .github/workflows/ are the templates, but for a gate, the
engine and their triggers
([ADR-0071](../adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md#1-the-line-runs-a-pinned-release)).

## The engine they run

- `gates` installs golangci-lint, pinned by its SHA-256, for
  [the gate](../../.workline/config.yaml);
- **the pinned release**: every job with a write token (develop, answer,
  apply, bounds, merge, gardening, sample), the loop, and gates and
  judge, run the release `WORKLINE_VERSION` names, as a user's line
  does: a defect on `main` never runs the merge of its own fix;
  - downloaded by [the engine's action](../../.github/actions/engine/action.yml),
    its checksum checked as the templates do; it must say that version,
    or the job fails;
  - one value, in each of those workflows' `env:`; no
    `x-release-please-version` marker: release-please never moves it;
  - moved only in its own pull request, after its try on a copy, on the
    person's label;
- **the head's engine**: warn runs the engine built from the pull
  request's commit, once a commit, by the same action without a
  version; it must name the commit checked out, unmodified, or the job
  fails;
- **the head pass**: a pull request changing the engine
  ([its paths](../adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md#1-the-line-runs-a-pinned-release)) also runs gates and judge on its head's
  engine, in two checks named "… on the head's engine (non-blocking)":
  they never hold the line's verdict;
- **the N-1 rule**: the check "workline: the pinned release reads the
  head" runs the pinned release on the head's `.workline/config.yaml`
  and the front matter of the docs it touches, and fails naming what
  it cannot read.

## Their triggers and tests

- no `merge_group`: a user's repository has no merge queue;
- no cron where the loop dispatches the job
  ([`workline tick`](../tick.md#the-github-loop)): gardening, the sample,
  test.yml's seeds; the templates keep theirs, for a loop left a dry run;
- workline-develop.yml, workline-answer.yml and
  workline-objection.yml start on no event: the reconcile the loop
  dispatches starts each issue's step, review round and tester's
  answer due
  ([#1871](https://github.com/JN0V/workline/issues/1871)); the templates
  keep their events. workline-merge.yml keeps its label and issue
  events; the loop starts its hourly run;
- the tools the tests need beyond Go (gitleaks, golangci-lint) come
  from [one action](../../.github/actions/tests/action.yml), in test.yml
  and in each job where a role runs the tests: the developer's and the
  tester's proofs, the curator's measures;
- test.yml runs the tests alone, the line simulator in a job beside the
  rest; its `test` job, the check `main` requires, waits for both.
  main's whole-repository checks are
  `route main`'s, once a merge ([main after a merge](../main-after-merge.md));
  the links to other sites are gardening's, at night;
- the loop dispatches the nightly (02:17 UTC) and test.yml's seeds
  (03:41 UTC) at the times `routing.timed` gives them in
  [the config](../../.workline/config.yaml), the docs on Sunday, the
  backlog when Ready is below its target; run by hand, gardening's
  input `skip` leaves roles out (`documentalist`: the backlog alone);
- the templates' pins, which release-please moves, are never what
  workline's own checks run: a pull request moving them to a version not
  yet tagged stays green
  ([ADR-0017](../adr/0017-the-release-manager.md)); the line's own pin
  is a published release, tried
  ([ADR-0071](../adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md#1-the-line-runs-a-pinned-release)).

## How workline releases

With release-please, never a tag by hand
([ADR-0017](../adr/0017-the-release-manager.md)):

- on each push to `main`, release-please.yml, with the App's token, keeps
  one pull request open, "chore(main): release X": the next version from
  the conventional commits since the last tag (`feat` moves the minor
  while workline is at 0.x, as a breaking change does), [CHANGELOG.md](../../CHANGELOG.md), and
  the templates' `WORKLINE_VERSION`, each on a line marked
  `x-release-please-version`, in the files release-please-config.json
  lists;
- the App's token, not the job's: a pull request opened with
  `GITHUB_TOKEN` starts no workflow, so its checks would never run;
- on that pull request, `judge` holds it while an open issue is labelled
  `workline:hold-release` or main's tip is not green
  ([what holds a release](../release-holds.md)); release-please keeps it
  on `main` (`always-update`);
- the person merges it each Monday, sooner for a fix that touches a
  real user: that is the decision. release-please tags the
  merged commit `vX.Y.Z` and writes the release with its notes; in the
  same run, release.yml builds the binaries with GoReleaser, which uploads
  them and keeps the notes (`release.mode: keep-existing`), and pushes the
  image;
- a failed publish is run again from the Actions page, never tagged again:
  a tag pushed is never moved (the Go proxy keeps its first commit).
