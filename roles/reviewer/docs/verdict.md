---
sources: [roles/reviewer/role.yaml, roles/reviewer/prompts/lenses, roles/reviewer/code, internal/codenav]
checked: e81df7b34
verified: agent:claude-code
---
# Reviewer — the verdict, the budget

Part of [the reviewer](../README.md): step 8 of [a run](run.md), and what a run may spend. Step 7: [the judge](judge.md).

## 8. The verdict, and what goes wrong

- The rules block (the long comment warns); one the project sets to `warn`
  in `enforce` only warns, and never holds the lenses
  ([#550](https://github.com/JN0V/workline/issues/550)).
- What the lenses find warns (`ai-findings: warn`) until the evaluation
  has measured it ([#90](https://github.com/JN0V/workline/issues/90)),
  lens by lens ([settings](settings.md)).
- A question for a person neither warns nor blocks, the summary counting
  it.
- Past `findings-max` on the change, or `issues-max` outside it, the rest
  is counted.
- One summary comment on a merge request, edited each run
  (`forge-writes`), a blocked run's too ([on a merge request](run.md#on-a-merge-request)).
- A finding that blocks stands until a commit answers it
  ([below](#a-block-stands-until-a-commit-answers-it)).
- Each finding judged leaves a hidden line in the summary comment, every
  run's, for the weekly sample
  ([below](#every-finding-judged-kept-for-the-weekly-sample)); the lenses
  that warn, read apart, in their own comment ([apart](apart.md)).

When something fails:

- A lens whose answer does not read is asked again once, with what the
  YAML reader said, or where the keys of an item holding several go
  (`promote-after`).
- A lens that fails is said (`lens-failed`), the commits left unrecorded
  and the lens named in the record: the next run asks it first
  ([the record](run-record.md#3-the-record)).
- An agent unreachable: `blocked-external`.
- No agent: the rules alone, the change left for a person (`not-reviewed`).

### A block stands until a commit answers it

[#326](https://github.com/JN0V/workline/issues/326): the record keeps
each finding that blocked, its cause quoted, beside the commits reviewed
([the record](run.md), step 3).

- **Nothing new to read** — the job run again on the same commits, a merge
  of the base that settled nothing, new commits changing no code, a rule
  holding the run, no agent: the block stands. No agent is asked; the
  summary comment names the finding again, marked `**blocks**`.
- **Its file moved**: it stands, found and kept on the new path
  ([#358](https://github.com/JN0V/workline/issues/358)).
- **Looked for where it was found**
  ([#570](https://github.com/JN0V/workline/issues/570)):
  - found on the head's lines (added, kept beside a removal): it stands
    only while the head still holds its quote there, never on a line
    the change removes that holds it too;
  - found on a removed line: it stands while the change still removes it;
  - recorded before this was kept: looked for on both, as before.
- **It goes** when a commit answers it:
  - its cause is no longer found on the change, on that side (the line
    changed, or removed);
  - or its lens reads the change again, in a whole review whose commits
    change code, and does not find it, or its judge says no: a fix may
    lie in another file than its cause's
    ([#1598](https://github.com/JN0V/workline/issues/1598)). Found again and verified, it is
    the new review's finding, shown once.
- **Its lens reads each push changing code**, whosever turn it is,
  beside the lens in turn, the cause's file or another, so the commit
  that may answer it is never recorded unread by it
  ([ADR-0020, amended](../../../docs/adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-28-a-blocks-lens-reads-a-commit-to-its-file)).
- **It stays** while its lens did not read the change again. One quoted
  from an issue stays until its lens reads the issue again.
- **The tester answers that no test is needed**: a judge weighs the
  reason, even with no new commit; the block goes if it holds
  ([weighed](tester-answers.md)).
- **Its lens set back to `warn`** (`ai-findings`): it goes.
- **Resolving its comment on the line does not lift it**: a person who
  disagrees merges where the forge lets them, or sets its lens to `warn`.

### Every finding judged, kept for the weekly sample

[ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md):
the auditor re-checks a share of the findings the judge kept or dropped
([the weekly sample](../../../docs/weekly-sample.md)). The summary
comment hides a line for each.

- **A line**: the lens, the place, the head the review read, yes (kept)
  or no (dropped), the lens's model and the judge's, the title.
  - What it was judged on: the patch and the finding's key, a yes's
    independence, a no's reason, so it is not judged again
    ([judged once](once.md)). A line written before them keys nothing.
- **No line**: a finding shown not verified, a nit, a question for a
  person, a spec's finding, a verdict kept, not judged again.
- **Every run's**, for the merge request's life, on GitHub and GitLab:
  each run writes the comment again, the lines before kept, its own
  added. A run that asks no lens keeps them too, and one applied after
  another it never read keeps that one's
  ([#375](https://github.com/JN0V/workline/issues/375)). GitLab keeps no
  earlier version of a comment: the lines do.
- **Within 60,000 characters**, under GitHub's 65,536: past it, the
  oldest lines go first, and the comment counts them
  (`<!-- workline:judged-lines gone=N -->`). The sample says them not
  drawn. A line takes about 400 characters, a no with its reason up to
  600: about 100 to 150 fit.
- **On a machine**: no line; the local record keeps none.

## The budget

- **`ai-max-tokens`**, 300000 a run: once spent, nothing more is asked,
  said (`ai-max-tokens`); the call that crosses it is paid. Raised from
  200000 so a large review is judged whole when its judges read
  ([ADR-0045](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md#the-token-budget));
  a run spends what it uses.
- **Not whole**: a lens not asked, a finding not judged, or a change
  past `parts-max` (`review-not-whole`), leaves the commits unrecorded,
  reviewed again next run. Several specs a run: those after the cap are not read, each said
  ([a spec on the forge](spec-forge.md)).
- **Said**: the summary gives the tokens used against the cap; each
  call's, by lens and by judged finding, is in `out/review.json`.
- **A call**: `context.budget`, 100000 tokens, estimated 711 + 0.82 a
  character before the call
  ([#235](https://github.com/JN0V/workline/issues/235)). A change past it
  is read in parts, each within it ([in parts](parts.md)); a prompt still
  larger is not sent, the lens failing, said (`lens-failed`).

What a run cost when measured: [research](../../../docs/research/code-review.md#the-reviewers-cost-measured).
