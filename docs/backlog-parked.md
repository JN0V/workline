<!-- workline
sources: []
-->
# Backlog: parked topics

> Moved from [BACKLOG.md](BACKLOG.md), archived on 2026-10-04 with it:
> no longer updated.

## Roles and method

- **Consolidate the test corpus.** Tests should prove behaviour over time, not
  the one fix just made. Before adding a test, look for one covering the same
  behaviour and extend it; periodically merge redundant tests, using coverage
  overlap and mutation score. A tester or test-curator role.
- **Five whys as a shared skill.** Root cause before any countermeasure in the
  defect ledger; used by reviewer, developer, tester, architect; triggered when
  a defect is recorded or a role blocks repeatedly.
- **Context and memory management.** Long agent sessions fill their context and
  degrade (compaction, context rot). Make My Dreams addressed it; decide whether
  the framework handles it, e.g. what a role writes down for the next run
  instead of keeping it in context.
- **Rebuild Make My Dreams on this framework**, once it works.
- **Style (vale).** Parked 2026-09-28: without AI, the risk for a doc is to
  be wrong or out of reach, which the checks already cover; a style linter
  adds a binary, rules to keep and findings people learn to ignore. Worth it
  when a project already keeps a vale config: run vale when `.vale.ini`
  exists, as lychee is run, and report without blocking.
- **Release notes must not claim what is not built.** The first trial release
  listed the documentalist as working, because its role *definition* was
  committed as `feat(documentalist): …`. A change that only specifies a role is
  `docs`, not `feat`; and the notes — the release tool's changelog since
  ADR-0017 — should be checked against what the code can actually do (the
  "capability lie" Make My Dreams detected).

## Documentalist beyond our repositories

ADR-0014's checks run on six public repositories with no agent
(docs/research/documentalist-genericity.md): fitted to C-like code, English,
`.md`, three-part versions. In order:

1. **Paths not ASCII.** Done: every git call reading paths sets
   `core.quotePath=off`, the documentalist's lists use `-z`
   (`path-not-ascii-judged`).
2. **History and decision records by setting.** Done: changelogs and
   release notes however named or foldered, `architecture-decisions/`,
   `adr001-…`; the `history` and `decisions` settings add globs
   (`count-off-not-in-history-by-convention`,
   `superseded-decision-other-conventions`).
3. **The comment test, wider and loud.** Done: docstrings, `#!` scripts,
   `--`, `{{! }}`, `{# #}` and others; a type it does not know is said
   (`comment-style-unknown`; `docstring-not-evidence-refused`).
4. **The 20,000-character cap a setting.** Done: `whole-chars`, in
   characters as the engine measures a task before the call, 20,000 by
   default; the task grows with it (`whole-chars-setting`). Raising it
   costs tokens each night: measure on the evaluation before raising the
   default.
5. **Doc-like files the globs skip, reported.** Done: `docs-not-read`,
   when gardening, on `init` and by the doctor, with how to include a
   README (`docs-not-read-reported`, `doctor-docs-not-read`). Reading
   `.rst`, `.mdx`, `.adoc` waits for a header form for each.
6. **`value-left`: history markers and version formats.** Done: markers
   skipped as ranges are; `versions.pattern` and `versions.files`
   (`value-left-not-a-marker`, `value-left-version-pattern`,
   `value-left-version-file`).
7. **Words by language.** Done for the removal rule: typography made
   plain, glue and fact words in English and French, by `language` or
   read from the doc (`removal-rule-reads-french`). Count and limit
   words (`count-off`) are still English: "1 200 lignes" unread.
8. **Sources proposed without an agent.** Done: by name, on `init --ai
   none`, never written (`init-without-ai-proposes-by-name`); 261 of
   backstage's 452 docs, about half right, 0 to 5 elsewhere.
