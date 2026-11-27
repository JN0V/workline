# Diagnosing the developer — measured (2026-11-26/27)

**Question.** The evaluation of 2026-11-25 gave the developer 6 of 10.
Why are its pull requests sent back, and which change avoids the
reasons a careful developer could have avoided? The method is on
[diagnosing-a-role.md](diagnosing-a-role.md).

**Verdict.**

- **8 of 21 send-backs are the agent's craft**; 6 are the engine writing
  `Closes #n … proven` over a gap the agent had said itself.
- **Opus with a checklist states the most**: 16, 16, 15 of 19 reachable
  reasons avoided, against 7, 12, 7 for the same checklist on Sonnet.
  Every run above every run; p = 0.10, the floor of a triple. No winner
  under the criteria: one triple, a baseline of one run.
- **It costs 2.5 times production's fresh tokens.** The owner chose
  reliability over tokens: shipped with a guard
  ([#2304](https://github.com/JN0V/workline/pull/2304), v0.92.0).
- **The engine's part shipped too**: `Part of #n`, never `Closes`, over a
  part left ([#2291](https://github.com/JN0V/workline/pull/2291),
  v0.92.0).
- **Unproven**: the checklist was written from these send-backs; no
  held-out case ran.

## From the records, no agent run

78 pull requests opened 2026-11-23 to 26 (DomoticsCore 37, TraceGlass 6,
workline 35), 133 review rounds.

- **43 rounds pushed no fix**: 19 were woken by the assistant's
  label-trace comment. v0.91.0 asks no round for a comment within 2
  minutes of the same account's accepted label
  ([#2220](https://github.com/JN0V/workline/pull/2220)); replayed on the
  record, all 23 trace-woken rounds would not have run.
- **A person's send-back is answered**: 15 of 18 fixed green in one
  round. The 3 others needed a workflow file, a board measure, or hit a
  merge skew.
- **Reviewer Importants declined**: 0 of 14 were true, fixable and argued
  away; 1 got an empty round.

Causes of the 21 send-backs, one primary each:

| Cause | Count |
|---|---|
| Did not do what the issue says | 4 |
| Regression or side effect unseen | 2 |
| Platform knowledge, or the code it calls not opened | 2 |
| Partial work under `Closes`: in 6 of 6 its own text said what was missing | 6 |
| Round mechanics, merge skew | 3 |
| The issue: wrong, obsolete, an owner's choice, a weak Validation | 4 |

## The bench

- **11 DomoticsCore send-backs replayed**, 19 reasons a developer can
  reach; a person's or a board's part is counted apart.
- **Judged by meaning**, per reason: avoided in the change (`done`), said
  as not done, or neither. One judge for every arm, blind to the arm.
- **The judge checked on known answers first**: the real sent-back head
  must fail each reason, the accepted head pass. Fable agreed on 26 heads
  of 26, but only after its inputs were corrected twice and the expected
  answers three times; its prompt never changed.
- **The judge switched to Opus** for the results: 25 heads of
  26; against Fable on the three runs both read, 54 of 57 verdicts, Opus
  stricter on all 3 differences. On the one Opus run both read, Opus
  marked its own model down (13 → 11).
- **Same commit**: the checklist arms at 273f6429d; baseline and plain
  Opus at c1417ab0f, which differs by the prompt alone.

## Arms, and the owner's cuts

Planned: baseline, a checklist in the prompt, Sonnet at effort high, a
second pass by a fresh call, Opus medium; 6 runs each.

- **Reduced to 3 arms and 3 runs**, before any second-pass or effort-high
  run: those two were never run, nothing is known of them.
- **19:30Z, stop only what is surely worst**: nothing was stopped (one
  run an arm proves nothing); TraceGlass and two cases no developer can
  reach were taken out.
- **About 20:05Z, the baseline dropped** as "clearly too weak": it rests
  on one run (4 of 19).
- **About 20:07Z, Opus with the checklist added**, 3 runs; plain Opus
  kept as a one-run reference.

The checklist (7 points): each sentence of the issue done and tested on
the case it names; a new test red without the change and run by the
test command; callers and callees opened; uses elsewhere brought in
line; the whole suite run last; what worked before kept; what is left in
`Not done:`.

## Results

| Arm | Reasons avoided of 19, per run | Fresh tokens a case | Cache read a case | Drafts of 11, per run |
|---|---|---|---|---|
| Sonnet medium, production's prompt | 4 | 28.4k | 259k | 10 |
| Sonnet medium, checklist | 7 · 12 · 7 | 34.6k | 313k to 491k | 9 · 5 · 2 |
| Opus medium, production's prompt | 11 | 54.9k | 707k | 10 |
| Opus medium, checklist | 16 · 16 · 15 | 70.5k | 890k to 1 264k | 2 · 0 · 2 |

- **Tokens**: Opus with the checklist is 2.04 times Sonnet with it, 2.5
  times production, 1.3 times plain Opus.
- **Wall**: 3 772 to 12 551 s a run of the set; not compared, the
  machine's load differed by arm.
- **Noise**: Sonnet with the checklist spreads 7 to 12, Opus with it 15
  to 16. A triple cannot show a gap below about 5 on Sonnet.
- **Where the gain sits**: the checklist makes Opus run the whole suite
  (3 → 11 of 11) and open the docs (1 → 6 to 9); on Sonnet it barely
  moves either.
- **Reached by no arm**: wiring a new test into what CI runs (dc-252),
  and testing the case the issue names when a neighbour's test exists
  (dc-262: Opus with the checklist fixes the code 3 of 3, tests it 0).

## What shipped

- **[#2304](https://github.com/JN0V/workline/pull/2304)**: Opus at medium
  and the checklist, no Go. The guard: the first 10 pull requests are
  watched for send-backs, proof time and fresh tokens; one setting rolls
  it back ([the developer's model](../../roles/developer/docs/agent.md#the-model)).
  The skeptic's review asked the owner to re-confirm at 2.5 times, not
  the "about 2 times" first said.
- **[#2291](https://github.com/JN0V/workline/pull/2291)**: a part left
  makes the pull request `Part of #n` with `## Left for a person` on top.

## Held out, and unproven

- **No held-out case**: each checklist bullet maps to a named send-back.
  The gain is an upper bound. The skeptic's cheapest check, Opus with the
  checklist and production on 3 or 4 send-backs from after 2026-11-26,
  was not run: no send-back came after 2026-11-26 to run it on.
- **The checklist's own effect** on either model rests on one run each
  (baseline, plain Opus).
- **Not measured**: scope creep ("bring each use in line" invites wider
  diffs), answers to the reviewer in rounds, the whole native suite
  against the proof's time cap on DomoticsCore.
- **What would refute it**: a send-back rate on these pull requests not
  below the 2026-11-23 to 26 window, a checklist cause coming back, or
  fresh tokens a pull request above 2.5 times.
