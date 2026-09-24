---
sources: [internal/builtin/committer, roles/committer/role.yaml]
checked: d30d22a
---
# Committer

What runs, for humans. The AI never reads this file.

## Check (`pre`, no AI)

On `commit-msg` (the git hook), the message being written; on `merge-request`,
every commit of the range given as `--input range=<base>..<head>`.

| Rule | Refuses |
|---|---|
| `format` | a subject that does not read `type(scope): summary`, with a type from `types` |
| `blank-after-subject` | a second line that is not empty: git would read it as part of the subject |
| `subject-length` | a subject over `subject-max` characters (72) |
| `body-length` | a body over `body-max-lines` lines (12), not counting comments, blank lines and trailers |
| `internal-code` | a reference that means nothing outside the project (`AC-3`) in the subject; it belongs in a trailer (`Refs: AC-3`). `internal-codes-allow` lists standard identifiers that look like one (`SHA-256`, `RFC-1234`) |

Messages git writes itself (`Merge …`, `Revert "…"`, `fixup! …`, `squash! …`,
`amend! …`) are not checked. A message that passes asks no agent.

## Rewrite (the agent)

On `commit-msg`, a refused message goes to the agent with the findings and the
staged diff. It proposes one `commit-message`, or a `note` when the diff mixes
unrelated changes and the commit should be split. On `merge-request`, the
commits are already made: the verdict lists them, to fix with `git rebase -i`.

## Judge (`post`, no AI)

- The rewrite must pass the same checks.
- It must keep every trailer of the original (`Co-Authored-By:`, `Refs:`…), on
  its own line in the last paragraph (`trailer-dropped`).
- One commit, one message: several proposals are refused (`several-messages`).
- A `note` ends the run as `human`: a person splits the commit.

A refused rewrite is asked once more, of a stronger model (`promote-after: 1`).
A rewrite that passes replaces the message, and the commit goes on: read it in
`git log`, since it is no longer the one you wrote.

## Without AI

A bad message still fails, with the findings; you rewrite it. The message is
kept in the file git named, so nothing typed is lost.

Not built yet: running the checks listed in `uses` (forbidden terms, identity,
secrets); the global hook hands over to them instead, when they were installed
before workline.
