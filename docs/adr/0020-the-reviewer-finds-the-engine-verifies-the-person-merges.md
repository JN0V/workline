# ADR-0020: The reviewer finds, the engine verifies, the person merges

- **Status:** accepted; the code subject built (local, merge request), the
  spec subject and the developer's loop to come (roles/reviewer/docs/status.md)
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
   claims to check against the code, not as proof (amended: the claims
   lens contests them, #126). A lens may be asked to
   look for a number of candidates first, from the change's size (a floor
   on candidates only, never on what is shown; its worth measured by #90);
   finding nothing is not a failure, and nothing requires a finding to be
   shown.
   *(Amended 2026-10-06: on a machine the lenses share one call, the
   change given once; see Amendment, #147.)*
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
   *(Amended 2026-10-06: a kept line beside a removal is the change's too;
   see Amendment, #224.)*
5. **A judge for each important finding**, apart, at the best independence
   available, the level said in the verdict (ADR-0005): at least `model`
   for a merge request the developer role opened, `context` for a person's
   (`judge-at-least`). A no drops the finding, said with the judge's reason
   (`finding-judged-no`); a finding not verified is shown as such. A nit is
   not judged, and one outside the change is left, counted. The judge asks
   the lens's question (#223): whether the quoted code fails, or, for the
   tests lens, whether a test exercises the behaviour, read from the tests
   that touch the cause's file.
6. **The verdict by rules.** The rules' findings block; what the lenses
   find warns until #90 has measured it, then a verified important finding
   blocks (`ai-findings: block`). The reviewer never approves, never
   patches, never merges: a lens proposing a patch is refused, and the lens
   counted as not answered. The person merges.
   *(Amended 2026-10-06: `ai-findings` is set by lens too, each lens
   blocking once it earned it; see Amendment.)*

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

## Amendment (2026-10-06)

- **`ai-findings` by lens** (#222): one value, `warn` or `block`, still
  sets every lens; a map sets each lens apart,
  `{correctness: block, edge-cases: warn}`, a lens not named warning.
  #90's step 1 found the lenses unequal (roles/reviewer/docs/tried.md):
  correctness ready to block, edge cases in doubt, tests not ready. One
  switch for all would hold a merge request on the weakest lens's guess,
  or let the strongest lens's defect through.
- **A value or a lens the reviewer does not know stops the run**, never
  read as `warn` (principle 12).
- **Two findings on one line**: of two important ones, the one from a lens
  that blocks leads, so a merge never turns a blocking finding into a
  warning.
- **Workline's own line stays at `warn`** until a measure over five runs a
  case ([ADR-0014](0014-checked-is-earned-by-what-was-read.md)); the
  maintainer switches correctness to `block`.

## Amendment (2026-10-06, #229)

- **Findings on one line are grouped, each judged by its lens.** Merged
  and judged once by the leading lens's question, a tests-lens "no test"
  under a correctness finding was refused with it
  (roles/reviewer/docs/tried.md, restock).
- Grouping only shapes what the author reads: one row, the others beside
  the leader, each with its own verdict. A refused finding is dropped
  alone (`finding-judged-no`).
- **The leader**, once judged: a verified finding first, then one from a
  blocking lens, then an important one; the group blocks when a verified
  important finding of a blocking lens is in it.
- **Cost**: one judge call for each important finding, shared line or not.

## Amendment (2026-10-06, #226)

- **A blocked merge request still gets the comment.** A block applied
  nothing, as for every role: no comment, the record not moved, the
  next push reviewing the same commits again, and the author reading the
  finding only in the job's verdict. A block is when the author most
  needs it.
- **The engine**: a role names in `on-block` the intentions a blocked run
  still applies, those that only say why (docs/spec/role-contract.md,
  step 5); the run still blocks. The reviewer names `comment` and
  `issue`: its summary comment, the record in it, and the issues outside
  the change the comment counts.
- **The comment**: what blocks first, marked `**blocks**`, then the
  warnings. The record moves when every lens answered, as on a pass; a
  rule that blocks writes the comment too, the record left as it was.
- **The templates**: the trust split is kept. The judging job, with the
  AI key, fails by the verdict; the applying job, with the write token
  and no AI key, runs whatever the judge's outcome (`if: always()`,
  `when: always`) and writes what is pending. GitLab's gardening apply
  had no `when: always` (#142); GitLab's judging jobs now warn on an
  agent out of reach (3), as GitHub's do.
- **Rejected**: posting the comment from the judging job (a write token
  beside the AI key, against
  [principle 6](../PRINCIPLES.md)); a third job only for the comment
  (one more job for what `workline apply` already does).

## Amendment (2026-10-06, #224)

- **A removal's exposed lines are the change's.** #90's step 1: every
  lens found the guard removed from `Page`, each quoting
  `from := (p - 1) * size`, a line the change kept; the engine sent the
  change's defect to an issue (roles/reviewer/docs/tried.md).
- **The rule**, in order: a cause on a line the change added; else on a
  kept line within three lines of a removal; else on a line the change
  removed, even one whose text is also kept elsewhere; else outside the
  change.
- **A removal** is a hunk taking away more lines than it adds; three is a
  unified diff's default context. A hunk replacing lines one for one
  exposes nothing: its added lines are the change.
- **The cost**: a defect from before the change, quoted within three
  lines of a removal, goes to the author, not to an issue. Kept small by
  the window; the judge still verifies it.
- **Rejected**: asking the lens which lines a finding is about (the model
  says, the engine does not check: the split this ADR made mechanical);
  the whole function around a removal (a long function sends every
  defect in it to the author).

## Amendment (2026-10-06, #147)

- **A full review's cost, measured** (roles/reviewer/docs/tried.md):
  #146's eight commits, every lens, 430k tokens in. Its prompts replayed
  with no agent: the lenses 64%, each sent the same change and files; two
  tests judges 29%, 68k characters of tests each.
- **The lenses together** (`lenses-together`): one call for the lenses of
  a run, the change given once, each finding naming its lens and still
  judged by its lens's question. "Lenses run apart" gives way on a
  machine; the setting turns it back, and a merge request asks one lens a
  push anyway.
- **Each file once**: whole, the change marked in it, while the files fit
  `code-lines-max` (600); the others by the change's hunks.
- **The judge's material**: the change within 40 lines of the cause; a
  tests judge 300 lines of tests (`tests-lines-max`), the rest named.
- **A budget**: `ai-max-tokens`, 200000 a run. Once spent, nothing more is
  asked, said; a lens not asked or a finding not judged leaves the commits
  unrecorded. The summary says the tokens used against it, each call's in
  `out/review.json`.
- **The result**: 453k tokens in, estimated, down to 142k for the same
  review.
- **Rejected**: one judge call for several findings (a judge answers one
  question, [ADR-0005](0005-independence-takes-the-best-level-available.md));
  estimating a call before making it (the cap is checked against what was
  spent, docs/spec/role-contract.md); characters / 4 as the estimate (1.2
  characters a token on this code).

## Amendment (2026-10-06, #126)

- **Intent.** A change saying it closes an issue (`Closes #4`, `fixes`,
  `resolves`, `implements`; in a commit or the merge request) has the
  issue read from the forge: its Need, Verification and Scope, up to
  `issue-lines-max` (80) lines, given to the lenses. The `intent` lens
  says what the issue asks and the change does not do or prove, its
  cause quoted from the issue (`path: "#4"`), found again there and the
  change's; and what the change does past the Scope, its cause the code.
  Its judge is shown the issue, and for a cause in the issue the whole
  merge request's change (base..head, commits reviewed before included)
  up to `code-lines-max`: on a merge request reviewed push by push, the
  lens reads the new commits only, and a part an earlier one did is
  refused by the judge, not reported missing.
- **No issue, no intent lens**: a lens with `needs: issue` is not asked
  of a change closing none, said in the summary; one it cannot read is
  said (`issue-unread`), never a silent pass (principle 12).
- **The author's claims contested.** The commit messages are given whole
  (trailers left), with the merge request's title and body, as testimony,
  up to `testimony-lines-max` (80) lines. The `claims` lens
  (`cites: claim`) quotes each claim the change contradicts, found again
  in what the author said or dropped, its cause the contradicting line;
  its judge asks whether that line contradicts that claim.
- **The floor on finders only**: `finder-floor` stays on candidates; a
  case pins that a lens under it answering nothing leaves the review
  clean, nothing judged, the commits recorded. Its worth: #90.
- **Same call**: both are lenses, asked in the lenses' one call
  (`lenses-together`), each with its own judge question in its front
  matter (#223); on a merge request, in turn with the others.
- **Cost**, offline on #146's review: the lenses' prompt 0.7% larger,
  3.0% with an issue closed (about 2k tokens); a judge call for each
  important intent or claims finding.
- **Rejected**: the issue in every lens's call (paid on each push of a
  merge request whatever the lens); a claim checked by the model's word
  (the engine finds the quote again, as for a cause); an intent finding
  with no quote (the issue's words are the cause).

## Amendment (2026-10-06, #126: a decision for a person)

- **A third outcome.** A finding was the change's (the author fixes) or
  an issue. A trade-off, a design choice, a behaviour the issue leaves
  open is neither: forced into a fix, it asks the author to guess; into
  an issue, nobody decides it.
- **Marked by the lens**: `decision`, the question for a person, in its
  finding; one line in the lenses' instruction, one in the intent lens.
- **Guarded by the engine**, so it is no way past the judge:
  - its cause found again like any other, in the change or the issue it
    closes; elsewhere it is dropped, said (`finding-unfounded`), never
    an issue;
  - never important, never judged, never blocking, never counted as a
    finding on the change; `questions-max` (3) a run, the rest counted;
  - the lens is told what fails is a finding, never a decision; a defect
    marked as one still reaches a person, as a question, not lost.
- **Shown**: the summary comment asks each under **Questions for a
  person**, one line, its cause quoted, apart from the findings' table;
  the verdict's summary counts them; locally, a `decision` finding of
  level `question` (SARIF `note`, Code Quality `info`), and `questions`
  in `out/review.json`.
- **Answered** by a reply on the merge request: the reviewer does not
  wait, nor read it back; the record moves as on any run.
- **Not the product owner's tick**
  ([ADR-0025](0025-a-persons-tick-is-done-ignored-runs-pause.md)): a tick
  is a yes to a proposal the engine then applies; a question leaves the
  engine nothing to apply: the person decides, the author follows.
- **Cost**: no judge call; the lenses' prompt 0.4% larger on #146's
  review, offline (roles/reviewer/docs/tried.md).
- **Rejected**: a decision judged (a judge says whether code fails, not
  what a person wants); one opened as an issue (nobody decides it before
  the merge); a `severity: decision` (the question itself is the mark).
