---
sources: [ci/github, ci/gitlab, ci/forgejo/workline-forge.sh, roles/product-owner/role.yaml, internal/builtin/documentalist/sample.go, internal/sample/apply.go, internal/sample/acts.go, cmd/workline]
checked: 5f21056
verified: agent:claude-code
---
# Setting up workline in CI

Docs are judged on each merge request and by gardening at night
([ADR-0010](adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md)):
both run in CI. What workline writes goes where the project lives
([ADR-0016](adr/0016-writes-go-where-the-project-lives.md)). This page
says what the jobs are; each forge has its page:

- **GitHub**: [ci-github.md](ci-github.md).
- **GitLab**, gitlab.com or self-managed: [ci-gitlab.md](ci-gitlab.md);
  pipelines a tool starts: [gitlab-trigger.md](gitlab-trigger.md).
- **Another forge** (Gitea, Forgejo, Codeberg…) **or none**:
  [ci-other-forges.md](ci-other-forges.md).
- The bare command of each job, for any other trigger:
  [triggers.md](triggers.md).

Each template's header says the same, beside the code.

## The jobs

Each task is a pair: a job that judges, with the agent and read rights,
then one that applies, with the write token and no AI key
([principle 6](PRINCIPLES.md)).

| Task | When | Judges | Applies |
|---|---|---|---|
| merge request | each one | the line on its commits | the fixes, to its branch |
| follow | a push to the default branch | — | the release fix rebuilt on it ([ADR-0034](adr/0034-the-release-fix-follows-its-base.md)) |
| gardening | nightly while a backlog is caught up, then weekly | the `schedule` line | one merge request a task ([ADR-0006](adr/0006-gardening-opens-one-merge-request-per-task.md)) |
| sample | weekly | the docs and acts drawn | an issue comment, a merge request |
| import | by hand | a file's items | the issues |

- **A judge that fails still hands over**: the applying job runs whatever
  the judge's outcome, and the pipeline fails by the judge's. It applies
  what the steps that passed proposed, and what a blocked role says why
  it blocks (`on-block`: the reviewer's comment, #226). An agent out of
  reach (3) warns on both forges.
- **The agent is a choice** (`--ai`): `none`, `claude` (`claude:opus`…),
  or `cmd:` running any other. The templates call Claude when its token is
  set, no agent otherwise: every check still runs, and what needs
  judgement is listed for a person.
- **The version**: the templates run the release named by
  `WORKLINE_VERSION` at their top; update it to take a newer one.
  `workline version` in a job's log says which ran.

## Before any forge

In the repository:

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
3. The reviewer, to have each merge request's code read
   ([ADR-0020](adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)):
   `workline init --review` adds it to the line; each forge page says the
   token it reads with.
4. The agent's token, if you want one: for Claude, a subscription's,
   `claude setup-token` prints it (`sk-ant-oat01-…`).

## What a job shows

Each job's summary — the verdict, each step's verdict and findings, the
agent's calls, what was applied and what waits to be — is written by the
engine (`--summary`,
[ADR-0035](adr/0035-the-engine-writes-the-jobs-summary.md)), for a person
who reads neither the log nor `line.json`:

- **GitHub**: the job's summary page ([ci-github.md](ci-github.md#what-the-jobs-may-do)).
- **GitLab**: an HTML page linked from the job's page, the Markdown at the
  end of the log ([ci-gitlab.md](ci-gitlab.md#what-the-template-does)).
- **Another CI**: pass `--summary` to `route`, `apply` or `sample`, as
  `.md` or `.html`, whichever your CI shows.

## The product owner

To keep the backlog too
([ADR-0018](adr/0018-the-product-owner.md)), add it to the schedule line;
gardening then runs it after the documentalist:

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

What it does, its acts and settings: [its page](../roles/product-owner/README.md).
In short:

- its acts and proposals go to one issue, "Backlog — product owner", which
  opens with what is next and what is stuck past `stuck-days` (14)
  ([ADR-0031](adr/0031-the-report-opens-with-what-is-next-and-what-is-stuck.md));
- it refines issues to `ready`, drafts labelled `workline:draft`; the label
  `workline:accepted`, on one issue or many from the list, moves them to
  `ready` at the next run, with no agent;
- an outsider's issue gets the sections proposed in a comment; the same
  label agrees for the project, and a reply `agreed`, by its reporter or a
  person of the project, has them written
  ([ADR-0021](adr/0021-the-product-owner-talks-with-the-reporter.md)); it
  asks again only after an answer, three times at most;
- an issue form with the four sections,
  [ci/github/issue-form-need.yml](../ci/github/issue-form-need.yml), copied
  to `.github/ISSUE_TEMPLATE/need.yml`, has a person's issue arrive with
  them written;
- it also splits, renames, orders and says what an issue waits on
  ([ADR-0022](adr/0022-a-split-keeps-the-need-a-rename-keeps-a-persons-title.md),
  [ADR-0028](adr/0028-an-issue-names-what-it-waits-on.md)): a person's
  title, priority or link is kept.

### Importing a file

A roadmap or backlog file is moved to issues once: by hand, `workline
issues import <file>`, then `--apply`; or in CI, split as the other roles
are ([ADR-0030](adr/0030-an-import-maps-every-item-to-its-issue-or-why-not.md)):

- **Judge**: `workline issues import <file> --json > line.json` — the
  agent, a read token; writes nothing; the map says what would be opened;
  `pending` lists its runs.
- **Apply**: `workline apply --line line.json` — the write token, no
  agent; opens them.
- Needs the first release after v0.17.0. A map with items not covered ends
  the judge 2: the apply still opens what was judged.
- The jobs: [GitHub](ci-github.md#importing-a-file),
  [GitLab](ci-gitlab.md#the-product-owner).

## The weekly sample

Every week, one in ten of the docs the documentalist vouched for — whose
`checked` it moved — is read whole against its sources, at the commit
`checked` names, by a judge apart from the model that vouched
([ADR-0014](adr/0014-checked-is-earned-by-what-was-read.md), step 4;
[ADR-0015](adr/0015-the-weekly-sample-is-written-on-the-forge.md)).

```yaml
roles:
  documentalist:
    settings:
      sample: {judge: "claude:opus"}   # Sonnet vouches by default: Opus reads
```

- **The judge**: an `--ai` value, here or as the CI variable
  `WORKLINE_JUDGE`; a `cmd:` running another provider's agent stands
  further apart. On workline itself, Opus reads what Sonnet vouched for:
  another model of the same provider, the best independence there.
  Without a judge, the docs drawn are listed for a person.
- **The result**: one issue, "workline: the weekly sample of the docs
  vouched for", a comment a week; a `checked` found false labels it
  `documentalist-step-0` and opens a merge request putting it back, for
  you to review.
- **The cost**: one call of about 8k to 12k tokens a doc; one doc on most
  weeks.
- **`after: <tag>`** beside the judge leaves out what was vouched for
  before that commit: before the release whose engine earns `checked`
  (v0.2.1 on workline), a `checked` could be moved unread. A commit is best
  quoted (`after: "7515148"`): unquoted, one with a leading zero is
  refused, YAML reading it as another number.
- **The week** is the last whole one; for another (`--week 2026-W40`),
  each forge page says how.
- **The product owner's acts**, on a project that runs it
  ([ADR-0033](adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)):
  one in ten of those it did alone in the week, onto the issue "workline:
  the weekly sample of the product owner's acts" — each with its day, its
  level, and whether a person undid it — for you to judge; undo one you
  find wrong. It may suggest another `autonomy` level; nothing to set up,
  no agent asked.
