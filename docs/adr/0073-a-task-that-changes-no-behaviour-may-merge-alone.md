# ADR-0073: A task that changes no behaviour may merge alone, where the project lets it

- **Status:** accepted (built)
- **Date:** 2026-11-25
- **Decided by:** the maintainer, 2026-11-25
- **Amends:** [ADR-0060 §4](0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind)
  (a task waits for a person's yes, at `normal`)
- **Builds on:** [ADR-0042](0042-the-engine-merges-by-the-projects-policy.md)
  (`merge.alone`), [ADR-0065 §3](0065-code-only-what-converges-language-knowledge-in-packs.md#3-one-definition-of-a-test-file)
  (one definition of a test file), [ADR-0005](0005-independence-takes-the-best-level-available.md)
  (the judge); principles 4, 12 and 15

## Context

- **At `normal`, only a reproduced bug's fix merges alone.** A task
  waits for a person's yes: `task` is a label an agent may put
  ([ADR-0060, amended 2026-11-03](0060-the-person-is-on-the-loop.md#amendment-2026-11-03-a-task-waits-as-a-feature)).
- **Most tasks change no behaviour**: a refactor, a rename inside a
  package, a comment. The person's yes on them reads code the tests
  already hold.
- **What makes a fix safe is evidence**, not its label: a test red on
  the base. A task needs its own evidence.

## Decision

**At `normal`, a task merges with no person's yes when it shows it
changes no behaviour, and only where the project lists it:
`merge: {autonomy: normal, alone: [task]}`. Off by default.**

### The facts that decide it

All of a fix's gates, as they are
([the developer's code](../merge-policy-code.md)): workline's own
commits, its head's verdict passed, the reviewer's verdict on its head,
no finding open that blocks or is Important. Then:

- **The tests as they were**: the change modifies, deletes or renames
  no file the project's test definition calls a test (the tester's
  `test-paths`). A test added is allowed. The checks green on the head
  then say the tests that passed before still pass.
- **The judge's yes on the head**: the reviewer asks the judge
  ([roles/judge](../../roles/judge/role.yaml)) one question on the whole
  change and the issue: does it keep the behaviour? A yes is written on
  the reviewer's summary, by head; `workline follow` reads it there.

### Excluded, always a person's yes

- **Person-only paths**: a file `CODEOWNERS` lists (CI files,
  `.workline/`): never alone, any kind
  ([ADR-0068 §4](0068-the-line-steps-back-on-its-own-drift.md#4-the-guard-is-guarded)).
- **A public interface changed**, **the toolchain or build platform
  changed**, **a limit or value a person chose changed**: what the
  judge is asked. These need judgement; no path list or language rule
  tells them everywhere (principle 4).
- **Unsure is no**: a judge's no, unsure, no answer, a diff past the
  reviewer's `diff-lines-max`, or a head the judge did not answer on,
  waits (principle 15: a guess may only pass).

### Off by default, measured before it is turned on

- **A setting, not a level**: `task` joins `reader` in `merge.alone`;
  `cautious` refuses it.
- **No project turns it on before a measure**: DomoticsCore only after
  20 pull requests counted in
  [#2190](https://github.com/JN0V/workline/issues/2190), the tasks
  counted apart.
- **A revert demotes it**, as any kind merged alone
  ([reverts](../merge-policy-reverts.md)).

## Consequences

- **One judge call a push** on a task's pull request, only where the
  setting is on.
- **No new role, no new check**: the reviewer asks, the judge answers,
  the engine reads.
- **The auditor reads it** as any `code` merged alone: its record says
  the kind, `task`.

## Not chosen

- **A path list for the toolchain** (`go.mod`, `Makefile`): language
  knowledge, partial, and the public interface and chosen values need
  judgement anyway.
- **Asking the judge in `workline follow`**: the job holding the write
  token calls no agent
  ([ADR-0034](0034-the-release-fix-follows-its-base.md)).
