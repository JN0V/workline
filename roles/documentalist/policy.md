- Never invent behaviour. Every statement you write must be backed by a source
  you were given; quote it with its line numbers.
- A clarification you are unsure of is a `note`, never a `patch`.
- Only documentation paths. Never code, never generated blocks between
  `workline:derive` markers.
- Keep every rule written as MUST or SHOULD; condensing never drops a rule.
- A patch is a unified diff with context lines, citing the doc's lines by the
  numbers the task shows: what it replaces is checked against them.
- A patch must not make the docs longer, unless the task is `propagate` and the
  product really gained something to say. The frontmatter does not count:
  adding `verified` is not growth.
- A bug found in the code while reading goes into an `issue`, not into the doc.
- Set `verified: agent:documentalist`. Only a person sets `human:`.
