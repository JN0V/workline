---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:apply
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:ready
  - internal/forge/gitlab.go#gitlab.connect
  - internal/forge/gitlab.go#gitlab.Notes
  - internal/forge/gitlab.go#gitlab.botNamed
  - internal/forge/gitlab.go#gitlab.editors
  - internal/forge/own.go#gitlab.own
  - internal/forge/own.go#ReadKept
  - internal/forge/own.go#Kept.Last
  - internal/builtin/reviewer/record.go#loadRecord
  - roles/inspector/code/record.go#loadRecord
  - internal/builtin/documentalist/kept.go#stickyOn
  - internal/builtin/documentalist/kept.go#othersKept.findings
  - internal/builtin/documentalist/kept.go#readKept
checked: e2e165c2d
verified: agent:claude-code
---
# The records read back on GitLab

Part of [what the GitLab template does](template.md): how the reviewer
and the documentalist read back what their comments keep.

- **The reviewer** ([ADR-0020](../adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)):
  - the proposing job reads its summary comment to review only new commits,
    with `GITLAB_TOKEN` (`read_api`): the notes alone, never the
    project's members, which a token may not list (401,
    [#552](https://github.com/JN0V/workline/issues/552)); the job's own
    token cannot ask who last edited it; a comment it cannot read is said
    ([`record-unread`](../../roles/reviewer/docs/run.md#3-the-record)),
    and the push reviews the whole merge request again;
  - the applying job posts each finding as a discussion on its line, with
    `WORKLINE_GITLAB_TOKEN`
    ([more](../../roles/reviewer/docs/lines.md));
  - a draft's pipelines ask no agent; marking it ready starts no
    pipeline: the next one, a push or **Run pipeline**, has every lens
    read the whole change once, as the draft's summary says
    ([how](../../roles/reviewer/docs/ready.md); not tried on a GitLab yet).
- **The documentalist** reads the records its note keeps instead of
  commits ([more](../../roles/documentalist/docs/push-records.md)):
  - only a note workline's own account wrote: `WORKLINE_GITLAB_TOKEN`'s
    user, as GitLab says, or one named in `gitlab: {bots: [<username>]}`
    in `.workline/config.yaml`
    ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md));
    the reviewer and the inspector read theirs so too;
  - nor one a Maintainer edited last: GitLab says who in
    GraphQL (`lastEditedBy`), which `GITLAB_TOKEN` may ask and a job's
    own token may not; unasked, no record is read, said
    (`record-unread`);
  - a note carrying its marker that another wrote, a person of the
    project's too: not read, its docs judged again, said
    (`record-unread`);
  - a note it cannot read: the records go in its comment's diff, said
    (`record-unread`).
