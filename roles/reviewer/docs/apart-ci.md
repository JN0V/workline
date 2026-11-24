---
sources:
  - roles/reviewer/role.yaml#apart
  - internal/line/line.go#RunWith
  - roles/reviewer/code/lines.go#reviewOf
  - defaults/ci/github/workline-apply.yml#jobs.verdict
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:bot-token
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:child
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn
checked: 0bbe8345
verified: human:JN0V
---
# Reviewer — the verdict, and the run apart in CI

Part of [the lenses that warn, apart from the verdict](apart.md).

## What the verdict holds, and what it never waits for

- **It holds** the committer, the gates, the documentalist, the rules,
  and the lenses that block with their judges.
- **It never waits** for a lens that only warns, nor its judges.
- **A warn finding arriving after the verdict** is shown, in its comment
  and, important, on its line; it asks nothing and never changes a
  verdict given.
- **A lens that fails** — an answer that never reads, the tokens spent —
  says so in that comment (`lens-failed`). Its record holds none of the
  commits and names the lens (`failed=`): the next push, or the run run
  again, asks it first, whosever turn it is
  ([the record](run-record.md#3-the-record),
  [what goes wrong](verdict.md#8-the-verdict-and-what-goes-wrong)). The agent out of reach is said
  in the job's log. It never
  turns a pass into a fail, nor a fail into a pass; the job stays green,
  only the engine's own error fails it.
- **The merge policy merges on the head's verdict**
  ([ADR-0042](../../../docs/adr/0042-the-engine-merges-by-the-projects-policy.md)):
  warn findings, and a lens that warns failing, may come after it, and
  after the merge. Read their comment before you accept, or accept
  knowing it may still post.

## In CI

- **GitHub**: workline.yml's `propose` leaves them (`--warn-apart`);
  [workline-warn.yml](../../../defaults/ci/github/workline-warn.yml) asks them
  beside it (`--warn-only`); workline-apply.yml applies both, and keeps
  the verdict from `propose` alone
  ([CI on GitHub](../../../docs/ci-github.md#set-up)). Without workline-warn.yml,
  `propose` warns that nothing reads them; with an older
  workline-apply.yml, which does not name it, `warn` warns and reads
  nothing: unposted, its record would never move.
- **GitLab**: `workline:warn`, after `workline:gates`, then
  `workline:warn:apply`, with the agent's token and the bot's; without
  the bot's, `workline:warn` says so and stops, and the lenses that warn
  read in `workline:propose`, said in its log and summary. Both run in a
  child pipeline the merge request's does not wait for: "merge when the
  pipeline succeeds" waits for the verdict alone
  ([what the template does](../../../docs/gitlab/template.md)).
- **No agent, or neither the reviewer nor the inspector in the line**:
  the run apart ends at once.
- **The inspector's triage** reads in the same run, given what the gates
  found, its own comment beside this one; with it routed, the run waits
  for the gates' job first, but on a draft on GitHub
  ([the inspector](../../inspector/README.md),
  [its triage in CI](../../../docs/gates-triage.md)).
