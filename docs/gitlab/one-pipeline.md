---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: eebe4270
verified: agent:documentalist
judged: e8a46f7
---
# One pipeline per merge request on GitLab

Step 6 of [workline on GitLab CI](../ci-gitlab.md#6-one-pipeline-per-merge-request).

A push cancels the pipeline still judging the merge request's head before
([#359](https://github.com/JN0V/workline/issues/359)): `workline:judge` is
`interruptible`, the jobs that write are not. GitLab does it when:

- **Auto-cancel redundant pipelines** is checked (Settings → CI/CD →
  General pipelines), `workflow:auto_cancel:on_new_commit` left at its
  default, `conservative`;
- your own jobs in that pipeline are `interruptible: true` too: once one
  that is not has started, GitLab cancels nothing
  ([GitLab's docs](https://docs.gitlab.com/ci/yaml/#interruptible)).

One review at a time: `workline:judge` waits while `workline:apply` writes
the review before it, one resource group per merge request, then reviews
only what is new ([#375](https://github.com/JN0V/workline/issues/375));
`workline:warn` and `workline:warn:apply` likewise, in a resource group of
their own.

- **Two applies waiting** may start in either order: GitLab's resource
  groups default to `unordered`, and workline sets nothing there.
- **The comment says which run wrote it**: the reviewer's and the
  inspector's name the head and the pipeline (`CI_PIPELINE_ID`) their
  run read, hidden.
- **An older pipeline's apply, run last**, keeps the newer comment's text
  and merges only its record into it
  ([#598](https://github.com/JN0V/workline/issues/598)).

Not tried on a GitLab yet.
