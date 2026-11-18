---
sources: [defaults/ci/github, .github/workflows/workline.yml, .github/workflows/workline-apply.yml, .github/workflows/workline-gardening.yml, .github/workflows/workline-sample.yml, .github/workflows/workline-answer.yml, .github/workflows/workline-develop.yml, .github/workflows/workline-merge.yml, .github/workflows/workline-bounds.yml, .github/workflows/workline-warn.yml, .github/actions/engine/action.yml, .github/actions/tests/action.yml, .github/workflows/release.yml, .github/workflows/release-please.yml, release-please-config.json, .goreleaser.yaml, cmd/workline]
checked: ca55e23d
verified: agent:claude-code
judged: 220b6dc
---
# workline's own workflows

Part of [workline on GitHub Actions](../ci-github.md): how workline's own .github/workflows/ differ from the templates.

workline's .github/workflows/ are the templates, but for a gate and the
engine ([ADR-0071](../adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md#1-the-line-runs-a-pinned-release)):

- `gates` installs golangci-lint, pinned by its SHA-256, for
  [the gate](../../.workline/config.yaml);
- **the pinned release**: every job with a write token (develop, answer,
  apply, bounds, merge, gardening, sample), and gates and judge, run the
  release `WORKLINE_VERSION` names, as a user's line does: a defect on
  `main` never runs the merge of its own fix;
  - downloaded by [the engine's action](../../.github/actions/engine/action.yml),
    its checksum checked as the templates do; it must say that version,
    or the job fails;
  - one value, in each of those workflows' `env:`; no
    `x-release-please-version` marker: release-please never moves it;
  - moved only in its own pull request, after its try on a copy, on the
    person's label;
- **the head's engine**: warn, and test's `go test`, run the pull
  request's commit, built once a commit by the same action without a
  version; it must name the commit checked out, unmodified, or the job
  fails;
- **the head pass**: a pull request changing `cmd/`, `internal/`,
  `roles/`, `skills/`, `roles.go`, `defaults/routing.yaml`, `go.mod` or
  `go.sum` also runs gates, `route main` and judge on its head's engine,
  in two checks named "… on the head's engine (non-blocking)": they never
  hold the line's verdict;
- **the N-1 rule**: the check "workline: the pinned release reads the
  head" runs the pinned release on the head's `.workline/config.yaml`
  and the front matter of the docs it touches, and fails naming the file
  it cannot read;
- no `merge_group`: a user's repository has no merge queue;
- a job [`workline tick`](../tick.md#the-github-loop) lists loses its
  cron, one workflow at a time: the loop dispatches it, the loop's
  hourly schedule the restart;
  [the triggers test](../../tests/conformance/triggers_test.go) marks
  each cron gone `onTemplate`;
- the tools the tests need beyond Go (gitleaks) come from
  [one action](../../.github/actions/tests/action.yml), in test.yml and in
  each job where a role runs the tests: the developer's and the tester's
  proofs, the curator's measures;
- test.yml runs the tests alone, the line simulator in a job beside the
  rest; its `test` job, the check `main` requires, waits for both.
  main's whole-repository checks are
  `route main`'s, once a merge ([main after a merge](../main-after-merge.md));
  gardening installs lychee, for the links to other sites, at night;
- gardening's backlog runs hourly too, at :47 UTC, one at a time, for
  a push of six hours; run by hand, its input `skip` leaves
  roles out (`documentalist`: the backlog alone);
- the templates' pins, which release-please moves, are never what
  workline's own checks run: a pull request moving them to a version not
  yet tagged stays green
  ([ADR-0017](../adr/0017-the-release-manager.md)); the line's own pin
  is a published release, tried
  ([ADR-0071](../adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md#1-the-line-runs-a-pinned-release));

**How workline releases** ([ADR-0017](../adr/0017-the-release-manager.md)),
with release-please, never a tag by hand:

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
  `workline:hold-release` or main's tip is not green, and builds the
  binary with GoReleaser: `workline version` must say the version it
  releases ([what holds a release](../release-holds.md));
- on that pull request, `judge` passes its branch (`--branch`): it is
  judged as any other, the reviewer leaving the versions it bumps; no
  doc holds it
  ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly));
  release-please keeps it on `main` (`always-update`);
- the person merges it each Monday, sooner for a fix that touches a
  real user: that is the decision. release-please tags the
  merged commit `vX.Y.Z` and writes the release with its notes; in the
  same run, release.yml builds the binaries with GoReleaser, which uploads
  them and keeps the notes (`release.mode: keep-existing`), and pushes the
  image;
- a failed publish is run again from the Actions page, never tagged again:
  a tag pushed is never moved (the Go proxy keeps its first commit).
