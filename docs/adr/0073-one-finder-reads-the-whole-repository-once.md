# ADR-0073: One finder reads the whole repository, once a merge request is otherwise mergeable

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-11-24
- **Proposed by:** a contributor, on the maintainer's instruction of
  2026-11-24 to make him less needed on the line's pull requests
- **Builds on:** [ADR-0005] (independence), [ADR-0020] (the reviewer
  finds, the engine verifies), [ADR-0045] (the judge asks for the code
  it needs); principles 7, 12 ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends:** [ADR-0051] §1, "AI judgement once per patch": one
  call more a merge request, at the end

## Context

- **The lenses read the change only**: each file it changes, whole or
  by its hunks, what the author says and the issue it closes
  ([the lenses](../../roles/reviewer/docs/run-lenses.md#4-the-lenses)).
  The agent gets no tools and no checkout ([ADR-0045], option B).
- **Only the judge reads past it**, on demand, for one finding already
  raised ([ADR-0045]): nothing outside the change is ever looked for.
- **Measured on 11 pull requests sent back by the maintainer**
  (2026-11-24, two projects): 3 defects lay outside the lines changed and
  no lens could see them: a callee's contract, a test no workflow runs,
  a target platform's memory limit.

## Decision

**Once nothing else holds a merge request, the reviewer asks one finder
that reads the repository at the head, blind to the lenses' findings
and the judges' verdicts. Its finding is judged as a lens's, and holds
the merge as one does.**

- **What it is**: a question that may read, the judge's own mechanism
  ([ADR-0045]): the engine serves its reads from git at the head, within
  `judge-read-rounds` and `judge-read-lines-max`. No new protocol, no
  tool given to an agent.
- **What it is given**: the whole change, base to head, and the
  repository's files by name; its prompt is a lens file,
  `lenses/repository.md`, which a project may replace.
- **Blind**: not one finding, verdict or summary of the run is in its
  question; its agent is picked apart from the lenses' as a judge is.
- **Its answer**: yes, one defect, the line causing it quoted; no; or
  unsure. A yes's quote found again at the head is a finding of the
  lens `repository`, on the change or outside it as a lens's.
- **Judged**: by a judge apart from the finder, the default question:
  does the code quoted fail as it says.
- **Holds the merge** as a lens's finding does: by `ai-findings`
  (`{repository: block}`), and stands until a commit answers it.
- **When**: on a merge request, once every lens of the run answered
  and no other finding blocks; in a review split in two ([ADR-0050]),
  in the verdict's run when it blocks, the run apart when it warns. The reviewer sees its
  own verdict only: the gates in other jobs run beside it, and waiting
  for them would hold its job.
- **Once**: the record keeps the head it read. It is asked again only
  while its own block stands at a newer head: it reads again; its block
  goes if it finds it no more.
- **Said**: the summary says what it came to, or that it waits for
  nothing else to hold the merge, or that it did not run (no agent, the
  tokens spent): then it is asked at the next run.
- **Off**: `repo-finder: false`.

## Cost

- **About one call more a merge request**, plus up to two read rounds
  (`judge-read-rounds`) and one judge for a finding: 2 to 6 calls, once.
  Within the run's `ai-max-tokens`.
- **A run's judge rounds**: the lenses' judges, then the finder, then
  its finding's judge, within the engine's 10 rounds at the default
  `judge-read-rounds: 2`. Past them, a question is said not asked.

## Options set aside

- **Every lens reads the repository**: its cost on every push.
- **The agent's own read tools**: set aside by [ADR-0045], option B.
- **Several findings a run**: one, the most serious, keeps the answer
  the judge's; more when a measure shows one is too few.

## Measure

- **To keep it on**: over the next 20 pull requests the line declares
  mergeable, its verified findings that the maintainer agrees with,
  and its false alarms, from the weekly sample ([ADR-0046]).

[ADR-0005]: 0005-independence-takes-the-best-level-available.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0045]: 0045-the-judge-asks-for-the-code-it-needs.md
[ADR-0046]: 0046-the-auditor-re-checks-every-roles-acts.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
