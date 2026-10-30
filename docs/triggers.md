---
sources: [cmd/workline, internal/line, internal/hooks, internal/forge/gitlab.go, ci/gitlab/workline.gitlab-ci.yml, ci/github/workline.yml, ci/github/workline-gardening.yml, ci/github/workline-sample.yml, routing.default.yaml]
checked: 99a14dc0
verified: agent:claude-code
reader: 2de353fab113
---
# Running each role from any trigger

workline listens for nothing: whatever starts the work — a git hook, a
scheduler, a CI pipeline, an internal tool — runs one command, the same on
a laptop and in a job. The CI templates ([ci.md](ci.md)) are these commands
wrapped; this page gives them bare. GitLab pipelines started by the API, a
trigger token or another pipeline: [gitlab-trigger.md](gitlab-trigger.md).

## On your machine

| What | Command | Agent |
|---|---|---|
| each commit's message | the global `commit-msg` hook (`workline hooks install --global`) | yours, to rewrite a refused message, 30 s at most (`rewrite: false`, the project's or yours, turns it off) |
| before a push | the global `pre-push` hook, when the project routes `pre-push`; by hand: `workline route pre-push --ai none --input range=origin/main..HEAD` | never |
| docs made suspect | `workline docs` (`--since <rev>`; `--review` with no agent) | yours |
| a branch's code | `workline review` (`--base`, `--lenses`, `--json`) | yours |
| a spec, before it is built | `workline review --spec <file>` or `--issue <n>` | yours |
| gardening | `workline route schedule` (`--forge local --open-merge-request` keeps its merge requests as local branches) | yours |
| the backlog | `workline run-role product-owner --event schedule --forge gitlab` | yours |
| a ready issue's tests | `workline test write --issue <n>`; red-first on any change: `workline test check --base <rev>` ([tester](../roles/tester/README.md)) | yours; `check`, never |
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

- one judges with the agent and no write token (`--no-apply`);
- one applies with the write token and no AI key;
- `WORKLINE_RUNS_DIR` names where the runs are kept: carry that folder and
  the `--json` result from the first job to the second, at the same path;
- where the merge request's code can change the first job (a GitHub pull
  request's own run), the second runs from the base: `--from-merge-request
  <n> --head <sha>` ([more](commands.md#workline-apply));
- the gates run first, in a job with no write token and no AI key:
  `route <event> --gates-only --json > gates.json`, which the judge reads
  (`--gates-from gates.json`) and the second job shows (`apply --gates
  gates.json`, on GitHub; [gates in CI](gates-in-ci.md)).

`$ai` below is your choice of agent: `none`, `claude`, `cmd:<command>`
([usage](usage.md)).

### Merge request

The committer, the documentalist, the reviewer if routed; one run at a time ([#359](https://github.com/JN0V/workline/issues/359)):

```sh
workline route merge-request --ai "$ai" --no-apply \
  --forge gitlab --target merge-request:<iid> --branch <source-branch> \
  --input range=<base-sha>..<head-sha> --code-quality gl-code-quality.json --json > line.json
workline apply --line line.json          # second job: commits the doc fixes to the branch
```

`--input draft=<true|false>`, on each run: `true` asks no agent of any
step, the AI review waiting for ready; `false` has the reviewer read every
lens once, on the whole change, on a merge request its record does not
say every lens read; `--input lenses=all` asks it outright, when it becomes
ready, as the templates do ([how](../roles/reviewer/docs/ready.md)).

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
```

The last command, with the applying job's token and no agent, tends the
merge requests gardening opened before: rebuilt on the branch's new tip,
or closed when nothing is left to merge
([ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)).

### The product owner alone

On its own trigger:

```sh
workline run-role product-owner --event schedule --ai "$ai" --no-apply --forge gitlab --json > po.json
workline apply "$(jq -r '."run-dir"' po.json)"
```

### A ready issue

With the developer routed on `issue`, when an issue is labelled
`workline:ready` ([the triggers](../roles/developer/docs/triggers.md)):

```sh
workline route issue --target issue:<n> --ai "$ai" --no-apply --forge gitlab --open-merge-request --json > line.json
workline apply --line line.json
```

### On a role's own merge request

A note on the tester's Draft (an objection) or on the developer's merge
request (a review round); the run applied from the default branch, as
data ([the tester's jobs](objection-jobs.md),
[the review loop](../roles/developer/docs/review-loop.md#what-wakes-it)):

```sh
workline test answer --merge-request <iid> --ai "$ai" --no-apply --forge gitlab --json > answer.json
workline develop --merge-request <iid> --ai "$ai" --no-apply --forge gitlab --json > round.json
workline apply <run-dir> --from-merge-request <iid> --head <sha> --forge gitlab
```

### Release

Before your release tool tags; a non-zero exit holds it.

- On a machine: `workline route release`.
- At a tag (GitLab's `workline:release`): `workline route release --ai
  none --forge none`, no token. The jobs that publish need it.
- Each night, a dry run: `workline route release --dry-run --forge
  gitlab --json > dry-run.json` (no agent, writes nothing), then
  `workline apply --release-dry-run dry-run.json --forge gitlab` (token,
  no agent) ([the release](release.md)).
- On a push to the default branch: `workline follow --base <branch>
  --forge gitlab` (token, no agent), `user.name` and `user.email` set
  first, as for `apply`. With `merge:`, it runs
  the [merge policy](merge-policy.md): after each judge too, given the
  head's verdict `route --verdict` wrote (`--verdict <file>`).

### Weekly sample

`workline sample --forge gitlab --out sample.json` (agent, reads the
forge, writes nothing; `--week 2026-W40`): every role's acts drawn
([ADR-0046](adr/0046-the-auditor-re-checks-every-roles-acts.md)). Then `workline sample --apply
sample.json --forge gitlab` (token, no agent), which writes the week on
one issue, the level the product owner's acts suggest among it
([ADR-0033](adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)).

### Import

- `workline issues import <file> --apply` has the agent and the write
  token in one process: run it by hand.
- In a pipeline, split it as any other role: `workline issues import
  <file> --json > line.json` (agent, a read token, writes nothing), then
  `workline apply --line line.json` (write token, no agent);
  [ci.md](ci.md#importing-a-file).

### Your own scanners

`--sarif <file>` beside `--code-quality` writes the findings for GitHub's
code scanning, on a merge request not the documentalist's, the
reviewer's nor the gates' ([#412](https://github.com/JN0V/workline/issues/412)).
`gate:<name>` steps, or `workline gate <name>`, run your own scanners; on
a merge request given its `range`, or with `workline gate <name> --base
<rev>`, only the findings the change adds count
([new findings](spec/gates.md#on-a-merge-request-the-findings-the-change-adds)).

## What a job needs and gives back

The clone, variables and tools each job needs, GitLab's tokens, the exit
codes and what a run writes: [jobs' needs and outputs](triggers-outputs.md).
