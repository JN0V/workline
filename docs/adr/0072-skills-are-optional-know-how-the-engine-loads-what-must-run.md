# ADR-0072: skills are optional know-how; the engine loads what must run

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-11-20
- **Proposed by:** the developer, from [#1494] and its survey
  ([skills.md](../research/skills.md)); the maintainer's boundary of
  2026-11-14 taken as given
- **Builds on:** [ADR-0046], [ADR-0063] §8, [ADR-0065], [ADR-0068] §1,
  §2 and §4; principles 6 and 7 ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends, once built:** the
  [role contract's "Shared skills"](../spec/role-contract.md#shared-skills)
  (a pack names skills; an agent that reads gets a catalogue)

## Context

"Why not skills?" is asked by every newcomer. The survey found:

- the Agent Skills format (a `SKILL.md` with `name` and `description`)
  is loaded by about 46 clients; only those two keys are portable;
- maintained public skills exist under MIT or Apache-2.0 for Kotlin/Android
  testing and C++ testing; none for C++ on PlatformIO;
- public skills carry a measured risk (prompt injection, run-time fetch);
- workline already holds one skill, `five-whys`, engine-triggered, and
  logs what a reading agent reads;
- the roles' prompts hold no language know-how, and no Go table is prose.

## Decision

**The engine triggers what must always run. A skill is shared, optional
know-how that the agent chooses to load. A fact that converges is a
pack's field, not a skill.** This ADR decides the design; it changes no
code.

### 1. Three homes, one test each

| What | Home | Loaded by |
|---|---|---|
| a fact that converges (a path pattern, an error's shape) | a pack's field ([ADR-0065]) | the engine |
| judgement that must always run (a lens, a gate, a method due) | an engine-loaded prompt | the engine, by the role's `pre` |
| know-how that helps some tasks | a skill | the agent, from a catalogue |

A reviewer lens or a gate is never left to the agent to load. A skill
never decides a verdict ([ADR-0065]'s rule A): its miss costs a worse
try, which the red and green proofs and the reviewer catch.

### 2. What moves first

- **No Go table becomes a skill.** The language tables (extensions,
  marker files, regexes, runner settings) converge: they move to packs
  ([#1319]), not to skills. The first knowledge to move is therefore
  the language tables, to packs; the first skill is new prose, not
  moved code.
- **The first skill** is `kotlin-android-testing`, about 100 lines, in
  workline's words, answering what the trials paid for: which source
  set, a local test before a device test, which task writes the report,
  what a broken test build leaves. It is credited in
  [skills.md](../research/skills.md) to Google's `testing-setup` and
  chrisbanes's `compose-ui-testing-patterns` (Apache-2.0); no text is
  copied. Then `c-platformio-testing`, where nothing public exists.
- **A skill answers a stated need** (a defect or miss seen in a trial,
  linked from its `metadata`), never "a collection has one"
  ([ADR-0068] §1). What a skill says that converges moves to a pack
  ([ADR-0068] §2).

### 3. Where skills live

- **`skills/<name>/SKILL.md` at the repository root, embedded** in the
  binary, where `five-whys` already is. Know-how is shared, so not
  under a role; a role's own folder stays an override.
- **A pack names its skills** in a field, `skills: [<name>]`. It is
  data: the engine still names no language, and a project with that
  pack gets that catalogue.
- **Frontmatter**: `name` and `description`, plus `license` and a
  `metadata` line for sources. No `scripts/`, no `allowed-tools`: the
  role's duties bound the agent, not the skill. A skill is vendored as
  reviewed text and never fetched at run time.
- **Borrowing**: public skills are read, never copied verbatim.
  A source of a licence that does not allow adaptation (CC-BY-SA-4.0)
  is read only. Each source, licence and date is credited in
  `docs/research/`.

### 4. How a project adds its own

- **`.workline/skills/<name>/SKILL.md`**: the lookup already reads it,
  and a project's copy replaces a shipped one of the same name.
- **Offered** through the project's pack (`.workline/packs/`) or a
  `skills:` list in its config, for know-how tied to no language.
- **On a pull request the base's copy is read**, as for the files that
  steer a run, so a change to a skill does not steer the run that
  reviews it; the reviewer reads such a change as any prompt
  ([ADR-0068] §4).
- **Not first**: also reading the project's `.agents/skills/`; it
  widens what a repository can inject.

### 5. How each agent gets a skill

The agent stays a choice (`none`, `claude`, `cmd:<any command>`).

| Agent | Optional skill | A skill due |
|---|---|---|
| reads files (`claude` in a worktree, a `cmd:` that reads) | a catalogue after the task: name, description, path; the agent reads the file | the body, whole, as today |
| no tools (`claude` on a prompt alone, other `cmd:`) | the bodies of the pack's skills inlined, within the role's context budget; one left out is said | the body, whole |
| `none` | nothing | nothing |

- With agent `none`, the assembled prompt and the run's proposals are
  unchanged: that is the comparison, and a conformance case holds it.
- No skill found, no catalogue: an empty block is not written.
- **Native loading** (the agent's own skill folder) is not the start:
  for `claude` it means project settings read and one more tool, which
  workline closed on purpose. The catalogue and a file read give the
  same progressive loading with what exists. A native folder only if a
  measure shows the agent loads better. Whether headless Claude Code
  with a closed tool list loads a skill from `--add-dir` is not
  verified.

### 6. How a skill's use is traced

- **On the call's record**: skills offered, skills loaded (a read of a
  skill's path, from the tool events the engine already parses,
  [ADR-0063] §8), skills inlined. A worktree run needs the read log a
  read tree has.
- **Sampled with the act it served**: the auditor's page on a code or
  tests act says which skills were loaded; its question is whether the
  act follows the skill it loaded ([ADR-0046]). No new kind of sample
  first.
- **Measured**: a skill ships with one evaluation case that fails
  without it (principle 7), run with and without, tokens told. A skill
  offered and never loaded over a period is reported, then cut.

### 7. Builds that follow, each its own issue

1. The pack's `skills` field and the catalogue for an agent that reads.
2. `kotlin-android-testing`, one run with it and one without, the use
   traced.
3. Inlining for an agent with no tools; a conformance case for `none`.
4. The read log for a worktree run.
5. Language tables to packs ([#1319]), on its own track.

Each shows the Validation of [#1494]: Kotlin with `claude`; a `cmd:`
agent that loads none; `none` unchanged; always-run prompts loaded by
the engine; a project's own skill like a built-in; a borrowed skill
credited in `docs/research/`.

## Consequences

- Skills add a layer the prompts lack; they move nothing out of Go.
- Role know-how in a language becomes readable, editable and swappable
  by a person, without a release.
- A skill is text with the agent's rights: vendoring and review answer
  the risk, and the guide's warning (load a project's skills only for a
  trusted project) holds.

[#1319]: https://github.com/JN0V/workline/issues/1319
[#1494]: https://github.com/JN0V/workline/issues/1494
[ADR-0046]: 0046-the-auditor-re-checks-every-roles-acts.md
[ADR-0063]: 0063-every-open-issue-is-due-waiting-or-done.md
[ADR-0065]: 0065-code-only-what-converges-language-knowledge-in-packs.md
[ADR-0068]: 0068-the-line-steps-back-on-its-own-drift.md
