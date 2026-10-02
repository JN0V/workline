# ADR-0016: What workline writes goes where the project lives

- **Status:** accepted; built 2026-10-02
- **Date:** 2026-10-02
- **Amends:** ADR-0010 and ADR-0015, whose merge requests, tracking issues
  and comments assumed GitHub or GitLab

## Context

The roles write to a forge: a merge request's comment, gardening's merge
requests (ADR-0006, ADR-0010), the tracking issues, the weekly sample's
issue, label and merge request (ADR-0015), a release's notes. The engine
spoke GitHub and GitLab only. A project elsewhere — Gitea, Forgejo,
Codeberg, Bitbucket — could not take those writes, and a project with no
forge at all (`forge: none`, the default) had them refused, but for one: an
issue a role opened went to `.workline/issues/` in the working tree, where
the next `git add` would commit it — work state in the history, which
docs/BACKLOG.md asks to stop.

The person's rule: on GitHub into GitHub, on GitLab into GitLab, on another
forge into that forge, and with no forge, purely local.

What exists: the `cmd:` agent, a command plugged by a small contract (the
prompt in, the proposals out; internal/agent/command.go); git's own
`.git/` folder, which is never committed and already holds the run folders
(`.git/workline/runs/`); `git-bug` and `git-appraise` keep issues and reviews
in a repository's refs — more than workline needs, and a tool to install.

## Decision

- **`forge: local`** keeps the writes in the clone, never committed: each
  issue a Markdown file, `.git/workline/issues/<n>.md` — title, state and
  labels in its front matter, then the body, then each comment, kept in
  place by its marker; each merge request a local branch, nothing pushed,
  recorded the same way in `.git/workline/merge-requests/<n>.md`; a
  release's notes in `.git/workline/releases/<tag>.md`. The person merges
  with git: a merge request whose branch its base holds reads as merged, one
  whose branch is gone as closed. `workline issues` lists and shows them.
  Every write that reaches a forge works on it: the merge request's comment
  and labels, `--push-to-merge-request` (a commit on the local branch),
  `--open-merge-request`, the tracking issues, the sample's `--apply`, the
  release, `workline item ready --forge local`, and `workline docs`
  through the engine.
- **`local` in CI refuses its writes** — when `CI`, `GITHUB_ACTIONS` or
  `GITLAB_CI` is set: the job's clone is thrown away, and what it would
  hold with it. Refused, not warned: a warning in a green job's log is read
  by no one, and the write is lost all the same (principle 12). Only the
  writes: a project whose config says `local` for its laptops still runs in
  CI what writes nothing, and passes `--forge` to the jobs that write.
- **`forge: cmd:<command>`** plugs any other forge: one JSON request per
  operation on the command's input, one JSON answer on its output, an exit
  other than 0 the forge unreachable (docs/spec/forge-command.md). Git stays
  the engine's: it pushes the branches, the command does what is the
  forge's. A sample for Forgejo and Gitea is in ci/forgejo/.
- **`none` stays "no forge"**, and the default. A write that needs a forge
  is refused, loud, naming `forge: local` — the issue a role opens too,
  which went to `.workline/issues/` in the working tree before: work state
  is never committed, so it goes to a forge, or `local`, or nowhere.
  `none` is not made an alias of
  `local`: the default would then start keeping state in every clone
  unasked, and in CI — where a clone is thrown away after the job — a
  write refused today would pass silently into a folder no one reads
  (principle 12). A project that wants the local forge says so.

## Consequences

- Every project has a place for what workline writes: its forge, a command
  for a forge not spoken natively, or its own clone.
- What the local forge holds is the clone's alone: not shared, not backed
  up, gone with the clone. A team shares a forge.
- A project that relied on `.workline/issues/` sets `forge: local`, or its
  forge; the files already there are left to the person to move or delete.
- `local` in CI is caught by the variables CI sets; a CI that sets none of
  them is not.
- Native Gitea and Forgejo support, and the GitLab side of the sample, are
  not built or not tried (docs/BACKLOG.md).
