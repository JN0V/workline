---
sources: [roles/reviewer/role.yaml#events, roles/reviewer/role.yaml#apart, roles/reviewer/role.yaml#duties, roles/reviewer/role.yaml#part-intentions, roles/reviewer/role.yaml#without-ai, roles/reviewer/role.yaml#settings, cmd/workline/main.go#reviewCmd, internal/routing/adopt.go#AddReviewer, roles/reviewer/code/reviewer.go#settings, roles/reviewer/code/reviewer.go#ahead, defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose.script]
checked: e81df7b34
verified: agent:claude-code
---
# Reviewer — outputs and cost

Part of [the reviewer](../README.md): what a run writes, what it costs, and what it does with no AI.

## Outputs

- **Outputs**: on a machine, the findings and `out/review.json` for your
  agent to fix; on a merge request, each important finding as a comment
  on its line in the forge's own review ([more](lines.md)), one summary
  comment edited each run, the nits last, hiding a line per finding judged for the
  weekly sample ([more](verdict.md#every-finding-judged-kept-for-the-weekly-sample)),
  the findings for GitHub's code scanning or
  GitLab's Code Quality (`--sarif`, `--code-quality`), and an issue
  (`needs-triage`) for each verified finding outside the change
  ([more](run-quotes.md#56-the-quotes-and-whose-finding-it-is)).

## Cost

- **Cost**: one call for the lenses:
  - all of them on a machine, together; one a push on a merge request,
    none on a draft;
  - one judge call for each important finding, up to three when it
    reads the code it needs, none for one judged before on the same
    patch ([judged once](once.md));
  - one for each tester's answer that no test is needed
    ([weighed](tester-answers.md)); a spec, one call;
  - a change too large for one call: a call a part
    ([in parts](parts.md)).
  - Capped at 300000 tokens a run (`ai-max-tokens`) and by the `*-max`
    [settings](settings.md#caps). Measured:
    [research](../../../docs/research/code-review.md#the-reviewers-cost-measured).

- **The run when a merge request becomes ready** costs more: every lens
  on the whole change, a full review as on a machine, under the same cap
  ([how the templates tell it](ready.md)); the templates split it in
  two calls, the lenses that block and those that warn, the change given
  to each ([apart](apart.md)).

## Without AI

- **Without AI**: the rules alone; the change is left for a person
  (`not-reviewed`, [more](verdict.md#8-the-verdict-and-what-goes-wrong)).
