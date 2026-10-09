# Product owner — tried for real

Each try with a real agent or on a real backlog, newest last.

## 2026-10-03 — planted issues, local forge, Sonnet

A copy of a small project: `WriteRows` once stopped one row short, a
commit fixed it. Four issues planted on the local forge: #1 the bug, fixed
since (obsolete); #2 the same symptom in a user's words, naming no code
(duplicate of #1); #3 the header row never written, on the same function
(a look-alike, still true); #4 a JSON export asked for (a need).

- **First run**: no issue had a state; each got one, the agent not asked.
- **Second run** (2,994 tokens in, 261 out): #2 closed as a duplicate of #1,
  #1's words quoted; #3 and #4 left open, the look-alike told apart.
  The agent said it had not been given the code: `pre` gave none, so #1
  could not be found obsolete. **Fixed**: `pre` now gives each file an
  issue names, whole, with its last commits (case
  `product-owner/code-named-is-given`).
- **Third run** (3,429 in, 682 out), #2 reopened: #1 proposed as obsolete,
  the fixed loop quoted and the fixing commit named, left to a person as
  `close-obsolete` starts at `propose`; #3 and #4 left open. #2 not closed
  this time: "it names no code", said in a note. One run each: whether a
  duplicate naming no code is closed varies; the evaluation's five runs
  will say how often.
- Seen, not fixed: the agent wrote "changed it after this was reported",
  though no issue's date is given; the report's record reads `{}` when
  empty.

## 2026-10-03 — the evaluation, five runs, Sonnet

`product-owner/keeps-a-planted-backlog` (tests/evaluation): five issues
planted on the local forge of a project whose last-row bug a commit fixed —
#1 that bug (obsolete), #2 no header row and #3 its duplicate in other
words, #4 empty rows written as blank lines (still true, on the loop the fix
touched: the trap), #5 a need. Five runs, 11 of 11 each: #1 proposed as
obsolete and left open, #3 closed as a duplicate of #2, #2, #4 and #5 left
alone and not proposed. About 3,800 tokens in and 360 to 570 out a run, one
call, 5 to 6 seconds.

What it does not measure yet: a duplicate by likeness only (proposed, not
closed, by ADR-0018 — the engine does not tell it apart yet); a real
backlog, its issues long and many; a wrong closing in the wild.

## 2026-10-03 — DomoticsCore's roadmap, on a copy, Sonnet

