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
- DomoticsCore's gardening with parts on (2026-10-01), run by hand: the
  night's run, due at 02:00 UTC, had not come. 44 docs
  suspect: one judged, 9 left over ai-max-calls or the task's size, 34
  waiting to be judged in parts "in a later round". The one judged was the
  OTA doc refused on 2026-09-29, version 1.4.1 where the code says 1.11.0:
  Sonnet 5.5 cited a line wrongly, the judge refused it, Opus fixed the
  five places, `checked` left (ADR-0012) — pull request #109, 6 lines.
  Two calls, 21.2k tokens in, 5.0k out. The night's run came after all,
  6 h 17 late (run 36835421241, 08:17 UTC, on 2e25ea0): it judged the same
  doc whole (Sonnet 5.5, 10.6k tokens in, 2.0k out), fixed it, `checked`
  not moved, and force-pushed #109's branch with the same patch, byte for
  byte (0eaa6b5 → 62a822f), merged as 92feb08. A schedule can be hours
  late, not only skipped. No part was asked: a CI run judges
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
- the first tag, moved (2026-10-01): v0.1.0 was pushed before the release
  workflow was on main, then main's history was rewritten to drop 70 MB
  of binaries committed by mistake, and the tag put back. The Go module
  proxy had kept the first v0.1.0 for good: every `go list …@main` with
  GOPROXY=direct — workline's own CI, DomoticsCore's — failed its checksum
  ("does NOT match the one reported by the checksum server"). v0.1.0 is
  retracted in go.mod; v0.1.1 is the first release to use. A tag pushed is
  never moved.
- a solo repository, without pull requests (2026-10-01, a copy of
  WaterMeter — 12 docs, no pull request ever — pushed to a local bare
  remote, Claude Sonnet): `workline init` proposed every doc's sources in
  43 s, from the file names alone, as the task shows no code; the push
  counted the docs in 1.3 s with no agent; `workline docs` judged them
  from the last tag; the release was held by the docs left. Three defects,
  each now a case: the adoption commit, changing only headers, counted the
  docs following those docs as made suspect by the push; a right fix
  (GPIO2 → GPIO34) refused twice, Sonnet then Opus, its last hunk having
  no context after it — git apply takes that for the end of the file,
  where the engine, applying with `--unidiff-zero`, would not — and the
  run stopped there, saying only "2 doc(s) changed"; seven docs too large
  to be judged whole went to a person, the output showing the first line
  of their finding only. 8 agent calls, 217k tokens in all (two runs of
  `workline docs`, the first stopped). Judging in parts is off
  by default: those seven wait for a person.
