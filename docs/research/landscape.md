# Landscape — who does this already

**Verdict.** "The agent proposes, a deterministic tool decides" became a common
pattern in 2026, under the names *software factory*, *harness engineering* and
*control plane*. No project combines everything we want: GitHub and GitLab, an
agent with no write token, a mode without AI, committer / release manager /
documentalist roles, and a model grid. Building our engine is justified;
borrowing from these projects is mandatory.

## Read before writing code

| Project | ★ | Why it matters |
|---|---|---|
| [lfreleng-actions/github-issues-triage](https://github.com/lfreleng-actions/github-issues-triage) | 0 | Linux Foundation, in production. Three jobs: *Prepare* (trusted), *Propose* (no token), *Apply* (validates, then writes). Apply checks digests of what Prepare produced; the design covers recovery after a partial apply (`docs/development/DESIGN.md` §13.7). |
| [jniox/otodev-public](https://github.com/jniox/otodev-public) | 0 | A French "usine logicielle" on a Linux box: bash dispatcher, `claude -p`, agent job sheets. Each status file is a single-use token for one transition; every turn ends with a status; an auth failure is `BLOCKED_EXTERNAL`, not a failure. |
| [schezwansoftware/hermit_ai_sdlc](https://github.com/schezwansoftware/hermit_ai_sdlc) | 0 | 14 roles, each seeing only its declared context, including a documenter and release notes. "The workflow server is a ledger that decides nothing." Four stages no prompt can skip. |
| [tolboy/warden-agentic-sdlc](https://github.com/tolboy/warden-agentic-sdlc) | 0 | Zero dependencies. `policy.yaml` maps role → vendor profiles; the reviewer must come from another vendor. Budget counts calls. Zero-token demo. |
| [kpenfound/busybees](https://github.com/kpenfound/busybees) | 4 | Go single binary; PM, dev, reviewer, QA roles driven by GitHub labels; model and fallback per role; `bees eval <role>` grades one role on fixture repos with an in-memory GitHub. |
| [basilisk-labs/agentplane](https://github.com/basilisk-labs/agentplane) | 79 | "The CLI is mechanically authoritative and semantically blind." The agent gets a packet (objective, writable scope, result schema); the CLI owns state, git and PRs. |
| [LF-Decentralized-Trust-labs/gitmesh](https://github.com/LF-Decentralized-Trust-labs/gitmesh) | 143 | The only central shared-role server found (Docs, Release, Triage, Security) with native GitHub and GitLab, and OPA policy forbidding agents to merge or touch CI. |
| [makeaiwork/agent-pipeline](https://github.com/makeaiwork/agent-pipeline) | 0 | The only explicit committer role. A script, not the model, sets review depth from the diff; the committer commits an explicit path list; only the owner pushes. |
| [Chachamaru127/claude-code-harness](https://github.com/Chachamaru127/claude-code-harness) | 3.1k | Japanese. Release stage builds changelog and tag only from verified evidence; sync stage reports plan-vs-code drift. |

## Defect ledgers and guards

- [nicolasmelo1/software-factory](https://github.com/nicolasmelo1/software-factory) (Rust): every rule
  is prose plus a failing check, and every check has a mutation proving it
  fires. Existing violations are frozen with an expiry date. A rule matching
  nothing is rejected.
- [Doucs91/hivelore](https://github.com/Doucs91/hivelore): a guard may block only after it
  fails on the pre-fix commit and passes on the current tree (`red-proven`).
- [azrtydxb/procoder](https://github.com/azrtydxb/procoder) (Go): each escaped bug must land a
  lint rule, rubric line or test before the work counts as done.
- [PrimeFoldTools/andon](https://github.com/PrimeFoldTools/andon): defect → countermeasure →
  mechanical guard; new guards start in warn mode.

## Deterministic orchestration

- [coleam00/Archon](https://github.com/coleam00/Archon) (23k): a step is `bash` or `prompt`, never both.
- [sipyourdrink-ltd/bernstein](https://github.com/sipyourdrink-ltd/bernstein): YAML phases with
  `allowed_roles`, no LLM in scheduling, signed receipts.
- [addyosmani/factory](https://github.com/addyosmani/factory): no orchestrator service; issue labels
  are the queue; a versioned `factory-handoff:v1` comment is the contract between stages.
- [disler/super-simple-software-factory](https://github.com/disler/super-simple-software-factory):
  "deterministic Python owns the graph; agents are bounded nodes".
- [accidental-hedge-fund/agent-pipeline](https://github.com/accidental-hedge-fund/agent-pipeline):
  18-stage label state machine, implementer and reviewer from different AIs, capped fix rounds.
- [lacs-project/sysknife](https://github.com/lacs-project/sysknife): outside dev, but the best
  reference for typed actions: the model picks an action, the executor builds the command.

## Large frameworks, and why not them

| Project | Take | Leave |
|---|---|---|
| BMAD | role personas, document chain | local only, LLM drives the flow |
| spec-kit | pinned bundles per role | one catch-all constitution |
| gstack | document-release, ship, cross-vendor review | Claude-centred |
| [nrslib/takt](https://github.com/nrslib/takt) (1.4k) | faceted prompts (persona, knowledge, instruction, output contract, policy last), model promotion ladder, strict rule routing, exit codes | 566 npm packages, agent edits directly, no mode without AI, one maintainer |
| gh-aw | read-only agent, "safe outputs" applied by a separate job | GitHub only |
| aide-loop | engine separate from AI adapter, context budget per role | Claude adapter only, one maintainer |

## Also seen

Superpowers, wshobson/agents (model tier per agent), MoAI-ADK (numeric gates),
loki-mode (verdict computed from facts, BUSL), Gas Town, Spec Kitty, cc-sdd,
OpenAI Symphony, ruflo (the overengineering to avoid), microsoft/conductor,
Shopify/roast, doordash-oss/agentic-orchestrator (Go), owainlewis/machinist (Go,
named commands only), vercel-labs/eve-software-factory-template, GAAI-framework,
Kaademos/secure-sdlc-agents (release gate in security roles).

**Jev** (TypeSafe AI, 2026-09-15) is not a framework: a closed model returning
only typed decisions. At most an optional backend for gate verdicts.
