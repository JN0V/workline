---
sources:
  - roles/committer/role.yaml#settings.subject-max
  - roles/committer/code/committer.go#checkRange
  - roles/committer/code/committer.go#titleFix
  - roles/committer/code/committer.go#squashed
  - internal/forge/github.go#github.Squash
  - internal/forge/gitlab.go#gitlab.Squash
  - internal/forge/forge.go#Squash
  - internal/forge/forge.go#TitleMax
  - roles/documentalist/code/documentalist.go#mergeRequest
  - internal/engine/engine.go#openMergeRequest
  - roles/documentalist/code/title_test.go#TestMergeRequestTitlesPassTheSquashRule
  - internal/sample/title_test.go#TestEngineTitlesPassTheSquashRule
checked: e81df7b34
verified: agent:claude-code
judged-in-parts: eb16524
---
# Committer — the squash title

Part of [the committer](../README.md), on a merge request.

A squash lands one subject on the base, which no commit carried. On a
merge request the forge may squash, that subject is checked by the
commits' rules, and blocks the same way
([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)):

- **The subject**: the title; on GitHub, set to take a lone commit's
  subject, that commit's when there is one.
- **Room for the forge's suffix**: GitHub adds ` (#N)`, so a title holds
  `subject-max` less its length: 65 characters for ` (#123)`. GitLab
  adds nothing.
- **The forge squashes** when its settings allow it: on GitHub, squash
  merging; on GitLab, any squash option but "Do not allow". A project
  that never squashes turns it off there.
- **Not read**: a forge that does not say, or could not be read, gives a
  warning: the title is not checked.
- **Checked on each push**, not on an edit. A title refused says the
  fix: edit the title, then re-run the job that checked it (one click)
  or push.
- **A person edits the title**: the AI rewrite is not built.
- **Squash only, the title alone blocks**: where the forge lands a merge
  request by squash and no other way, the branch's commit messages never
  reach the base. Their findings are warnings; secrets, forbidden terms
  and identities still block
  ([ADR-0051's amendment](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#amendment-2026-11-23-squash-only-the-title-blocks)).
  Read from the forge, nothing to set:
  - GitHub: squash merging on, merge commits and rebase merging off,
    the squash's message not "Default to commit messages";
  - GitLab: the squash option "Require", the squash template quoting no
    commit (`%{all_commits}`, `%{first_commit}`,
    `%{first_multiline_commit}`).
- **workline's own merge requests** (the documentalist's, the weekly
  sample's, a role's) get titles of 63 characters at most, room left
  for ` (#12345)`: a long path gives way to a count ("docs: bring 2 docs
  in line with the code"). Tests hold each one to these rules.
