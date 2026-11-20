---
sources:
  - defaults/ci/github/workline-answer.yml#on
  - defaults/ci/github/workline-answer.yml#jobs.propose
  - defaults/ci/github/workline-answer.yml#jobs.apply
  - internal/line/answerstate.go#answerState
  - internal/line/answerstate.go#MarkAnswerTried
  - cmd/workline/main.go#answerStateCmd
checked: c46a93d6c
verified: agent:claude-code
---
# One objection a run

Part of [An objection's jobs](objection-jobs.md): the queue drained,
on GitHub only.

- An answer run answers one objection
  ([#1701](https://github.com/JN0V/workline/issues/1701)).
- apply then starts `workline-answer.yml` again on the same Draft
  (`workflow_dispatch`, input `branch`): the next one waiting. With
  none left, the run ends before any agent. 5 answers at most in a
  chain (input `chain`); the reconciler resumes after
  ([who wakes it](objection-jobs.md#who-wakes-it-github)).
- Each run writes on the Draft the state it ran on, whatever it
  wrote, unless its agent was out of reach: the tip and each
  objection waiting
  (`workline test answer --tried`). A run on that same state
  (`--state` says `tried`) calls no agent: only a new comment in a
  waiting thread, a new objection or a new head starts one again.
- A state not read (a release before `--state`, or the forge
  failing): the answer runs as before, a warning in the log, and no
  state is written
  ([the N-1 rule](adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md#consequences)).
