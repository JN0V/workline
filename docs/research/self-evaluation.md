# Self-evaluation — focused research (2026-10-03)

**Verdict.** Automation is judged by **what people do after it acts**, not
by how much it does: a finding nobody acts on, a bot's pull request closed
unmerged, an issue it closed reopened, a fix reverted. The tools that
observe agents in production (Langfuse, Phoenix, Opik, LangSmith) score the
model's output and need a server; only the code-review bots (Copilot,
CodeRabbit) follow a proposal to what became of it, inside their own
service. The systems that improve their own agents (Warp's factory,
auto-harness, autocontext) agree on one shape: an outer agent reads the
failures and the people's corrections, **proposes** a change with its
evidence as a pull request, and a person merges; the evaluator is kept out
of its reach. Nobody evaluates the pipeline around the agent — whether a
check still fires, a step still earns its cost — from files, with no
server. That is what a self-evaluation role would add.

Stars from the GitHub API that day. "Unverified" marks what was only read in
a summary.

## Observing agents in production

| Tool | What it records and scores | To borrow |
|---|---|---|
| [Langfuse](https://langfuse.com/docs/evaluation/evaluation-methods/llm-as-a-judge) (35.3k★) | Online LLM judges on live traces, by filters and a sampling share; a score is typed, attached to one run, with its reason. Needs Postgres, ClickHouse, Redis, S3 | A score on one run, with a reason |
| [Arize Phoenix](https://github.com/Arize-ai/phoenix) (11.7k★) | Tracing on SQLite; evaluations as batch jobs over stored runs; drift only in the paid product (unverified) | A scheduled batch over stored runs |
| [LangSmith](https://docs.langchain.com/langsmith/online-evaluations) | Online evaluators filtered ("runs a user disliked"), sampled, with a weekly spend cap on the judge | A budget on the evaluator; judge first where people disagreed |
| [Opik](https://www.comet.com/docs/opik/production/online-evaluation/rules) (22.4k★) | Production runs scored apart from test runs; evaluation to discover failures, guardrails to block them | Evaluate, then turn what it finds into a hard check |
| [OpenTelemetry GenAI](https://github.com/open-telemetry/semantic-conventions-genai) (403★, in development) | `gen_ai.request.model`, `gen_ai.usage.input_tokens`, `invoke_agent` spans, `gen_ai.evaluation.result` | Field names for a run's record, even without OTel |
| [Copilot metrics](https://docs.github.com/en/copilot/reference/copilot-usage-metrics/copilot-usage-metrics), [CodeRabbit](https://docs.coderabbit.ai/guides/dashboard-metrics) | Suggestions shown and applied; a comment is accepted only when the next review sees it fixed, not when its thread is resolved | Acceptance read from what happened, per kind of proposal |

Judges are biased toward their own outputs, the more so the better they
recognise them ([Panickssery et al., NeurIPS 2024](https://arxiv.org/abs/2404.13076));
MT-Bench found position and verbosity bias ([2306.05685](https://arxiv.org/abs/2306.05685)).

## Measuring automation itself

| Practice | Signal | To borrow |
|---|---|---|
| Google Tricorder ([paper](https://research.google/pubs/tricorder-building-a-program-analysis-ecosystem/), unverified in full) | **Effective false positive**: a finding after which the developer takes no positive action; a check above about 10% is fixed or removed | Usefulness from the next human act; a budget for being ignored |
| Dependabot ([Mohayeji et al., MSR 2023](https://aserebre.win.tue.nl/MSR2023.pdf); [He et al.](https://arxiv.org/abs/2206.07230)) | Merge ratio, time to merge; a pull request closed then redone by hand months later; projects muting or removing the bot (11.3%) | Closed and redone by hand: right but unusable; closed and never redone: noise; a role muted is a signal |
| Stale bot ([Khatoonabadi et al.](https://arxiv.org/abs/2305.18150), unverified) | The backlog shrinks, active contributors fall, novices hit hardest | Who a role acts on, and its side effects, not its throughput |
| Bots on pull requests ([Wessel et al.](https://arxiv.org/abs/2103.13950), unverified) | Noise is the first complaint: verbosity, unasked actions | Comments per useful act are a cost |
| Flaky checks ([Trunk](https://docs.trunk.io/flaky-tests/overview.md), [Mergify](https://docs.mergify.com/ci-insights/)) | Same commit, different verdict; *broken* (fails always) apart from *flaky* | A role run twice on one commit gives one verdict |
| Google SRE ([monitoring](https://sre.google/sre-book/monitoring-distributed-systems/), [toil](https://sre.google/sre-book/eliminating-toil/)) | Every alert actionable; a rule rarely exercised — "less than once a quarter" — up for removal; precision, recall | A check that never fires is dead, not passing |
| [ESLint unused disables](https://github.com/mysticatea/eslint-plugin-eslint-comments/blob/master/docs/rules/no-unused-disable.md), [SonarQube resolutions](https://docs.sonarsource.com/sonarqube-server/9.9/user-guide/issues) | Suppressions that suppress nothing; a rule's share of *won't fix* | An override that no longer overrides; a rule people keep lowering |
| [DORA](https://dora.dev/guides/dora-metrics/history/) | Change fail rate, rework rate (2024) | A role's changes reverted or amended by a person |
| GitLab Duo root cause, [Copilot fix for failing Actions](https://github.blog/changelog/2026-05-18-one-click-fixes-for-failing-actions-with-copilot-cloud-agent/) | One failure explained, a fix proposed or pushed for review | — : one failure at a time, no pattern across failures |

Pitfalls: a merge rate alone (trivial fixes inflate it); a shrinking backlog
(stale bot); silence read as health; retries that hide breakage; volume of
acts read as value; percentages on three events — one person's repository
gives few, so counts over a window.

## Systems that improve their own agents

| Project | Mechanism | Human control |
|---|---|---|
| [Warp's factory](https://www.warp.dev/blog/engineering-self-improving-software-factories) (oz-for-oss, 312★) | Every few hours, an agent reads the scorers' unprocessed failures and the people's corrections of review comments, proposes changes to skills and config | A pull request with the failed runs it answers and the effect expected; a person merges |
| [neosigmaai/auto-harness](https://github.com/neosigmaai/auto-harness) (545★) | Benchmark, analyse, edit, gate (suites, validation), append to `learnings.md` | A person's `PROGRAM.md`; the gate |
| [greyhaven-ai/autocontext](https://github.com/greyhaven-ai/autocontext) (1.3k★) | Outcome-gated promotion, fail-closed on quality and cost, protected holdout | "Nothing is enabled automatically"; people's labels only |
| [canvas-org/meta-agent](https://github.com/canvas-org/meta-agent) (78★) | A change accepted only on a held-out split | The holdout |
| [Darwin Gödel Machine](https://sakana.ai/dgm/) | Agents rewrite their own code, validated on benchmarks | Sandbox, an archive to roll back: the agent **removed the markers its reward used to detect hallucination**, caught by the history |
| [SICA](https://arxiv.org/abs/2504.15228) | The best agent becomes the meta-agent; utility counts score, cost and time | An overseer agent cancels runs that drift |
| [Harness engineering](https://martinfowler.com/articles/harness-engineering.html) (Böckeler) | Guides and sensors, computational and inferential; "whenever an issue happens multiple times, the controls should be improved" | People keep accountability; agents draft rules and linters |
| DSPy [GEPA](https://dspy.ai/api/optimizers/GEPA/overview/), OPRO, TextGrad, Promptbreeder | Prompts mutated against a metric and a dataset, feedback as text | The person's metric; all overfit a small set |

Safety: models edit tests and graders when rewarded through them
([METR](https://metr.org/blog/2025-06-05-recent-reward-hacking)); from
sycophancy to editing their own reward and hiding it
([Anthropic](https://www.anthropic.com/research/reward-tampering)); a proxy
optimised far enough makes the goal worse (Goodhart).

## Patterns worth borrowing

1. **Outcome per proposal kind**, read afterwards from git and the forge:
   applied, undone (reopened, reverted, closed unmerged), redone by hand,
   left alone — never asked of the agent.
2. **Dead and noisy mechanisms**: a check that has not fired in a quarter,
   one that fires every time, one people keep lowering or bypassing.
3. **Same input, one verdict**: a role rerun on a commit it judged.
4. **Cost per useful act**, tokens first, from the engine's records.
5. **Measured without AI, judged with it**: counts first; the agent only on
   what the counts flag, at a judge's independence (ADR-0005), within a
   budget.
6. **A change proposed with its evidence** — the runs, the counts, the
   change, the effect expected and how it will be measured — for a person
   to merge; a recurring defect proposed as a check, not as prompt text.
7. **The evaluator out of reach**: the role never edits a role, the engine,
   a conformance case or an evaluation case; its own suggestions measured
   by their follow-through.

## Gaps no tool covers

1. Run records kept **without a server**, when CI throws the run folders
   away.
2. **What became of a proposal** across comments, issues, patches,
   closings, on GitHub, GitLab and in a clone.
3. The **pipeline around the agent** evaluated: pre steps, checks, caps,
   routing — not only the model's answer.
4. **Improvements proposed as reviewable changes**, by a role that cannot
   apply them, with one provider's models judging another's work only when
   another is there.
