# Principles

The rules every role, spec and line of engine code follows. When a design
choice is unclear, it is settled against this page.

1. **People at both ends, the machine in between.** People state the need, how
   the result is verified and who validates it; people accept the result. The
   line does the work in between.
2. **One role, one job.** Each role does one thing and hands over, like a post
   on an assembly line.
3. **Each role reads only what it needs.** No catch-all constitution; a context
   budget per role.
4. **Tools first, AI for judgement.** Scripts do everything mechanical. The AI
   is called only when a decision needs judgement.
5. **It works without AI.** No model, no quota, no network: every check still
   runs and still blocks; judgement calls go to a person.
6. **The agent proposes, the engine acts.** The agent writes proposals from a
   closed catalogue; a separate step, holding the write token and no AI key,
   validates and applies them.
7. **A rule that matters is a check.** A rule written only in prose is a wish.
   Every defect that got through ends as a proven, mechanical guard.
8. **Stay on the task.** Anything found outside the task's scope is reported,
   not fixed on the spot.
9. **Cut cascades.** Only direct consequences are handled in a change; the rest
   is listed and due at a chosen moment.
10. **Same everywhere.** A role runs the same on a laptop, in GitHub Actions, in
    GitLab CI or on a server, with any coding agent.
11. **Few dependencies.** Single binaries, pinned; nothing pulls a tree of
    packages into a job that holds tokens.
12. **Fail loud, never pass silently.** A check that did not run, a setting
    that is missing, an outside service that failed: each is said, and none is
    read as a pass.
13. **Borrow before building.** Look for what exists — in the words the
    ecosystem uses — and build only what is missing.
