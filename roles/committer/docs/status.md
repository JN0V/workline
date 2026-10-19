# Committer — where it stands (2026-10-19)

**In one line:** runs on every commit and push of workline's author, and
on each merge request in CI; its rewrites tried with Claude on refused
messages of this repository; its hooks tried on a copy of DomoticsCore
(its [page](../README.md#tried-for-real)). Built: first on 2026-10-09,
back to beta on 2026-10-13 for a bug of its own,
[#107](https://github.com/JN0V/workline/issues/107), built again on
2026-10-14 once it was fixed.

A try since 2026-10-13 is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## The three criteria

Built again on 2026-10-14, by the maintainer, against the three criteria
of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md):

1. **Nothing buildable in Missing**: met. Missing holds only tries on
   real use.
   - [#107](https://github.com/JN0V/workline/issues/107), the user's
     allow-list refusing the forge's own commits, is fixed: a project
     names them in `forge-identities`.
   - [#370](https://github.com/JN0V/workline/issues/370), a commit with
     an empty address skipping the check, is fixed: it is refused.
   - One commit, one change is parked as
     [#192](https://github.com/JN0V/workline/issues/192) (closed, not
     planned).
2. **The main issue validated on real use**: met. The maintainer
   validated the role on 2026-10-09 from its real use on all their
   commits.
3. **AI verdicts measured**: in part, and the maintainer moved the status
   to built on 2026-10-09 and 2026-10-14 knowing it.
   - Measured: the weekly evaluation's four rewrite cases
     (tests/evaluation/results.tsv). From 2026-09-25 to 2026-09-30, 42 of
     56 runs passed every check (275 of 293 checks). The misses: too few
     of the author's words kept, a dropped word the judge says changed the
     meaning, or a subject still over 72 characters.
   - Not measured: the judgement on a code with no dash
     (`possible-internal-code`) has no case.

## Built

Cases in
[tests/conformance/cases/committer](../../../tests/conformance/cases/committer);
its evaluation in
[tests/evaluation/cases/committer](../../../tests/evaluation/cases/committer).

- **A message's form**, an internal code in the subject:
  `internal-code-in-subject`, `code-in-trailer-passes`,
  `code-without-dash-in-subject`, `standard-*`, `one-message-per-commit`.
- **The rewrite opt-in**
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)):
  refused at once by default, the message kept: `refused-at-once-by-default`,
  `hook-refuses-and-keeps-the-message`; turned on, said before it starts,
  ended past `ai-max-seconds`: `hook-says-the-rewrite-before-it-starts`,
  `rewrite-ends-past-its-seconds`, `rewrite-retry-within-its-seconds`.
- **A person's own settings**: the rewrite and its wait, held by a
  project's `ai: none`; names added to the project's list; any other said:
  `person-*`.
- **A code refused names the line that lets it through**, the person's and
  the project's: `refusal-gives-the-allow-line`; the ESP32 and ESP8266
  chips in the role's list: `chip-names-pass`.
- **The agent's rewrite**, keeping the type, the scope, the trailers and
  the author's words: `agent-rewrites-message`, `rewrite-*`,
  `public-name-kept-by-the-agent`; marked by workline's two trailers,
  never the old message's words, still the person's commit:
  `rewrite-carries-its-trailers`, `no-rewrite-no-trailer`,
  `rewritten-commits-pass-their-checks`, routing's
  `rewritten-commit-is-judged`
  ([#205](https://github.com/JN0V/workline/issues/205)).
- **Secrets and forbidden terms**, gitleaks: `secret-*`,
  `secrets-not-checked`, `forbidden-terms-common-and-repo`, `term-*`.
- **The author's identity**: `identity-*`; the forge's own addresses a
  project names: `identity-forge-*`.
- **On a merge request**, every commit checked:
  `merge-request-checks-every-commit`,
  `possible-code-in-pushed-commits-warns`.

## Missing

- **Tries on real use**: a hook that reads `pre-push`'s input after
  workline's line; husky or lefthook. No issue yet.
