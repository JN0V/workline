# Committer — where it stands (2026-10-09)

**In one line:** runs on every commit and push of workline's author, and
on each merge request in CI; its rewrites tried with Claude on refused
messages of this repository; its hooks tried on a copy of DomoticsCore
(its [page](../README.md#tried-for-real)). Built.

## Built

Built on 2026-10-09, by the maintainer, against the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md):

1. **Nothing buildable in Missing**: met. What is left is tries on real
   use (below); one commit, one change is a new check, parked as
   [#192](https://github.com/JN0V/workline/issues/192) (closed, not
   planned).
2. **The main issue validated on real use**: met. The maintainer
   validated the role on 2026-10-09 from its real use on all their
   commits, on workline and at work.
3. **AI verdicts measured**: in part, and the maintainer moved the status
   knowing it.
   - Measured: the weekly evaluation's four rewrite cases
     (tests/evaluation/results.tsv). From 2026-09-25 to 2026-09-30, 42 of
     56 runs passed every check (275 of 293 checks). The misses: too few
     of the author's words kept, a dropped word the judge says changed
     the meaning, or a subject still over 72 characters.
   - Not measured: the judgement on a code with no dash
     (`possible-internal-code`) has no case.

## Missing

1. Tries on real use: a hook that reads `pre-push`'s input after
   workline's line; husky or lefthook.