A clone of DomoticsCore (26b2ee0), no remote, the local forge. Its
CODE-ROADMAP.md (7,580 lines, 192 entries) had 38 entries whose heading
says nothing of done; a one-shot script, without AI, opened each as an
issue keeping its id (`BUG-4 — NTP: …`), its sources the files it names,
else the code its backticked names are in, else its component's header.
Ten runs of four issues, about 45 to 62k tokens in and 230 to 1,300 out
each (Claude Code's own prompt is most of it), 7 to 12 seconds.

- **Eight issues proposed as obsolete**, the code quoted: BUG-1, BUG-4,
  BUG-5, BUG-7, BUG-14, BUG-15, BUG-20, BUG-24. Each checked by hand
  against the code: all eight right; the lots table agrees for NTP and
  Storage. None closed: `close-obsolete` starts at `propose`.
- **None proposed wrongly.** Where it could not see enough, it said so in
  a note and closed nothing: BUG-6 (the member's type cut from what it was
  given), BUG-9 (the config path cut), MEM-1 (partly done, re-scope it),
  BUG-48 (a function's body not given), DOC-1 (two parts, one done).
- **No `sources` proposed**, though twice it said the code given was not
  the issue's (SEC-3's route, BUG-56's browser script): it could not tell
  which file was. Only files it is given can be named.
- **Defects found, fixed** (cases in tests/conformance/cases/product-owner):
  - the report was rewritten each run with that run's proposals only: seven
    of the eight were lost. A proposal now stays until a person settles it
    (`proposals-kept-until-settled`);
  - an answer that did not read (prose around YAML) was not asked for
    again — the role had no `promote-after` — and the issues it held were
    still recorded as read. It is asked for again once now, and an issue
    is recorded as read only when the answer reads
    (`unreadable-answer-reads-nothing`, `if-answered` in the role contract);
  - the first try's prompt, 2,000 lines of code, was over the role's budget
    and refused before any call: `code-lines-max` stays at 1,500.
- Not rerun after the fixes: the 570k tokens again to see the report hold
  eight lines, which the cases now prove.

## 2026-10-03 — the import as a command, milestones, Sonnet

A fresh clone of DomoticsCore, no remote. `workline issues import
docs/CODE-ROADMAP.md --apply --forge local`: 188 entries, 150 marked done,
38 issues opened, the same 38 as the script's; its sources, by path or by
backticked name, tests last, left none for 7 entries. Two runs of four
issues (57k and 48k tokens in, 1.5k and 0.7k out):

- **Milestones**, the last release v2.13.0 given: SEC-3 (HIGH) put in
  `v2.14.0`, MEM-7 and MEM-8 (LOW) in `v2.15.0`, each with why. SEC-3's
  code was not given — the import found no file for it — and the lots
  table says it was merged long ago: put in a milestone on its title
  alone. **Fixed** in the instruction: only an issue the code given shows
  still true; not rerun.
- **`sources` used for the first time**: BUG-5, which the import tied to
  no file, named `NTP.h`, a line of it quoted; it is read again with it.
- BUG-1 and BUG-4 proposed as obsolete again, as in the first try.

## 2026-10-03 — workline's own BACKLOG.md imported by the agent, Sonnet

The parser of the first import read DomoticsCore's headings only, and found
0 entries in workline's docs/BACKLOG.md (numbered and bulleted items, no
ids, "Done … Left: …" inside the text). Replaced: the agent reads the file
a share at a time and proposes `open`, its text quoted; the engine checks
the quote and opens once.

On a copy of workline, local forge: one share (238 lines), one call, 14.6k
tokens in, 6.7k out. 30 issues opened; the eight items of the last section,
each "Done", left; an item done in part titled by what is left ("Reusable
action for install without Go in CI"). **Defect found, fixed**: an issue's
body began with the last two lines of the item before — the quote was
located from the first line holding its first word (`-`); it now starts at
the last line that still holds it all (`TestLocate`).

## 2026-10-04 — live on GitHub, JN0V/workline-sandbox, Sonnet

Three issues planted on the sandbox (`src/auth/token.go`, one-hour tokens):
#2 two-hour sessions (true), #3 the same need in a user's words, #4 "give
3600 a name" (solved: `const TokenTTL = 3600`). The engine on this machine,
`--forge github`. Each run 4.2 to 4.4k tokens in, 0.2 to 0.8k out.

- **Run 1**, no agent: each issue got its state comment.
- **Run 2**: #3's code named (`token.go`), its duplicate left for the next
  run; #4 proposed as obsolete, in the report issue #5.
- **Run 3**: #3 read again, not closed: "#2 looks like a duplicate, but
  only its title was available". **Fixed**: an issue on the same code as
  one read is given whole beside it (`original-given-whole`). Rerun: #3
  closed as a duplicate of #2, GitHub's reason `DUPLICATE`, #2's words
  quoted.
- **#3 reopened by hand**, with a comment: "not the same, the session must
  survive a browser restart". The next run found nothing to read and did
  not see it. **Two defects, fixed**: the record of closings was read only
  when a run had acts — it is read on every run of the role now
  (`wrong-closing-found-without-acts`); an issue was read again only when
  its code changed — also now when a person commented since
  (`read-again-when-someone-wrote`) or after the role's closing was undone.
  Rerun: `wrong-closing` on #3, `close-duplicate` back to propose, the
  report saying so; the agent read the comment, left #3 open and asked a
  person to reword it around the restart.

## 2026-10-04 — DomoticsCore's roadmap, read by the agent, before applying

`workline issues import docs/CODE-ROADMAP.md --forge github` on
DomoticsCore itself, without `--apply`: 7,580 lines, 32 shares, Sonnet.

- **First run**: 69 issues would be opened, 0.75M tokens in. About fifteen
  were done: MEM-1, BUG-1, BUG-4 to BUG-7, BUG-9 to BUG-12, BUG-15, SSE-1,
  their headings without a "DONE", the file saying so elsewhere — the
  lots table at its top, the "Resolved in v2.0.1" table near its end —
  in another share. BUG-6 twice, under two titles. **Fixed**: each share
  comes with the lines of the rest of the file that name its ids
  (`import-sees-the-rest-of-the-file`).
- **Rerun**: 55, none of the fifteen; 1.25M tokens in — the ids' lines
  cost a share about two thirds more. Three items still to do that the
  first run had found were left out this time (BUG-94, CI-7, CI-16): the
  agent's variance, not the lines given. Not fixed: an import run again
  opens only what no open issue holds, so a second run picks them up.
- **Applied** (`--apply`), 1.24M tokens in: 52 issues opened, JN0V/DomoticsCore
  #134 to #186, the report #135. MEM-1 opened though the file says it is done
  (the "Resolved in v2.0.1" table, the tracking summary): the agent's miss,
  closed by hand with those lines quoted. **Two defects, fixed**: the share
  holding the LO table had 35 items, `open` capped at 30 — the five past it
  were proposed, then dropped from the report by the next share, having
  no issue number to stay by; and the `act-cap` finding named them "#0".
  A proposed open is now kept until an open issue holds its text, and named
  by its title (`import-capped-kept-proposed`). The five were opened by one
  more run on that share (`run-role product-owner --event import`), #187 to
  #191: 56 issues from the roadmap.

## 2026-10-04 — refining, live on DomoticsCore and on the sandbox, Sonnet

The engine on this machine, `run-role product-owner --event schedule
--forge github`, on DomoticsCore's imported issues, then on two issues
opened on JN0V/workline-sandbox for it.

- **No refine at all, three runs**, each found and fixed:
  - the imported issues name their files bare (`MQTT_impl.h:8`) or name
    only a symbol (`MQTTPublishEvent`), and the line the import ends with,
    and its hidden key, named the roadmap itself — the agent got no code
    and said so (`code-named-by-its-file-name`, `code-named-by-a-symbol`;
    GitHub's bodies end their lines in CRLF, which the first fix missed);
  - with the code, the prompt was 32.7k tokens, over the role's 32k: the
    budget is 40k now;
  - then, in the agent's own note: "the answer format offers no refine or
    ask" — the output contract had no shape for them. A test now holds
    every intention of the catalogue to have one
    (`TestEveryIntentionHasItsShape`).
- **DomoticsCore** (66.7k tokens in, 3.2k out): #163, #167, #168, #171,
  #172 refined — Scope and Verification from the code (a native test, the
  hook it can use), Need and Validation drafted; #173 past the cap of 5,
  proposed. The drafts began "Draft:", which the engine's line says
  already: the instruction asks for the text alone.
- **The sandbox**, each run 5 to 7k tokens in: #6 (clear) refined; its two
  drafts accepted by hand (their lines deleted); the next run read it
  again for its body changed, proposed `ready`, the engine checked it and
  labelled it `workline:ready`. #7 ("It is slow sometimes") asked four
  questions, `@JN0V` named; answered, the next run drafted its Need and
  Validation from the answer. **Defect found, fixed**: the file the answer
  named was not given — only the body was searched (`code-named-in-a-comment`);
  rerun, #7 got its Scope and Verification, its drafts left alone.
- A run seconds after the one that wrote the issues' state comments did
  not see them — GitHub listed the comments late — and wrote them again,
  in place: no duplicate, the run lost.
- Not tried: `ready` on an outsider's issue (proposed; conformance only),
  an issue whose reporter never answers.

## 2026-10-04 — drafts accepted with a label, live on the sandbox

The person said deleting a line in each issue would not do: no one does
that for ten issues. Accepting is a label now, `workline:accepted`, set
on one issue or on many from the list of issues; only who may triage sets
a label. JN0V/workline-sandbox #7, its Need and Validation drafted: the
label set by hand; the next run, no agent call, took the draft lines out
and moved it to `workline:ready`. Not tried live: an issue form's
`### ` sections (conformance only), the label created by the engine with
the first draft (created by hand on workline and DomoticsCore, whose drafts
predate it).

## 2026-10-04 — ordering, live on the sandbox, Sonnet

JN0V/workline-sandbox, a fresh clone, the engine on this machine, `--forge
github`. Set up by hand: a tag `v0.1.0` in the clone (not pushed), the
milestones `v0.1.0` and `v0.2.0`, #4 in `v0.1.0`, `workline:priority/2` set
on #3 by a person; a comment on #2, #6, #7, then #3, to have them read again.

- **Run 1** (7.5k tokens in, 1.6k out), five open issues, so one move a
  run (a fifth): #4 **moved by the engine** from `v0.1.0`, released, to
  `v0.2.0`, no agent needed for it; the agent's orders — #7 first (sign-in
  slow for everyone), #2 second, #6 third — and a milestone for #2 were
  past the share: proposed (`moved-cap`), the report saying each issue's
  priority and milestone before.
- **Run 2** (8.3k in, 1.1k out), `moved-percent-max: 100` in the clone's
  config: the capped issues read again first; #7 got `workline:priority/1`,
  #2 and #6 `/3`, #2 the milestone `v0.2.0`; the labels `/1` and `/3`
  created by the engine; each issue's state recorded its priority; the
  report listed the three as they were before. #3, its priority shown to
  the agent as a person's, was not ordered: the agent asked its reporter
  instead which need it holds, #2's or #6's.
- Seen, fixed: the labels' description named the role `product-owner`;
  the slip's line ran into the next sentence.
- Not tried live: the engine refusing an order over a person's priority
  (`priority-kept`, conformance only: the agent did not propose one);
  a slipped issue with no open milestone left; GitLab.

## 2026-10-04 — the one way to open issues

Tried live through the reviewer, on JN0V/workline-sandbox #9
(roles/reviewer/docs/tried.md): a subject held by an open issue not opened
again, then by one closed as completed said once on it. The product
owner's side — a role's issue read first as its draft to refine — is in
conformance only (`role-finding-read-as-draft`): no product owner run
followed.

## 2026-10-05 — splitting and renaming, live on GitHub and GitLab, Sonnet

The engine of this branch, built on this machine; the sandboxes'
fresh clones. Two issues planted on each: "Session handling", three needs
in one (a two-hour lifetime, a silent refresh, signing out everywhere),
and "pls fix", a `TokenTTL` that should be a `time.Duration`.

- **GitHub, JN0V/workline-sandbox** (`--forge github`). Run 1 (9.3k
  tokens in, 1.3k out): the two issues got their state. Run 2 (8.9k in,
  2.4k out): #10 split into #12 "Refresh an expired token silently" and
  #13 "Sign out on every device" — the agent left the two-hour lifetime
  out, "already #2" —, each with its four sections, Need and Validation
  drafts, `Part of #10.`, linked as **native sub-issues**; #10's state
  `split: [12, 13]`. #10 renamed "Sign-in sessions: silent token refresh
  and sign out on every device", #11 "Make TokenTTL a time.Duration
  instead of a bare number of seconds". Run 3 (two calls, 23.6k in, 1k
  out): only the children read — refined, ordered —, nothing split or
  renamed again. Run 4, #11 renamed by hand and both commented on to have
  them read again (12k in, 1k out): the task said "Split into: #12, #13
  (not split again)" and "Title: a person's (kept)"; the agent proposed
  neither, and said so in its note.
