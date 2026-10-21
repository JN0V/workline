<!-- workline
sources: [internal/forge/gitlab.go]
checked: 73bdc933
-->
# The product owner on GitLab

From [workline on GitLab CI](../ci-gitlab.md#the-product-owner).

- a project that runs only the product owner may give its token the
  Reporter role and `api` alone; not Planner, which may not give a split's
  child its parent;
- who is of the project is a member with the Planner role or above; a
  Guest's issue, or a stranger's, is its reporter's;
- on GitLab Free, which keeps `blocks` links for Premium, what an issue
  waits on is a line `Blocked by #12.` in its body.
