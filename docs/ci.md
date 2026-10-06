---
sources: [ci/github, ci/gitlab, ci/forgejo/workline-forge.sh, Dockerfile, .goreleaser.yaml, .github/workflows/release.yml, .github/workflows/release-please.yml, release-please-config.json, .github/workflows/workline.yml, .github/workflows/workline-gardening.yml, .github/workflows/workline-sample.yml, roles/product-owner/role.yaml, internal/builtin/documentalist/sample.go, internal/sample/apply.go, internal/sample/acts.go, internal/forge/local.go, internal/forge/gitlab.go]
checked: b196291
verified: agent:claude-code
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
   ([adopting a repository](../roles/documentalist/docs/push.md#adopting-a-repository)).
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
3. The reviewer, if you want each merge request's code read (ADR-0020,
   roles/reviewer): `workline init --review` adds it to the line. The
   judging job reads its summary comment, to review only the commits it
   has not: the GitHub template gives it a read token (`pull-requests:
   read`); on GitLab, CI's job token may not read a merge request's notes,
   so give `GITLAB_TOKEN` (`read_api`), or each push reviews the whole
   merge request again, said (`record-unread`).
4. A Claude subscription's token, for Claude to judge: `claude setup-token`
   prints one (`sk-ant-oat01-…`). Without it, every check still runs and the
   docs needing judgement are listed for a person.

The templates run a release of workline, `WORKLINE_VERSION` at their top:
update it to take a newer one. `workline version` in a job's log says which
ran.

workline's own workflows (.github/workflows/) are the templates, but for
the engine: they build it from the commit they run on rather than run a
release — gardening and the sample, both jobs, from `main`; a pull
request's `judge`, from the pull request's commit. A project never runs, in the checks of a commit, the pin that
commit ships: a pull request moving the templates to a version not yet
tagged would download nothing, and its check go red, as on each release
from v0.2.0 to v0.2.3 (ADR-0017). The pull request's `apply`, which holds
the write token, runs the last release of workline holding its engine,
looked up when it runs: a change to the engine never runs with that token
before it is released, and no pin of its own waits for the tag.

**How workline itself releases** (ADR-0017): with release-please, and no
tag by hand. On each push to `main`, release-please.yml, with the GitHub
App's token, keeps one pull request open, "chore(main): release X": the
next version from the conventional commits since the last tag (`feat`
moves the minor while workline is at 0.x, as a breaking change does),
CHANGELOG.md, and the templates' `WORKLINE_VERSION`, each on a line marked
`x-release-please-version` (release-please-config.json). The App's token,
not the job's: a pull request opened with `GITHUB_TOKEN` starts no
workflow, so its checks would never run. On that pull request, workline.yml
holds it as the release (ADR-0017): the judging job passes the branch it
comes from (`--branch`), a release tool's, and the documentalist holds it
on every doc made suspect since the last release; its fix goes to a pull
request of its own on `main`, which release-please brings in once merged,
and which workline.yml's `follow` job rebuilds on `main` at each push there,
as release-please does its own (ADR-0034) — that job builds the engine
from `main`'s commit, merged and reviewed.
The person merges it; that is the decision. release-please then tags the merged commit, `vX.Y.Z`, and writes
the GitHub release with its notes; in the same run, release.yml builds the
binaries with GoReleaser, which uploads them to that release and keeps its
notes (`release.mode: keep-existing`), and pushes the image. A failed
publish is run again from the Actions page, never tagged again: a tag
pushed is never moved (the Go proxy keeps its first commit).

## GitHub

1. **Copy the workflows** into `.github/workflows/`:
   - [ci/github/workline.yml](../ci/github/workline.yml): each pull request,
     and, on a push to `main`, `workline follow` (ADR-0034): the
     documentalist's merge request for the release, when one is open, is
     rebuilt on `main`'s new tip, so a pull request merged meanwhile that
     touched the same doc leaves it mergeable; one a person committed to is
     left alone. Rename `main` in its `push:` for another default branch;
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
4. **Protect main** (Settings → Rules): pull requests only, the `judge`
   check required (workline.yml's verdict). The review is on the pull
   request (ADR-0011): an agent never merges.
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
   - `WORKLINE_GITLAB_TOKEN`: a project access token — a bot user of this
     project only (ADR-0023) —, role Developer, scopes `api` and
     `write_repository`. One command makes it and stores it, never shown
     (glab, logged in as a Maintainer):

     ```sh
     glab token create workline --repo <group/project> --access-level developer \
       --scope api --scope write_repository --duration 8760h \
       | glab variable set WORKLINE_GITLAB_TOKEN --masked --repo <group/project>
     ```

     Or Settings → Access tokens → Add new token, the same role and
     scopes, then the variable. It expires within a year: rotate it the
     same way (`glab variable update`). Where project access tokens are not
     offered (GitLab documents them as Premium on gitlab.com; one was made
     on a Free user's project, 2026-10-05), a personal access token, `api`
     and `write_repository`, of an account made for the line, added to the
     project as Developer. The job's own token (`CI_JOB_TOKEN`) reaches
     no issue: it is never enough.
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

The template's jobs fetch the whole history (`GIT_DEPTH: 0`, over the
project's "Git shallow clone" setting, 20 commits on a new project): the
documentalist compares each doc's sources since the commit its `checked`
names, and the release looks for its last tag. A job that sets its own
`GIT_DEPTH` too shallow for that gets `shallow-clone`, never docs found
suspect or passed for it.

A fork's merge request runs in the fork, which has neither token: it is
judged without an agent, and nothing is applied or commented.

On each push to the default branch, `workline:follow` rebuilds the
documentalist's merge request for the release, when one is open, on the
branch's new tip, with `WORKLINE_GITLAB_TOKEN` and no agent (ADR-0034).

## What a job shows

Each job's summary — the verdict, each step's verdict and findings, the
agent's calls, what was applied and what waits to be — is written by the
engine (`--summary`, [ADR-0035](adr/0035-the-engine-writes-the-jobs-summary.md)),
for a person who reads neither the log nor `line.json`:

- **GitHub**: the job's summary page (`$GITHUB_STEP_SUMMARY`); a fork's
  pull request gets it as a comment.
- **GitLab**: `workline-summary.html`, linked from the job's page (its
  annotations) and, for a merge request, from the merge request's page
  (`expose_as`); `workline-summary.md` printed at the end of the log. Both
  kept a day as artifacts, a week for the applying jobs, which add what
  they applied to the judging job's.
- **Another CI**: pass `--summary` to `route`, `apply` or `sample`, as
  `.md` or `.html`, whichever your CI shows.

The HTML opens in the browser through GitLab Pages, on by default on
gitlab.com; without Pages, the link downloads it.
Tried on gitlab.com (2026-10-06), the engine of the branch: a scheduled
gardening job, the weekly sample's read, and a gardening job started
through the pipelines API ([gitlab-trigger.md](gitlab-trigger.md)) each
printed it, and their job's page linked the HTML, which Pages showed. Not
tried: a merge request's `expose_as` link, an applying job's page, a
private project's Pages preview, a self-managed instance.

## The product owner

To keep the backlog too (ADR-0018), add it to the schedule line in
`.workline/config.yaml`; gardening then runs it after the documentalist:

```yaml
routing:
  events:
    schedule: [documentalist, product-owner]
roles:
  product-owner:
    settings:
      issues-per-run: 8        # issues read a night
      # autonomy: cautious     # a person is the Product Owner: what sets direction is proposed (ADR-0026)
      acts:
        close-duplicate: {mode: propose}   # a person closes, until trust is earned
```

The GitHub template gives its jobs the issues they need (read when judging,
write when applying); on GitLab, `WORKLINE_GITLAB_TOKEN`'s `api` scope
covers them — a project that runs only the product owner may give its
token the Reporter role and `api` alone; not Planner, which may not give a
split's child its parent. Who is of the project there is a member with the
Planner role or above; a Guest's issue, or a stranger's, is its reporter's
(ADR-0023).
Its acts and proposals are listed in one issue, "Backlog — product owner",
which opens with what is next — the first ready issues of the order — and
what is stuck — each issue waiting on a person past `stuck-days` (14),
with since when (ADR-0031).
A roadmap or backlog file is moved to issues once, by hand:
`workline issues import <file>`, then `--apply`.

It refines issues to `ready`, drafting their Need and Validation, and
labels them `workline:draft`. To accept the drafts, set the label
`workline:accepted` — on an issue, or on many at once from the list of
issues: the next run moves them to `ready`, with no agent. An outsider's
issue gets the sections proposed to its reporter in a comment instead; the
same label agrees for the project, and the next run writes them and moves
it to ready; a reply `agreed`, by its reporter or a person of the project,
has them written, ready still the label's (ADR-0021).
It asks a reporter again only after an answer, three times at most, then
lists the issue in the report. An issue form
with the four sections, [ci/github/issue-form-need.yml](../ci/github/issue-form-need.yml),
copied to `.github/ISSUE_TEMPLATE/need.yml`, has a person's issue arrive
with what they know already written.

It splits an issue too big to be one need into issues of their own,
linked to it — sub-issues on GitHub, tasks on GitLab, a task list in its
body elsewhere — and renames a title that says nothing; a title a person
set after its own is kept (ADR-0022). As the parts close, it says on the
parent, with no agent, what each delivered and which items of its
Verification are proved; a person accepts it by closing it (ADR-0029).

It says what an issue waits on — a split's second part on its first, an
issue that builds on another's code — in GitHub's issue dependencies, or,
on GitLab Free, which keeps `blocks` links for Premium, a line `Blocked by
#12.` in the issue's body; a link or a line a person set is read and kept.
An issue waiting on an open one is ordered after it and never offered
first; each run says the first ready issue that waits on nothing
(ADR-0028).

It orders the backlog: a priority label, `workline:priority/1` (the most
pressing) to `/4`, on the issues it reads — one a person set is kept —
and an issue whose milestone's release is tagged moved to the next open
milestone. A run moves at most `moved-percent-max` of the open issues —
20% at `normal`, 10% `cautious`, 30% `enterprising`; the report lists them as they were, to put back.

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

On a project with the product owner, the same job draws its acts too
(ADR-0033): one in ten of those it did alone in the week, from the record
on its report, onto the issue "workline: the weekly sample of the product
owner's acts" — each with its day, the level it was done at, and whether a
person undid it — for you to judge; undo one you find wrong. It may
suggest another `autonomy` level from the acts undone; nothing to set up,
no agent asked.

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
pull requests, write; its user an administrator of the repository for the
product owner, which asks who of a comment's authors may write) and, unless `origin` names it, `FORGEJO_REPO`
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
