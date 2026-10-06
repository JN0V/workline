# ADR-0017: workline does not cut releases; it works with the release tools a project has

- **Status:** accepted (2026-10-03)
- **Date:** 2026-10-02; rewritten 2026-10-03 on the person's decision
- **Amends:** ADR-0010 (the hold "at the release" moves onto the release
  pull request a tool opens); ADR-0016 (a release's notes are no longer one
  of workline's writes). Builds on ADR-0011 (the person merges).

## Context

The release manager (roles/release-manager/) computes the next version and
the changelog with no AI, writes notes with AI, and applies with
`flow: direct`: commit, tag, forge release in one run. `flow:
merge-request` is refused (internal/engine/engine.go). On a `main` taking
pull requests only, `direct` cannot push, so workline released v0.2.0 to
v0.2.3 by hand (PRs #40 to #48): a pull request moved the pins in `ci/**`
and `.github/workflows/workline.yml` to a version not yet tagged, its
`judge` check went red each time because it downloads that version, a
person pushed the tag, and the GitHub releases' notes are empty:
`.goreleaser.yaml` disables its changelog for a release manager that does
not run. v0.1.0 was tagged too early and moved; the Go proxy kept the first
commit.

The research (docs/research/release-manager.md) found the ecosystem's
answer for protected branches already built and widely used: the release
pull request — release-please (7.6k stars), release-plz, changesets,
releaser-pleaser (GitHub and GitLab) — one pull request kept up to date
with the next version, the version files and the changelog; merging it is
the decision; the tool tags the merge. semantic-release covers the direct
flow; GoReleaser publishes from the tag; git-cliff, towncrier and changie
write changelogs. Declared version locations (release-please's
`extra-files`, generic `x-release-please-version` markers) cover what the
research thought only workline would do.

A round table (product manager, architect, developer; adversarial,
edge-case and verification-gap lenses) read the research and the code. Its
findings, kept here as the evidence on what was built:

- **The tag is never pushed.** Apply tags locally and never pushes the tag
  nor the `chore(release)` commit; GitHub's `POST /releases`
  (internal/forge/github.go) sends `tag_name` with no `target_commitish`,
  so GitHub creates a lightweight tag on the default branch's head, with
  the job's token, which starts no workflow: nothing publishes, nothing is
  red. GitLab's call with no `ref` fails.
- **The last release is `git describe`**, the nearest tag, not the
  highest (a hotfix tag merged back after `v1.3.0` gives `v1.2.x`); its
  error is dropped, so a depth-1 clone finds no tag and starts again from
  `0.0.0` — on workline, `v0.1.0`, the retracted tag. `--match v*` also
  matches `vendor-*`.
- **Two "last release" lookups**: the role's (`--match <prefix>*`) and the
  documentalist's hold (internal/builtin/documentalist/documentalist.go,
  `holdTheRelease`, no `--match` at all).
- An existing forge release counts as done without its notes; a
  `v1.0.0-rc.1` tag makes every later release fail; breaking is a substring
  anywhere in a body; a revert still ships its `feat`; `packages: glob:
  packages/*` loses the path; `"version"` is rewritten at its first key;
  only JSON version files are read; `post` judges the version, not the
  notes; no status.md, no tried.md, no evaluation case.

Fixing all of that is rebuilding release-please inside workline. The
person decided on 2026-10-03: workline does not build a release manager.
Where a tool exists, workline works with it rather than redoing it
(principle 13); and a workline release manager was not shown to add value
over those tools.

## Who uses the role today

Found on 2026-10-03, in this repository, DomoticsCore and the configs on
this machine:

- **No project routes it.** workline's `.workline/config.yaml` and
  DomoticsCore's route `pre-push` only; no CI template runs `workline
  route release` (the templates route `merge-request` and `schedule`);
  DomoticsCore releases with its own `tools/bump_version.py`,
  `tools/check_versions.py --check-tag` and `release.yml` (`gh release
  create`, `pio pkg publish` on a tag). The sandboxes' configs are not on
  this machine; their templates route no release.
- **The engine's defaults**: `routing.default.yaml` routes `release:
  [documentalist, release-manager]` and the one shipped handoff,
  release-manager → documentalist; `cmd/workline/main.go` dispatches
  `builtin release-manager pre|post`; the `release` intention
  (internal/intent, internal/agent/prompt.go) and `applier.release`
  (internal/engine/engine.go) exist for it alone, as does each forge's
  `Release` (github, gitlab, command, local, fake; the `release` operation
  of docs/spec/forge-command.md and ci/forgejo/workline-forge.sh).
- **Tests**: the eight cases in tests/conformance/cases/release-manager/;
  the role used as a fixture by routing/handoff-runs-next-role,
  routing/apply-defers-handoff, routing/no-apply-defers-handoff,
  routing/undeclared-handoff-refused, engine/no-apply-then-apply and
  setup/doctor-docs-judged-nowhere; no evaluation case.
- **Docs**: README.md (the roles table, the `release` event row),
  docs/spec/routing.md, docs/usage.md, docs/spec/role-contract.md (an
  example), roles/documentalist/role.yaml and docs/checks.md (the handoff),
  docs/research/docs-and-release.md, `.goreleaser.yaml`'s comment.

The documentalist's `release` event is not the role's: its hold
(`holdTheRelease`) and the product docs due at the release
(conformance cases documentalist/release-*, evaluation case
propagates-a-product-doc-at-release) stay.

## Decision

### (a) No release manager; the existing one is retired

The role, its builtin (internal/builtin/releasemanager), the `release`
intention, `applier.release`, every forge's `Release` and the
forge-command `release` operation are removed, with their cases. Retired,
not frozen: no project runs it, so freezing keeps nobody working; and a
frozen role stays routed by default while it tags the wrong commit
silently, which principle 12 does not allow, and keeps its cases, docs and
code to carry. Routing's default becomes `release: [documentalist]`, with
no handoff shipped; the handoff mechanism keeps its cases on a fixture
role of the tests.

**Migration.** No project to move. A config that still names
`release-manager` is refused loudly, the message naming this ADR and a
release tool (below), never ignored; doctor says the same. A project with
DomoticsCore's rules (packages moving independently, the root moved by a
person with a reason) keeps its own scripts, or takes release-please's
manifest (`separate-pull-requests`, `linked-versions`, a `generic` updater
for `metadata.version` and `library.properties`) — its choice.

### (b) What stays generic in workline

**The documentalist judges a release pull request like any pull request,
and ADR-0010's hold applies to it.** Checked against the code and the
templates, for a pull request a tool's bot opens:

- **Opened with `GITHUB_TOKEN`** (release-please's default), no workflow
  runs on it at all (release-please-action's README): no workline check,
  no `test` either. Where `test` is required, the pull request waits for a
  check that never comes — loud, but unexplained; where no workline check
  is required, it is merged with no docs judged — silent. **Must change**:
  the release tool runs with a token whose pull requests start workflows —
  the GitHub App workline's templates already use (`WORKLINE_APP_ID`;
  contents, pull requests and issues in write cover release-please) — and
  doctor reports a release workflow using `GITHUB_TOKEN`.
- **Opened with the App**, `workline.yml` runs the `merge-request` route:
  the committer passes the tool's `chore(main): release X` subject, and the
  documentalist judges the docs the pull request made suspect — the
  changelog, the manifest, the pinned templates. **That is not the
  release hold**: `holdTheRelease` runs only when `WORKLINE_EVENT` is
  `release`, which nothing sets on a pull request. Docs left suspect since
  the last tag, and the product docs due at the release, are not seen.
  **Must change**: the `merge-request` route recognises a release pull
  request — a setting naming its head branches, defaulting to the known
  tools' (`release-please--*`, `releaser-pleaser--*`, `release-plz-*`,
  `changeset-release/*`) — and runs the documentalist's release duties on
  it: the hold, and the docs due at the release.
- **The fix goes elsewhere.** These tools rewrite their branch from `main`
  when they run again (to confirm on the sandbox), so a `docs:` commit
  pushed onto it with `--push-to-merge-request` may be lost. On a release
  pull request the documentalist's fixes go to a pull request of their own
  on `main` (`--open-merge-request`, as gardening's); the hold blocks the
  release pull request until that one is merged, the tool rewrites its
  pull request, and the next run clears the hold.
- **One "last release"**: the hold takes the highest tag matching the
  project's tag pattern (a setting, `v*` by default) in `git tag --merged
  HEAD`, prereleases and aliases ignored, and refuses a shallow clone
  rather than starting from no tag.
- **With a direct tool** (semantic-release), there is no pull request to
  hold: the release job runs `workline route release` before the tool, in
  the same job, and the tool runs only if it passed. With no forge, the
  person runs it before tagging (ADR-0010, unchanged).

**A project never executes, in the checks of a commit, the pin that
commit ships.** Its own CI builds itself from the commit under test; only
what is handed to others pins (templates, an install line). Doctor checks
it later (principle 7).

**Doctor and `init` detect the release tool present** (release-please's
config or manifest, `.releaserc`, `release-plz.toml`, `.changeset/`,
`.goreleaser.yaml`, `towncrier.toml`, `cliff.toml`) and, when none,
recommend one per forge: GitHub, release-please; GitHub and GitLab,
releaser-pleaser or semantic-release; no forge, semantic-release or
git-cliff and a tag by hand. A later step, below.

### (c) workline's own releases

- **release-please** (`release-type: go`; the tag is the version) keeps
  the release pull request, `CHANGELOG.md` and, through `extra-files`
  with a `generic` updater, the `WORKLINE_VERSION` pins of `ci/github/*.yml`
  and `ci/gitlab/workline.gitlab-ci.yml` (each pin's line marked
  `x-release-please-version`). Run with the App's token. This replaces
  the "one `versions` list" the person had chosen: the tool's list.
- **GoReleaser** runs in the same workflow when release-please says a
  release was created, on the tag release-please made on the merged
  commit; it keeps the release and the notes release-please wrote
  (`release.mode: keep-existing`, changelog still disabled) and uploads
  the binaries; the image as today. The person's answer "the notes handed
  to GoReleaser" holds in that form.
- **This repository's `workline.yml` builds the engine from the commit**,
  as `workline-gardening.yml` and `workline-sample.yml` already do; its
  pin is gone. The release pull request then has no red check.
- **What a person still does**: merge the release pull request. Nothing
  else — no tag by hand, no pin moved by hand.
- The person's other answers: prereleases later (release-please's
  `prerelease` when wanted); "stay out of changesets" is superseded by
  "integrate": changesets is one of the tools workline works with.

## Build order

Each step starts as conformance cases where the engine changes
(`pending.txt` and README's count with them), and meets its bar before the
next starts.

1. **Retire the role.** Remove what (a) lists, with its eight cases; the
   six cases using it as a fixture moved to a test role or rewritten;
   `release: [documentalist]`; a config naming it refused with this ADR's
   name; the docs listed above brought up to date. *Bar*: `go test ./...`
   green; `grep -r release-manager` finds only this ADR, the research and
   the refusal.
2. **workline's own CI builds from the commit.** *Bar*: a pull request
   moving the pins to an untagged version has every check green.
3. **release-please for workline**, with the App and GoReleaser after it.
   *Bar*: one real release where the person's only step is the merge; the
   tag on the merged commit; binaries, image and non-empty notes on the
   release; the four template pins moved in the release pull request.
   *Met* with v0.3.0 (2026-10-03): pull request #52 opened by the App,
   merged by the person, nothing else done by hand; the tag on the merge
   commit a8b1742; seven assets, the image `v0.3.0` and the notes
   release-please wrote on the release; the four pins moved in #52.
4. **The documentalist on a release pull request**: release branches
   recognised, the hold and the docs due at the release run there, fixes
   to their own pull request, the shared last-release lookup. *Bar*: on
   github.com/JN0V/workline-sandbox, a doc made suspect before the last
   tag holds the release pull request release-please opened, then clears
   once its fix is merged; a depth-1 clone is refused; a hotfix tag merged
   back is not taken for the last release.
   *Built* (2026-10-03), the bar's last two met by conformance cases
   (documentalist/release-*); the judging job, which holds no forge token,
   is given the branch (`--branch`), the forge asked otherwise; the
   setting is `release: {branches, tags}`. The sandbox try waits.
5. **Doctor and `init`**, later: the tool detected, one recommended when
   none, `GITHUB_TOKEN` on a release workflow reported, a shipped pin
   executed by the project's own CI reported. *Bar*: each said on a
   sandbox, none on a clean one.

## Consequences

- workline holds no version arithmetic, tag or registry logic; the bugs
  above go with the code that had them.
- A release is the person's merge of a pull request a tool keeps; the
  docs are judged on it before anything is tagged.
- The documentalist's release hold depends on the tool running with a
  token that starts workflows; doctor says when it does not.
- `forge: local` no longer keeps release notes (ADR-0016); a project with
  no forge tags by hand, after `workline route release`.
- The notes check the backlog wished for ("claims only what is built")
  becomes a question for the tool's changelog, out of workline's scope.
