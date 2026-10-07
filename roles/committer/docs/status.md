# Committer — where it stands (2026-10-07)

**In one line:** runs on every commit and push of workline's author, and
on each merge request in CI; its rewrites tried with Claude on refused
messages of this repository; its hooks tried on a copy of DomoticsCore
(its [page](../README.md#tried-for-real)). Beta.

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: met. What is left is tries on real
   use (below); one commit, one change is a new check, parked as
   [#192](https://github.com/JN0V/workline/issues/192).
2. **The main issue validated on real use**: open. No issue of the role
   holds a Validation section the maintainer has signed.
3. **AI verdicts measured**: in part. The weekly evaluation runs its four
   rewrite cases (tests/evaluation/results.tsv), no pass rate recorded
   here yet; the judgement on a code with no dash (`possible-internal-code`)
   has no case.

## Missing

1. Tries on real use: a hook that reads `pre-push`'s input after
   workline's line; husky or lefthook; a work repository.
