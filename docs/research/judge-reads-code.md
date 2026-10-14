# A judge that reads the code — focused research (2026-10-14)

**Question.** How do others give an LLM verifier or judge the code it
needs — a context fixed by the tool, or the model reading and searching
the repository itself — with what caps, through which agent features,
at what cost? For the reviewer's judge, which drops true findings for
want of the proof
([measured](judge-measured.md), [#362](https://github.com/JN0V/workline/issues/362)).

**Verdict.** Code review products and research judges moved from a fixed
context to a model that locates, reads and searches, and measured gains
from it. They pay in calls and tokens, and bound it by caps on what one
read shows, on turns and on spend. The reads are either the agent's own
tools, set up differently for each agent, or tools the caller runs: the
model names what it wants, the program serves it. None of the sources
read checks a verifier's "no" against the code it read.

## Where the proof lay: the 8 refusals measured

workline's own, no agent: each judge's reason read in the reviewer's
comment and its edit history, the code read at the commit reviewed.

| Pull request | The judge said | The proof lay |
|---|---|---|
| [#280](https://github.com/JN0V/workline/pull/280) | "nothing shown proves that an empty Level meant error" | in the cause's own function, `run`, 331 lines: 31 shown |
| [#144](https://github.com/JN0V/workline/pull/144) | "never shows where `res.RunDir` is set" | in a function it calls, `run`, 271 lines |
| [#257](https://github.com/JN0V/workline/pull/257) | "the block comes from `Pre` (not shown)" | in its caller, `Pre`, 570 lines |
| [#170](https://github.com/JN0V/workline/pull/170) | "rests on code not shown" | two calls away, in `Decide` |
| [#238](https://github.com/JN0V/workline/pull/238) | "no 48000-token budget, no `taskMaxChars`" | a constant in another file, the budget in `role.yaml`, the check in another package |
| [#137](https://github.com/JN0V/workline/pull/137) | "about tests we are not shown" | a test missing: only a search shows it |
| [#199](https://github.com/JN0V/workline/pull/199) | "no test file is shown, so it only guesses" | a test missing: only a search shows it |
| [#219](https://github.com/JN0V/workline/pull/219) | weighed the code shown | nothing missing: a judgement |

- `judge-lines-max` is 200 lines today
  ([the judge's material](../../roles/reviewer/docs/judge.md#7-the-judge)).
- **A larger fixed context**, near 600 lines, would have reached the
  first three at most. The other four needed a search, or a name the
  calls of the cause's function do not give.
- **Exact replays are possible**: each run's judge questions
  (`in/judge/<key>/question.yaml`) are kept in the `workline-runs`
  artifact for 90 days. Checked on
  [#280](https://github.com/JN0V/workline/pull/280)'s run: they expire on
  2027-01-07. Its five judge calls took 3.9k to 7.0k tokens in.

## A fixed context, or the model reading

- **PR-Agent** ([dynamic context](https://raw.githubusercontent.com/qodo-ai/pr-agent/main/docs/docs/core-abilities/dynamic_context.md)):
  fixed; the hunk widened to its enclosing function, at most 8 lines
  back. Why: more context means "increasing processing time and cost".
- **Agent-as-a-Judge** ([arXiv 2410.10934](https://arxiv.org/html/2410.10934)):
  a judge with locate, read, search, retrieve. Agreement with people 90%
  against 60% for a judge with no tools. Adding read, then locate, took
  it from 76% to 90%. Memory of earlier verdicts "proved detrimental".
- **Cursor Bugbot** ([blog](https://cursor.com/blog/building-bugbot)):
  the largest gains came from "a fully agentic design", moving
  information "out of static context and into dynamic context".
- **GitHub Copilot code review**
  ([changelog, 2026-03-05](https://github.blog/changelog/2026-03-05-copilot-code-review-now-runs-on-an-agentic-architecture/)):
  "agentic tool calling to gather broader repository context as needed";
  without its runners, "a more limited review".
- **Claude Code Review** ([docs](https://code.claude.com/docs/en/code-review)):
  a verification step checks candidates "against actual code behavior".
  Which tools it uses is not said.
- **CodeRabbit** ([Google Cloud](https://cloud.google.com/blog/products/ai-machine-learning/how-coderabbit-built-its-ai-code-review-agent-with-google-cloud-run)):
  the repository cloned in a jailed sandbox; the agent writes shell
  scripts with `cat`, `grep`, `ast-grep` to navigate it.
- **Coding agents**: Claude Code's team moved from an index to "agentic
  search… glob, grep", "at the cost of latency and tokens"
  ([Latent Space](https://www.latent.space/p/claude-code)); Cline does not
  index ([blog](https://cline.bot/blog/why-cline-doesnt-index-your-codebase-and-why-thats-a-good-thing));
  Cursor measured 12.5% more accuracy with search by meaning added to
  grep ([blog](https://cursor.com/blog/semsearch)).

## The model names, the program serves

- **Agentless** ([arXiv 2407.01489](https://arxiv.org/html/2407.01489)):
  no tool loop. The model is shown the tree and names files, then shown
  their signatures and names functions, then shown those. 32% of
  SWE-bench Lite at $0.70 an issue, 78k tokens, against 23% at $1.62 and
  521k tokens for SWE-agent, reading with tools (its table; the models
  differ).
- **Client tools** ([tool use](https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview)):
  the model asks, the application runs the tool and answers. The caller
  builds the reads, the path checks, the truncation, the loop and its cap.

## Reading through the agent's own tools

- **Claude Code** ([CLI reference](https://code.claude.com/docs/en/cli-reference),
  checked in `claude --help`, 2.1.299): `--tools "Read,Grep,Glob"`;
  `--restricted` confines the file tools to the working directories;
  `--max-turns`; `--max-budget-usd`, in dollars. Read returns up to 25k
  tokens a call; nothing caps the total read. Its sandbox covers commands,
  not the file tools.
- **Codex** ([non-interactive mode](https://learn.chatgpt.com/docs/non-interactive-mode)):
  `codex exec` runs in a read-only sandbox by default; no turn nor token
  cap.
- **Gemini CLI** ([configuration](https://geminicli.com/docs/reference/configuration)):
  `tools.core` allows the read tools; policy rules deny the rest; turns
  unlimited by default.
- Each agent names its tools, its boundary and its caps differently: a
  read-only judge is set up once per agent, and kept so by that agent.

## Caps on reading

- **SWE-agent** ([arXiv 2405.15793](https://arxiv.org/html/2405.15793v3)):
  a file shown 100 lines at a time; 30 lines, or the whole file, did
  worse. A search shows at most 50 results; past that, none, and "a more
  specific query" is asked.
- **CodeVisionary** ([arXiv 2504.13472](https://arxiv.org/html/2504.13472)):
  a judge agent capped at 40 interactions.
- **Products**: a spend cap and a time limit (Claude Code Review), a
  timeout of an hour (CodeRabbit).

## Saying it cannot tell

- **Trust or Escalate** ([arXiv 2407.18370](https://arxiv.org/abs/2407.18370)):
  a judge abstains when unsure and the item goes to a stronger judge, or a
  person; over 80% agreement with people kept on almost 80% of items.

## Cost

- Agent-as-a-Judge: $30.58 and 118 minutes, against 86.5 hours of people.
- Claude Code Review: $15 to $25 a review, 20 minutes on average.
- Copilot code review: about $0.05 to $5 a review by its mode (its docs,
  read in a summary).
- workline's judge today: 3.4k to 8.4k tokens in a call on the tries
  ([reviewer's tries](../../roles/reviewer/docs/tried.md)); 3.9k to 7.0k
  on the run above.

## What workline takes, and what it refuses

| Taken | From |
|---|---|
| The judge chooses what to read: a file's lines, a definition by name, a word searched | Agent-as-a-Judge's locate, read, search |
| The model names it, the engine serves it, read-only by construction | Agentless; client tools |
| A window of lines a read, a cap on search results, a cap on rounds | SWE-agent; CodeVisionary |
| "Cannot tell", the item left to a person | Trust or Escalate |
| Replays of the exact questions asked | the run folders kept as artifacts |

| Refused | Why |
|---|---|
| The agent's own read tools as the way | one setup per agent; read-only and the cap the agent's promise, not the engine's (principles 6, 10) |
| A larger fixed context alone | 3 of the 7 at most, paid on every call |
| Memory of earlier verdicts | it carried errors from one verdict to the next |
