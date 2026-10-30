---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: eebe4270
verified: agent:documentalist
judged: e8a46f7
---
# What the GitLab template does

Part of [workline on GitLab CI](../ci-gitlab.md): what each job of
[the template](../../ci/gitlab/workline.gitlab-ci.yml) does, and why.

- `workline:gates` runs your gates alone, before `workline:judge`, and
  writes what they found the change adds as its Code Quality report;
  install their tools in it ([gates in CI](../gates-in-ci.md)). No token:
  `workline:base`, before it, reads the merge base's findings
  ([the environment](environment.md)). In a merge train's
  pipeline, they are workline's only jobs; a merged results pipeline
  runs them all ([a merge train](../merge-queue.md#gitlab-a-merge-train)).
- **The reviewer's lenses that only warn** read apart from the verdict
  ([apart](../../roles/reviewer/docs/apart.md)):
  - `workline:judge` asks those that block (`--warn-apart`; without
    `WORKLINE_GITLAB_TOKEN`, every lens, said in its log and summary).
  - `workline:warn`, with the agent's token and the bot's, asks the
    others (`--warn-only`), and the inspector's triage given the gates'
    `gates.json` ([the triage in CI](../gates-in-ci.md#the-inspectors-triage));
    `workline:warn:apply` posts them, each in a comment of its own.
  - **Both in a child pipeline**, `workline:warn:child`, started after
    `workline:gates` whatever its verdict, with no `strategy`: "merge
    when the pipeline succeeds" waits for `allow_failure` jobs too, never
    for a child
    ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#4-also)).
  - The child gets the merge request's `CI_MERGE_REQUEST_*` variables and
    the tokens scoped to `workline`. `workline:warn` reads `gates.json`
    from the merge request's pipeline with the bot's token: GitLab's
    `needs:pipeline:job` gives no failed job's artifacts.
  - `workline:merge` reads the verdict from `workline:judge` alone.
- **The whole history** (`GIT_DEPTH: 0`, over the project's "Git shallow
  clone" setting, 20 commits on a new project): the documentalist compares
  each doc's sources since its `checked`, the release looks for its last
  tag. A job set too shallow gets `shallow-clone`, never a pass.
- **A fork's merge request** runs in the fork, which has neither variable:
  judged without an agent, nothing applied or commented
  ([its gates](../gates-in-ci.md#a-forks-pull-request)).
- **A judge that fails**: the applying jobs still run (`when: always`),
  the pipeline failed by the judge's job.
  - Gardening: what steps passed before one blocked is applied
    ([#142](https://github.com/JN0V/workline/issues/142)).
  - A role that blocks still posts its comments, the reviewer's as the
    documentalist's ([`on-block`](../spec/role-contract.md#one-run)).
  - An agent out of reach (3) warns (`allow_failure: {exit_codes: [3]}`),
    as on GitHub: a person judges what it would have; a GitLab call is
    tried again first ([which](calls.md)).
- **`workline:follow`**, each push to the default branch, its bot's token,
  no agent: the [merge policy](../merge-policy.md), when set, also after
  each judge (`workline:merge`).
- **`workline:main`**, each push to the default branch, no token, no
  agent: the gates, their findings kept in its artifacts as the next base
  `workline:gates` reads, and the whole repository's checks;
  `workline:main:issue`, the bot's token, keeps what it found in one issue
  ([main after a merge](../main-after-merge.md#gitlab)).
- **`workline:garden:apply`** then rebuilds gardening's earlier merge
  requests, or closes those with nothing left to merge
  ([ADR-0034, amended](../adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)).
- **The summary** ([what a job shows](../ci.md#what-a-job-shows)):
  `workline-summary.html`, linked from the job's page (its annotations)
  and, for a merge request, from the merge request's page (`expose_as`);
  `workline-summary.md` printed at the end of the log. Both kept a day as
  artifacts, a week for the applying jobs, which add what they applied to
  the judging job's. The HTML opens through GitLab Pages,
  on by default on gitlab.com; without Pages, the link downloads it.
  - Tried on gitlab.com (2026-10-06), the engine of the branch: a
    scheduled gardening job, the sample's read, and a gardening job
    started through the pipelines API each printed it, and the job's page
    linked the HTML, which Pages showed.
  - Not tried: a merge request's `expose_as` link, an applying job's page,
    a private project's Pages preview, a self-managed instance.
- **The reviewer** ([ADR-0020](../adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)):
  - the judging job reads its summary comment to review only new commits,
    with `CI_JOB_TOKEN`: the notes alone, never the project's members,
    which that token may not list (401,
    [#552](https://github.com/JN0V/workline/issues/552)); a comment it
    cannot read is said (`record-unread`), and the push reviews the whole
    merge request again;
  - the applying job posts each finding as a discussion on its line, with
    `WORKLINE_GITLAB_TOKEN`
    ([more](../../roles/reviewer/docs/lines.md));
  - a draft's pipelines ask no agent; marking it ready starts no
    pipeline: the next one, a push or **Run pipeline**, has every lens
    read the whole change once, as the draft's summary says
    ([how](../../roles/reviewer/docs/ready.md); not tried on a GitLab yet).
- **The documentalist** reads the records its note keeps instead of
  commits ([more](../../roles/documentalist/docs/push.md#a-doc-found-with-nothing-to-fix-no-commit)):
  - only a note workline's own account wrote: `WORKLINE_GITLAB_TOKEN`'s
    user, as GitLab says, or one named in `gitlab: {bots: [<username>]}`
    in `.workline/config.yaml`
    ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md));
    the reviewer and the inspector read theirs so too;
  - nor one a Maintainer edited last: GitLab says who in GraphQL
    (`lastEditedBy`), which `WORKLINE_GITLAB_TOKEN` may ask and a job's
    own token may not; unasked, no record is read, said
    (`record-unread`);
  - a note carrying its marker that another wrote, a person of the
    project's too: not read, its docs judged again, said
    (`record-unread`);
  - a note it cannot read: it commits the records, said (`record-unread`).
