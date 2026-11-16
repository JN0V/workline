# Repository layout

*Target, decided by [ADR-0070](../adr/0070-a-new-role-runs-from-its-files-on-a-shared-pre-and-post.md#6-the-layout-reached-before-10);
reached by its five moves before v1.0. Until then, the
[role contract](role-contract.md#layout) describes the folders as they are.*

Two parts, one rule: **the core never imports a role; a role depends
on the core only.**

- **The common part**: the engine, the shared `pre` and `post`, the
  default line, shared skills and the contract every role keeps.
- **A role's part**: one folder, `roles/<name>/`, the same skeleton for
  every role, shipped or taken from elsewhere.

## The tree

```
workline/
├─ cmd/workline/            CLI entry only; imports roles/ to compile them in
├─ internal/                THE CORE: no role logic
│  ├─ engine/               run-role, shared pre and post, apply
│  ├─ contract/             role interfaces and the registry roles join
│  ├─ kit/                  helpers role code may use: test definitions,
│  │                        code picking, commit-message checks, verdicts
│  ├─ role/  routing/       role.yaml loader, routing
│  ├─ line/  sample/        the line, wake, act record, the generic draw
│  ├─ forge/  agent/        GitHub, GitLab, Forgejo; none, claude, cmd:
│  └─ …                     the other shared packages, as today
├─ roles/                   THE ROLES: one folder each, same skeleton
│  ├─ builtin.go            the compiled-in roles, as blank imports
│  ├─ committer/  developer/  tester/  reviewer/  documentalist/
│  ├─ inspector/  product-owner/  auditor/  judge/
│  └─ process-engineer/     prompt-only
├─ skills/                  shared Agent Skills a role may load
├─ defaults/                THE LINE AS SHIPPED
│  ├─ routing.yaml          the default line (was routing.default.yaml)
│  └─ ci/                   github/  gitlab/  forgejo/  job templates
├─ docs/                    FOR PEOPLE: guides, spec/, adr/, research/
├─ tests/                   CHECKS ACROSS THE ENGINE
│  ├─ conformance/          runner, engine cases; runs each role's cases/
│  ├─ evaluation/           real-agent evals (cost tokens)
│  └─ *_test.go             the checks once at the root (roles, agents' rules)
├─ .workline/               this repository's own line: config.yaml, roles.lock
├─ workline.go              go:embed what the binary ships (below)
└─ go.mod  README.md  AGENTS.md
```

- **Routing** is common: `defaults/routing.yaml` ships the line; a
  project's `.workline/config.yaml` changes it, guarded.
- **Skills** are common: `skills/<name>/SKILL.md`, loaded by a role
  that declares them ([shared skills](role-contract.md#shared-skills)).
- **Language packs** stay in `internal/packs/`, as data.

## A role folder

```
roles/<name>/
├─ role.yaml        required  the contract: mission, events, inputs, acts, duties
├─ README.md        required  for people, opens with a diagram
├─ docs/status.md   required  criteria to leave beta, built, missing
├─ cases/           required  its conformance cases
├─ prompts/         optional  what the agent reads: instruction, policy,
│                             persona, knowledge/, lenses/, tasks/, skills/
├─ criteria/        optional  <kind>.md: what a well-done act of that kind is
├─ code/            optional  Go package compiled in (shipped roles only)
└─ pre, post        optional  executables (a role from elsewhere: if trusted)
```

- **What runs**: the role's `code/` hook, else its `pre` or `post`
  file, else the shared one; each of `pre` and `post` on its own.
- **`criteria/`**: per kind of act, the fixed, versioned question saying
  what a well-done act is; the auditor draws acts at random and judges
  them against it, and its verdicts live on the forge, never here.
- **The auditor's criteria win**: `roles/auditor/criteria/<role>/<kind>.md`,
  guarded, is read before the role's own `criteria/<kind>.md`.
- **The binary ships** `role.yaml`, `prompts/` and `criteria/` of each
  role, named one by one in `workline.go`; never `cases/`, `docs/` nor
  `code/` as files. A check in `tests/` fails if a case is embedded.

## Kinds of role

| Kind | Beyond the required items, its folder holds |
|---|---|
| Fully coded (committer) | `code/`; `prompts/` only for an agent fallback |
| Partly coded (developer, tester, reviewer…) | `code/` for what converges, `prompts/` for judgement, `criteria/` |
| Prompt-only (process engineer) | `prompts/`, `criteria/`; the shared `pre` and `post` run it |
| External (another repository) | as prompt-only; `pre`/`post` only if trusted; never `code/` |

## A person's overrides

A file in the project or the person's config folder replaces the
shipped one at the same place in the role's folder:

```
.workline/roles/<role>/prompts/policy.md        # the project's
~/.config/workline/roles/<role>/prompts/policy.md   # the person's
```

- **The old flat path** (`.workline/roles/<role>/policy.md`) is no
  longer read once [move b](../adr/0070-a-new-role-runs-from-its-files-on-a-shared-pre-and-post.md#6-the-layout-reached-before-10)
  lands: `workline doctor` names each file found there and where it must
  go, and the role's run says it, never ignores it (principle 12).

## An external role

A project names it in its config, pinned:

```yaml
# .workline/config.yaml
roles:
  release-notes:
    from: "https://git.example.com/team/roles.git//release-notes"
    ref: v1.4.0        # a tag or a commit, never a branch
    trust: files       # files: prompts only; code: its pre/post may run
```

- **`from`**: a git URL and the folder in it, `repo//path`.
- **`ref`**: the version fetched.
- **`.workline/roles.lock`**: the commit and content digest of what was
  fetched, checked on every run, so every machine runs the same role.
- **`trust`**: `files` by default; `code` lets its `pre`/`post` run in
  the trusted prepare step, with the forge and agent keys.
- **A change of `from`, `ref` or `trust`** is a pull request on a
  guarded file, the person's label.
- *Not built yet*: the engine refuses `from` today; `--roles <dir>` and
  `WORKLINE_ROLES` exist.

## Borrowed

- **VS Code** keeps built-in extensions in its repository, shaped like
  marketplace ones: shipped and external roles share one skeleton.
- **Claude Code plugins**: an optional subfolder present means the
  capability is there (`code/`, `prompts/`, `criteria/`).
- **Terraform providers, GitHub Actions**: a source and a version,
  pinned in a lock file with a digest.
- **Caddy modules, Go's `database/sql` drivers**: a package registers
  itself in `init()`; one list of blank imports compiles it in, so the
  binary stays one ([ADR-0001](../adr/0001-engine-in-go.md)).
- **Go project layout**: `cmd/` for entry points, `internal/` for the
  core.
