# Release manager — focused research (2026-10-02)

> **Decided since (2026-10-03).** workline builds no release manager: it
> works with the release tools a project has — release-please, release-plz,
> releaser-pleaser, semantic-release, changesets, GoReleaser — and the
> existing role is retired (docs/adr/0017-the-release-manager.md). The
> verdict below stands as the evidence; what it says workline would add is
> covered by those tools, but for the docs hold, which stays workline's.

**Verdict.** The *release PR* — a merge request the tool keeps up to date
with the next version, the version files and the changelog, tagged when a
person merges it — is the ecosystem's answer for teams and protected
branches (release-please, release-plz, changesets, releaser-pleaser). It
fits ADR-0011 as it stands: the agent proposes the merge request, the person
merges, a job with no AI tags. What no tool does generically: say where a
project keeps its versions without a plugin per language, keep the pins a
project's own files hold on its *next* version, and hold a release on docs
not yet true. Publishing stays the project's own pipeline, on the tag: every
registry makes a published version immutable, so the one irreversible step
must come last and be done by a job, never by an agent.

Stars and activity from the GitHub API that day; behaviour from each tool's
docs, linked.

## Landscape

Model: **release PR** (a merge request carries the release; merging it
tags), **tag-driven** (a pushed tag starts the publishing), **direct** (a CI
run on main computes, tags and publishes in one go).

