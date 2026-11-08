#!/bin/sh
# A cmd: agent that keeps issue 3 blocked for a person, read again for its
# size (ADR-0064, step 3): its need cannot be narrowed. Whoever asks it
# gets the same answer; a role that may not block refuses it.
cat >/dev/null
cat <<'TEXT'
- blocked: {issue: 3, why: "one file of 100,000 characters is the whole need: no smaller Scope proves it", person: true}
TEXT
