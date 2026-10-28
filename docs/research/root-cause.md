# Finding a cause before a fix — five whys, its limits, and how agents load a method (2026-10-28)

**Question.** A role meets a problem that comes back: a builder's
objection still open after two answers, a review finding still there,
a role blocked again, a defect recorded. Which root-cause practice fits
a short, traced analysis by an agent, and how do agent frameworks ship
such a method once for every role?

**Verdict.**

- **Five whys fits, bounded**: one chain, short, any reader can check
  it; it needs no diagram and no meeting. Its known flaws are real and
  each has a cheap guard.
- **Its flaws**: one path, one root cause, an arbitrary depth, and the
  last why taken as the best place to act.
- **What workline takes from the alternatives**: a second cause may be
  named (the fishbone's many branches, the "contributing factors" of
  incident reviews); each step quoted from the material (cause-and-effect
  with evidence); stop at the first cause the project can change, not
  at five; no person as a cause (blameless reviews).
- **How it ships**: a skill in the Agent Skills format, `SKILL.md` with
  `name` and `description`, shared by every role. The role's own
  deterministic part decides when it is due, so the body is given whole
  rather than chosen by the agent from its description.
- **What is left for workline**: the trigger (a role's pre), a closed
  form the engine checks (chain, root cause, one action of a closed
  kind), a bound in characters, and the act drawn by the auditor.

## Five whys, where it comes from

- Taiichi Ohno's method in the Toyota Production System: ask why, then
  why of the answer, about five times, until the cause appears
  ([Wikipedia, Five whys](https://en.wikipedia.org/wiki/Five_whys)).
- Taught everywhere for its simplicity: no tool, no training, a chain a
  newcomer reads.

## Its limits

- **One path, one cause**: Card's review in BMJ Quality & Safety finds
  it forces a single line of reasoning to a single root cause, while
  failures in complex work have several
  ([Card, 2017](https://qualitysafety.bmj.com/content/26/8/671)).
- **Depth is arbitrary**: the answer may come at the second why or the
  seventh; five is a habit, not a law (same source,
  [ECRI's summary](https://www.ecri.org/search-results/member-preview/ccrm/pages/iiccrm081117_should/)).
- **The last why is not always the best lever**: the most distant cause
  can be the hardest to change; the right action is the one that stops
  the problem at an acceptable cost.
- **The answers are only as good as what the asker knows**: an
  analysis with no evidence can be a plausible story. For an agent,
  that is the main risk: a fluent chain that nothing in the material
  shows.

## Alternatives

| Practice | What it adds | Why not the default here |
|---|---|---|
| Fishbone (Ishikawa) diagram | many causes, sorted in categories (people, method, tools…) ([Wikipedia](https://en.wikipedia.org/wiki/Ishikawa_diagram)) | a diagram, for a team in a room; heavy for one thread |
| Fault tree, causal tree | causes joined by and/or, each branch proven | a model to draw; worth it for a safety case, not a test dispute |
| Incident reviews ("contributing factors") | several factors, blameless, actions owned ([Google SRE book, postmortem culture](https://sre.google/sre-book/postmortem-culture/)) | the right spirit; their format is a document, written after an outage |
| Cause-and-effect with evidence | each cause shown by a fact | taken: each why quotes the material |

## How agent frameworks ship a shared method

- **Agent Skills**: a folder holding `SKILL.md`, YAML frontmatter with
  `name` (lowercase, hyphens, the folder's name) and `description`
  (what and when, 1,024 characters at most), then instructions; under
  5,000 tokens advised
  ([specification](https://agentskills.io/specification)).
- **Progressive loading**: the agent sees each skill's name and
  description, and loads the body when the task matches
  ([Anthropic, equipping agents with skills](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)).
- **Read by about forty agents**, from different folders
  ([portability.md](portability.md)).
- **Root-cause skills already exist** in public skill collections, as
  prompts with no check of their output
  ([an example](https://www.skills.sh/product-on-purpose/thinking-framework-skills/think-five-whys)).

## What workline does with it

- The skill: `skills/five-whys/SKILL.md`, shipped in the binary, any
  role may declare it; a project replaces it with its own copy.
- A role's deterministic step decides it is due (two answers did not
  settle a thread, a third review round): the agent does not choose.
- Its answer: a note in a closed form — problem, numbered whys each
  quoting the material, root cause, an optional second cause, one
  action of a closed kind (`issue`, `test`, `doc`, `code`, `decision`),
  1,500 characters at most. The engine checks the form and posts it.
- Built in [ADR-0055](../adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md)'s
  step 8; the method in
  [role-adapting.md](../spec/role-adapting.md#defects-become-guards).
