---
sources: [roles/product-owner/prompts/instruction.md, internal/backlog/backlog.go, internal/backlog/repeat.go, internal/work/work.go, internal/backlog/plain.go, internal/backlog/assume.go]
checked: cf8d4d9f
verified: human:JN0V
---
# Writing an issue: what the product owner says it changed

Part of [writing an issue](writing.md#what-the-product-owner-says-it-changed).

- **One line on the issue**, in its comment: "Added: Scope. Changed: —.
  See the issue's edit history."
- **Changed** lists the sections it rewrote:
  - its own drafts, or a text as it wrote it;
  - the Scope, even yours, to answer the reviewer
    ([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md)):
    it is its to keep, at every level; never the Verification, the
    tester's, a finding there answered by the tester;
  - never a sentence of your Need, Example or Validation.
- **Rewrote: the text on top as the Need, in plain words** when it took
  a role's text on top into the Need.
- **Not written: the Example, which said again what is above it** when
  it left a draft out.
- **The text before** is in the forge's edit history: "edited" above the
  description on GitHub, Gitea and Forgejo; the "changed the
  description" note on GitLab, compared with the version before on its
  Premium plan.
