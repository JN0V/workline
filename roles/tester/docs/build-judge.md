---
sources:
  - roles/tester/code/judged.go#redQuestion
  - roles/tester/code/redfirst.go#redFirst.facts
  - roles/tester/code/judged.go#proveJudged
  - roles/tester/code/judged.go#inputKey
  - roles/tester/code/judged.go#judging.answered
  - roles/tester/code/judged.go#judging.settle
  - roles/tester/code/judged.go#judging.left
  - roles/tester/code/judged.go#judging.material
  - roles/tester/code/judged.go#doubt.broken
  - roles/tester/code/judged.go#doubt.unproven
  - roles/tester/code/build.go#unrooted
  - roles/tester/code/tester.go#errorLines
  - roles/tester/code/tester.go#runIn
  - roles/tester/role.yaml#settings.judge-read-rounds
  - roles/tester/role.yaml#settings.judge-read-lines-max
checked: e81df7b34
verified: agent:claude-code
---
# The judge of a broken build

Part of [A broken build, red for the right reason](missing-name.md).
The fast path passes only what it is sure of; any other build broken
on the base is read by a judge
([ADR-0045](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md)).

## The judge

- **Asked**: "Does each test fail only because what the issue asks for
  does not exist yet, and not by a mistake of its own?"
- **What it reads, raw**: the issue's text, its Scope and scenarios; the
  change's test files; the compiler's lines and the output's end,
  clipped and masked, the checkout's folder taken off their paths; and,
  for each name the errors give, whether the base holds it.
- **Asked once on the same facts**: the agent asked again writes the
  same tests, broken the same way, in the same run or a later one: the
  judge's first answer stands, kept beside the project's runs, said on
  the finding. Other tests or other errors: it is asked again.
- **It may read the base first**: a file's lines, a definition, a word's
  places, served from git (`judge-read-rounds`, 2; `judge-read-lines-max`,
  400). Either at 0: it answers from the facts alone.
- **The proof kept while it reads** lost (unreadable): the run fails
  once, said; the judge is not asked again.
- **Picked** at the best independence from the tester's agent
  ([ADR-0005](../../../docs/adr/0005-independence-takes-the-best-level-available.md));
  `WORKLINE_JUDGE` names one. Its calls count in the tester's
  `ai-max-tokens` ([how](../../../docs/spec/role-contract.md#one-run)).

## Its answers

| The judge | The proof |
|---|---|
| yes | red: `red-judged` says why |
| no | `build-broken`, the judge's line quoted |
| unsure, past its reads, failed, or at the tokens' cap | **red unproven**, marked |

What each does to the run: [what follows](build-outcomes.md).

## No report

- **`report-missing`** quotes the error lines the `build-errors` rows
  read, else the lines saying an error, then the output's end: never
  only the build tool's last line (`BUILD FAILED`)
  ([#1192](https://github.com/JN0V/workline/issues/1192)).
