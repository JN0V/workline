---
sources:
  - defaults/ci/github/workline-apply.yml#jobs.apply
  - defaults/ci/github/workline-apply.yml#jobs.follow
  - defaults/ci/github/workline-gardening.yml#jobs.apply.environment
  - defaults/ci/github/workline-gardening.yml#jobs.apply.permissions
  - defaults/ci/github/workline-sample.yml#jobs.write.environment
  - defaults/ci/github/workline-answer.yml#jobs.apply.environment
  - defaults/ci/github/workline-develop.yml#jobs.apply.environment
  - defaults/ci/github/workline-develop.yml#jobs.review-apply.environment
  - defaults/ci/github/workline-bounds.yml#jobs.bounds-apply.environment
  - internal/forge/annotate.go#FindingsCheck
  - internal/forge/annotate.go#GatesCheck
  - internal/engine/annotate.go#Annotate
  - internal/forge/merge.go#github.mergeToken
checked: e2e165c2d
verified: agent:claude-code
---
# The GitHub jobs that write

Part of [what the GitHub jobs may do](jobs.md): `apply`, `follow` and
the other jobs holding the write token.

- `apply`, `follow` and each pair's second job hold the write token, the
  App's when there is one, and no AI key, in the environment
  `workline-app`. `follow`'s merge into a pull request runs its checks
  again only with the App's token; the [merge policy](../merge-policy.md) acts only with it,
  and needs "Allow auto-merge" (Settings → General).
- `apply` reads the pull request's run as data, only the agents'
  proposals ([more](../commands.md#workline-apply)), whatever `propose`
  concluded: one the reviewer holds still gets its comment
  ([#226](https://github.com/JN0V/workline/issues/226)). Its job is in
  Actions, "workline apply"; the check is `propose`'s.
- `apply` posts the reviewer's findings on their lines, a pull request
  review (`pull-requests: write`); a token without it leaves them in the
  summary comment, said
  ([more](../../roles/reviewer/docs/lines.md)).
- `apply` writes the documentalist's findings as the annotations of the
  check `workline findings`, the [gates'](../gates-in-ci.md) as those of
  `workline gates` (`checks: write`; an App's **Checks**): beside their
  lines, none a conversation; a refusal, said.
