# Inspector — which tools a project needs, and AI in their place (2026-10-18)

**Question.** Should the inspector ([#202](https://github.com/JN0V/workline/issues/202))
make sure each language a project holds has a linter and a static
analyser, suggest the right one, install it, and lint with an AI where
no tool exists? A panel of four (users, principles, prior art, skeptic)
answered; this card checks what it cited. Extends
[inspector.md](inspector.md); the decision is
[ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md).

**Verdict.**

- **Detection and suggestion are solved, as data**: GitHub Linguist
  tells languages, trunk-io/plugins tells which linter fits, both MIT.
  Project-dependent linters are suggested only when their config is
  there.
- **Pinned installs exist in user space**: mise's aqua backend verifies
  checksums and signatures itself; pre-commit's `rev` is immutable;
  bumpers keep them current.
- **Licences decide what may be suggested**: CodeQL only on open-source
  code (else a paid licence), Semgrep's registry rules for internal use.
- **One product lints with an LLM where no rule exists**, GitHub Code
  Quality, and keeps it off the pull request and off the gate. The
  others put the model beside the tools, never in their place.
- **Embedded C and C++ need a build** to be read well.

Searched in the ecosystem's words: *language detection*, *linguist*,
*suggest_if*, *known_good_version*, *descriptor*, *default setup*,
*automatic analysis*, *aqua*, *lockfile*, *autoupdate*, *AI findings*,
*LLM advisor*, *MISRA rule texts*. Read on 2026-10-18; "inferred" marks
what no page states; "corrects" marks where the panel was wrong.

## Telling languages and suggesting tools

| Source | Found | Link |
|---|---|---|
| GitHub Linguist (MIT) | 836 languages by extension, file name and interpreter (a shebang); `.gitattributes` `linguist-vendored`, `linguist-generated`, `linguist-documentation` | [languages.yml](https://github.com/github-linguist/linguist/blob/73ee9a3b277f398c37ca4ab0dea0193849de6460/lib/linguist/languages.yml), [overrides](https://github.com/github-linguist/linguist/blob/main/docs/overrides.md) |
| trunk-io/plugins (MIT) | 116 linter folders, about 90 file types (extensions, shebangs: `shell` is `.sh`, `.bash`, `bash`, `sh`); each linter a `known_good_version` (tested, not latest: ESLint 8.10.0), download URLs per OS and CPU, `suggest_if`: `files_present` (shellcheck, golangci-lint v2), `config_present` (ESLint, clang-tidy), `never`; no checksum in the three definitions read | [plugin.yaml](https://github.com/trunk-io/plugins/blob/8318561ebafe9973fb481281557fb0f089e8e283/linters/plugin.yaml), [eslint](https://github.com/trunk-io/plugins/blob/8318561ebafe9973fb481281557fb0f089e8e283/linters/eslint/plugin.yaml), [clang-tidy](https://github.com/trunk-io/plugins/blob/8318561ebafe9973fb481281557fb0f089e8e283/linters/clang-tidy/plugin.yaml) |
| Qlty (BSL 1.1) | its grant bars use "to produce … an Artificial Intelligence Coding Service"; GPL-3.0 from 2028-12-10. Read, never copied: ESLint is taken from the project's `package.json` (`package_file_candidate`), suggested on its config | [LICENSE.md](https://github.com/qltysh/qlty/blob/main/LICENSE.md), [eslint](https://github.com/qltysh/qlty/blob/354dd5b99199c56675d01b227e3567689971609a/qlty-plugins/plugins/linters/eslint/plugin.toml) |
| MegaLinter (AGPL-3.0) | 57 descriptors: `file_extensions`, and `file_contains_regex` for shebangs (`^#!/bin/sh`) on files with no extension | [bash descriptor](https://github.com/oxsecurity/megalinter/blob/66064a317ff41c7e9bd451967a08aa4f5b134b7c/megalinter/descriptors/bash.megalinter-descriptor.yml) |
| Super-Linter (MIT) | extension and shebang tests; a file marked `@generated` skipped when `IGNORE_GENERATED_FILES`; `VALIDATE_ALL_CODEBASE=false` lints changed files only | [detectFiles.sh](https://github.com/super-linter/super-linter/blob/660f7941b2c2793aa514a737ee4043f8e9c2269f/lib/functions/detectFiles.sh) |
| CodeQL default setup | "If the code in a repository changes to include any CodeQL-supported languages", it adds them; if that fails, the previous configuration resumes | [setup types](https://docs.github.com/en/code-security/concepts/code-scanning/setup-types) |
| SonarQube Cloud | automatic analysis "only if *at least 20%* of the lines of code" are in a supported language; not Objective-C, Dart, Rust; no coverage, no external reports | [automatic analysis](https://docs.sonarsource.com/sonarqube-cloud/analyzing-source-code/automatic-analysis.md) |

Inferred: Linguist answers "which languages", trunk "which linter for
them"; neither reads what the project already runs (its CI steps,
pre-commit, a `lint` target).

## Pinned installs and bumpers

| Tool | Found | Link |
|---|---|---|
| mise, aqua backend | "mise implements checksum, GitHub artifact attestation, Cosign, SLSA and Minisign verification itself", on by default; a verified `mise.lock` is trusted; installs under `~/.local/share/mise` | [aqua](https://mise.jdx.dev/dev-tools/backends/aqua.html), [directories](https://mise.jdx.dev/directories.html) |
| pre-commit | `rev` "an immutable ref (such as a tag or SHA)"; "Using a branch name (or HEAD) … is not supported"; `autoupdate` to the latest tag, `--freeze` stores hashes | [pre-commit](https://pre-commit.com/) |
| Renovate | the pre-commit manager "is disabled by default and must be opted into"; the mise manager reads `mise.toml`, its core and aqua tools | [pre-commit](https://docs.renovatebot.com/modules/manager/pre-commit/), [mise](https://docs.renovatebot.com/modules/manager/mise/) |
| `trunk upgrade` | linters to "the latest validated version that have passed tests in our plugins repo" | [upgrades](https://docs.trunk.io/code-quality/overview/linters/upgrades) |

## Licences that limit a suggestion

- **CodeQL CLI** ([terms](https://github.com/github/codeql-cli-binaries/blob/c9e1ebc8d88ba207ca596a120fbae1cecd3c47f5/LICENSE.md)):
  analysis "only with an Open Source Codebase" (an OSI licence), its CI
  only if "hosted and maintained on GitHub.com"; not "any codebase that
  is not an Open Source Codebase (e.g., code in a private repo in
  GitHub)", unless a paid GitHub Advanced Security licence. Inferred:
  not in GitLab CI, even for open source.
- **Semgrep**: the engine is LGPL-2.1; the registry's rules are "only
  for your own internal business purposes", and the licence "does not
  allow you to distribute the rules, or to make them available to others
  as a service" ([rules licence](https://semgrep.dev/legal/rules-license)).
  A project's own rules are its own.
- **Qlty** (BSL, above) and **MegaLinter** (AGPL-3.0) are read for
  ideas; workline is MIT, trunk-io/plugins and Linguist are too.

## AI in place of a linter

| Product or study | Found | Link |
|---|---|---|
| GitHub Code Quality | corrects the panel: an LLM scan of recently merged files, "across all languages, without being limited to predefined rules", 5 findings a file in 5 files at most; on pull requests "rules-based CodeQL findings only"; its "AI findings" "cannot be set as a threshold" | [recent merges](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/fix-findings-in-recent-merges), [overview](https://docs.github.com/en/code-security/concepts/code-quality/code-quality), [thresholds](https://docs.github.com/en/code-security/how-tos/maintain-quality-code/set-pr-thresholds) |
| Copilot code review | runs ESLint, PMD and CodeQL beside the model, set by the ruleset "Manage static analysis tools in Copilot code review"; preview, 2025-11-20 | [changelog](https://github.blog/changelog/2025-11-20-linter-integration-with-copilot-code-review-now-in-public-preview/) |
| MegaLinter LLM Advisor | fix suggestions for linter errors; by default "only linters that are blocking", `LLM_ADVISOR_LEVEL: WARNING` the others too (corrects "only blocking") | [LLM advisor](https://github.com/oxsecurity/megalinter/blob/66064a317ff41c7e9bd451967a08aa4f5b134b7c/docs/llm-advisor.md) |
| GPTLint (MIT) | "augments `eslint` instead of trying to replace it"; prints each run's cost; its docs site did not resolve | [README](https://github.com/gptlint/gptlint) |
| CASTLE, 2025-03 | 250 programs, 25 CWEs; 13 static analysers, 10 LLMs, 2 formal tools; LLMs strong on small snippets, "their accuracy declines, and hallucinations increase as the code size grows"; analysers raise many false positives | [arXiv:2503.09433](https://arxiv.org/abs/2503.09433) |

Rules instead of a model each push: a Semgrep rule is tested with
`ruleid:` and `ok:` comments and `semgrep --test`
([testing](https://docs.semgrep.dev/writing-rules/testing-rules));
an ast-grep rule (MIT) with `valid` and `invalid` snippets and
`ast-grep test` ([testing](https://ast-grep.github.io/guide/test-rule.html)).

## Embedded C and C++

- **PlatformIO**: `pio check` uses `check_tool`, "Default: `cppcheck`",
  clang-tidy as another; "make sure your project builds without errors";
  `--fail-on-defect`, `--json-output`
  ([analysis](https://docs.platformio.org/en/latest/advanced/static-code-analysis/index.html),
  [check_tool](https://docs.platformio.org/en/latest/projectconf/sections/env/options/check/check_tool.html),
  [pio check](https://docs.platformio.org/en/latest/core/userguide/cmd_check.html)).
- **ESP-IDF**: `idf.py clang-check` regenerates the compilation database
  and runs clang-tidy, clang toolchain only; "still under development"
  ([clang-tidy](https://docs.espressif.com/projects/esp-idf/en/latest/esp32/api-guides/tools/idf-clang-tidy.html)).
- **CodeQL with no build**: C/C++ "may produce less accurate results",
  with custom macros or many external headers
  ([build options](https://docs.github.com/en/code-security/reference/code-scanning/codeql/build-options-for-compiled-languages)).
- **MISRA**: cppcheck's `misra.py` "does not provide rule texts, those
  should be downloaded from MISRA"; the open-source checkers "only cover
  MISRA rules partially" ([manual](https://github.com/danmar/cppcheck/blob/99ad2a83ab14c29d753272388b916887d936af03/man/manual.md)).

## What the panel's claims became

- Confirmed: trunk's table and `suggest_if`, Qlty's licence and
  `package.json`, MegaLinter's descriptors, CodeQL default setup,
  SonarQube's 20%, mise's verification, pre-commit's `rev`, the bumpers,
  Copilot's linters, GPTLint, CASTLE, `pio check`, `idf.py clang-check`,
  MISRA's texts.
- Corrected: trunk has 116 linters, not about 117; MegaLinter's advisor
  can read non-blocking linters too; an LLM linting where no rule exists
  does exist (GitHub Code Quality), after merge and never as a threshold.
- Added: CodeQL's terms (open source only, else paid), Semgrep's rules
  licence, Linguist as the language table.