- the documentalist's fixes reviewed (2026-10-01, workline's and
  DomoticsCore's, by an agent reading each against the code): versions,
  flags and CI permissions right every time, small diffs; wrong when it
  trusts a stale code comment over the record (#29, d38a0e4), removes a
  true claim it cannot see backed (d43b3f2), or moves `checked` over
  frozen numbers still false — line counts, a stale date — on six
  DomoticsCore docs whose sources it saw as diffs only, the judge letting it;
  rewrites a bug fixed in 2025 rather than removing it (92feb08, `checked`
  left). workline's bot never moved `checked`. It fixes one doc and
  leaves the same version stale in its siblings.
- DomoticsCore's docs too wide to be judged in parts (2026-10-01, counted
  with the engine's own planning, no agent): six past 8 parts, not five.
  webui-developer.md 12 → 8 and observing-a-device.md 10 → 8 by narrowing
  their sources; the four others hold more than one doc each, to split.
- ADR-0014's baseline (2026-10-01, Claude Sonnet 5.5 at medium effort,
  the `drifted` cases, five rounds on this engine, d66eeea, and five on
  the one before step 0, 224a0d7 with today's cases): the bar step 4
  compares against. Every run loses the three `count-off` findings, step 2
  not being built, so the pass rate over every point is 0/5 everywhere;
  read per grade, over five runs each:

  | Grade | gardening, before / after | version bump, before / after |
  |---|---|---|
  | never `checked` over a planted falsehood | 5/5 / 5/5 | 3/5 / 2/5 |
  | `status.md` judged, `checked` kept | 5/5 / 5/5 | 5/5 / 4/5 |
  | the clean control ends `checked` | 5/5 / 5/5 | 5/5 / 5/5 |
  | the right fixes (version, tree, sibling; bug removed) | — | 5/5 / 5/5 |
  | what is true stays (default over comment, limits, record) | 5/5 / 5/5 | 5/5 / 5/5 |
  | no `count-off` on a limit | 5/5 / 5/5 | 5/5 / 5/5 |
  | every point but the three `count-off` findings | 5/5 / 5/5 | 3/5 / 2/5 |

  One agent call a run; about 36k tokens in and 6k out for gardening,
  39k and 5k for the bump; 0.85M tokens for the twenty runs. Step 0
  changes nothing here, as the fixture means it to: every source fits
  whole in the task, so no `checked` is refused, and what is vouched for
  is the agent's own doing. Two failures, both on the version bump: the
  README vouched for with "| `Clock.h` | 524 |" still in it (2 runs on
  each engine), the agent counting nothing — on another doc it reasons
  that the diff "replaced one line with one line, so the count is
  unchanged", trusting the doc's count; and `status.md` vouched for with
  "33 tests" from a file no source names (1 run, this engine), with no
  note. Elsewhere the agent says when it cannot count ("about 236 lines,
  not ~283"), leaves `checked` and sets `judged`. Never seen: a comment
  winning over the code (an issue is opened every run), a true claim
  removed, a sibling left at the old version, a fixed bug rewritten.
- ADR-0014 step 2, `count-off` and `value-left` measured with no agent
  (2026-10-01, the engine at 782f65d, on copies of DomoticsCore at
  origin/main 98e016d and of workline at a96306c). Every report read
  against the doc and the file:

  | Check | DomoticsCore | workline |
  |---|---|---|
  | `count-off`, every doc, `workline run-role … --ai none` | 60 reports, 60 real, 0 false | 0 (no doc states a file's line count) |
  | `value-left`, the bot's fixes replayed (bc0bd26 … 5ae8335) | 54 → 51 after tuning: 47 real, 4 false | 0 (no bot fix, nor any of 266 doc commits, replaces a three-part version) |

  `count-off`: the 60 are in 12 docs, each a count stated for a source
  file the file no longer has — fenced listings (ota/README.md, ntp),
  `| File | Lines |` tables (ntp, ota, storage, webui), prose
  ("`EventBus.h` is currently ~283 lines"), a `LOC` count in the deep
  dive (159 when written, 180 now). The three known cases are found:
  EventBus.h ~283 for 462, Platform_Stub.h ~630 for 930, Storage.h 655
  for 771 (three places). No miss within the rule; outside it,
  remote-console/index.md's "RemoteConsole.h (714 lines)" (960; its
  sources are other docs, not the header) and home-assistant's "1030
  raw" (1137; no "lines"). Counts within a tenth of a `~` stay quiet
  (ComponentRegistry.h ~378 for 406), so does every "< 800 lines".
  `value-left`: first built on sets, it missed the badge's link (the
  fix's line still said 2.0.0, so 2.0.0 was not "replaced"); counted
  per line now, it reports the link and "Version 2.0.0 Released!" (5ae8335)
  and MQTT's siblings at 1.9.0 (16c660b). A sibling shares a source
  file that now says the new version, so Core's 1.4.0 is not looked for
  in LED's docs. Seven false alarms first: three a version range
  (`>=1.13.0`, `>=1.4.0`, `>=1.3.0`), tuned out with a case
  (`value-left-not-a-range`); four left — HomeAssistant's "(v2.0.0)"
  marking when a feature came, in three docs, and Wifi's version-history
  row. A real report often lists other lines too: the OTA row's 1.4.1
  beside System's and WiFi's.
  The `drifted` cases with fake agents (no real one): the three
  `count-off` grades pass in every mode; vouching for every doc is now
  refused on the README and project-context (`checked-over-count-off`),
  still losing the "33 tests" no source shows.
- ADR-0014 step 2, the removal rule and "a comment is not evidence"
  replayed with no agent (2026-10-01, the engine at 44c0ce4, on copies of
  workline and DomoticsCore): each reviewed bot fix put through the new
  checks, at its parent commit, as an agent would have proposed it, with
  the claim it could honestly have given — the line it followed, or none
  when nothing in the doc's sources backs it:

  | | workline | DomoticsCore | |
  |---|---|---|---|
  | wrong, refused | 3 of 3 | — | d43b3f2 `removal-uncited` (nothing contradicts "a comment edited in place"); d38a0e4 and 3cd196d `comment-not-evidence` (the template's header, main.go's comment), each comment reported |
  | right, passed | 4 of 5 | 7 of 7 | versions cited from `library.json`; CI permissions, `cmd:`, the gitleaks rule from the code; 972afa9, 2e25ea0 and the LED states take no word out |
  | right, refused | 1 (172493e) | 0 | its only evidence the GitLab template's header comment, the record the wrong d38a0e4 followed |

  172493e only undid d38a0e4: with the rule, d38a0e4 is refused, the README
  never holds its wrong text, and 172493e is never needed; counted anyway,
  as the bar asks. Read strictly, 11 of 12 right fixes pass where 12 of 12
  did. Where a real agent would need to cite and could not: d43b3f2 (no
  source backs the removal), d38a0e4 and 172493e (only a comment says
  what was tried); 92feb08's wrong edit, the fixed bug's version rewritten,
  passes, cited from `library.json`: not what these rules are for. Per
  run of changed lines, one claim suffices: 5ae8335's twelve-row table is
  cited by one row's `library.json`. A claim is checked to exist where it
  says, not to support the change (ADR-0014 step 3): an agent quoting an
  unrelated line of code would pass. Not yet run with a real agent: the
  tokens, and whether agents give claims, are step 4's to measure. The
  `drifted` cases with fake agents (headers only) grade as before.
- ADR-0014 step 3, the gate, with a real agent (2026-10-01, Claude Sonnet
  5.5 at medium effort, the engine at 7427ae8: step 0 and step 2 built).
  **The `drifted` cases, five rounds** against the baseline (d66eeea,
  the same cases, the engine after step 0), read per grade:

  | Grade | gardening, baseline / now | version bump, baseline / now |
  |---|---|---|
  | never `checked` over a planted falsehood | 5/5 / 5/5 | 2/5 / 5/5 |
  | `status.md` judged, `checked` kept | 5/5 / 4/5 | 4/5 / 5/5 |
  | the clean control ends `checked` | 5/5 / 5/5 | 5/5 / 5/5 |
  | the right fixes (version, tree, sibling; bug removed) | — | 5/5 / 5/5 |
  | what is true stays (default over comment, limits, record) | 5/5 / 5/5 | 5/5 / 5/5 |
  | the three `count-off` findings | 0/5 / 5/5 | 0/5 / 5/5 |
  | every point | 0/5 / 4/5 | 0/5 / 5/5 |

  Tokens a run, in and out: gardening 41.9k → 49.2k on average, one run
  asked again (82.5k), the four others 40.6k to 41.4k; the version bump
  43.7k → 44.1k. Nine runs in ten took one call: the claims the removal
  rule needs were given, and held, at the first answer — the bump's fixes
  take out versions, two line counts and the fixed bug every time. No
  `removal-uncited`, `citation-unchecked`, `comment-not-evidence`,
  `checked-over-count-off` nor `checked-unread` in any run. The one miss:
  gardening, round 4, project-context.md and status.md refused twice as
  `still-suspect` ("the patch does not set `checked` … so the doc would
  stay suspect"), so status.md did not record `judged`; the answer is not
  kept, so whether it set `judged` is not known. One more round, its runs kept for reading
  (results in the scratchpad, not in results.tsv): the claims cannot be
  read there either — the engine drops them from `out/intentions.yaml`
  once judged, and the agent's raw answer is not kept.

  **The reviewed bot fixes replayed**, one run each, on scratchpad copies
  at each fix's parent: a workline merge request with its range (the
  pull request's commits before the fix), a DomoticsCore merge request
  likewise (16c660b with the release as last commit, the bot's own being
  skipped), gardening for the night runs (`workline route … --no-apply
  --forge none`, the repository's `.workline/config.yaml` of the time,
  your own config left out). The `docs` setting narrowed to the reviewed
  doc(s), so a run judges that doc only: siblings, and `value-left` across
  them, were not exercised.

  | Commit | Then | Now: right fix? | Known error repeated? | `checked` over a falsehood? | New error |
  |---|---|---|---|---|---|
  | d43b3f2 (README, in 5 parts) | removed a true claim | yes: line 28 kept, its parts finding it backed (one claim quoting a comment) | no | no | — |
  | 3cd196d (README, in 5 parts) | stale comment over the setting; a right fix at line 60 | line 148 kept, cited from the setting's field; line 60 not fixed | no | no | a right fix missed: no part claimed line 60 |
  | 92feb08 (OTA) | versions right, a fixed bug rewritten | versions right, four places, cited | no: the bug line left, with a note; `value-left` reports its old version | no, `judged` | ten line counts the engine gave, not fixed |
  | bc0bd26 (docs/README.md, hal) | version right, `checked` over a stale date | version right | no | no: `judged`; hal-architecture.md too large, left for a person | — |
  | 2e25ea0 (MQTT, hal) | TLS needs a CA; `checked` on hal | the CA said, as a new bullet | no | no: both `judged` | — |
  | 25e22dd (docs/README.md, core) | version right, `checked` over a stale date | version right | no | no, `judged`, the date left for a person | — |
  | 16c660b (core, MQTT) | versions right, `checked` over false counts | versions right | no | no, `judged` | the two counts dropped after a refusal |
  | d784398 (OTA, storage) | storage version right | right, at the second answer | no | no, `judged` | line counts not fixed |

  Not replayed, for tokens: 4ac4f44, 445bed2, 972afa9, d38a0e4 (README in
  five parts, about 200k each), 0f4c01c (two parts), 5ae8335 (eight
  parts); 172493e only undoes d38a0e4, which the rule refuses, and
  fea16c7's commit was rebased away. Asked again twice, in eight runs:
  d784398's first answer gave no claim for the versions it replaced
  (`removal-uncited`); 16c660b's quoted "930 lines" from Platform_Stub.h,
  a line count as if it were the file's words, and the rewording beside
  it ("Watch the 800-line limit" → "It is over …") was refused
  (`citation-unchecked`) — the second answer dropped both counts, the
  EventBus one included, which needed no claim. In all three
  DomoticsCore runs where the engine gave counts, the agent left them,
  saying a count "cannot be quoted as a source": the task says to bring
  each to the count given, not that the engine's count stands as the
  citation. Never seen: a `checked` moved, a true claim removed, a
  comment winning, a fixed bug rewritten. Tokens: about 25k a doc
  judged whole (24.7k to 29.3k a run, 66k to 77k when asked again), about
  200k a README in five parts (192.8k, 206.0k). In all, 1.33M tokens:
  0.47M for the five rounds, 0.22M for the kept rounds (the second one
  stopped), 0.65M for the eight replays.
- ADR-0014 step 4, the acceptance, on the held-out set (2026-10-02,
  Claude Sonnet 5.5 at medium effort, the engine at ffdb06d; graded by
  another agent reading each change against the code at its commit).
  The gardening path, `workline route schedule --no-apply --forge none`
  then `workline apply`, with each repository's own `.workline` config;
  a fresh copy per run, and nights chained in it, each night's change
  committed as if merged, until the queue held only docs for a person or
  the run's share of tokens was spent. WaterMeter: the solo copy of the
  tried.md entry above (adopted, f98150f's debounce change), nights 1–2
  of 5 runs. DomoticsCore: origin/main 91100ed (after #114), 8 nights in
  runs 1–3, 7 in runs 4–5. workline: this branch at ffdb06d, nights 1–2.
  The baseline engine (a96306c) run on the same first night, five times
  each, for the tokens; and once on workline's second night.

  | | WaterMeter (5 runs) | DomoticsCore (5 runs) | workline (5 runs) |
  |---|---|---|---|
  | docs judged whole | 3 a run | 8 a run (7 in run 5) | 2 a run |
  | `checked` moved | 2 docs in runs 3, 4 | none (no doc's sources fit) | 1 doc in run 5 |
  | moved over a falsehood | 0 | — | 0 |
  | right fixes | GPIO2 → GPIO34, 4 places, 5/5 | versions in 4 docs 5/5, index's table 4/4 reached; 2 line counts 5/5, OTA's table 5/5; HeapTracker's pitfall 3/5 | — (nothing wrong in the two docs) |
  | a doc in parts | — (off) | README, 3 runs: 3 right fixes, recorded once | README, 5 runs: nothing changed, never recorded |
  | `count-off` reported | 2, both real | 58, all real | 0 |
  | `value-left` reported | 0 | 57 real, 25 false a run (21 and 4 in run 5) | 0 |

  `checked` moved five times, each doc read whole against its sources:
  BREADBOARD_LAYOUT.md and TROUBLESHOOTING_ASSEMBLY.md (WaterMeter, runs 3
  and 4) say nothing the config header contradicts — GPIO34 is
  `pulseInputPin = 34` — but are wiring guides the header cannot show; in
  run 3 the agent moved `checked` while its note said the resistor values
  "are not in the given source". The other three runs set `judged`, as the
  baseline engine does in 3 of 5. roles/documentalist/README.md (workline,
  run 5) is right, every page and check it names is there; its one source,
  role.yaml, does not show them, so the four runs that set `judged` are as
  right. Nothing false was vouched for.
  The fixes, read against the code: versions from each `library.json` and
  `metadata.version`; counts the engine's; the pitfall rewritten from
  `ownBytes_` (HeapTracker.h 103–105), left with a note in two runs ("I only
  saw the diff"), where the baseline fixed it 5/5; README's bump and check
  bullets from tools/bump_version.py and check_versions.py. No true claim
  removed, no comment winning, no `comment-not-evidence` refusal. Wifi's
  history row "1.4.1 | Current release" became "1.7.0 | Current release" in
  every run, so the history skips 1.4.1: not false, history thinned.
  Misses: OTA's "**Total** 1483" (1841 now), fixed in run 1's first answer
  and withdrawn after `removal-uncited` (not an engine count); WaterMeter's
  PULSE_LOGIC.md still says the debounce is 500 ms, too large to be judged
  whole, left for a person.
  `count-off`: every report real; none missed within the rule; outside it,
  WaterMeter's ARCHITECTURE.md "**Size**: 92 lines" and "265 lines", the
  file named in the heading above. `value-left` false alarms: Home
  Assistant's "(v2.0.0)" marking when a feature came (19), version-history
  rows and "Removed Fields (v1.4.1)" (4), the CHANGELOG notes (3), an
  example tag, and Wifi's WebUI fallback `"1.4.1"`, which the code still
  returns. No `value-left` missed in a check of every doc declaring Wifi's
  or SystemInfo's sources.

  Tokens, the first night, the same task on both engines, five runs each:
  WaterMeter 32.4k → 33.2k (+3%), workline 18.8k → 19.9k (+6%),
  DomoticsCore 34.6k → 43.6k (+26%; 36.2k, +5%, without run 5, asked
  twice: a patch that did not apply, then `removal-uncited` on "Watch"
  beside a count, the doc left out that night); in all 85.7k → 96.7k
  for six docs, **+12.8%**. A doc judged whole costs 21.8k on average
  over the 65 judged. Judging in parts: workline's README 228.6k a night,
  7 calls (the baseline 208.9k, 6 calls, +9%); DomoticsCore's 323k, 9
  calls. In 8 of 11 such nights `claims-dropped` kept the doc from being
  recorded, so it is asked again the next night — the baseline engine
  too. 4.63M tokens in all: 4.00M for the runs (WaterMeter 0.48M, of
  which 0.24M two nights condensing CHANGELOG.md, refused; DomoticsCore
  2.04M; workline 1.47M), 0.64M for the baseline.
- ADR-0014 step 4 again (2026-10-02, the engine at a15fc0d, after the
  three costs were cut; the same method, repositories, commits, scripts
  and caps as the entry above; Claude Sonnet 5.5 at medium effort; graded
  by reading each change against the code at its commit). The baseline
  engine's first nights of the entry above are reused: same task, same
  copies. Per bar, first run → now:

  | Bar | First run (ffdb06d) | Now (a15fc0d) |
  |---|---|---|
  | 1. no `checked` over a falsehood | held: 5 moves | held: 8 moves, WaterMeter's two wiring guides in runs 1, 2, 4, 5; read whole, only GPIO34 is the header's, and it holds |
  | 2. no true claim removed, no comment winning | held; Wifi's history row thinned | held; the same row thinned 5/5; every removal checked below |
  | 3. counts and versions reported | `count-off` 58 + 2, X no miss within the rules |
  | 4. what is right ends `checked` | WaterMeter 2/5, the role README 1/5 | WaterMeter 4/5, the role README 0/5 (`judged`, as right) |
  | 5. right fixes | versions 5/5, index 4/4 reached, OTA's `Total` 0/5, HeapTracker 3/5 | versions 5/5, identical across runs; index 5/5; OTA's `Total` 5/5; HeapTracker 4/5 (baseline 5/5); README's tools bullets 3/3 |
  | 6. tokens a judged doc, first night | +12.8% | **+13.9%**: WaterMeter 32.4k → 33.4k (+3%), workline 18.8k → 20.1k (+7%), DomoticsCore 34.6k → 44.1k (+28%; 36.9k, +7%, without run 1) |

  Bar 6 fails on the same shape as before: DomoticsCore's run 1 asked
  twice on its first night (72.9k), the HeapTracker pitfall rewritten
  without a claim (`removal-uncited`, rightly: "so creation itself
  affects heap measurements" is a fact), then withdrawn. Without it,
  +5.5% in all. Over every night, a doc judged whole costs 22.0k (21.8k
  before, the same 65 docs). Asked again three times over every night
  judged whole, as before (no condense refused, two before): that one;
  OTA's "`OTA.cpp` line count (607)", which `count-off` does not cover
  (no "lines"), the right fix to 759
  refused uncited (run 1, night 2) and left in the four others; a patch
  misquoting a line (run 2, night 2), right at the second answer.
  The narrowed removal rule let nothing false through: every line taken
  out without a claim, in every run, is a count the engine reported
  (core's two, OTA's table, its `Total` and its size row), the line
  otherwise unchanged; or a README line only added to. No glue, no word
  beside a count, taken out in any run.
  A doc in parts whose claims were dropped is now recorded
  (`judged-in-parts`): workline's README 5/5 (0/5 before),
  DomoticsCore's 3/3 (1/3), and not asked again — workline's third night
  judged docs/spec/conformance.md in parts instead, rightly adding
  `go.mod`, the fixtures and "not their READMEs" from eval_test.go's
  `evaluated`. A night in parts costs as before: 229.7k (workline), 325k
  (DomoticsCore). CHANGELOG.md was never picked: WaterMeter's run 1 went
  on to condense six other docs on nights 3–8, each a faithful move
  (every line taken out is in its companion, born without `checked`),
  15k to 25k a night. In parts, DomoticsCore's README fixes are right
  (tools/bump_version.py and check_versions.py: the `library.properties`
  sync, the skipped `.pio`/`test`/`examples` paths, a component with no
  `metadata.version` skipped). No engine bug found. 3.96M tokens
  (WaterMeter 0.37M, workline 1.54M, DomoticsCore 2.05M); the baseline
  not run again.
- What holds of a fix applied, with no agent (2026-10-02, the engine after
  c3487a4; replays only). Every re-ask on record was read first, the
  refused answer beside the one that followed: after a citation refusal,
  the place came back right once in five (d784398's versions, claimed at
  the second answer); 16c660b's count, HeapTracker's pitfall, OTA's
  `Total` and OTA's "`OTA.cpp` line count (607)" were withdrawn. After a
  misquote, three in four came back right (DomoticsCore's runs 2 of step
  4 and step 4 again, step 3's evaluation run); one withdrew every body
  fix. So a citation refusal is withheld and reported, never asked again;
  a misquote is asked again only when no hunk quotes right. Replayed:
  the reviewed bot fixes through the removal rule — the three wrong
  places withheld, 11 of 12 right ones pass (172493e, a template's header
  comment, withheld as before), and 3cd196d's right line 60, reverted with
  the wrong line 148, now applied. Step 4's refused answers through
  `post`, each on a copy of DomoticsCore at the commit its night began:
  HeapTracker's (run 1, night 1) passes, its two counts applied, the
  pitfall withheld (`removal-uncited`) and the doc recorded `judged`; OTA's
  607 (run 1, night 2) passes whole, `count-off` now reading its shape;
  the misquoted OTA answer (run 2, night 2) applies all but the hunk
  misquoting a table row — a hunk that also brought the size row to 759
  and 558, which the re-ask had mended: now a person's. `count-off` on
  DomoticsCore at 91100ed: 59 reports before, 60 after, the one added
  real; none on workline; WaterMeter's 2 unchanged.
- ADR-0014 step 4 a third time (2026-10-02, the engine at d890be5: what
  holds of a fix applied, the place refused reported; the same method,
  repositories, commits, scripts and caps as the two entries above; Claude
  Sonnet 5.5 at medium effort; graded by reading each change against the
  code at its commit). The baseline's first nights reused. **Holds.**

  | Bar | ffdb06d | a15fc0d | d890be5 (now) |
  |---|---|---|---|
  | 1. no `checked` over a falsehood | held, 5 moves | held, 8 moves | held: 6 moves, WaterMeter's two wiring guides in runs 1, 3, 5, read whole; no doc with a place withheld moved |
  | 2. no true claim removed, no comment winning | held | held | held: every line out without a claim an engine count, a version value, or a line only added to |
  | 3. counts and versions reported | held | held, 61 | held: `count-off` 62 places, all real (OTA's "line count (607)" now read); `value-left` 56–57 real, 26 false a run |
  | 4. what is right ends `checked` | WaterMeter 2/5 | 4/5 | 3/5; the role README `judged` 5/5, as right |
  | 5. right fixes | HeapTracker 3/5 | 4/5; core counts 5/5 | versions 5/5 on the first nights, GPIO34 5/5, OTA's table, `Total` and 607 → 759 5/5; core counts 4/5, HeapTracker 4/5 (baseline 5/5), index's footer version 3/5 (5/5) |
  | 6. tokens a judged doc, first night | +12.8% | +13.9% | **+5.7%**: WaterMeter +3.1%, workline +7.2%, DomoticsCore 34.6k → 37.1k (+7.4%) |

  No re-ask in 65 nights (three before). Over every night judged whole, a
  doc costs 20.8k (22.0k, 21.8k); a night in parts 258k (268k). Three
  places withheld, each reported in the run's findings and the gardening
  request's body, none moving `checked`: HeapTracker's pitfall (run 1,
  `removal-uncited`; the rewrite was right and claimed, but the claim
  named no `doc` in a two-doc answer, so it was not read — the message
  says "no claim"); a hunk quoting a line the doc has not, "`IIComponent`
  placeholder" (run 3), which carried core's two right counts — left,
  still reported by `count-off`; a hunk skipping a blank line (run 4,
  index), which carried the footer's right 2.12.0 — left, reported by
  `value-left` too. Right refusals by the rule; the two misquotes are
  what a re-ask mended 3 times in 4. In run 2 the agent left the footer
  itself, `value-left` reporting it. Run 2's night 6 answered YAML broken
  by an unescaped quote in a note: `agent-invalid-output`, nothing
  applied or recorded, the doc judged again the next night (+33k).
  WaterMeter's run 1 condensed six docs on nights 3–8, each a faithful
  move; DomoticsCore's README in parts, run 1: two right additions
  (check_versions.py's `check_properties_vs_root`; library.properties'
  dependencies). 3.56M tokens (WaterMeter 0.37M, workline 1.51M,
  DomoticsCore 1.67M).
