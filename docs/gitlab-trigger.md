---
sources:
  - internal/line/payload.go#payload
  - internal/line/payload.go#readPayload
  - internal/line/payload.go#readNote
  - internal/line/payload.go#onLine
  - internal/forge/gitlab_hook.go#gitlab.SetIssueHook
  - internal/doctor/hook.go#IssueNotes
  - internal/doctor/hook.go#issueNotes
  - internal/doctor/hook.go#issueHook
  - defaults/routing.yaml#max-hops
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:issue.rules
  - internal/forge/gitlab.go#gitlab.connect
  - internal/forge/local.go#Local
  - internal/forge/gitlab.go#gitlab.AddBlocker
  - internal/backlog/depend.go#Blockers
  - internal/backlog/depend.go#WithBlockers
  - internal/backlog/depend.go#blockedByLine
checked: b2a36983f
judged: 9a88170
verified: agent:documentalist
---
# GitLab: pipelines a tool starts

The GitLab template ([ci-gitlab.md](ci-gitlab.md)) runs on merge request
events and pipeline schedules. Where a webhook or a tool starts the jobs
instead — the trigger API, the pipelines API, another pipeline — this page
does the same with a plain script: the template's commands under other
rules. The commands themselves: [triggers.md](triggers.md).

## A label or a comment, with GitLab alone

GitLab has no CI rule on a label or a comment, but nothing outside
GitLab is needed:

- **A project webhook** (Settings → Webhooks) on issue events, and note
  events for comments, its URL the project's own trigger URL:
  `https://gitlab.com/api/v4/projects/<id>/ref/<default branch>/trigger/pipeline?token=<trigger token>`.
- **Each event starts a pipeline** with `CI_PIPELINE_SOURCE=trigger`; the
  event's JSON is in the file `$TRIGGER_PAYLOAD`: `object_kind`, the
  issue's `iid`, `user.username`, `changes.labels.previous` and
  `current`; on a comment, its `note` and the issue's `labels`.
- **Variables in the URL** reach the jobs:
  `&variables[WORKLINE_TASK]=issue`. Event fields such as the issue's
  number do not: the engine reads them from `$TRIGGER_PAYLOAD` itself.
- **What it needs**: any tier, Free included; a Maintainer to create the
  hook and the trigger token. `workline setup --yes`, run in the
  project's clone with a Maintainer's token, creates both: the hook on
  issue events, and on note events where the reviewer or the product
  owner is routed on `issue`; `workline doctor` checks them
  ([the commands](commands-setup.md#workline-setup)).
  On a self-managed instance, an administrator allows
  webhooks to reach the local network (Admin → Network → Outbound
  requests), as the hook calls GitLab itself (untried).
- **Every issue change fires it**: a title edit, a new issue, a label
  the engine itself sets; with note events, every comment, on a merge
  request too. The engine exits at once on any other event than a
  state's label put on an issue, the reviewer's comment on a spec in
  refine, or a person's comment or body edit on an issue on the line; a
  chain of the engine's labels stops at the hop cap.

### The line of states on it

What the hook starts, a state's label, a comment or the hourly
reconcile: [the line of states on GitLab](gitlab/line-of-states.md).

## Before CI: on your machine

Everything runs locally first. `forge: gitlab` in `.workline/config.yaml`
and `glab auth login` (or `GITLAB_TOKEN` and `GITLAB_HOST`) let the
product owner's run read your issues from a laptop
([its command](triggers.md)); `forge: local` keeps issues and merge
requests in the clone, nothing pushed ([no forge](no-forge.md)).

## What Premium adds

- **Blocks links**: the product owner's `depend` writes GitLab's
  `is_blocked_by` link; on Free, GitLab refuses it and the issue's body
  gets `Blocked by #12.` instead. Both are read back.
- **Project and group access tokens** on gitlab.com (every tier on a
  self-managed instance): a bot user, the right identity for workline.
  A group token can serve several projects; untried with workline.

## How the pipeline knows what to do

The tool, or the webhook's URL, passes what to do as variables
(`WORKLINE_TASK` and the rest): [the variables](gitlab/trigger-variables.md).

## The jobs, without the template

A `.gitlab-ci.yml` and a script that run the same commands, for a project
that does not include the template: [the jobs, without the template](gitlab/without-template.md).

## Tokens and variables

What each token holds, masked or protected, and the environment
`workline` that keeps them from your tools:
[tokens and variables](gitlab/trigger-tokens.md).
