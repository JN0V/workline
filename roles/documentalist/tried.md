# Documentalist — tried for real

What each try on a real repository or with a real agent showed, newest last.
The role's README says what the role does.

On 2026-09-24, with Claude (sonnet, then opus when asked again), on copies of
the `documented` fixture:

- tokens moved from one hour to two, three runs: the agent's answer was right
  each time, in one call; the first run lost the fix to the defect below, the
  two after it applied it;
- a function added, the doc still true: the first answer was refused for
  growing the doc's body; the second, one tier up, moved `checked` and nothing
  else, and opened an issue for a comment the new code contradicts;
- two defects found this way, now guarded: a hunk header counting fewer lines
  than it holds made git drop the fix itself (hence `--recount`), and a commit
  like `11180e1` was read by YAML as a number, so the doc passed for unchecked.

On workline's own repository, without AI: no false positive; one real finding
(docs/spec/role-contract.md is over its line budget).

On 2026-09-24, before a push of workline itself (a copy, pushed to a local
remote): the line found three docs whose code had changed since they were
checked — two of them real, left behind by the day's own commits. Claude judged
all three still true and said why, line by line. Its first patches set
`checked` to the commit that had changed each source, not to the one the task
gave: the task now says "your patch sets `checked: …`", and a refusal repeats
the commit. The next run was right at the first attempt; the push stopped for
review, the docs were committed, and the push went through.

On 2026-09-24, gardening workline itself (a copy): the documentalist chose
docs/spec/role-contract.md, 330 lines for a budget of 200. Three defects came
out before it worked, each now guarded: the agent's time was capped at three
minutes (a role now sets `model.timeout`); an answer holding a code block of
its own was cut at that block (the answer is now read whole first); and a new
file after another in one diff, without `diff --git` lines, was read by
`git apply --recount` as lines of the first (those lines are now added). Then
Claude moved 142 lines, unchanged, into role-outcome.md and role-adapting.md,
each tracking the contract's sources, linked both ways — at the second
attempt, the first citing a wrong line. That split is the one in this
repository.

On 2026-09-27, on workline's own repository:

- links to other sites, with lychee: 75 checked, none broken; a 404 added on a
  copy was reported at its line; behind a proxy that does not answer, each doc
  got a count of links not reached, and no link was called broken;
- a superseded decision, on a copy: the one doc citing it without its
  successor was reported, not the two naming both;
- stale docs, on the `documented` fixture with Claude (Sonnet): a doc edited
  into a claim its source does not back was confirmed once in four runs; the
  policy now says moving `checked` vouches for every sentence kept, and a case
  guards it (tests/evaluation);
- a spec the code disagrees with (`truth.doc`): the doc kept as it was, and an
  issue quoting both sides, in three runs out of three;
