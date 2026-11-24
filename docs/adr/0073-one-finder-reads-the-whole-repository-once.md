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
- **Holds the merge only with `ai-findings: {repository: block}`**
  (or `block` for every lens); otherwise it warns, as a lens does. A
  block stands until a commit answers it.
- **When**: on a merge request, once every lens of the run answered
  and no other finding blocks; asked alone when no lens is asked (nothing
  new to read, no lens of its side). In a review split in two
  ([ADR-0050]), in the verdict's run when it blocks, the run apart when
  it warns. The reviewer sees its own verdict only: the gates in other
  jobs run beside it, and waiting for them would hold its job.
- **Once**: the record keeps the head it read. It is asked again only
  while its own block stands at a newer head: it reads again; its block
  goes if it finds it no more.
- **Unsure, or not run** (no agent, the tokens spent): the try is
  recorded, its block stands, and it is asked once more. A second time,
  it stops; its own block, if any, is put to a judge, which keeps or
  drops it.
- **Said**: the summary says what it came to, that it waits for nothing
  else to hold the merge, and what its material left out past a cap
  (`diff-lines-max`, the first 400 file names): never "read the whole
  repository" when it was cut.
- **Shipped off**: `repo-finder: false` by default; on in workline's own
  `.workline/config.yaml`, warning, to measure it first.

## Cost

- **Measured** (2026-11-24, 4 runs on DomoticsCore code, Opus): 2 to 3
  calls, 35k to 70k tokens in, 0.8k to 4.9k out, 12 to 50 s. About 70k
  in for 3 calls: some 4 times a one-lens run without it (15k to 20k in).
- **Once a merge request**, plus one judge for a finding, and one retry
  when it could not tell. Within the run's `ai-max-tokens`.
- **A run's judge rounds**: the lenses' judges, then the finder, then
  its finding's judge, within the engine's 10 rounds at the default
  `judge-read-rounds: 2`. Past them, a question is said not asked.

## Options set aside

- **Every lens reads the repository**: its cost on every push.
- **The agent's own read tools**: set aside by [ADR-0045], option B.
- **Several findings a run**: one, the most serious, keeps the answer
  the judge's; more when a measure shows one is too few.

## Measure

- **Replayed on the three defects that led to it** (DomoticsCore, the
  code each pull request held when the maintainer commented, a real
  agent, the finder alone): **none caught**. It read the right files on
  two (the event bus's header; the version check and the test script),
  and found nothing. So it ships off.
- **To turn it on by default**: on workline's own line, over 20 pull
  requests the line declares mergeable, its verified findings the
  maintainer agrees with, and its false alarms, from the weekly sample
  ([ADR-0046]).

[ADR-0005]: 0005-independence-takes-the-best-level-available.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0045]: 0045-the-judge-asks-for-the-code-it-needs.md
[ADR-0046]: 0046-the-auditor-re-checks-every-roles-acts.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
