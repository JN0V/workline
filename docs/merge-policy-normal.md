---
sources: [internal/engine/routine.go, internal/engine/code.go, internal/engine/demote.go, internal/engine/merge.go, internal/forge/merge.go, internal/steer/steer.go]
checked: 6163e1c4
verified: agent:claude-code
reader: efab3a2a0b35
judged-in-parts: 87b34bc
---
# The merge policy at `normal`

Part of [the merge policy](merge-policy.md): at `normal`, workline's own
routine pull requests merge with no person's yes; everything else waits
for one, as at `cautious`
([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md)).

```yaml
merge: {autonomy: normal}   # a project's choice, never a default
```

## Routine work with no yes

- **Merged with no label**: workline's own pull requests, as the forge
  says it — a commit says anything of itself:
  - from a role's branch (`workline/<role>/…`), opened by workline's App;
  - every push to it the App's, every commit committed by it and its own
    by its tree;
  - on the local forge, with no App: a role's branch, every commit a
    role's by its trailer and tree, whatever its committer — the roles
    commit under your name there;
  - only docs of the documentalist's, none of
    [what never merges alone](#what-never-merges-alone);
  - in a role's folder, only its README and `docs/` pages, never what
    the agent is given — persona, knowledge, instruction, policy, lenses
    ([the layout](spec/role-contract.md#layout));
  - their head's verdict passed, every agent's review whole.
- **Four kinds, by what changed** (and the developer's `code`,
  [below](#the-developers-code-a-fix-alone-a-feature-with-a-yes)):
  - `records`: only when docs were read (`judged`) — gardening's
    `workline/documentalist/records` among them, for the docs judged on
    merged merge requests
    ([more](../roles/documentalist/docs/gardening.md#records-kept-on-merged-merge-requests));
  - `derived`: derived blocks, as their commands give them;
  - `docs`: doc text a role wrote — gardening's;
  - `reader`: pages an agent reworded for their reader, which wait for a
    yes ([below](#the-readers-rewordings-wait-for-a-yes)).
- **The same path as a yes**: `workline/accepted` set on the head, auto-merge
  on. The veto is the forge's, as
  [at `cautious`](merge-policy.md#the-veto-is-the-forges).
- **Each one recorded**: a comment on it — no person's yes, its kind,
  why —, and the job's summary, whose first line says which kinds merge
  alone and which wait. Its head's checks still running, the comment
  says it merges once they pass, never before.
- **A revert demotes its kind**
  ([why](adr/0042-the-engine-merges-by-the-projects-policy.md#each-merge-recorded-a-revert-demotes-its-kind)):
  `follow` reads every revert on the base once, opens an issue "workline:
  its `<kind>` merge requests wait for a person's yes again (revert …)",
  and that kind needs the label until a person who may push and merge
  **closes the issue**; on GitHub, `follow` runs at once on that close.
  A revert it cannot tie to a pull request holds every kind.
- **What it read is the App's mark** on the base, a status only the App's
  counts. None found, while its history holds a merge workline made
  alone: every kind waits, in an issue, until such a person closes it.
  None found and no such merge on the base's last 1000 commits — a first
  install —: the App marks the base read, and nothing waits; a merge
  request whose comments it cannot read counts as such a merge
  ([#1018](https://github.com/JN0V/workline/issues/1018)).
- **Never alone**: the release tool's, a fork's, a person's, another
  bot's, a Draft, one changing a guarded file
  ([below](#what-never-merges-alone)); with a person's token, nothing.
- **The governance line**: a change an AI wrote and an AI reviewed merges
  with no person's approval — past the two-person rule of SLSA and SOC 2.
- **The App's key, out of a pull request's own run**: none opens a
  `workline/` pull request as the App ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).
- **Tried on workline's own repository first**
  ([#322](https://github.com/JN0V/workline/issues/322); the
  [criteria](adr/0042-the-engine-merges-by-the-projects-policy.md#the-veto-what-never-merges-alone-the-criteria)).

## What never merges alone

- **The files that define oversight and authority**, listed in the
  project's `CODEOWNERS`: a pull request changing one waits for a
  person's `workline:accepted`, whatever its kind
  ([ADR-0068 §4](adr/0068-the-line-steps-back-on-its-own-drift.md#4-the-guard-is-guarded)).
  workline's own list: [CODEOWNERS](../CODEOWNERS).
- **One list**: the engine holds every path `CODEOWNERS` gives an owner
  (a folder named with or without a trailing `/`, all of it), and a
  `CODEOWNERS` file itself, added or deleted, nothing else. It reads the first found:
  `CODEOWNERS`, `.github/`, `docs/`, `.gitlab/`; on the base, never the
  pull request's own copy.
- **Other role prompts take the normal path**: the reviewer reads them,
  the auditor samples the acts.
- **With no `CODEOWNERS`**: the files that steer agents —
  [AGENTS.md](../AGENTS.md), CLAUDE.md and their likes, the pages they
  link to or import. `workline doctor` says which list it reads.
- **GitHub's required code-owner review stays off** with one owner: an
  author cannot approve their own pull request. A second owner may turn
  it on.
- **CI and workline's config** are held apart, whatever the list says:
  never workline's routine docs, never the developer's
  ([its bounds](../roles/developer/docs/bounds.md)).

## The developer's code: a fix alone, a feature with a yes

- **A fifth kind, `code`**: the developer's pull request for an issue
  its labels call a fix, every gate passed and no finding of
  the reviewer's open that blocks or is Important; findings it cannot
  read from the reviewer's record hold it.
- **A fix, once the reviewer read its head**: the reviewer's summary
  names the pull request's head, read by its lenses, or its record
  covers a head rebased since, or the engine's own clean merge of the
  base since; else it waits for the reviewer, 24 h
  after the push where the line runs it, then for a person's yes, said
  on it
  ([ADR-0065](adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence),
  [the detail](../roles/developer/docs/merge.md#what-each-gets-at-normal)).
- **A fix, once its bug was reproduced**: its code starts from the
  tester's Draft, whose tests failed on the base; else it waits for a
  person's yes, said once on it
  ([ADR-0060](adr/0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind)).
- **A feature or a task waits for a person's yes**, said once on it: a
  `task` is a label an agent may put
  ([#937](https://github.com/JN0V/workline/issues/937)).
- **Told by the issue it closes**, never by the agent: the rules, the
  labels and the record on [the developer's merge](../roles/developer/docs/merge.md).

## The reader's rewordings wait for a yes

- **The kind `reader`**: the documentalist's
  [reader task](../roles/documentalist/docs/reader.md) has an agent
  reword pages for their reader. Told by the `reader:` line it writes in
  each page's header, never by the branch's name.
- **It waits for a person's yes**, the label `workline:accepted`, as at
  `cautious`: a rewording may change what a page says, and only a person
  reading it sees that
  ([why](adr/0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-10-17-438-an-agents-rewording-waits-for-a-yes)).
- **Said twice**: in the job's summary, and once on the pull request, a
  comment naming the label and the setting below.
- **To let it merge alone**, a project's choice:

  ```yaml
  merge: {autonomy: normal, alone: [reader]}
  ```

  It then merges as the other kinds, recorded; a revert demotes it.

## Set up

- As at `cautious` ([set up](merge-policy.md#set-up)); the App marks each
  base commit it read for reverts with the status `workline/merge-read`.
- **The local forge**: the marks and the demotion issues are kept in the
  clone; a demotion issue you close is lifted
  ([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-11-10-1322-the-local-forge-merges-itself)).
- **Another forge** (`cmd:`): besides the operations `cautious` needs,
  `app`, `merge-state`'s `pushers`, `closing`, `mark`, `marked`,
  `comments`, `comment`, `issue` and `open-issue`
  ([forge command](spec/forge-command.md)).

## What the summary says

As at `cautious` ([each word](merge-policy.md#what-the-summary-says)),
and:

| Said | When |
|---|---|
| `review-to-come` | the developer's code, the reviewer in the line, its verdict not on the head yet: it waits for it, 24 h at most |
| `waits-for-a-yes` | the release tool's; a push, a commit or a page not workline's; a kind demoted; `reader`, not let merge alone; the developer's feature or task, a fix whose bug no test reproduced, a fix no review read at its head, or a finding open |
| `merge-app-unknown` | the token is no App's: nothing merges alone |
| `merge-demoted`, `demotion-unrecorded` | a revert of one merged alone read: its kind demoted, its issue opened or not |
| `revert-unread` | a revert tied to no merge request: every kind demoted |
| `merge-restored`, `restore-ignored`, `restore-unread` | a demotion's issue closed by a person who may lift it; by anyone else; unread |
| `merge-record-lost`, `merge-unmarked` | no mark of the App's on the base's last 100 commits, or none kept: every kind held |
