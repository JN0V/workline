---
sources:
  - cmd/workline/main.go#hooksCmd
  - cmd/workline/main.go#prePushLine
  - roles/committer/role.yaml
  - internal/kit/commitmsg/commitmsg.go#Settings
  - cmd/workline/main.go#routeCmd
  - cmd/workline/main.go#applyCmd
  - cmd/workline/main.go#reviewCmd
  - cmd/workline/main.go#docsCmd
  - cmd/workline/main.go#testCmd
  - cmd/workline/main.go#developCmd
  - cmd/workline/main.go#runRole
  - cmd/workline/tick.go#tickCmd
  - internal/tick/tick.go#Due
  - internal/forge/gitlab.go#gitlab.connect
  - defaults/ci/gitlab/workline.gitlab-ci.yml
checked: 542fc9cb6
verified: agent:claude-code
reader: 2de353fab113
---
# Running each role from any trigger

workline listens for nothing: whatever starts the work — a git hook, a
scheduler, a CI pipeline, an internal tool — runs one command, the same on
a laptop and in a job. The CI templates ([ci.md](ci.md)) are these commands
wrapped; this page gives them bare, and
[more triggers](triggers-more.md) the product owner alone, a release,
the weekly sample, an import and your own scanners. GitLab pipelines started by the API, a
trigger token or another pipeline: [gitlab-trigger.md](gitlab-trigger.md).

## On your machine

