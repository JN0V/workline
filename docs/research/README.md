# Research

What already exists, so we build only what is missing. Findings as of 2026-09-23;
stars and dates were checked against the GitHub API that day.

## Cards

### Panoramas

| Card | Question it answers |
|---|---|
| [landscape.md](landscape.md) | Who already does role-based or "software factory" development, and what to take from them |
| [portability.md](portability.md) | How to define a role once and run it on any coding agent |
| [server-side.md](server-side.md) | How to run roles in GitHub Actions, GitLab CI or on a server, safely |
| [docs-and-release.md](docs-and-release.md) | Tools for the committer, release manager and documentalist |
| [gates.md](gates.md) | Tools for review, architecture, security, performance and test integrity |
| [model-selection.md](model-selection.md) | How to pick a model per role without naming one |
| [install-and-adoption.md](install-and-adoption.md) | How other CLIs diagnose a machine, set it up and adopt a repository |
| [push-approval.md](push-approval.md) | How a git hook asks a person with no terminal: an editor's button, an agent's shell |
| [ci-and-forge.md](ci-and-forge.md) | The documentalist in CI on GitHub (models, protected main, a bot's commits, forks, conversations to resolve); a bot writing GitLab issues |
| [roles-panorama.md](roles-panorama.md) | Which roles a line needs for any project, which are gates, SonarQube read not rerun, and the order to build them |

### Focused

| Card | Question it answers |
|---|---|
| [documentalist.md](documentalist.md) | How to keep docs true, short and linked to product docs |
| [release-manager.md](release-manager.md) | Release PR vs tag vs direct, where versions live, what stays human |
| [product-owner.md](product-owner.md) | What forges, bots and AI triage tools do for a backlog, stale bots included, and what a product owner adds |
| [surfacing-proposals.md](surfacing-proposals.md) | Where backlog bots show proposals, how a person says yes, no or "change this", what forges record |
| [issue-formats.md](issue-formats.md) | Where forges keep an issue format, the names of its sections, a bug with no reproduction, an issue's kind |
| [priority-and-weight.md](priority-and-weight.md) | Where forges keep an issue's priority and effort, the scales in use, who estimates |
| [self-evaluation.md](self-evaluation.md) | How automation and agents are judged in production, and how self-improving systems keep people in control |
| [code-review.md](code-review.md) | How AI code reviewers find, verify and post findings; what reviews a spec before it is built |
| [code-navigation.md](code-navigation.md) | How tools find a function's bounds, its callers and callees with no build, in any language, for what a judge reads |
| [judge-reads-code.md](judge-reads-code.md) | How judges and reviewers get the code beyond a fixed context — the model reading, its caps, its cost |
| [auditing-agent-acts.md](auditing-agent-acts.md) | How a sample of automated decisions is drawn, read again by another reader, disputes settled and an error rate reported |
| [merge-conflicts.md](merge-conflicts.md) | Why open pull requests conflict with `main`, what forges and git offer, what workline adds |
| [merging.md](merging.md) | What a person does at merge, who merges elsewhere and how autonomy is graded, whether merging is a role |
| [inspector.md](inspector.md) | How SARIF, the forges and linters tell a merge request's new findings, where the base's come from, AI triage measured |
| [tests-first-and-corpus.md](tests-first-and-corpus.md) | How tests are written from acceptance criteria before the code, kept red off `main`, selected per change, cut when they protect nothing |
| [inspector-coverage.md](inspector-coverage.md) | Which linter fits each language, how it is suggested and installed pinned, the licences that limit it, AI in its place |

### Measured

| Card | Question it answers |
|---|---|
| [documentalist-genericity.md](documentalist-genericity.md) | Whether ADR-0014's checks hold beyond our three repositories: six public ones, no agent |
| [documentalist-fixes-reviewed.md](documentalist-fixes-reviewed.md) | Whether the documentalist's fixes were right, read against the code |
| [judge-measured.md](judge-measured.md) | Whether the reviewer's judge is right when it drops or shows a finding |
| [documentalist-measured.md](documentalist-measured.md) | Whether every documentalist case earns every point over five runs on one engine, and what a run costs |
| [reviewer-measured.md](reviewer-measured.md) | What each reviewer lens finds and gets wrong over five runs, whether correctness should block, and a cheaper push |
| [claude-session-env.md](claude-session-env.md) | Why a call started from a Claude Code session cost twice the tokens, and which variable did it |
| [moments.md](moments.md) | How long a developer waits at each moment, the budgets others keep, where a check is skipped |
| [catch-and-cost.md](catch-and-cost.md) | What each check of workline's own line catches and costs a product pull request, per period, and the circles that recur |

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
