---
sources:
  - cmd/workline/cli.go#init
  - defaults/routing.yaml#events
  - defaults/routing.yaml#states
  - defaults/routing.yaml#reconcile-max
  - roles/committer/role.yaml#events
  - roles/committer/role.yaml#settings.ai-max-seconds
  - roles/documentalist/role.yaml#events
  - roles/reviewer/role.yaml#events
  - roles/inspector/role.yaml#events
  - roles/product-owner/role.yaml#events
  - defaults/ci/github/workline.yml#on
  - defaults/ci/github/workline-gardening.yml#on
checked: e81df7b34
verified: agent:claude-code
---
# Where each role runs

What starts which role, on your machine, on a merge request, on main and
on the issues. The overview and the diagram are in
[the README](../README.md#where-it-runs); the triggers in detail are in
[triggers](triggers.md).

## Your machine

| When | Roles | Agent |
|---|---|---|
| `git commit` (the global hook) | committer | yours, to rewrite a refused message, 30 s at most |
| `git push` (the global hook, once `workline init` routes it) | committer; documentalist, counting the docs made suspect | never |
| `workline review` | reviewer, before you push | yours |
| `workline review --spec <file>`, `--issue <n>` | reviewer, on a spec before it is built | yours |
| `workline docs` | documentalist: fixes the docs made suspect, you keep or drop each | yours |

## The merge request

| What | Roles | Writes |
|---|---|---|
| each push to it; on GitHub, marked ready too | committer; documentalist; reviewer and inspector, opt-in | the docs' fix in a comment, the agent once at ready; each finding on its line, and one summary comment; the findings in the forge's code-scanning view; the job's summary |
| a release tool's (release-please…) | as any other, the docs never holding it; an issue labelled to hold it, or main not green, holds it ([how](release-holds.md)) | its checks |
| a fork's | judged with no agent | on GitHub, a comment; on GitLab, nothing |
| a role's own: a comment on the tester's Draft, a verdict on the developer's | [tester](../roles/tester/README.md): answers an objection; [developer](../roles/developer/README.md): fixes its review | a reply or a rewritten test; the fix, pushed on its branch |

- The review is on the merge request; a person accepts it, or, by the
  [merge policy](merge-policy.md), a reproduced bug's fix merges alone.
- Templates: [GitHub Actions](../defaults/ci/github/workline.yml),
  [what only warns, beside it](../defaults/ci/github/workline-warn.yml),
  [its jobs that write](../defaults/ci/github/workline-apply.yml) and
  [its fork job](../defaults/ci/github/workline-fork.yml),
  [the release's holds read again](../defaults/ci/github/workline-release.yml),
  [GitLab CI](../defaults/ci/gitlab/workline.gitlab-ci.yml). What each job
  writes, and setting up: [CI](ci.md#the-jobs).

## main: schedules and releases

| When | Roles | Writes |
|---|---|---|
| after each merge (`workline route main`) | the merge request's gates, every finding kept as the next base; documentalist, the whole repository, no agent | one issue, edited when what it lists changes ([how](main-after-merge.md)) |
| gardening, nightly or weekly (`workline route schedule`) | documentalist; product owner, reviewer, tester, developer, opt-in | one merge request per task, kept on main or closed; the issues |
| the weekly sample (`workline sample`) | [auditor](../roles/auditor/role.yaml): one in ten of every role's acts, each re-checked | one issue, a comment a week; a merge request undoing a wrong confirmation |
| each push to main (`workline follow`) | the engine, no agent: with the [merge policy](merge-policy.md), turns auto-merge on for what a person accepted | the forge's auto-merge |
| with the gardening (`workline route release --dry-run`) | what would hold the release if it were cut now, no agent | a part of main's issue ([how](release.md)) |
| before your release tool tags (`workline route release`); on GitLab, at a tag | your release's gates, none by default | a non-zero exit holds the release |

- workline cuts no releases: it runs before your release tool.
- Templates: GitHub's [gardening](../defaults/ci/github/workline-gardening.yml) and
  [sample](../defaults/ci/github/workline-sample.yml), `follow` in
  [workline-apply.yml](../defaults/ci/github/workline-apply.yml); on GitLab, three pipeline
  schedules, a push to the default branch and a tag, all in the same
  [template](../defaults/ci/gitlab/workline.gitlab-ci.yml).
- What each job writes: [CI](ci.md#the-jobs); each command in detail:
  [usage](usage.md#commands).

## The issues

| Who | What |
|---|---|
| [product owner](../roles/product-owner/README.md), on gardening or at once on an issue to refine | reads a share of the open issues against the code: completes them, readies the evident ones, proposes the rest on each issue, `workline:proposed` |
| `workline issues import <file>` | a roadmap file to issues |
| [tester](../roles/tester/README.md), opt-in | a ready issue's tests, proven red, in a Draft |
| [developer](../roles/developer/README.md), opt-in | a ready issue's code, proven, in a pull request; the issue labelled ready, or the gardening |
| the line of states | each state label wakes its role: to refine, needs tests, ready; an hourly reconciler catches what an event missed ([how](spec/routing.md#the-line-of-states)) |
| [reviewer](../roles/reviewer/README.md) | a finding outside the change: an issue, `needs-triage` |
| [documentalist](../roles/documentalist/README.md) | the code disagrees with a decision a doc records: an issue |
| a person | accepts: the label `workline:accepted`, a parent closed |
