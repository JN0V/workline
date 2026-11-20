---
sources:
  - cmd/workline/main.go#initCmd
  - internal/routing/adopt.go#RoutePrePush
  - roles/documentalist/code/adopt.go#noSources
  - roles/documentalist/code/adopt.go#sourcesTask
  - roles/documentalist/code/adopt.go#judgeSources
  - roles/documentalist/code/byname.go#codeByName
  - roles/documentalist/code/byname.go#byName
  - roles/documentalist/code/byname.go#isTest
checked: e2e165c2d
verified: agent:claude-code
---
# Documentalist — adopting a repository

Part of [at the push and the adoption](push.md).

On a repository where no doc says what code it describes, nothing can be
found suspect. `workline init` routes `pre-push` to the committer and the
documentalist, unless the project routes it already, then runs the
documentalist on `init`.

## With an agent

- Each doc the `docs` setting covers whose header has no `sources` is put
  before the agent with the repository's files. The agent proposes the
  code it describes — `sources: []` for a doc describing none, a decision
  or a changelog.
- Its `checked` is the commit that last changed the doc, never HEAD: the
  doc was written against the code as it was then, so a source changed
  since makes it suspect, to be judged (`workline docs`, a merge request)
  rather than vouched for unread.
- The judge refuses a patch touching more than the header, a source the
  repository does not hold, or another `checked`.
- The patches land in the working tree for the person to review and
  commit.

## Without an agent

- Each doc is listed (`no-sources`) with the commit its `checked` would
  name.
- When its name or a folder holding it up to `docs/` is a code folder's or
  file's, those sources are proposed — the shallowest, tests aside,
  `docs/cli.md` → `src/cli/` — for the person to review: never written.
- One finding (`sources-by-name`) says how many docs matched. On six public
  repositories: 0 to 5 of 5 to 30 docs, and 261 of backstage's 452, about
  half of them right (docs named after its plugins and packages; `faq.md`
  under `features/` matched a `features/` folder far away).
