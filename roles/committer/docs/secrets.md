---
sources:
  - roles/committer/code/secrets.go#termList
  - roles/committer/code/secrets.go#Secrets
  - roles/committer/code/secrets.go#scan
  - roles/committer/code/secrets.go#MessageLeaks
  - roles/committer/code/secrets.go#scannedPart
  - roles/committer/code/secrets.go#gitleaks
  - roles/committer/code/secrets.go#commonList
  - roles/committer/code/secrets.go#stagedTermList
  - roles/committer/code/identity.go#allowedIdentities
  - roles/committer/code/identity.go#forgeIdentities
  - roles/committer/code/identity.go#Identity
  - roles/committer/code/committer.go#Pre
  - roles/committer/code/committer.go#Post
checked: dd77bf922
verified: agent:claude-code
---
# Committer — secrets and identity

Part of [the committer](../README.md).

## Secrets and forbidden terms

- **What the commit adds** (on `commit-msg`), or every commit of the
  range, is scanned with [gitleaks](https://github.com/gitleaks/gitleaks),
  before the message: no rewrite fixes a secret, so a `leak` blocks without
  asking the agent, at its file and line.
- **The message** is scanned too (`gitleaks stdin`), without what git
  strips from it (comments, the diff of `commit -v`): a `leak` there names
  its line, and the agent is not asked either, since the message would
  carry the match to it. A rewrite is scanned like the message it replaces.
- The two scans run side by side: a commit waits for one, about 0.4 s.
- The match is never printed. Without gitleaks, the commit goes on and the
  verdict says so (`secrets-not-checked`).

gitleaks' own rules find secrets. Terms that must never reach a repository
are rules of the user's, in lists kept outside it, gitleaks taking the first
config it finds:

1. `GITLEAKS_CONFIG` or `GITLEAKS_CONFIG_TOML`, when set;
2. the repository's `.gitleaks.toml`, gitignored: this repository's terms,
   which may extend the common list (`[extend] path = "…"`);
3. the common list, `workline/gitleaks.toml` in the user's config folder
   (`~/.config/` on Linux), which extends gitleaks' rules
   (`[extend] useDefault = true`);
4. gitleaks' rules alone.

A rule's id is printed: name no term in it. gitleaks lets
its own config through, so the committer refuses a commit adding a private
`.gitleaks.toml` — gitignored, or a link — (`term-list-staged`); one a
project commits to share its allowlist is not private.

## Identity

An author or committer address outside the allow list blocks (`identity`).

- **Checked**: the one git will record, on `commit-msg` — `git var`, so an
  address given for one commit (`-c user.email`, `--author`) is caught too
  — and those of every commit of the range. An empty address is refused.
- **The list**: the project's `allowed-identities` setting (patterns;
  noreply addresses, for a public repository) and the user's own, one
  pattern a line, in `workline/allowed-identities` in their config folder.
  Both empty, identities are not checked, and nothing is said.
- **The forge's own commits**: the project's `forge-identities` names the
  addresses its forge commits with — on GitHub, its App's bot
  (`<id>+<app>[bot]@users.noreply.github.com`) and the web's merges
  (`noreply@github.com`) — so a user's list lets them through
  ([#107](https://github.com/JN0V/workline/issues/107)). Each is matched
  exactly, case included, never as a pattern; alone, they never turn the
  check on.
