---
sources: [ci/github, ci/gitlab, Dockerfile, .goreleaser.yaml, .github/workflows/release.yml]
checked: 64eee70
---
# Setting up workline in CI

Docs are judged on each merge request and by gardening at night (ADR-0010):
both run in CI. This page sets it up on GitHub, on gitlab.com and on a
self-managed GitLab. Each template's header says the same, beside the code.

What every forge needs first, in the repository:

1. Docs that name their sources: `workline init` on your machine proposes
   them, you review and commit (README, "Adopt a repository").
2. `.workline/config.yaml`, if the defaults do not suit — for a backlog of
   docs far behind, judging in parts and its caps:

   ```yaml
   roles:
     documentalist:
       settings:
         judge-in-parts: true       # docs too large for one call (ADR-0009)
         parts-max-per-run: 8       # one such doc a night
         ai-max-tokens: 400000      # what a run may spend, all calls together
         max-open-merge-requests: 3 # gardening waits while this many wait
   ```
3. A Claude subscription's token, for Claude to judge: `claude setup-token`
   prints one (`sk-ant-oat01-…`). Without it, every check still runs and the
   docs needing judgement are listed for a person.

The templates run a release of workline, `WORKLINE_VERSION` at their top:
update it to take a newer one. `workline version` in a job's log says which
ran.

## GitHub

1. **Copy the workflows** into `.github/workflows/`:
   - [ci/github/workline.yml](../ci/github/workline.yml): each pull request;
   - [ci/github/workline-fork.yml](../ci/github/workline-fork.yml): comments
     a fork's pull request, which gets no secret and no right to write;
   - [ci/github/workline-gardening.yml](../ci/github/workline-gardening.yml):
     gardening, nightly while a backlog is caught up, then weekly.
2. **The Claude token**: Settings → Secrets and variables → Actions → New
   repository secret, `CLAUDE_CODE_OAUTH_TOKEN`.
3. **A GitHub App** commits the fixes, so the checks run again by
   themselves (with the job's own token, GitHub waits for someone to approve
   them). Your settings → Developer settings → GitHub Apps → New GitHub App:
   no webhook; repository permissions **Contents**, **Pull requests** and
   **Issues** in read and write. Generate a private key, then install the
   App on the repository. In the repository: variable `WORKLINE_APP_ID` (the
   App's ID), secret `WORKLINE_APP_PRIVATE_KEY` (the whole `.pem`).
   Without an App, gardening needs Settings → Actions → General → "Allow
   GitHub Actions to create and approve pull requests".
4. **Protect main** (Settings → Rules): pull requests only, the `workline`
   check required. The review is on the pull request (ADR-0011): an agent
   never merges.
5. **A private repository** without GitHub Advanced Security: remove the
   `upload-sarif` step of workline.yml, code scanning being paid there.

Then open a pull request changing code a doc describes: the job judges it,
and the App commits the fix to its branch, `Workline-Role: documentalist`.

## gitlab.com

1. **Include the template**: copy
   [ci/gitlab/workline.gitlab-ci.yml](../ci/gitlab/workline.gitlab-ci.yml)
   to `ci/` and add to `.gitlab-ci.yml`:

   ```yaml
   stages: [test, deploy]
   include: {local: ci/workline.gitlab-ci.yml}
   ```
2. **The tokens**, Settings → CI/CD → Variables, both **masked** and **not
   protected** — GitLab gives a protected variable only to protected
   branches, never to a merge request's:
   - `WORKLINE_GITLAB_TOKEN`: scopes `api` and `write_repository`. A project
     access token (a bot user) where the plan allows it; else a personal one.
   - `CLAUDE_CODE_OAUTH_TOKEN`: the Claude token.

   Anyone who may push a branch can read them from a pipeline they change:
   people who may already write to the repository.
3. **Gardening**: Build → Pipeline schedules → New schedule, nightly or
   weekly, on main.
4. **Protect main**: merge requests only, pipelines must succeed.

A fork's merge request runs in the fork, which has neither token: it is
judged without an agent, and nothing is applied or commented.

## A self-managed GitLab

As on gitlab.com, with what an instance of your own changes. **Not tried yet
on one**: tell us what differs.

- **The image**: jobs run in `ghcr.io/jn0v/workline:<version>`. Runners that
  cannot reach ghcr.io pull it from your registry: copy it there
  (`docker pull`, `docker tag`, `docker push`) and set `WORKLINE_IMAGE` to
  its name, without the tag. The runners need the Docker executor.
- **Your certificate authority**: the jobs add `CI_SERVER_TLS_CA_FILE`, which
  the runner gives when the instance uses its own, to the image's trusted
  ones: git, workline and glab then accept the instance.
- **The address**: glab talks to the instance the job runs on
  (`GITLAB_HOST: $CI_SERVER_URL`); nothing to set.
- **The agent** reaches out: npm's registry, to install Claude Code, and
  Anthropic's API. Without that, set no `CLAUDE_CODE_OAUTH_TOKEN`: the docs
  are listed for a person.
- **Tokens**: project access tokens exist on every tier of a self-managed
  instance.
