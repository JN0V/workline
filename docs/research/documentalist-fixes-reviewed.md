# The documentalist's fixes, reviewed (2026-10-01)

Every doc change the documentalist made on workline and on DomoticsCore up
to 2026-10-01, read by an agent against the code as it is now, then a solo
repository tried (roles/documentalist/tried.md). Read by one agent
(Claude), not yet confirmed by a person; corrected on one point by a second
review (92feb08 left `checked`). The evidence behind
[ADR-0014](../adr/0014-checked-is-earned-by-what-was-read.md).

## What it changed, and whether it was right

workline (the bot's commits, "docs: bring the docs in line with the code"):

| Commit | Change | Verdict |
|---|---|---|
| 4ac4f44 | README: `cmd:` runs any other agent; the approval in a dialog | right |
| 445bed2 | README: the judge job holds code scanning's write token | right; the diagram above it left saying "no write token" |
| 972afa9 | README: the templates run `workline route` | right |
| d38a0e4 | README: GitLab "tried … the template never run live" | wrong: followed the template's stale header comment; tried.md recorded the run |
| 172493e | README: the "not yet" column | right, undoes d38a0e4 |
| d43b3f2 | README: removed "a comment edited in place", tried live | wrong: a true claim removed because the sources did not show it |
| fea16c7 | docs/spec/routing.md: `judged:` only | bookkeeping; the commit named was rebased away |
| 0f4c01c | roles/committer/README.md: both gitleaks variables, the rule id | right; the code comment beside them left stale |
| 3cd196d (#29) | README: "a push approval … unless your config turns it off" | wrong: a stale code comment won over the setting; the same commit's right change to README:60 was lost when it was reverted |

DomoticsCore (squashed into main):

| Commit | Change | Verdict |
|---|---|---|
| bc0bd26 (#106) | docs/README.md: 2.0.0 → 2.11.0 | right; `checked` moved over "Last Updated: 2026-03-10" on the same line |
| 2e25ea0 (#108) | MQTT TLS needs a root CA | right; `checked` moved on hal-architecture.md, whose table misses a file and says "each file < 800 lines" (one has 930) |
| 25e22dd (#110) | docs/README.md: 2.12.0 | right |
| 16c660b (#110) | Core 1.13.1, MQTT 1.10.0 | versions right; `checked` moved over line counts false at that commit (283 for 462, 630 for 930); siblings left at 1.9.0 |
| 92feb08 (#109) | OTA 1.4.1 → 1.11.0, six lines | versions right; `checked` left; rewrote the number inside a bug fixed in 2025 ("returns hardcoded 1.4.0 instead of 1.11.0"): a wrong edit, not a wrong vouch; test, line and field counts left false |
| d784398 (#112) | storage 1.6.1 | right, `judged` set as ADR-0013 asks; a line count left false |
| 5ae8335 (#113) | README: twelve versions, the LED states | right; the badge's link to v2.0.0 and "Version 2.0.0 Released!" left |

Still stale after 2.12.0: the versions of home-assistant, wifi, system, ntp,
remote-console and system-info's project-context.md, and getting-started.md.

## Patterns

What it writes is right about four times in five: versions read from
`library.json`, flags, CI permissions, small diffs. What fails:

1. **It vouches for what it did not read.** Moving `checked` says every
   sentence is true. It moved on DomoticsCore only — four commits, six docs
   (bc0bd26, 2e25ea0, 25e22dd, 16c660b) — each over false frozen numbers or
   a stale date. Every one of those docs had sources far past what a task
   shows whole (20,000 characters; 28 to 165 KB): the agent saw diffs alone,
   and the judge accepted the move. workline's bot never moved `checked`:
   its three wrong fixes are wrong content, a class of their own.
2. **It takes anything it reads as evidence.** A stale code comment, a
   template's header, over the code and the record (d38a0e4, #29).
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
