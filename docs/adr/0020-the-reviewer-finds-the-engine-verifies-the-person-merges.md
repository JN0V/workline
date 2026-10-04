# ADR-0020: The reviewer finds, the engine verifies, the person merges

- **Status:** accepted; the code subject built (local, merge request), the
  spec subject and the developer's loop to come (roles/reviewer/status.md)
- **Date:** 2026-10-04
- **Builds on:** ADR-0005 (a judge's independence), ADR-0009 (a question in
  parts), ADR-0011 (the review is on the merge request), ADR-0013 (what was
  judged is not asked again), ADR-0018 (one way in for every role's
  issues), principles 1, 2, 4, 6, 8, 12

## Context

workline's lines check commit messages and docs; nothing reads the code a
change brings. The review is the merge request's (ADR-0011), done by a
person, and a person reading every line of what agents write does not
scale. Issue #87 asks for a reviewer role.

docs/research/code-review.md found that the AI reviewers that hold up split
finding from verifying, let no reviewer merge, and keep what is posted
capped; that a cynical persona changes nothing while asking for concrete
findings and what is missing does; that self-triage by the same model
is common and weak; and that none decides mechanically whether a finding
belongs to the change or was there before — the model says "pre-existing",
or nothing.

Borrowed, credited there: BMAD-METHOD's lenses run apart, its floor on
candidate findings, its verified triage and its bounded rounds; Claude Code
Review's verification step; gh-aw's cap; CodeRabbit's warn before block.

## Decision

**One role, two subjects, two moments.** The `reviewer` reviews code — and,
next, specs — at two moments: on the machine, at the end of development,
before the push; and on the merge request, a safety net.

| | code | spec |
|---|---|---|
| on the machine | `workline review`: base..HEAD (main..HEAD by default), every lens; the findings in the terminal and as JSON, for the author's agent to fix before pushing | a file, or an issue by its number |
| on the merge request | the commits not reviewed yet, one lens a push, every lens once when it becomes ready; warns at first | after the product owner refines an issue: `ready` only once no important finding is open |

**The same pipeline in every cell.**

1. **Rules before any AI.** On the comments the change adds: a bug's story
   ("used to", "the bug was", "previously"), a code internal to the project
   (the committer's own list), a comment block too long. They block, but the
   long block, which warns. While they block, no agent is asked: the author
   fixes them first.
2. **Finders.** Each lens — correctness, edge cases, tests — is a part of
   one question (ADR-0009), in a context of its own, answering `finding`s
   only: a new intention, `{severity, title, why, cause, symptom, fix}`,
   whose quotes the engine finds again. The commit messages are given as
   claims to check against the code, not as proof. A lens may be asked to
   look for a number of candidates first, from the change's size (a floor
   on candidates only, never on what is shown; its worth measured by #90);
   finding nothing is not a failure, and nothing requires a finding to be
   shown.
3. **The engine checks the quotes.** A finding whose cause, or symptom, it
   does not find again — in the file at the head of the range, or among the
   lines the change removed — is dropped and said (`finding-unfounded`),
   never silently.
4. **Related or not, decided by the cause.** Each finding quotes its
   *cause*. If the quote lies on a line the change added or removed, the
   finding is the change's — even when its symptom is elsewhere, a caller
   the change broke — and its author fixes it: the developer or the person
   for code, the product owner for a spec. Otherwise it is not theirs to fix
   there: deeper, structural, there before. It becomes an issue, through
   the one way in (ADR-0018): a key per subject hidden in its body, an open
   issue holding it left as it is, `needs-triage`, the product owner's state
   naming its file and the commit, a cap a run. It is not posted on the
   merge request.
5. **A judge for each important finding**, apart, at the best independence
   available, the level said in the verdict (ADR-0005): at least `model`
   for a merge request the developer role opened, `context` for a person's
   (`judge-at-least`). A no drops the finding, said with the judge's reason
   (`finding-judged-no`); a finding not verified is shown as such. A nit is
   not judged, and one outside the change is left, counted.
6. **The verdict by rules.** The rules' findings block; what the lenses
   find warns until #90 has measured it, then a verified important finding
   blocks (`ai-findings: block`). The reviewer never approves, never
   patches, never merges: a lens proposing a patch is refused, and the lens
   counted as not answered. The person merges.

**Outputs.** The findings as SARIF and GitLab's Code Quality report (as
every role's), and one summary comment on the merge request, edited on each
run. Inline review comments, native to each forge, wait for the bot's own
identity (#81).

**Caps.** Findings shown a run (`findings-max`, the rest counted); issues
opened a run (`issues-max`); tokens (`ai-max-tokens`); only commits not
reviewed yet: the record, hidden in the summary comment on a merge request
and in the git directory on a machine, holds the commits a review answered
whole (ADR-0013's way). A lens that failed leaves them unrecorded, and the
run says it did not review them whole. An agent unreachable ends the run
`blocked-external`, the rules' findings still said.

**Depth.** On a merge request, one lens a push, in turn; every lens on the
machine, and once on the merge request when it becomes ready
(`--input lenses=all`).

**What it does not do.** It runs no Verification command — the tests and
the gates do; its tests lens checks that a test covers each behaviour the
change adds or changes.

**Spec review** (next). Its input is a file or an issue's number; on the
forge it runs after the product owner refines an issue, which goes `ready`
only once no important finding is open: a loop between the product owner
and the reviewer, five rounds at most, then a question to a person.

**The developer's loop** (with the developer role, #117): the reviewer hands
over to the developer, five rounds at most, then `human`.

**Opt-in.** `workline init --review` adds the reviewer to a project's
merge-request line; on in workline's own repository.

## Consequences

- An agent's change is read before a person reads it, and the person reads
  findings that were quoted, found again and judged, not guesses.
- What was there before goes to the backlog, once, not to the author of an
  unrelated change.
- A finding costs a call to a judge; a push, one lens; a review on the
  machine, every lens: tokens the record and the caps bound.
- Until #90 measures them, the lenses' findings only warn: a merge request
  is not held by a model's opinion.
- The engine learned two things any role may use: parts answering what the
  role names (`part-intentions`), and questions for a judge written by
  `pre`, each answered apart (docs/spec/role-contract.md).
