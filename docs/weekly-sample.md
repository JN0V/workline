---
sources: [internal/sample, internal/builtin/documentalist/sample.go, ci/github/workline-sample.yml, ci/gitlab/workline.gitlab-ci.yml, roles/auditor, roles/product-owner/role.yaml]
checked: 8425efe1
judged: c9d96e2
verified: agent:claude-code
---
# The weekly sample

Every week, the auditor re-checks one in ten of every role's acts, per
kind of act
([ADR-0046](adr/0046-the-auditor-re-checks-every-roles-acts.md)): the
docs the documentalist vouched for, read whole against their sources
([ADR-0014](adr/0014-checked-is-earned-by-what-was-read.md), step 4), the
product owner's acts done alone, the findings the reviewer's judge kept
or dropped on every run
([a line each in its summary comment](../roles/reviewer/docs/judge.md#every-finding-judged-kept-for-the-weekly-sample),
and in that of its lenses that warn, [read apart](../roles/reviewer/docs/apart.md)),
the committer's rewrites. How to schedule it: on
[GitHub](ci-github.md), [GitLab](ci-gitlab.md#4-the-weekly-sample),
[another forge or none](ci-other-forges.md).

```yaml
roles:
  documentalist:
    settings:
      sample: {judge: "claude:opus"}   # Sonnet vouches by default: Opus reads
```

## Who reads

- **The judge**: an `--ai` value, here or as the CI variable
  `WORKLINE_JUDGE`; a `cmd:` running another provider's agent stands
  further apart. On workline itself, Opus reads what Sonnet vouched for:
  another model of the same provider, the best independence there.
  Never a model that acted. Without a judge, what is drawn is listed for
  a person.
- **False** only with the material's words, which the engine finds again;
  else the act is for a person.
- **A doc whose `checked` moved after parts**
  ([#482](https://github.com/JN0V/workline/issues/482)): its sources do
  not fit to be read whole, so it is listed for a person, never called
  unearned nor put back.
- **What a product owner's act was done on**: the issue's text it read,
  found by the digest its record keeps — the body now, else the version
  GitHub's edit history keeps; for a refine, the issue as it reads now
  beside it —, and the code at the commit it read. A text gone from the
  forge is not read, for you: GitLab keeps no edit history. An act
  recorded before this record is read as its issue reads now, and named.
- **A squash**: a merge request's commits are read on its head, which
  the forge keeps (GitHub's `refs/pull/<n>/head`, GitLab's
  `refs/merge-requests/<iid>/head`). The commit there that moved a doc's
  `checked` says whose the move was, and which model vouched: a person's
  move is not drawn. With no forge, a `checked` naming one of those
  commits is not read, and the reason says to give one (`--forge`).
- **A dispute**: a verdict against the act goes to three judges, Claude's
  three models by default (`sample: {panel: […]}`), each choosing between
  the two readings, not told whose; the majority settles it.

## What it writes

- **One issue**, "workline: the weekly sample of every role's acts" (its
  former title, "… of the docs vouched for", renamed), a comment a week,
  its acts numbered.
- **A `checked` found false** labels it `documentalist-step-0` and opens a
  merge request putting it back, for you to review.
- **A true finding the judge dropped**, still in the code, is opened as an
  issue, `needs-triage`, three a week at most.
- **The product owner's acts**: each one a person undid is said, and the
  level its acts done alone suggest, never set
  ([ADR-0033](adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)).
  Its former issue of its own, still open, is told where the weeks went,
  and closed.
- **What could not be drawn** is listed, with why: the findings judged
  whose lines a merge request's summary comment no longer holds, past its
  size; a comment written before the lines, its last run's alone.
- **Your answer**: 👍 on the week's comment agrees with all of it; 👎 and a
  reply naming rows (`not 3, 7`) contests those, the rest agreed. A reply
  answers the week it follows, or the week it names (`2026-W41: not 3`).
  The count, one comment on the same issue, is written again each week
  from every week
  ([ADR-0046](adr/0046-the-auditor-re-checks-every-roles-acts.md#kept-and-counted-with-the-maintainers-agreement)).

## What it costs

- **One call an act**, about 4k to 12k tokens; about 380k on a week as
  busy as workline's busiest
  ([ADR-0046](adr/0046-the-auditor-re-checks-every-roles-acts.md#cost-a-week)).
- **The cap**: `sample: {ai-max-tokens: 400000}` by default; the acts
  left past it are listed unread.

## Settings

- **`after: <tag>`** beside the judge leaves out what was vouched for
  before that commit: before the release whose engine earns `checked`
  (v0.2.1 on workline), a `checked` could be moved unread. A commit is
  best quoted (`after: "7515148"`): unquoted, one with a leading zero is
  refused, YAML reading it as another number.
- **The week** is the last whole one; for another (`--week 2026-W40`),
  each forge page says how.
- **Nothing to turn on for the product owner**: its acts are drawn like
  any role's. Its former `weekly-sample` setting is no longer read, and
  its run says so.
