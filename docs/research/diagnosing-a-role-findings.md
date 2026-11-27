# Diagnosing a role — findings per role, and the pitfalls (2026-11-26/27)

The benches of 2026-11-26/27 have ended. Each role's measures, cuts and
verdict are on its own page; this page is the overview, the pitfalls
met, and every strategy's status. The method is on
[diagnosing-a-role.md](diagnosing-a-role.md).

## Findings per role

| Role | 11-25 | First cause found | What shipped | Page |
|---|---|---|---|---|
| Reviewer | 5 | had the code, did not say it; out-of-diff points never read | 2 fixes; an Opus verifier in shadow | [diagnosing-the-reviewer.md](diagnosing-the-reviewer.md) |
| Developer | 6 | craft (8 of 21 send-backs); the engine's `Closes` over a gap (6) | Opus and a checklist; `Part of #n` | [diagnosing-the-developer.md](diagnosing-the-developer.md) |
| Tester | 6 | already two thirds fixed by v0.91.0; refusals never re-asked | refusals mended in the run | [diagnosing-the-tester.md](diagnosing-the-tester.md) |
| Product owner | 5 | never told the line's bounds; the code picker cut the file | 25 lines of instruction | [diagnosing-the-product-owner.md](diagnosing-the-product-owner.md) |
| Documentalist | 4 | parts cannot show an absence; the citation rule held fixes back | reading the repository and citing | [diagnosing-the-documentalist.md](diagnosing-the-documentalist.md) |

- **The model was a lever for two roles only**: the reviewer (Opus
  states 2 to 3.7 more points a run in every structure) and the developer
  (Opus with a checklist). Opus gained nothing on the product owner's
  causes, lost points as the tester's writer, and missed the
  documentalist's absent claims too.
