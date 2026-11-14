# Principles

The rules every role, spec and line of engine code follows; an unclear
choice is settled against this page.

1. **People at both ends, the machine in between.** People state the need, how
   the result is verified and who validates it; they accept the result, or
   write beforehand the narrow class the line accepts for them.
2. **One role, one human job.** Each role holds the job, craft and scope
   of a human role, and hands over, like a post on an assembly line.
   Unsure what it should do? Ask what a human in that post would do
   ([ADR-0066](adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md)).
3. **Each role reads only what it needs.** No catch-all constitution; a context
   budget per role.
4. **Tools for what converges, AI for judgement.** Scripts do what is
   mechanical and converges
   ([ADR-0065](adr/0065-code-only-what-converges-language-knowledge-in-packs.md)).
   For now, where code would make a role rigid, the AI goes first; its
   independent parts become code later, to save tokens
   ([ADR-0066](adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md)).
   A new role starts as a prompt on the engine's plumbing, safe and
   observed from day one
   ([ADR-0068](adr/0068-the-line-steps-back-on-its-own-drift.md#2-prompt-first-code-what-converges)).
5. **Its checks work without AI.** No model, no quota, no network: every
   check on converging facts still runs and still blocks; the agent stays
   a choice. Relaxed for now: a role need not work with agent `none`
   where that makes it rigid; with `none` it says what it skipped
   ([ADR-0066](adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md)).
6. **The agent proposes; the forge's guards decide.** What an agent writes
   reaches the default branch only through the forge's guards — protection,
   required checks, review, the merge policy — which hold whoever pushes.
   Where the forge splits rights natively, as in CI, the agent's step holds
   no write token; a separate step applies its proposals.
7. **A rule that matters is a check.** A rule written only in prose is a wish.
   A defect that got through ends as a conformance or evaluation case
   failing without its fix. The case is the default; a rule code, a
   setting or a marker (a label or comment the engine reads) is added
   only where a case cannot hold the defect, and its issue says why
   ([ADR-0068](adr/0068-the-line-steps-back-on-its-own-drift.md#3-principles-4-and-7-reworded)).
8. **Stay on the task.** Anything found outside the task's scope is handed
   to the role whose scope it is, not fixed on the spot.
9. **Cut cascades.** Only direct consequences are handled in a change; the rest
   is listed and due at a chosen moment.
10. **Same everywhere.** A role runs the same on a laptop, in GitHub Actions, in
    GitLab CI or on a server, with any coding agent.
11. **Few dependencies.** Single binaries, pinned; nothing pulls a tree of
    packages into a job that holds tokens.
12. **Fail loud, never pass silently.** A check that did not run, a setting
    that is missing, an outside service that failed: each is said, and none is
    read as a pass.
13. **Borrow before building, and while running.** Look for what the
    forge or ecosystem has, in their words: use it; if it fits partly,
    complete it; replace it only when it cannot be, and say why. A role
    brings judgment, not screens the forge has.
14. **Automatic where it can be, written where it cannot.** What the
    machine can do, it does, without being asked. What needs a person is
    written where they look — what to do, why, and how — never left in a
    job's log or in someone's memory.
15. **A guess may only pass.** A check on converging facts may block.
    A heuristic may pass a case it is sure of; its miss goes to
    judgement, never to a block nor a silent pass. Language knowledge
    is data in packs, not engine code
    ([ADR-0065](adr/0065-code-only-what-converges-language-knowledge-in-packs.md)).
