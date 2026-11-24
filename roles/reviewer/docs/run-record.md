---
sources: [roles/reviewer/role.yaml, roles/reviewer/prompts/instruction.md, roles/reviewer/prompts/policy.md, roles/reviewer/prompts/lenses, roles/reviewer/code]
checked: e81df7b34
verified: agent:claude-code
---
# Reviewer — the record

Part of [a run on code](run.md): step 3, what a review answered whole is not asked again.

## 3. The record

**The record.** Commits a review answered whole are not asked again.

- It is kept in the summary comment on a merge request, read only from the one workline's own account wrote and no one else edited last ([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)),
  `.git/workline/reviewer-record` on a machine. One that does not read, or only another's carrying its marker, or one another edited, is said (`record-unread`, [#302](https://github.com/JN0V/workline/issues/302)): every commit reviewed again, once, as a whole review writes it anew.
- **A change to what its own pipeline runs** (`.github/`,
  `.gitlab-ci.yml`, `.workline/`, and `pipeline-paths`), by any of its
  commits: no commit is skipped on the record, nor left past five a
  push; every one is read again, at each push, said (`line-changed`).
  Such a merge request runs its own workflow, which wrote what the
  record says.
- **`pipeline-paths`** (default none): what else the pipeline runs
  scripts from, `[ci/, scripts/ci/]` say. Set it in
  `.workline/config.yaml`, under `roles: {reviewer: {settings: …}}`.
- Beside it on a merge request, a line per finding judged, every
  run's, for the weekly sample; kept even when the record does not
  read ([more](verdict.md#every-finding-judged-kept-for-the-weekly-sample)).
- And a line of the tester's answers a judge weighed, so the same
  reason is weighed once ([more](tester-answers.md)).

### Rebases, merges and pushes

- **A rebase** ([#286](https://github.com/JN0V/workline/issues/286)): it
  holds each change's patch id too, so a commit rebased with its change
  intact is not asked again. One a conflict changed is, and those after
  it.
- **A merge of the base**
  ([#134](https://github.com/JN0V/workline/issues/134)): a lens is
  given what the merge settled, both sides of a conflict and the
  resolution; never the base's code. Nothing settled: nobody asked.
- New commits that change no code are recorded, nobody asked, the turn
  of the lenses kept.
- **A lens that does not answer** owes the commits: the record names
  it (`failed=`), holds none of them, and the next run asks it first,
  whosever turn it is ([#588](https://github.com/JN0V/workline/issues/588)).
- **A push before the last review is written** waits for it, and reads it;
  a review written after another keeps its commits and lines ([#375](https://github.com/JN0V/workline/issues/375)).
  The comment names the head and the CI run it is at, hidden: an older
  run applied last keeps the newer one's text, its record merged in
  ([#598](https://github.com/JN0V/workline/issues/598)).
- **What blocked stands**, recorded too, until a commit answers it
  ([#326](https://github.com/JN0V/workline/issues/326), [more](verdict.md#a-block-stands-until-a-commit-answers-it)).
- The run on ready reads the whole change, commits reviewed before too ([at ready](run.md#every-lens-once-when-it-becomes-ready));
  after it, a lens that warns reads five new commits a push ([more](ready.md#after-ready-only-what-is-new)).
- Why, and what git it needs: [ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-12).
