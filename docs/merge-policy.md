---
sources: [internal/engine/merge.go, internal/engine/queue.go, internal/forge/merge.go, internal/forge/github_merge.go, internal/forge/gitlab_merge.go, ci/github/workline-apply.yml, ci/github/workline-merge.yml, ci/gitlab/workline.gitlab-ci.yml]
checked: 8ccdfb74
verified: agent:claude-code
---
# The merge policy

```mermaid
flowchart LR
  yes["<b>A person</b><br/>labels it accepted"] --> follow["<b>workline follow</b><br/>checks the head's verdict"]
  follow --> on["<b>The forge</b><br/>auto-merge on"]
  on --> merged["merged,<br/>its own head green"]
  merged --> main["<b>main's run</b><br/>a red main said"]
```

A person says yes with a label; the forge merges, by its own rules.
`workline follow` turns the forge's auto-merge on for what was accepted;
each merges once its own head is green. No agent is called
([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md)).

```yaml
# .workline/config.yaml
merge: {autonomy: cautious}   # off unless set: no auto-merge is touched
                              # normal: workline's own routine work with no yes, below
```

## What it does

- **Turns auto-merge on** for a pull request labelled `workline:accepted`
  (GitLab: `workline::accepted`), its head's verdict passed:
  - the label put by someone who may push and merge: GitHub's write
    access, GitLab's Developer or above;
  - the line passed on that head, every step that needs an agent answered
    by one, whole. With no agent, every review falls short: nothing is
    turned on where the reviewer runs.
  - **The verdict's run only**: the reviewer's lenses that only warn read
    apart, after or beside it. Their findings, and one of them failing,
    may post after the verdict, and after the merge: read their comment
    before you accept ([apart](../roles/reviewer/docs/apart.md)).
- **The forge merges**: GitHub's auto-merge, GitLab's "merge when the
  pipeline succeeds", each checking its own rules. One the forge could
  merge at once is merged at once.
- **No one at a time, no update first**: a pull request behind its base
  merges as it is, none waiting for another
  ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-2-strict-up-to-date-dropped)).
  Two green alone may break the base together: main's run says it
  ([below](#a-red-main)).
- **Auto-merge a person turned on by hand** is their yes, weighed as the
  label's.
- **Nothing turned on, said in the job's summary**: a verdict or a check
  red, a review short. A run the forge cancelled is no result: another
  run of that check counts — on GitLab, the newest other pipeline of that
  head —, else it waits. One not judged yet waits, named.
- **The label** is made in the forge's list when missing: put it in bulk,
  from a saved filter.

## The yes covers one head

- **A push after the yes takes it back**: auto-merge off, the label off,
  one comment. When each head was pushed is the forge's word: GitHub's
  repository activity, GitLab's versions.
- **workline's own commit is no push**, checked by its tree; its trailer
  is needed, never taken on trust. The committer checks every commit all
  the same:
  - `follow`'s update: the base merged in, never another branch, or a
    role's branch rebuilt on it — exactly what workline's own merge or
    rebuild gives;
  - a record: one commit changing, against its parent, only lines
    workline writes in the docs — a header's `checked`, `judged`,
    `judged-in-parts` or `verified`, a derived block as its command gives
    it ([ADR-0041](adr/0041-workline-settles-its-own-lines-when-the-base-moves.md)).
- **Still a push**: a docs fix changing a doc's text or another key, and
  any commit with no workline trailer, whatever it changes; one whose
  message the committer rewrote (`Workline-Role: committer`) is a
  person's.
- **Held by the forge, not by a race**: GitHub keeps auto-merge on after a
  writer's push. So the base requires the status `workline/accepted` from
  the App, which `follow` sets on the head a yes covers: a new head lacks
  it and cannot merge. A base that does not require it, or from any
  source, gets nothing turned on, said. On GitLab, the merge waits for
  the new head's pipeline, which holds `workline:merge`.
- **A merge queue** tests a group's commit against the same checks:
  `follow --merge-group` sets `workline/accepted` there as the head the
  group ends with holds it; the queue's run checks only the gates, the
  yes and the verdict staying the head's ([a merge queue](merge-queue.md)).

## Never without a person's yes

- **A release pull request** merges only labelled; it never waits for
  the docs ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly)).
  An issue labelled to hold it, or main not green, turns its verdict red
  until they are gone ([what holds a release](release-holds.md)).
- **A fork's**: its CI run is the fork's own, so the templates keep no
  verdict of it: it waits for a person.
