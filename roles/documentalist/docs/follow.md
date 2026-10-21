---
sources: [internal/engine/settle.go, internal/engine/follow.go, roles/documentalist/role.yaml]
checked: 0839bca7
verified: agent:claude-code
---
# Documentalist — a pull request in conflict on workline's lines

Two pull requests often move the same doc's `checked`: the one merged
first leaves the other in conflict on that line alone. With
`update-branches: conflicts`, when the default branch moves, `workline
follow` merges it into such a pull request, workline's lines settled
([ADR-0041](../../../docs/adr/0041-workline-settles-its-own-lines-when-the-base-moves.md)).
No agent. Off by default.

```yaml
# .workline/config.yaml
roles:
  documentalist:
    settings:
      update-branches: conflicts   # default: off
```

The `follow` job of the [CI templates](../../../docs/ci.md) runs it on each
push to the default branch.

## Which pull requests

- **Open, into the branch that moved, from this repository**, opened by a
  person.
- **Left without a word**: workline's own (`workline/…`, kept on the base
  apart: [gardening](gardening.md#when-the-base-moves)); a bot's — opened by an
  app or a token's user, a release tool's branch (`release.branches`),
  `dependabot/…` or `renovate/…` — which its bot rebuilds. On GitLab, a
  bot is whom the users API calls one: a project or group access token's
  user, a service account.
- **Left, named once in the summary**: a long-lived branch, the base of
  another open pull request (`develop`, `release/1.x`). One the rule
  misses, protected, gets `merge-refused`.
- **Left, named**: one from a fork, in conflict (the job's token may not
  write there).
- **Left**: one behind but merging clean: it merges as it is
  ([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-2-strict-up-to-date-dropped)).

## Which lines are workline's

- In a doc's header — a `---` front matter, or README's `<!-- workline
  … -->` — the keys `checked`, `judged`, `judged-in-parts`, `verified`,
  each on its own line. Docs are what the `docs` setting names.
- A derived block (`<!-- workline:derive … -->…<!-- workline:end -->`).
- Everything else is the person's: the text, the code, `sources` and the
  header's other keys.

## How it is settled

- **The header, key by key**: a key one side changed keeps that side's
  value; one of the four both changed, the base's; `sources` or another
  key both changed is left to the person.
- **A derived block**: the base's, which gardening regenerates there. A
  marker with no end of its own is no block: left as it is.
- **The rest**: git's own merge; a conflict there is the person's.
- **Any conflict left**: nothing is pushed, the files named.

## What it writes

- **A merge commit, never a rewrite**: the branch's tip first parent, the
  base's second, as the forge's Update branch makes it; every commit of
  the branch keeps its id.
- **The forge's message**, `Merge branch 'main' into <branch>`, the files
  settled, the trailer `Workline-Role: documentalist`: the line on it
  judges the commits before it, through its first parent, so a red verdict
  stays red; the reviewer asks no lens for what it read already, and a
  block of its own stands
  ([#326](https://github.com/JN0V/workline/issues/326)).
- **Pushed as a fast-forward, never forced**, a lease on the tip it read:
  a push of theirs meanwhile stays, a branch deleted is not brought back;
  the next push to the base tries again.
- **Signed when git is set to sign** (`commit.gpgsign`), with its key, as
  `git commit` signs: a forge refusing unsigned commits takes it. A key
  missing or refused: nothing pushed, the job fails and says so
  ([signed commits on GitLab](../../../docs/gitlab/signed-commits.md)).
- **Only files are settled**: a submodule or a link in conflict is the
  person's.
- **The token**: on GitHub, the App's runs the pull request's checks
  again; the job's own leaves them for a person. On GitLab, the project's
  bot.

## For whoever works on the branch

- **Pull before pushing again**: git refuses the push otherwise ("fetch
  first"). A force-push over the merge, or a rebase on the base, brings
  the header conflicts back.
- **The line on the merge judges again** each doc whose `checked` took
  the base's value: the branch's change to its sources is still to read.

## In the job's summary

| Rule | Said when |
|---|---|
| `merged` | the base merged in; the files settled |
| `not-merged` (warning) | a conflict of the person's too, its files named; or a fork's |
| `merge-refused` (warning) | the branch moved or was deleted meanwhile, or the forge — or a hook on the machine — refused the push, in its words |
| `conflict-unread` (warning) | git could not tell whether it merges |
