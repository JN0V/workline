---
sources: [internal/builtin/documentalist/documentalist.go]
checked: 347b403
---
# Several repositories — v1 (draft)

A product often spans several repositories: the code in one, the user guide in
another, the backlog in a third. Links, suspicion and pending work must cross
those boundaries the same way they work inside one repository.

## The dependent declares, the source does not need to know

A repository declares the other repositories it depends on, in
`.workline/config.yaml`:

```yaml
repos:
  api:  {url: https://gitlab.example.com/team/api.git,  branch: main}
  docs: {url: https://gitlab.example.com/team/user-guide.git, branch: main}
```

A doc then points into them with the repository's name:

```yaml
sources: [api:docs/tech/auth.md#token-refresh, api:src/auth/token.go]
checked: {api: 3f2a91c}          # one commit per repository it depends on
```

The source repository never lists who depends on it — the same way a library
does not list its users. This keeps each repository independent: adding a
consumer never requires a change in the source.

## Pull, not push

The dependent checks its sources; the sources do not notify it.

- **In the dependent repository** — on its own merge requests and on a schedule,
  `pre` fetches the declared repositories read-only, with enough history to
  cover each `checked` commit, and finds what became suspect.
- **For a whole team** — the central runner (the renovate-runner pattern) walks
  every repository of the group, so it sees the graph in both directions. When
  a change in `api` makes a page of the user guide pending, it opens or updates
  the pending issue *in the user guide's project*, before anyone there runs
  anything.

## Access

- Reading another repository needs only read access; the prepare step has it,
  the agent does not need it.
- Writing stays local: a role writes to its own repository. The only thing it
  may create elsewhere is an `issue` (or a comment on one) in a repository
  declared in `repos`, and only in central mode, where the applier holds a
  group token.
- A repository that cannot be reached is `blocked-external`, not "no change":
  a source you could not read has not been checked.

## Scope across repositories

A work item's scope may name paths in several repositories
(`api:src/export/**`, `docs:guide/export.md`). Each repository's run applies the
part that belongs to it; a patch is never applied across repositories.
