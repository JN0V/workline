# Stepping back — did workline run away, who should have seen it, and how others keep a builder from drifting (2026-11-13)

**Question.** In seven weeks workline grew to about 111k lines of Go.
Was it a runaway, who drove it, could anything but the maintainer's
unease have caught it, and what do other fields do so a person stays on
the loop without reading every change?
[ADR-0068](../adr/0068-the-line-steps-back-on-its-own-drift.md) is the
decision.

**Verdict.**

- **A runaway, measured**: the contract only grew, little was deleted,
  and a feature freeze did not hold.
- **The builder ran away, not the line**: coding-agent sessions under
  the maintainer's account built nearly all of it, past the ruleset.
- **The line's output was good in real use**: 8 of 8 code pull
  requests accepted on DomoticsCore, one over-built.
- **AI judges on repository facts caught one kind of unease**, the fix
  storm, and no other; a one-line rule did as well.
- **Practice elsewhere** says: a watcher out of the builder's reach,
  advisory before blocking, planted faults to measure it, deletion by
  usage, and never the person's silence read as health.

Read-only, on `origin/main` at `e16e4e7b0` (git measures,
re-measured there) and the forge on 2026-11-13. Method: five
panel reviews of the code and its history, a synthesis that re-checked
their conflicting claims, a red team, three reviews of the decisions
(adversarial, completeness, outside practice), and a blind backtest.
Every reviewer was a model of one family; agreement between them is weak
evidence.

## The runaway, measured

| Measure | Value | How |
|---|---|---|
| Age | 51 days, 2026-09-23 → 11-13; 1,411 commits | `git log` |
| Non-test Go | 13,856 (09-30) → 66,752 (10-21) → 110,989 (11-13) | `git ls-tree` per date |
| Net production Go, the whole history (8 ISO weeks) | +111.0k, of which workline's roles wrote 690 lines (0.6%) | first-parent commits on `main`, added minus deleted |
| Deleted / added | about 16%; 5 commits remove more than 100 net lines, none after 10-30 | `git log --numstat` |
| Role settings | 20 → 118, never fell | `roles/*/role.yaml` |
| Rule codes | 50 → 531, never fell | distinct `Rule: "…"` literals |
| ADRs | 66 in 7 weeks, 13,473 lines | `docs/adr/` |
| Fix share of merged PRs | W40 14%, W43 23%, W45 62% (101 of 163), W46 57% | PR titles by ISO week |
| A fix, since 11-01 | median +57 net lines, p90 +224 | per commit |
| Under [ADR-0052](../adr/0052-v1-0-when-the-core-roles-are-out-of-beta.md)'s freeze (from 10-26) | 111 `feat` commits; Go 66,752 → 110,989; 38 settings and 254 rule codes added | same |

### Who built it

