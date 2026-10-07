# Research

What already exists, so we build only what is missing. Findings as of 2026-09-23;
stars and dates were checked against the GitHub API that day.

| Card | Question it answers |
|---|---|
| [landscape.md](landscape.md) | Who already does role-based or "software factory" development, and what to take from them |
| [portability.md](portability.md) | How to define a role once and run it on any coding agent |
| [server-side.md](server-side.md) | How to run roles in GitHub Actions, GitLab CI or on a server, safely |
| [docs-and-release.md](docs-and-release.md) | Tools for the committer, release manager and documentalist |
| [documentalist.md](documentalist.md) | Focused: how to keep docs true, short and linked to product docs |
| [gates.md](gates.md) | Tools for review, architecture, security, performance and test integrity |
| [model-selection.md](model-selection.md) | How to pick a model per role without naming one |
| [install-and-adoption.md](install-and-adoption.md) | How other CLIs diagnose a machine, set it up and adopt a repository |
| [push-approval.md](push-approval.md) | How a git hook asks a person with no terminal: an editor's button, an agent's shell |
| [ci-and-forge.md](ci-and-forge.md) | Running the documentalist in CI on GitHub: models, protected main, a bot's commits, forks; a bot writing GitLab issues: tokens, roles, who is of the project |
| [release-manager.md](release-manager.md) | Focused: release PR vs tag vs direct, where versions live, what stays human |
| [product-owner.md](product-owner.md) | Focused: what forges, bots and AI triage tools do for a backlog — stale bots closing on silence included — and what a product owner role would add |
| [self-evaluation.md](self-evaluation.md) | Focused: how automation and agents are judged in production, and how systems that improve their own agents keep people in control |
| [documentalist-genericity.md](documentalist-genericity.md) | Whether ADR-0014's checks hold beyond our three repositories: six public ones, no agent |
| [code-review.md](code-review.md) | Focused: how AI code reviewers find, verify and post findings, and what a reviewer role takes from them; what reviews a spec before it is built |
| [code-navigation.md](code-navigation.md) | Focused: how tools find a function's bounds, its callers and callees with no build, in any language, for what a judge reads |
| [roles-panorama.md](roles-panorama.md) | Which roles a line needs for any project, which are gates, SonarQube read not rerun, and the order to build them |

## Method

The first pass searched by product category, in English, and missed projects
with over a thousand stars. What worked on the second pass:

- **Use the ecosystem's words, not ours.** Nobody says "intentions",
  "documentalist" or "model grid". They say *software factory*, *harness
  engineering*, *control plane*, *proposals*, *gates*, *ledger*, *evidence*.
- **Curated lists about harnesses and factories**, read in full:
  varun1505/awesome-software-factories, ai-boost/awesome-harness-engineering,
  walkinglabs/awesome-harness-engineering, bradAGI/awesome-cli-coding-agents,
  hashgraph-online/awesome-codex-plugins, Engineering4AI/awesome-spec-driven-development.
- **Hacker News "Show HN"** through the Algolia API, filtered by date.
- **GitHub topics** (`software-factory`, `agentic-sdlc`, `harness-engineering`,
  `spec-driven-development`) with star ranges, so small repos are not drowned.
- **Package registries**: crates.io and pkg.go.dev surface single-binary tools.
- **Japanese sources** (Zenn, Qiita): several of the best projects have JA READMEs.
- **Describe the mechanism** in a web search ("agent writes proposals,
  deterministic applier, no write token") rather than a category.
- Several agents in parallel, each with its own strategy and a shared list of
  projects already known, so each one only reports new ones.

Dry: Reddit (blocked), Chinese sites (mostly noise), GitHub code search (slow,
loose matching), searches in our own vocabulary.
