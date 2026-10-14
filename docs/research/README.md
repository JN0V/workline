# Research

What already exists, so we build only what is missing. Findings as of 2026-09-23;
stars and dates were checked against the GitHub API that day.

## Cards

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
| [ci-and-forge.md](ci-and-forge.md) | The documentalist in CI on GitHub (models, protected main, a bot's commits, forks); a bot writing GitLab issues |
| [release-manager.md](release-manager.md) | Focused: release PR vs tag vs direct, where versions live, what stays human |
| [product-owner.md](product-owner.md) | Focused: what forges, bots and AI triage tools do for a backlog, stale bots included, and what a product owner adds |
| [surfacing-proposals.md](surfacing-proposals.md) | Focused: where backlog bots show proposals, how a person says yes, no or "change this", what forges record |
| [issue-formats.md](issue-formats.md) | Focused: where forges keep an issue format, the names of its sections, a bug with no reproduction, an issue's kind |
| [priority-and-weight.md](priority-and-weight.md) | Focused: where forges keep an issue's priority and effort, the scales in use, who estimates |
| [self-evaluation.md](self-evaluation.md) | Focused: how automation and agents are judged in production, and how self-improving systems keep people in control |
| [documentalist-genericity.md](documentalist-genericity.md) | Whether ADR-0014's checks hold beyond our three repositories: six public ones, no agent |
| [documentalist-fixes-reviewed.md](documentalist-fixes-reviewed.md) | Measured: whether the documentalist's fixes were right, read against the code |
| [code-review.md](code-review.md) | Focused: how AI code reviewers find, verify and post findings; what reviews a spec before it is built |
| [code-navigation.md](code-navigation.md) | Focused: how tools find a function's bounds, its callers and callees with no build, in any language, for what a judge reads |
| [judge-measured.md](judge-measured.md) | Measured: whether the reviewer's judge is right when it drops or shows a finding |
| [auditing-agent-acts.md](auditing-agent-acts.md) | Focused: how a sample of automated decisions is drawn, read again by another reader, disputes settled and an error rate reported |
| [roles-panorama.md](roles-panorama.md) | Which roles a line needs for any project, which are gates, SonarQube read not rerun, and the order to build them |
| [merge-conflicts.md](merge-conflicts.md) | Focused: why open pull requests conflict with `main`, what forges and git offer, what workline adds |
| [merging.md](merging.md) | Focused: what a person does at merge, who merges elsewhere and how autonomy is graded, whether merging is a role |

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
