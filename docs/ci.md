---
sources: [ci/github, ci/gitlab, ci/forgejo/workline-forge.sh, roles/product-owner/role.yaml, cmd/workline]
checked: 1c0f2082
judged: a8574a5
verified: agent:claude-code
reader: 8aab3bd261d9
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
| merge request | each one | the line on its commits | the fixes, to its branch; the doc findings beside their lines |
| follow | a push to the default branch; with the merge policy, after each judge and when a label changes | — | the release fix rebuilt ([ADR-0034](adr/0034-the-release-fix-follows-its-base.md)); when set, people's pull requests in conflict only on workline's lines get the branch merged ([ADR-0041](adr/0041-workline-settles-its-own-lines-when-the-base-moves.md)); auto-merge on for what a person accepted ([merge policy](merge-policy.md)) |
| gardening | nightly while a backlog is caught up, then weekly | the `schedule` line | one merge request a task ([ADR-0006](adr/0006-gardening-opens-one-merge-request-per-task.md)); then those the branch moved under, rebuilt or closed ([ADR-0034](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)) |
| sample | weekly | the docs and acts drawn | an issue comment, a merge request |
| import | by hand | a file's items | the issues |

- **The gates run first, alone**, with no write token and no AI key
  ([gates in CI](gates-in-ci.md)).
- **What only warns runs apart**: the reviewer's lenses that warn read
  in a job of their own beside the judge, posting in a comment of their
  own; the head's verdict waits only for what can block
  ([apart](../roles/reviewer/docs/apart.md)).

- **A judge that fails still hands over**: the applying job runs whatever
  the judge's outcome, and the pipeline fails by the judge's. It applies
  what the steps that passed proposed, and what a blocked role says why
  it blocks (`on-block`: the reviewer's comment, [#226](https://github.com/JN0V/workline/issues/226)). An agent out of
  reach (3) warns on both forges.
- **One judge per merge request at a time**: a push cancels the run still
  judging the head before, and nothing of it is applied; the newer run
  reviews its commits too ([#359](https://github.com/JN0V/workline/issues/359)).
  GitLab needs a project setting ([how](ci-gitlab.md#6-one-pipeline-per-merge-request)).
- **One review at a time**: a push landing while the run before it is
  applied is judged once that apply is done, and reviews only what is new
  ([#375](https://github.com/JN0V/workline/issues/375),
  [ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md#amendment-2026-10-15-the-review-of-a-head-the-branch-moved-on-from)).
- **The write token stays with the base**: on GitHub, a pull request's
  own run can be changed by its branch, so the applying jobs run the
  default branch's workflow and read that run as data
  ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).
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
   docs far behind, the caps on judging in parts, on by default when
   gardening ([ADR-0009](adr/0009-docs-far-behind-are-judged-in-parts.md)):

   ```yaml
   roles:
     documentalist:
       settings:
         # judge-in-parts: false    # docs too large for one call go to a person
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
- **The applying job** writes its own: what it did on each issue, linked,
  grouped as the judge's; what failed, the runs it left unapplied, and the
  command that resumes them.

On the merge request, each finding is beside its line, and none is a
conversation to resolve but the reviewer's comments
([ADR-0047](adr/0047-findings-on-a-merge-request-open-no-conversation.md)):

- **GitHub**: the documentalist's as the annotations of the check
  `workline findings`; the gates' new ones as those of `workline gates`,
  with one summary comment ([gates in CI](gates-in-ci.md)); code scanning
  keeps the committer's.
- **GitLab**: the Code Quality report, a report, not a thread; the gates'
  job writes its own.
- `workline doctor` says whether merging requires every conversation
  resolved.

## The product owner

To keep the backlog too
([ADR-0018](adr/0018-the-product-owner.md)), add it to the schedule line;
gardening then runs it after the documentalist, even when the
documentalist blocks: the job fails all the same
([ADR-0037](adr/0037-the-schedule-runs-every-step.md)).

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

- what it proposes goes on each issue: the label `workline:proposed`
  (GitLab: `workline::proposed`) and its one comment there; a saved
  filter on the label is what waits on you; the night's summary is in
  the job's summary ([ADR-0038](adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md));
- it refines issues to `ready`, drafts labelled `workline:draft`; the label
  `workline:accepted`, on one issue or many from the list, is your yes:
  done at the next run, with no agent; a comment asks it to revise;
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

Once a week, after the gardening, the auditor re-checks one in ten of
every role's acts: what it reads, writes and costs, in
[the weekly sample](weekly-sample.md). Each forge's page says how to
schedule it.
