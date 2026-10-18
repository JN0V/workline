---
sources: [internal/linters, internal/doctor/languages.go]
checked: c3ba67b5
---
# Languages and the tools that read them

`workline doctor` says, for each language a repository holds, which tools
read it, which it lacks, and the gate check to add
([ADR-0049](adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md),
[#244](https://github.com/JN0V/workline/issues/244)).

```
! shell (71 files, 17 told by their #! line): lint by nothing: shellcheck 0.11.0
    add to a gate's checks in .workline/config.yaml (docs/languages.md):
    - {id: shellcheck, run: "git ls-files -z -- '*.bash' '*.sh' | xargs -0 -r shellcheck", output: exit}
```

- **A warning, never a failure**: nothing blocks on it yet. A rule that
  blocks, when a project asks for it, comes later.
- **No agent is called**: the output is the same with an agent or none.
- **Nothing is built nor run**: files and configs are only read.

## Languages

- **Told from the files git tracks**: their extension, their name
  (`Dockerfile`), else their `#!` line (`#!/usr/bin/env bash`).
- **One line a language**; a file can count twice: a workflow is YAML
  and a GitHub workflow.
- **Vendored and generated code is left out**: what `.gitattributes`
  marks `linguist-vendored` or `linguist-generated`, as GitHub reads it,
  and the folders `vendor/`, `node_modules/`, `.pio/`, `third_party/`.
  `-linguist-vendored` takes a file back.
- **A file type no tool reads** (an image) is not listed.

## Four kinds

Each language is read for **lint**, **types**, **security** and
**dependencies**; lockfiles (`go.sum`, `package-lock.json`) are a line of
their own. Formatters are not counted: their fixes are applied, not
reported.

| Doctor says | Means |
|---|---|
| ✓ `language-covered` | a tool runs for every kind a tool is suggested for |
| ! `language-gaps` | a kind is read by nothing, and a tool is suggested: the gate check to add |
| · `language-unchecked` | nothing reads it, and no tool is suggested without its config: the tools that would, each with the config it needs |

## What counts as running

A tool runs when the project names it in:

- a gate check of `.workline/config.yaml`, said first: it is what the
  line reads;
- its CI: GitHub, Forgejo and Gitea workflows and actions,
  `.gitlab-ci.yml`, `.gitlab/ci/`;
- its hooks: `.pre-commit-config.yaml`, lefthook, husky;
- `Makefile`, `justfile`, `Taskfile.yml`, the scripts of `package.json`.

A tool named only in a comment, a step's name, an install
(`pip install ruff`), a download or an `echo` does not count. A tool run
by a script of your own is not seen: name it in a gate check.

## What is suggested, in order

1. **A tool whose config the repository holds**: `.golangci.yml`,
   `eslint.config.js`.
2. **The language's own toolchain**: `go vet`, `cargo check`, `tsc` (with
   a `tsconfig.json`), `pio check` (with a `platformio.ini`).
3. **The tools [trunk-io/plugins](https://github.com/trunk-io/plugins)
   suggests** wherever their files are: shellcheck, yamllint,
   markdownlint, actionlint, hadolint, ruff, osv-scanner…

- **A tool that depends on its project's config** (ESLint, clang-tidy) is
  suggested only when that config is there.
- **The version named** is the one trunk-io/plugins tested; pin it where
  you install the tool.
- **The gate check runs the tool as it is**: it reads your config, never
  one of workline's.

## The gate check to add

```yaml
gates:
  lint:
    checks:
      - {id: shellcheck, run: "git ls-files -z -- '*.bash' '*.sh' | xargs -0 -r shellcheck", output: exit}
routing:
  events: {merge-request: [committer, documentalist, gate:lint]}
```

- `output: exit`: the tool's exit code decides
  ([gates](spec/gates.md#where-gates-are-declared)).
- A tool reading files gets them from `git ls-files`, by extension: name
  the files told by their `#!` line too.
- A tool run once from a folder whose config is only in subfolders
  (`pio check`): one check from each such folder.
- The command is a start: flags vary by version, as for any gate check.

## The table it reads

- **Borrowed from trunk-io/plugins** (MIT, its notice kept in
  `internal/linters/trunk.json`): its file types, its linters, when each
  is suggested, the version it tested.
- **Regenerated, never edited**: `go generate ./internal/linters` reads
  the commit pinned in `internal/linters/linters.go`. Moving it is a pull
  request of its own.
- **Left out**: formatters, tools that read every file (secrets,
  spelling), file types told by their content (CloudFormation).
- **Kept by workline**: the language's own toolchain, and the kinds trunk
  files otherwise (mypy and pyright check types, brakeman security), in
  `internal/linters/ecosystem.go`.
