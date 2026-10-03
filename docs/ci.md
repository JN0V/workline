---
sources: [ci/github, ci/gitlab, Dockerfile, .goreleaser.yaml, .github/workflows/release.yml]
checked: 64eee70
judged: a0f1e8a
---
# Setting up workline in CI

Docs are judged on each merge request and by gardening at night (ADR-0010):
both run in CI. This page sets it up on GitHub, on gitlab.com and on a
self-managed GitLab, then on another forge, and with none. What workline
writes goes where the project lives (ADR-0016). Each template's header says
the same, beside the code.

What every forge needs first, in the repository:

1. Docs that name their sources: `workline init` on your machine proposes
   them, you review and commit
   ([adopting a repository](../roles/documentalist/push.md#adopting-a-repository)).
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
     gardening, nightly while a backlog is caught up, then weekly;
   - [ci/github/workline-sample.yml](../ci/github/workline-sample.yml): the
     weekly sample of the docs vouched for (below), from the first release
     after v0.2.1.
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
   weekly, on main. **The weekly sample** (below): a second schedule,
   weekly, with the variable `WORKLINE_TASK` set to `sample`. A schedule's
   variables need Settings → CI/CD → Variables, "Minimum role to use
   pipeline variables", at **Maintainer**: a new gitlab.com project allows
   no one, and setting the variable is refused (403, "not authorized to
   set pipeline schedule variables").

   gitlab.com runs a schedule at its own interval, not at the minute
   written: a cron off the hour (`13 17 * * *`) was listed as due at 18:00
   and ran at 18:08. Off the hour or not, expect it within the hour after.
4. **Protect main**: merge requests only, pipelines must succeed.

A fork's merge request runs in the fork, which has neither token: it is
judged without an agent, and nothing is applied or commented.

## The weekly sample

Every week, one in ten of the docs the documentalist vouched for — whose
`checked` it moved — is read whole against its sources, at the commit
`checked` names, by a judge apart from the model that vouched (ADR-0014,
step 4; [ADR-0015](adr/0015-the-weekly-sample-is-written-on-the-forge.md)).
Set the judge, an `--ai` value, in `.workline/config.yaml`:

```yaml
roles:
  documentalist:
    settings:
      sample: {judge: "claude:opus"}   # Sonnet vouches by default: Opus reads
```

or as the CI variable `WORKLINE_JUDGE`; a `cmd:` running another provider's
agent stands further apart when you have one. Without a judge, the docs
drawn are listed for a person. The result goes to one issue, "workline: the
weekly sample of the docs vouched for", a comment a week; a `checked` found
false labels it `documentalist-step-0` and opens a merge request putting it
back, for you to review. On workline itself, Opus reads what Sonnet vouched
for: another model of the same provider, the best independence there.
A read costs one call of about 8k to 12k tokens a doc; one doc on most weeks.

`after: <tag>` beside the judge leaves out what was vouched for before that
commit: before the release whose engine earns `checked` (v0.2.1 on
workline), a `checked` could be moved unread, and was put back since. A
commit there is best quoted (`after: "7515148"`); unquoted, one with a
leading zero is refused, YAML reading it as another number.

The week sampled is the last whole one. For another (`--week 2026-W40`):
on GitHub, run the workflow by hand with its `week` input; on GitLab, set
the variable `WORKLINE_SAMPLE_WEEK` on the schedule, then play it.

## A self-managed GitLab

As on gitlab.com, with what an instance of your own changes. **Not tried yet
on one**: tell us what differs.

- **The image**: jobs run in `ghcr.io/jn0v/workline:<version>`. Runners that
  cannot reach ghcr.io pull it from your registry: copy it there
  (`docker pull`, `docker tag`, `docker push`) and set `WORKLINE_IMAGE` to
  its name, without the tag. The runners need the Docker executor.
- **Your certificate authority**: the jobs add `CI_SERVER_TLS_CA_FILE`, which
  the runner gives when the instance uses its own, to the image's trusted
  ones: git and workline then accept the instance.
- **The address**: workline talks to the API of the instance the job runs
  on, which CI names (`CI_API_V4_URL`); nothing to set, nothing to install.
- **The agent** reaches out: npm's registry, to install Claude Code, and
  Anthropic's API. Without that, set no `CLAUDE_CODE_OAUTH_TOKEN`: the docs
  are listed for a person.
- **Tokens**: project access tokens exist on every tier of a self-managed
  instance.

## Another forge: Gitea, Forgejo, Codeberg…

workline speaks GitHub and GitLab natively; another forge is plugged by a
command that answers its small contract, one JSON request per operation
(docs/spec/forge-command.md). For Forgejo and Gitea, whose API has
GitHub's shape, copy [ci/forgejo/workline-forge.sh](../ci/forgejo/workline-forge.sh)
into the repository and name it in `.workline/config.yaml`:

```yaml
forge: 'cmd:sh ci/forgejo/workline-forge.sh'
```

It needs `curl` and `jq`, and `FORGEJO_URL`, `FORGEJO_TOKEN` (issues and
pull requests, write) and, unless `origin` names it, `FORGEJO_REPO`
(`owner/name`). The engine pushes the branches itself, with the job's git
credentials; the script does the rest. Forgejo Actions reads, largely, GitHub's
workflow syntax: the GitHub templates are a start, `--forge github`
replaced by the command, the GitHub App's steps by `FORGEJO_TOKEN`.
**Not tried yet** on a live instance — the script was run against a mock of
the API only: tell us what differs. For a forge of another shape, write the
command the contract asks.

## No forge

A project with no forge — pushed to `main`, no merge request — keeps what
workline writes in its own clone:

```yaml
forge: local
```

Issues and merge requests are then files under `.git/workline/`,
never committed; gardening's merge requests are local branches, nothing
pushed; `workline issues` lists them, `workline issues show <n>` (or `!<n>`
for a merge request) shows one, and you merge a branch with git. There is no
CI to set up: gardening and the weekly sample run on your machine
(`workline route schedule --open-merge-request`, `workline sample --out
sample.json` then `workline sample --apply sample.json`), and `workline
docs` judges before a release. In CI — `CI`, `GITHUB_ACTIONS` or
`GITLAB_CI` set — the local forge refuses its writes, loud: the job's clone
is thrown away, and what it would hold with it. What writes nothing still
runs, so a project whose config says `local` for its laptops passes
`--forge` to its CI jobs that write.

With `forge: none` (the default), nothing is written to a forge: a write
that needs one — the issue a role opens included — is refused and says so,
naming `forge: local`.
