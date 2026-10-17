# ADR-0013: What was judged is not asked again before something changes

- **Status:** accepted; amended 2026-10-02 (a task proposed earlier in the
  run waits too); amended 2026-10-12 (the reviewer knows a change rebased,
  and what a merge of the base settled; a doc's `judged` says what was
  read); amended 2026-10-14 (a doc waits while a merge request holds it);
  amended 2026-10-17 (on a person's merge request, the record waits in a
  comment)
- **Date:** 2026-10-01
- **Amends:** ADR-0012 (a fix that cannot vouch is kept), ADR-0006 (one
  merge request per task)

## Context

DomoticsCore's first gardening with parts on (2026-10-01, roles/documentalist/docs/tried.md)
judged one doc: Opus fixed a version five times and left `checked`, as
ADR-0012 asks of a fix it cannot vouch for. Run again before the person
merged it, the line put the same doc first, judged it again — 26k tokens —
and would have rewritten its pull request to the same end; merged, the doc
was still suspect, and first again the night after. Thirty-four docs waited
to be judged in parts behind it: a run judging docs whole asks no parts, and
a CI run has one round. ADR-0012 said a doc judged once is not judged again
until its sources change; nothing made it so.

Others (docs/research/ci-and-forge.md, "Caps and caches"): Renovate waits
while its pull requests are open; CodeRabbit reviews only what changed since
its last review; agentics' wiki writer keeps the hashes of what it read, but
lets the LLM compute them, and leaves the model out of the key.

## Decision

- **`judged: <commit>`**, in the doc's header, records the commit a doc was
  judged whole at without being vouched for. The agent's patch sets it when
  it leaves `checked`; the judge refuses such a patch without it
  (`judged-not-set`). Like `judged-in-parts` (ADR-0009), the doc is not put
  before an agent again until one of its sources changes after that commit,
  and stays suspect, for a person. A doc whose sources do not tell at all
  gets a patch setting only `judged`, and a note.
- **A task waits for its merge request.** When gardening, the docs a task
  would judge are not put before an agent while that task's merge request
  is open (`workline/documentalist/suspect`, `stale`; `fix` for a doc
  judged in parts). The engine gives the role the open tasks
  (`WORKLINE_OPEN_MERGE_REQUEST_TASKS`). So a doc judged in parts no longer
  waits behind the docs judged whole: their pull request waiting, the
  parts go.
- **A run may cap its tokens** (`ai-max-tokens`, docs/spec/role-contract.md):
  checked against what was spent, never estimated.

The record is the doc's header, in the repository, not a cache beside it:
it holds on the merge request, at night and on a laptop, and a person sees
it in review.

## Consequences

- Gardening goes as fast as its pull requests are reviewed, and spends
  nothing while they wait.
- A doc judged without being vouched for waits for a person, as one judged
  in parts does; one more header line.
- An agent forgetting `judged` is asked again, with the reason: a call more.
- A note alone, with no patch, records nothing: that doc is asked again.

## Amendment (2026-10-02)

workline's first nightly gardening (roles/documentalist/docs/tried.md,
2026-10-02) judged one doc whole and put thirteen off "to be judged in parts
in a later round": a CI run judged with `--no-apply` has one round, and a
round judging docs whole asks no parts. The waiting above lets the parts go
only on a night when the `suspect` merge request is still open; a person
who merges it each morning, while the code keeps making docs suspect, starves
the parts for good (seen on DomoticsCore too).

- **A task proposed earlier in the run waits too.** A run judged with
  `--no-apply --open-merge-request` goes round again when its round
  proposed a merge request for a task no earlier round proposed, and it
  deferred work (`in/more`). The next round is told that task waits, as an
  open one (`WORKLINE_OPEN_MERGE_REQUEST_TASKS`, and alone in
  `WORKLINE_PROPOSED_TASKS`), and counts it against
  `max-open-merge-requests`. Each round is a run folder to apply, a merge
  request of its own (ADR-0006). A doc put off to parts is so judged the
  same night as the docs judged whole, unless the night's caps say
  otherwise — `max-open-merge-requests`, `ai-max-tokens`,
  `parts-max-per-run` — each of which says what waits.

Chosen over alternating whole and parts across nights: alternating needs a
memory of the last night's task, which the repository does not hold — and
ADR-0013 keeps the record in the repository, not beside it; it would also
leave a doc judged whole waiting a night it need not. Going round is what
the engine already does on a machine that applies (the merge request opened,
the next round sees it waiting); the change makes the CI run, which applies
in another job, do the same. Its cost is a second task a night, within the
caps above.

