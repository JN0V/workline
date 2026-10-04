# AI code review — focused research (2026-10-04)

**Question.** How do AI code reviewers find defects, keep false positives
down, say where a finding belongs, and stay out of the merge — and what
would a reviewer role on the line take from them?

**Verdict.** The tools that hold up split *finding* from *verifying*: several
finders in contexts of their own, then a step that checks each candidate
against the code, then rules for what is shown. None lets the reviewer merge;
most never approve. What none does is decide mechanically whether a finding
belongs to the change or was there before, nor send the second kind
somewhere else than the pull request. That, and verifying quotes rather than
trusting the finder, is what workline adds (ADR-0020).

Already covered elsewhere, not repeated here: reviewdog, PR-Agent, revmux
and "the reviewer sees only the pushed branch" (gates.md, "Review");
CodeRabbit's incremental review among the caps and caches (ci-and-forge.md,
"Caps and caches"); how automation is measured in production
(self-evaluation.md).

## Finders and verification

- **BMAD-METHOD, `bmad-code-review`**
  ([skills/bmad-code-review](https://github.com/bmad-code-org/BMAD-METHOD/tree/main/skills/bmad-code-review)).
  Lenses, each a subagent: the *Blind Hunter* sees the diff only; the *Edge
  Case Hunter* does "exhaustive path enumeration — mechanically walk every
  branch"; a *Verification Gap Reviewer*; an *Intent Alignment Auditor*.
  Only the Blind Hunter has a finding floor: "N = min(floor(sqrt(kB) + 1),
  10), where kB is the file's size in kilobytes" (`customize.toml`). A claims
  check reads the pull request's story as "the author's testimony, not
  evidence" (`references/claims-check.md`).
  Triage (`step-03-triage.md`): "Disregard any severity a reviewing subagent
  assigned", verify "at the cited file and line", verdicts `high`, `medium`,
  `low`, `false`, `maybe-false`, routed to patch, decision_needed or defer —
  "never drop, merge, or silently skip one". Verification-gap findings
  arrive pre-verified. `bmad-build-auto/step-04-review.md` halts a repair
  loop past 5 iterations: "non-convergence". The v6.0.0 "ADVERSARIAL Senior
  Developer" review asked for "3-10 specific issues in every review minimum".
  CHANGELOG #2675: "the 'cynical, jaded reviewer' framing made no difference
  to residual-bug hit rate, while requiring at least ten concrete findings
  and asking what is missing did."
- **Claude Code Review** ([docs](https://code.claude.com/docs/en/code-review)):
  finders in parallel, then "a verification step checks candidates against
  actual code behavior to filter out false positives"; findings labelled
  Important, Nit, Pre-existing; "The check run always completes with a
  neutral conclusion so it never blocks merging".
- **Anthropic's code-review plugin**
  ([commands/code-review.md](https://github.com/anthropics/claude-code/blob/main/plugins/code-review/commands/code-review.md)):
  skips a pull request closed, draft, one that "does not need code review
  (e.g. automated PR, trivial change…)", or that Claude "has already
  commented on"; a false-positive list opening with "Pre-existing issues",
  "Pedantic nitpicks that a senior engineer would not flag", "Issues that a
  linter will catch". Only the bug finders' candidates are validated.
- **Qodo / PR-Agent** ([qodo-ai/pr-agent](https://github.com/qodo-ai/pr-agent),
  `docs/docs/core-abilities/`): self-reflection — "the AI model reflects,
  scores, and re-ranks its own suggestions, eliminating irrelevant or
  incorrect ones": the same model judging itself. Ticket compliance checks
  "how well a Pull Request (PR) adheres to its original purpose/intent as
  defined by the associated ticket" (Fully, Partially, Not Compliant).

## Outputs, caps, who merges

- **GitHub Copilot code review**
  ([how-to](https://docs.github.com/en/copilot/how-tos/copilot-on-github/use-copilot-agents/copilot-code-review)):
  "By default, Copilot leaves a 'Comment' review, not an 'Approve' review or
  a 'Request changes' review". A public preview now lets it "submit an
  approving review that satisfies your repository's required-approval rule"
  ([concepts](https://docs.github.com/en/copilot/concepts/agents/code-review)).
- **CodeRabbit** ([configuration](https://docs.coderabbit.ai/reference/configuration)):
  `request_changes_workflow` requests changes, and approves once its
  comments are resolved; `pre_merge_checks`: "`off` disables the check,
  `warning` posts a warning, and `error` requires resolution before
  merging"; `auto_incremental_review` re-reviews on each push.
- **reviewdog** ([README](https://github.com/reviewdog/reviewdog)): a filter
  — `added` (the default), `diff_context`, `file`, `nofilter` — keeps
  findings to the lines a change touches; many reporters; `-fail-level`
  sets what fails the job, none by default.
- **GitHub Agentic Workflows, safe outputs**
  ([reference](https://github.github.com/gh-aw/reference/safe-outputs/)):
  the agent asks for review comments on code lines, a separate job writes
  them, ten by default.
- **Google's engineering practices**
  ([the standard](https://google.github.io/eng-practices/review/reviewer/standard.html)):
  "favor approving a CL once it is in a state where it definitely improves
  the overall code health … even if the CL isn't perfect"; optional polish
  prefixed "Nit: ".

## Measurement

- **Signal65**, March 2026
  ([PDF](https://signal65.com/wp-content/uploads/2026/03/Signal65-Insights_Evaluating-AI-Code-Review-Tools.pdf)):
  CodeRabbit, Cursor BugBot, Copilot, Greptile and Qodo Merge on 6
  repositories, 10 bug-introducing pull requests each; precision 95.88%
  (CodeRabbit), 95.95% (BugBot), 64.35% with 41 false positives (Copilot).
  A companion infographic is CodeRabbit-branded: read it as a vendor's.
- **Kodus**: across tools, recall only, 30 of 38 known issues
  ([benchmark](https://kodus.io/en/benchmark-ai-code-review)); recall
  against false positives within its own configurations, 72 bugs for 170
  false positives, then 84 for 328
  ([recall](https://kodus.io/en/ai-code-review-recall/)): more found costs
  more noise, roughly double for a sixth more.

## What workline takes, and what it refuses

| Taken | From |
|---|---|
| Lenses as finders, each in a context of its own; commit messages are testimony | BMAD |
| A verification step for every important candidate, at another model when one is there | Claude Code Review; ADR-0005 |
| A finding floor on candidates only, its value to be measured (#90) | BMAD's formula |
| Important and Nit, pre-existing apart | Claude Code Review |
| Skip what is not code, and what was reviewed already | the plugin; CodeRabbit's incremental review |
| Never drop silently: each finding dropped is said, with why | BMAD's triage |
| Warn before block, the level per rule | CodeRabbit's pre-merge checks; role-outcome.md |
| Findings kept to what the change touches | reviewdog's `added` filter, decided by the cause's quote |
| A cap on what is posted, the rest counted | gh-aw's safe outputs |
| A loop bounded at five rounds, then a person | BMAD's non-convergence |
| Never approve, never merge | Copilot's default, Claude Code Review's neutral check |

| Refused | Why |
|---|---|
| A minimum of *posted* findings (v6's 3 to 10) | it manufactures noise; the floor stays on candidates, which the engine and the judge filter |
| Self-triage by the same model (Qodo's self-reflection) | ADR-0005: the best independence available, said |
| The reviewer patching (BMAD's `patch` route) | one role, one job: the author fixes (principle 2) |
| A menu asking the person at each step | people at both ends only (principle 1) |
| Approving (Copilot's preview, CodeRabbit's auto-approve) | the person merges |

## Gaps no tool covers

- Telling mechanically whether a finding is the change's or was there
  before: Claude Code Review labels Pre-existing by the model's say.
  workline decides by where the quoted cause lies, in the diff or not.
- Sending what was there before to the backlog, once, rather than to the
  pull request's author.
- Re-finding the finder's quote before anything is shown.
