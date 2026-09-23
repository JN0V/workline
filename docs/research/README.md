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