- **GitLab, JN0V/workline-sandbox** (`--forge gitlab`, glab's token).
  GitLab takes every reporter for an outsider, so both issues were opened
  with `workline:accepted`, as a person of the project accepts. Run 1, no
  agent: states. Run 2 (15.3k in, 2.7k out): #5 split into three —
  opened as issues, converted to **tasks** and given #5 as parent through
  GraphQL, read back under #5 as its children; #6 renamed "TokenTTL is a
  bare number of seconds instead of a time.Duration"; the report saying
  each, and how to undo it. Run 3, #5 commented on (18.4k in, 1.3k out):
  the tasks read and ordered as any issue, through REST; #5 not split
  again ("#5 was split into #7, #8 and #9 and is not split again").
- About 88k tokens in and 10k out in all, eight calls.
- Seen: the agent refined a parent in the run it split it, writing the
  whole need's four sections — allowed, the parent keeps the need. The
  other product owner branch ran on the GitHub sandbox at the same time
  and rewrote its report issue (#5) after run 2: the report of a split
  was seen on GitLab's (#10), not on GitHub's.
- Not tried live: a split stopped half-way and resumed (conformance:
  children keyed, state written last); the task list fallback (local
  forge, conformance only); an outsider's split proposed; a forge plugged
  by a command answering `add-sub-issue`.

## 2026-10-05 — asked again after an answer; an outsider's issue proposed, Sonnet

The engine built from this branch, on a fresh clone of
JN0V/workline-sandbox. Five agent calls, 41.6k tokens in and 7.1k out in
all; two of them thrown away (below).

- **An answer read, live on GitHub.** #3 had been asked once (2026-10-04)
  which need it held. Answered twice by hand: an idle session ending after
  two hours, any request pushing the end back. The run (9.4k in, 2.8k out)
  read #3 again, the task showing "Written to its reporter: 1 of 3 times;
  answered since", the question it asked among the comments; the agent
  drafted Need and Validation from the answer, Scope and Verification
  from the code, saying #2 and #6 are something else. Applied: #3 refined,
  its drafts marked, the report saying so.
- **Scoped by hand.** Another agent ran the product owner on the same
  sandbox at the same time (#10, #11 opened, a run of theirs consumed the
  first answer). The run was judged with `--no-apply`, its intentions cut
  to #3's — its state and its refine — then applied: the other issues left
  to that agent's try. A first run used a binary the other agent had
  overwritten in a shared folder: thrown away, as was a run whose answer
  the other agent's run had already read.
- **An outsider's issue: a real agent, a simulated forge.** No account
  without write access to the sandbox exists, so its author cannot be
  outside on GitHub. The sandbox's code with `--forge fake:`, one issue
  by `zed`, without write access, in a user's words ("logged out while
  typing a long message"). Run 1 (6.1k in, 0.4k out): a comment to
  `@zed` — what it understood, Verification and Scope as it would write
  them, three questions; the body untouched. Zed's reply added by hand;
  run 2 (7.5k in, 0.5k out): a second proposal (`proposal=2`), Need and
  Validation now drafted from the reply, "nothing more is needed". The
  label `workline:accepted` added by hand; run 3, no agent: the four
  sections written in the body, the drafts accepted, `workline:ready`.
- Not tried live: the proposal comment on GitHub itself (the same comment
  call as an ask, tried live); a real outsider read as one (GitHub's
  author association); a follow-up question (`ask=2`), the rounds spent,
  a question refused as asked before, a proposal not repeated without an
  answer — conformance only; the reporter agreeing by editing their own
  body; GitLab.

## 2026-10-05 — GitLab's members, a bot's token, a reply `agreed`, Sonnet

The engine of feat/po-gitlab on JN0V/workline-sandbox (gitlab.com, a Free
user's project), a fresh clone, with project access tokens made on it
(ADR-0023), never printed: a Planner bot for the local runs, a Guest bot
to stand for an outsider, a Developer bot (`api`, `write_repository`)
stored by the one glab command of docs/ci.md as `WORKLINE_GITLAB_TOKEN`
for CI. Three agent calls, 41.7k tokens in and 4.4k out in all.

- **Insider and outsider told apart.** Planted: #13 "bug" and #14, #16
  by the owner; #15 "Logged out while typing" by the Guest bot. Run 1, no
  agent: the states, written by the Planner bot. Run 2 (18.8k in, 2.0k
  out): #13 — the owner's — **renamed in place** ("Intermittent logout on
  page reload in Firefox") and asked which page; #15 — a Guest's — got
  the **proposal** to its reporter, its body untouched. #14's split was
  dropped (`sources-unknown`: the sandbox holds no account code).
- **A reply agrees.** The owner answered #13, and replied "Agreed." to
  #15's proposal. Run 3 (10.6k in, 0.7k out): #15's Need and Validation
  written with no agent, the report saying "agreed to by @JN0V (a person
  of the project) in a reply"; #13 refined in its body from the answer,
  Need and Validation drafts. Ready stayed the label's.
- **A split, and the Planner role's limit.** #16 "Token rules", three
  needs. Run 4 (12.3k in, 1.8k out): split into #17 "Revoke a token by
  its id" and #18 "Sign tokens with a key rotated every month" (the
  two-hour lifetime left out, "already #7"), opened by the bot — but
  listed in #16's body, not linked as tasks: GitLab refused the Planner
  token the parent ("it's not allowed to add this type of parent item"),
  where a Reporter token and the owner's succeed (both tried by hand on
  #17, #18). Hence Reporter, not Planner, in ADR-0023.
- **The state edits, and a bug found.** The owner's state notes on the
  older issues were refused to the bot (403) and written anew after them,
  as designed — but GitLab lists notes newest first unless asked, so the
  engine took the oldest for the last and wrote a new one at each write
  (#9 got three, #15 two in the CI run). Fixed (f4855eb, 759a951 once rebased on main), with a test
  that serves notes newest first.
- **GitLab CI.** A branch of the sandbox ran the engine of this branch
  (`go install …@<commit>`, golang:1.27), no agent. The job's own token
  alone: exit 3, "reading the project's members… 401 Unauthorized" —
  loud. With the Developer bot: #15, the label `workline:accepted` set
  by the owner, got the last proposal's Verification and Scope written
  and `workline:ready`. Again with the fix (f4855eb), #13 accepted by the
  label: its drafts' lines out, `workline:ready`, its state — the
  Planner bot's — written anew once by the Developer bot, no more.
- Not tried live: a real person outside the project (the Guest is a
  bot: its own `agreed` would not count, so the owner agreed instead); a
  reporter's own `agreed`; a group access token; a self-managed
  instance; the gardening template itself with this engine (it pins a
  release).

## 2026-10-05 — obsolete announced, then closed or kept, live on GitHub and GitLab, Sonnet

The engine of feat/po-close-obsolete, built in its worktree; fresh clones
of both sandboxes, a clone's `.workline/config.yaml` setting
`issues-per-run: 2`, `close-obsolete: {mode: act, days: 0}` — the delay
simulated by the setting, no clock moved: 0 makes an announcement due at
the next run — and the other moves off. On each, an issue planted the
code had already settled (the lifetime it asks for is the one
`TokenTTL` holds), and one asking for a name the constant already has.
Nine calls, about 79k tokens in and 5.2k out in all.

- **GitHub, JN0V/workline-sandbox** (`--forge github`). Run 1 (11.6k in,
  0.6k out) stopped before writing anything of the act, loud: GitHub
  refused the label `workline:obsolete` (HTTP 422), its description over
  100 characters. **Fixed**, shortened. Run 2 (11.9k in, 0.7k out): #14
  and #4 announced — the label, a comment to `@JN0V` quoting `const
  TokenTTL = 3600`, the commit, "closed at a run from 2026-10-05", how to
  keep it open; the report listing both. #4 answered by hand ("not done
  for me"). Run 3 (11.9k in, 1.0k out; the judge 2.9k in, 0.3k out): #4
  kept open, its label off, `kept` in its state, nothing written to the
  person; **#14 closed as completed** by the engine, the judge's yes and
  "independence: model, claude-sonnet-5-5 → claude-opus-5-5" in the
  closing comment. The agent, reading the reply on #4, proposed `keep`
  as well: both applied, harmless; **fixed**, one `keep` an issue a run.
- **GitLab, JN0V/workline-sandbox** (`--forge gitlab`, glab's token).
  Run 1 (12.3k in, 0.6k out): states. Run 2 (11.2k in, 0.7k out): #19 and
  #20 announced, labelled. #20 answered by hand. Run 3 (11.2k in, 0.8k
  out): #20 kept open, its label off, `kept` in its state; #19's judge
  (2.9k in, 0.3k out) answered `note: "yes: …"` without the list's dash,
  which did not read: **not closed, said** (`judge-unavailable`, it
  waits). **Fixed**: a note read with or without the dash. Run 4, no agent
  of the role's (the judge 2.9k in, 0.3k out): #19 closed, the label off,
  the judge and its level said.
- Not tried live: the label taken off by hand, an exempt label, a judge's
  no, the cap shared by announcements and closings, a closing reopened —
  conformance only (`product-owner/obsolete-*`); a delay of days waited
  for real.

## 2026-10-05 — the person's hand, live on GitHub and GitLab, no agent

The engine of feat/po-persons-hand, built in its worktree; fresh clones of
both sandboxes, `.workline/config.yaml` setting `issues-per-run: 1` and
`rename: {mode: propose}`. The report's record and lines **planted by
hand** as the engine now writes them — a rename proposed with the act it
stands for — then the box ticked through each forge's API by `JN0V`, the
owner, in an edit of its own. No agent answered in any run: **0 tokens**.

- **First, what the forges say** (docs/research/product-owner.md, "A tick
  and who ticked it"). GitHub: the report's `userContentEdits` hold each
  version of its body, whole, with its editor, newest first; `JN0V`'s
  permission reads `admin`, a stranger's `read`. GitLab: an issue opened
  for the probe and deleted after; a description changed through the API
  wrote "marked the checklist item **…** as completed" by `JN0V`, the
  item's markdown escaped and the hidden key's text kept without its
  delimiters; a line added already ticked wrote none.
- **GitHub, JN0V/workline-sandbox** (`--forge github`). Run 1, `--ai
  claude`, the record saying `ignored: 3`, nothing done by a person
  since: `paused`, **no agent asked** though issues waited to be read.
  Then two boxes ticked by `JN0V` — the rename of #7 and "Set
  close-duplicate back to act" (the kind back to propose since #3 was
  reopened, 2026-10-03). Run 2, `--ai none`: **#7 renamed** to "Sign-in
  is slow at times", "Ticked by @JN0V." in the report; **close-duplicate
  back to act** (`back-to-act`), the record's `propose` emptied, `ignored`
  gone; the proposal left the record. Seen there and **fixed**: the line
  "#3 … was reopened" stayed alone once the kind was back to act; now
  said only while the kind is back to propose.
- **GitLab, JN0V/workline-sandbox** (`--forge gitlab`, glab's token).
  The record planted in a note of `JN0V`'s, after the bot's (ADR-0023:
  the last is read). Run 1, `--ai none`: **#14 renamed** to "Account
  settings: change email and password, delete the account", read from the
  system note, "Ticked by @JN0V." in the report.
- Not tried live: a tick by an outsider or a bot (one account on each
  sandbox; the project bot's token lives in GitLab CI only) — conformance
  `tick-by-outsider-ignored`, `tick-by-a-bot-ignored`; an author the forge
  does not say (`tick-author-unknown`); a proposal written by a real
  agent's run, then ticked; the pause reached by three real runs rather
  than planted; a closing as obsolete ticked (`tick-closes-obsolete-at-once`).
- **Fixed after**, seen reading the API: GitHub lists `userContentEdits`
  newest first, so `last: 100` read the oldest hundred; now `first: 100`,
  and the oldest version read, when older ones are left, only a baseline.
  Run 3 on GitHub, `--ai none`, the report then at 20 versions: a box of
  an older proposal, recorded before its act was kept, ticked by `JN0V` —
  said in the report as one to do by hand, and gone from it.

## 2026-10-05 — autonomy levels, cautious then enterprising, live on GitHub, Sonnet

The engine of feat/po-autonomy-levels, built in its worktree; a fresh
clone of JN0V/workline-sandbox, `--forge github`, its
`.workline/config.yaml` setting `issues-per-run: 4` and the level. The
sandbox's eleven open issues set aside (closed as not planned, reopened
after), so the agent read only the backlog planted: "bug" (signed out
after an hour, `TokenTTL` 3600, eight hours wanted), "Signed out after
one hour" (its duplicate), "Sessions: last a working day, and sign out on
every device" (two needs), "Name the token lifetime's unit". The same
four planted twice — #15 to #18 for cautious, #20 to #23 for
enterprising, the first set and its report closed between — each first
given its state by a run with no agent. **Three agent calls, 25.6k
tokens in and 5.0k out in all**; the undo runs, none.

- **cautious** (#15–#18, report #19; 8.5k in, 2.2k out): the acts that
  check facts **done** — sources named on #16, #17, #18; #15 and #18
  refined with **Verification and Scope only**, labelled to-refine, no
  draft label (`drafts-proposed` said). The acts that set direction
  **proposed**, eight boxes: #16 closed as a duplicate of #15, two
  milestones, two priorities, the Need and Validation drafts of #15 and
  #18, #17 split in two. Nothing closed, moved or split. The report's
  `Autonomy: **cautious**` line gave each kind's mode, from the level.
- **enterprising** (#20–#23, report #24; 8.1k in, 0.9k out): **#21 closed**
  as a duplicate of #20, #20 put in v0.1.0 and given priority 2, refined
  whole (drafts written, labelled draft). The agent named no sources and
  proposed nothing on #22 and #23 this time: the same backlog, a
  different reading — the level changes what is done with what is read,
  not what the agent sees.
- Neither run renamed "bug": the agent judged it, twice, worth no rename.
- **Undone** (no agent, 0 tokens): #20's priority label taken off by hand
  → the next run found it, `undone`: "#20's priority put back to none by
  a person; the role had set 2", **order demoted** (`order: propose
  (demoted)` at enterprising), a box to set it back. A third run with the
  agent (9.0k in, 1.9k out; #20 read again for a comment) proposed its
  priorities, now demoted, instead of setting them, and set a milestone.
  The rename was **planted** in the record as the role writes it (`done:
  {issue: 20, act: rename, was: bug, set: …}`), #20 given that title; a
  run left it standing; #20 renamed back to "bug" by hand → `undone`,
  **rename demoted**, the evidence in the report.
- **Seen, not fixed here** (issue #156): an agent's unquoted `why` holding
  " #20" lost all after it — YAML reads a comment —, three of seven acts'
  reasons cut short in the report.
- Not tried live: `workline init --human-po` on a terminal (conformance
  `setup/init-asks-human-po`); `ready` taken off and a split's child
  closed as not planned (conformance `undo-ready-demotes`,
  `split-child-closed-not-planned-demotes`); `ignored-runs-max` reached;
  the report's suggestion, which needs ten proposals settled at cautious.

## 2026-10-05 — what an issue waits on, live on GitHub and GitLab, Sonnet

The engine of feat/po-dependencies, built in its worktree; fresh clones of
JN0V/workline-sandbox, a local `.workline/config.yaml` (`issues-per-run:
3`), level normal. Three issues planted on each forge: "Read the token
lifetime from the config file", "Let an admin set the session length per
tenant" — its body saying the config-file lifetime must land first —, and
"Session length: a config setting, then an admin page to edit it", two
needs, the second only once the first exists. Each first given its state
by a run with no agent. **Three agent calls, 41.4k tokens in and 4.6k
out in all**; the runs with no agent, none.

- **GitHub** (#25–#27, report #5). Run 2 (13.7k in, 1.7k out): #27
  **split** into #28 and #29, the second with `after: [1]` → **#29
  blocked by #28 in GitHub's own dependencies**, nothing in its body; #25
  refined, ordered. #26 not read (the cap: #2 read again first). Run 3
  (14.6k in, 1.5k out): **#26 blocked by #25** (`depend`, native); the
  agent, reading "Waits on: #28 (open)" on #29, closed #28 as a duplicate
  of #25 and proposed `depend` #29 on #25: #29 now waits on #25, #28's
  closed link left as it was. Findings `waiting` #26 and #29, `next-ready`
  #7. The report's "Waiting" still listed #28 as open, closed by that very
  run: **fixed**, the run's closings left out.
- **GitLab Free** (#22–#24, report #10). Run 2 (13.2k in, 1.4k out):
  `depend` #23 on #22 and #24 on #22 — `is_blocked_by` refused for the
  license (403), so **`Blocked by #22. <!-- workline:blocked-by -->`** in
  each body, after its text, the state's digest moved; the agent did not
  split #24, noting its part (1) is #22 itself. A run with no agent read
  the lines back: `waiting` #23 and #24, the report's "Waiting" saying
  both and **Next** #13. #22 closed by hand → the next run: no `waiting`,
  both ordered by their labels again; #22 reopened after.
- Not tried live: a GitLab Premium link; a cycle (conformance
  `cycle-reported-not-looped`, `depend-cycle-dropped`); a link the role
  set taken off by a person (`undo-depend-demotes`); `cautious`
  proposing `depend` (`depend-mode-per-level`); a plugged forge's
  `add-blocker`.

## 2026-10-05 — a parent's parts as they close, live on GitHub and GitLab

The engine of feat/po-parent-goal, built in its worktree; fresh clones of
JN0V/workline-sandbox, `issues-per-run: 3`, level normal. On each forge a
parent planted, "Session limits: an idle timeout and a cap on sessions
per user", its Verification two items, and two parts linked natively —
GitHub sub-issues, GitLab tasks (converted and parented through
GraphQL) —, the first part's Verification quoting the parent's first
item word for word. **One agent call, 16.0k tokens in and 1.4k out**
(GitHub, Sonnet); every other run with no agent.

- **GitHub** (#30, parts #31, #32; report #5). With no agent: the comment
  written on #30 ("0 of 2 parts closed", both items "Not proved yet"),
  and on the older parents #10 and #27 (#28 shown closed as a duplicate,
  its part not delivered). #31 closed by a commit pushed with "Closes
  #31": the next run edited the same comment — "closed as completed |
  commit efee34d", the first item proved, "quoted in #31's Verification".
  #32 closed as not planned by hand; the run with the agent: "All 2 parts
  are closed: for a person to accept", "Before accepting: the part of #32
  not delivered; 1 item(s) of its Verification not proved", the finding
  `parent-to-accept`, and the report's "To accept" listing #30. The
  agent's task said "Split into: #31 (closed), #32 (closed) — … never
  closed by the role"; it proposed `sources` on #30 and nothing else
  there; #30 left open, no label. A run more with nothing changed left
  the comment unedited (its `updated_at` the same). #30 then closed by
  hand, as a person accepting it: the next run's report has no "To
  accept" left.
- **GitLab** (#25, tasks #26, #27; report #10). The tasks read through the
  work items' GraphQL; #5's and #16's parts reported too. #26 closed by a
  commit pushed with "Closes #26": shown **closed by hand** — GitLab
  writes no "closed via commit" note, which the first build read. **Fixed**:
  what closed it is read from `resource_state_events` (`source_commit`,
  `source_merge_request_id`); after it, "commit 557866f5". #27 closed by
  merging merge request !4, its description quoting the second item:
  "merge request !4", the item "quoted by merge request !4, which closed
  #27"; all closed, the ask written, the same note edited in place, the
  report's "To accept" listing #25.
- Not tried live: a part gone from the forge; a part of another
  repository (GitHub); the local forge and a plugged forge's `closers`
  (conformance only).

## 2026-10-05 — the import's map, live on GitHub, no agent

`workline issues import` with its map (ADR-0030), on a local copy of
JN0V/workline-sandbox, writing to its GitHub issues. No agent called: the
answer planted (`--ai fake:`), or none (`--ai none`).

- **A planted answer**, a 13-line file (`IMPORT-163.md`, not pushed): two
  `open`s, a `done` with its words, two `not-item`s, one item left out.
  First run: #33 opened, the item left out under "Not covered" with the
  last one — #34, opened too, but missing from the map. **Fixed**: GitHub's
  list of issues, read just after, did not show #34 yet; which issue an
  opening produced is now read from the run folder (`out/openings.yaml`).
  Second run, the left-out item answered: #35 opened, #33 and #34 "already
  open", exit 0. #34 then closed by hand as completed: third run, "#34,
  closed", nothing written to it.
- **DomoticsCore's roadmap** (`docs/CODE-ROADMAP.md`, 7,587 lines), on the
  same clone, `--ai none`, without `--apply`: 32 shares in two minutes, the
  forge read each share; 560 paragraphs "Not covered", the summary saying
  so, exit 2. As it should: without an agent nothing is judged.
- Not tried: a real agent's answer with `skip`s — the token cost of a
  `skip` a line, and whether the agent answers for every line — and the
  maintainer reading DomoticsCore's map against its 56 issues (#163's
  Validation).

## 2026-10-05 — Next and Stuck, live on GitHub and GitLab, no agent

The report's opening (ADR-0031), the engine built from the branch, on a
clone of JN0V/workline-sandbox and a clone of JN0V/workline; every run
`--ai none`: no token spent.

- **GitHub, the sandbox** (report #5). Run 1, defaults: the report
  rewritten though the run did nothing else, opening with "Next" — "1. #7
  Sign-in is slow at times — no milestone, priority 1", "2. #6 … priority
  3" — and "Stuck: Nothing waits on a person for more than 14 days"; #26
  and #29 left in "Waiting". Run 2, `next-max: 1`, `stuck-days: 1`: #6
  out of Next, so its trail asked: `workline:ready` since 2026-10-04 read
  from the timeline, nothing naming it — one day, not past one: not
  stuck. The first build said "1 days"; **fixed**.
- **GitLab, the sandbox** (report #10), `next-max: 0`, `stuck-days: 1`:
  the label days of #6, #13 and #15 read from `resource_label_events`,
  #15's last question to its reporter from its note's `created_at`, all
  2026-10-05: none stuck yet; "## Stuck" written.
- **workline itself**, `--no-apply` (nothing written): Next is #79 alone,
  the only ready issue; #65, #91 and #92 wait on their reporters since
  2026-10-05; the record's proposals have no day yet (recorded before),
  so the first run will date them. Rendered as on 2026-10-21, the report
  would say #65, #91, #92 "its reporter written to on 2026-10-05 (16
  days); no answer since" and #83, #85, #87 "`refine` proposed here
  since 2026-10-05". #79's timeline holds a reference by pull request
  #112, older than its label (the backlog's move to issues): not taken
  for work started.
- Not tried: an issue past 14 days on a real forge — every day on both
  sandboxes is this week's; an announcement due with no judge, live; a
  plugged forge's `trail`; workline's #110 rewritten (that is the
  nightly run's, after the merge).

## 2026-10-05 — a changed need, live on GitHub, no agent called

A changed need flags the issues built on it (ADR-0032), the engine built
from the branch, on a fresh clone of JN0V/workline-sandbox, `--forge
github`. No agent called: `--ai none`, or the answer planted (`--ai
fake:`).

- **Run 1**, `--ai none`: every open issue's state kept its Need and
  Scope, once, with no agent (15 state comments edited); nothing flagged.
- **#10's Need rewritten by hand** (a split need: #12, #13), sign-out on
  every device dropped; #13 labelled `workline:ready` by hand, to have a
  ready part. **Run 2**, a planted answer (`unready` on #13, `order` on
  #12): #12 and #13 read first, the Need as it was and as it is in the
  task; the report's "Changed needs" listed #10 with #12 ("proposed below
  — order") and #13 ("— unready"), both proposed, neither done.
  **Fixed**: nine other issues were listed too, all only naming
  `src/auth/token.go`, as #10 does — noise on a small codebase. The code
  shared now counts only when the Scope changed.
- **The box ticked by hand** (JN0V), #10's Need edited again. **Run 3**,
  an answer proposing nothing: #13 moved back to refine — `workline:ready`
  off, `workline:to-refine` on, a comment saying why and who ticked it;
  the change listed #12 and #13 alone. **Run 4**, `--ai none`: nothing
  flagged — the same change flags once.
- Not tried: a real agent reading the parts against the change; an
  imported file's lines changed, live (conformance only); GitLab.

## 2026-10-06 — the weekly sample of its acts, live on GitHub, no agent

The weekly sample drawing the role's acts (ADR-0033), the engine built
from the branch, on a clone of JN0V/workline-sandbox, `--forge github`.
No agent called: the answer planted (`--ai fake:`), or none (`--ai none`).

- **Run 1**, a person's comment on #33 and #35 to have them read again,
  then a planted answer: #33's priority set to 3, #35's to 4, #35 renamed
  "Warn when an imported file is empty". The record's `did` kept the
  three, each with its day (2026-10-06), level (`normal`) and line.
  **Fixed**: the first build named the day `on`, which YAML 1.1 reads as
  a boolean — written quoted (`"on":`); it is `day` now (the sandbox's
  record edited to match).
- **#35 renamed back by hand. Run 2**, `--ai none`: the undo found,
  `rename` back to propose, the record's `undone` saying so.
- **`workline sample --week 2026-W41 --judge none`**, then `--apply
  --forge github`: the docs' tracking issue #36 ("no doc vouched for"),
  and #37, "workline: the weekly sample of the product owner's acts": 3
  acts done alone that week, 1 drawn — #33's order, standing, with its
  line —; "3 acts done alone at normal …, 1 undone … Too few acts … to
  suggest a level: 3, at least 10." A second `--apply` edited the
  comment in place (one comment on each issue).
- Not tried: an undone act drawn, live (#35's rename was not the one
  drawn; conformance shows it, and a closing reopened); a suggestion,
  live (three acts, not ten); a real agent's week; GitLab; workline's own
  report (#110), whose record has no `did` until its next nightly run.

## 2026-10-06 — an import judged in one job, applied in another, GitLab, no agent

`workline issues import` without `--apply`, its runs then applied by
`workline apply --line` (#174), the engine of the branch, on
JN0V/workline-sandbox's GitLab issues. No agent called: the answer
planted (`--ai fake:`) each time.

- **On a clone**: the judge with a project token `read_api`, Reporter,
  as `GITLAB_TOKEN` — no write token: the map said two items "would be
  opened", one run pending, exit 0. `workline apply --line` with the
  write token: #28 and #29 opened, their bodies the file's lines, their
  state; the summary file holding both jobs. The import again: "#28,
  already open", "#29, already open".
- **A GitLab pipeline of two jobs**, started through the pipelines API on
  a branch with no merge request (the sandbox's merge-request pipeline,
  which calls Claude, not started): docs/gitlab-trigger.md's script, its
  `import` task, the engine built from the branch's commit
  (`go install`), the agent a planted answer, the AI key unset. The judge
  held the read token (a masked variable, removed after, the token
  revoked); the apply, `WORKLINE_GITLAB_TOKEN`: #30 and #31 opened, each
  job's summary in its log, the map in the judge's.
- Not tried: GitHub Actions' pair (docs/ci.md); the template's own image,
  which has no release with this yet; a real agent's answer; an import of
  several shares split this way (conformance holds one share).

## 2026-10-06 — the context budget against the calls kept (#235)

No tokens: the DomoticsCore runs of 2026-10-04 kept on this machine,
each one call, its task's characters against the tokens reported.

- **The estimate**: 711 + 0.82 a character (`agent.Tokens`), not
  characters / 4; on these calls, 15 to 20% over what Claude reported.
- **Imports**: tasks of 34k to 81k characters, 29.7k to 65.1k tokens.
- **Refining, with the code**: tasks of 87k to 126k characters, 66.7k
  to 97.6k tokens — all passed the budget of 40000 at characters / 4.
- **The budget**: 120000 now, the largest of them (about 112k estimated)
  still passing; `code-lines-max` (1500) is what sizes it.
- **Not tried**: a real call refused for its size.

## 2026-10-07 — a spec read before ready (#128), live on GitHub

The sandbox's gardening line with the reviewer after the product owner;
Sonnet. Details of the reviewer's side in roles/reviewer/docs/tried.md.

- **Planted**: #44 refined (Verification, Scope), its `ready` held
  (`spec-not-reviewed`); the reviewer's finding given at the next run, its
  own Verification rewritten (`wrote` matched), `answered: 1`; released
  by the engine, no agent. #45's planted `ready` held at the sixth round
  (`spec-rounds-spent`); `workline:accepted` set by hand, ready.
- **Real, run 1** (10.8k tokens in, 0.5k out): #46 named no code; the
  agent set it waiting on #44 and asked, in a note, for its files.
- **Real, run 2** (17.7k in, 0.9k out, two issues), after a person's
  comment naming `Idle`: #46 refined, read by the reviewer, released the
  next run. #12, held: its Verification read as a person's — refined
  before `wrote` existed — the agent asked its reporter the reviewer's
  question, as told.
- **Found and fixed**: a split's children now record their four sections
  as the role's (`wrote`); the issues refined before still read as a
  person's.
- **Prompt**: the findings given clipped at 3000 characters, about 2.3k
  on #12; offline, the task of one issue 14.7k characters, two 25.8k.
- **Side effects**: the planted runs paused the role on the sandbox
  (three runs nobody answered); a comment on the report resumed it.

## 2026-10-07 — an import's plan sees a closed issue, live on GitHub, no agent

A clone of JN0V/workline-sandbox, `IMPORT-163.md` written again from the
bodies of #33, #34 and #35 and committed locally, not pushed;
`workline issues import IMPORT-163.md --forge github --ai none`, without
`--apply`: the forge read, nothing written.

- **Seen**: the task given to the agent lists "#34 Keep the import's log
  for a week — closed as completed; line 13" under "The closed issues
  opened from these lines", its text found in the file; #33 and #35 among
  the open issues.
- **Not tried**: a real agent's answer to it — whether it answers the
  line `held`, #34, rather than proposing it again; the engine would
  leave #34 closed either way.

## 2026-10-07 — milestones ranked by their due date, live on GitHub, no agent

JN0V/workline-sandbox: v0.1.0 given a due date of 2026-12-31, v0.2.0 of
2026-10-31, and #7 (ready, priority 1, no milestone) put in v0.2.0; then
`workline run-role product-owner --event schedule --forge github
--no-apply`, an agent planted to propose nothing: the forge read, nothing
written.

- **Seen**: "next-ready #7" — v0.2.0, due first — where the engine before
  this said #44, of v0.1.0, first by title.
- **Put back**: #7 out of v0.2.0, both dates taken off.
- **Not tried**: GitLab's `due_date` live; a slipped issue moved by date
  live (conformance only); the task's list of milestones with their dates
  live (the run asked no agent: nothing to read).

## 2026-10-07 — a link taken off, a test read as proof, live on GitHub, no agent

JN0V/workline-sandbox, the engine of the branch, `workline run-role
product-owner --event schedule --forge github --ai none` on a clone, twice.
Set up by hand: #44 — which #46 waits on by the role's `depend`, kept in
the record — closed as completed; a parent #47 ("`TestIdleSignOut`
passes.", "The test `TestIdleWarns` tells the user.") with one sub-issue
#48, closed as completed, quoting both; a test file holding
`TestIdleSignOut` committed in the clone only, never pushed.

- **Run 1**: #46's dependency on #44 deleted from GitHub; the report:
  "Took off the link the role set from #46 to #44: #44 is closed: the wait
  the role set is over." #29's link to #28 — closed, set by the split of
  #27 before the parent's state kept `after` — left: the role cannot tell
  it its own. #47 and #48 given their state.
- **Run 2**: #47's comment: "Proved: … the test `TestIdleSignOut` in
  src/auth/idle_test.go, read from the code" and "**Not proved**: … but
  the code holds no test `TestIdleWarns`"; findings `proof-test-missing`
  and `parent-to-accept` (1 item not proved).
- **Put back**: #44 reopened, #46's dependency on #44 added again, the
  report's body and record comment written back as they were, #47 and #48
  deleted, the clone's commit dropped.
- **Not tried**: GitLab — a Premium link deleted, or the engine's line
  taken out of a body there (conformance only); a plugged forge's
  `remove-blocker`; a real agent proposing `undepend` on a link whose
  blocker is open, and a person ticking it live (conformance only); a
  split's `after` kept and taken off live (conformance only).

## 2026-10-07 — the report opens with what to do, live on GitHub, no agent

The redesigned report, read first on workline's own (#110: ten "changed
needs" boxes, nine with nothing to decide; the pause warning mid-page),
then tried twice.

- **#110's record, rendered locally** (the engine of the branch, its
  record and open issues read from GitHub, nothing written): "What to do"
  first — 5 proposals to decide, "answer before the next run, or the role
  pauses", 1 act done alone —; the proposals under #83, #85, #87, #108
  with their titles, "Add Need and Validation to it (Need and
  Validation as drafts for you to correct)"; #108's change said
  beside its title; the nine changes with nothing proposed in one folded
  line, no box. With `archived: [docs/BACKLOG.md]`: no change at all.
- **JN0V/workline-sandbox**, `workline run-role product-owner --event
  schedule --forge github --ai none` on a clone: report #5 rewritten —
  #12's order proposal under its title with the change it was read for,
  the box to set `rename` back to act; GitHub rendered the `<details>`
  folds and the task-list boxes. **Put back**: #5's body as it was; the
  record comment was left unchanged by the run.
- **Why #83, #85, #87 had no act to do**: proposed on 2026-10-04 by an
  engine before #152, when refine started at `propose` and the record
  kept the line alone; carried since, their issues never read again —
  nothing changed in them. Now read again first, and, ticked (as the
  maintainer did), drafted then done (conformance
  `tick-undrafted-drafted-then-done`, `tick-undrafted-waits-for-an-agent`,
  `undrafted-proposal-read-again`).
- **Not tried**: a box ticked inside the new layout live (the keys are
  unchanged, conformance only); GitLab's rendering of the folds; a real
  agent's run settling a change with nothing proposed (conformance
  `changed-import-lines-read-again`, `changed-read-nothing-proposed-settles`).


## 2026-10-08 — ADR-0038 on both sandboxes, planted answers

The engine of the branch, `workline run-role product-owner --event
schedule`, on a clone of JN0V/workline-sandbox; on GitLab, the project
named by `CI_API_V4_URL` and `CI_PROJECT_PATH`, glab's token (JN0V's own,
not the project's bot). The agent was none, or `fake:` with an answer
written for it: no paid agent was called. A trial issue made on each,
labelled `test:adr-0038`.

- **The old report moved**: GitHub #5 and GitLab #10, the first run with
  no agent. Each issue's part of the record moved to its state — acts
  done, a closing on a closed issue —, the reports closed with a link to
  the filter on the label. Found and fixed on the way: GitHub refuses a
  label description over 100 characters (422; now a test); `issue: 0`
  written in each moved act (now left out); the old record's closings
  found wrong were not carried (now undone on their issue).
- **Every issue with drafts labelled**: 11 on GitHub, 7 on GitLab. On
  GitHub this went past `proposals-max` (10): the next run read only the
  answered issues, as built. Workline's own backlog has 7 issues with
  drafts and no proposal in #110's record: under the cap.
- **Completing**: #50 (GitHub) and #33 (GitLab) refined, the label
  `workline:proposed` / `workline::proposed` set, created on GitLab, the
  role's comment saying what it did and what it wants. A Validation
  ending with a line `/close` — and, at the revision, `/label ~bug` —
  written escaped: on GitLab the issue stayed open, no label came, at
  the description's edit as at the note's.
- **Revise**: JN0V's comment on each, read the next run. 👀 on it — on
  GitLab, where JN0V had put 👀 first, the 404 "already taken" read as
  done —, the Validation rewritten in place, the Need kept, one line in
  reply mentioning JN0V, `revisions: 1`, `heard: 1` in the state.
- **Yes**: `workline:accepted` (GitLab: `workline::accepted`, kept beside
  `workline::proposed` on the Free plan) — the next run with no agent
  took the draft lines out, moved the issue to ready, took both labels
  off; the summary said "Accepted by @JN0V", read from the label events.
- **Not now**: `workline:proposed` taken off GitHub #2, #3 and GitLab #7
  by hand; the next run wrote `aside` in their states, the comment saying
  it, the label not put back.
- **Not tried live**: a comment by an outsider or a bot (one account on
  each sandbox: conformance `who-counts-and-rounds`); a third revision
  left to a person (conformance); a real agent revising; an undo found
  live; the week's trial itself. The trial issues were closed after.

## 2026-10-09 — the first nights after ADR-0038, and what they showed

**The night** ([run 37895990993](https://github.com/JN0V/workline/actions/runs/37895990993),
Sonnet, 31,344 tokens in, 1,287 out, five issues read): the maintainer's
new [#275](https://github.com/JN0V/workline/issues/275),
[#276](https://github.com/JN0V/workline/issues/276),
[#277](https://github.com/JN0V/workline/issues/277) — a Need and a
Validation each, no Scope nor Verification — got nothing.

- **Why**: they were read, and the agent answered well — a `refine` each,
  a Scope and a Verification drafted from their Validation. Each Scope
  named its code in words (`internal/doctor`, `roles/auditor`), but the
  `sources` field was empty: the task had given no code, the issues
  linking their files as `https://…/blob/main/…`, which `pre` did not
  read as files. The engine dropped each refine whole
  (`sources-unknown`, "a scope names 1 to 5 files"), Verification
  included, and wrote nothing on the issues: silence.
- **Fixed**: a link to a file on the forge is code the issue names; an
  issue naming none gets the repository's folders listed; a Scope with
  no `sources` has them read from its text; one naming none is left out,
  the rest written, said in plain words; an issue read and left with
  nothing is said.
- **Replayed**: the night's own answer, on a copy of workline at
  `3eebaef` and a fake forge holding the three issues as the task gave
  them, with the branch's engine: #275 and #276 refined (Verification,
  Scope; sources `internal/doctor`, `docs/spec/forge-command.md`,
  `roles/auditor`, `roles/auditor/docs/status.md`), #277's Verification
  written and its Scope left out — "could not tell which files #277 is
  about…: add a Scope naming them, or leave it".
- **The summary** said `sources-unknown #275: a scope names 1 to 5 files
  (sources)` and "(12 proposals)", counting state comments. Now the
  role's line counts issues — "issues: 2 done alone, 1 set aside by a
  person" — and each line is plain, no rule's name.

**The drafts the maintainer triaged** (#81, #83, #87, #88, #108, #109,
written before ADR-0038): a Need rephrasing the item's solution in its
own jargon; a Validation as a formula ("JN0V looks at … and sees …"); a
Scope naming docs, not code; a closed issue cited as a part to wait for;
several topics, some done, drafted as one. The prompt now asks for a Need
from the user's side, a concrete Validation, the project's user-docs
rules, and one plain question rather than a guess; the engine links a
decision cited bare and marks a closed issue cited. A closing with
`close-obsolete` off is said on the issue for a person to close.

**On the sandboxes**, the branch's engine, a `cmd:` agent answering a
written answer (no paid agent):

- GitHub: #51 (a Need and Validation, a blob link) got its Verification
  and Scope, sources `src/auth` read from the Scope; #52's drafts written
  under a hidden mark, "#32 (closed)" marked, the task saying "Cites: #32
  (closed as not planned)"; #53 said done ("Close it yourself if you
  agree…"), left open, `workline:proposed` on; #4, its drafts rewritten
  by JN0V and `workline:proposed` taken off, lost `workline:draft` and
  `workline:to-refine`. A second run with no agent changed nothing.
- GitLab: #34, a `/-/blob/main/` link read as its code, the Scope's file
  taken from its text, "#27 (closed)" marked, the task linking files at
  `/-/blob/HEAD/` (checked: GitLab resolves `HEAD`).
- The trial issues were closed after; #4 keeps JN0V's rewrite.

**Not tried**: the new prompt with a real agent — whether it asks rather
than invents, writes a Need from the user's side and a concrete
Validation; the next real night will say, read by the maintainer.
