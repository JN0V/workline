<!-- workline
sources: [defaults/ci/github/workline-bounds.yml, defaults/ci/github/workline-merge.yml, internal/doctor/event_policy.go]
-->
# pull_request_target in a public repository

From [workline on GitHub Actions](../ci-github.md#public-repositories-and-pull_request_target).
GitHub's own pages: [the default policy](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target#default-policy-for-pull_request_target),
[event policies](https://docs.github.com/en/actions/how-tos/administer/control-workflow-execution),
[their REST API](https://docs.github.com/en/rest/actions/policies).

- From December 2, 2026, GitHub blocks `pull_request_target` by default
  in public repositories (brownouts on November 9 and 16, 16:00–18:00
  UTC). Private and internal repositories are not affected.
- [workline-bounds.yml](../../defaults/ci/github/workline-bounds.yml) and
  [workline-merge.yml](../../defaults/ci/github/workline-merge.yml) run on it.
  Blocked, their runs end in `startup_failure`, the required check
  `workline bounds` never reports, and no pull request merges.
- The fix is an Actions event policy that allows the event. It is an
  **allow list**: an active policy covering a workflow file blocks every
  event of that file it does not list.
- So target the two files only, and allow every event they use:

| Workflow file | Events to allow |
|---|---|
| `.github/workflows/workline-bounds.yml` | `pull_request_target` |
| `.github/workflows/workline-merge.yml` | `pull_request_target`, `issues`, `schedule` |

1. Settings → Actions → Policies → new policy; target these two workflow
   files; a rule "Restrict events" allowing `pull_request_target`,
   `issues` and `schedule`.
2. Enforcement: **Evaluate** first where it is offered (GitHub Enterprise
   Cloud), then check **Policy insights**: no run of these files would be
   blocked. Then set it **Active**: Evaluate allows nothing.
3. Or by the REST API, as below.

```sh
gh api -X POST repos/OWNER/REPO/actions/policies --input - <<'EOF'
{"name": "workline line", "enforcement": "active",
 "conditions": {"workflow_path": {"exclude": [], "include": [
   ".github/workflows/workline-bounds.yml", ".github/workflows/workline-merge.yml"]}},
 "rules": [{"type": "restrict_action_events",
   "parameters": {"allowed_events": ["pull_request_target", "issues", "schedule"]}}]}
EOF
```

- An organization's or enterprise's policies stack with the repository's:
  each active one covering a file must allow all of its events.
- `workline doctor` reads the policies (the organization's too) and warns
  when a line workflow is left blocked, or a policy leaves out one of a
  file's events.
