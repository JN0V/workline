# Developer — where it stands (2026-11-12)

**In one line:** `workline develop` picks the first ready issue nobody
took and, with an agent set, claims it with a label and a comment
holding its record, then has the agent write the code, held to its
bounds at apply, proves it on the project's tests and opens a pull
request on the tester's Draft, ready when green with no cap reached,
else a Draft, its record hidden on it; with none, it names the issue.
Routed, it runs when an issue is labelled ready and on the schedule.
On its pull request, a fresh run fixes each review: the five whys at
round three, a person past round five. At the merge policy's `normal`,
its pull request for a reproduced bug's fix, reviewed at its head, merges with no person's
yes; a feature or a task waits for one. Measured on planted issues and a copy of a
real repository: the eight steps of
[ADR-0054's plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#build-plan)
built; beta; routed where a project lists it, in workline's own line
since [#696](https://github.com/JN0V/workline/issues/696), its pull
requests merged there since 2026-11-04 (a fix,
[#1244](https://github.com/JN0V/workline/pull/1244), by the policy).
In the V1 acceptance, it took 9 issues on three repositories to a
merge
([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764);
[V1 accepted](../../../docs/adr/0052-v1-0-when-the-core-roles-are-out-of-beta.md#amendment-2026-11-11-1266-v1-accepted)).

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

1. **Nothing buildable in Missing**: open. Two items are kept for
   1.0: its review rounds on GitLab
   ([#1381](https://github.com/JN0V/workline/issues/1381)) and a bug in
   its pick ([#1382](https://github.com/JN0V/workline/issues/1382)); the
   rest of [Missing](#missing) is parked or tries.
2. **The main issue validated on real use**: **awaits the
   maintainer**. [#696](https://github.com/JN0V/workline/issues/696)'s
   Validation: a maintainer reads a pull request the developer opened
   for a `ready` issue, its Verification proved in it, and merges it or
   sends it back. The maintainer would sign that one of these meets it:
   - on workline's own line, its fixes merged by the policy with no
     person's yes, e.g. [#1244](https://github.com/JN0V/workline/pull/1244)
     (2026-11-09), and a `feat`,
     [#1141](https://github.com/JN0V/workline/pull/1141); none read by
     a person before its merge;
   - in V1 round 3, 9 of 9 issues merged on Kotlin, C++ and Python
     copies, 3 features after a person's yes
     ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764);
     rounds [1](https://github.com/JN0V/workline/issues/1266#issuecomment-6779607472)
     and [2](https://github.com/JN0V/workline/issues/1266#issuecomment-6796923928)
     before it).
3. **Observable**: open, in part met. Its acts are kept for the
   auditor
   ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row)):
   the pick's record from step 1, its refusals at apply in it from
   step 2, its proof and caps reached from step 3; its code's record
   on its pull request from step 4, drawn by the weekly sample (the
   auditor's `sample-draws-the-developers-code`); its review rounds
   from step 6, drawn with it (`sample-draws-the-developers-rounds`).
   Its code merged on workline is drawn: 4 acts, none re-checked by a
   judge yet
   ([the count](https://github.com/JN0V/workline/issues/155#issuecomment-6265992275);
   [#1390](https://github.com/JN0V/workline/issues/1390)). Measured
   with a real agent on planted issues, 29 of 30 ([Measured](#measured)).

## Built

- **The pick** (step 1): the first ready issue in the backlog's order,
  assigned to nobody, not taken, no open merge request closing it, its
  Scope clear of the files the issues held change (their pull requests'
  diffs, [#1256](https://github.com/JN0V/workline/issues/1256)); claimed by a label, since GitHub
  assigns no issue to an App
  ([the pick](pick.md#the-claim)); its record hidden in its comment.
  Cases `developer/picks-first-ready-unassigned`,
  `skips-assigned-issue`, `skips-issue-with-open-pr`,
  `skips-overlapping-scope`, `overlap-reads-the-held-pull-requests-diff`,
  `overlap-predicted-only-does-not-wait`,
  `overlap-reads-the-diff-in-a-shallow-clone`, `none-ready-says-so`,
  `no-agent-opens-nothing`.
- **The bounds** (step 2): the agent's patches held, file by file, to
  the frozen acceptance tests, the files that steer agents and a gate's
  definition; a label or comment of its own refused; an objection
  posted on the test's line of the tester's Draft
  ([the bounds](bounds.md)). The Scope is a hint since [#1228](https://github.com/JN0V/workline/issues/1228): a
  file beyond it is applied, listed on the pull request with its
  agent's reason, judged by the reviewer's `intent` lens
  ([beyond the Scope](bounds.md#beyond-the-scope)). Cases
  `developer/applies-a-file-beyond-the-scope`,
  `worktree-beyond-the-scope-said`, `takes-code-only-beyond-the-scope`,
  `name-only-is-beyond-the-scope`, `round-fixes-beyond-the-scope`,
  `pr-bounds-says-a-file-beyond-the-scope`,
  `refuses-acceptance-test-edit`, `refuses-steering-file`,
  `refuses-gate-file`, `has-no-merge-intention`, `objects-to-a-test`,
  `objection-names-a-test-mapped`, `objection-names-no-scenario-held`.
- **The proof and the caps** (step 3): its code laid on the commit read,
  the tester's `test-command` run on it; ready when green with no cap
  reached (`ai-max-tokens`, `proof-seconds`), else a Draft, the proof
  and caps in the claim's record ([the proof](proof.md)). Cases
  `developer/green-opens-ready`, `red-opens-draft`,
  `cap-reached-pushes-draft`; a tool not found named on a red
  ([#760](https://github.com/JN0V/workline/issues/760)): `proof-names-missing-tool`.
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
- **One note on its own issue**
  ([#694](https://github.com/JN0V/workline/issues/694)): what the agent
  left out, posted once on its issue, inert and cut to 4,000 bytes, the
  same text in the pull request's `## Left out`; a second note, or one
  elsewhere, refused ([the bounds](bounds.md#what-else-it-refuses)).
  Case `developer/note-on-its-own-issue`.
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
- **A pull request that does not open**
  ([#856](https://github.com/JN0V/workline/issues/856)): its code
  merged onto a Draft the default branch has moved past; a failure
  gives the claim back at once, the reason on the issue, and blocks
  the issue the second time
  ([the pull request](pull-request.md#when-it-does-not-open);
  [tried](https://github.com/JN0V/workline/pull/863)). Cases
  `developer/pr-from-draft-behind-main`, `unopened-gives-claim-back`,
  `unopened-twice-blocks`.
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
  bounds read at the base's tip, whoever pushed, a file beyond the
  Scope said, never refused; its record kept on the pull request
  ([the bounds](bounds.md#on-the-pull-request)). Cases
  `developer/pr-bounds-says-a-file-beyond-the-scope`,
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
- **A gate red is a round** ([#891](https://github.com/JN0V/workline/issues/891)):
  the gates' new findings at its head asked once a report, the
  inspector's triage beside each; the reconciler starts a round an
  event missed ([the review loop](review-loop.md#a-round)). Cases
  `developer/gate-finding-is-a-round`, `gate-finding-read-once`,
  `gate-finding-of-an-older-head-waits`,
  `routing/reconcile-lists-a-developer-round-due`.
- **A project check red is a round** ([#917](https://github.com/JN0V/workline/issues/917)):
  its CI's checks at its head, workline's own left out, asked once a
  head when all are done, the end of the failed steps' log in the task;
  a failed workflow wakes it, the reconciler as fallback
  ([the review loop](review-loop.md#a-round)). Cases
  `developer/red-check-is-a-round`, `red-check-read-once`,
  `red-check-waits-for-the-others`, `red-check-of-workline-asks-nothing`,
  `routing/reconcile-lists-a-red-check-round`.
- **A fix outside the Scope asked the product owner** ([#921](https://github.com/JN0V/workline/issues/921)),
  until [#1228](https://github.com/JN0V/workline/issues/1228) made the Scope a hint: a round's fix beyond it is
  pushed and listed ([the review loop](review-held.md#a-fix-beyond-the-scope)).
  An issue still blocked by an ask resumes as before. Cases
  `developer/round-fixes-beyond-the-scope`,
  `round-resumes-on-a-widened-scope`,
  `round-resumes-after-a-person-widens`,
  `routing/scope-ask-wakes-the-product-owner`,
  `scope-ask-refused-stays-blocked`,
  `reconcile-lists-a-round-back-from-scope`. Tried: its pull request's
  `## Tried`.
- **A round's move of its issue applied in CI** ([#923](https://github.com/JN0V/workline/issues/923)):
  the apply job refused the whole run, its label on the issue its pull
  request closes read as on another issue. Now that issue alone, to
  Blocked or Doing. Case `developer/round-hold-applied-from-the-default-branch`.
- **A log quoted in the ask, its marks broken** ([#925](https://github.com/JN0V/workline/issues/925)):
  a red check's log that quoted workline's hidden marks hid the ask: the
  line read a test's record as the issue's, the product owner its state,
  and the line wrote over the ask. Now the quote's marks are broken.
  Cases `developer/round-hold-quotes-a-log-with-workline-marks`,
  `routing/scope-ask-with-a-quoted-log-wakes-the-product-owner`.
- **A widened Scope resumes its round at once** ([#1013](https://github.com/JN0V/workline/issues/1013)):
  back to ready, the issue's step was a new claim the developer passed
  over, and no round ran; the ask waited for the hourly reconcile. Now
  the job that moved the issue starts its round, and the job that
  applied a round wakes the issue's next step
  ([triggers](triggers.md#on-its-pull-request)). Cases
  `routing/ready-back-from-scope-lists-its-round`,
  `ready-back-from-scope-says-its-round`,
  `ready-with-a-persons-pull-request-still-due`. Tried: its pull
  request's `## Tried`.
- **A red check with no fix and no file to ask blocks** ([#938](https://github.com/JN0V/workline/issues/938)):
  its pull request sat red for hours, the check not asked again at its
  head. Now its issue goes Blocked for a person, the check, the note and
  the objection's place on it; moved back, the round runs again
  ([the review loop](review-held.md#a-red-check-it-does-not-fix)).
  Cases `developer/round-holding-a-check-wrong-blocks`,
  `round-held-before-runs-once-more`, `round-resumes-after-a-held-check`.
  Tried: its pull request's `## Tried`.
- **A round's objection waits for the tester; a green head is kept** ([#987](https://github.com/JN0V/workline/issues/987)):
  a review finding's fix undid the code a frozen test needed, red; the
  next round objected and blocked its issue for a person. Now a fix red
  on a frozen test, on a green head, is not pushed; a round that
  objects waits for the tester in Doing, a test kept asks it again
  ([a finding against a frozen test](review-held.md#a-finding-against-a-frozen-test)).
  Cases `developer/round-keeps-a-green-head`,
  `round-objecting-to-a-red-test-waits`,
  `round-asks-again-once-the-tester-keeps`, `round-undoing-green-blocks`,
  `round-objecting-again-on-a-finding-blocks`,
  `merge-alone-said-while-checks-run`. Tried: its pull request's
  `## Tried`.
- **A Draft made green is marked ready** ([#969](https://github.com/JN0V/workline/issues/969)):
  its Draft, opened red, stayed a Draft once its rounds made it green,
  so nothing reviewed it. Now, every project check at its head green
  and no cap reached, a round with no agent marks it ready; all green
  wakes it on a Draft, the reconciler as fallback
  ([a Draft made green](review-loop.md#a-draft-made-green)). Cases
  `developer/green-draft-made-ready`, `green-draft-capped-stays-draft`,
  `red-check-round-keeps-draft`, `routing/reconcile-lists-a-green-draft`.
  Tried: its pull request's `## Tried`.
- **A test finding on code goes to the tester** ([#1265](https://github.com/JN0V/workline/issues/1265)):
  the tests lens asked missing tests of the developer, who may not
  write acceptance tests; past the last round, the issue was blocked.
  Now each goes to the tester's Draft, an objection of kind `missing`,
  once; alone, no agent and no round counted
  ([the review loop](review-loop.md#a-round)). Cases
  `developer/round-sends-a-test-finding-to-the-tester`,
  `round-splits-code-and-test-findings`,
  `round-five-test-findings-not-blocked`,
  `round-five-blocks-on-code-sends-tests`,
  `round-asks-no-test-finding-sent-before`,
  `round-blocks-on-a-test-finding-held-needless`. Tried: its pull
  request's `## Tried`.

- **The merge** (step 7): at `normal`, `workline follow` merges its
  pull request with no person's yes, the kind `code`, when the issue it
  closes is a fix by its labels, every gate passed and no finding open
  that blocks or is Important; a feature, a task or no kind waits for a
  person's yes ([#937](https://github.com/JN0V/workline/issues/937),
  [tried](https://github.com/JN0V/workline/pull/942)). Each decision
  recorded on the pull request, read by the auditor with its code
  ([the merge](merge.md)). Cases
  `developer/merge-normal-admits-developer-fix`,
  `merge-normal-holds-developer-feature`,
  `merge-normal-holds-developer-task`,
  `merge-enterprising-admits-developer-task`,
  `merge-enterprising-candidate-refused-tasks-wait`.
- **A bug's fix, reproduced first**
  ([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind)):
  merges alone only when its code starts from the tester's Draft, a
  tester's commit on its branch; else it waits, said. Cases
  `merge-normal-holds-unreproduced-fix`,
  `merge-cautious-holds-reproduced-fix`. Tried on
  [#876](https://github.com/JN0V/workline/pull/876#tried), the merge
  itself not on a forge.
- **A fix reviewed at its head**
  ([#1254](https://github.com/JN0V/workline/issues/1254),
  [ADR-0065](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence)):
  merges alone only when the reviewer's summary names its head, its
  lenses run, or its record covers a head rebased since; none, or one
  on an older commit, waits for the reviewer where the line runs it,
  24 h at most, then for a person, said. Cases
  `merge-normal-holds-unreviewed-fix`,
  `merge-normal-holds-fix-reviewed-before-head`,
  `merge-normal-admits-fix-rebased-since-review`,
  `merge-normal-fix-waits-for-the-reviewer`,
  `merge-normal-fix-unreviewed-past-the-delay`. Tried: its pull
  request's `## Tried`.
- **A feature at `enterprising`**
  ([the release candidate](../../../docs/release-candidate.md)): merges
  alone, put in a milestone first; a refused candidate sends features
  back to a person's yes. Cases
  `merge-enterprising-admits-developer-feature`,
  `merge-enterprising-candidate-refused-features-wait`. The candidate
  tried on the sandbox
  ([#883](https://github.com/JN0V/workline/pull/883#tried)), a feature
  merged alone not.
- **Behind main** ([#1353](https://github.com/JN0V/workline/issues/1353),
  [ADR-0067](../../../docs/adr/0067-a-role-request-behind-main-merges-main-in.md)):
  its merge refused for a conflict, one round merges `main` in, the
  markers settled, the engine's two-parent commit pushed with no force;
  clean and held by the forge, the engine merges it in alone. Cases
  `merge/local-conflict-merged-in-by-a-round`,
  `merge/behind-held-merged-in-clean`,
  `merge-round-dropping-main-not-pushed`,
  `merge-round-markers-left-stops`. Tried: its pull request's `## Tried`
  ([#1368](https://github.com/JN0V/workline/pull/1368)); in V1 round 3,
  one merge round settled the C++ feature's conflict with no hand act
  ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).
  So `workline follow` no longer leaves its branch behind main.
- **The reviewer's findings, whole**
  ([#705](https://github.com/JN0V/workline/issues/705)): read from the
  hidden record of the reviewer's comment, never its table's words; a
  finding on its line read whole from that line's comment
  ([the review loop](review-loop.md);
  [#1312](https://github.com/JN0V/workline/pull/1312)). Case
  `developer/fixes-finding-with-the-table-reworded`.
- **A claim let go puts ready back**: its card leaves Doing
  ([#1115](https://github.com/JN0V/workline/pull/1115); the line's
  states, [#756](https://github.com/JN0V/workline/issues/756)).
- **Real issues, with `claude` and its tools**: the V1 acceptance, three
  rounds of 9 issues on Kotlin, C++ and Python copies, a local forge;
  round 3 merged 9 of 9, each issue's whole line 75k to 445k fresh
  tokens, every role counted
  ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).

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

Kept for 1.0, each an open issue:

- **A failed pipeline or a gate red on GitLab**: the project's checks
  are read on GitHub only; on GitLab, the merge request's head
  pipeline and its failed jobs' traces would be the round's input, a
  pipeline event its wake; its gates show in the Code Quality report,
  no comment a round reads. So a Draft made green is not marked ready
  there yet ([#969](https://github.com/JN0V/workline/issues/969)). The
  reviewer's verdict and a person's comment still wake a round there
  (kept for 1.0, [#1381](https://github.com/JN0V/workline/issues/1381)).
- **"Implements #4" on GitHub**: the pick reads `implement` as a
  closing word, as GitLab does; GitHub does not close on it, so there
  the issue is skipped as taken though nothing will close it (kept for
  1.0, a bug, [#1382](https://github.com/JN0V/workline/issues/1382)).

Parked, closed as not planned with `workline:parked`, reopened when
real use asks:

- **Its shell boxed in its worktree**: Claude Code's sandbox, writes in
  the worktree only, no network by default (parked,
  [#719](https://github.com/JN0V/workline/issues/719)).
- **A test runner named like a test**: `run-tests.sh` beside the Scope
  passes the bounds as the agent's own test, so its agent can rewrite
  the runner the proof calls (parked,
  [#715](https://github.com/JN0V/workline/issues/715)).
- **The issue's kind from the forge's types and the project's labels**:
  read from a fixed list of labels only
  ([the merge](merge.md#not-yet); parked,
  [#1383](https://github.com/JN0V/workline/issues/1383)).
- **A finding the developer holds wrong**: said in its round's
  comment; no thread with the reviewer, who decides on its next run
  (parked, [#1384](https://github.com/JN0V/workline/issues/1384)).
- **Two runs coding at once, and the claim on GitLab by assignment**:
  the claim is read again just before apply, not held by the forge;
  two runs whose agents code at the same time can still both take an
  issue ([the triggers](triggers.md#two-runs-on-one-issue)); a label
  on every forge, where a GitLab service account could be assigned
  (parked, [#1385](https://github.com/JN0V/workline/issues/1385)).
- **A refused patch asked again**: today it is left out and the rest
  applied; so for a diff that does not apply: unproven, a Draft
  ([measured](#measured); parked,
  [#692](https://github.com/JN0V/workline/issues/692)).
- **A test of its own with no tester's Draft**: asked by the issue's
  Verification, the agent wrote none ([measured](#measured); parked,
  [#693](https://github.com/JN0V/workline/issues/693)).
- **Scope files with accented names, or listed twice**: git quotes
  `café.sh`, so the agent is not shown it; a folder and its file both in
  the Scope show the file twice (parked,
  [#695](https://github.com/JN0V/workline/issues/695)).
- **A new file in a new folder**, from an agent with no tools: the
  engine writes a `{file, content}` patch only where its folder exists;
  a diff creates it (parked,
  [#1386](https://github.com/JN0V/workline/issues/1386)).
- **The proof's other parts, and a run's time as one cap**: the
  project's gates and the commands the issue's Verification names
  ([ADR-0054](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#it-proves-locally-first))
  do not run in it; the pull request's checks run the gates. The
  agent's wait (`ai-max-seconds`) and the tests' (`proof-seconds`) are
  two caps whose defaults sum to the ADR's 30 min a run; a project
  raising one is not held to the sum (parked,
  [#1387](https://github.com/JN0V/workline/issues/1387)).

Tries on real use (parked,
[#1390](https://github.com/JN0V/workline/issues/1390)):

- **The bounds under a GitLab pipeline execution policy**: written,
  not tried; no Ultimate project at hand
  ([on GitLab](../../../docs/gitlab/bounds.md#ultimate-a-pipeline-execution-policy)).
- **The auditor's draw of its code, by a real judge**: drawn on
  workline's merges, 4 acts, none re-checked yet
  ([the count](https://github.com/JN0V/workline/issues/155#issuecomment-6265992275)).
