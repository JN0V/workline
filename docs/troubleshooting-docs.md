---
sources: [internal/builtin/documentalist/documentalist.go, cmd/workline/main.go, internal/builtin/documentalist/kept.go, internal/builtin/documentalist/releasekept.go]
checked: 1c9912ae
verified: agent:claude-code
---
# Troubleshooting the docs

What the documentalist says when docs wait, and what to do. Part of
[troubleshooting](troubleshooting.md); the role:
[the documentalist](../roles/documentalist/README.md).

## Docs suspect

`docs suspect: N` at the push counts the docs whose `sources` the pushed
commits touched; it never asks and never blocks. They are judged on the
merge request, by gardening, or when you run `workline docs`. To clear one
by hand: read it against its sources, fix it, and set its `checked` to the
commit you read them at. A doc that names no `sources` is never suspect:
`workline init` proposes them.

On a release, a doc made suspect since the last release holds it (`due`,
`suspect`) until judged.

`record-unread` (a warning): the documentalist's comment keeping the
records of docs found with nothing to fix, of the merge requests merged,
or of the heads a release pull request's fix named, could not be read:
the records are committed, or those docs judged again, at night. On GitLab, give the judging job `GITLAB_TOKEN`
([more](../roles/documentalist/docs/push.md#a-doc-found-with-nothing-to-fix-no-commit)).

`members-unread` (a warning, once a run): the token may not list the
project's members, as GitLab's `CI_JOB_TOKEN`. That comment is read only
when a bot wrote it; one a person's token wrote is not, and its docs are
judged again. `GITLAB_TOKEN` (`read_api`) on the judging job tells a
person of the project too. A service account holding workline's token
is a bot once named: `gitlab: {bots: [<username>]}`
([more](../roles/documentalist/docs/push.md#a-doc-found-with-nothing-to-fix-no-commit)).

`shallow-clone`: the clone lacks history the documentalist needs — the
last release, or the commit a doc's `checked` names, so whether its
sources changed cannot be told. It holds the release, and any run whose
task is such a doc; `workline doctor` warns too. Fetch the whole history:
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
