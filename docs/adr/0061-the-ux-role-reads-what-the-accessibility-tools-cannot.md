# ADR-0061: The UX role reads what the accessibility tools cannot; the tools decide

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-11-02
- **Builds on:** [ADR-0048], [ADR-0049], [ADR-0051]; principles 1, 4
- **Research:** [roles-panorama.md](../research/roles-panorama.md#ux-and-accessibility)
- **Issue:** [#208](https://github.com/JN0V/workline/issues/208)

## Context

- **Many projects have a user interface** (web, mobile, desktop), and
  nothing on the line looks at it.
- **Tools check accessibility well**: axe-core, pa11y, Lighthouse CI
  (`lhci assert` exits non-zero on a failed assertion). No gate recipe
  uses them.
- **Tools cannot judge** a flow against the need, consistency between
  screens, or wording.

## Decision

### 1. A project declares its interface; otherwise the role is skipped

- **`.workline/config.yaml` declares the user interface** under a
  `ui:` key: the kind of interface, and how its screens are reached, a
  preview URL or a build command that serves them.
- **No `ui:` key, no UX role**: it does not run, and says nothing.
  A project that declares one runs it.

### 2. Accessibility tools are gate checks

- **Three recipes**, one per tool: axe-core, pa11y, Lighthouse CI
  (an `lhci assert` on the accessibility score, with a threshold the
  project sets).
- **Each is proved by a planted page**: an image with no text
  alternative fails axe-core and pa11y; a page whose accessibility
  score is under the threshold is blocked by Lighthouse CI; a clean
  page passes all three.
- **A tool that did not run is an error, never a pass**: a missing
  binary, an unreachable URL, no report, a crash. The gate fails
  closed.

### 3. The agent reads the screens; its findings never override the tools

- **With an agent**, the UX role reads a change's screens and flows
  against the issue's Need: what a user would miss, where screens
  disagree (the same action named two ways), unclear words (an
  unclear message).
- **Each finding quotes** the screen's text or markup and the issue's
  Need ([ADR-0048]: the agent reads what the checks flag, and says
  where).
- **Findings are advisory.** The gate's verdict is the tools'. Tools
  passing, findings leave it unchanged; tools failing, it stays failed
  whatever the agent finds. The agent only adds findings.

## Build plan

Outside this pull request's scope (docs only), in issues:

1. The `ui:` declaration in the config and its reader, with the
   skip-when-absent case.
2. The three gate recipes and their planted pages.
3. The role (`roles/ux`): its prompt, its findings with quotes.
4. The conformance cases of the issue's Verification.

## Consequences

- **A project with no interface pays nothing.**
- **Accessibility is judged by tools that replay**, as principle 1
  asks; the agent adds what only a reader sees.
- **A wrong agent finding costs a read, never a blocked merge.**

## Not chosen

- **Letting the agent's findings fail the gate**: a reading is not
  deterministic; the tools' verdict must stay reproducible.
- **Guessing the interface from files** (a `package.json`, an
  `index.html`): wrong in both directions; a project says it.
- **One tool only**: they catch different violations; each gets a
  recipe, the project picks.

[ADR-0048]: 0048-the-agent-reads-the-pages-the-reader-checks-flag.md
[ADR-0049]: 0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