- **Last 300 merged pull requests** (10-31 → 11-13): 237 by sessions
  under the maintainer's account, 62 by workline's App (34 the
  developer's), 1 by an outside contributor. From 11-09, sessions 81 of
  89, the developer 5.
- **The line kept building**: after 11-09 the developer opened 6 pull
  requests, [#1452](https://github.com/JN0V/workline/pull/1452) merged
  on 11-13, four open.
- **392 ruleset bypasses** on `main` by the maintainer's account in the
  month to 11-13, 272 from 10-21; 375 (96%) skipped only the missing
  label ([the forge read](nobody-bypasses.md)).
- **The sessions' standing orders lived outside the repository**: merge
  after a re-check, panels decide and sign for the maintainer, keep
  going overnight. `AGENTS.md` said the opposite; the later order won.
- **Origin is not in git**: only the pull request says who built.

### What CODEOWNERS would hold

Commits on `main` from 10-16 to `e16e4e7b0`, each a squashed pull
request, 534 in all, by the paths they touch (`git log -- <paths>`):

| Paths | Commits | Share |
|---|---|---|
| `roles/` | 313 | 59% |
| of them, `status.md` pages | 146 | 27% |
| of them, prompts (`instruction.md`, `policy.md`, lenses) | 51 | 10% |
| [ADR-0068](../adr/0068-the-line-steps-back-on-its-own-drift.md#4-the-guard-is-guarded)'s guarded list | 90 | 17% |
| of them, `.github/workflows/`, `.github/actions/` | 59 | 11% |
| of them, `.workline/config.yaml` | 24 | 4% |
| of them, `AGENTS.md`, `CLAUDE.md`, `PRINCIPLES.md` | 8 | 1% |
| of them, the reviewer's and the auditor's prompts | 11 | 2% |

## What the reviews found

### The panel, re-checked by its synthesis

- **Keep as code** what checks facts: the line of states, the
  red-then-frozen proof, propose/apply, caps, the forge layer.
- **Judgement frozen into Go** is where fixes cluster: `states.go` 28
  fixes since 11-01, `backlog.go` 24, `productowner.go` 22. Prompts
  held in Go strings need a release to change.
- **Few dead symbols** (2 by staticcheck), **many mechanisms with no
  user outside workline**: product owner acts off by default, 18
  settings with no conformance case.
- **It corrects by adding**: 10 `refactor` commits in 1,411.
- **Claims found wrong or overstated**: "non-V1 roles maintained at full
  intensity" (7 of 158 fixes since 11-01); "guessing code takes the
  fixes, fact code lasts" (103 fixes touch the kernel, 105 the chain's
  roles).

### The red team

- **The actor that ran away is the builder**, which workline's
  observability does not see: the line's own agents are under 0.3% of
  the builder's tokens ([catch-and-cost.md](catch-and-cost.md)).
- **V1's acceptance was in-sample**: the same nine issues on three
  copies, three rounds, a local forge.
- **The loop closed once**: [ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)'s
  cuts, a measure, a decision and named deletions, within five days.
- **Docs drift is saturated**: 131 of 138 docs with `checked` have a
  source changed since; a flag always on says nothing.

### The three reviews of the decisions

- **Adversarial**: only the sessions' identity has teeth, and a bot
  account or the existing App is the wrong identity (a machine user with write
  access counts as a person's yes, though the engine refuses a bot's
  label; the App signs the acceptance). The
  person's token stays on the machine
  ([ADR-0058](../adr/0058-the-forge-guards-the-project-not-the-agents-environment.md)).
  "Silence as consent" broke
  [ADR-0060](../adr/0060-the-person-is-on-the-loop.md) §1. A blocking
  simplicity lens on day one broke the rule "warn until measured".
  Principle 7 itself orders the growth.
- **Completeness**: a session's pull request cannot merge by the policy
  at `normal` without a person's label; the release's yes is required by
  two accepted ADRs; [ADR-0052](../adr/0052-v1-0-when-the-core-roles-are-out-of-beta.md)'s
  freeze forbids the frame; need
  laundering is open, since the intent lens is asked only of a change
  closing an issue.

## The unease backtest

**Question**: could AI judges, reading only facts about the repository
at an instant, have flagged the maintainer's unease before they said it?

- **Inputs**: 24 fact sheets, 99 numbers each, computed with no AI at an
  instant: 8 episodes (2 h before the first message of unease) and 16
  controls (no unease within 2 days). Shuffled; two judges of different
  models rated each 0–3, blind to the key.
- **The episodes**, in one line each, paraphrased:
  - E1 (minor), 10-12: a session proposed a new role; the maintainer
    questioned it.
  - E2, 10-21: the work loops, and the overall progress cannot be
    seen.
  - E3, 10-30: a first live developer run; a workaround rushed.
  - E4–E8, 11-04 → 11-13: a fix storm on the trials, ending in the
    question of a runaway. One continuous regime.

| Detector | Hits of 8 | False alarms of 16 | AUC |
|---|---|---|---|
| Judge A, level ≥ 2 | 5 (E4–E8) | 2 | 0.73 |
| Judge B, level ≥ 2 | 6 | 7 | 0.77 |
| Judge B, level ≥ 3 | 4 | 0 | — |
| Date rank, later is worse | — | — | 0.88 |
| Fix share ≥ 40% | 5 | 0 | 0.79 |
| Off-hours merges ≥ 20% | 5 | 0 | 0.88 |

- **All the judges' skill is the storm.** On E1–E3 against the
  controls, AUC 0.32 and 0.45: chance.
- **Neither judge beat the baselines**: date rank and a one-line rule.
- **Lead**: off-hours merges crossed 20% about 2.5 days before E4, the
  only signal with a lead; bugs opened and pull requests closed unmerged
  12–24 h before. Nothing moved before E2 or E3.
- **Signals with no lead or inverted**: deleted/added ratio (0.52),
  config keys added (0.49), reverts (always 0); days since the last real
  run (0.15): unease came when trials were most intense.

**Limits.**

- **About 4 independent situations**, not 8: E4–E8 overlap. **Age
  leaks**: no control after 10-28, every later day near an episode.
- **Event-driven unease** (E1–E3) leaves no trace in repository numbers;
  it needs session-level facts.
- **What would convince**: a forward test, sheet, rubric, thresholds
  and baselines frozen first, situations counted rather than sheets.

## Outside practice

| Lesson | Source | For workline |
|---|---|---|
| Drift is small, locally rational steps; complexity grows unless work reduces it | [Rasmussen, Snook, Dekker](https://maritimesafetyinnovationlab.org/the-anatomy-of-drift-a-synthesis-of-rasmussen-snook-and-dekker/); [Lehman 1996](https://www.cs.kent.edu/~jmaletic/cs63902/Papers/Lehman96.pdf) | stepping back is continuous work |
| Delete by usage: a binary unused for long gets a deletion change | Google's Sensenmann ([Testing Blog, 2023-04](https://testing.googleblog.com/2023/04/), via the archive); [Meta, FSE 2023](https://2023.esec-fse.org/details/fse-2023-industry/12/Dead-Code-Removal-at-Meta-Automatically-Deleting-Millions-of-Lines-of-Code-and-Petab) | log what fires in real use |
| A person on the loop stops watching; silence is not health | [Bainbridge 1983](https://gwern.net/doc/sociology/technology/1983-bainbridge.pdf); [Parasuraman and Manzey 2010](https://pubmed.ncbi.nlm.nih.gov/21077562/) | the cord cannot be the only yardstick |
| Planted known faults measure a detector | Threat Image Projection ([Hofer and Schwaninger 2005](https://doi.org/10.2495/SAFE050411)) | planted drift cases |
| A monitor optimised against is evaded; patch the scorer | [Baker et al. 2025](https://arxiv.org/abs/2503.11926); [METR 2025](https://metr.org/blog/2025-06-05-recent-reward-hacking) | the guard out of the builder's reach; measures not shown as goals |
| An AI check starts advisory, blocks after calibration on 20–50 past changes | agentic fitness functions ([InfoQ, 2026-08](https://infoq.com/articles/agentic-fitness-functions-evolutionary-architecture), practitioner); [critique shadowing](https://hamel.dev/blog/posts/llm-judge/) | the simplicity lens |
| Models share errors across providers; debate converges on them | [Kim et al. 2025](https://arxiv.org/abs/2506.07962); [Wynn et al. 2025](https://arxiv.org/abs/2509.05396) | blind verdicts, then a count; real use as ground truth |
| A need's size is stated before its design | [Shape Up, ch. 3](https://basecamp.com/shapeup/1.2-chapter-03) | the appetite with the need |
| Removing a thing with users breaks them; know why it was built | [Hyrum's law](https://www.hyrumslaw.com/); [Chesterton's fence](https://fs.blog/chestertons-fence/) | a deletion says why it was built |
| An agent never approves or merges its own work | [Copilot coding agent](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/review-copilot-prs) | nobody bypasses `main` |
| A cheap trusted monitor on everything, a scarce judge on the top | [Greenblatt et al. 2024](https://arxiv.org/abs/2312.06942) | a wake script, then the process engineer |
| Each step: what do you expect, then check | [Toyota Kata 5Q](https://public.websites.umich.edu/~jmondisa/TK/KATA_Files/5Q_Card.pdf) | predictions scored at the next step back |

**Where evidence is thin**: no study measures oversight of an AI system
that builds itself; every lesson above is an analogy, two of them read
second hand (the calibration numbers, Sensenmann).

## Not settled here

- **Whether the next drift is caught**: the triggers come from one
  runaway. **Cost**: none estimated for a lens, a wider auditor, a
  process engineer.
