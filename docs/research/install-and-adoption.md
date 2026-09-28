# Install and adoption — doctor, setup, init

**Verdict.** Take the doctor from pnpm and flutter, the re-runnable init from
terraform and mise, the prompts from gh. Nobody prints the command that
installs a missing tool on this machine, and nobody proposes which code a doc
describes: both are ours to build. Findings as of 2026-09-28.

## Doctor

| Tool | Takes |
|---|---|
| [pnpm doctor](https://pnpm.io/cli/doctor) | each check `pass`, `warn` or `fail`; exits non-zero on a fail, never on a warning; `--json` |
| [flutter doctor](https://github.com/flutter/flutter/blob/master/packages/flutter_tools/lib/src/doctor.dart) | one line per check marked ✓ ! ✗, grouped under a heading; a closing summary; only a missing check fails |
| [mise doctor](https://mise.jdx.dev/cli/doctor.html) | read-only, documented as such; machine and project checked apart |
| [claude doctor](https://code.claude.com/docs/en/setup) | a state and its cause on each line, a fix under a warning |
| [gh auth status](https://cli.github.com/manual/gh_auth_status), [brew doctor](https://docs.brew.sh/Manpage) | the opposite choice: exit 1 on any warning, which teaches people to ignore the exit code |

## Setup and init

- **Safe to run again**: [terraform init](https://developer.hashicorp.com/terraform/cli/commands/init)
  never deletes existing configuration; `-input=false` fails where an answer was needed.
- **Merge, do not overwrite**: [mise use](https://mise.jdx.dev/cli/use.html) writes
  into the nearest existing config file.
- **Prompts**: [gh](https://cli.github.com) asks for what is missing on a
  terminal, and without one fails naming the flag; `npm init -y` takes every
  default; [commitizen init](https://commitizen-tools.github.io/commitizen/commands/init/)
  offers what it detected as the default.
- **Hooks already there**: pre-commit runs old and new side by side; lefthook
  refuses a foreign `core.hooksPath` unless forced. workline already hands over
  (`internal/hooks`).

## Install commands

From each project's own page: gitleaks has `brew` and release binaries
([README](https://github.com/gitleaks/gitleaks)); `go install
github.com/zricethezav/gitleaks/v8@latest` is not documented but works (tried
2026-09-28). lychee: `brew`, `cargo install`, many distribution packages
([README](https://github.com/lycheeverse/lychee)). Claude Code: the script
`curl -fsSL https://claude.ai/install.sh | bash`, `brew install --cask
claude-code`, or npm ([setup](https://code.claude.com/docs/en/setup)).

## Mapping docs to code

[Fiberplane drift](https://blog.fiberplane.com/blog/drift-documentation-linter/)
is closest to `sources` and `checked`: frontmatter naming files, a symbol and
the commit last reviewed. [Swimm](https://swimm.io) links docs to code, and is
proprietary. Both are adopted by hand, one link at a time: none proposes the
links on an existing repository.
