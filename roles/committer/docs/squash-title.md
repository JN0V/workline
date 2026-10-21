---
sources: [internal/builtin/committer/committer.go, internal/forge/github.go, internal/forge/gitlab.go, internal/forge/forge.go]
checked: 8e3fad75
verified: agent:claude-code
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
  fix: edit the title, then re-run the job (GitHub's `judge`, GitLab's
  pipeline; one click) or push.
- **A person edits the title**: the AI rewrite is not built.
- **workline's own merge requests** (the documentalist's, the weekly
  sample's, a role's) get titles of 63 characters at most, room left
  for ` (#12345)`: a long path gives way to a count ("docs: bring 2 docs
  in line with the code"). Tests hold each one to these rules.
