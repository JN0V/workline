---
sources: [internal/engine/backlog.go#report, internal/engine/demote.go#logged, internal/engine/demote.go#holds, internal/engine/engine.go#deferred, internal/engine/engine.go#invalid, internal/engine/engine.go#unanswered, internal/engine/merge.go#later, internal/engine/routine.go#alone, internal/engine/settle.go#derived, internal/role/config.go#shape, internal/role/config.go#check, internal/role/role.go#Role.Asked, internal/role/role.go#Role.Writes, internal/agent/agent.go#unavailable, internal/agent/agent.go#unavailable.Propose, internal/agent/claude.go#claude, internal/agent/claude.go#claude.Propose, internal/agent/claude.go#without, internal/agent/command.go#command, internal/agent/command.go#command.Propose, internal/agent/prompt.go#fence, tests/roles_test.go]
checked: a19e8cb0
verified: agent:claude-code
judged: b530b72
---
# Role contract — one run

Part of [Role contract](role-contract.md): what one run of a role does, step by step.

## One run

```
run-role <name> --event <event> [--ai <agent>|none]
```

Three trust zones, in the order the Linux Foundation uses for its issue triage:
**prepare** (trusted, no AI), **propose** (the agent, untrusted, no write
token), **apply** (trusted, no AI key).

1. **Check.** The engine reads `role.yaml`, verifies `requires`, creates a run
   directory with `in/` and `out/` inside the repository's git directory
   (`.git/workline/runs/<id>/`, so a run never shows up as a change), and
   writes the merged settings to `in/settings.json` — a level laid over the
   defaults first, picked by the role's own setting, else the project's
   `autonomy`, else the role's default; `by-level` and `level-from` beside
   them ([role-adapting.md, "Levels"](role-adapting.md#settings)).
2. **Prepare — `pre`.** Gathers what the role needs into `in/`. If a decision
   needs judgement, it writes the question to `in/task.md`. No `task.md`, no AI
   call: the AI is paid for decisions, not for routine. It may also write
   `in/fallback.yaml`: the proposals to use when no agent answers — the
   documentalist's derived blocks, for example; one marked `if-answered:
   true` is written only when the agent's answer was read (the product
   owner's record that it read an issue). And `in/task-kind`: one word
   naming the kind of question, when `model.tasks` asks something else of it.
   And `in/cuts`: what it cut of the task past one of its caps, one a line
   — what, from what size to what cap, the setting that raises it
   (`verdict.SayCut`); the engine says each as a warning of the run,
   `input-cut`, so the person reads it too, not the agent alone
   ([#1159](https://github.com/JN0V/workline/issues/1159)).
   The engine then records a digest of `in/`.

   **In parts** (ADR-0009). A question too large for one call is written as
   several instead: `in/parts/<name>/task.md`, each whole in itself, and no
   `in/task.md`; or several lenses on one change, each a question of its
   own (ADR-0020), or all of them in one part (`lenses-together`, #147).
   The engine asks each part in a context of its own, one after the other,
   on the tier `model.tasks.part` names, each call counted and kept like
   any other, named by its part (`for: <name>` in `out/calls.jsonl`); a part answers only what the role's
   `part-intentions` name — `claim`s when it names none, a reviewer's lens
   `finding`s, the inspector `triage`s — which are never applied: they inform. The answers are put
   back as `in/parts/<name>/answer.yaml`, and `pre` runs again, with
   `WORKLINE_PARTS=answered`, to put them together — no AI — and write the
   one question that follows, `in/task.md`, or none. Only then is the digest
   recorded. A part that fails, or answers what it may not, is said
   (`part-unanswered`), never read as an answer, and why is written beside
   its task, `in/parts/<name>/unanswered`, its first word the kind of
   failure (`unavailable`, `spent`, `invalid`, `refused`); a lens proposing
   anything but findings is refused besides (`intention-refused`). An answer
   that reads not at all is asked for again once, as the main question's
   (step 3), when the role asks again at all (`promote-after`): the call
   counted like any other, said (`part-asked-again`), the answer not read
   kept as `out/unread-answer.txt`; still unreadable, the part fails. One
   broken in places is read claim by claim (`part-partly-read`), an item
   starting only at a line `- claim:`, so a pasted line starting with `- `
   stays in the claim it is in; without an
   agent, no part is asked, and `pre` runs again with none answered.

   **Questions for a judge** (ADR-0020). `pre` may then write questions
   only a judge answers, one each: `in/judge/<key>/question.yaml`,
   `{question, material}`, and `author`, the model whose work is judged
   when an earlier run's agent did it (an issue announced obsolete,
   ADR-0024). The engine asks each apart, at the best independence
   available from that author, else from the agents that answered the run
   (as in step 4), writes its answer beside it — `answer.yaml`, `{yes, why, model,
   judge, author, level}`, or `{error}`; the call kept with the run's,
   `out/calls.jsonl`, named by its question (`for: judge/<key>`) — and runs `pre` again with
   `WORKLINE_JUDGED=answered`, to read them. A no refuses nothing there:
   `pre` reads the answers one by one (the reviewer drops a finding judged
   no; the product owner keeps an announced issue open). Without an agent, or once `ai-max-tokens` is spent, none is asked;
   the agent unreachable, the rest are not either.

   **A judge that reads** (ADR-0045). A question saying `reads: true`
   lets its judge answer `read` items instead of a verdict (a file's
   `path` and `lines`, a definition's `name`, a word to `grep`), `unsure`,
   or a no with the `quote` it rests on; its answer then holds `reads`,
   `unsure: true`, or `yes` and `quote`. `pre`, run again, may serve
   those reads and ask the judge again: it rewrites the question and takes
   its answer away (`answer-<n>.yaml` keeps it). The engine asks again,
   apart, every question whose answer `pre` took away, then runs `pre`
   again: 10 rounds at most, past which such a question is answered
   `{error: not asked}` for `pre` to settle; every judge answers once
   before any is asked again. The reviewer's code review is the one role that reads so
   ([reviewer](../../roles/reviewer/docs/judge.md#when-the-judge-reads)).
3. **Propose — agent.** Only if `in/task.md` exists and `--ai` is not `none`.
   The agent runs on the model the grid resolves for the role's `model` needs
   (`model-grid.md`), receives the facets and `in/task.md`. It runs read-only
   and writes one file: `out/intentions.yaml`. The agent's proposals replace
   the fallback ones of the same kind — for patches, only those of the files
   the agent's patches touch; fallback proposals of other kinds stay. A
   fallback patch given as a diff (the documentalist's derived blocks, with no
   line of context) is never replaced: it is applied after the agent's.
   With no agent, or none that answered, the fallback proposals are used alone.
   An answer that cannot be read as proposals (`agent-invalid-output`, with
   what the YAML reader said) is asked for again once, on the same tier, with
   that error, when `post` did not block, or blocked for a role that never
   asks a refused answer again (`promote-after: 0`); a final block, never:
   nothing of it was judged, and a quote left unescaped is a slip, not a
   refusal. The documentalist's `post` passed with no proposal, and such a
   doc was judged again, whole, the next night (DomoticsCore, ADR-0014
   step 4); the tester's blocked, no test written, and no second call was
   made ([#761](https://github.com/JN0V/workline/issues/761)). The answer
   not read is kept as `out/unread-answer.txt`; `post` is told why it did
   not read, `WORKLINE_AGENT_UNREAD`; why the engine refused the answer's
   proposals whole, `WORKLINE_AGENT_REFUSED`; why the agent was not asked,
   its prompt over the role's context budget, `WORKLINE_AGENT_OVER_BUDGET`
   ([#1017](https://github.com/JN0V/workline/issues/1017)); else, the
   agent asked and its answer read, `WORKLINE_AGENT_ANSWERED=true`: what
   it holds, or lacks, is the agent's word.
   A block whose verdict says `again: true`, one the answer can mend
   (the tester's tests that do not pass the project's lint,
   [#964](https://github.com/JN0V/workline/issues/964), or do not build,
   [#1048](https://github.com/JN0V/workline/issues/1048)), is asked again
   the same way, for a role with `promote-after: 0`, with its findings:
   the one more call an unread answer gets, never a second one. The
   answer refused is kept as `out/refused-1.yaml`.
   An item holding several keys where one kind goes, or none, is refused
   so too, the keys named: several kinds, an item each; a key no kind is
   named by, the role's kind whose shape holds it, under which it goes (a
   lens wrote a finding's `decision` and `cause` as an item of their own,
   #493). Beside a diff under `patch`, a `file` or `path` naming the file
   the diff changes, and a `content` that is the text the diff gives its
   new file, say nothing more: they are left out, said (`answer-mended`),
   and the item read; one that says otherwise is refused, named (#1148).
   Prose before the list, no fence around it (a line announcing the
   answer), is left out, said, and the list read from its first item.
   The output contract asks for text holding code, quotes, a backtick, `: `
   or ` #` as a YAML block scalar (`|`), which takes code as it is; the
   request again repeats it (#138). YAML, not JSON: code wrapped in JSON
   comes out worse (docs/research/portability.md).
   Before any of that, the engine mends three slips agents keep making, in
   the one place every role's answers are read (`agent.Mend`), and says
   what it mended (`answer-mended`, the answer as it came kept in
   `out/agent-answer.txt`): code quoted with its tabs under a block
   scalar — its first line starting with a tab, or lines pasted at the
   start of the line — which the YAML reader refuses as indentation, is
   indented with spaces, and given an indentation indicator (`|2`) when
   its first line starts with a tab, the text of the block kept as
   written (PR #157); this only when the answer does not read for that
   reason, and never tabs indenting the YAML itself. A value opening on a
   quoted phrase and going on after it (`title: "60 minutes" is unclear`),
   which the reader refuses, is read whole, its quotes kept, when the
   answer does not read for that reason (#128), never a line of a block
   scalar; one closing on the quote it opened with, its inner quotes
   unescaped, is asked again. An answer holding both slips has each
   mended in turn. A plain value
   followed on its line by ` #…`, which the reader would drop as a
   comment, is read whole, as written, when it is free text — several
   words, or a `#` stuck to what follows (`#20`) — while a single word
   annotated `# a note` (a severity, a path) keeps only its value (#156).
   Read whole rather than asked again: the text on the line is exactly
   what the agent wrote, a quote included, and asking costs a call that
   may repeat the slip.
   *Claude Code runs without tools, save in a worktree a role's `pre`
   made (the developer's, [ADR-0056](../adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md)):
   there it reads, edits and runs commands, and `post` bounds its diff;
   and save in a read tree (`readtree`, the tester's under `claude`,
   [ADR-0063 §8](../adr/0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them)):
   there it reads and searches only (Read, Glob, Grep), inside the tree alone, each file it read
   logged with its call. A `cmd:` agent never sees a read tree.
   On Linux its shell boxes each command with bubblewrap and socat: either
   missing, or a container that lets no namespace be made, the agent is
   unavailable, the finding naming what to install, rather than an agent
   whose commands all fail ([#900](https://github.com/JN0V/workline/issues/900),
   [#906](https://github.com/JN0V/workline/issues/906)).
   It runs apart from a Claude Code session it may be started from: that session's variables are dropped
   ([measured](../research/claude-session-env.md)). A `cmd:` agent is not
   sandboxed, so read-only is its command's promise, not the engine's.*
4. **Judge — `post`.** Reads `in/` and, if present, `out/intentions.yaml`.
   Writes `out/verdict.yaml`. It guards what the AI must not decide (for
   example, the committer's `post` refuses a rewritten message that fails the
   same checks, or carries a secret). A question no check can answer, `post` does not answer: it
   writes it to `out/judge.yaml` (`{question, material}`), and once `post`
   passes, the engine puts it to a judge model at the best independence
   available from the agent (ADR-0005): `WORKLINE_JUDGE` when set, else
   another Claude model (Sonnet for work of Opus, Opus otherwise), else the
   same agent in a context of its own. A no refuses the proposal with its
   reason (`judged-no`), and the agent is asked again; a yes is reported
   (`judged`), with the level and both models. Asked again as often as the
   role allows and still refused, the files the refusals name are taken out
   of the patches, and what is left — some of the agent's own — is judged
   once more, without asking: passed, it is applied, and each file left out
   is a `left-out` finding with the refusal's reason. Still refused on an
   event whose steps all run (`fail-fast: false`, the shipped `schedule`),
   the step passes with nothing of the proposal applied but what `post`
   wrote in `out/on-block.yaml` (step 5), each refusal a
   warning ([ADR-0037, amended](../adr/0037-the-schedule-runs-every-step.md#amendment-2026-10-09-a-refused-proposal-is-said-not-failed)).
   So is it on a merge request for a role whose `merge-request-ai` is
   `ready`, refused once: never asked again, no stronger model (ADR-0051). A role listing `claim`
   among its intentions may give claims beside a patch, saying why it takes
   words out (the documentalist, ADR-0014): `post` reads them; alone, with no
   patch, they are refused, and they are never applied. A `post` that passes
   may narrow the proposals in `out/intentions.yaml` — a place it refuses
   taken out, the rest kept — rather than block for the whole task to be
   asked again (the documentalist, ADR-0014 step 4); what it leaves there is
   what is applied, held to the same catalogue and bounds. Where its pre
   gave the agent a worktree, `post` writes them from its diff, even when
   the agent answered none (ADR-0056); its closing words never fail the
   call there: a YAML list among prose is read alone, and words that do
   not read are kept for `post` to say (#746).

   **Questions `post` asks a judge** ([ADR-0065](../adr/0065-code-only-what-converges-language-knowledge-in-packs.md#1-red-for-the-right-reason-goes-to-judgement)).
   `post` may write questions as `pre` does, under `out/judge/<key>/`:
   `question.yaml`, `reads: true` allowed. The engine asks each with no
   `answer.yaml` yet, as `pre`'s, then runs `post` again with
   `WORKLINE_JUDGED=answered`; `post` may serve reads and ask again, 10
   rounds at most. With no agent, past the rounds, or once
   `ai-max-tokens` is spent, the answer is `{error: not asked …}`, for
   `post` to settle. The verdict read is the last `post`'s. The tester's
   red proof of a broken build is the one `post` that asks so
   ([missing name](../../roles/tester/docs/missing-name.md)).
5. **Apply.** The engine checks that `in/` still matches its digest, then
   validates the intentions against the catalogue, the role's `intentions` list
   and its `duties.writes`. An invalid intention set is refused whole. Valid
   intentions are applied in the catalogue's order — files first, then what
   depends on them, then what only informs — whatever order the agent used,
   and each one is recorded in `out/run.yaml` (`applied:`) as it succeeds.
   A role that keeps a backlog comes here with no intention too, when its
   `pre` did not end the run: each issue's state is read — a person's
   answer, an act undone — and written back (docs/spec/backlog-acts.md).
   A run that blocks applies nothing, but the kinds its role names in
   `on-block`, of its intentions: what only says why it blocks. Every
   proposal of those kinds is applied, the agent's or the role's own, on
   any event. The reviewer's (`comment`, `issue`): its summary comment,
   the record in it, its issues
   ([#226](https://github.com/JN0V/workline/issues/226)). The
   documentalist's (`comment`): each of its comments, as the docs to check on a merge request run without an agent, when a
   derived block's command fails or it cannot tell whether a source
   changed. The run still blocks; judged with `--no-apply`, they wait in
   `pending` like any other, for the applying job, which runs whatever
   the judge's outcome.
   A proposal `pre` made itself (`in/fallback.yaml`) and marked
   `on-block: true` is applied on a block too, whatever `on-block` names:
   a refusal written on its issue
   ([ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)).
   The agent's never is, marked or not. So is what a blocking `post`
   writes in `out/on-block.yaml`, removed before `post` runs, held to
   the role's catalogue and bounds: the tester's refusal of a test it
   wrote, on its issue.
   Applied by a job of the base once the merge request's branch moved on
   from the head judged (`--from-merge-request`), a run applies only the
   kinds its role names in `on-moved-on`, said (`moved-on`); none, nothing
   ([ADR-0044, amended](../adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md#amendment-2026-10-15-the-review-of-a-head-the-branch-moved-on-from)).
6. **Again.** If `pre` took less than there was to do, it writes `in/more`:
   the findings it defers, one `<rule> <where>` a line. When the run passed
   and applied something other than a note, the engine then runs the role
   again, from step 2, in a new run folder that sees what was applied — five
   rounds at most. A `held` round goes on too, when it deferred something
   and applied it in the clone, with no merge request. The result adds up
   the rounds' calls and applied intentions; a later round's finding
   replaces an earlier one of the same rule and place, and a deferred one
   is dropped once the next round runs. A run judged with `--no-apply`
   applies nothing a next round could see: it goes round again only when,
   with `--open-merge-request`, the round proposed a merge request for a
   task no earlier round of the run proposed (the key of
   `out/merge-request.yaml`). The next round is told that task waits, as
   one open on the forge, and takes up what was deferred — on a gardening
   night, the docs judged in parts after those judged whole (ADR-0013,
   amended). Each round is a run folder to apply, listed in the result's
   `pending`, a merge request each.

   **Then.** A round that passed may name in `in/then` a task asked once
   the others are done: one round more, `WORKLINE_THEN` naming it, within
   what the run may still spend ([ADR-0048](../adr/0048-the-agent-reads-the-pages-the-reader-checks-flag.md)).

   **A cap on tokens.** A role setting `ai-max-tokens` caps what one run may
   spend, every round, part and retry together: the fresh tokens, as the
   agents report them: the input not read from cache, cache writes
   included, and the output. Cache reads are kept apart in
   `out/calls.jsonl` (`tokens-cached`), never counted: an agent that loops
   reads its context back each turn, at a tenth of the price
   ([ADR-0056](../adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md#caps)).
   It is checked before each
   call against what was spent, never estimated, so the call that crosses it
   is paid; the agent is asked nothing more, and the run says so
   (`ai-max-tokens`, a warning): what is left waits for the next run. 0 or
   unset, no cap.

   **A cap on the wait.** A role setting `ai-max-seconds` caps how long one
   run waits on its agents, from its first call: every round, part, retry
   and judge together ([ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md)).
   Each call may take what is left at most, its role's `model.timeout` when
   shorter; the answer that crosses it is not waited for, a child of the
   agent's still holding its output given 1 s more. Nothing more is asked,
   and the run says so (`ai-max-seconds`, a warning); the role's post
   judges as if the agent had not answered. 0 or unset, no cap; the
   committer's is 30. An agent that works in a worktree is killed with
   its process group, and the call is marked `time-cut` in
   `out/calls.jsonl`, for post to read.

   **A cap on turns.** A role setting `ai-max-turns` is told to an agent
   that loops: `claude`'s `--max-turns`, a command's `WORKLINE_MAX_TURNS`.
   It holds one call. The turns it reports land in the call's `turns`;
   post judges each call's, and the record keeps the run's sum
   ([ADR-0056](../adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md#caps)).
   0 or unset, none told. Reached in a worktree, its edits are its
   answer; out of one, a call that stopped before it wrote its answer is
   an answer unread, its cap said, asked again once
   ([#1194](https://github.com/JN0V/workline/issues/1194)).
