# The documentalist's fixes, reviewed (2026-10-01)

Every doc change the documentalist made on workline and on DomoticsCore up
to 2026-10-01, read by an agent against the code as it is now, then a solo
repository tried (roles/documentalist/tried.md). Read by one agent
(Claude); corrected on one point by a second review (92feb08 left
`checked`). Every verdict below is **confirmed 2026-10-01, second check**
(ADR-0014, amended): another agent of the same provider, sharing no context
with the first review (ADR-0005 level 3, not another provider), reread each
one at its own commit (`git show <commit>:<path>`); its corrections are in
the verdicts. No disagreement was left for the person. The evidence behind
[ADR-0014](../adr/0014-checked-is-earned-by-what-was-read.md).

## What it changed, and whether it was right

workline (the bot's commits, "docs: bring the docs in line with the code"):

| Commit | Change | Verdict |
|---|---|---|
| 4ac4f44 | README: `cmd:` runs any other agent; the approval in a dialog | right |
| 445bed2 | README: the judge job holds code scanning's write token | right; the diagram above it left saying "no write token" |
| 972afa9 | README: the templates run `workline route` | right |
| d38a0e4 | README: GitLab "tried … the template never run live" | wrong in fact, yet nothing in its tree contradicted it: the template's header said not tried, tried.md recorded the REST access only ("no agent"); the real try (b738e35, #30's branch) was not an ancestor. A race between branches, not a misreading; not an evaluation case |
| 172493e | README: the "not yet" column | right, undoes d38a0e4 |
| d43b3f2 | README: removed "a comment edited in place", tried live | wrong: a true claim removed; internal/forge, among the sources, showed the edit in place (PATCH): only "tried live" needed tried.md |
| fea16c7 | docs/spec/routing.md: `judged:` only | bookkeeping; the commit named was rebased away |
| 0f4c01c | roles/committer/README.md: both gitleaks variables, the rule id | right; the code comment beside them left stale |
| 3cd196d (#29) | README: "a push approval … unless your config turns it off" | wrong: a stale code comment won over the setting; the same commit's right change to README:60 was lost when it was reverted |

DomoticsCore (squashed into main):

| Commit | Change | Verdict |
|---|---|---|
| bc0bd26 (#106) | docs/README.md: 2.0.0 → 2.11.0 | right; `checked` moved over "Last Updated: 2026-03-10" on the same line |
| 2e25ea0 (#108) | MQTT TLS needs a root CA | right; `checked` moved on hal-architecture.md, whose table misses three `*_HAL.h` files (CoreLog_HAL.h, Testing/HeapTracker_HAL.h, WebUI/WebResponse_HAL.h) and says "each file < 800 lines" (one has 930) |
| 25e22dd (#110) | docs/README.md: 2.12.0 | version right; `checked` moved again over "Documentation Last Updated: 2026-03-10", stale, as in bc0bd26 |
| 16c660b (#110) | Core 1.13.1, MQTT 1.10.0 | versions right; `checked` moved over line counts false at that commit (283 for 462, 630 for 930); siblings left at 1.9.0 |
| 92feb08 (#109) | OTA 1.4.1 → 1.11.0, six lines | versions right; `checked` left; rewrote the number inside a bug fixed in 2025 ("returns hardcoded 1.4.0 instead of 1.11.0"): a wrong edit, not a wrong vouch; field count left false ("13 fields", OTA.h has 10); test and line counts not recounted one by one |
| d784398 (#112) | storage 1.6.1 | right, `judged` set as ADR-0013 asks; a line count left false |
| 5ae8335 (#113) | README: twelve versions, the LED states | right; the badge's link to v2.0.0 and "Version 2.0.0 Released!" left |

Still stale after 2.12.0: the versions of home-assistant, wifi, system, ntp,
remote-console and system-info's project-context.md, and getting-started.md.

## Patterns

What it writes is right about four times in five: versions read from
`library.json`, flags, CI permissions, small diffs. What fails:

1. **It vouches for what it did not read.** Moving `checked` says every
   sentence is true. It moved on DomoticsCore only — four commits, five docs
   (bc0bd26, 2e25ea0, 25e22dd, 16c660b) — each over false frozen numbers or
   a stale date. Every one of those docs had sources far past what a task
   shows whole (20,000 characters; 28 to 165 KB): the agent saw diffs alone,
   and the judge accepted the move. workline's bot never moved `checked`:
   its three wrong fixes are wrong content, a class of their own.
2. **It takes anything it reads as evidence.** A stale code comment over
   the code's setting (#29). d38a0e4 followed a template's header too, but
   nothing in its tree said otherwise.
3. **It removes what it cannot see backed** (d43b3f2), where the sources
   given are not all there is.
4. **It sees one doc at a time.** The same value fixed in one doc stays
   stale in its siblings; a claim about a file outside the doc's sources is
   neither checked nor said unchecked.
5. **The evaluation did not see it.** Every documentalist case passes at its
   best (status.md, "Measures"): the cases are cleaner than real docs, and
   none plants a frozen number, a stale comment or a sibling.

## The solo repository (a copy of WaterMeter)

Three engine defects, fixed and guarded since: a header-only commit counted
as touching the docs following that doc; a right fix refused twice because
the judge checked the patch more strictly than the engine applies it
(`--unidiff-zero`), the run stopping there silently; docs too large to be
judged whole listed without saying a person must judge them. Seven of ten
suspect docs went to a person, judging in parts being off by default.

## New code no doc describes

A file added under a folder a doc names in `sources` makes that doc suspect.
A file anywhere else is reported (`undocumented`) only when the project sets
`documented` globs, empty by default and set on neither workline nor
DomoticsCore: today a new source file is, by default, silently ignored, and
never proposed to any doc.

## `checked` put back (ADR-0014, step 0)

Every `checked` moved since adoption (workline from c1ae34e, DomoticsCore
from 6a979a6), by the bot or by a person, measured at the commit of the move
with the engine's own rule (`sourcesNow`: every text file under each source,
or the section of a source doc, together at most 20,000 characters). A doc
whose `checked` is still an unearned move, or a run of them, has `checked`
put back to its last earned value — the one the first unearned move
replaced, often the value set at adoption — and `judged` set to the commit
the last move named (ADR-0013): that commit was looked at, nothing in it was
vouched for. Each doc is suspect again, for a person to read whole; bodies
untouched. A header written with the doc, at adoption or creation, is not a
move.

workline: 132 moves, all by a person (JN0V), 5 earned (routing.md
three times, conformance.md at 07a5d55, roles/committer/README.md at
dfe6338). Put back in 0669684:

| Doc | Unearned moves | First (sources) | Last (sources) | Header now |
|---|---|---|---|---|
| README.md | 10 | 07a5d55 (41,177) | 047deb6 (61,419) | checked 5b8b173, judged 6013a50 |
| docs/spec/conformance.md | 19 | 71f289a (46,028) | 047deb6 (76,506) | checked edfd466, judged 6013a50 |
| docs/spec/model-grid.md | 11 | 78f400d (37,937) | 047deb6 (64,580) | checked d30d22a, judged 6013a50 |
| docs/spec/multi-repo.md | 18 | dfe6338 (20,745) | 047deb6 (39,994) | checked d30d22a, judged 6013a50 |
| docs/spec/role-adapting.md | 11 | bb8aa81 (58,093) | 047deb6 (84,608) | checked 347b403, judged 6013a50 |
| docs/spec/role-contract.md | 13 | 2c0e022 (51,061) | 047deb6 (84,608) | checked d30d22a, judged 6013a50 |
| docs/spec/role-outcome.md | 11 | bb8aa81 (58,093) | 047deb6 (84,608) | checked 347b403, judged 6013a50 |
| docs/usage.md | 14 | dfe6338 (60,783) | 047deb6 (77,502) | checked d30d22a, judged 6013a50 |
| roles/committer/README.md | 2 | 3b799e6 (27,763) | 8d58c25 (30,848) | checked 82b6394, judged d77c33b |
| roles/documentalist/README.md | 18 | dfe6338 (54,357) | 047deb6 (123,660) | checked d30d22a, judged 6013a50 |

roles/documentalist/README.md's sources were narrowed since (47917f1) to
one file that fits: it can be judged whole now.

DomoticsCore: 22 moves, none earned — 14 docs by a person in 9e20ce2, and
the bot's eight moves on five docs (bc0bd26, 2e25ea0, 25e22dd, 16c660b;
first counted as six). Put back on the branch
docs/undo-unearned-checked:

| Doc | Moves | First (sources) | Last (sources) | Header now |
|---|---|---|---|---|
| docs/README.md | 2 (bot) | bc0bd26 (52,534) | 25e22dd (52,536) | checked 436872f, judged f734747 |
| docs/architecture/hal-architecture.md | 2 (bot) | bc0bd26 (28,060) | 2e25ea0 (28,274) | checked ae5715e, judged d7afb75 |
| docs/architecture/component-lifecycle.md | 1 | 9e20ce2 (265,030) | — | checked ae5715e, judged 6a979a6 |
| docs/components/core/README.md | 2 | 9e20ce2 (265,030) | 25e22dd, bot (112,186) | checked ae5715e, judged f734747 |
| docs/components/core/project-context.md | 2 | 9e20ce2 (273,579) | 16c660b, bot (165,182) | checked 4cdb3e6, judged 0820cf2 |
| docs/components/home-assistant/index.md | 1 | 9e20ce2 (361,648) | — | checked ae5715e, judged 6a979a6 |
| docs/components/home-assistant/project-context.md | 1 | 9e20ce2 (193,564) | — | checked 967d328, judged 6a979a6 |
| docs/components/home-assistant/README.md | 1 | 9e20ce2 (118,157) | — | checked c5aef9f, judged 6a979a6 |
| docs/components/led/project-context.md | 1 | 9e20ce2 (75,064) | — | checked ae5715e, judged 6a979a6 |
| docs/components/led/README.md | 1 | 9e20ce2 (50,405) | — | checked ae5715e, judged 6a979a6 |
| docs/components/led/technical-reference.md | 1 | 9e20ce2 (49,472) | — | checked 3c4d96c, judged 6a979a6 |
| docs/components/mqtt/project-context.md | 1 | 9e20ce2 (106,806) | — | checked fa51735, judged 6a979a6 |
| docs/components/mqtt/README.md | 3 | 9e20ce2 (106,806) | 16c660b, bot (63,577) | checked ae5715e, judged 0820cf2 |
| docs/components/mqtt/technical-reference.md | 1 | 9e20ce2 (74,535) | — | checked e927998, judged 6a979a6 |
| docs/components/ntp/README.md | 1 | 9e20ce2 (70,287) | — | checked ae5715e, judged 6a979a6 |
| docs/components/ntp/technical-reference.md | 1 | 9e20ce2 (39,168) | — | checked 281c95a, judged 7920678 |

Sources in characters, at the commit of the move. A run of the
documentalist without an agent lists all 26 docs suspect afterwards.