- **The engine was a cause in every role**: a sentence it writes
  (developer's `Closes`, tester's "nothing guards it"), an input it cuts
  (the product owner's and documentalist's parts), a filter (the
  reviewer's same-model judge), a timeout.
- **Every shipped prompt was measured on the cases it was written
  from**: each gain is an upper bound until held-out or shadow evidence.

### Developer

Kept as an anchor for older links. Bench: 11 send-backs replayed, 19
reachable reasons; Opus with the checklist 16 · 16 · 15, Sonnet with it
7 · 12 · 7, production 4 (one run), at 2.5 times production's fresh
tokens. Details: [diagnosing-the-developer.md](diagnosing-the-developer.md).

## Pitfalls met

| Pitfall | Measure |
|---|---|
| An oriented 2-call replay taken for a root cause | The first diagnosis said read tools found every point on 3 pull requests: 2 calls with tools ran, both on one pull request, one with a leading prompt. On the full set, repository reading recovers only out-of-diff points: 6 of 34 |
| Code removal pursued by principle | Agents reading the repository, the input pickers deleted: equal within noise at 1.6 to 2.8 times the tokens (reviewer [#2273](https://github.com/JN0V/workline/pull/2273), documentalist [#2276](https://github.com/JN0V/workline/pull/2276)), and kept catches lost by the reviewer's lenses. Removing dead code gave 122 lines ([#2272](https://github.com/JN0V/workline/pull/2272)), not the 4.7k first counted |
| Place-based scoring rewards volume | 61 of 310 reviewer points were by place only, 79 true statements scored nothing; by meaning, free agent and lenses are level at the same model |
| A bench forcing a non-production judge | Every reviewer bench forced a Sonnet judge; production judges Sonnet lenses with Opus. The production baseline was added as its own arm |
| Arms on different commits | Opus lenses alone gave 19, 19 at one commit and 22, 21, 22 at another |
| Criteria fixed after the first arms | The criteria were written after the arms of [#2273](https://github.com/JN0V/workline/pull/2273) and [#2279](https://github.com/JN0V/workline/pull/2279) had run |
| Bench slots starved | Two slots, `/tmp` in RAM, no order: a batch waited 40 to 70 minutes; 6 runs × 6 arms of a role did not fit |
| A judge fed the wrong input | The product owner's first rep, 53 runs, gave the judge a diff, not the issues; the developer's judge first read a pull request's text as it reads today, and an empty diff |
| An official check that marks true edits false | The documentalist's `judge` check, given the case's facts and no repository |

Also met, smaller:

- `clean-tg-11` has an untested scenario: most of its alarms are true.
- A developer check (`dc-240`) failed runs that kept a real measure,
  which the issue allowed: counted for no arm.
- The summary counted a case twice when some runs had no judge
  ([#2280](https://github.com/JN0V/workline/pull/2280)).

## Strategies, and their status

| Strategy | Role | Status |
|---|---|---|
| Agents read the repository, pickers deleted ([ADR-0074](../adr/0074-agents-read-the-repository-the-engine-stops-preparing-their-input.md) step 2) | reviewer | closed ([#2273](https://github.com/JN0V/workline/pull/2273)): equal, +55 % tokens |
| Same | documentalist, product owner | held ([#2276](https://github.com/JN0V/workline/pull/2276), [#2277](https://github.com/JN0V/workline/pull/2277)) |
| ADR-0074's bar amended to not worse | all | open ([#2274](https://github.com/JN0V/workline/pull/2274)) |
| Lenses then an Opus verifier (`hso`) | reviewer | in shadow ([#2312](https://github.com/JN0V/workline/pull/2312)): 24.2 of 34 against 17.3 |
| One free Opus agent with merge standards | reviewer | not shipped: level with `hso`, drops weak tests |
| Lenses on Opus | reviewer | stopped: below the leaders, more clean alarms |
| A kept behaviour left as it was owes no test | reviewer | merged ([#2288](https://github.com/JN0V/workline/pull/2288)) |
| Outside findings judged at the project's floor | reviewer | merged ([#2287](https://github.com/JN0V/workline/pull/2287)) |
| Sonnet verifier; lens prompt; effort high | reviewer | dropped untested |
| Opus medium and a checklist | developer | merged with a guard ([#2304](https://github.com/JN0V/workline/pull/2304)) |
| `Part of #n`, never `Closes`, over a part left | developer | merged ([#2291](https://github.com/JN0V/workline/pull/2291)) |
| Only an explicit request wakes a round | developer | merged in v0.91.0 ([#2220](https://github.com/JN0V/workline/pull/2220)) |
| Second pass; effort high | developer | never run |
| Refusals asked again with their reason (`mend`) | tester | merged ([#2306](https://github.com/JN0V/workline/pull/2306)) |
| An Opus second reader (`vm`) | tester | provisional winner, one triple; not built |
| Shell; prompt; Opus as writer; three retries | tester | stopped after one run |
| The line's bounds, quotes, own questions (S4) | product owner | merged ([#2297](https://github.com/JN0V/workline/pull/2297)), text v4 |
| Opus; effort high; reading alone | product owner | stopped: no gain on the causes |
| A doc too long for a task read in place | documentalist | merged ([#2289](https://github.com/JN0V/workline/pull/2289)) |
| Reading and citing an absence (arm C) | documentalist | merged as [#2313](https://github.com/JN0V/workline/pull/2313); search of the project next ([#2317](https://github.com/JN0V/workline/issues/2317)) |
| Citation rule off for removals; effort high | documentalist | never run / stopped, not shown worse |
| Score by meaning | bench | merged ([#2282](https://github.com/JN0V/workline/pull/2282)) |
| Replay sets on GitHub's runners | bench | merged ([#2293](https://github.com/JN0V/workline/pull/2293)) |
| Baseline rows of the replay set | bench | open ([#2271](https://github.com/JN0V/workline/pull/2271)) |
| Narrow what the bench's commit covers | bench | not built |
