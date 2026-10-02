# ADR-0017: The release manager keeps a release merge request; a job tags its merge

- **Status:** proposed
- **Date:** 2026-10-02
- **Amends:** ADR-0010 (the hold at the release moves onto the release merge
  request); builds on ADR-0011 (the person merges) and ADR-0016 (where the
  writes go)

## Context

The release manager computes the next version and the changelog with no AI,
writes notes with AI, and applies with `flow: direct`: commit, tag, forge
release, all in one run (roles/release-manager/README.md). `flow:
merge-request` is refused (engine.go). On `main` taking merge requests only,
`direct` cannot push; so workline releases by hand. v0.2.0 to v0.2.3
(PRs #40 to #48): a pull request moved the pins to a version not yet tagged,
its `judge` check went red each time because it downloads that version, a
person pushed the tag, and the GitHub releases' notes are empty: GoReleaser
disables its changelog for the release manager, which does not run. v0.1.0
was tagged too early and moved; the Go proxy kept the first commit.

The ecosystem's answer for protected branches is the release merge request
(release-please, release-plz, changesets, releaser-pleaser): one merge
request kept up to date with the next version, version files and changelog;
merging it is the decision to release; a job tags the merge
(docs/research/release-manager.md). No tool says where a project keeps its
versions without a plugin per language, keeps a project's own pins, or holds
a release on docs not yet true.

A round table read the research and the code (BMAD product manager,
architect, developer; adversarial, edge-case and verification-gap lenses).
It found defects the release merge request would inherit, all in code built
today:

- **The last release is `git describe`**, the nearest tag, not the highest:
  a hotfix tag merged back after `v1.3.0` gives `v1.2.x` next. Its error is
  dropped: a depth-1 checkout finds no tag and starts again from `0.0.0`
  (on workline, `v0.1.0`, the retracted tag). `--match v*` also matches
  `vendor-*`; the documentalist's hold (documentalist.go) has no `--match`
  at all, a second definition of "last release".
- **The forge can tag the wrong commit**: apply tags locally and never pushes
  the tag; GitHub's `POST /releases` with no `target_commitish` then creates
  a lightweight tag on the default branch's head, with the job's token,
  which starts no workflow: nothing publishes, nothing is red.
- **An existing forge release counts as done** without its notes: workline's
  empty releases would stay empty.
- **A release tag before a prerelease fails**: a `v1.0.0-rc.1` tag makes
  `NextSemver` fail for every later release.
- **Breaking is a substring anywhere in a body**; a revert still ships its
  `feat`; `packages.glob: packages/*` loses the path; `"version"` is
  rewritten at its first key, a dependency's when one comes first; only JSON
  version files are read (DomoticsCore's `library.properties` is not).
- **`post` judges the version, not the notes**: nothing checks they hold the
  generated changelog, claim only what was built (docs/BACKLOG.md), or leak
  nothing, though they end in an immutable tag message.
- No `roles/release-manager/status.md` or `tried.md`; the eight conformance
  cases are all `flow: direct`; no evaluation case for the notes.

## Decision

### What the role does, and does not

- **Does, with no AI**: compute the next version per release unit; rewrite
  the declared version locations and the changelog; keep one release merge
  request per release unit up to date; hold it on docs suspect since the
  unit's last release and on the project's own version check; tag the
  merged release commit, push the tag, create the forge release,
  idempotently.
- **Does, with AI**: the notes on top of the generated changelog, nothing
  else (`policy.md`), judged by `post`.
- **Does not**: merge or turn on auto-merge (ADR-0011); push to a protected
  branch; move or delete a pushed tag — a bad release is fixed by the next
  one, with `retract`, yank or deprecate advised; publish to a registry or
  hold a registry token (principle 6; the project's pipeline publishes from
  the tag, by OIDC where the registry offers it); plug per-language logic
  into the engine; choose the bump — a person overrides it, with a reason.
- **Stops for a person**: a Go major from 2 (a new module path), a first
  `1.0.0`, a breaking change in `0.y` beyond the configured rule, a release
  when nothing is releasable. Each said, none guessed.
- **Always says how it ended**: tagged, published, refused with a reason, or
  nothing to release. A green job that did nothing is a defect (principle 12).

### The data model

```yaml
release-manager:
  settings:
    flow: merge-request          # direct stays, for a project with no protected branch
    base: main                   # a setting from day one, for hotfix lines later
    packages:                    # a release unit is a package or a fixed group
      - {id: root, path: ., tag: "v{version}", truth: tag}
      - {id: core, path: DomoticsCore-Core, tag: "DomoticsCore-Core@{version}",
         truth: file, counts: ["include/**", "src/**"]}
    groups: [{packages: [...], mode: fixed | independent, publishes: true}]
    versions:                    # declared, never discovered (principle 13: release-please's extra-files)
      - {file: library.json, json: version, package: root}
      - {file: library.properties, pattern: "version={version}", package: root}
      - {file: "DomoticsCore-*/src/**", pattern: 'metadata.version = "{version}";', expect: 1}
      - {file: "ci/**/*.yml", pattern: "WORKLINE_VERSION: v{version}", kind: pin}
```

- **A location** has a file (glob), a locator (`json`, `toml`, `yaml` path,
  `pattern`, or a marker comment), the package it follows, how many times it
  must match (`expect`, default: at least once), and a kind: `version`
  (rewritten in the release merge request), `pin` (the project's own version
  in files others run), `range` later (a sibling's dependency range).
- **A location matching nothing, or not the expected count, blocks** — the
  check that stops a `version = "…"` pattern rewriting a dependency's.
  A project whose version is its tag (Go, setuptools-scm) declares none, and
  its release merge request holds the changelog only.
- **Each package has one truth**: the tag, or a file; the other is checked
  to agree, not read as truth.
- **`versions` replaces `version-files`, `packages.version-file` and
  `version-patterns`**, which are read through a shim, with a doctor hint.
- **One "last release" function**, shared by the role and the
  documentalist's hold: the highest version matching the unit's tag pattern
  in `git tag --merged HEAD`, prereleases and aliases (`v1`) ignored. A
  shallow clone, or no tag where `git tag -l` lists some, is refused.

All six agreed on declared locations and one `versions` list; the
adversarial lens added `expect`, the architect the single truth and the
`kind`.

### The flow on a forge (GitHub, GitLab, `cmd:`)

1. **On each merge to the base** (`merge` event, a concurrency group per
   unit): the engine computes the next version against the base's head,
   writes the locations and the changelog on branch `workline/release[-<unit>]`,
   the AI writes the notes into the same merge request (a notes file the
   person reads in the diff, before anything is immutable), and
   `OpenMergeRequest` opens or rewrites it. Nothing releasable: the merge
   request is closed (a new forge verb, `close`, in github, gitlab, `cmd:`,
   local and the fake) and the run says so.
2. **Its checks**: the documentalist's hold on docs suspect since the unit's
   last release (ADR-0010's hold, moved here: DomoticsCore's `v2.12.1`
   was cut before its docs were made true); the project's version check
   (`check_versions.py`); the notes checks (below); and *up to date* — the
   merge request was computed on the base's current head. All required.
3. **A person merges it.** That is the decision to release, and nothing else
   is.
4. **The tag job** (`merge` event on the base, no AI key, the default
   branch's code, never the merge request's) recognises a release merge:
   the merged commit carries the engine's marker, and its diff touches only
   the declared locations, the changelog and the notes, with the content the
   engine regenerates there. It reads the version from the merged commit
   (the architect's "one decision, read everywhere") and recomputes it from
   the last release as a check (the adversarial lens): a difference is
   refused, loudly. It tags the commit that landed — merge, squash or
   rebase — with `git tag -a` (never preventing `-s`), pushes the tag with a
   token whose pushes start workflows (a GitHub App, as ADR-0010; a GitLab
   project token allowed on protected tags), checks the remote holds the tag
   on that exact commit, then creates or completes the forge release
   (`release.create` with the target commit; notes filled when empty or
   different). Tag already on that commit: only what is missing is done. On
   another commit: refused, never moved. In a fork: refused.
5. **The project's pipeline publishes from the tag.** A failed publish is
   rerun on the existing tag, never re-tagged.

### With no forge (`forge: local`, ADR-0016)

The release merge request is a local branch recorded in
`.git/workline/merge-requests/`; the person merges it with git, then runs
`workline release --tag`, which does step 4 and pushes the tag when a remote
exists. No `post-merge` hook: hooks skip fast-forward pulls and run
anywhere — all six agreed. The notes go to `.git/workline/releases/<tag>.md`.
In CI, `local` refuses its writes, as for every write.

### With ADR-0010, 0011 and 0016

- ADR-0010's "at the release" hold becomes a required check on the release
  merge request, per unit, through the shared last-release function.
- ADR-0011 stands: the person merges. Merging is now publishing, so
  auto-merge on a release merge request is refused by the tag job (it reads
  who merged; see *For the person to decide*).
- ADR-0016 carries every write: the merge request, its close, the release.

### The self-reference and the red check

Generic form: a check on the release merge request consumes the artifact
the merge request is about to create. Rule: **a project never executes, in
the checks of a commit, the pin that commit ships** (the architect's AD-4;
doctor checks it, principle 7). Its own CI builds itself from the commit
under test; only what is handed to others pins. On workline:
`.github/workflows/workline.yml`'s `judge` builds the engine from the
commit; `apply`, which holds the write token, keeps running the last
released binary, so a pull request cannot change the code that holds the
token (the architect's trust zones). That pin is a `pin` moved after the tag,
by a follow-up merge request the tag job opens. A pin only shipped (the
templates in `ci/**`, a README's install line) moves in the release merge
request: text, not run. A red check accepted as normal is not an option.

### The notes

`post` also checks the notes: the generated changelog as a literal part;
gitleaks and the project's forbidden-terms rules; a `feat` or `fix` whose
commits touch no `counts` path or only docs and role definitions is
reported (the release manager, not the committer: the committer sees one
commit, not what a release claims; the adversarial lens argued the
committer). A secret or a forbidden term blocks; the claim check is
reported, not blocking, until its false alarms are measured.

## Build order

Each step starts as conformance cases (`tests/conformance/cases/release-manager/`,
`pending.txt` listing what does not pass, README's count with it), and is
measured with the bar written here before the next one starts.

0. **The defects under the release.** Shared last-release function
   (highest, prefix-exact, shallow clone refused) used by the documentalist's
   hold; breaking from footers only; revert drops its entry; prerelease tags
   do not block; `pre-major: minor`; `packages/*` keeps its path; the tag
   pushed before the forge release, with its target, refused when the remote
   tag is elsewhere; empty notes filled; `roles/release-manager/status.md`
   and `tried.md` written. *Bar*: each case failed before its fix; a depth-1
   copy of workline refuses instead of computing `v0.1.0`; a hotfix tag
   merged back gives the highest next version.
1. **workline's own red check.** `judge` builds from the commit, `apply` on
   the last release. *Bar*: the next release pull request has no red check.
2. **`versions`.** Typed locations, `expect`, `kind`, the shim. *Bar*: on a
   copy of DomoticsCore, `check_versions.py --check-tag` passes with both
   root files moved; a tag-only Go or setuptools-scm copy writes nothing; a
   location matching nothing blocks.
3. **The release merge request.** Open, rewrite, close; the hold, the
   version check and *up to date* as its checks; fake, local, `cmd:`.
   *Bar*: on a throwaway GitHub repository, three ordinary merges rewrite it
   three times, no version drifts, the hold blocks then clears.
4. **The tag job.** Templates for GitHub and GitLab; `workline release
   --tag` for local. *Bar*: on GitHub and GitLab sandboxes, the tag lands on
   the merged commit for merge, squash and rebase; it starts the publish
   workflow; killed between tag and release then rerun, one tag and one
   release; a fork copying the marker is not tagged; a stale merge request
   is refused.
5. **The notes checks**, with evaluation cases (a breaking change first with
   its upgrade step, a role-definition-only `feat` not advertised, no
   internal codes). *Bar*: a seeded secret is blocked; false alarms counted
   on workline's last four releases, fewer than the real ones; tokens per
   release recorded.
6. **Doctor**: tags not protected, a release tag made with `GITHUB_TOKEN`,
   a decision tool present (changesets, towncrier), a project executing its
   own shipped pin. *Bar*: each reported on a sandbox, none on a clean one.
7. **workline releases through it.** *Bar*: one release where the person's
   only step is the merge; notes non-empty; every check green. Counted from
   then on: release merge requests merged on red, releases finished by hand;
   both stay at zero.

Later, each its own decision: per-package merge requests beyond the default
one per unit, prereleases, hotfix lines, dependents' ranges, an API-break
gate that blocks.

## Tried, and to try

- **Tried**: `flow: direct` on a copy of DomoticsCore (README; its
  `check_versions.py --check-tag` passed); workline's four releases by hand
  (above). Nothing of the merge-request flow is built or tried.
- **To try**: every bar above, on real copies and sandboxes
  (github.com and gitlab.com), recorded in `roles/release-manager/tried.md`;
  the Forgejo sample through `cmd:`; GitHub's immutable releases with
  GoReleaser after the forge release; GitLab merge trains.

## Consequences

- A release is one merge by a person; the tag, the forge release and the
  notes follow with no one at a keyboard, or they say why not.
- The notes are read by the person in the merge request before they become
  immutable.
- Every check on a release merge request can be green; a red one means
  something.
- A project declares where its versions live; a declaration that matches
  nothing stops the release instead of passing.
- Configurations with `version-files`, `packages.version-file` or
  `version-patterns` keep working through a shim, with a doctor hint.
- A project whose own CI runs its shipped pin must build itself from the
  commit, or move that pin after the tag; doctor says which.
- The tag job needs a token that can push tags and start workflows: a GitHub
  App, a GitLab project token; `GITHUB_TOKEN` is reported.

## For the person to decide

1. **Pins: one `versions` list or a separate `pins` setting.** *Options*:
   a `kind: pin` in `versions` (product manager, architect, developer,
   verification: one place to assert every location matched); a `pins`
   setting of its own (adversarial, edge: pins move at another moment, and
   folding them in brought the red check). *Recommended*: one list with
   `kind` and a `when: release | after-tag` — the timing is an attribute,
   and the red check is stopped by the self-reference rule, not by where the
   pin is declared.
2. **Decision tools already in a project (changesets, towncrier).**
   *Options*: stay out — doctor says so and the role stops (product manager,
   developer, adversarial, verification); read their fragments instead of
   the commits (architect, edge). *Recommended*: stay out in the first
   version; an adapter later, if a project asks. Two deciders give two
   versions.
3. **Prereleases in the first version.** *Options*: later (product manager,
   adversarial); a `-rc.N` state of the release merge request set by a
   label, promoted by removing it (architect, edge, verification).
   *Recommended*: later; step 0 only makes a prerelease tag stop blocking.
4. **Who creates the forge release when the project's pipeline also does
   (GoReleaser).** *Options*: the tag job, before the pipeline (the
   architect's `release.create`; the developer fills empty notes) — which
   GitHub's immutable releases may lock before the assets upload; the
   pipeline, handed the notes from the merged notes file (the adversarial
   lens: irreversible last). *Recommended*: the job creates it only when the
   project declares no publisher; with one, it hands over the notes file
   (GoReleaser `--release-notes`) and checks after that the release holds
   them. On workline, `.goreleaser.yaml` takes the notes file.
5. **Guard against an agent merging the release.** ADR-0010 has `main` at
   zero approvals with auto-merge; merging a release merge request is
   publishing. *Options*: the tag job refuses a merge by a bot or by
   auto-merge (adversarial); a protected environment with a required
   reviewer on the publish job (adversarial; costs a second click, against
   the product manager's one-click release). *Recommended*: the tag job's
   refusal by default, said in the job; the environment a project's opt-in,
   suggested by doctor.
6. **The release merge request's checks blocking on the notes.** *Options*:
   report only (product manager); block (verification). *Recommended*:
   a secret or a forbidden term blocks from the start; the "claims what is
   not built" check reports until step 5 has measured its false alarms,
   then blocks once they are fewer than its real findings.
