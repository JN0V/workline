---
sources:
  - roles/product-owner/role.yaml#settings.issues-per-run
  - roles/product-owner/role.yaml#settings.autonomy
  - roles/product-owner/role.yaml#settings.acts
  - roles/product-owner/role.yaml#levels
  - defaults/ci/github/issue-form-need.yml#name
  - defaults/routing.yaml#fail-fast
checked: 3f432ab14
verified: agent:claude-code
---
# The product owner in CI

Part of [setting up workline in CI](ci.md).

To keep the backlog too
([ADR-0018](adr/0018-the-product-owner.md)), add it to the schedule line;
gardening then runs it after the documentalist, even when the
documentalist blocks: the job fails all the same
([ADR-0037](adr/0037-the-schedule-runs-every-step.md)).

```yaml
routing:
  events:
    schedule: [documentalist, product-owner]
roles:
  product-owner:
    settings:
      issues-per-run: 8        # issues read a night
      # autonomy: cautious     # a person is the Product Owner: what sets direction is proposed (ADR-0026)
      acts:
        close-duplicate: {mode: propose}   # a person closes, until trust is earned
```

What it does, its acts and settings: [its page](../roles/product-owner/README.md).
In short:

- what it proposes goes on each issue: the label `workline:proposed`
  (GitLab: `workline::proposed`) and its one comment there; a saved
  filter on the label is what waits on you; the night's summary is in
  the job's summary ([ADR-0038](adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md));
- it refines issues to `ready`, drafts labelled `workline:draft`; the label
  `workline:accepted`, on one issue or many from the list, is your yes:
  done at the next run, with no agent; a comment asks it to revise;
- an outsider's issue gets the sections proposed in a comment; the same
  label agrees for the project, and a reply `agreed`, by its reporter or a
  person of the project, has them written
  ([ADR-0021](adr/0021-the-product-owner-talks-with-the-reporter.md)); it
  asks again only after an answer, three times at most; past its wait, it
  goes on, its assumption written ([ADR-0060](adr/0060-the-person-is-on-the-loop.md));
- an issue form with the sections an issue holds,
  [defaults/ci/github/issue-form-need.yml](../defaults/ci/github/issue-form-need.yml), copied
  to `.github/ISSUE_TEMPLATE/need.yml`, has a person's issue arrive with
  them written;
- its acts turned on (off by default), it also splits, renames, orders
  and says what an issue waits on ([ADR-0022](adr/0022-a-split-keeps-the-need-a-rename-keeps-a-persons-title.md),
  [ADR-0028](adr/0028-an-issue-names-what-it-waits-on.md)): a person's
  title, priority or link is kept.
