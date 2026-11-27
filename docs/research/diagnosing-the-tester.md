# Diagnosing the tester — measured (2026-11-26/27)

**Question.** The evaluation of 2026-11-25 gave the tester 6 of 10: its
test plans held weak tests and false "nothing guards it" claims. What is
left at v0.91.0, and which change proves more scenarios without losing
the kept ones? The method is on [diagnosing-a-role.md](diagnosing-a-role.md).

**Verdict.**

- **v0.91.0 had already fixed about two thirds**: 13 to 15 of 22 target
  scenarios proved, against 0 in the original Drafts.
- **`mend`**, a refusal asked again with its own reason, up to three
  times: runs with no Draft fall from 7 to 1 of 45, at the same tokens;
  no gain on proved scenarios. Shipped
  ([#2306](https://github.com/JN0V/workline/pull/2306), v0.92.0).
- **`vm`**, `mend` plus an Opus second reader asking "what cheap wrong
  change would this test still pass?": 17, 18, 16 of 22, every run above
  every baseline run, at 1.87 times the fresh tokens. A provisional
  winner on one triple; not shipped.
- **Four of the five arms predicting a target gain did not gain**: a
  shell, a prompt, Opus as the writer, three retries.

## The set

- **24 replay cases**, one per issue whose Draft held a bad scenario,
  2026-11-23 to 25 (10 DomoticsCore, 4 TraceGlass, 10 workline; 69
  scenarios), in [#2295](https://github.com/JN0V/workline/pull/2295),
  open.
- **Arms compare on 15 of them**, fixed at 15:35Z: 22 target scenarios
  (not proved in the original Draft), 23 kept, 3 flagged and reported
  apart. 4 heavy workline cases were not played on this machine.
- **The originals, read by 3 independent readers**: all 24 "kept as
  today" claims are true of the code; in 10 (partly 4 more) an existing
  test already guarded them, where the Draft said none did. The engine
  wrote that sentence after a grep for a test name.

Causes of the 37 bad scenarios of the originals, one primary each:

| Cause | Scenarios |
|---|---|
| Engine rule (red first) and the instruction "write none" | 8 |
| Existing guard not found | 14 (10 + 4 partly) |
| Model or instruction: the easier path, the letter of the Then | 6 |
| Needs hardware or a forge run | 4 |
| The Validation itself weak (the product owner's) | 3 |
| Toolchain: the runner cannot run the proving test | 2 |

## Judges

- **Fable** first; **Opus** from 21:19Z, when Fable's allowance reached
  94 %. Every compared run was judged again by Opus.
- Fable and Opus agree on 235 of 240 scenario-runs; Opus is 1 to 2
  points more lenient on targets. Arms are compared inside one judge.
- **Sonnet re-judge** on 2026-11-27, the skeptic's request (in `vm` the
  second reader and the judge were both Opus): 423 of 432 scenario-runs
  agree (98 %).

## Arms, predictions and results

Predictions written at 15:46Z, before any arm result; amendments before
the runs they touch. Points of 22 (target) and 23 (kept).

| Arm | What it changed | Predicted | Found (judge) |
|---|---|---|---|
| `shell` | Bash in its copy, runs the tests | +3 to +5 | 8 target: −4 to −5, time × 2.4 (Fable) |
| `prompt` | 3 bullets: runner and build, answer format, fail on a half fix | +2 to +4 | 12: −1 to 0 (Fable) |
| Opus as writer | the model | +1 to +3 | 11, kept 8, 8 cases with nothing delivered; 139k fresh (2.1×) (Fable) |
| `retry3` | 3 asks instead of 1 shared | +2 to +3 | 10: −2 to −3 (Fable) |
| `mend` | `retry3` plus 3 refusals made mendable | nothing delivered 0–1 of 15 | 1, 0, 0 (Opus) |
| `vm` | `mend` plus an Opus second reader | 15–17, × 1.9 tokens | 17, 18, 16; × 1.87 (Opus) |

Opus judge, 3 runs an arm:

| Arm | Target of 22 | Kept of 23 | Nothing delivered of 15 | Fresh tokens a case run | Agent time a case |
|---|---|---|---|---|---|
| Originals (v0.89, v0.90) | 0 | 23 | 0 | — | — |
| Baseline v0.91.0 | 13, 15, 14 | 20, 16, 17 | 2, 3, 2 | 64.7k | 81 s |
| `mend` | 13, 16, 14 | 17, 23, 20 | 1, 0, 0 | 63.7k | 86 s |
| `vm` | 17, 18, 16 | 20, 20, 18 | 1, 1, 2 | 121k | 165 s |

- **Under Sonnet**: base 12, 14, 13; `mend` 13, 16, 13; `vm` 17, 17, 17.
  `vm` vs base +4.0, vs `mend` +3.0 (p 0.10 each, the floor of 3 against
  3). The self-judging doubt is settled.
- **Noise**: the baseline spreads 2 on targets, 4 on kept. `vm`'s +3.0
  over the baseline is just beyond it; `mend`'s target and kept changes
  are inside it.
- **What `vm` fixes beyond `mend`**: DomoticsCore #157's stale literal and
  TraceGlass #6's bounds, weak in every `mend` run. Still weak in both:
  DomoticsCore #158, #189, TraceGlass #5's rotation (a Validation point).
- **`vm`'s fear did not show**: offline it objected to 11 of 30 sound
  tests; in the runs, kept weak-path verdicts were 0 of 69.
- **A no-agent arm**, each hunk of the real fix reverted alone: as
  predicted, it flags grep tests on multi-hunk fixes, is blind to the
  easier path and to a missing scenario, and false-alarms on 2 clean
  fixes of 4.

## Cuts, and why

- **19:55Z, the owner (the subscription spent too fast): stop an arm
  after run 1 when it is not above the baseline's range.** `shell`,
  `prompt` and `retry3` stopped at one run.
- **20:40Z, the owner: `retry3` stopped, find another approach on the
  residual causes, 3 runs each.** Opus as the writer stopped as surely
  worse. `mend` and `vm` were then reasoned from the failing cases.
- **Never run**: Sonnet at effort xhigh, the baseline's second triple,
  the held-out issues (`forensics/heldout.json`), a shadow run.

## What shipped

- **[#2306](https://github.com/JN0V/workline/pull/2306)**: `mend`, built
  apart from the bench's patch (a re-ask helper, net −6 lines of Go). Its
  own real run on dc-181, which delivered nothing in every bench run on
  v0.91.0: refused twice, asked again, Draft opened, the case's checks
  3 of 3; 169k fresh tokens over 3 calls.
- **`vm`**: not built. The skeptic: "not yet"; it adds engine Go against
  the code budget until 1.0, and doubles agent time.

## Held out, and unproven

- **`vm` stands on one triple** of the cases it was designed on; no
  held-out issue and no shadow run.
- **Engine causes left, at 0 tokens**: one test seen under two ids
  (`declared-test-ambiguous`, dc-146); the hint offering
  `proof: existing-suite` after a green-on-base refusal; the engine's
  "nothing guards it" after a name grep.
- **Weak Thens** (DomoticsCore #157, #158, TraceGlass #5) are the product
  owner's Validation, not the tester's.