| Tool | Stars, last push | Model | Decides | A person decides | Scope | Reported failure modes |
|---|---|---|---|---|---|---|
| [release-please](https://github.com/googleapis/release-please) | 7.6k, 2026-09 | release PR | next version from conventional commits; changelog; version files (`extra-files`, `x-release-please-version` markers, jsonpath/xpath updaters); tag + GitHub release on merge | when to merge; `Release-As:` footer overrides the version | generic, GitHub only (API-bound) | tags and PRs made with `GITHUB_TOKEN` trigger no workflow, a PAT is needed; a stale `autorelease: pending` label blocks the next PR; wants squash merges for a clean changelog |
| [release-please manifest](https://github.com/googleapis/release-please/blob/main/docs/manifest-releaser.md) | — | release PR, monorepo | each package's version in `.release-please-manifest.json`; `separate-pull-requests`; `linked-versions` plugin | which packages are linked | generic | config drift between manifest and files |
| [releaser-pleaser](https://github.com/apricote/releaser-pleaser) | 55, 2026-10 | release PR | as release-please; prerelease handled in the PR | merge | GitHub **and GitLab**, Go binary | young, "not all features work yet" |
| [semantic-release](https://github.com/semantic-release/semantic-release) | 24k, 2026-10 | direct | version, tag, notes, publish, on every push to a release branch | nothing per release; the commit types | Node, plugins per registry | [FAQ](https://semantic-release.gitbook.io/semantic-release/support/faq): commits the version nowhere, because a bot pushing to main must bypass protection; no monorepo support in core (multi-semantic-release, semantic-release-monorepo, which [cannot release several packages at once](https://github.com/qiwi/multi-semantic-release)) |
| [changesets](https://github.com/changesets/changesets) | 12.5k, 2026-09 | release PR ("Version Packages") | aggregates changeset files, bumps dependents, publishes on merge | **the contributor**, per change: a file stating bump and note | npm monorepos | prerelease mode is easy to leave on by mistake; a forgotten changeset releases nothing |
| [Nx release](https://nx.dev/docs/guides/nx-release/release-projects-independently) / [Lerna](https://github.com/lerna/lerna) (36k) | — | direct or CLI | fixed or independent groups; `updateDependents`; per-project tags `pkg@1.1.0` | groups; version plans | JS monorepos | independent mode drops the workspace changelog |
| [release-drafter](https://github.com/release-drafter/release-drafter) | 3.9k, 2026-10 | draft release | notes from merged PR labels; version from labels | publish the draft (or `publish: true`) | GitHub | labels forgotten = wrong version |
| [release-it](https://github.com/release-it/release-it) | 9k, 2026-09 | local CLI, interactive | bump, commit, tag, push, publish | answers prompts | Node, generic plugins | made for a person at a terminal |
| [GoReleaser](https://github.com/goreleaser/goreleaser) | 16k, 2026-10 | tag-driven | builds, archives, checksums, release assets, images | the tag | Go and others | refuses a dirty tree or a tag not on HEAD; notes empty when `changelog.disable` and nothing else writes them |
| [cargo-release](https://github.com/crate-ci/cargo-release) | 1.6k, 2026-09 | local CLI | bump, commit, tag, `cargo publish`, push | the bump level, run by hand | Rust | dry-run by default, `--execute` to act |
| [release-plz](https://release-plz.dev/docs/why) | 1.5k, 2026-10 | release PR | next version (commits **plus** cargo-semver-checks on the API), changelog via git-cliff; publishes what the registry lacks | merge | Rust; GitHub, [GitLab](https://release-plz.dev/docs/gitlab) | a release PR for every change to a package, conventional or not |
| [git-cliff](https://github.com/orhun/git-cliff) | 12k, 2026-09 | building block | changelog; `--bumped-version` | everything else | any, single binary | — |
| [changie](https://github.com/miniscruff/changie) (913), towncrier | — | building block | changelog from fragments | the fragment's text | any / Python | fragments forgotten |
| [python-semantic-release](https://github.com/python-semantic-release/python-semantic-release) | 1k, 2026-09 | direct | as semantic-release | — | Python | — |
| [cocogitto](https://github.com/cocogitto/cocogitto) | 1.2k, 2026-04 | local CLI | verify, bump, changelog, tag | run it | any | slowing down |
| [GitLab releases](https://docs.gitlab.com/user/project/releases/release_cli/) | — | CI `release:` keyword | a release object from a job | — | GitLab | `release-cli` deprecated in 18.0, removed in 20.0: `glab release create` |

Ecosystem constraints, which the role must know but not own:

| Ecosystem | Where the version lives | Published is | Undo |
|---|---|---|---|
| Go module | the tag only; `/v2` in the module path from major 2 ([ref](https://go.dev/ref/mod#major-version-suffixes)) | the proxy and sumdb keep a tag's first commit forever | `retract` in go.mod, then a newer version (workline v0.1.0) |
| npm | `package.json` | a name@version is never reusable ([docs](https://docs.npmjs.com/unpublishing-packages-from-the-registry)) | unpublish within 72 h; `deprecate` after |
| PyPI | `pyproject.toml`, or the tag (setuptools-scm, hatch-vcs) | a file name is never reusable, even deleted | yank (PEP 592, [docs](https://docs.pypi.org/project-management/yanking/)) |
| crates.io | `Cargo.toml`, each crate of a workspace | never overwritable | `cargo yank` |
| PlatformIO | `library.json` | — | `pio pkg unpublish` within 72 h ([docs](https://docs.platformio.org/en/latest/core/userguide/pkg/cmd_unpublish.html)) |
| Arduino Library Manager | `library.properties` | indexed hourly from tags; a version already indexed is never taken again ([FAQ](https://github.com/arduino/library-registry/blob/main/FAQ.md)) | a removal request |
| GitHub release | the tag | [immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases) (GA 2025-10) lock the tag and assets, with a Sigstore attestation | none, once immutable |

Provenance: npm trusted publishing by OIDC from GitHub Actions and GitLab CI
([GA 2025-07](https://github.blog/changelog/2025-07-31-npm-trusted-publishing-with-oidc-is-generally-available/)),
PyPI and crates.io trusted publishers the same way. The publishing job holds
no long-lived token, and it runs from the tag: another reason it stays the
project's pipeline, not the role. Signed tags (`git tag -s`, or keyless with
gitsign) are the project's choice; the role must not make them impossible
(it tags with `git tag -a` today).

Versioning schemes: [SemVer 2.0](https://semver.org) (pre-release
`-rc.1`, build metadata `+x` ignored in precedence, `0.y.z` promises
nothing), PEP 440 for Python (`1.0.0rc1`, `.post1`, no build metadata on
PyPI), [CalVer](https://calver.org) (pip's `26.2`). Commit signal:
[Conventional Commits](https://www.conventionalcommits.org).

## How workline's own releases went

v0.2.0 to v0.2.3, four releases on 2026-10-02 (PRs #40, #43, #46, #48):

1. A pull request moved `WORKLINE_VERSION` in five templates and this
   repository's own `workline.yml` to a version not yet tagged.
2. Its `judge` check went red: it downloads the pinned release, which does
   not exist until the tag. Only `test` is required on main, so the merge
   went through; the PR body warned of it each time.
3. A person pushed the tag by hand on the merge commit; `release.yml` ran
   GoReleaser and pushed the image.
4. The GitHub releases' notes are **empty** (one character):
   `.goreleaser.yaml` disables its changelog because "the release manager
   writes the notes", and the release manager does not run on workline.
   There is no CHANGELOG.md.
5. v0.1.0 was tagged too early, then moved: the Go proxy had kept the first
   commit, every `GOPROXY=direct` build failed its checksum; fixed by v0.1.1
   and `retract v0.1.0`. No ruleset protects tags.

The version is in the tag only (Go), plus the pins — files that name the
project's own next version.

## DomoticsCore

Twelve PlatformIO components, each with a semver in its `library.json`
(1.4.3 to 2.6.0) repeated as `metadata.version = "X.Y.Z";` in its sources;
the root at 2.12.1 in `library.json` **and** `library.properties` (Arduino);
components depend on each other by ranges (`DomoticsCore-Core >=1.0.0`).
Only the root is published (`pio pkg publish` at the root, on a tag).
`tools/bump_version.py` and `tools/check_versions.py` (`--check-tag`) are
the project's own; `version-check.yml` runs the check on every PR. The
v2.12.1 tag is on a `docs:` commit after `chore(release): 2.12.1`: the docs
were made true after the version was cut, which ADR-0010's hold would have
ordered the other way. The role's `packages` setting reproduces its rules
(roles/release-manager/README.md), but does not list `library.properties`.

### release-please on DomoticsCore, tried (2026-10-03)

A dry run (`release-please release-pr --dry-run --local`, v17) on a copy of
DomoticsCore at b6c1b67, nothing written on GitHub: manifest mode, the root
`.` and the twelve `DomoticsCore-*` folders as packages, `release-type:
simple`, `bootstrap-sha` the v2.12.1 commit, each `library.json` by a `json`
updater (`$.version`), each `metadata.version` line in `include/` and `src/`
(ten files) and the root's `library.properties` (a start/end block) by
generic markers. What it would have opened, from the commits since v2.12.1:
root 2.12.2, MQTT 1.10.1, Storage 1.6.2, WebUI 1.12.1, the `refactor(ha)`
moving nothing — what `bump_version.py` gives. A `feat(mqtt)` added: root
2.13.0, MQTT 1.11.0; the root package sees every commit, so it moves by
the highest level of any. A `Release-As: 3.0.0` footer in an empty commit
moved all thirteen packages; `release-as` on the root package in the
config moved the root alone, the person's override with its reason in that
commit. The root CHANGELOG.md entry goes above `## [2.12.1]`, the
hand-written preamble kept; release.yml's `awk` on `## [X.Y.Z]` still
finds it. `version.txt` is not created (`createIfMissing: false`); each
component's CHANGELOG.md would be (`skip-changelog` per package). Not
tried: the tags and GitHub releases made per component, and whether
`skip-github-release` keeps the next lookup right; a prose note in an
entry, which the generated changelog does not carry.

## Patterns worth borrowing

- **The release PR** (release-please, release-plz, changesets, releaser-pleaser):
  one open merge request, rewritten after each merge to main, holding the
  version files, the changelog and the notes. Merging it is the person's
  decision to release; nothing else is. It works with a protected main and
  no bot bypass, which is the case semantic-release's FAQ calls hard.
- **Tag on merge by a job that recognises the release PR** (release-please's
  `autorelease: pending` → `tagged` labels): deterministic, no AI, idempotent
  (tag already on that commit → only publish what is missing — the engine
  does this already).
- **Version files declared, not discovered**: release-please's `extra-files`
  with a type (json path, yaml, toml, xml) or a generic marker comment
  (`x-release-please-version`). Generic, no plugin per language; DomoticsCore's
  `version-patterns` is the same idea.
- **The registry as truth for "was it published"** (release-plz): compare with
  what the registry holds, not only the tags; a failed publish is retried,
  not re-tagged.
- **API-based bump checks** (cargo-semver-checks in release-plz; Go's
  `gorelease`, `apidiff`): a mechanical second opinion on "breaking", run as
  a gate when the ecosystem has one.
- **A person's override with a reason** (`Release-As:`; DomoticsCore's
  `root-bump` + `root-reason`, built).
- **Fixed vs independent groups** (Nx, Lerna, release-please `linked-versions`):
  a setting per group of packages, not a mode for the whole repository.
- **Dependents bumped** (changesets `updateInternalDependencies`, Nx
  `updateDependents`): when a package moves, the ranges that name it.
- **Retract, never move** (Go `retract`, PyPI yank, `cargo yank`, npm
  `deprecate`): a bad release is fixed by the next one.
- **Prerelease as a state of the release PR** (releaser-pleaser): the PR
  carries `-rc.N`; promoting is the person's merge of a PR without it.

## Anti-patterns

- **An agent or a person tagging by hand.** The one irreversible step done
  by whoever is at the keyboard (workline v0.1.0). A tag belongs to a job
  that checks it is on the merged release commit, and a ruleset forbids
  moving it.
- **A bot pushing to a protected main** to commit the version
  (semantic-release's FAQ): bypass rights for a token that runs plugins.
- **`GITHUB_TOKEN` for the tag**: the tag starts no workflow, so publishing
  silently never happens (release-please's README). A GitHub App's token,
  as ADR-0010 already uses for docs commits.
- **Publishing from the release role.** Registry tokens in the job that runs
  an agent breaks principle 6; the project's pipeline publishes from the tag.
- **A changelog that claims what is not built** (BACKLOG: the first trial
  release listed a role whose definition only was committed as `feat`).
- **Notes disabled in one tool, expected from another** (workline today).
- **A red check accepted as normal** on every release PR: a red that means
  nothing teaches people to merge on red.
- **Release on every push** (semantic-release's default) for a project
  whose release is a decision; fine for libraries that want it, not a default.

## The two generic problems

**Self-reference.** A project whose own files name its next version: CI
templates pinning `WORKLINE_VERSION`, a README's install line
(`go install …@v0.2.3`, ripgrep README still installing 14.1.1 while
Cargo.toml says 15.2.0, docs/research/documentalist-genericity.md), a GitHub
Action's `uses: owner/repo@v1`, a Dockerfile `FROM` its own image. These
are version files like any other: declared, rewritten in the release PR.
But a pin that is *executed* by the release PR's own CI cannot resolve
before the tag. Ways out seen: (a) the project's own CI never runs the pin
it ships — it builds from the commit under test, and only the templates
handed to others pin (workline's own `workline.yml` uses the template's
pin today); (b) the pins move in a follow-up
PR after publishing (Renovate or Dependabot on the project itself); (c) a
floating major tag (`v1`) moved by the publishing job — which contradicts
"never move a tag" unless restricted to an alias outside the Go proxy's view.

**The red check.** Generic form: a check that consumes the artifact the
release PR is about to create. Either the check knows it is on the release
PR and resolves from the commit (a), or the pin moves after (b). Leaving it
red is not an option under principle 12: a red read as fine is a pass by
habit.

## Generic, configured, human

| Part | Who |
|---|---|
| Next version from conventional commits, per package; changelog section | engine, no AI (built) |
| Keep one release MR up to date after each merge to main (open, rewrite, close when nothing to release) | engine, through the forge (ADR-0016: GitHub, GitLab, `cmd:`, `local` branch) |
| Docs suspect since the last tag hold the release MR (ADR-0010) | documentalist as a check on that MR |
| Release notes for users on top of the changelog | AI, bounded (built); checked against what shipped |
| Tag the merge commit of the release MR, publish the forge release | a job with no AI key, on merge (`merge` event) |
| Publish to registries, sign, attest | the project's pipeline, on the tag |
| Where versions live: files, json path or pattern, root vs packages, which paths count | project config, like the documentalist's `docs`/`derive` |
| Fixed vs independent, per group; dependents' ranges | project config |
| Scheme (semver, calver, PEP 440 forms), tag prefix, per-package tag pattern | project config |
| The project's own version check (`check_versions.py`) | a gate calling the project's script (built) |
| Merge the release MR = decide to release; override the bump with a reason; promote a prerelease; retract/yank | a person, always |
| Go major ≥ 2 (a new module path), any breaking release of a 1.x | a person: the role says so and stops |

## Cases the role must handle

| Repository | Versions live in | Publish | What is hard |
|---|---|---|---|
| **workline** | the tag; pins in `ci/**` templates and `.github/workflows/workline.yml` | GoReleaser + image on tag | self-reference and red check; empty notes; no CHANGELOG yet; Go proxy immutability; tag protection |
| **DomoticsCore** | 12 `library.json` + `metadata.version` in sources; root in `library.json` and `library.properties` | `pio pkg publish` on tag; Arduino index from tags | independent packages, fixed root with a person's override; inter-component ranges; the second root file; docs judged before the cut, not after |
| **A Python library** (pip-like, httpx-like) | `pyproject.toml`, `__version__`, or the tag via setuptools-scm | trusted publishing to PyPI on tag | PEP 440 forms (`rc1`, `.post1`, calver `26.2`); a file name never reusable; nothing to write when the tag is the version |
| **An npm monorepo** (backstage-like) | each `package.json`; internal ranges; maybe changeset files already | `npm publish --provenance` per package | independent vs fixed groups; dependents; per-package tags `@scope/pkg@1.2.0`; changesets already owning the decision |
| **A Rust crate / workspace** (ripgrep-like) | `Cargo.toml` per crate, `Cargo.lock`; README install line | `cargo publish` in dependency order | API breaks found by tools, not commits; publish order; the README pin |
| **No forge** (`forge: local`) | as configured | a person, by hand, or none | the release MR is a local branch; nothing tags on merge unless a hook does; ADR-0010's hold is the only moment docs are made true |

## Open questions for the round table

1. **Who tags on merge?** A `merge`-event job in the template recognising
   the release MR (by branch name, label, or a marker in the body), with a
   GitHub App token so the tag starts `release.yml`? On `forge: local`, a
   `post-merge` hook, or `workline release --tag` by the person?
2. **The red check on workline itself:** should `.github/workflows/workline.yml`
   build the engine from the commit (the repository is its own consumer),
   leaving the pins to the templates only? Or move the pins after the tag?
3. **Pins as version files:** declare them in the same `version-files`
   setting with a pattern (`WORKLINE_VERSION: v{version}`), or a separate
   `pins` setting because they move *after* the release on some projects?
4. **One release MR or one per package** (release-please's
   `separate-pull-requests`)? DomoticsCore publishes only the root; an npm
   monorepo publishes each package.
5. **Is `versions` a setting like the documentalist's `docs`:** a list of
   `{file, path|pattern, package}` replacing `version-files`,
   `packages.version-file` and `version-patterns`? Where does
   `library.properties` go?
6. **Changesets coexistence:** when a project already has a decision tool
   (changesets, towncrier fragments), does the role read its files instead
   of the commits, or stay out?
7. **API-break checks:** gate with the ecosystem's tool when present
   (cargo-semver-checks, gorelease, api-extractor), or leave to the reviewer?
8. **Prereleases:** a state of the release MR (`-rc.N`) toggled by a label,
   or a separate branch? How does a person promote?
9. **Hotfix branches:** a release MR on `release/1.x` from a maintenance
   branch — in scope for the first version?
10. **Tag protection and signing:** should `workline doctor` report a
    repository whose tags are not protected (ruleset, GitLab protected
    tags), and should the tag job sign?
11. **Notes checked against what shipped** (BACKLOG): a mechanical check that
    a `feat` touched code, not only a role definition — committer or release
    manager?
12. **The documentalist's hold** uses `git describe --tags` without the
    role's `tag-prefix`: with per-package tags, which tag is "the last
    release"?
13. **When nothing is releasable** but the person wants a release (docs
    only, a rebuild): `Release-As`-like override by a person, with a reason?
