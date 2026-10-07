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

- **BMAD-METHOD's code review**
  ([BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD)). Several
  finders, each a subagent in a context of its own: one given the diff and
  nothing else; one for edge cases, following each branch and boundary the
  change touches; one for the verification gap — would the checks fail if
  the behaviour broke where it is used; one for intent, the change against
  the issue it links. The author's account (commit messages, the pull
  request's body) is a set of claims to contest, not evidence. Only the
  diff-only finder has a floor on its number of candidates:
  min(floor(sqrt(kB) + 1), 10), kB the size read in kilobytes. A triage step
  sets each finder's severity aside, verifies each claim at the file and
  line it cites, and gives every finding a verdict and a route — fix it,
  ask for a decision, or defer it — none dropped without one. A repair loop
  stops after five rounds and hands over to a person. An earlier version
  required a minimum number of *reported* findings; its changelog later
  found a harsh reviewer persona did nothing for the bugs left, while a
  floor on concrete findings and asking what is missing did.
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
| Lenses as finders, each in a context of its own; commit messages are claims to contest | BMAD-METHOD |
| A verification step for every important candidate, at another model when one is there | Claude Code Review; ADR-0005 |
| A finding floor on candidates only, its value to be measured (#90) | BMAD-METHOD's formula |
| An intent lens: the change against the issue it closes, what it asks and what its Scope leaves out (#126) | the intent finder above; Qodo's ticket compliance |
| The commit messages and the pull request's body given as claims, each contradicted one a finding quoting the claim and the line (#126) | the author's account as claims to contest, above |
| A facet given the diff and nothing else, in a call of its own, its findings judged like the others; off by default, as it found nothing the lenses missed on #90's cases (#126) | the diff-only finder above |
| Important and Nit, pre-existing apart | Claude Code Review |
| Skip what is not code, and what was reviewed already | the plugin; CodeRabbit's incremental review |
| Never drop silently: each finding dropped is said, with why | BMAD-METHOD's triage, verified at each finding's line |
| Every finding routed: its author fixes, an issue defers, a person decides | BMAD-METHOD's fix, decision and defer routes, chosen here by the cause's quote |
| Warn before block, the level per rule | CodeRabbit's pre-merge checks; role-outcome.md |
| Findings kept to what the change touches | reviewdog's `added` filter, decided by the cause's quote |
| A cap on what is posted, the rest counted | gh-aw's safe outputs |
| A loop bounded at five rounds, then a person | BMAD-METHOD's bounded repair loop |
| Never approve, never merge | Copilot's default, Claude Code Review's neutral check |

| Refused | Why |
|---|---|
| A minimum of *reported* findings (BMAD-METHOD's earlier version) | it manufactures noise; the floor stays on candidates, which the engine and the judge filter |
| Self-triage by the same model (Qodo's self-reflection) | ADR-0005: the best independence available, said |
| The reviewer patching (BMAD-METHOD's fix route) | one role, one job: the author fixes (principle 2) |
| A menu asking the person at each step | people at both ends only (principle 1) |
| Approving (Copilot's preview, CodeRabbit's auto-approve) | the person merges |

## Gaps no tool covers

- Telling mechanically whether a finding is the change's or was there
  before: Claude Code Review labels Pre-existing by the model's say.
  workline decides by where the quoted cause lies, in the diff or not.
- Sending what was there before to the backlog, once, rather than to the
  pull request's author.
- Re-finding the finder's quote before anything is shown.
- Re-finding the author's claim, and the issue's words a finding quotes,
  before it is judged: Qodo grades a whole pull request against its
  ticket, with no quote to check (#126).

## Spec review (2026-10-07, #128)

What reviews a spec before it is built, in the field's words:
*requirements smells*, *requirements quality*, *spec analysis*.

- **Requirements smells** (Femmer, Méndez Fernández, Wagner, Eder,
  [arXiv:1611.08847](https://arxiv.org/abs/1611.08847), JSS 2017): eight
  smells from ISO/IEC/IEEE 29148's language criteria — subjective
  language, ambiguous adverbs and adjectives, loopholes, open-ended
  non-verifiable terms, superlatives, comparatives, negative statements,
  vague pronouns — found by their tool Smella at 59% precision, 82%
  recall on industrial specs; precision 0.26 to 0.96 by smell.
- **GitHub Spec Kit's `analyze`**
  ([templates/commands/analyze.md](https://github.com/github/spec-kit/blob/main/templates/commands/analyze.md)):
  read-only, before implementation; passes for duplication, ambiguity,
  underspecification, coverage gaps and inconsistency; severities
  critical to low. Its `clarify` asks at most five questions and writes
  the answers into the spec.
- **Taken**: ambiguity and verifiability as two lenses (29148's
  unambiguous, verifiable); a scope lens for Spec Kit's coverage gaps; a
  question for a person rather than a defect, as `clarify` does
  (`decision`, #126); read-only, before the build.
- **Added**: each finding's quote found again in the spec, the important
  ones judged; a lens reading the code the spec names (no tool checks a
  spec against today's code).
- **Refused**: word lists as rules (Smella's precision on vague pronouns
  and comparatives would block on noise); writing answers into the spec
  (the product owner does, part 2 of #128).
