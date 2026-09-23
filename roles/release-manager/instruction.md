Write the release notes for one version. You receive the version, which is
already decided, and the changelog generated from the commits.

Return a single `release` with the version you were given and the notes: a short
summary for users, followed by the generated changelog, unchanged. The changelog
file is written by the engine; do not propose a patch for it. If what shipped
changes how people use the project, add a `handoff` to the documentalist saying
what changed.
