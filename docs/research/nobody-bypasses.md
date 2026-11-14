# Nobody bypasses — what `main`'s bypass list served, and what an empty one breaks (2026-11-13)

**Question.** workline's `main` ruleset lets the repository's admin
bypass it on a pull request. Does anything legitimate need that, and
what holds a person and their assistant once the list is empty, on
GitHub and on GitLab?
[ADR-0068](../adr/0068-the-line-steps-back-on-its-own-drift.md#6-nobody-bypasses-on-main)
is the decision.

**Verdict.**

- **Nothing legitimate needs it**: workline's App, releases, gardening
  and a labelled pull request all merge with none.
- **Nearly every bypass skipped only the label**, the other checks
  green: a flag on a routine command, not a rescue.
- **An empty list holds the routine path**; the label and a ruleset
  edit stay in reach of the person's token, each leaving a trace.

Read on 2026-11-13 with the maintainer's `gh` login, read-only (the
rulesets, the rule-suites API, the last 300 merged pull requests). No
write was tried. An independent check read the same forge first; its
bypass count included pagination duplicates, re-counted here by id.

## What the forge holds

- **Repository**: user-owned, one collaborator with admin. No classic
  branch protection, no `CODEOWNERS` file, no tag ruleset.
- **`main: pull requests with green checks`**: no deletion, no force
  push, a pull request, required checks `test`, `judge`,
  `workline bounds` and `workline/accepted`, the last pinned to
  workline's App. Bypass list: the repository admin role, on pull
  requests (`current_user_can_bypass: pull_requests_only`).
- **`release branch: the App alone pushes`**: only workline's App
  creates, updates or deletes `release-please--*`. Its bypass list is
  an allow-list and stays.

## What the bypasses skipped

Rule suites on `refs/heads/main`, 2026-10-13 → 11-13, each read by id:

| Result | Count |
|---|---|
| Bypass, the maintainer's account | 392 (272 from 10-21) |
| of them, only `workline/accepted` expected (no label) | 375 (96%) |
| of them, `workline/accepted` failing | 3 |
| of them, other checks not yet reported or failing | 14 |
| Pass, workline's App | 192 |

## Who merged the last 300

| Opened by | Merged by | Labelled | Count |
|---|---|---|---|
| the maintainer's account | itself | no | 141 |
| the maintainer's account | workline's App | yes | 95 |
| workline's App | itself | no (routine) / yes | 30 / 20 |
| workline's App | the maintainer's account | no / yes | 7 / 5 |
| others | the maintainer's account | yes / no | 2 |

The 141 are the bypass path: with an empty list, each takes the label.

## What an empty list breaks

- **`gh pr merge --admin`**, the routine path above. That is the point.
- **A merge before checks report, or over a red one**: 17 in the month;
  the last over a failing check on 11-05.
- **One real deadlock**: the merge and apply workflows build the engine
  from `main`. A merge that breaks `follow`, the App's token or those
  workflows leaves no head able to get `workline/accepted`, its own
  fix included. The way out is a person's deliberate ruleset edit,
  written as a procedure.

## What a token can still do

- **Set `workline:accepted`**: a label needs the same right as opening
  a pull request; no scope tells the person from their session.
- **Edit the ruleset**: the daily token holds `repo` scope on an admin
  account (by the documentation; not tried).
- **Trace left**: the ruleset's history (`/rulesets/{id}/history`), the
  personal security log, the rule suites; all need an admin's token.
  Any token reads the ruleset's `rules`, `enforcement` and
  `updated_at`, and a merged head's statuses.
- **A cheaper guard**: a fine-grained daily token without the
  Administration permission. Not tried.

## On GitLab Premium

From GitLab's documentation, not tried on a project:

- **Protected `main`**: "Allowed to push and merge": No one, since
  anyone listed there skips approvals, code owners included. Force push
  off. "Require Code Owner approval" on.
- **Merge checks**: "Pipelines must succeed"; approval rules locked in
  merge requests.
- **Code Owner approval carries weight there**: a merge request's
  pipeline runs its own `.gitlab-ci.yml`, so the CI files need an
  owner's approval. No required status pinned to an App on Premium.
- **Emergency**: a Maintainer changes the protection; the project's
  audit events record it.
- **Alone**, a person clears the self-approval settings and their
  approval is a click their token can give too, as on GitHub.

## Not verified

- No write: the ruleset edit by the daily token, the fine-grained
  token's refusal, a self-approval refused under code-owner review.
- Whether workline's App can read the ruleset's history or the rule
  suites.
- GitLab: read from its documentation only.
