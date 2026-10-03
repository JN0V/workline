#!/bin/sh
# A cmd: agent whose every answer is prose around YAML, as Sonnet answered
# the product owner once on DomoticsCore's roadmap: nothing of it reads.
cat >/dev/null
cat <<'TEXT'
```yaml
- note: "a first try"
```

Retracting that: the final answer is below.

```yaml
- note: "the issues still hold"
```
TEXT
