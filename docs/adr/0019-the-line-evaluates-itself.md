# ADR-0019: The line evaluates itself, and proposes how to improve

- **Status:** proposed — the points under "Open" wait for the person
- **Date:** 2026-10-03
- **Builds on:** ADR-0005 (a judge's independence), ADR-0015 (the weekly
  sample), ADR-0018 (the product owner; one way in for every role's
  issues), principles 1, 4, 6, 7, 12

## Context

On 2026-10-03, every serious defect of the product owner was found by a
person running it by hand and reading what it wrote: a report that lost
seven proposals in eight, issues recorded as read from an answer that did
not read, a prompt over the role's budget, a milestone set on code never
seen (roles/product-owner/docs/tried.md). Conformance could not see them: they
show only in real runs. The person asked for a role that does that work in
the background — checks that every automatic mechanism is relevant and
works, and suggests improvements.

Pieces of it exist, each on one mechanism: the evaluation grades fixed cases
weekly (results.tsv), the weekly sample has an auditor re-read docs the
documentalist vouched for (ADR-0015), and the product owner drops an act
back to "propose" when a person undoes it (ADR-0018). Nothing looks at the
line as a whole, on its real runs.

docs/research/self-evaluation.md found that automation is judged by what
people do after it acts (Google's *effective false positive*, Dependabot's
pull requests closed then redone by hand, rules exercised less than once a
quarter); that the tools observing agents need a server and score the
model's text, not what became of it; and that systems improving their own
agents all propose a change with its evidence for a person to merge, the
evaluator kept out of their reach — the one that could reach it erased the
markers that graded it.

## Decision

### A role, the process engineer

**A role of its own** (`process-engineer`), run on `schedule`, weekly by
default. Named after the factory's job: the person who studies the line
itself — its stations, its waste, its defects — and proposes how to change
it, while the line keeps running. One job (principle 2): evaluate the
line's mechanisms and propose improvements. It **never changes them**: no
patch to a role, the engine, a setting, a conformance or evaluation case,
its own included (`duties.writes` empty; intentions `issue`, `note`).

### Measured by the engine, without AI

What a run did is recorded **by the engine, never by the agent** (METR,
the Darwin Gödel Machine): one line a run — role, event, commit, status, the
findings' rules, the acts proposed, applied, refused, dropped, the model,
tokens in and out, seconds. Field names follow OpenTelemetry's `gen_ai.*`
where one exists. A run in CI, whose folder is thrown away, keeps its line
where the project lives (see "Open").

From those lines, the forge and git, the role's `pre` counts, over a window
(a quarter by default), per role and per rule:

- **What became of each act**: applied and kept; undone by a person — an
  issue reopened, a fix reverted, a pull request of the line closed
  unmerged; redone by hand later (right but unusable); left alone.
- **Effective false positives**: findings after which nobody acted —
  a block bypassed, a rule lowered with `enforce`, a report not read.
- **Dead and noisy mechanisms**: a check that has not fired in the window;
  one that fires on every run; a step that never changes the outcome.
- **One verdict for one input**: a role's verdicts on the same commit and
  configuration, when it ran twice.
- **Cost per useful act**: tokens for each act kept; the share of runs
  ending without an answer that reads, over budget, or blocked outside.
- **The other measures already kept**: the evaluation's scores over time,
  the weekly sample's findings, the product owner's wrong closings.

Counts, not shares, when there are few events (one person's repository
gives few). Without AI, the role stops there: the counts are its report
(principle 5).

### Judged with AI, only where the counts point

The agent is asked only about what the counts flag — a mechanism dead,
noisy, often undone, costly — with the runs behind it. It says why, and
proposes one change: what to change, the evidence (runs, counts), the effect
expected, and how the next windows will show it. A defect that recurs is
proposed as a check — a conformance case, a rule — before any prompt text
(Böckeler; principle 7). It is asked at the best independence available
from the models that ran (ADR-0005), within a token budget per run.

### Written for a person, never applied

**One report issue**, kept in place (as the product owner's): the counts,
what changed since the last window, each flag. **Each proposal an issue**,
through the one way in ADR-0018 sets for every role — no duplicate, labelled
`needs-triage`, its evidence in it — for the person to accept, reject or
leave; the product owner keeps it from there. The role closes nothing.

**Its own measure** is the follow-through of its proposals: accepted and
done, rejected, left; and whether the next windows show the effect it
announced. Proposals ignored three windows running pause it, as the product
owner's ignored runs do.

### Borrowed, built, proved

**Borrowed**: the effective-false-positive signal, outcome per proposal
kind, dead rules after a quarter, one verdict per input, a judge's budget,
proposals with evidence merged by a person, `gen_ai.*` names. **To build**:
the engine's run line and where it is kept; reading what became of an act
on each forge; the counts; the role. **Proof**: conformance cases on the
counts (an act undone counted, a check never fired flagged, nothing written
without a forge); then a real window on workline and DomoticsCore, its
flags read by the person before any proposal is trusted.

## Open — for the person

1. **Where a run's line is kept** when CI throws the run away: on the
   forge (a comment kept on the report issue), on a ref of the repository
   (`refs/workline/runs`, written by the step holding the token), or in
   CI artifacts (gone after their retention).
2. The name: process engineer, or another.
3. The window (a quarter), the cadence (weekly), the token budget.

## Consequences

- The line's own defects are found in its real runs, not only when a
  person runs it by hand.
- Each automatic mechanism earns its place by measured use; a dead or noisy
  one is proposed for removal, with its evidence.
- Nothing the role finds changes the line without a person.

## Amendment (2026-10-06)

Decided by the person: the **process engineer** keeps its name (Open,
point 2), and stays apart from the **auditor**. The auditor re-checks a
sample of every role's acts and gives a verdict per act
([ADR-0015](0015-the-weekly-sample-is-written-on-the-forge.md), amended);
the process engineer reads the measures — the auditor's verdicts, the
evaluation's results, the run records, what people did after — and
proposes changes to the line as issues. The first never proposes, the
second never re-judges an act: a quality audit and industrial engineering,
as in a factory ([roles panorama](../research/roles-panorama.md)). It
comes fifth in the roadmap
([#89](https://github.com/JN0V/workline/issues/89)), after the auditor is
widened ([#205](https://github.com/JN0V/workline/issues/205)).
