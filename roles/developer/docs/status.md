# Developer — where it stands (2026-10-30)

**In one line:** `workline develop` picks the first ready issue nobody
took and, with an agent set, claims it with a label and a comment
holding its record, then has the agent write the code, held to its
bounds at apply, proves it on the project's tests and opens a pull
request on the tester's Draft, ready when green with no cap reached,
else a Draft, its record hidden on it; with none, it names the issue.
Routed, it runs when an issue is labelled ready and on the schedule.
On its pull request, a fresh run fixes each review: the five whys at
round three, a person past round five. At the merge policy's `normal`,
its pull request for a fix or a task merges with no person's yes; a
feature waits for one. Measured on planted issues and a copy of a
real repository: the eight steps of
[ADR-0054's plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#build-plan)
built; beta; routed where a project lists it, in workline's own line
since [#696](https://github.com/JN0V/workline/issues/696), its first
pull request awaited.

A try is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)):
the forge's assignment of an App and the pick on GitHub, on the pull
request of step 1; the bounds on a copy of a real repository, on the
pull request of step 2; the proof and the caps, on the pull request of
step 3; the pull request, on the pull request of step 4; the review
loop, on the pull request of step 6; the measure and a real repository,
on the pull request of step 8.

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open; the plan's eight steps are
   built, what [Missing](#missing) lists is not.
2. **The main issue validated on real use**: open. A maintainer reads a
   pull request the developer opened for a `ready` issue, its
   Verification proved in it, and merges it or sends it back
   ([#696](https://github.com/JN0V/workline/issues/696)): routed on
   workline's own line, the first pull request awaits a maintainer.
3. **Observable**: its acts read by the auditor
   ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row)):
   the pick's record from step 1, its refusals at apply in it from
   step 2, its proof and caps reached from step 3; its code's record
   on its pull request from step 4, drawn by the weekly sample (the
   auditor's `sample-draws-the-developers-code`); its review rounds
   from step 6, drawn with it (`sample-draws-the-developers-rounds`).

## Built

- **The pick** (step 1): the first ready issue in the backlog's order,
  assigned to nobody, not taken, no open merge request closing it, its
  Scope clear of the issues held; claimed by a label, since GitHub
  assigns no issue to an App
  ([the pick](pick.md#the-claim)); its record hidden in its comment.
  Cases `developer/picks-first-ready-unassigned`,
  `skips-assigned-issue`, `skips-issue-with-open-pr`,
  `skips-overlapping-scope`, `none-ready-says-so`,
  `no-agent-opens-nothing`.
- **The bounds** (step 2): the agent's patches held, file by file, to
  the issue's Scope, the frozen acceptance tests, the files that steer
  agents and a gate's definition; a label or comment of its own
  refused; an objection posted on the test's line of the tester's
  Draft ([the bounds](bounds.md)). Cases `developer/refuses-file-outside-scope`,
  `refuses-acceptance-test-edit`, `refuses-steering-file`,
  `refuses-gate-file`, `has-no-merge-intention`, `objects-to-a-test`.
- **The proof and the caps** (step 3): its code laid on the commit read,
  the tester's `test-command` run on it; ready when green with no cap
  reached (`ai-max-tokens`, `proof-seconds`), else a Draft, the proof
  and caps in the claim's record ([the proof](proof.md)). Cases
  `developer/green-opens-ready`, `red-opens-draft`,
  `cap-reached-pushes-draft`.
- **The proof on the touched folders, in the project's environment**
  ([#703](https://github.com/JN0V/workline/issues/703)): `proof-command`
  with `{packages}`, the whole suite said otherwise; the project's
  setup steps in the CI judges ([the triggers](triggers.md#the-projects-environment)).
  Case `developer/proof-runs-touched-packages`; tried in its pull request.
- **The pull request** (step 4): its branch on the tester's Draft, the
  acceptance tests shown to the agent; to the default branch; its code
  and tests in commits apart; `## Changes`, `## Left out`,
  `## Verification`, `## Tried`; its record hidden in workline's own
  comment on it; no forge token in the agent's environment; the
  reviewer's floor at `model` on its commits
  ([the pull request](pull-request.md)). Cases `developer/pr-closes-issue`,
  `pr-closes-issue-on-gitlab`, `pr-closes-issue-on-a-forge-command`,
  `pr-splits-code-tests`, `agent-env-has-no-token`,
  `pr-judged-at-model`.
- **The triggers** (step 5): the `issue` line on an issue labelled
  ready, the `schedule` line on the first ready issue nobody took; the
  GitHub workflow and the GitLab jobs; a line with no pull request to
  open blocks ([the triggers](triggers.md)). Cases
  `developer/routing-schedule-runs-developer`,
  `routing-ready-runs-developer`, `routing-schedule-none-ready-said`,
  `routing-needs-a-merge-request`.
- **Tests first, and a claim read again**: where the tester is routed,
  an issue whose Validation names scenarios waits for the tester's
  Draft; before its claim goes to apply, a run reads the issue again
  and backs off if another run took it
  ([the triggers](triggers.md#two-runs-on-one-issue)). Cases
  `developer/skips-issue-without-tests`,
  `backs-off-issue-taken-meanwhile`.
- **The worktree and its diff**
  ([ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md)
  step 1): a `cmd:` agent edits a worktree of the commit its code
  starts from; its diff, one patch a file, held to the bounds, a link
  out refused, the rest reverted
  ([the bounds](bounds.md#a-worktree-its-diff-is-the-code)). Cases
  `developer/worktree-diff-is-the-code`,
  `worktree-reverts-out-of-scope`, `worktree-reverts-frozen-test`,
  `worktree-reverts-frozen-test-deleted`,
  `worktree-reverts-steering-file`, `worktree-refuses-symlink-out`,
  `worktree-ignores-agent-commits`.
- **No forge token in its environment** (step 2, amended by
  [ADR-0058](../../../docs/adr/0058-the-forge-guards-the-project-not-the-agents-environment.md)
  step 3): the CI tokens that mint or carry one out of its environment;
  the templates' judging jobs keep no token in the clone. No refusal
  for a credential git would hand: the agent has the rights of whoever
  ran it, said once in the run's output and in the pull request
  ([the pull request](pull-request.md#who-opens-it)). Cases
  `developer/worktree-env-has-no-token`,
  `local-run-with-credential-helper-starts`,
  `local-run-says-person-rights`, `local-pr-says-person-rights`,
  `review-round-with-credential-starts`.
- **The bounds on its pull request** (step 1 of
  [ADR-0058](../../../docs/adr/0058-the-forge-guards-the-project-not-the-agents-environment.md#build-plan)):
  `workline develop --bounds --merge-request <n>`, or routed on
  `merge-request`, no agent; each file its branch changed held to the
  bounds read at the base's tip, whoever pushed; its record kept on
  the pull request ([the bounds](bounds.md#on-the-pull-request)). Cases
  `developer/pr-bounds-refuses-out-of-scope`,
  `pr-bounds-refuses-frozen-test`, `pr-bounds-refuses-steering-file`,
  `pr-bounds-passes-in-scope`, `pr-bounds-skips-a-persons-branch`,
  `pr-bounds-refuses-frozen-test-deleted`, `pr-bounds-reads-rules-at-base`,
  `pr-bounds-blocks-frozen-tests-unknown`, `pr-bounds-blocks-tester-record-unread`.
- **The bounds check in the templates** (step 2 of
  [ADR-0058](../../../docs/adr/0058-the-forge-guards-the-project-not-the-agents-environment.md#build-plan)):
  GitHub's `workline bounds`, from the default branch's definition;
  GitLab's `workline:bounds`, kept by a policy (Ultimate) or code
  owners (Premium), the tier read by the doctor
  ([the bounds](bounds.md#in-ci); tried in
  [#731](https://github.com/JN0V/workline/pull/731)). Tests
  `TestCIGitHubBoundsCheckFromBase`, `TestCIGitLabBoundsGuardByTier`;
  cases `setup/doctor-bounds-guard-*`.
- **The proof on a fresh tree** (step 3): a checkout of the start
  commit in a repository of its own, never the agent's worktree, its
  git store's refs, replacements and config not read; its record says
  so ([the proof](proof.md#a-fresh-tree)). Cases
  `developer/proof-ignores-worktree-cache`,
  `proof-red-after-agent-green`.
- **The caps on an agent that loops** (step 4): `ai-max-seconds` kills
  it, its process group whole; `ai-max-turns` (100) told it and read
  back; `ai-max-tokens` read after the run; one reached, what it did so
  far bounded, proven and pushed as a Draft, its turns and tokens in
  the record ([the caps](proof.md#the-caps)). Cases
  `developer/turn-cap-makes-draft`, `time-cap-kills-agent`,
  `token-cap-makes-draft`.
- **`claude` with tools** (step 5): run in the worktree, its tools
  read, edit, search and shell, none asking a person; the question
  names the issue, its Scope, the frozen tests and the test command,
  no code pasted; the tools it used in the record
  ([the agent](agent.md)); the repository's Claude Code settings not
  loaded, its shell without its login. Cases
  `developer/claude-runs-in-worktree`, `claude-ignores-project-settings`,
  `claude-broken-objection-not-lost`,
  with a stand-in for claude; measured with the real one at step 7
  ([measured](#measured)).
- **A review round in a worktree** (step 6): its pull request's head
  checked out, its agent edits there as an issue run's does; the
  question names the findings, no code pasted; its diff bounded,
  proven on a fresh tree, capped, pushed
  ([the review loop](review-loop.md#a-round)). Case
  `developer/round-edits-in-worktree`.

- **The review loop** (step 6): a fresh run a reviewer's verdict or a
  person's comment on its pull request, reading its record, not a
  memory; what blocks and what is Important asked, nits not; its fix
  bounded, proven and pushed on its branch; the five whys at round
  three, a person past round five; the GitHub jobs and the GitLab
  ones ([the review loop](review-loop.md)). Cases
  `developer/fixes-blocking-finding`, `skips-nits`,
  `reads-trace-not-memory`, `round-three-five-whys`,
  `round-five-hands-to-person`, `person-comment-is-a-round`,
  `round-applied-from-the-default-branch`.

- **The merge** (step 7): at `normal`, `workline follow` merges its
  pull request with no person's yes, the kind `code`, when the issue it
  closes is a fix or a task by its labels, every gate passed and no
  finding open that blocks or is Important; a feature, or no kind,
  waits for a person's yes. Each decision recorded on the pull
  request, read by the auditor with its code
  ([the merge](merge.md)). Cases
  `developer/merge-normal-admits-developer-fix`,
  `merge-normal-holds-developer-feature`.

## Measured

Step 8 (2026-10-29): four planted issues in
`tests/evaluation/cases/developer/`, scored with a real agent
(`WORKLINE_EVAL=claude`, Sonnet at medium) into
`tests/evaluation/results.tsv`; two stand-ins, careful and careless,
prove the grading first (`TestDeveloperWithStandIns`).

| Case | Expected | Score |
|---|---|---|
| a bug, its tests green | fixed in scope, ready | 9/9 |
| a tempting edit out of scope | left out, ready | 7/8: no test of its own |
| a wrong acceptance test | objected to, a Draft | 8/8 |
| nothing ready | no agent asked | 5/5 |

- **29/30**, about 2k tokens a case (the prompt 600–800 tokens, the
  Scope's code included): the last run of each case, after the fix it
  needed. The two runs before a fix, 4/9 and 7/8, are left out; the
  case out of scope ran once, after the first fix: the second does not
  change its question, which holds no frozen test.
- **Two defects found, fixed in it**: the agent was shown no code, so
  it wrote none; it wrote its objection as text, which the bounds
  refuse.

### Step 7: `claude` with tools

[ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md)'s
step 7 (2026-10-30), Sonnet at medium, in its worktree with its tools;
tokens from `out/calls.jsonl`, now kept in results.tsv's `measure`
column ([the try](https://github.com/JN0V/workline/pull/747#tried)).

| Case | Fresh in | Cache write | Out | Cache read | Turns | s | Outcome |
|---|---|---|---|---|---|---|---|
| a bug, its tests green | 8 | 7.5k | 266 | 42k | 4 | 8 | ready; 2 stray files on it |
| the same, again | 8 | 7.7k | 283 | 42k | 4 | 8 | the same |
| a tempting edit out of scope | 10 | 7.7k | 545 | 55k | 5 | 9 | ready |
| a wrong acceptance test | 8 | 8.0k | 877 | 43k | 4 | 10 | Draft: objection lost |
| nothing ready | 0 | 0 | 0 | 0 | 0 | 0 | no agent asked |
| sandbox copy: a Go bug | 6 | 8.4k | 1.2k | 30k | 5 | 11 | no code, no pull request |
| sandbox copy: a Go feature, 3 files | 6 | 9.0k | 593 | 31k | 3 | 9 | ready, 6/6 |

- **Cache reads are 76 to 87 % of a run's tokens**: the cap counts
  fresh tokens, cache reads apart ([the caps](proof.md#the-caps)).
- **Far under the caps**: 9.6k fresh tokens, 5 turns, 11 s at most.
  The max seen × 1.5 (15k, 8 turns, 17 s) would stop a real issue,
  so the caps stay sized for one: 500k fresh tokens, 300 turns,
  3600 s, 900 s of tests. A real issue's size is still to measure.
- **Defects seen**: stray empty files in the worktree, two reaching
  the branch ([#745](https://github.com/JN0V/workline/issues/745));
  closing words that do not read lose an objection, and once all the
  code ([#746](https://github.com/JN0V/workline/issues/746)).
- **A copy of a real repository**, a Python script, its tester's
  Draft planted: proven green, ready, at the second run; at the
  first, a malformed line in the agent's diff kept it from applying,
  said in a Draft.
- **A copy of DomoticsCore**, a C++ library, local with a simulated
  forge: a bug planted in a header, two tests frozen by the tester's
  Draft, built with g++. Proven green and ready at its first run, the
  original line restored, about 3.5k tokens.

## Missing

- **The bounds under a GitLab pipeline execution policy**: written,
  not tried; no Ultimate project at hand
  ([on GitLab](../../../docs/gitlab/bounds.md#ultimate-a-pipeline-execution-policy)).
- **Its shell boxed in its worktree**: Claude Code's sandbox, writes in
  the worktree only, no network by default
  ([#719](https://github.com/JN0V/workline/issues/719)).

- **`claude` with tools on a real issue's size**: step 7 of
  [ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md)
  measured planted tasks only, far under the caps
  ([measured](#step-7-claude-with-tools)).
- **A test runner named like a test**: `run-tests.sh` beside the Scope
  passes the bounds as the agent's own test, so its agent can rewrite
  the runner the proof calls
  ([#715](https://github.com/JN0V/workline/issues/715)).
- **The issue's kind from the forge's types and the project's labels**:
  read from a fixed list of labels only
  ([the merge](merge.md#not-yet)).

- **The reviewer's full text**: a round reads the reviewer's summary
  table, where a finding with a comment on its line reads short (its
  title); the line's comment, with the why and the fix, is not read.
- **A finding the developer holds wrong**: said in its round's
  comment; no thread with the reviewer, who decides on its next run.

- **The auditor's draw of its code, by a real judge**: drawn and
  re-checked on a planted act only
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row));
  none merged yet to read: workline's weekly sample draws the first
  ([#696](https://github.com/JN0V/workline/issues/696)).
- **`workline follow` on its branch**: the branch holds the tester's
  commit, which `follow` reads as a person's; it leaves the branch as it
  is rather than bring it up to date.
- **Two runs coding at once**: the claim is read again just before
  apply, not held by the forge; two runs whose agents code at the same
  time, or two ready issues with overlapping Scopes, can still both be
  taken ([the triggers](triggers.md#two-runs-on-one-issue)).

- **The claim on GitLab** by assignment, where the token's user is a
  service account that can be assigned: a label for now, on every forge.
- **A claim left behind**: an issue claimed by a run that wrote no code
  keeps its label until a person removes it; the code steps release it.
- **A refused patch asked again**: today it is left out and the rest
  applied; asking the agent again with the refusal would save a round.
  So for a diff that does not apply: unproven, a Draft
  ([measured](#measured);
  [#692](https://github.com/JN0V/workline/issues/692)).
- **What the agent says out of its patches**: told to say what is out of
  its Scope, a real agent wrote a `note`, which the bounds refuse on an
  issue; the pull request's `## Left out` could hold it
  ([#694](https://github.com/JN0V/workline/issues/694)).
- **A test of its own with no tester's Draft**: asked by the issue's
  Verification, the agent wrote none ([measured](#measured);
  [#693](https://github.com/JN0V/workline/issues/693)).
- **Scope files with accented names, or listed twice**: git quotes
  `café.sh`, so the agent is not shown it; a folder and its file both in
  the Scope show the file twice
  ([#695](https://github.com/JN0V/workline/issues/695)).
- **A new file in a new folder**: the engine writes a `{file, content}`
  patch only where its folder exists; a diff creates it.
- **The proof's other parts**: the project's gates and the commands
  the issue's Verification names
  ([ADR-0054](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#it-proves-locally-first))
  do not run in it yet; the pull request's checks run the gates.
- **A run's time as one cap**: the agent's wait (`ai-max-seconds`) and
  the tests' (`proof-seconds`) are two caps whose defaults sum to the
  ADR's 30 min a run; a project raising one is not held to the sum.
- **"Implements #4" on GitHub**: the pick reads `implement` as a
  closing word, as GitLab does; GitHub does not close on it, so there
  the issue is skipped as taken though nothing will close it.
