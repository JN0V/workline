# Merge conflicts between open pull requests — focused research (2026-10-12)

**Question.** Why do workline's open pull requests conflict with `main` so
often, what in them conflicts, and what do forges, git and other tools
offer to settle it before a person or an agent rebases by hand?

**Verdict.**

- **Most conflicts are workline's own header lines.** Of 150 conflict hunks
  met by people's pull requests in two weeks, 89 (59%) were `checked`,
  `judged` or `verified` lines and 7 (5%) were derived blocks; 9 pull
  requests conflicted on nothing else.
- **The cause is the docs' wide `sources`.** A doc naming a whole folder
  (`cmd/workline`, in 16 docs) has its `checked` moved by every pull request
  touching that folder: two such pull requests always change the same line.
- **Nothing outside settles it.** No merge queue on a user's repository,
  and it removes a pull request in conflict anyway; Update branch stops at
  one; forges never run merge drivers; rerere, Mergiraf, notes do not fit.
- **What fits is completing Update branch**: workline merges the base into
  the branch when every conflict is in a line it owns, settled key by key
  as `follow` does on its own branches. Proposed in
  [ADR-0041](../adr/0041-workline-settles-its-own-lines-when-the-base-moves.md).

Already covered elsewhere: merge drivers and derived blocks
([documentalist.md](documentalist.md#generated-sections-and-parallel-branches-2026-10-05)),
what a bot's commit resets ([ci-and-forge.md](ci-and-forge.md#what-a-team-adds-review-of-worklines-templates-2026-09-30)).

## Measures (2026-09-28 → 2026-10-12)

### Method

- **Population**: the 190 pull requests merged in the window, 147 by a
  person, 43 by a bot (release-please, workline's GitHub App).
- **Every update a branch went through, redone**, from the forge's
  timeline: a force push that moved where the branch left `main`, as git's
  three-way merge of the old head onto the new base, in memory (`git
  merge-tree`); a merge commit, as the merge of its parents. Every commit a
  force push left was fetched.
- **Each conflict hunk classed**: *header* when every line on every side
  sets `checked`, `judged`, `judged-in-parts` or `verified` (a hunk mixing
  one with another line counts as doc text); *derived* inside a derived
  block; *release* for the changelog, manifest and version pins; *doc
  text*, any other Markdown; *code*, anything else.
- **Clean updates**: merges of `main` into a branch that met no conflict.
  **Bots' rebuilds**: from a `main` commit to the bot's force push after
  it. The scripts and raw events were kept outside the repository.

### People's pull requests

| Window | Pull requests in conflict | Hunks | Header | Derived | Doc text | Code |
|---|---|---|---|---|---|---|
| 09-28 → 10-12 | 24 of 147 | 150 | 89 (59%) | 7 (5%) | 38 (25%) | 16 (11%) |
| 10-06 → 10-12 | 14 | 96 | 60 (62%) | 1 | 21 | 14 |

- **No `sources` line conflicted**: only the lines recording a judgement.
  13 of the 38 doc-text hunks and 4 of the 16 code hunks were two additions
  at the same place.
- **Header and derived lines only**, so the pull request would have merged
  clean without them: 9 — [#21](https://github.com/JN0V/workline/pull/21),
  [#144](https://github.com/JN0V/workline/pull/144),
  [#146](https://github.com/JN0V/workline/pull/146),
  [#157](https://github.com/JN0V/workline/pull/157),
  [#219](https://github.com/JN0V/workline/pull/219),
  [#221](https://github.com/JN0V/workline/pull/221),
  [#249](https://github.com/JN0V/workline/pull/249),
  [#253](https://github.com/JN0V/workline/pull/253),
  [#262](https://github.com/JN0V/workline/pull/262).
- **Mixed with a real overlap**: 9, [#285][285] and [#316][316] among them.
  **Real overlap only**: 6, such as [#255][255] (a list in
  `.workline/config.yaml` both appended to) and [#314][314] (a paragraph of
  a doc [#312][312] condensed whole).
- **Derived blocks almost gone since
  [ADR-0027](../adr/0027-derived-blocks-are-regenerated-on-the-default-branch.md)**
  (2026-10-05): README's count conflicted in six pull requests before; one
  derived hunk since ([#257][257]).
- **[#316][316], today.** Rebased onto [#315][315]'s squash, it met 17
  header hunks — 16 docs, and README.md's `<!-- workline` header
  (`checked: 68b431f` against `7fd4a71`) — 2 code hunks in
  tests/conformance/runner_test.go, 1 doc-text hunk in
  docs/spec/conformance.md. Its builder settled the headers by checking
  the docs again (adf03f5).
- **Why the header lines collide**: a feature pull request moves `checked`
  on 20 to 30 docs ([#299][299] 34, [#315][315] 32, [#316][316] 24), since
  `sources` name whole folders: `cmd/workline` in 16 docs, also
  `internal/engine` and `internal/agent`. docs/spec/backlog-acts.md's
  `checked` moved in 48 commits on `main` in the window.

### Clean updates forced by "up to date"

- `main`'s ruleset requires a branch up to date before it merges
  (`strict_required_status_checks_policy: true`); merges are squashes only.
- 30 clean Update branch merges — 13 on people's pull requests, 17 on
  bots' — and 4 local merges of `main`. None met a conflict: each waited
  only on a click and the checks run again.

### Bots' pull requests

- **The documentalist's release fix**: 15 rebuilds over a real header
  conflict, across 8 pull requests, a feature's `checked` beside the fix's
  `judged` ([ADR-0034](../adr/0034-the-release-fix-follows-its-base.md)'s
  case). A person rebased two by hand before the `follow` job
  ([#191][191], 2026-10-06); since, rebuilt a median of 33 s after the
  `main` commit, 90th percentile 45 s (45 rebuilds, one outlier at 22 min).
- **release-please**: 53 rebuilds, median 25 s; 4 over its own files, by
  design. The minute of "conflict" a person sees on a bot's pull request
  is the forge computing mergeability before the bot's push.

### How sure

- **High for what reached GitHub**: git's own merge, the one GitHub runs,
  redone on the exact commits; checked by hand on [#316][316], [#196][196]
  (version pins), [#255][255] and [#285][285] (Go code).
- **A lower bound**: a conflict settled before the first push leaves no
  trace; a whole-branch merge can meet fewer conflicts than a rebase
  replaying commit by commit; the header class is strict.

## `checked` after a squash

- A `checked` that `main` does not hold reads as the commit that brought
  that id into the doc, the squash of the pull request that checked it
  (`anchorFor`, internal/builtin/documentalist/documentalist.go; case
  documentalist/doc-fixed-in-a-squashed-merge-request). At 7df3ab2, 51 of
  52 docs name a branch's commit; all 51 are anchored.
- **For a merge**: after the squash, a `checked` kept from the branch reads
  as checked at this pull request's squash, which holds what `main` merged
  before; one kept from `main`, at the other pull request's squash, so this
  one's changes are listed since.

## What exists elsewhere

### Forges

| Mechanism | What it does | Why it does not settle these |
|---|---|---|
| GitHub merge queue | Merges pull requests in turn, each tested on those before it; "the same benefits as" requiring branches up to date, without the author updating ([docs](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue)) | Offered on Enterprise Cloud and on organisations' public repositories ([changelog, 2023-07-12](https://github.blog/changelog/2023-07-12-pull-request-merge-queue-is-now-generally-available/)); this repository is a user's (GraphQL `mergeQueue` is null). A pull request in conflict with the base is removed from the queue |
| GitHub Update branch (button; `PUT /repos/{o}/{r}/pulls/{n}/update-branch`) | Merges the base into the branch; the button can rebase instead | Stops at a conflict, left to the web editor or a local merge |
| Kodiak `update.always`, update-branch Actions | Update every pull request behind its base | Same stop: on a conflict Kodiak comments and removes its automerge label. Kodiak warns each base commit can update every open pull request |
| Renovate, Dependabot | Recreate their own branch on the base (Renovate: `rebaseWhen: conflicted` by default, `behind-base-branch` when the repository requires up to date) | Their own branches only; Renovate stops once a person commits to its branch. workline does the same for its own ([ADR-0034](../adr/0034-the-release-fix-follows-its-base.md)) |
| GitLab merge trains (Premium), the merge request's Rebase (`PUT /projects/:id/merge_requests/:iid/rebase`) | A queue; a rebase on the target | A merge request in conflict leaves the train; Rebase stops at a conflict |

### git

- **Merge drivers**, tried in a scratch repository: `merge=union` settles
  `checked: aaa→bbb` against `aaa→ccc` in a working tree by keeping both
  lines, a duplicate key; in a bare repository, where a forge merges, `git
  merge-tree` still conflicts (attributes come from the tree only with
  `--attr-source`). A custom driver is a command in git's config, which no
  forge holds
  ([rubocop#3594](https://github.com/rubocop-hq/rubocop/pull/3594),
  [gitlab#18830](https://gitlab.com/gitlab-org/gitlab/-/issues/18830),
  [BSERV-4566](https://jira.atlassian.com/browse/BSERV-4566),
  [BCLOUD-20234](https://jira.atlassian.com/browse/BCLOUD-20234)); a local
  one would help local merges and rebases only.
- **rerere** replays a resolution keyed on the conflict's text; each
  `checked` conflict holds other ids, so it never matches.
- **Syntax-aware mergers**: Mergiraf reads YAML, not Markdown
  ([languages](https://mergiraf.org/languages.html)); even in YAML, one key
  changed on both sides is a real conflict, and adjacent keys are what
  `follow`'s key-by-key merge settles already.

### Where else "checked at commit X" could live

- **A date in the file** (Apertis `lastreview`, Fedora's proposed
  `:last-reviewed:`): two pull requests the same day write the same line,
  which git merges; but a date cannot list what changed since.
- **Off the doc**: a commit trailer (`Docs-Checked: …`) is dropped by this
  repository's blank squash message (`squash_merge_commit_message: BLANK`);
  git notes sit on the branch's commits, which a squash drops; a ref per doc
  under `refs/workline/`, written on `main` after each merge, needs each
  pull request's verdict where that job reads it.
- **Kept in the file** for [ADR-0013](../adr/0013-what-was-judged-is-not-asked-again.md)'s
  reason: "a person sees it in review".

## Options

| | What | Gain | Cost |
|---|---|---|---|
| A | `follow` merges `main` into each open pull request left in conflict, when every conflict is in a line workline owns | Every conflict of 9 of the 24 pull requests; a person's commits never rewritten; the merge code exists | workline writes to branches it did not create; a fork's is not writable; each merge runs the checks again; a doc both sides checked is judged again |
| A+ | A, and branches behind but clean | The 30 clean Update branch clicks | The checks run on every open pull request at each push to `main`; a branch moves under the agent working on it |
| B | Take `checked` out of the doc: history, plus a trailer, a note or a ref | Removes the hotspot at its root | Reverses ADR-0013's record in review; depends on forge settings or a job knowing each verdict; weeks of work |
| C | Narrow the `sources` naming whole folders; or leave `checked` on a doc whose text a pull request did not change | Fewer collisions; the first with no engine change | More sources to keep; the second leaves `main`'s docs behind until gardening spends the tokens |
| D | One builder pull request open at a time | No code | Throughput: three agents work in parallel today |
| E | Move to an organisation, for the merge queue | The up-to-date clicks | The module path changes; no conflict settled |
| F | A local merge driver, set up by `workline setup` | Local merges and rebases, an agent's or a person's | Never runs on the forge; a complement to A |

A leaves the 9 mixed pull requests whole: a merge commit cannot hold a
conflict; F would settle their header hunks locally. Taken: A, in
[ADR-0041](../adr/0041-workline-settles-its-own-lines-when-the-base-moves.md);
A+ left to the person; C and F open.

[191]: https://github.com/JN0V/workline/pull/191
[196]: https://github.com/JN0V/workline/pull/196
[255]: https://github.com/JN0V/workline/pull/255
[257]: https://github.com/JN0V/workline/pull/257
[285]: https://github.com/JN0V/workline/pull/285
[299]: https://github.com/JN0V/workline/pull/299
[312]: https://github.com/JN0V/workline/pull/312
[314]: https://github.com/JN0V/workline/pull/314
[315]: https://github.com/JN0V/workline/pull/315
[316]: https://github.com/JN0V/workline/pull/316
