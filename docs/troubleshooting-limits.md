---
sources:
  - roles/documentalist/role.yaml#settings.ai-max-tokens
  - roles/committer/role.yaml#settings.ai-max-seconds
  - internal/line/size.go#issueSeed
checked: e81df7b34
verified: agent:claude-code
---
# Troubleshooting: token caps and the agent

Part of [troubleshooting](troubleshooting.md).

## Token caps and cuts

- `ai-max-tokens` (documentalist, reviewer): the run spent its cap; the rest
  waits for the next run. The reviewer leaves the commits unrecorded
  (`review-not-whole`, `lens-failed`) and says what each call used. `ai-max-calls`, `parts-max-per-run`, the product owner's
  `issues-per-run` and `code-lines-max`, the reviewer's `diff-lines-max`,
  `code-lines-max` and `tests-lines-max` size each run (each role's page).
- `input-cut` (a warning): the agent read part of what the run gives it;
  the line says what, how much of it, and the setting that raises it.
## An issue too big

- `size-crossed`: a role's run on an issue was too big for it: its task
  over the context budget, or its token cap reached; a cut alone is not
  one. Never run again on the same input: the issue goes to
  Blocked, and the product owner reads it at once, to narrow its Scope,
  grant it `workline:size/large`, or leave it to you
  ([routing](spec/routing.md#the-line-of-states)).
- `size-crossed-again`, `size-crossed-own`, `size-crossed-top`: the
  same, the product owner already asked from that state, its own run too
  big, or the issue at `workline:size/x-large` already. Narrow the issue
  or split it, put `workline:size/x-large` on it (every size target ×4),
  or raise the setting the line names, then move it back by its label.
- `backstop-crossed`: the issue's runs, all roles, spent 4× its normal
  (1.6M fresh tokens at the seed); `large` never raises it. It waits for
  you, no product owner asked (`backstop-waits` on news), no review
  round run on its pull request. Put
  `workline:size/x-large` on it (16× its normal) and move it back by its
  label, or split it into new issues
  ([routing](spec/routing.md#the-line-of-states)).
- `size-tier` (info): every size target of the run multiplied by its
  issue's tier label; take it off to undo. `size-person`, `size-kept`,
  `size-once`, `split-depth`, `split-off`, `too-big-unasked`: a grant or
  a split on size dropped
  ([why](spec/backlog-acts/refine.md#granting-a-size)).
## The agent and its limits

- `ai-max-seconds` (committer): the rewrite took more than the 30 s it
  may; the message is refused with its reasons, as with no agent.
- `personal-setting-ignored`: your own config sets what a role does not
  let a person set; the project's stands
  ([your own config](config.md#your-own-config)).
- `review-not-whole`, "past parts-max" (reviewer): the change is too
  large to read in the parts a run allows; split it, or raise
  `parts-max` ([in parts](../roles/reviewer/docs/parts.md)). Read whole
  by a person, `workline:accepted-whole` lets it merge
  ([merge policy](merge-policy.md#what-it-does)).
- `blocked-external` (exit 3): the agent's quota, a timeout, the forge or a
  repository failed. Never a verdict: run it again later. An agent call
  failed by a passing overload (a 529, a 5xx) is asked again once first,
  after a pause (`agent-asked-again`, `WORKLINE_RETRY_PAUSE`). The CI templates
  turn it into a warning, the docs listed for a person. A GitHub or
  GitLab call is tried again first (which: [GitHub](github/calls.md),
  [GitLab](gitlab/calls.md)).
- `agent-unavailable`: the agent named cannot be called; doctor says why.
  The developer's `claude` on Linux: bubblewrap or socat missing, or a
  runner that refuses its sandbox; the finding names which
  ([its agent](../roles/developer/docs/agent.md#claude)).
- `model-changed`: another model answered an alias than last time on this
  machine; said once, never blocking.
