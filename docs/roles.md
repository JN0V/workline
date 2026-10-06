---
sources: [routing.default.yaml, roles/committer/role.yaml, roles/documentalist/role.yaml, roles/reviewer/role.yaml, roles/product-owner/role.yaml, roles/judge/role.yaml, roles/auditor/role.yaml]
checked: 194bf36
verified: agent:claude-code
---
# The roles

Each role does one job, reads only what it needs, and works without AI
([principles](PRINCIPLES.md)). Its page says what it does and does not do,
its events, every setting with its default, what it writes, what it costs
and where it stands.

| Role | Does | Runs on | Page |
|---|---|---|---|
| Committer | checks each commit's message, secrets and author; rewrites a refused message | `commit-msg` (the git hook), `pre-push`, `merge-request` | [committer](../roles/committer/README.md) |
| Documentalist | keeps the docs true to the code they name | `pre-push`, `merge-request`, `schedule`, `release`, `init` | [documentalist](../roles/documentalist/README.md) |
| Reviewer | reads the code a change brings and says what it breaks, what the issue it closes asks and it leaves out, and what its author claims and it contradicts; asks a person what only a person decides; never approves | `review` (`workline review`), `merge-request` (opt-in) | [reviewer](../roles/reviewer/README.md) |
| Product owner | keeps the backlog — the open issues — true to the code and in order; opens its report with what is next and what is stuck; says on a split need what its parts delivered, for a person to accept; flags the issues built on a need a person changed; a weekly sample of its acts, for a person to judge | `schedule` (opt-in), `import`, the weekly sample | [product owner](../roles/product-owner/README.md) |

Which role runs on which event is the routing: `routing.default.yaml` as
shipped, changed in `.workline/config.yaml` ([config](config.md)). By
default: `commit-msg` → committer; `merge-request` → committer,
documentalist; `schedule` and `release` → documentalist. `workline init`
routes `pre-push` to the committer and the documentalist; `--review` adds
the reviewer to `merge-request`.

Two more roles run behind the others, never on an event of their own: the
**judge** answers one yes-or-no question a role's check cannot (a
reviewer's finding, a closing as obsolete), from another context or model
than the one judged; the **auditor** reads the weekly sample of the docs
the documentalist vouched for (`workline sample`). The same sample draws
the product owner's acts of the week, for a person, with no agent.

How to run each role from a hook, a script or a CI job:
[triggers.md](triggers.md). What a role is, for those who write one:
[the role contract](spec/role-contract.md).
