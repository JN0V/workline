---
sources: [roles/reviewer/role.yaml, roles/reviewer/instruction.md, roles/reviewer/policy.md, roles/reviewer/lenses, internal/builtin/reviewer]
checked: 2b1e9b2
verified: agent:claude-code
---
# Reviewer

What runs, for humans. The AI never reads this file.

It reads the code a change brings before a person does, and says what the
change breaks — each finding quoted, found again by the engine and checked
by a judge. It never approves, never changes the code, never merges: the
author fixes, the person merges (ADR-0020).

## A run

1. **The range.** `workline review`: `base..HEAD` (`base`, `--base`); on a
   merge request, the range CI gives. A change touching only what
   is not code (`ignore`) asks nobody.
2. **The rules**, on the comments the change adds, no agent: a story of
   the code (`bug-story`, `story-words`: "used to", "the bug was"); a code
   internal to the project (`internal-code`, by the committer's lists); a
   block over `comment-block-max` lines (`long-comment`, a warning). While
   a rule blocks, no agent is asked.
3. **The record.** Commits a review answered whole are not asked again;
   it is kept in the summary comment on a merge request,
   `.git/workline/reviewer-record` on a machine.
4. **The lenses**, each a part of the question in a context of its own
   (`lenses/<lens>.md`, a project's own in `.workline/roles/reviewer/lenses/`):
   correctness, edge cases, tests. Every one on a machine; on a merge
   request one a push, in turn (`lenses-per-push`), every one with
   `--input lenses=all`. Each gets the commits not reviewed yet (their
   messages as testimony), the change, and the files it changes, to read
   whole; it answers `finding`s: severity, title, why, its cause quoted,
   its symptom when elsewhere, a fix.
5. **The quotes.** The engine finds each cause again, spaces and line
   breaks aside, in the file at the head of the range or among the lines
   the change removed; a symptom, in its file. Not found: dropped, said
   (`finding-unfounded`). Two on one line are merged, the second said.
6. **Related or not.** A cause on a line the change added or removed: the
   change's, reported on that line, for its author to fix. Elsewhere: an
   issue, once (its key, the file and the cause's line, hidden in its body;
   an open one holding it is left),
   labelled `needs-triage`, with the product owner's state; never on the
   merge request. A nit there is left, counted.
7. **The judge**, apart, for each important finding, at the best
   independence (`judge-at-least`); its level and both models said. A no drops it, said (`finding-judged-no`).
8. **The verdict.** The rules block (the long comment warns); what the
   lenses find warns (`ai-findings: warn`) until the evaluation has
   measured it (#90). Past `findings-max` on the change, or `issues-max`
   outside it, the rest is counted. One summary comment on a merge
   request, edited each run (`forge-writes`).

A lens that fails is said (`lens-failed`), the commits left unrecorded. An
agent unreachable: `blocked-external`. No agent: the rules alone, the change
left for a person (`not-reviewed`).

## On a machine

```sh
workline review                 # main..HEAD, every lens
workline review --base develop --lenses correctness --json
```

The findings are printed; the run's `out/review.json` (its path printed, or
`--json`) gives them to the author's agent: each with its place, its cause
quoted, whether it is the change's, and how it was verified.

## On a merge request

`workline init --review` adds the reviewer to the project's merge-request
line. The judging job reads the summary comment, to know what was reviewed:
on GitHub it needs a read token (`GH_TOKEN`, `pull-requests: read`), which
ci/github/workline.yml gives it.

## Settings

```yaml
roles:
  reviewer:
    settings:
      base: main
      lenses: [correctness, edge-cases, tests]
      lenses-per-push: 1
      findings-max: 10
      issues-max: 3
      ai-findings: warn        # block, once measured
      judge-at-least: context  # model, provider
      forge-writes: true       # false: the summary and the issues only in the verdict
```

Where the role stands: [status.md](status.md).
