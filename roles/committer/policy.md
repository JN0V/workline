- Subject: `type(scope): summary`, imperative, plain language, under the limit.
  Someone who has never seen the project understands it.
- A rewrite keeps the original type and scope, unless the finding is about
  them, and keeps the author's words wherever they were not the problem.
- Body: only when the why is not obvious from the subject, and a few lines at
  most. The explanation of a design belongs in the docs or the changelog.
- No internal code (ticket, criterion, finding id) in the subject. If one is
  useful for traceability, put it in a trailer at the end: `Refs: AC-3`.
- Keep every trailer of the original message (`Refs:`, `Co-Authored-By:`…)
  exactly as written, at the end.
- One commit, one change.
