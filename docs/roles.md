---
sources: [routing.default.yaml, roles/committer/role.yaml, roles/documentalist/role.yaml, roles/reviewer/role.yaml, roles/product-owner/role.yaml, roles/judge/role.yaml, roles/auditor/role.yaml]
checked: 7ae4307
verified: agent:claude-code
---
# The roles

Each role does one job, reads only what it needs, and works without AI
([principles](PRINCIPLES.md)). Its page says what it does and does not do,
its events, every setting with its default, what it writes, what it costs
and where it stands.

| Role | Does | Runs on | Page |
|---|---|---|---|
| Committer | checks each commit's message, secrets and author | `commit-msg`, `pre-push`, `merge-request` | [committer](../roles/committer/README.md) |
| Documentalist | keeps the docs true to the code they name | `pre-push`, `merge-request`, `schedule`, `release`, `init` | [documentalist](../roles/documentalist/README.md) |
| Reviewer | says what a change, or a spec, breaks or leaves out; never approves | `review`, `spec`, `merge-request` and `schedule` (opt-in) | [reviewer](../roles/reviewer/README.md) |
| Product owner | keeps the open issues true to the code, refined and in order | `schedule` (opt-in), `import`, the weekly sample | [product owner](../roles/product-owner/README.md) |
| Judge | answers one yes-or-no question a role's check cannot | asked by the other roles | [judge](../roles/judge/role.yaml) |
| Auditor | re-checks a weekly sample of the docs the documentalist confirmed | `workline sample` | [auditor](../roles/auditor/role.yaml) |

## Which role runs when

Which role runs on which event is the routing: `routing.default.yaml` as
shipped, changed in `.workline/config.yaml` ([config](config.md)).

- **By default**: `commit-msg` → committer; `merge-request` → committer,
  documentalist; `schedule` and `release` → documentalist.
- **`workline init`** routes `pre-push` to the committer and the
  documentalist; `--review` adds the reviewer to `merge-request`.
- **The reviewer after the product owner** in `schedule` reads each spec it
  refines before it goes ready
  ([a spec on the forge](../roles/reviewer/docs/spec.md#on-the-forge)).

## Behind the others

Two roles never run on an event of their own:

- the **judge** answers one yes-or-no question a role's check cannot (a
  reviewer's finding, a closing as obsolete), from another context or
  model than the one judged;
- the **auditor** reads the weekly sample of the docs the documentalist
  confirmed — those whose `checked` it moved (`workline sample`). The same
  sample draws the product owner's acts of the week, for a person, with
  no agent.

How to run each role from a hook, a script or a CI job:
[triggers.md](triggers.md). What a role is, for those who write one:
[the role contract](spec/role-contract.md).