- the checklist, on a throwaway pull request of this repository on GitHub
  (#2): created, then edited in place on the next run, then emptied once the
  doc's `checked` moved; one comment throughout.
- a repeated passage, on the `untidy` fixture with Claude (Sonnet): kept in
  the sessions doc, the FAQ linking to its heading, at the first attempt, and
  in three evaluation runs out of three.
- a card too short, on the `documented` fixture with two cards, with Claude
  (Sonnet): merged into the other card, deleted, its link pointed there, at
  the first attempt — under a second top-level title, which the task now
  forbids; then three evaluation runs out of three.
- a merge request per gardening task, with Claude, a local bare remote and
  the simulated forge: the first attempt was refused by the machine's own
  commit hook, the fixture's identity not being allowed, and left the patch
  staged — it is now left unstaged, as the patches wrote it; with an allowed
  identity, the branch was pushed with one commit, the merge request opened,
  the tree back on main and clean, and a second run updated the same merge
  request.
- the same on GitHub (2026-09-28), from a clone of this repository, without
  AI, a derived block made stale: pull request #3 opened from
  `workline/documentalist/derived`, one commit, the README's count; a second
  run updated it; with `max-open-merge-requests: 1`, the next run paused —
  but still regenerated the block, which could have opened one merge request
  too many: a paused run now proposes no patch at all.
- splitting a card, on the `documented` fixture, Opus splitting and Sonnet
  judging: right at the first attempt by hand; measured, the judge said no
  twice in two runs, taking the sentence on how long a token lasts, in the
  sessions card, for a second concept, and Opus's second try was then refused
  for rewording a line. The question now says a sentence explaining the
  concept by what it depends on stays within it: three runs out of three
  since, the judge naming why each card holds one concept.
- condensing on Sonnet, four more runs: three right, one moving nothing;
  with the two of 2026-09-25, four in six, writing more tokens than Opus,
  which is right ten times in ten: condensing stays on `frontier`.
- a doc fixed on a merge request, on GitHub (2026-09-28): a throwaway pull
  request (#4) added a conformance case, so the README's count was stale; run
  as the CI's apply job would, on the detached merge checkout, the
  documentalist committed the count to the pull request's own branch, on top
  of the author's commit, and left the tree clean.
- propagating, on the `documented` fixture with the technical doc saying two
  hours: pending on a merge request, blocking the release without an agent,
  and at the release, with Claude (Sonnet), the product doc brought to two
  hours and checked again, at the first attempt and in two evaluation runs
  out of two.
- adopting a repository (`workline init`, 2026-09-28), with Claude, on a copy
  of this repository (18 docs saying nothing of their sources) and of a small
  firmware project (12 docs, none declaring any): each doc got its sources, or
  `sources: []` for a decision, a changelog, a research card, in three or four
  rounds of 22 to 38k tokens in. Sonnet's answer was refused in six rounds of
  seven, then Opus's taken: folders written `roles/`, with the slash, taken
  for missing; a blank line between the header and the title, taken for a
  change of the doc (three rounds); a diff git could not apply (two). The
  first two are now taken, and guarded. And a second note of one answer
  overwrote the first, and no note reached the person — yet the notes had
  found a doc behind the changelog, and one possibly replaced by another:
  every note is now in the result, and shown. Sonnet alone, on this
  repository, after the slash was taken: right at each first attempt.
- adopting DomoticsCore (2026-09-28), 64 docs: the run failed before any
  call, the task over the role's budget — its changelog, 134 kB, was put
  before the agent whole. A doc now shows its first lines only, enough to
  tell what it describes.
  Run again: 14 docs adopted in five rounds, then the rounds ran out; three
  answers were refused for a hunk numbered from line 0 (`@@ -0,0 +1,4 @@`,
  the doc's first line as context), which git reads from line 1 and the
  judge took for a line that does not exist: now read as git does.
  After two more runs, 63 of 64: the last, cut to its first lines, was left
  by the agent, which said, rightly, that the sections cut were unknown to
  it. A doc cut now lists the headings of what is not shown.
- catching DomoticsCore up (2026-09-29), its 53 suspect docs judged in one
  go: the first call failed before a token was spent, its first doc, with
  what changed in its folder sources, over the role's budget. A doc too large
  for a task alone is now left for a person, saying so, and the others are
  judged; cutting it would have its `checked` vouch for lines never read.
  Stopped after four runs, about a million tokens in, 46 docs still to
  judge. Their sources named whole folders; narrowed to the files each doc
  names (2663 files behind them down to 911), they cost more, not less: the
  diff shown was capped at 80 lines a source. It is now 240 lines a doc:
  estimated 517k tokens to 338k for the 42 left, and 12 docs too large for a
  task down to 2.
  Run again, on Sonnet 5.5 once the alias moved: five calls on one doc, the
  agent rightly refusing to vouch for it on diffs cut short, and the engine
  taking its note for progress. A round leaving only notes now ends the run;
  a doc whose changes do not fit is judged against its sources as they are
  now, like a stale one — 26 of the 42 left had more changes than a task
  shows, up to 17,099 lines of diff.
  Then one doc blocked the run: both Sonnet 5.5 and Opus wrote its patch as
  two hunks overlapping on a line of context, which git refuses, and git's
  message taught them nothing. Such hunks are now merged before judging.
  Where it stopped: 14 docs fixed, for a person to review; 40 left. 30 of
  them are far behind sources that do not fit a task even as they are now
  (large headers). The other 10 are refused by the agent itself: it finds
  what is wrong — a version 1.4.1 where the code says 1.11.0 — but cannot
  vouch for the rest from diffs, and a fix that leaves `checked` alone is
  refused; the doc it noted comes first again on the next run. About 1.3
  million tokens in, over the whole catch-up.
- a push on DomoticsCore (2026-09-29), three commits, no agent: nearly three
  minutes before any word, then the question; a person who had gone away
  found the push refused. 3,247 git calls — every commit of every source of
  every suspect doc, those left for gardening included — and six searches of
  the whole code for the names the docs cite, at each doc's last edit, up to
  40 seconds each. Each commit is now read once, a doc left for gardening is
  not read further once found suspect, and a push searches only the names its
  commits removed: six seconds, the same findings. The hook now says it runs;
  and the ten minutes to answer counted from the first question, so judging
  the docs with `d` could use them up: each question now has its own.
- a doc fixed on a merge request, then merged (2026-09-30, the `documented`
  fixture, by hand as CI would): merged with a merge commit, the doc stayed
  judged; squashed, it came back suspect, and once the branch was deleted
  and collected the documentalist blocked, its `checked` naming a commit
  that no longer existed; rebased, suspect again. A `checked` main does not
  hold now stands for the commit that brought it there: the squash, or the
  rebased commit. DomoticsCore's push check went from 6.1 to 7.4 seconds.
- a doc fixed on a pull request by CI (2026-09-30, workline, throwaway PR
  #10): `max-handoffs` raised to 5 in routing.default.yaml, the routing spec
  saying 3. Three attempts. The apply job first failed: the run named the
  roles cache of the judge's runner, gone on its own; then installed an
  engine the Go proxy remembered from before the fix. Both fixed. Then
  Claude (Sonnet 5.5, 4,651 tokens in, 1,138 out) brought the spec to 5,
  the GitHub App committed it to the branch with `Workline-Role:`, the
  checks ran again with no one approving them, and the line did not judge
  the App's commit. Two faults seen in that commit: the agent also aligned
  an example of the spec, which shows a project's routing, not the default,
  to the default — off its task; and `checked` named the merge commit
  GitHub builds for a pull request, which no branch holds: the commit that
  brought it stands for it, but the head of the branch should be named.
- judging in parts, measured (2026-09-30; ADR-0009, "Measured"): on a doc
  with four planted defects, Sonnet in parts found all four, three runs in
  three, as whole did, at five times the tokens; Haiku on the parts wrote ten
  times more and missed once. The first runs in parts failed on a part's
  one-line YAML with a brace too many: now read claim by claim.
- a real pull request on DomoticsCore (2026-10-01, #108: an ESP8266
  discovery reboot, MQTT TLS with a CA, a JSON stub): CI judged the docs it
  made suspect against their sources whole (Sonnet 5.5, 11.7k tokens in,
  1.9k out), the App committed the fix to the branch — the MQTT README and
  the HAL architecture doc — and it was merged with the change.
- DomoticsCore's gardening with parts on (2026-10-01): the night before
  ran nothing, the schedule never fired; run by hand instead. 44 docs
  suspect: one judged, 9 left over ai-max-calls or the task's size, 34
  waiting to be judged in parts "in a later round". The one judged was the
  OTA doc refused on 2026-09-29, version 1.4.1 where the code says 1.11.0:
  Sonnet 5.5 cited a line wrongly, the judge refused it, Opus fixed the
  five places, `checked` left (ADR-0012) — pull request #109, 6 lines.
  Two calls, 21.2k tokens in, 5.0k out. No part was asked: a CI run judges
  without applying, so it has one round, and a round judging docs whole
  asks no parts. At one doc a night, the parts wait for the other nine.
  Worse, run again before #109 was merged, the same doc came first: judged
  again, to the same end, and after the merge still suspect — `checked`
  not moved. Now (ADR-0013) a fix not vouched for records `judged`, and a
  task waits while its pull request is open. Checked on a copy, #109 open,
  no agent: the 11 docs judged whole wait for it, and 16 parts are
  prepared — README.md and docs/architecture.md, 8 each; 20 more docs in
  parts wait past `parts-max-per-run`, and 11 go to a person (5 too long
  to be judged in parts, 5 with sources too wide, 1 with no code).
- DomoticsCore, ADR-0013 and the caps on (2026-10-01, two gardening runs
  by hand). The first judged docs whole: two version tables, Sonnet 5.5
  citing a line wrongly again — the second time in two runs — and Opus
  fixing them; both docs record `judged` (pull request #112). 36.6k tokens
  in, 5.6k out. The second, #112 waiting, judged README.md in parts, the
  first real run in parts: 8 parts on Sonnet, 17k to 21k tokens in each,
  and one fix — 165k in, 15k out, nine calls, two minutes. Pull request
  #113: twelve component versions and an LED effect brought to the code,
  all right; one fault, the version badge bumped while its link still
  names the release v2.0.0. `uncovered` named a feature on the roadmap,
  rightly for a person, and `ini` — from `platformio.ini`, not a name of
  the code. Five docs need more than 8 parts (`sources-too-wide`, 10 to 16).
  Sonnet's two refused answers, replayed against the judge at their
  commits: each had one hunk a line off, its quoted lines found once in
  the doc. The judge now places such a hunk (3 lines at most): the Storage
  answer passes as Sonnet wrote it, without Opus; the OTA one, written
  before `judged` existed, is refused for that alone.
  The two small faults of #113 are now cases: a patch bumping a version on
  part of a line is refused, saying where (`replaced-in-part`); a code
  span naming a file (`platformio.ini`) is no longer a name of the code.
- a solo repository without pull requests (2026-10-01, conformance only):
  `workline docs` judged commits already pushed, kept its ref until a
  person committed the fix, then moved it, and said so when nothing was
  new; a release was held by a doc made suspect after the last tag, not by
  one suspect before it nor by one fixed without being vouched for. On a
  copy of DomoticsCore, no agent: `workline docs` judged from v2.11.0, and
  moved its ref though nothing was judged — it now waits for every doc of
  the range to be put before an agent or left to a person; a release there
  would wait for 21 docs, the version bumps of 2.12.0 having touched the
  `library.json` many docs name.
- workline PR #10 replayed (2026-10-01, evaluation `stays-on-the-task`):
  the default number of handoffs raised, the routing spec judged on
  Sonnet 5.5; the default fixed and the example of a project's routing
  left alone, three runs in three, 23k tokens each. The fault of PR #10
  came before ADR-0012 gave the agent its sources whole.
- a fork's pull request (2026-10-01, JN0V/workline-sandbox #1 from the
  fork jn0v-lab/workline-sandbox, the `documented` fixture with the
  templates of the branch): workline judged it with no agent, the fork
  getting no secret, and left its report; workline-fork.yml, run from the
  repository's side, checked the pull request's head and commented it —
  the doc the change made wrong, and the product doc following it. A
  second push edited that comment; none was added.
- GitLab, for real (2026-10-01, gitlab.com JN0V/workline-sandbox !1, the
  same fixture and the branch's template): the first pipelines failed
  before any job, "the user not being verified" — gitlab.com asks a free
  account to verify itself, by phone or card, before its shared runners
  run. Then Claude (Sonnet 5.5, 2.8k tokens in) fixed the doc, the apply
  job committed it to the merge request's branch with the tokens
  unprotected, and the pipeline that commit started judged nothing: "the
  last commit is workline's own". Installing Node and Claude in the
  golang image took 44 seconds. A fork's merge request is untried.
- workline's README judged in parts on pull request #29 (2026-10-01): the
  fix turned "a push approval … if you ask for it" into "unless your own
  config turns it off" — false (ADR-0011). One part, holding the setting,
  said supported; another, holding a comment of cmd/workline/main.go left
  from before ADR-0011, said partial, "the default is not in this share";
  the fix followed the comment. The comment is fixed, the line restored. A
  fix should not act on a partial claim another part supports.
- GitLab reached through its REST API, without glab (2026-10-01, the
  engine on a clone of gitlab.com JN0V/workline-sandbox, no agent): the
  merge request read, the checklist of docs to check posted on !1, then
  edited in place on a second run, not posted again.
- the templates on workline v0.1.0 (2026-10-01, both sandboxes): on
  GitHub, the fork's pull request judged with the engine and gitleaks
  downloaded in 1 s, where installing Go and building took about 40; on
  gitlab.com, the jobs in ghcr.io/jn0v/workline:v0.1.0, judge 28 s with
  Claude fixing the doc, apply 15 s committing it — 112 and 182 s before,
  building glab and workline.
