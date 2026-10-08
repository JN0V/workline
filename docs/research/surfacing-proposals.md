# Surfacing a bot's proposals, and a person's answer — focused research (2026-10-08)

**Question.** Where do backlog bots put what they did and what they
propose, how does a person say yes, no or "change this", and what does
each forge record and notify? Extends [product-owner.md](product-owner.md)
(dashboards, ticks and who ticked, stale bots).

**Verdict.** The ecosystem keeps **the control on the item** (a label, a
close, a comment on that issue or pull request) and **the view as a
filter** (a label search, a board, a label subscription). A dashboard
issue with checkboxes is Renovate's alone, and its own maintainers want
display split from controls. Label = decision, comment = reason is the
only pair carrying author and time on GitHub, GitLab and Gitea/Forgejo
with a bulk path from the issue list. Labels, reactions and edits notify
nobody; comments, closes and mentions notify.

Searched in the ecosystem's words: *dependency dashboard*, *triage
report*, *slash command*, *quick actions*, *label events*, *timeline*,
*award emoji*, *to-do*, *notification reasons*, *staged mode*, *step
summary*, *digest*, *triage inbox*. "To verify" = no doc confirms it; try
it on a sandbox first.

## Where bots put acts and proposals

| Tool | Acts | Proposals | Yes / no |
|---|---|---|---|
| [Renovate](https://docs.renovatebot.com/key-concepts/dashboard/) | a pull request each | one Dependency Dashboard issue, rewritten in place | tick a box; close the pull request = never again |
| [Dependabot](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-pull-request-comment-commands) | a pull request each | the pull request is the proposal | merge; close, or `@dependabot ignore …` on it |
| [release-please](https://github.com/googleapis/release-please) | — | one release pull request, labelled `autorelease: pending` | merge it |
| [gitlab-triage / triage-ops](https://handbook.gitlab.com/handbook/engineering/infrastructure-platforms/developer-experience/triage-operations/) | labels and comments on the item | a new dated report issue per period, closed by a job | act on the item; the report's boxes are the reader's progress, not commands |
| [Prow](https://prow.k8s.io/command-help) | labels | — | `/triage accepted`, `/close`, `/hold` at a comment's line start; OWNERS files say who may |
| [Mergify](https://docs.mergify.com/changelog/2026-06-19-queue-or-requeue-a-pull-request-with-a-checkbox.md) | labels mark state | a status comment on the pull request | `@mergifyio queue`, or its checkbox |
| [actions/stale](https://github.com/actions/stale) | label + comment on each item | — | any activity, or the label off |
| [GitHub AI issue intake](https://docs.github.com/en/issues/tracking-your-work-with-issues/administering-issues/triaging-an-issue-with-ai) | comment and labels on the issue | — | the person acts on the item |
| [gh-aw Auto-Triage](https://github.github.com/gh-aw/blog/2026-05-29-agent-of-the-day/) | labels alone, with a rationale | a dated Discussion per run; [staged mode](https://github.github.com/gh-aw/reference/staged-mode) previews writes in the Actions step summary | — (trust before go-live, not approval) |
| [Linear Triage Intelligence](https://linear.app/docs/triage-intelligence) | per property: auto-apply | a suggestion on the item, with why | accept / dismiss on the item; [Pulse](https://linear.app/docs/pulse) digests |
| [GitLab Duo Planner](https://docs.gitlab.com/user/duo_agent_platform/agents/foundational_agents/planner/) | edits after it asks | in a chat sidebar | interactive confirmation, not scheduled |

**Patterns**, from the widest:

1. **Control on the item**: Prow, Dependabot, Mergify, Renovate's per-PR
   box, Linear, `@gitlab-bot`.
2. **State as a label, view as a filter**: Kubernetes' `needs-triage` →
   `triage/accepted`, GitLab's scoped `workflow::*`, `autorelease:
   pending`; read through a saved search, a board, a label subscription.
3. **The proposal is the change**: a pull request to merge or close.
4. **A dated digest, append-only**, for what was done: triage-ops, gh-aw,
   [weekly-digest](https://github.com/probot/weekly-digest),
   [issue-metrics](https://github.com/github/issue-metrics).
5. **One dashboard rewritten in place, with command boxes**: Renovate only.

## What went wrong with reports

- Renovate's maintainers: [#10924](https://github.com/renovatebot/renovate/issues/10924)
  "separate out display from controls"; more detail "seems more messy";
  on forges without checkboxes the dashboard is read-only
  ([#9592](https://github.com/renovatebot/renovate/issues/9592)).
- GitLab's weekly triage report "has grown to a lot of items", asked to
  collapse sections ([triage-ops#728](https://gitlab.com/gitlab-org/quality/triage-ops/-/work_items/728));
  it reaches managers, never the people who act
  ([#1864](https://gitlab.com/gitlab-org/quality/triage-ops/-/work_items/1864)).
- Volume: "noise is how important updates get ignored"
  ([GitHub on Dependabot](https://github.blog/security/application-security/tame-dependabot-group-your-updates-slow-the-cadence-keep-security-fast/));
  stale-bot notification floods ([actions/stale#1122](https://github.com/actions/stale/issues/1122)).
- workline's own report issue ([#110](https://github.com/JN0V/workline/issues/110))
  mixed a call to act, a next list, acts done, undo instructions, the
  autonomy table and how it works: the display/controls/reference mix
  above. Its readers could not tell what to look at. Its state comments
  on each issue were YAML only, nothing for a person.

## What each forge records

| Mechanism | GitHub | GitLab | Gitea/Forgejo | Bulk from the list |
|---|---|---|---|---|
| Label set / removed | [issue events](https://docs.github.com/en/rest/issues/events), `actor` + time | [resource label events](https://docs.gitlab.com/api/resource_label_events/), `user` + time; [scoped](https://docs.gitlab.com/user/project/labels/) `key::value` replace each other | timeline type `label`, user ([API](https://docs.gitea.com/api/1.24/)) | yes, all three |
| Comment | `user.type` (`User`/`Bot`), `author_association` ([API](https://docs.github.com/en/rest/issues/comments#list-issue-comments-for-a-repository)) | note author, `system` flag | user + time | no |
| Line-start command | plain text | GitLab's own [quick actions](https://docs.gitlab.com/user/project/quick_actions/) run, leave a system note by the person; an unknown `/word` stays as text (to verify) | plain text | no |
| Reaction | user + time | award emoji, user + time | user + time | no; anyone can react |
| Body edit | `updated_at`; editor via GraphQL `userContentEdits` | `last_edited_by` | `updated_at` | no |
| Close | reason `completed` / `not_planned` / `duplicate` | close; `/duplicate`; no "not planned" | close | yes |
| @mention of the bot | `github-actions[bot]` is no useful mention target (to verify) | [Duo flows](https://docs.gitlab.com/user/duo_agent_platform/triggers/) fire on a service account mentioned | a bot user | — |

- **Who may decide**, with no list of our own: GitHub's
  `author_association` (OWNER, MEMBER, COLLABORATOR) in the comment
  itself; GitLab's member access level, one call per author, cached;
  Gitea's collaborator check.
- **A person, not the bot**: author ≠ the token's own user, and not
  `user.type == "Bot"` (GitLab: `system: false`).

## What each forge notifies

| Event | GitHub ([reasons](https://docs.github.com/en/rest/activity/notifications)) | GitLab ([notifications](https://docs.gitlab.com/user/profile/notifications/)) |
|---|---|---|
| New comment | every subscriber | participants, watchers |
| Label added / removed | nobody | label subscribers only (opt-in) |
| Body or comment edit | nobody | nobody, but a newly mentioned user ([#118779](https://gitlab.com/gitlab-org/gitlab/-/issues/118779)) |
| Reaction | nobody | nobody ([#17410](https://gitlab.com/gitlab-org/gitlab/-/issues/17410)) |
| Close / reopen | subscribers | participants, watchers |
| @mention | the person | the person, plus a to-do |

- A comment by `GITHUB_TOKEN` (`github-actions[bot]`) mails subscribers
  like any comment (to verify).
- GitLab's label subscription is a free inbox for "proposals labelled X";
  GitHub has none: a saved search, or a Projects view (GraphQL and a
  personal token or an App; `GITHUB_TOKEN` cannot reach user projects).

## How bots handle "change this"

- **Copilot coding agent** ([docs](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/make-changes-to-an-existing-pr)):
  a comment mentioning `@copilot`, from people with write access only;
  it acknowledges with a 👀 reaction, then pushes and comments.
- **CodeRabbit** ([commands](https://docs.coderabbit.ai/guides/commands),
  [learnings](https://docs.coderabbit.ai/guides/learnings)): a reply in its
  comment's thread; it may store a "learning" so it does not repeat.
- **triage-ops**: a note starting `@gitlab-bot label ~x`, from the author,
  an assignee or a team member only, rate-limited.
- **Renovate**: edit the config, or the pull request (it then stops
  rebasing it); no comment commands.
- **Convention**: yes / no = a forge-native state change; "change this" =
  a comment on the item by a write-level person, acknowledged by a
  reaction, then answered; "never" = closing the bot's object.

## Finding what was said, cheaply, each night

- **GitHub**: `GET /issues/comments?since=` (repo-wide, created or edited,
  with author type and association); `GET /issues?labels=…&since=` then
  the timeline of only those issues. The notifications API needs a
  classic personal token: not usable from CI.
- **GitLab**: `GET /projects/:id/events?target_type=note&after=<date>`
  (a date: overlap a day); issues `?labels=…&updated_after=` then their
  label events. A bot user's to-dos (`GET /todos?action=mentioned`) would
  be "what was addressed to me" (to verify for bot users).
- **Gitea/Forgejo**: `comments?since=` and a per-issue timeline.
- **"Handled" with no state of ours**: the bot's own reaction on the
  comment it read (Copilot's 👀), visible and silent on all three forges.

## Risks found

- **Strangers steering**: on a public project any comment can bring an
  issue back into a nightly run, and an outsider's "this duplicates #42"
  can steer an act on another issue; bots and "+1" spend the run's slots.
- **A bulk yes is a rubber stamp**: text laundered from a stranger's
  comment into a draft can be accepted unread.
- **Loops**: a person's edit of a drafted section shrinks what the bot may
  rewrite, so it converges; a disagreement in comments does not, and
  needs a cap. A section a person deleted may be written again by the
  next run (verified, and fixed by ADR-0038).
- **Labels nobody filters** rot like `needs-triage`; GitHub sends nothing
  for a label.
- **"No" read too widely**: a proposal set aside on one issue taken as
  "never do this kind of act" turns a level off with one click.

## Checked on the sandboxes

Tried 2026-10-08 on gitlab.com/JN0V/workline-sandbox (Free plan) and
github.com/JN0V/workline-sandbox, before building ADR-0038:

- **GitLab quick actions**: an unknown `/word` at a line's start stays
  text; a known one (`/label`, `/unlabel`) at a line's start runs and is
  stripped, when a note is written **and when it is edited**, and a note
  made only of commands is not stored. Mid-sentence, it stays text.
  ([docs](https://docs.gitlab.com/user/project/quick_actions/))
- **GitLab to-dos for a bot**: a project access token's user gets
  `mentioned`, `directly_addressed` and `assigned` to-dos, readable with
  its own token.
- **GitHub's Actions bot**: a comment by `github-actions[bot]` with no
  mention puts the thread unread for its subscribers, as any comment;
  mentioning the bot does nothing (no inbox, no mention event).
- **Scoped labels on GitLab Free do not replace each other**: with
  `x::proposed` on, adding `x::accepted` keeps both; exclusion is a
  Premium feature
  ([docs](https://docs.gitlab.com/user/project/labels/#scoped-labels)).
- **What the engine reads**: label events with actor and time (GitHub's
  issue events; GitLab's `resource_label_events`, readable with the bot's
  token); a comment author's rights (GitHub's `author_association`;
  GitLab's `members/all`); 👀 on a comment (GitHub answers 200 for one
  already there, GitLab 404 "Award Emoji Name has already been taken").