| What | Command | Agent |
|---|---|---|
| each commit's message | the global `commit-msg` hook (`workline hooks install --global`) | yours, to rewrite a refused message, 30 s at most (`rewrite: false`, the project's or yours, turns it off) |
| before a push | the global `pre-push` hook, when the project routes `pre-push`; by hand: `workline route pre-push --ai none --input range=origin/main..HEAD` | never |
| docs made suspect | `workline docs` (`--since <rev>`; `--review` with no agent; `--json`, the reason of a refusal staying on stderr) | yours |
| a branch's code | `workline review` (`--base`, `--lenses`, `--json`) | yours |
| a spec, before it is built | `workline review --spec <file>` or `--issue <n>` | yours |
| gardening | `workline route schedule` (`--forge local --open-merge-request` keeps its merge requests as local branches) | yours |
| the backlog | `workline run-role product-owner --event schedule --forge gitlab` | yours |
| a ready issue's tests | `workline test write --issue <n>`; red-first on any change: `workline test check --base <rev>` ([tester](../roles/tester/README.md)) | yours; `check`, never |
| an issue's next step, by its state | `workline route issue --target issue:<n>`: a role's merge request is opened, as in CI ([routing](spec/routing.md#the-line-of-states)) | yours |
| a merge request | `workline route merge-request --target merge-request:<n> --input range=<base>..<head>`; `forge: local`, its branch checked out: no range ([no forge](no-forge.md)) | yours |
| a ready issue's code | `workline develop` (`--issue <n>`; `--merge-request <n>`, a review round) ([developer](../roles/developer/README.md)) | yours |
| a roadmap file to issues | `workline issues import <file>`, then the same with `--apply`; in CI, `--json > line.json`, then `workline apply --line line.json` | yours |
| before tagging a release | `workline route release` | yours |
| what would hold the release now | `workline route release --dry-run` | never |
| the weekly sample | `workline sample --out s.json`, then `workline sample --apply s.json` | the judge |

`--ai none` on any of them runs every check with no agent. With `--forge
gitlab` on a laptop, the token is `GITLAB_TOKEN`, else glab's; the project,
`GITLAB_HOST` and the `origin` remote.

## In CI: one command per event

Each event that writes runs in two jobs
([ADR-0016](adr/0016-writes-go-where-the-project-lives.md),
[principle 6](PRINCIPLES.md)):

- one proposes with the agent and no write token (`--no-apply`);
- one applies with the write token and no AI key;
- `WORKLINE_RUNS_DIR` names where the runs are kept: carry that folder and
  the `--json` result from the first job to the second, at the same path;
- where the merge request's code can change the first job (a GitHub pull
  request's own run), the second runs from the base: `--from-merge-request
  <n> --head <sha>` ([more](commands.md#workline-apply));
- the gates run first, in a job with no write token and no AI key:
  `route <event> --gates-only --json > gates.json`, which the proposing job reads
  (`--gates-from gates.json`) and the second job shows (`apply --gates
  gates.json`, on GitHub; [gates in CI](gates-in-ci.md)).

A run the engine refuses or blocks, given a target and no `--no-apply`,
says why in one comment on its issue or merge request, updated in place;
with `--json`, the reason is also on stderr.

`$ai` below is your choice of agent: `none`, `claude`, `cmd:<command>`
([usage](usage.md)).

### Merge request

The committer, the documentalist, the reviewer if routed; one run at a time:

```sh
workline route merge-request --ai "$ai" --no-apply \
  --forge gitlab --target merge-request:<iid> --branch <source-branch> \
  --input range=<base-sha>..<head-sha> --code-quality gl-code-quality.json --json > line.json
workline apply --line line.json          # second job: the doc fixes as a diff in a comment
```

`--input draft=<true|false>`, on each run: `true` asks no agent of any
step, the AI review waiting for ready. When a merge request is ready and
not reviewed since, the templates add `--input lenses=all`: the reviewer
reads every lens once, on the whole change
([how](../roles/reviewer/docs/ready.md)).

In a merge queue or train, the same line with `--input queue=true` and
the merged result's range runs the gates alone, no agent, no verdict
written ([the commands](merge-queue.md#the-commands-for-another-ci)).
After a merge, `route main --ai none --gates-kept <folder>` on the new
tip, then `apply --main` with a token writing issues and comments; the gates' job
of a merge request reads that folder too
([the commands](main-after-merge.md#the-commands-for-another-ci)).

The templates also read the reviewer's lenses that only warn apart from
the verdict, and the inspector's triage: `--warn-apart` on the line
above, and beside it a job of their own, after the gates' job, its apply
after it ([apart](../roles/reviewer/docs/apart.md)):

```sh
workline route merge-request --warn-only --ai "$ai" --no-apply \
  --forge gitlab --target merge-request:<iid> --branch <source-branch> \
  --input range=<base-sha>..<head-sha> --gates-from gates.json --json > warn.json
workline apply --line warn.json          # its second job: their comments
```

### Gardening

The `schedule` line — the documentalist; the product owner, reviewer,
tester and developer when routed — one merge request per task:

```sh
git checkout -q -B <branch>              # CI checks out a detached HEAD
workline route schedule --ai "$ai" --no-apply --forge gitlab --open-merge-request --json > line.json
workline apply --line line.json
workline follow --gardening --base <branch> --forge gitlab
workline size --forge gitlab
```

`follow --gardening`, with the applying job's token and no agent, tends the
merge requests gardening opened before: rebuilt on the branch's new tip,
or closed when nothing is left to merge
([ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)).
`size` keeps an issue's normal size in one comment
([the command](commands.md#workline-size)).

### An issue's state

With the tester and the developer routed on `issue`, when a state's
label is put on an issue: the role its state wakes, the tester on
`workline:needs-tests`, the developer on `workline:ready`
([the triggers](../roles/developer/docs/triggers.md)):

```sh
workline route issue --target issue:<n> --ai "$ai" --no-apply --forge gitlab --open-merge-request --json > line.json
workline apply --line line.json
workline route reconcile --ai none --forge gitlab --open-merge-request --input reconcile=list --json > due.json
```

The last, hourly, writes nothing: it lists the issues an event missed
(`due`), the developer's merge requests a round is due on (`reviews`)
and the tester's Drafts an objection waits on (`answers`), each to start
as above or below ([on GitLab](gitlab/line-of-states.md)).

### What is due now

`workline tick --json` lists the line's jobs due now, each with why,
no agent, nothing written: a regular beat (a loop, a schedule, a timer)
starts those and nothing else ([the tick](tick.md)). It reads the runs
on GitHub only: elsewhere, GitLab included, it lists only a reconcile
with no time set, and says on stderr that the runs were not read.

### On a role's own merge request

A note on the tester's Draft (an objection), or a verdict, a gate, a
check red or a note on the developer's merge request (a review round);
the run applied from the default branch, as
data ([the tester's jobs](objection-jobs.md),
[the review loop](../roles/developer/docs/review-loop.md#what-wakes-it)):

```sh
workline test answer --merge-request <iid> --ai "$ai" --no-apply --forge gitlab --json > answer.json
workline develop --merge-request <iid> --ai "$ai" --no-apply --forge gitlab --json > round.json
workline apply <run-dir> --from-merge-request <iid> --head <sha> --forge gitlab
```

## What a job needs and gives back

The clone, variables and tools each job needs, GitLab's tokens, the exit
codes and what a run writes: [jobs' needs and outputs](triggers-outputs.md).

## Where this page reads from

- **The commands and their flags**: `cmd/workline/main.go`, and the tick
  in `cmd/workline/tick.go`.
- **The committer's rewrite and its 30 s**: `roles/committer/role.yaml`
  and `internal/kit/commitmsg/commitmsg.go`.
- **What the tick lists**: `internal/tick/tick.go`.
- **GitLab's token on a laptop**: `internal/forge/gitlab.go`.
- **The jobs in CI**: the GitLab template,
  `defaults/ci/gitlab/workline.gitlab-ci.yml`.
- **The line's states these commands move**:
  [routing](spec/routing.md#the-line-of-states).
