---
sources: [internal/builtin/productowner/productowner.go]
checked: edc4ffc2
verified: agent:claude-code
---
# Troubleshooting the product owner

When the product owner pauses, or an issue gets nothing. Part of
[troubleshooting](troubleshooting.md); the role:
[the product owner](../roles/product-owner/README.md).

## Paused, or nothing at all

- `proposals-waiting`: more issues wait on your answer than
  `proposals-max` (10): it reads only those you commented on. Answer
  some — `workline:accepted`, a comment, the label taken off.
  `proposal-held`: the same cap, for an issue not labelled yet; it is
  labelled once fewer wait.
- `issues-cut`: the night's share stopped where the prompt would pass
  the role's context budget; those issues are read first the next night.
  "The agent was not asked": one issue alone is over it; lower
  `code-lines-max`.
- `setting-gone`: `ignored-runs-max` or `weekly-sample` is no longer read;
  take it off.
- `values-where`, blocking: the `priority` or `effort` setting names a
  family of labels the project has none of, or GitLab's weight on GitLab
  Free; name one it has, or `auto`
  ([settings](../roles/product-owner/docs/settings.md#priority-and-effort)).
  `priority-elsewhere`: the issue holds a priority in another family; keep
  one.
- No effort set: `effort-nowhere`, the `effort` setting is unset;
  `effort-unread`, the issue names no file, or its code did not fit the
  run's budget: name its files in its Scope. `effort-once`: the role
  estimates each issue once; a person's change stands. `effort-parent`:
  GitLab sums a parent's weight from its parts'; estimate the parts.
- GitHub's fields visible to organization members only:
  `priority-blind`, the token is the job's own, an App's, or no member's:
  run with a member's token; `priority-hidden`, the act is set to propose,
  and a proposal would show the value: turn it to `act`.
- It runs only where routed: `schedule: [documentalist, product-owner]`.
- `autonomy: cautious` notes and proposes: see the issues bearing
  `workline:proposed`.

## An issue gets nothing

- An issue you commented on is read first, set aside or not: set aside,
  your comment brings it back, completed as any other unless it says "not
  now". A comment written before you took `workline:proposed` off goes
  with that "not now", and does not bring it back.
- An issue gets nothing: the summary's "read, left incomplete" says why —
  no file found for its Scope (link or name its files, or write the Scope
  yourself), or nothing drafted (read again once it changes or someone
  comments).
- An issue refined never goes ready, the reviewer after it in the line
  ([#128](https://github.com/JN0V/workline/issues/128)): `spec-not-reviewed` — the reviewer reads `specs-per-run` issues a
  run (5), the rest the next (`spec-waits`), none without an agent or
  with `forge-writes: false`; `spec-findings-open` —
  read the reviewer's comment on the issue; `spec-rounds-spent` — a
  person decides: `workline:accepted`, or `workline:ready` by hand.
- A kind of act stays proposed on an issue where a person undid one, and
  on every issue once undone `undone-max` times (`demoted`).

## A record or a file it cannot read

- A hidden record does not read
  ([#302](https://github.com/JN0V/workline/issues/302)): the issue is
  held, said each run, never read as none.
  - `state-broken`: mend the product owner's comment, or delete it.
  - `proposal-unread`: copy the proposed sections into the issue.
  - `spec-record-unread`: delete the reviewer's comment, or accept the
    issue (`workline:accepted`).
- A file the issues were imported from, archived since, flags them each
  time it is edited: name it in `archived`
  ([ADR-0032](adr/0032-a-changed-need-flags-the-issues-built-on-it.md)).
- Nothing said stuck though issues wait: `stuck-unknown` says
  which issues the forge gives no day for — the local forge keeps none, a
  plugged forge may refuse `trail` or give comments without `created`.
- `lines-unread`: an issue imported from a file, whose lines cannot be
  followed — the commit it was opened at gone after a force-push, or
  beyond a shallow clone: fetch the whole history (`fetch-depth: 0`).