## Amendment (2026-10-12)

The reviewer's record (roles/reviewer/docs/run.md, step 3) kept the
commits a review answered by their ids. A rebase gives a commit a new id,
not a new change: on [#285](https://github.com/JN0V/workline/pull/285),
ten commits rebased onto a newer `main` were all reviewed again, 103k
tokens in, no code changed
([#286](https://github.com/JN0V/workline/issues/286)).

- **The record holds each change's patch id** beside the commit ids
  (`patches=`): git's own id of a change, the same whatever commit it sits
  on (`git patch-id --verbatim`: the stable algorithm, whitespace kept,
  since an indent can be code), read from `git diff-tree`, which no diff
  setting of the machine alters.
- **A commit is reviewed** when its id is recorded, and with it every
  commit it holds; or when its change is recorded and its parents in the
  range are reviewed. A commit a conflict altered is reviewed again, and
  those after it; those before it are not. A merge has no patch id: its id
  alone tells.
- **A record written before still reads**: no `patches=`, the ids alone. A
  record with no patch id is written as before.
- **Said when git cannot**: git before 2.39 gives no `--verbatim` patch id;
  the run warns (`patch-ids-unread`) and a rebased commit is reviewed
  again, as before.

A merge of the base had the same flaw
([#134](https://github.com/JN0V/workline/issues/134)): the lens was given
`git diff <newest reviewed> <head>`, which holds the code the merge brought
from `main`, reviewed there (roles/reviewer/docs/tried.md, workline's pull
request [#132](https://github.com/JN0V/workline/pull/132)).

- **A merge's own change is what it settled.** When the commits after the
  newest reviewed merged the base in, a lens's diff starts from that
  commit with the base merged in by git, in memory (`git merge-tree
  --write-tree`): the tree a merge settling nothing by hand would have
  made, a conflict's markers left in it. Diffed against the head, it shows
  the branch's own commits and how a conflict was settled, both sides of
  one settled wrong; none of the base's code. Nothing left: no lens is
  asked, the merge recorded. The base's commit messages are not given as
  the author's either.
- **Chosen over `git show --remerge-diff`**, which shows the same for one
  merge commit: a push often holds commits besides the merge, and a lens
  reads one diff from where the review stopped, each file whole with the
  change marked. A tree to start from serves all three, as a commit does.
- **Said when git cannot**: before git 2.38, no merge in memory; the run
  warns (`base-not-set-apart`) and the lens is given the base's code too,
  as before.

Measured on copies of workline, with a fake agent
(roles/reviewer/docs/tried.md, 2026-10-12): the three commits up to
[#304](https://github.com/JN0V/workline/pull/304) rebased,
74k bytes put to a lens before, nothing after; `main` merged in with no
conflict, 74k before, nothing after; a conflict settled wrong, 73k
before, 5.7k after, the line dropped shown.

## Amendment (2026-10-12): what a doc's `judged` vouches for

About a doc's `judged` header only; the product owner's own `judged`, on
an issue, is unchanged.

DomoticsCore's gardening judged a doc whose sources were too large to go
whole ([ADR-0014](0014-checked-is-earned-by-what-was-read.md)): the agent
read only what changed in them, found nothing wrong there, and the run said
"nothing found wrong, `judged` recorded" of the doc
([#120](https://github.com/JN0V/workline/issues/120)). Five claims outside
every diff were false. `judged` keeps such a doc from an agent until a
source changes, and then only the new diff is read: those claims are never
read against the code, while the finding and the pull request read as all
clear.

- **`judged` records when, not what was read.** "Judged whole" above means
  judged as one doc, not in parts; it does not mean every source was read.
- **A doc not given every source whole says what was read.** Its finding,
  the run's summary and the gardening merge request name what the task
  gave — what changed in a source since `checked`, the doc it follows — and
  mark the rest "not read". "Nothing found wrong" is said of what was read
  ("nothing found wrong in what was read"), never of the rest. `pre`
  records it in `in/read-not-whole.yaml`, beside `in/read-whole.yaml`.

Not chosen, for now: the agent reading the doc's claims against the code
as it is now, not only the diff, before saying nothing is wrong. It would
vouch for more, but would cost a read of each such doc against its sources
every night one changes. Left as a follow-up; the doc still waits for a
person, as before.

## Amendment (2026-10-14): a doc waits while a merge request holds it

DomoticsCore, 2026-10-05 to 10-08
([#356](https://github.com/JN0V/workline/issues/356)): two docs judged in
parts, the fix changing nothing; their `judged-in-parts` record went to the
`derived` pull request, left open. A doc judged in parts waited only for
the `fix` task: both were judged again each night, two rounds a night,
0.98M tokens in, while 27 other docs never got their turn. A copy of
DomoticsCore on `main`'s engine, with a fake agent, did the same.

On workline, `docs/spec/conformance.md` was condensed 11 times while its
pull request ([#269](https://github.com/JN0V/workline/pull/269)) waited:
no rewrite waited for its own merge request, and neither
[#312](https://github.com/JN0V/workline/pull/312) nor
[#315](https://github.com/JN0V/workline/pull/315) changed that.

- **The engine says what waits, file by file.** With
  `--open-merge-request`, `in/waiting.yaml` names, for each task waiting,
  the files its change touches: an open merge request's branch from where
  it left the commit judged, fetched as `workline follow` does; an earlier
  round's patches. A branch that cannot be read is said
  (`merge-request-unread`), its task known by name only.
- **A doc judged in parts waits while a merge request changes it**, its
  fix or only its record, or while an earlier round proposed it; the docs
  none holds go. A `fix` task whose files are unknown holds them all.
- **A run judging docs in parts opens a merge request of its own**, keyed
  by its first doc (`workline/documentalist/fix-<doc>`): never the derived
  blocks', never one the next run's proposal would replace.
- **A rewrite waits for its own merge request**: a doc to condense or
  split, a card to merge, a repeated passage; the next one goes, said.
- **Docs judged whole wait as before**, for `suspect` or `stale`: one
  merge request they share, which another doc's judgement would replace.

Not chosen:

- **Reading the record on the branch**, not which files change: exact for
  a doc whose derived block alone waits, but a round proposed and not
  applied has no branch. Such a doc waits for `derived` too, said.
- **One merge request per doc judged in parts**: one fix call per doc,
  fewer docs a night.

## Amendment (2026-10-17): on a person's merge request, the record waits in a comment

From 2026-10-04 to 2026-10-17, the documentalist pushed 42 commits to 20
people's pull requests on workline: 41 only moved a record line
(`judged`, `verified` or `judged-in-parts`), one changed a doc's text
([#463](https://github.com/JN0V/workline/issues/463)). Each was a new
head: the checks ran again (`test`, 2 min 46 s on
[#451](https://github.com/JN0V/workline/pull/451)), and the person pulled
before pushing again: the record, the doc's header (above), took a commit.

- **A doc found with nothing to fix gets no commit** on a person's merge
  request: a patch changing only those lines is taken out of what is
  committed.
- **Its record is kept in the documentalist's comment** on the merge
  request, hidden, as the reviewer keeps its own: the commit judged at,
  and the git object id of the doc and of each of its sources there (a
  blob, or a tree for a folder). Only a comment a bot or a person of the
  project wrote is read: on a public merge request, anyone may comment.
- **The next push reads it**: a doc whose ids are all the same at the new
  head is not put before an agent again; one whose doc or source changed
  is judged again.
- **A real fix is still one commit**, its message naming the doc. A move of
  `checked` is still a commit too: what earns it does not change
  ([ADR-0014](0014-checked-is-earned-by-what-was-read.md)).
- **After the merge, gardening writes `judged` on the default branch**,
  naming the commit the merge request became, when the doc and its sources
  are still as judged
  ([ADR-0010, amended](0010-docs-judged-on-the-merge-request-and-by-gardening.md#amendment-2026-10-17-the-commit-a-merge-request-became-stands-for-its-record)).
- **A fork's merge request**, which gets no push, gets its record too.
- **Where the comment cannot serve, the record is committed, as before**:
  a forge that does not say which commit a merge request became (`local`,
  `cmd:`), a comment that could not be read (`record-unread`), a doc with
  a source in another repository.

The record is still the doc's header once merged, in the repository. While
the merge request is open it is a comment, which anyone with write access
may edit, as the reviewer's.

Not chosen ([the options](https://github.com/JN0V/workline/issues/463#issuecomment-6239964209),
[the decision](https://github.com/JN0V/workline/issues/463#issuecomment-6240008472)):

- **One record commit, at the yes**: a merge request merged with the
  forge's button and no label gets none.
- **Committing only real fixes, judging again after the merge**: each doc
  judged twice for one change.
- **Keeping it as it was**: 41 commits in two weeks that change no text.
