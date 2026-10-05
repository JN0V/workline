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
(roles/reviewer/tried.md): a subject held by an open issue not opened
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
