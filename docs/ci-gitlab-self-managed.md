---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, Dockerfile]
checked: 2b82821
verified: agent:claude-code
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
