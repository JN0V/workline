---
sources: [ci/github, .github/workflows/workline.yml, .github/workflows/workline-apply.yml, .github/workflows/workline-gardening.yml, .github/workflows/workline-sample.yml, .github/workflows/workline-answer.yml, .github/workflows/workline-develop.yml, .github/actions/engine/action.yml, .github/workflows/release.yml, .github/workflows/release-please.yml, release-please-config.json, .goreleaser.yaml, cmd/workline]
checked: eebe4270
verified: agent:claude-code
judged: 220b6dc
---
# workline's own workflows

Part of [workline on GitHub Actions](../ci-github.md): how workline's own .github/workflows/ differ from the templates.

workline's .github/workflows/ are the templates, but for a gate and the
engine:

- `gates` installs golangci-lint, pinned by its SHA-256, for
  [the gate](../../.workline/config.yaml);
- they build the engine from the commit they run on rather than run a
  release: gardening, the sample and the developer's
  (workline-develop.yml), every job, from `main`; a pull
  request's jobs, from its commit; workline-apply.yml's and
  workline-merge.yml's, from `main`'s commit, merged and reviewed: what a
  pull request changes in the engine never runs with the write token;
- once a commit ([the action](../../.github/actions/engine/action.yml),
  [ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#2-an-act-over-its-budget)):
  the first job on a commit builds it and keeps it in Actions' cache, the
  jobs after take it, nothing to build. Taken or built, it must name the
  commit checked out, unmodified, or the job fails. A pull request's runs
  keep theirs in its own cache, which `main`'s never read;
- no `merge_group`: a user's repository has no merge queue;
- test.yml runs the tests alone: main's whole-repository checks are
  `route main`'s, once a merge ([main after a merge](../main-after-merge.md));
  gardening installs lychee, for the links to other sites, at night;
- a project never runs, in a commit's checks, the pin that commit ships: a
  pull request moving the templates to a version not yet tagged would
  download nothing, as on each release from v0.2.0 to v0.2.3
  ([ADR-0017](../adr/0017-the-release-manager.md));

**How workline releases** ([ADR-0017](../adr/0017-the-release-manager.md)),
with release-please, never a tag by hand:

- on each push to `main`, release-please.yml, with the App's token, keeps
  one pull request open, "chore(main): release X": the next version from
  the conventional commits since the last tag (`feat` moves the minor
  while workline is at 0.x, as a breaking change does), [CHANGELOG.md](../../CHANGELOG.md), and
  the templates' `WORKLINE_VERSION` and the one in
  [gitlab-trigger.md](../gitlab-trigger.md)'s example, each on a line marked
  `x-release-please-version` (release-please-config.json);
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
