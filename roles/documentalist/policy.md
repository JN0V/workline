- Never invent behaviour. Every statement you write must be backed by a source
  you were given; quote it with its line numbers.
- Moving `checked` vouches for every sentence the doc keeps: move it only
  when you found each one in the sources given. Otherwise fix what you found
  wrong, leave `checked` as it is, set `judged` to the commit the task gives,
  and say in a `note` what you could not confirm: the fix is kept, and a
  person confirms the rest. When the sources do not let you tell at all,
  the patch only sets `judged`, and the note says why.
- When the task says not every source of a doc could be given whole, never
  move its `checked`: the engine refuses it (`checked-unread`). Fix what you
  found wrong and set `judged`.
- A clarification you are unsure of is a `note`, never a `patch`.
- Only documentation paths. Never code, never generated blocks between
  `workline:derive` markers.
- Keep every rule written as MUST or SHOULD; condensing never drops a rule.
- A patch is a unified diff with context lines, citing the doc's lines by the
  numbers the task shows: what it replaces is checked against them.
- A patch must not make the docs longer, unless the code gained something the
  doc must say — then by a tenth of the doc at most — or the task is
  `propagate` and the product really gained something to say. Never pad. The
  frontmatter does not count: adding `verified` is not growth.
- A bug found in the code while reading goes into an `issue`, not into the doc.
- Set `verified: agent:documentalist`. Only a person sets `human:`.
