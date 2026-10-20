---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, Dockerfile]
checked: a88dc872
judged: c9d96e2
verified: agent:claude-code
judged-in-parts: 220b6dc
---
# workline on a self-managed GitLab

As on [gitlab.com](ci-gitlab.md#gitlabcom), with what an instance of your
own changes. **Not tried yet on one**: tell us what differs.

- **The image**: jobs run in `ghcr.io/jn0v/workline:<version>`. Runners
  that cannot reach ghcr.io pull it from your registry: copy it there
  (`docker pull`, `docker tag`, `docker push`) and set `WORKLINE_IMAGE` to
  its name, without the tag. The runners need the Docker executor.
- **Your certificate authority**: the jobs add `CI_SERVER_TLS_CA_FILE`,
  which the runner gives when the instance uses its own, to the image's
  trusted ones: git and workline then accept the instance.
- **The address**: workline talks to the API of the instance the job runs
  on (`CI_API_V4_URL`); nothing to set, nothing to install.
- **The agent** reaches out — Claude Code, for one, needs npm's registry
  and Anthropic's API. Without that, set no `CLAUDE_CODE_OAUTH_TOKEN`: the
  docs are listed for a person.
- **Tokens**: project access tokens exist on every tier of a self-managed
  instance.
- **The reviewer when a merge request becomes ready**: before GitLab
  17.10, no `CI_MERGE_REQUEST_DRAFT`, so the merge request is read from
  the API, before the agent is installed, none on a draft; a job token
  that reads no notes leaves one lens, said in the
  log, unless the job has `GITLAB_TOKEN`
  ([how](../roles/reviewer/docs/ready.md)).
- **One pipeline per merge request at a time**: the same project setting
  as on gitlab.com, **Auto-cancel redundant pipelines**
  ([how](ci-gitlab.md#6-one-pipeline-per-merge-request)).
- **The [merge policy](merge-policy.md)**: a merge request is set to merge
  when its pipeline succeeds by the merge API, both its names sent —
  `auto_merge`, and the older `merge_when_pipeline_succeeds` an instance
  from before it reads.
