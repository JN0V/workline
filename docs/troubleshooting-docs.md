---
sources: [internal/builtin/documentalist/documentalist.go, cmd/workline/main.go, internal/builtin/documentalist/kept.go]
checked: b6085f38
verified: agent:claude-code
---
# Troubleshooting the docs

What the documentalist says when docs wait, and what to do. Part of
[troubleshooting](troubleshooting.md); the role:
[the documentalist](../roles/documentalist/README.md).

## Docs suspect

`docs suspect: N` at the push counts the docs whose `sources` the pushed
commits changed: a file the doc names, or a line holding a name it
writes in backticks or a value it states; any line for a doc writing no
name in backticks
([ADR-0014, amended](adr/0014-checked-is-earned-by-what-was-read.md#amendment-2026-10-21-suspect-only-when-a-line-it-names-changes));
it never asks and never blocks. They are judged on the
merge request, by gardening, or when you run `workline docs`. To clear one
by hand: read it against its sources, fix it, and set its `checked` to the
commit you read them at. A doc that names no `sources` is never suspect:
`workline init` proposes them.

`record-unread` (a warning): the documentalist's comment keeping the
records of docs found with nothing to fix, or of the merge requests
merged, could not be read:
the records are committed, or those docs judged again, at night. On GitLab, give the judging job `GITLAB_TOKEN`
([more](../roles/documentalist/docs/push.md#a-doc-found-with-nothing-to-fix-no-commit)).

- **Its marker in another's comment**: only workline's own account's
  comment is read; the warning names whose was not, and the account read.
- **Not the account you expect**: on GitHub, give the judging jobs the
  App's id, `WORKLINE_APP_ID` (the templates do); on GitLab, name a
  service account holding workline's token, `gitlab: {bots:
  [<username>]}`
  ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)).

`shallow-clone`: the clone lacks history the documentalist needs — the
commit a doc's `checked` names, so whether its sources changed cannot be
told. It holds any run whose task is such a doc; `workline doctor` warns too. Fetch the whole history:
`git fetch --unshallow`; in CI, GitHub's `fetch-depth: 0`, GitLab's
`GIT_DEPTH: 0` (workline's templates set both).

## Gardening paused

`gardening-paused`: as many of the documentalist's merge requests wait for
review as `max-open-merge-requests` allows (5). Review or close them;
gardening proposes again when fewer wait.

`merge-request-unread` (a warning): the branch of one of its merge
requests could not be fetched from `origin`. The documentalist then knows
that merge request by its task only: a doc it holds may be asked again;
for one judging docs in parts (`fix-…`), every doc to judge in parts
waits. Give the job read access to the repository's branches.
