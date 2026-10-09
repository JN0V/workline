# Product owner — where it stands (2026-10-09)

**In one line:** refocused by
[ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
— it completes issues (sources, drafts, a question to the reporter),
closes an evident duplicate and readies an evident issue alone, proposes
the rest on each issue with the label `workline:proposed` and one comment
edited in place, and reads a person's answer there: `workline:accepted`,
a comment to revise, the label taken off, a closing. No report issue: the
first run moves an old report's record to the issues and closes it; the
night's summary is the CI job's. Milestones, priorities, splits, renames,
links, obsolete issues, changed needs, the weekly sample are off by
default, their code kept. Built and tried on both sandboxes with planted
answers (tried.md, 2026-10-08); a week's trial on workline and the
maintainer's GitLab project, at most five issues a night, is running.
Its first nights led to the amendment of 2026-10-09: drafts written for
a reader and never invented, a Scope's files read from its text, a
closing said for a person when `close-obsolete` is off, drafts marked
unseen, labels following the state, the summary in plain words
(tried.md, 2026-10-09).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: nearly. What is left is mostly tries
   on real use; the forge's native rank (deferred by ADR-0018) and the
   items "if live use asks" are to park as issues.
2. **The main issue validated on real use**: open. The maintainer found
   the report ([#110](https://github.com/JN0V/workline/issues/110))
   unreadable; ADR-0038 replaced it with proposals on each issue. Its
   week's trial measures them: accepted unchanged, edited, revised, set
   aside, ignored.
3. **AI verdicts measured**: in part. `keeps-a-planted-backlog` passed 5
   in 5 on Sonnet; closing as obsolete, refining, splitting and renaming
   are not measured; a real week's sample of its acts is not yet read
   (Missing 5).

## Built and tried with a real agent

| What | Tried on |
|---|---|
| Proposals on each issue (ADR-0038): `workline:proposed` (GitLab `workline::proposed`) and one comment of the role, created once then edited in place, saying what it did, proposes and wants; `workline:accepted` a yes, the role taking `proposed` off itself (both kept on GitLab Free); a person's comment read as "revise" — 👀, the role's own drafts rewritten in place, one line in reply, two revisions at most —; `proposed` taken off a "not now" for that issue; a report's record moved to the issues and the report closed; no line of its text starting with `/` | JN0V/workline-sandbox on GitHub (#50; report #5 moved, closed; #2, #3 set aside) and on GitLab (#33, `/close` and `/label` escaped in a body and a note edit, not run; report #10 moved, closed; #7 set aside), live, planted answers, no paid agent (tried.md); conformance `proposal-*`, `accepted-*`, `revise-*`, `who-counts-*`, `proposed-taken-off-*`, `report-moves-*`, `no-quick-action-*`, `scoped-labels-*`, `undone-*`, `outsider-evidence-*`, `deleted-section-*` |
| What the first nights showed (ADR-0038, amended 2026-10-09): a Scope's files read from its text, a Scope naming none left out and the rest written, an issue read and left with nothing said; a file linked on the forge given as the issue's code, the repository's folders listed when an issue names none; the issues an issue cites said open or closed, a decision cited bare linked and a closed issue cited marked "(closed)"; a closing with its kind off said on the issue for a person to close; drafts marked by a hidden line, an edited draft a person's; an import's origin hidden; labels following the state; the summary counted by issue, in plain words | the night's own answer on [#275](https://github.com/JN0V/workline/issues/275) to [#277](https://github.com/JN0V/workline/issues/277) replayed on a copy of workline (a fake forge); JN0V/workline-sandbox on GitHub (#51 to #53, #4 set aside) and on GitLab (#34), live, planted answers, no paid agent (tried.md); conformance `scope-files-*`, `linked-file-*`, `folders-listed-*`, `edited-draft-*`, `labels-follow-*`, `obsolete-said-when-off`, `draft-links-*`, `summary-in-plain-words`. The prompt's writing rules not yet with a real agent |
| A duplicate closed, its original quoted, with GitHub's own reason | JN0V/workline-sandbox, live (#3 → #2) |
| A closing a person undid found at the next run, that act back to propose, the report saying so; the person's comment read | the same (#3 reopened) |
| An issue the code solved proposed as obsolete, the code quoted | the sandbox (#4); a copy of DomoticsCore, 8 of 38, all right |
| A real backlog read a share a run; an issue read again when its code changed or a person wrote | copies of DomoticsCore; the sandbox |
| A duplicate's original given whole though read before | the sandbox |
| The code an issue is about named (`sources`) | the sandbox (#3); DomoticsCore (BUG-5) |
| Issues put in the next releases' milestones | a copy of DomoticsCore |
| A backlog file, whatever its form, opened as issues (`workline issues import`) | workline's BACKLOG.md, 30 issues, one call |
| The import's map (ADR-0030): each item to its issue — opened, already open, closed — or its reason checked — done, the words quoted; not an item; past the cap —, what is left listed as not covered, its share flagged, the import ending for a person (exit 2) | JN0V/workline-sandbox, live on GitHub, the agent's answer planted (#33 to #35: opened, then already open, then #34 closed); DomoticsCore's roadmap without an agent, 32 shares, 560 paragraphs not covered; conformance `import-map*`, `import-flags-*`. Not with a real agent's answer yet |
| The import's plan sees the closed issues an import opened from its share's lines — found by their text in the file, else by the lines their body names — each with how it was closed, for the agent to answer `held` rather than propose again (ADR-0018, amended) | JN0V/workline-sandbox on GitHub, read live without `--apply`, no agent: #34, closed as completed, listed at line 13 of `IMPORT-163.md` (tried.md); conformance `import-sees-a-closed-issue`. Not with a real agent |
| An import split as in CI (#174): judged without `--apply` with a read token, its runs listed as pending; `workline apply --line`, with the write token and no agent, opens them; the map and the pending runs in `--summary` | JN0V/workline-sandbox on GitLab: a pipeline of two jobs started through the API (the judge with a `read_api` token, the apply with the write token), #30 and #31 opened; locally the same, #28 and #29, the import again mapping them already open; the answer planted, no agent; conformance `import-judged-then-applied`. Not in GitHub Actions; not with a real agent |
| A true issue on the code a fix touched left alone; a look-alike left open | evaluation `keeps-a-planted-backlog`, 5 in 5 on Sonnet |
| Refining: Scope and Verification from the code, Need and Validation drafted; the drafts accepted by a person, the issue moved to `ready` by the engine's check; a vague issue's reporter asked, the answer read, the issue refined | JN0V/workline-sandbox, live (#6, #7); DomoticsCore, live (#163, #167, #168, #171, #172) |

| Ordering: one priority label of four set, a person's kept; an issue whose milestone is released moved to the next by the engine, no agent; a fifth of the backlog moved a run, the rest proposed; the order before the run in the report | JN0V/workline-sandbox, live (#2, #4, #6, #7) |
| Milestones ranked by their due date, a dated one before one without, then by title in version order — in the backlog's order, Next, the move of what slipped and the task's list of milestones; read from GitHub's `due_on`, GitLab's `due_date`, a plugged forge's `due` | JN0V/workline-sandbox on GitHub, read live with `--no-apply`, nothing written: v0.2.0 given an earlier date than v0.1.0, #7 put in it — Next #7 before #44, the reverse of the titles; dates and milestone taken off after (tried.md); conformance `slipped-milestone-moved-by-due-date`, `report-next-milestones-by-due-date`. Not on GitLab |
| One way for every role to open an issue: a key per subject made by the engine, looked for in the issues open and closed — open, left; closed as not planned or duplicate, left; closed as done, said once — the role named, `needs-triage`, a cap a run; a role's issue read as its draft to refine | JN0V/workline-sandbox, live with the reviewer (#9 found again open, then closed: said once); cases in tests/conformance/cases/backlog (`issue-*`) and product-owner |
| The conversation with the reporter (ADR-0021): asked again only after an answer, the conversation shown to the agent, never the same question twice, three rounds then the report; an outsider's issue proposed to its reporter in a comment, written in the body once a person sets `workline:accepted` | JN0V/workline-sandbox, live (#3: the answer read, refined from it); an outsider's issue with a real agent on a simulated forge (two proposals, then agreed, ready); the rest in conformance |

| Splitting and renaming: a need too big for one issue split into children with their four sections, opened through the one way, linked as sub-issues (GitHub) or tasks (GitLab); a vague title renamed; a rerun splitting and renaming nothing again; a person's title kept | JN0V/workline-sandbox on GitHub (#10 → #12, #13; #11) and on GitLab (#5 → tasks #7, #8, #9; #6), live (ADR-0022) |
| GitLab as GitHub (ADR-0023): its members from the Planner role up are of the project, a Guest's issue proposed to its reporter; the project's bot writes with a project access token made and stored in one glab command; a state note another token wrote written anew; the job token alone refused, loud. A reply `agreed` by the reporter or a person of the project has the proposed text written (ADR-0021, amended); each comment given to the agent with who wrote it, on every forge | JN0V/workline-sandbox on GitLab, live and in GitLab CI (#13 renamed and refined in place, #15 a Guest's proposed, agreed by a reply, readied by the label; #16 split); conformance `forge/gitlab-*`, `product-owner/*-reply-*`, `comment-author-read` |

| Obsolete announced, then closed (ADR-0024): a comment to the reporter quoting the code and the label `workline:obsolete`; at a run `days` later, nobody having written and the label still there, a second judge of another model asked apart, the issue closed as completed with its yes and level; a reply keeps it open for good on that quote | JN0V/workline-sandbox on GitHub (#14 closed, #4 kept) and on GitLab (#19 closed, #20 kept), live, the delay set to 0; label removal, exempt labels, a judge's no, the cap in conformance (`obsolete-*`) |
| Autonomy levels (ADR-0026): `cautious` doing what checks facts — sources, Scope and Verification — and proposing what sets direction — a duplicate, milestones, priorities, Need and Validation drafts, a split —; `enterprising` closing, moving and refining whole; each kind's mode and where it comes from in the task and the report. An act a person undid found at the next run with no agent — a priority put back, a title renamed back — and its kind demoted whatever the level | JN0V/workline-sandbox on GitHub, live (#15–#19 cautious, #20–#24 enterprising; the rename planted); `ready` taken off, a split's child closed as not planned, `ignored-runs-max`, init's question in conformance |
| *Removed by ADR-0038, with the report issue:* the person's hand (ADR-0025): a box in the report ticked by a person of the project — who ticked it read from GitHub's edit history, GitLab's system notes — done at the next run as the record keeps it, no agent; an outsider's, a bot's or an unknown author's tick said, not done; a kind back to propose set back to act by a tick; three runs nobody answered pause the agent, a person's act resumes it | JN0V/workline-sandbox on GitHub (#7 renamed, `close-duplicate` back to act, a paused run asking no agent) and on GitLab (#14 renamed), live, planted, no agent; the rest in conformance (`tick-*`, `back-to-act-by-a-tick`, `*-pause`, `paused-*`, `person-acts-resumes`) |

| A parent and its parts (ADR-0029): one comment on the parent, with no agent, edited in place and not at all when unchanged — each part open, closed as completed with the commit, pull or merge request that closed it, or not delivered; each Verification item proved by a quote or "not proved"; all closed, a person asked to accept by closing it, the report's "To accept"; the role never closes a parent; a parent never offered first | JN0V/workline-sandbox on GitHub (#30: a commit, a part not planned, a real agent's run; #10, #27) and on GitLab (#25: a commit, merge request !4; #5, #16), live; conformance `parent-*`, `forge/gitlab-parent-closers` |
| A test named as proof read from the code (ADR-0029, amended): an item naming a test in a code span — a test file, `path::name`, `TestX`, or a name after "test", "test case" or "conformance case" — proved only when the code at the run's commit holds it, the file said; one not there leaves the item not proved, `proof-test-missing`; git failing said apart, `proof-test-unread` | JN0V/workline-sandbox on GitHub, no agent (#47, #48 made for it, deleted after: one test found in a commit of the clone, one missing); conformance `parent-test-named-read-from-code`. Not on GitLab |

| What an issue waits on (ADR-0028): `depend` and a split's child's `after` written in GitHub's dependencies, or a marked `Blocked by #n.` line in the body on GitLab Free; read back with a person's links and lines; a blocked issue ordered after its open blockers, never `next-ready`; a closed blocker unblocking; the report's "Waiting" | JN0V/workline-sandbox on GitHub (#29 after #28 by a split, #26 and #29 on #25) and on GitLab (#23, #24 on #22, a body line; #22 closed then reopened), live; cycles, a person's link kept, the level, an undo in conformance (`depend-*`, `backlog/*blocker*`, `cycle-*`, `undo-depend-demotes`) |
| A link the role set taken off once its reason is gone (ADR-0028, amended): its own told from a person's by its record, a split's `after` in the parent's state and the engine's marked line; a closed blocker's link taken off by the engine with no agent, `blocker-not-delivered` when it closed without delivering; one whose blocker is open proposed by the agent (`undepend`), done on a person's tick; a person's never | JN0V/workline-sandbox on GitHub, no agent (#46's dependency on #44, #44 closed then reopened, the link put back; #29's to #28, a split's from before, left); conformance `undepend-*`, `split-child-waits-on-sibling`. Not on GitLab live; not with a real agent's `undepend` |

| *Said in the job's summary since ADR-0038:* what is next and what is stuck (ADR-0031), no agent: the first `next-max` ready issues of the order waiting on nothing, with milestone and priority; each issue waiting on a person past `stuck-days` — ready with no pull request nor commit since the label, its reporter not answering, a proposal unticked, an announcement due and unjudged —, with since when; an issue once; rebuilt each run, the report rewritten when it changed | JN0V/workline-sandbox on GitHub (report #5: Next #7, #6; #6's label day read from the timeline) and on GitLab (report #10: the label days of #6, #13, #15 from its label events, #15's question from its note's day), live; workline itself without applying (Next #79; #65, #91, #92 asked today; #83, #85, #87 proposed); conformance `report-*` |
| *Removed by ADR-0038, with the report issue:* the report's top, what to do (ADR-0031, amended): the proposals to decide and the changes to check counted, the pause warned a run ahead, each proposal under its issue's title in plain words, the rest folded; a change read again with nothing proposed settled with no box; a file `archived` flags nothing (ADR-0032, amended); a ticked proposal an older engine recorded without its act drafted, then done (ADR-0025, amended) | #110's real record rendered locally; JN0V/workline-sandbox on GitHub (report #5, `--ai none`, put back), live (tried.md); conformance `changed-*`, `archived-*`, `report-*`, `*undrafted*`. Not on GitLab; the drafting of an old tick not with a real agent |

| A changed need (ADR-0032): a person's rewrite of an issue's Need or Scope found with no agent against what its state kept; its parts read again first with the text as it was and as it is, every act on them proposed (`need-changed`); `unready`, back to refine, always proposed, done on a person's tick; the issues waiting on it — and, its Scope changed, those on the same code — listed; each issue not read a box under "To check"; the same change flagged once | JN0V/workline-sandbox on GitHub (#10's Need rewritten: #12, #13 read again, a planted answer; #13 moved back to refine by a tick), live, no agent called; an imported file's lines changed, the tick on a change, an edit outside the sections in conformance (`changed-*`, `unchanged-need-flags-nothing`, `unready-ticked-moves-back`) |

| The weekly sample of its acts (ADR-0033): each act done alone kept in the record with its day and level (`did`), a tick's and a slip's left out; `workline sample --apply` drawing one in ten of a week's onto their own tracking issue, each with its issue, kind, day, level and whether a person undid it; the level the acts undone suggest — none at `normal` suggests `enterprising`, more than one in ten the level below —, the setting untouched; the same suggestion in the role's own report, beside the one from the proposals settled, counted over every act the record keeps at the level | JN0V/workline-sandbox on GitHub (three acts planted on #33, #35; #35's rename undone by hand; tracking issue #37, #33's order drawn, a rerun editing the comment in place), live, no agent called; conformance `acts-*`, `sample-*`, `report-suggests-from-the-acts-done-alone` (product-owner; the report's suggestion in conformance only) |
| A spec read before ready (#128, ADR-0020): with the reviewer after it in the line, `ready` held while the reviewer has not read the body as it is, a finding is open, or the rounds are spent — the agent's, an accepted draft's, a tick's alike; the findings given at the next refine, its own sections rewritten (`wrote`), a person's asked of the reporter; released by the engine with no agent; `workline:accepted` a person's yes | JN0V/workline-sandbox on GitHub: #44 held, answered, released (planted); #45 held at the sixth round, released by the label (planted); #46 refined by a real agent, read by the reviewer, released; #12 held, its reporter asked the reviewer's question by a real agent (tried.md); conformance `ready-held-while-a-spec-finding-is-open`, `ready-released-once-the-spec-is-answered`, `refine-answers-the-spec-findings` |

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
   for; the label taken off, an exempt label, a judge's no tried live; 7
   days and 3 a run measured. (The weekly sample draws these closings
   with the other acts: ADR-0033.)
3. **Answers on the issue, what is left** (ADR-0038): the week's trial
   on workline and the maintainer's GitLab project, measured; a real
   agent's revision after a comment; a comment by an outsider and a bot
   live (one account on each sandbox); `workline:accepted` set by a bot
   read as the project's; the Forgejo sample's `label-events` and
   `react`; an outsider's proposal labelled `workline:proposed` (its
   comment to the reporter is the proposal today). The night's summary
   grouped and linked, the judging job saying what the applying one will
   do, and a body's footer kept out of its sections (seen on
   [#261](https://github.com/JN0V/workline/issues/261)'s Scope): in
   conformance only (`summary-judged-says-what-will-be-done`,
   `summary-applied-does-what-the-judge-said`,
   `refine-leaves-a-footer-out`), seen on the night of 2026-10-09, its
   wording since made plain. The new writing rules of the prompt — a
   Need from the user's side, a concrete Validation, one plain question
   rather than a guess — read on a real night by the maintainer.
4. **Autonomy, what is left** (ADR-0026): the acts' suggestion seen
   live in the role's own report; a rename by a real
   agent undone, `ready` taken off and a split's child closed live;
   `workline init`'s question on a terminal; an undone split on GitLab,
   which keeps no reason for a closing.
5. **The weekly sample of its acts, what is left** (ADR-0033): a real
   week's acts drawn and read by the maintainer — the sandbox's were
   planted, a closing undone and a suggestion not yet seen live; ten or more
   acts in the window, live; a GitLab project's; workline's own (its
   record has no `did` until its next nightly run with this); a person's
   tick in the sample read as a verdict, if undoing proves too coarse. A
   file the agent was not shown named as a source. The one way to open issues is built (ADR-0018,
   amended); its import's plan sees a closed issue; left: a real agent's
   answer to one.
   **The import's map, what is left** (ADR-0030): a real agent's import
   answered with `skip`s — DomoticsCore's roadmap again, the map read by
   the maintainer against its 56 issues, each reason agreed with, none
   missing; the tokens a `skip` a line costs, measured.
6. **Ordering, what is left**: a forge's native rank (GitLab's reorder, a
   GitHub project's position), deferred (ADR-0018); milestones ranked by
   their due date tried on GitLab; the engine refusing a person's
   priority, tried in conformance only.
7. **A parent, what is left** (ADR-0029): an agent drafting which
   Verification items look proved in other words, if live use asks; a
   test named in prose, outside a code span; a test named as proof read
   live on GitLab; a part of another repository;
   a plugged forge's `closers` tried.
8. **What an issue waits on, what is left** (ADR-0028): a GitLab Premium
   link tried live; a relation across projects; a link taken off on
   GitLab live, and a real agent's `undepend` ticked live; the links a
   split set before the parent's state kept `after`, unknown to the role;
   the Forgejo sample's `add-blocker` and `remove-blocker`; the
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
