# Skills for optional know-how — public skills, their directories, who loads the format (2026-11-14)

**Question.** [#1494] puts optional, shared know-how in skills: five
whys, how to test Kotlin/Android or C++, a language's conventions. What
must always run stays an engine-loaded prompt. Which public skills and
directories exist for workline's needs, under which licence and with
which risks? Which agents load the Agent Skills format themselves, and
how is a skill given to one that does not? What should workline try
first?

**Verdict.**

- **Nothing to adopt as it is; much to read.** For each need a
  maintained public skill exists under MIT or Apache-2.0, except
  embedded C++ on PlatformIO, where nothing maintained was found. They
  are references: workline writes its own words and credits them here.
- **The format won.** The specification's own showcase lists 46
  clients; one installer writes to about 70 agents' folders. Only
  `name` and `description` are portable; every other frontmatter key
  is one agent's.
- **Reading the file is the common denominator.** The standard's
  client guide names two ways to load a skill: the agent reads
  `SKILL.md` with its file tool, or the harness gives the body. Both
  need no per-agent folder. workline already logs what an agent reads.
- **No Go table becomes a skill.** The tables are data that converge:
  they go to packs ([#1319]). A skill holds prose an agent judges
  with. The role prompts hold no language know-how today, so skills
  add a missing layer; they move nothing out of Go.
- **The risk is real and has a cheap answer.** Public marketplaces
  carry malicious skills. A skill vendored as reviewed text in the
  repository, with no script, never fetched at run time, removes most
  of it.
- **First slice**: one skill, `kotlin-android-testing`, named by the
  `kotlin-gradle` pack, offered to the developer on the Kotlin/Android
  trial copy; one run with it and one without, then one inlined for an
  agent with no tools.

Read on `main` at [`d99ef5adb`](https://github.com/JN0V/workline/tree/d99ef5adb).
Stars, licences and last pushes are the GitHub API's on 2026-11-14.
Sizes are each `SKILL.md`'s bytes that day. Earlier cards:
[root-cause.md](root-cause.md) (five whys, how a method ships),
[portability.md](portability.md) (the standards), [code-review.md](code-review.md).

## The format, and who loads it

- **A folder holding `SKILL.md`**: frontmatter `name` (64 characters,
  the folder's name) and `description` (1,024 characters, what and
  when), then free Markdown. Optional `license`, `compatibility`,
  `metadata`, and an experimental `allowed-tools`. Optional folders
  `scripts/`, `references/`, `assets/`
  ([specification](https://agentskills.io/specification)).
- **Three tiers of loading**: name and description at start (50 to 100
  tokens a skill), the body when the agent decides the task matches
  (under 5,000 tokens and 500 lines advised), other files when the
  body names them
  ([client guide](https://agentskills.io/client-implementation/adding-skills-support)).
- **46 clients** in the specification's showcase, among them Claude
  Code, Claude, ChatGPT and Codex, Gemini CLI, Cursor, GitHub Copilot,
  VS Code, OpenCode, OpenHands, Goose, Amp, Junie, Kiro, Roo Code,
  Mistral Vibe, Factory, pi
  ([the list's source](https://github.com/agentskills/agentskills/blob/main/docs/snippets/clients.jsx)).
  [root-cause.md](root-cause.md) said "about forty" on 2026-10-28.
- **Folders still differ**
  ([the installer's table](https://github.com/vercel-labs/skills/blob/main/docs/agents.md)):

| Project folder | Agents |
|---|---|
| `.agents/skills/` | Codex, Cursor, Gemini CLI, Antigravity, GitHub Copilot, OpenCode, Amp, Cline, Zed, Pi, Kilo Code, Droid |
| `.claude/skills/` | Claude Code; its documentation names no `.agents/skills/` ([Claude Code, skills](https://code.claude.com/docs/en/skills)) |
| its own (`.goose/`, `.junie/`, `.kiro/`, `.roo/`, `.windsurf/`…) | Goose, Junie, Kiro CLI, Roo Code, Windsurf, and about forty more |

- **Codex** reads `.agents/skills` from the working folder up to the
  repository's root, then `$HOME/.agents/skills` and
  `/etc/codex/skills`; a skill loads by its description or by `$name`
  ([Codex, skills](https://learn.chatgpt.com/docs/build-skills)).
- **Claude Code** adds its own keys (`paths`, `context: fork`,
  `disable-model-invocation`, `hooks`…), loads the `.claude/skills/`
  of a folder given by `--add-dir`, and cuts a description at 1,536
  characters in its listing (same page).
- **Only "basic skills" work everywhere**: `allowed-tools` is ignored
  by some agents, `context: fork` and hooks are Claude Code's alone
  (the installer's compatibility table, same link).
- **Not listed anywhere**: Aider. A `cmd:` agent may be one of these.

## How an agent without skills gets one

| Way | How | Who does it |
|---|---|---|
| Catalogue, then a file read | name, description and path of each skill in the prompt; the agent reads the path with its own file tool | the [client guide](https://agentskills.io/client-implementation/adding-skills-support)'s "file-read activation": "no special infrastructure needed" |
| Catalogue, then a tool | an `activate_skill` tool returns the body, frontmatter cut, wrapped in a tag; lets the harness count activations | same guide; most clients with a tool cut the frontmatter |
| Catalogue in `AGENTS.md`, body by a shell command | an `<available_skills>` block written into `AGENTS.md`; the agent runs a command that prints a skill | [openskills](https://github.com/numman-ali/openskills) (10.8k stars, last push 2026-01) |
| Catalogue generated | `skills-ref to-prompt` prints the `<available_skills>` block | [skills-ref](https://github.com/agentskills/agentskills/tree/main/skills-ref), "for demonstration purposes only" |
| The body, whole | the harness puts the body in the prompt: required "when the model can't read files" | same guide; workline's `# Skill: <name>` today |
| A folder per agent | one source copied into each agent's folder | [`npx skills`](https://github.com/vercel-labs/skills), [rulesync](https://github.com/dyoshikawa/rulesync) |

- **The guide's warning on a project's skills**: they come from the
  repository, which may be untrusted; load them only for a trusted
  project.
- **No skill found, no catalogue**: an empty block confuses the model
  (same guide).

## Directories and marketplaces

| Kind | Where | Stars, licence, last push | What it gives |
|---|---|---|---|
| The standard | [agentskills/agentskills](https://github.com/agentskills/agentskills) | 27.3k, Apache-2.0, 2026-11-04 | the specification, the client guide, [how to evaluate a skill](https://agentskills.io/skill-creation/evaluating-skills) |
| Installer and index | [vercel-labs/skills](https://github.com/vercel-labs/skills), [skills.sh](https://skills.sh) | 39.3k, MIT, 2026-11-12 | installs from a git repository; a lock file with a hash and the commit; telemetry on unless turned off; needs Node 22.20 |
| A vendor's own | [anthropics/skills](https://github.com/anthropics/skills) (194k, no licence at the root), [anthropics/claude-plugins-official](https://github.com/anthropics/claude-plugins-official) (47.3k, Apache-2.0), [openai/skills](https://github.com/openai/skills) (28.7k, no licence at the root), [google/skills](https://github.com/google/skills) (24.2k, Apache-2.0), [android/skills](https://github.com/android/skills) (10.5k, Apache-2.0), [Kotlin/kotlin-agent-skills](https://github.com/Kotlin/kotlin-agent-skills) (1.1k, Apache-2.0), [microsoft/skills](https://github.com/microsoft/skills) (3.2k, MIT) | all pushed in 2026-10 or 2026-11 | their products' know-how; mostly migrations and SDKs |
| A team's own | [trailofbits/skills](https://github.com/trailofbits/skills) (8.2k, CC-BY-SA-4.0), [getsentry/skills](https://github.com/getsentry/skills) (1.2k, Apache-2.0) | 2026-11 | what the team uses; both ship evaluations |
| One person's method | [obra/superpowers](https://github.com/obra/superpowers) (317k, MIT), [mattpocock/skills](https://github.com/mattpocock/skills) (324k, MIT), [addyosmani/agent-skills](https://github.com/addyosmani/agent-skills) (117k, MIT), [affaan-m/ECC](https://github.com/affaan-m/ECC) (303k, MIT), [wshobson/agents](https://github.com/wshobson/agents) (42.5k, MIT) | 2026-11 | a whole way of working; skills call each other |
| Community collections | [github/awesome-copilot](https://github.com/github/awesome-copilot) (41.2k, MIT, 481 skills), [VoltAgent/awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills) (37.2k, MIT), [ComposioHQ/awesome-claude-skills](https://github.com/ComposioHQ/awesome-claude-skills) (79.5k, none), [sickn33/agentic-awesome-skills](https://github.com/sickn33/agentic-awesome-skills) (49.7k, MIT, "2,955+") | 2026-09 to 2026-11 | volume; no vetting said |
| Aggregators | tessl.io's registry, SkillsMP, Smithery, ClawHub, openskillindex, playbooks.com, skills.cat | not checked | mirrors of GitHub; some add a score or a scan |

- **Stars measure fame, not a skill**: the four largest personal
  collections each pass 100k; none of the stars says a given
  `SKILL.md` was tried.
- **Evaluations are the better signal**: a run with the skill and one
  without, on the same prompt, assertions graded, tokens and time
  compared ([evaluating skills](https://agentskills.io/skill-creation/evaluating-skills)).
  Seen in the tree of
  [samber/cc-skills-golang](https://github.com/samber/cc-skills-golang)
  (its README gives a with/without table),
  [trailofbits/skills](https://github.com/trailofbits/skills),
  [product-on-purpose/pm-skills](https://github.com/product-on-purpose/pm-skills),
  [getsentry/skills](https://github.com/getsentry/skills),
  [Kotlin/kotlin-agent-skills](https://github.com/Kotlin/kotlin-agent-skills).
  Their results were not rerun here.

## Candidates, a need at a time

"Read" says how far this survey went: `head` is the frontmatter and
the headings; `listed` is the name and size alone.

### 1. Five whys, root cause

| Skill | Maintainer, licence | Stars, last push | Size | Fit, and risk |
|---|---|---|---|---|
| [`think-five-whys`](https://github.com/product-on-purpose/thinking-framework-skills/tree/main/skills/think-five-whys) | product-on-purpose, Apache-2.0 | 26, 2026-11-12 | 7.4 kB | closest in spirit: opens on the method's weak evidence (Card 2017), asks at each why whether several causes apply; a tiny repository (head) |
| [`systematic-debugging`](https://github.com/obra/superpowers/tree/main/skills/systematic-debugging) | obra, MIT | 317k, 2026-11-03 | 9.5 kB | a bug's cause before its fix, four phases; about debugging, not a dispute or a repeated block; its description fires on any test failure (head) |
| [`diagnosing-bugs`](https://github.com/mattpocock/skills/tree/main/skills/engineering/diagnosing-bugs) | mattpocock, MIT | 324k, 2026-11-13 | 8.7 kB | builds a loop that goes red first, then a hypothesis; a developer's skill (head) |
| [`root-cause-analysis`](https://github.com/rohitg00/skillkit/tree/main/packages/core/src/methodology/packs/debugging/root-cause-analysis) | rohitg00, Apache-2.0 | 1.6k, 2026-06-02 | 5.4 kB | five whys with git evidence; five months without a push (listed) |
| [`5-whys-skill`](https://github.com/awesome-skills/5-whys-skill) | awesome-skills, MIT | 63, 2025-12-03 | 7.5 kB | the plain method with two examples; stale (listed) |

- **Kept: workline's own** [`skills/five-whys`](../../skills/five-whys/SKILL.md),
  built for [#92]: each why quotes the material, a closed answer the
  engine checks, an act the auditor samples. No public one checks its
  output.
- **Under the new boundary it is an engine-loaded prompt**: the role's
  `pre` decides it is due. It keeps the format; it may also be listed
  as optional know-how for a role that does not trigger it.

### 2. A language's testing and conventions

| Language | Skill | Maintainer, licence | Stars, last push | Size | Fit, and risk |
|---|---|---|---|---|---|
| Kotlin/Android | [`testing-setup`](https://github.com/android/skills/tree/main/testing/testing-setup) | Google, Apache-2.0 | 10.5k, 2026-11-11 | 10.2 kB | the platform owner's; a 14-step set-up of a whole test strategy (libraries, fakes, screenshot tests, coverage), wider than one issue; links to pages outside the skill (head) |
| | [`compose-ui-testing-patterns`](https://github.com/chrisbanes/skills/tree/main/skills/compose-ui-testing-patterns), [`gradle-run`](https://github.com/chrisbanes/skills/tree/main/skills/gradle-run) | chrisbanes, Apache-2.0 | 1.2k, 2026-11-14 | 7.4 kB, 7.0 kB | patterns for one test at a time: semantics first, deterministic UI tests; the right grain (head) |
| | [Kotlin/kotlin-agent-skills](https://github.com/Kotlin/kotlin-agent-skills) | JetBrains, Apache-2.0 | 1.1k, 2026-10-30 | 10 skills | migrations and tooling; none on testing (listed) |
| | [`kotlin-testing`](https://github.com/affaan-m/ECC/tree/main/skills/kotlin-testing) | affaan-m, MIT | 303k, 2026-11-09 | 20.8 kB | long, near the advised ceiling (listed) |
| C++ | [`modern-cpp`](https://github.com/trailofbits/skills/tree/main/plugins/modern-cpp/skills/modern-cpp) | Trail of Bits, CC-BY-SA-4.0 | 8.2k, 2026-11-12 | 8.7 kB | idioms by tiers, hardening flags, when not to use it; share-alike licence (head) |
| | [`cpp-testing`](https://github.com/affaan-m/ECC/tree/main/skills/cpp-testing) | affaan-m, MIT | 303k, 2026-11-09 | 9.2 kB | GoogleTest and CTest, sanitizers, flaky tests; desktop C++ (head) |
| C++ on ESP32, PlatformIO | none maintained | | | | [espressif/skills](https://github.com/espressif/skills) holds no skill (4 stars); no PlatformIO repository; community ones under 20 stars ([one](https://github.com/johnkozaris/jko-claude-plugins), [another](https://github.com/o2scale/electronics-agent-kit), stale since 2026-02) |
| Python | [`python-testing-patterns`](https://github.com/wshobson/agents/tree/main/plugins/python-development/skills/python-testing-patterns), [`python-code-style`](https://github.com/wshobson/agents/tree/main/plugins/python-development/skills/python-code-style) | wshobson, MIT | 42.5k, 2026-11-09 | 7.3 kB, 8.0 kB | pytest patterns; a short one (listed) |
| | [`modern-python`](https://github.com/trailofbits/skills/tree/main/plugins/modern-python/skills/modern-python), [`property-based-testing`](https://github.com/trailofbits/skills/tree/main/plugins/property-based-testing/skills/property-based-testing) | Trail of Bits, CC-BY-SA-4.0 | 8.2k, 2026-11-12 | 10.0 kB, 4.3 kB | project tooling (uv, ruff), not testing; the second is about properties in six languages (head) |
| Go | [`golang-testing`](https://github.com/samber/cc-skills-golang/tree/main/skills/golang-testing), [`golang-code-style`](https://github.com/samber/cc-skills-golang/tree/main/skills/golang-code-style), `golang-naming` | samber, MIT | 3.9k, 2026-11-08 | 17.9 kB, 9.9 kB | versioned, evaluated with and without; leans on one assertion library; its frontmatter asks for tools and installs a binary, one agent's keys (head) |

- **Kotlin/Android has the best material**: the platform owner's and a
  known practitioner's, both Apache-2.0.
- **Embedded C++ is the gap**: what workline's trial learnt there (a
  `test_filter` that lists suites by name, a native environment's
  record of a suite) is written nowhere an agent reads.

### 3. Code review, simplicity

| Skill | Maintainer, licence | Stars, last push | Size | Fit, and risk |
|---|---|---|---|---|
| [`code-simplifier`](https://github.com/getsentry/skills/tree/main/skills/code-simplifier), [`code-review`](https://github.com/getsentry/skills/tree/main/skills/code-review) | Sentry, Apache-2.0 | 1.2k, 2026-11-05 | 4.0 kB, 2.8 kB | short; five principles, the change's scope only (head) |
| [`code-simplification`](https://github.com/addyosmani/agent-skills/tree/main/skills/code-simplification), [`code-review-and-quality`](https://github.com/addyosmani/agent-skills/tree/main/skills/code-review-and-quality) | addyosmani, MIT | 117k, 2026-11-14 | 13.7 kB, 21.3 kB | understand before touching, the excuses to refuse; examples in three ecosystems only (head) |
| [`code-review`](https://github.com/mattpocock/skills/tree/main/skills/engineering/code-review) | mattpocock, MIT | 324k, 2026-11-13 | 6.7 kB | two axes, the repository's standards and the issue's spec, each its own sub-agent: workline's lenses in small (head) |
| [`code-review`, `code-simplifier`, `pr-review-toolkit`](https://github.com/anthropics/claude-plugins-official/tree/main/plugins) | Anthropic, Apache-2.0 | 47.3k, 2026-11-14 | plugins | commands and agents, not `SKILL.md` (listed) |
| [`differential-review`](https://github.com/trailofbits/skills/tree/main/plugins/differential-review/skills/differential-review) | Trail of Bits, CC-BY-SA-4.0 | 8.2k, 2026-11-12 | 7.4 kB | a security review of a diff (listed) |

- **Not a skill for workline**: a review must always run, so it is a
  lens the engine loads ([#1494]'s boundary; the simplicity lens of
  [#1496]). These are references for a lens's words
  ([code-review.md](code-review.md)).

### 4. Writing an issue, a user story

| Skill | Maintainer, licence | Stars, last push | Size | Fit, and risk |
|---|---|---|---|---|
| [`deliver-acceptance-criteria`](https://github.com/product-on-purpose/pm-skills/tree/main/skills/deliver-acceptance-criteria), [`deliver-user-stories`](https://github.com/product-on-purpose/pm-skills/tree/main/skills/deliver-user-stories) | product-on-purpose, Apache-2.0 | 805, 2026-11-14 | 4.3 kB, 4.2 kB | Given/When/Then for a slice: the happy path, the key failures; versioned, with evaluations and an output contract (head) |
| [`user-story`](https://github.com/deanpeters/Product-Manager-Skills/tree/main/skills/user-story), `user-story-splitting` | deanpeters, a licence of its own | 8.0k, 2026-11-13 | 11.5 kB, 12.8 kB | a story and Gherkin criteria; many frontmatter keys outside the standard; the licence to read before any reuse (head) |
| [`to-tickets`, `to-spec`, `triage`](https://github.com/mattpocock/skills/tree/main/skills/engineering) | mattpocock, MIT | 324k, 2026-11-13 | | a spec cut into tickets (listed) |
| [`014-agile-user-story`](https://github.com/jabrena/plinth) | jabrena, Apache-2.0 | 462, 2026-11-13 | | asks questions, writes a story and a `.feature` file (search result) |
| [`github-issues`](https://github.com/github/awesome-copilot/tree/main/skills/github-issues) | community, MIT | 41.2k, 2026-11-13 | 9.2 kB | the forge's issue mechanics (listed) |

- **Not a skill for workline either**: the product owner's writing
  rules always apply and the engine checks them
  ([writing an issue](../../roles/product-owner/docs/writing.md),
  [issue-formats.md](issue-formats.md)). A candidate for later:
  splitting a story, which is know-how.

## Risks

- **Malicious skills, measured**: of 3,984 skills from two
  marketplaces, 13.4% held a critical issue and 76 a confirmed
  malicious payload; 91% of those used prompt injection. The curated
  top 100 of one marketplace held none, yet 9% fetched outside
  content at run time
  ([Snyk, ToxicSkills, 2026-02-05](https://snyk.io/blog/toxicskills-malicious-ai-agent-skills-clawhub/)).
  A later report counts over 3,000 malicious among about 900,000
  ([as reported](https://expertinsights.com/news/1-in-300-ai-agent-skills-is-directly-malicious-eset-research-finds); not read at the source).
- **A skill is instructions with the agent's rights**: the developer's
  agent runs commands in its worktree. A script, an install step or
  "read this URL first" in a skill runs with them. One candidate
  above links to pages outside itself; another installs a binary.
- **A project's skill is a pull request's text**: a change to it would
  steer the run that reviews that change, unless the run reads the
  base's copy.
- **Licences**:
  - MIT and Apache-2.0: reuse with credit.
  - CC-BY-SA-4.0 (Trail of Bits): an adaptation stays CC-BY-SA, so it
    cannot ship under workline's MIT; read, do not adapt.
  - No licence at the root (several vendor and list repositories): all
    rights reserved unless a skill says otherwise.
  - workline copies no prompt verbatim in any case ([#1494]).
- **Staleness**: skills name tool versions; three candidates had no
  push for five months or more. A vendored copy does not update by
  itself: its source and date are written beside it.
- **Size and triggers**: several pass 17 kB, near the advised 5,000
  tokens. A description like "use when implementing any feature"
  loads every run, which is the opposite of optional.
- **Frontmatter beyond `name` and `description`** is one agent's:
  ignored elsewhere, or refused.

## What workline holds today

- **One skill, engine-triggered**: [`skills/five-whys`](../../skills/five-whys/SKILL.md),
  embedded in the binary, declared in a role's `context.skills`, named
  by its `pre` in `in/skills`, given whole after the task
  ([role contract, "Shared skills"](../spec/role-contract.md#shared-skills)).
  Looked up in the role's folders, then the project's
  `.workline/skills/<name>/`, then as shipped.
- **Six packs of data** ([packs](../packs.md)): detection, test paths
  and homes, a suggested test command, code navigation rows.
- **Rule B's baseline**: 61 lines over 16 Go files, each an extension
  or a marker file in a literal
  ([rule-b-baseline.txt](../../internal/packs/testdata/rule-b-baseline.txt)).
  The other tables are regexes and settings names ([#1319]).
- **No language know-how in a prompt**: the developer's and the
  tester's instructions and policies name no language.
- **What an agent reads is already logged**: in a read tree, each file
  read is kept on the call's record, from the agent's tool events
  ([ADR-0063 §8](../adr/0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them)).
- **The `claude` agent gets a closed tool list and no project
  settings**, so its own skill folders are not read today.

## Recommendation, for the panel

### Which knowledge moves first

- **No Go table**: extensions, marker files, error regexes and runner
  settings converge. They are a pack's fields, read by the engine
  ([ADR-0065](../adr/0065-code-only-what-converges-language-knowledge-in-packs.md#language-packs), [#1319]).
- **First, what the trials paid for and no agent is told**: how a
  toolchain behaves when a test is written and run. Today it lives in
  Go comments, a README's rows and closed fixes
  ([round6-root-causes.md](round6-root-causes.md)): a build that
  stops at the tests' compiling writes no report; a runner's filter
  may skip a new suite; a project not installed needs its root on the
  import path.
- **Three homes, one test each**:

| What | Home | Who loads it |
|---|---|---|
| a fact that converges (a path pattern, an error's shape) | a pack's field | the engine |
| judgement that must always run (a lens, a gate, a method due) | an engine-loaded prompt | the engine, by the role's `pre` |
| know-how that helps some tasks | a skill | the agent, from a catalogue |

- **A skill never decides a verdict**, as a pack never does: its miss
  costs a worse try, which the red and green proofs and the reviewer
  catch ([ADR-0065](../adr/0065-code-only-what-converges-language-knowledge-in-packs.md#rule-a-a-guess-may-only-pass)'s rule A holds).

### Where skills live

- **`skills/<name>/SKILL.md` at the root, embedded**: where the one
  skill already is. Not under a role: know-how is shared; a role's own
  copy stays an override.
- **A pack names its skills**: a field `skills: [kotlin-android-testing]`
  in `kotlin-gradle.yaml`. Data, so the engine still names no
  language; a project with that pack gets that catalogue.
- **Only `name` and `description` in the frontmatter**, plus `license`
  and a `metadata` line for the sources. No `scripts/`, no
  `allowed-tools`: the role's duties bound the agent, not the skill.

### How a project adds its own

- **`.workline/skills/<name>/SKILL.md`**: the lookup already reads it,
  and a project's copy already replaces a shipped one of its name.
- **Offered through the project's pack** (`.workline/packs/`), or a
  `skills:` list in its config for know-how tied to no language.
- **On a pull request, the base's copy is read**, as for the files
  that steer a run; the reviewer reads a change to a skill as any
  prompt ([ADR-0068 §4](../adr/0068-the-line-steps-back-on-its-own-drift.md#4-the-guard-is-guarded)).
- **Open**: also read the project's `.agents/skills/`, which other
  agents share? It widens what a repository can inject; not first.

### How each agent gets a skill

| Agent | Optional skill | A skill due |
|---|---|---|
| reads files (`claude` in a worktree or a read tree; a `cmd:` agent that reads) | a catalogue after the task: name, description, path; the agent reads the file | the body, whole, as today |
| no tools (`claude` on a prompt alone; other `cmd:` agents) | the bodies of its packs' skills inlined, within the role's context budget; one left out is said | the body, whole |
| `none` | nothing | nothing |

- **For the panel**: [#1494] says an agent that loads skills uses them
  "natively". For `claude` that means its own folder, its project
  settings read, and one more tool: what workline closed on purpose.
  The catalogue and a file read give the same progressive loading
  with what is already there, for every agent that reads. Suggested:
  start there; a native folder only if a measure shows the agent
  loads better.
- **Not verified**: whether Claude Code, run headless with a closed
  tool list, loads a skill from a folder given by `--add-dir`.

### How the auditor observes it

- **Traced on the call's record**: skills offered, skills loaded (a
  read of a skill's path, from the tool events the engine already
  parses), skills inlined. The read log exists for a read tree; a
  worktree run needs the same.
- **Sampled as part of the act it served**, not a new kind first: the
  auditor's page on a code or tests act says which skills were
  loaded ([ADR-0046](../adr/0046-the-auditor-re-checks-every-roles-acts.md)).
  Its question: does the act follow the skill it loaded?
- **Measured, not assumed**: a skill ships with one evaluation case
  that fails without it (principle 7), run with and without, tokens
  told. A skill offered and never loaded over a period is reported,
  then cut.

### How it fits "prompt first, code what converges"

- **A skill is a prompt**: no code of its own beyond the catalogue
  ([ADR-0068 §2](../adr/0068-the-line-steps-back-on-its-own-drift.md#2-prompt-first-code-what-converges)).
- **A skill answers a stated need**: a defect or a miss seen in a
  trial, linked from its `metadata`. Never "the public collections
  have one" ([ADR-0068 §1](../adr/0068-the-line-steps-back-on-its-own-drift.md#1-drift-defined)).
- **What a skill says that converges moves to a pack**: a path pattern
  repeated in prose becomes a field.

### The smallest first slice

- **One skill**: `kotlin-android-testing`, about 100 lines, in
  workline's words. What a test of a Gradle module needs: which source
  set, a local test before a device test, a fake before a mock, which
  task writes the report and where, what a broken test build leaves.
- **Credited**: Google's [`testing-setup`](https://github.com/android/skills/tree/main/testing/testing-setup)
  and chrisbanes's [`compose-ui-testing-patterns`](https://github.com/chrisbanes/skills/tree/main/skills/compose-ui-testing-patterns),
  both Apache-2.0, and round 6's findings.
- **One language**: the `kotlin-gradle` pack names it.
- **One real run**: the developer, agent `claude`, on a clone of the
  Kotlin/Android trial copy, one issue. Then the same issue with the
  catalogue off.
  - Told: was the skill read, at which turn; tokens, review rounds,
    the green proof, each way.
  - One pair says whether it loads and what it costs, not whether it
    helps: that needs the evaluation case.
- **Then one inlined run**: the same issue with a `cmd:` agent with no
  tools; agent `none` unchanged, by a conformance case.
- **Next, if it loads**: the same skill offered to the tester, then
  `c-platformio-testing`, where nothing public exists.

## Not done here

- No candidate was run; none of their evaluations was rerun.
- Bodies were read to their headings, not line by line: a skill
  adopted as a reference is read whole first.
- Agents' folders come from one installer's table and three vendors'
  pages, not from a try of each agent.
- Aggregators and their scores were not compared.

[#92]: https://github.com/JN0V/workline/issues/92
[#1319]: https://github.com/JN0V/workline/issues/1319
[#1494]: https://github.com/JN0V/workline/issues/1494
[#1496]: https://github.com/JN0V/workline/pull/1496
