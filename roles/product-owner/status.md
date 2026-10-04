# Product owner — where it stands (2026-10-04)

**In one line:** it keeps a backlog on GitHub for real — reads it a share a
run, closes duplicates, proposes what the code made obsolete, names an
issue's code, sets milestones, finds a closing a person undid, refines an
issue to `ready` and asks its reporter what is missing, orders the backlog
and moves what slipped — nightly in
DomoticsCore's CI since v0.5.0 (its roadmap imported, 56 issues), refining
tried live there and on JN0V/workline-sandbox, ordering on the sandbox,
not released yet (tried.md).

## Built and tried with a real agent

| What | Tried on |
|---|---|
| A duplicate closed, its original quoted, with GitHub's own reason | JN0V/workline-sandbox, live (#3 → #2) |
| A closing a person undid found at the next run, that act back to propose, the report saying so; the person's comment read | the same (#3 reopened) |
| An issue the code solved proposed as obsolete, the code quoted | the sandbox (#4); a copy of DomoticsCore, 8 of 38, all right |
| A real backlog read a share a run; an issue read again when its code changed or a person wrote | copies of DomoticsCore; the sandbox |
| A duplicate's original given whole though read before | the sandbox |
| The code an issue is about named (`sources`) | the sandbox (#3); DomoticsCore (BUG-5) |
| Issues put in the next releases' milestones | a copy of DomoticsCore |
| A backlog file, whatever its form, opened as issues (`workline issues import`) | workline's BACKLOG.md, 30 issues, one call |
| A true issue on the code a fix touched left alone; a look-alike left open | evaluation `keeps-a-planted-backlog`, 5 in 5 on Sonnet |
| Refining: Scope and Verification from the code, Need and Validation drafted; the drafts accepted by a person, the issue moved to `ready` by the engine's check; a vague issue's reporter asked, the answer read, the issue refined | JN0V/workline-sandbox, live (#6, #7); DomoticsCore, live (#163, #167, #168, #171, #172) |

| Ordering: one priority label of four set, a person's kept; an issue whose milestone is released moved to the next by the engine, no agent; a fifth of the backlog moved a run, the rest proposed; the order before the run in the report | JN0V/workline-sandbox, live (#2, #4, #6, #7) |

Conformance: tests/conformance/cases/product-owner and backlog. GitHub's
issues, comments, closings and the report tried live; its milestones, and
GitLab's whole backlog, untried live.

## Missing, in the order to build it

1. **Released with refining**: v0.5.0 runs the PO nightly on DomoticsCore
   without it.
2. **Refining, what is left**: splitting a need into sub-issues;
   renaming; asking again after an answer; an outsider's issue proposed
   to its reporter rather than in the report; GitLab's write access.
3. **Obsolete closed, not only proposed**: announced first, closed at the
   next run if nobody answered and a second judge agreed.
4. **The person's hand**: a tick read with its author — a proposal
   accepted, an act set back to `act`; ignored runs pausing it.
5. **One issue-opening mechanism for every role**; the weekly sample over
   its acts; a file the agent was not shown named as a source.
6. **Ordering, what is left**: a forge's native rank (GitLab's reorder, a
   GitHub project's position), deferred (ADR-0018); milestones ranked by
   their due date, not only their title; the engine refusing a person's
   priority, tried in conformance only.
