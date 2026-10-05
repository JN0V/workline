# Product owner — where it stands (2026-10-05)

**In one line:** it keeps a backlog on GitHub, and on GitLab as on GitHub
(its members told from outsiders, a bot's token), for real — reads it a share a
run, closes duplicates, announces what the code made obsolete and closes
it a week later on silence and a second judge's yes, names an issue's
code, sets milestones, finds a closing a person undid, does what a
person of the project ticks in its report and pauses when nobody answers, goes as far
as its autonomy level says — cautious, normal or enterprising — and
demotes a kind of act a person undid, refines an
issue to `ready` and talks with its reporter until it is, splits a need
into sub-issues — then says on the parent, as they close, what each
delivered and which of its Verification is proved, for a person to
accept — and renames a vague title, orders the backlog
— an issue that waits on another after it, never offered first — and
moves what slipped, opens its report with what is next and what is
stuck, flags the issues built on a need a person rewrote — its parts
read again, every act on them proposed, a ready one moved back to
refine only on a person's tick —, and keeps the one way every role opens an issue —
nightly in DomoticsCore's CI (v0.9.0 there since its #201, refining and
ordering included, the reviewer not enabled: its roadmap imported, 56
issues), refining tried live there and on JN0V/workline-sandbox, ordering
on the sandbox (tried.md).

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
| The import's map (ADR-0030): each item to its issue — opened, already open, closed — or its reason checked — done, the words quoted; not an item; past the cap —, what is left listed as not covered, its share flagged, the import ending for a person (exit 2) | JN0V/workline-sandbox, live on GitHub, the agent's answer planted (#33 to #35: opened, then already open, then #34 closed); DomoticsCore's roadmap without an agent, 32 shares, 560 paragraphs not covered; conformance `import-map*`, `import-flags-*`. Not with a real agent's answer yet |
| A true issue on the code a fix touched left alone; a look-alike left open | evaluation `keeps-a-planted-backlog`, 5 in 5 on Sonnet |
| Refining: Scope and Verification from the code, Need and Validation drafted; the drafts accepted by a person, the issue moved to `ready` by the engine's check; a vague issue's reporter asked, the answer read, the issue refined | JN0V/workline-sandbox, live (#6, #7); DomoticsCore, live (#163, #167, #168, #171, #172) |

| Ordering: one priority label of four set, a person's kept; an issue whose milestone is released moved to the next by the engine, no agent; a fifth of the backlog moved a run, the rest proposed; the order before the run in the report | JN0V/workline-sandbox, live (#2, #4, #6, #7) |
| One way for every role to open an issue: a key per subject made by the engine, looked for in the issues open and closed — open, left; closed as not planned or duplicate, left; closed as done, said once — the role named, `needs-triage`, a cap a run; a role's issue read as its draft to refine | JN0V/workline-sandbox, live with the reviewer (#9 found again open, then closed: said once); cases in tests/conformance/cases/backlog (`issue-*`) and product-owner |
| The conversation with the reporter (ADR-0021): asked again only after an answer, the conversation shown to the agent, never the same question twice, three rounds then the report; an outsider's issue proposed to its reporter in a comment, written in the body once a person sets `workline:accepted` | JN0V/workline-sandbox, live (#3: the answer read, refined from it); an outsider's issue with a real agent on a simulated forge (two proposals, then agreed, ready); the rest in conformance |

| Splitting and renaming: a need too big for one issue split into children with their four sections, opened through the one way, linked as sub-issues (GitHub) or tasks (GitLab); a vague title renamed; a rerun splitting and renaming nothing again; a person's title kept | JN0V/workline-sandbox on GitHub (#10 → #12, #13; #11) and on GitLab (#5 → tasks #7, #8, #9; #6), live (ADR-0022) |
| GitLab as GitHub (ADR-0023): its members from the Planner role up are of the project, a Guest's issue proposed to its reporter; the project's bot writes with a project access token made and stored in one glab command; a state note another token wrote written anew; the job token alone refused, loud. A reply `agreed` by the reporter or a person of the project has the proposed text written (ADR-0021, amended); each comment given to the agent with who wrote it, on every forge | JN0V/workline-sandbox on GitLab, live and in GitLab CI (#13 renamed and refined in place, #15 a Guest's proposed, agreed by a reply, readied by the label; #16 split); conformance `forge/gitlab-*`, `product-owner/*-reply-*`, `comment-author-read` |

| Obsolete announced, then closed (ADR-0024): a comment to the reporter quoting the code and the label `workline:obsolete`; at a run `days` later, nobody having written and the label still there, a second judge of another model asked apart, the issue closed as completed with its yes and level; a reply keeps it open for good on that quote | JN0V/workline-sandbox on GitHub (#14 closed, #4 kept) and on GitLab (#19 closed, #20 kept), live, the delay set to 0; label removal, exempt labels, a judge's no, the cap in conformance (`obsolete-*`) |
| Autonomy levels (ADR-0026): `cautious` doing what checks facts — sources, Scope and Verification — and proposing what sets direction — a duplicate, milestones, priorities, Need and Validation drafts, a split —; `enterprising` closing, moving and refining whole; each kind's mode and where it comes from in the task and the report. An act a person undid found at the next run with no agent — a priority put back, a title renamed back — and its kind demoted whatever the level | JN0V/workline-sandbox on GitHub, live (#15–#19 cautious, #20–#24 enterprising; the rename planted); `ready` taken off, a split's child closed as not planned, `ignored-runs-max`, init's question in conformance |
| The person's hand (ADR-0025): a box in the report ticked by a person of the project — who ticked it read from GitHub's edit history, GitLab's system notes — done at the next run as the record keeps it, no agent; an outsider's, a bot's or an unknown author's tick said, not done; a kind back to propose set back to act by a tick; three runs nobody answered pause the agent, a person's act resumes it | JN0V/workline-sandbox on GitHub (#7 renamed, `close-duplicate` back to act, a paused run asking no agent) and on GitLab (#14 renamed), live, planted, no agent; the rest in conformance (`tick-*`, `back-to-act-by-a-tick`, `*-pause`, `paused-*`, `person-acts-resumes`) |

| A parent and its parts (ADR-0029): one comment on the parent, with no agent, edited in place and not at all when unchanged — each part open, closed as completed with the commit, pull or merge request that closed it, or not delivered; each Verification item proved by a quote or "not proved"; all closed, a person asked to accept by closing it, the report's "To accept"; the role never closes a parent; a parent never offered first | JN0V/workline-sandbox on GitHub (#30: a commit, a part not planned, a real agent's run; #10, #27) and on GitLab (#25: a commit, merge request !4; #5, #16), live; conformance `parent-*`, `forge/gitlab-parent-closers` |

| What an issue waits on (ADR-0028): `depend` and a split's child's `after` written in GitHub's dependencies, or a marked `Blocked by #n.` line in the body on GitLab Free; read back with a person's links and lines; a blocked issue ordered after its open blockers, never `next-ready`; a closed blocker unblocking; the report's "Waiting" | JN0V/workline-sandbox on GitHub (#29 after #28 by a split, #26 and #29 on #25) and on GitLab (#23, #24 on #22, a body line; #22 closed then reopened), live; cycles, a person's link kept, the level, an undo in conformance (`depend-*`, `backlog/*blocker*`, `cycle-*`, `undo-depend-demotes`) |

| The report opens with what is next and what is stuck (ADR-0031), no agent: the first `next-max` ready issues of the order waiting on nothing, with milestone and priority; each issue waiting on a person past `stuck-days` — ready with no pull request nor commit since the label, its reporter not answering, a proposal unticked, an announcement due and unjudged —, with since when; an issue once; rebuilt each run, the report rewritten when it changed | JN0V/workline-sandbox on GitHub (report #5: Next #7, #6; #6's label day read from the timeline) and on GitLab (report #10: the label days of #6, #13, #15 from its label events, #15's question from its note's day), live; workline itself without applying (Next #79; #65, #91, #92 asked today; #83, #85, #87 proposed); conformance `report-*` |

| A changed need (ADR-0032): a person's rewrite of an issue's Need or Scope found with no agent against what its state kept; its parts read again first with the text as it was and as it is, every act on them proposed (`need-changed`); `unready`, back to refine, always proposed, done on a person's tick; the issues waiting on it — and, its Scope changed, those on the same code — listed; the report's "Changed needs", a box to tick once checked; the same change flagged once | JN0V/workline-sandbox on GitHub (#10's Need rewritten: #12, #13 read again, a planted answer; #13 moved back to refine by a tick), live, no agent called; an imported file's lines changed, the tick on a change, an edit outside the sections in conformance (`changed-*`, `unchanged-need-flags-nothing`, `unready-ticked-moves-back`) |

Conformance: tests/conformance/cases/product-owner and backlog. GitHub's
issues, comments, closings and the report tried live; its milestones untried
live. GitLab: refining, ordering, a split and a rename tried live on its
sandbox, three runs, then members, a bot's token and a reply's
agreement, four runs and GitLab CI (tried.md); its milestones there not
checked.

## Missing, in the order to build it

1. **Refining, what is left**: a real outsider's issue tried live, and
   a reporter's own `agreed`; a person of the project's reply taken for
   the label, if live use asks for it (ADR-0021, amended); a GitLab role
   too low reported as such, not only by the split's fallback
   (ADR-0023); the gardening template on GitLab run with a release
   holding this.
2. **Obsolete, what is left** (ADR-0024): a real delay of days waited
   for; the label taken off, an exempt label, a judge's no tried live; the
   weekly sample over these closings; 7 days and 3 a run measured.
3. **The person's hand, what is left** (ADR-0025): a tick by an
   outsider or a bot tried live (one account on each sandbox); a
   proposal of a real agent's run ticked, not one planted; the pause
   reached by three real runs; a tick on a plugged forge.
4. **Autonomy, what is left** (ADR-0026): the weekly sample over its
   acts recording the level, and its suggestions from it (only the
   report's, from the proposals settled at cautious, is built); a
   suggestion for enterprising from its acts undone; a rename by a real
   agent undone, `ready` taken off and a split's child closed live;
   `workline init`'s question on a terminal; an undone split on GitLab,
   which keeps no reason for a closing.
5. **The weekly sample over its acts**; a file the agent was not shown
   named as a source. The one way to open issues is built (ADR-0018,
   amended); left: its import's plan does not yet see a closed issue
   (the engine leaves it closed, said, and the import's map names it).
   **The import's map, what is left** (ADR-0030): a real agent's import
   answered with `skip`s — DomoticsCore's roadmap again, the map read by
   the maintainer against its 56 issues, each reason agreed with, none
   missing; the tokens a `skip` a line costs, measured.
6. **Ordering, what is left**: a forge's native rank (GitLab's reorder, a
   GitHub project's position), deferred (ADR-0018); milestones ranked by
   their due date, not only their title; the engine refusing a person's
   priority, tried in conformance only.
7. **A parent, what is left** (ADR-0029): an agent drafting which
   Verification items look proved in other words, if live use asks; a
   test named as proof, read from the code; a part of another repository;
   a plugged forge's `closers` tried.
8. **What an issue waits on, what is left** (ADR-0028): a GitLab Premium
   link tried live; a relation across projects; removing a link the role
   set once its reason is gone; the Forgejo sample's `add-blocker`; the
   developer role taking the `next-ready` issue (#117).
9. **Next and stuck, what is left** (ADR-0031): an issue truly stuck
   past 14 days seen live (the sandboxes' days are all this week's); the
   maintainer reading #110 after a run on workline (#164's Validation);
   the Forgejo sample's `trail`; a pull request closed unmerged still
   counting as started.
10. **A changed need, what is left** (ADR-0032): a real agent's answer on
   the parts read again — whether it proposes `unready` when it should,
   and only then; an imported file's lines changed, tried live (the
   sandbox's imported file was never pushed); GitLab; a change to
   Verification or Validation, if live use asks; the maintainer reading
   the report against #165's Validation.