- **Any other** with no label, whoever opened it — but, at `normal`,
  workline's own routine work. A bot's own auto-merge (a dependency
  bot's) is left to it.
- **An agent never puts the label**: it is the person's yes.

## `normal`: routine work with no yes

```yaml
merge: {autonomy: normal}   # a project's choice, never a default
```

At `normal`, workline's own routine pull requests merge with no label
too — an agent's rewording of a page for its reader only once the
project lets it; a revert sends their kind back to a person's yes. What
counts as workline's own, the kinds, the revert and the limits:
[the merge policy at `normal`](merge-policy-normal.md).

## The veto is the forge's

- **Draft**: nothing is turned on; auto-merge on goes off. A draft asks
  no agent, so nothing it skips can merge
  ([ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md#4-what-the-table-holds-to)).
- **Auto-merge turned off by a person** stays off until the label is put
  again.
- **The label taken off** turns auto-merge off.

## Where the head's verdict lives

- The judging job writes it: `workline route … --verdict <file>`.
- **GitHub**: a job of its own keeps it, `workline follow --verdict <file>
  --verdict-only`: the status `workline/verdict` on the head, set by the
  App. Only the statuses `follow`'s own token set are read: one a person
  sets with their own token never counts.
- **GitLab**: in hand in the merge request's pipeline, where
  `workline:merge` reads it; nothing kept elsewhere.
- No verdict on the head yet: auto-merge waits for it.
- **A draft's verdict** is a review to come (`waits`): the rules and
  gates passed, the AI review waits for it to be marked ready. GitHub's
  `workline/verdict` is pending. Marked ready, auto-merge waits for the
  run on ready's verdict (`review-to-come`), never taking the draft's
  for a pass ([the reviewer on a draft](../roles/reviewer/docs/ready.md)).
- **The App's key, out of a pull request's reach**: GitHub runs a pull
  request's own copy of the workflow, which its branch can change. The key
  sits in an environment only the default branch may use; the jobs
  holding it run the default branch's workflow and read the pull
  request's run as data
  ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).
- **The limits**:
  - the verdict is the pull request's run's word: one changing its
    workflow can pass its own head, as its own `judge`; the yes covers
    that head;
  - on GitLab, whoever can run a pipeline can read the project bot's
    token, so a person who may push could act as it.

## Set up

- **GitHub**: the [templates](ci-github.md) — `workline-apply.yml`'s
  `verdict` and `follow` jobs run after `judge`, `queue` after a merge
  queue's group, from the default branch;
  `workline-merge.yml` runs when a label is put or taken off, or a draft
  made. Then:
  - Settings → General → "Allow auto-merge";
  - the [GitHub App](ci-github-app.md), with **Commit statuses** read and
    write besides its other permissions, its key in the environment
    `workline-app`; a merge made with the job's own token starts no
    workflow on `main` — nothing is turned on then, said;
  - a ruleset on the base requiring the checks, and `workline/accepted`
    from the App; "Require branches to be up to date" left off.
- **GitLab**: the [template](ci-gitlab.md)'s `workline:merge` job, after
  the judge; the project bot's token. Merged results pipelines on
  (Premium): without them a merge request's pipeline tests its branch
  alone. A label starts no pipeline there: it
  waits for that merge request's own next pipeline.
- **Another forge** (`cmd:`): the operations `merge-state`, `auto-merge`,
  `guarded`, `accept`, `keep-verdict` and `verdict`
  ([forge command](spec/forge-command.md)); at `normal`, more
  ([set up](merge-policy-normal.md#set-up)). The local forge has no
  auto-merge: said, nothing turned on.

## What the summary says

| Said | When |
|---|---|
| `auto-merge-on`, `merged-at-once` | accepted, its head's verdict passed; at `normal`, "with no person's yes", its kind |
| `verdict-kept`, `verdict-missing`, `review-to-come` | the verdict kept; one not judged yet; marked ready, the draft's verdict still on its head |
| `verdict-red`, `review-short`, `checks-red` | nothing turned on, why |
| `accepted-reset` | a push after the yes: the yes and auto-merge off |
| `vetoed`, `draft`, `auto-merge-off` | the forge's veto, or the label taken off |
| `label-ignored` | put by someone who may not push nor merge |
| `merge-unguarded`, `merge-token`, `merge-unsupported` | a base, a token or a forge it cannot act with |
| `checks-unread` | its head's checks could not be read: only the line's verdict counts |
| `merge-state-unread`, `verdict-unread` | its state or its head's verdict could not be read: left as it is |
| `auto-merge-refused`, `label-not-made` | the forge refused to turn auto-merge on, or to make the label |

## A red main

- **The net is main's run**, after each merge
  ([main after a merge](main-after-merge.md)): red, it says so in its one
  issue, and once on each open pull request into it, "base red, not
  yours".
- **Counted a week**, in that issue. Above one a week, turn on the
  forge's queue: GitHub's merge queue in an organisation, GitLab's merge
  train ([a merge queue or train](merge-queue.md)).

At `normal`, more words:
[what the summary says there](merge-policy-normal.md#what-the-summary-says).

What was tried, live: the documentalist's
[tried.md](../roles/documentalist/docs/tried.md#2026-10-12--the-merge-policy-cautious-live);
what is left: [status.md](../roles/documentalist/docs/status.md).
