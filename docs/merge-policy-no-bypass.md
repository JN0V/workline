---
sources: [internal/doctor/guard.go, internal/forge/github_merge.go]
checked: 70dfeb18
verified: agent:claude-code
---
# Nobody bypasses `main`

Every merge into `main` goes through its checks and `workline/accepted`,
the person's own included: a session holding their token cannot force
past the line
([ADR-0068 §6](adr/0068-the-line-steps-back-on-its-own-drift.md#6-nobody-bypasses-on-main)).
Nothing legitimate needs a bypass: workline's App, releases, gardening
and a labelled pull request all merge with none
([the forge read](research/nobody-bypasses.md)).

## On GitHub

- **`main`'s ruleset, an empty bypass list**: Settings → Rules →
  Rulesets → the ruleset on `main` → Bypass list, every entry removed.
  Or with `gh`:

  ```sh
  gh api repos/{owner}/{repo}/rulesets/<id> > ruleset.json   # kept, to restore
  jq '{name, target, enforcement, conditions, rules, bypass_actors: []}' ruleset.json |
    gh api -X PUT repos/{owner}/{repo}/rulesets/<id> --input -
  ```

- **Only `main`'s**: a ruleset whose bypass list is an allow-list, as
  "only workline's App pushes the release branch", keeps it.
- **`workline doctor` checks it**, one API read a ruleset, no agent:
  - `no-bypass`: GitHub says this token can bypass none of the default
    branch's rulesets (`current_user_can_bypass: never`);
  - `bypass-allowed`, a warning: one it can bypass, named, with how to
    empty its list;
  - `bypass-unread`: the rulesets could not be read, said.
- **What a person then does**: puts `workline:accepted` on the pull
  request; workline's App merges it once its checks are green
  ([the merge policy](merge-policy.md)). `gh pr merge --admin` and a
  push to `main` are refused.

## The emergency

When no head can get `workline/accepted`, say `follow` broken on `main`
so its own fix cannot merge:

1. **Add yourself**: the ruleset's bypass list, "Repository admin", "For
   pull requests only". Say why on the fix's pull request. By command:
   read the ruleset with `gh api repos/{owner}/{repo}/rulesets/<id>`,
   then PUT it back with
   `"bypass_actors": [{"actor_id": 5, "actor_type": "RepositoryRole", "bypass_mode": "pull_request"}]`
   (`gh api -X PUT repos/{owner}/{repo}/rulesets/<id> --input ruleset.json`;
   5 is the Repository admin role).
2. **Merge the fix**: `gh pr merge <number> --squash --admin`.
3. **Remove yourself**: empty the list again, as above;
   `workline doctor` says `no-bypass` again.

The ruleset's history and the rule suites keep each step; the
emergency is a deliberate edit, never a flag on a routine command.
[The wake](wake.md) prints the ruleset's change at the next merge, and
a head merged without the App's `workline/accepted`.

## On GitLab Premium

From GitLab's documentation, not tried on a project yet:

- **Settings → Repository → Protected branches**, `main`: "Allowed to
  push and merge": No one; "Allowed to merge": Maintainers; force push
  off; "Code owner approval" on. Anyone allowed to push skips the
  approvals, code owners included.
- **Settings → Merge requests**: "Pipelines must succeed"; approval
  rules locked in merge requests.
- **`CODEOWNERS`** owns the CI files: a merge request runs its own
  branch's pipeline ([guarding the bounds' job](gitlab/bounds.md)).
- **The emergency**: a Maintainer changes the protection, merges, puts
  it back; the project's audit events record it.

## The limits

- **The label and the ruleset stay in the person's reach**: a session
  holding their token can set `workline:accepted`, or edit the ruleset.
  It cannot force without breaking a written rule, and leaves a trace.
- **A fine-grained daily token without the Administration permission**
  would hold the ruleset too: an option, not tried, never required.
- **With no forge**, nothing enforces who merges
  ([the line with no forge](local-line.md)).
