package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/work"
)

// contracts describes each intention's shape, for the generated output contract.
var contracts = map[string]string{
	"commit-message": "- commit-message: |\n    type(scope): subject\n\n    Body, if any.\n\n    Trailer: value",
	"patch":          "- patch: |\n    a unified diff",
	"comment":        `- comment: "text of the comment"`,
	"label":          `- label: {add: [name], remove: [name]}`,
	"issue":          `- issue: {title: "short title", body: "what is wrong, where, how it was noticed", at: {path: "src/file.go", text: "the line it is about, as it reads"}}  # at: when it is about code; an issue open or closed on that line already, it is not opened again`,
	"close":          `- close: {issue: 12, reason: duplicate, duplicate-of: 7, quote: {issue: 7, text: "the words, as they read"}, why: "why"}  # or reason: obsolete, quote: {path: "src/file.go", text: "the code, as it reads"}`,
	"keep":           `- keep: {issue: 12, why: "what shows it is not solved"}  # an issue the task says is announced obsolete, kept open: a reply or the code shows it is still true`,
	"sources":        `- sources: {issue: 12, sources: ["src/file.go"], quote: {path: "src/file.go", text: "the code, as it reads"}, why: "why this is the code the issue is about"}`,
	"open":           `- open: {title: "short title", quote: {path: "docs/ROADMAP.md", text: "the item's text, as the file has it"}}`,
	"skip":           `- skip: {lines: "12-14", reason: done, quote: {path: "docs/ROADMAP.md", text: "the words that say it is done"}}  # importing only: or reason: held, issue: 7 (an issue holds it); or reason: not-item, why: "an introduction"`,
	"milestone":      `- milestone: {issue: 12, milestone: "v2.14.0", why: "why it belongs to that release"}`,
	"order":          `- order: {issue: 12, priority: 1, why: "why it comes before the others"}  # 1 the most pressing, to ` + strconv.Itoa(scale.Priority.Count()),
	"refine":         `- refine: {issue: 12, need: "one line: who needs what, and why (a draft)", example: "one real case, today and wanted (a draft)", validation: "named scenarios, each a bullet - **Name:** Given …, When …, Then … (a draft)", scope: "the part of the code", sources: ["src/file.go"], why: "why", questions: "what is still missing (an outsider's issue only)"}  # only the sections it lacks; a bug: steps instead of example — the steps, Expected, Actual, the version (a draft); description: the text above its sections in plain words, only where the task says it is a role's`,
	"ready":          `- ready: {issue: 12, why: "its four sections are there, ` + work.Name(work.Need) + ` and ` + work.Name(work.Validation) + ` a person's"}  # too-big: true on a bug the task says is past its size, its tests not yet written: they go first, then its split is asked`,
	"unready":        `- unready: {issue: 12, why: "what in the change it no longer fits"}  # a ready issue back to refine: done, its reason on the issue; a person puts workline:ready back to undo`,
	"unblock":        `- unblock: {issue: 12, to: refine, why: "what changed, so the block no longer holds"}  # a blocked issue the task says is read again: to the state it came from, or refine; done, its reason on the issue`,
	"blocked":        `- blocked: {issue: 12, why: "one line: why it stays blocked, as things are now", person: false}  # a blocked issue the task says is read again, kept blocked; person: true when only a person can lift it`,
	"ask":            `- ask: {issue: 12, questions: "what its reporter should say", why: "why the issue cannot be refined without it"}`,
	"assume":         `- assume: {issue: 12, assumption: "one plain sentence: your best answer to the question", why: "why it is the likeliest answer"}  # only where the task says the question got no answer in its delay, or your level does not wait: written under the issue's Assumptions, for a person to correct`,
	"split":          `- split: {issue: 12, into: [{title: "a short title", need: "who needs this part, and why (a draft)", validation: "who accepts it (a draft); quotes word for word, in the scenario that proves it, each item of the issue's ` + work.Name(work.Verification) + ` this part delivers", scope: "the part of the code", sources: ["src/file.go"]}, {title: "…", …}], why: "why it is several needs"}  # 2 to 6 children, each its need, validation and scope; too-big: true when the task says the issue is past its size (on even where split is off; a part never split again)`,
	"depend":         `- depend: {issue: 14, blocked-by: [13], why: "what #14 needs from #13 before it can start"}  # only what it cannot start without; a split's child says it with after: [1], the children it waits on by their place`,
	"undepend":       `- undepend: {issue: 14, blocked-by: [13], why: "why #14 no longer needs #13"}  # only a link the role set; a closed blocker's link the engine takes off itself; proposed to a person`,
	"kind":           `- kind: {issue: 12, kind: bug, why: "what in it says so: a behaviour that worked and broke"}  # bug, feature (a function people have not seen) or task (no behaviour change); only an issue no kind label tells`,
	"size":           `- size: {issue: 12, tier: large, why: "why its need is one no smaller part proves apart"}  # large alone, ×2 every size target of its runs; x-large is a person's`,
	"rename":         `- rename: {issue: 12, title: "about ten words: the problem, not the fix", why: "why the title it has does not say it"}`,
	"handoff":        `- handoff: {role: "next role", reason: "why"}`,
	"note":           `- note: "a message for a person"`,
	"finding":        `- finding: {lens: correctness, severity: important, title: "what is wrong, in a few words", why: "how it fails, and when", cause: {path: "src/file.go", quote: "the line that causes it, as it reads"}, symptom: {path: "src/other.go", quote: "where it shows, as it reads"}, fix: "what would fix it, in a sentence", seen: "what a person using the project sees when it fails, in plain words"}  # lens: the one that found it; severity: important or nit; symptom only when elsewhere; claim: "the author's words, as they read", only for a lens checking claims; a cause in an issue the change closes: path "#4"; decision: "the question for a person", a key of the finding, only for a choice that is not a defect`,
	"triage":         `- triage: {finding: t1, matters: "why it matters in this project, in its own words", fix: "what to change, in a sentence", false-positive: false, why: "why it looks like a false positive", order: 1}  # finding: the id the task gives it; why only with false-positive: true; order: its place among every finding listed, 1 to fix first; one the task says was triaged before: {finding: r1, order: 2}, its order only`,
	"claim":          `- claim: {lines: "12-14", status: contradicted, quote: "the doc's words, as they read", source: {path: "src/file.go", lines: "40-41", quote: "the source's words, as they read"}, why: "why they disagree"}`,
	"read":           `- read: {path: "src/file.go", lines: "40-80", why: "what you look for there"}  # or name: "Decide", where it is defined; or grep: "a_word", the lines that name it, always inside read:; instead of a note, when the task lets you read`,
	"quote":          `- quote: {path: "src/file.go", text: "the lines that refute the finding, as they read"}  # beside a note saying no`,
	"covers":         `- covers: {scenario: "Two by three", tests: ["CalcTest.twoByThree"]}  # one for each scenario, its name as the issue writes it; each test as the project's test report names it, its class and name dot-joined, or its name alone; one test may cover several scenarios; or reason: "why no test can check it", in place of tests`,
}

// engineOnly are the kinds the engine makes and the agent never writes,
// left out of the output contract: an effort is split from an order
// (ADR-0040), the task telling how when the project keeps one; the
// tester's Verification is written from the tests it proved (ADR-0066).
var engineOnly = map[string]bool{"effort": true, "verification": true}

// BlockScalars is how a text holding code is written so that it reads: a
// plain text starting with a backtick, or holding `: ` or ` #`, does not
// (#138), and a block scalar takes code as it is, with no escaping. Said in
// the prompt, and again when an answer did not read.
const BlockScalars = "An item may be written over several lines, one key a line. A text holding code, a quote, a backtick, `: ` or ` #`, or several lines is written as a block scalar: the key, then `|`, then the text on the lines below, indented under the key with spaces (its first line never starting with a tab)."

// userConfigDir and userHomeDir are the person's folders; a test sets its own.
var userConfigDir, userHomeDir = os.UserConfigDir, os.UserHomeDir

// facetDirs are the folders a facet is looked for in, in order: the
// project's copy; the person's, in the folder their config.yaml is read
// from (~/Library/Application Support/workline on macOS), then in
// ~/.config/workline, where they were read from before; the role's own
// (docs/spec/role-contract.md, "Facets").
func facetDirs(req Request) []string {
	dirs := []string{filepath.Join(req.Repo, ".workline", "roles", req.Role.Name)}
	if cfg, err := userConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(cfg, "workline", "roles", req.Role.Name))
	}
	if home, err := userHomeDir(); err == nil {
		if d := filepath.Join(home, ".config", "workline", "roles", req.Role.Name); d != dirs[len(dirs)-1] {
			dirs = append(dirs, d)
		}
	}
	return append(dirs, req.Role.Dir)
}

// facet finds a facet file in the first of facetDirs that holds it.
func facet(req Request, name string) (string, bool) {
	for _, dir := range facetDirs(req) {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return string(data), true
		}
	}
	return "", false
}

// Prompt assembles what the agent reads, in the contract's order. The
// system prompt is what every call of the role shares, byte for byte from
// one task to the next: persona, knowledge, instruction and output
// contract, which Claude Code caches and the role's next call reads back
// (#1128). The user message is what the call alone holds: the task, the
// skills its pre named, a refused answer's feedback, and the policy,
// last on purpose.
func Prompt(req Request) (system, user string, err error) {
	persona, _ := facet(req, "persona.md")
	if persona == "" {
		persona = fmt.Sprintf("You are the %s of a software project. %s", req.Role.Name, req.Role.Mission)
	}
	var s strings.Builder
	s.WriteString(strings.TrimRight(persona, "\n") + "\n\n")
	for _, k := range req.Role.Context.Knowledge {
		text, ok := facet(req, filepath.Join("knowledge", k))
		if !ok {
			return "", "", fmt.Errorf("knowledge file %q not found", k)
		}
		s.WriteString(text + "\n\n")
	}
	name := "instruction.md"
	if req.Role.InstructionFacet != "" {
		name = req.Role.InstructionFacet // a kind of task's own (role.ForTask)
	}
	instruction, ok := facet(req, name)
	if !ok {
		return "", "", fmt.Errorf("role %q has no %s", req.Role.Name, name)
	}
	s.WriteString(instruction + "\n\n")
	tree := Worktree(req.RunDir) != ""
	if tree {
		// It edits in place (ADR-0056): its diff is its code, no patch.
		s.WriteString("# Answer\n\nYour working directory is a worktree of the code: edit its files there and run its tests. " +
			"Its diff is your code, held to the task's bounds once you are done.\n\n" +
			"When you are done, your last message is read by the engine, not by a person: YAML only, nothing before or after it, no code fence. " +
			"`[]` when you propose nothing beside your edits; else a list, each item holding exactly one of:\n\n")
	} else {
		s.WriteString("# Answer\n\nAnswer with YAML only, no prose and no code fence: a list of proposals, each item holding exactly one of:\n\n")
	}
	for _, k := range req.Role.Intentions {
		if !engineOnly[k] && (!tree || k != "patch") {
			s.WriteString(contracts[k] + "\n")
		}
	}
	if tree {
		s.WriteString("\n" + BlockScalars + "\n\nBeyond the worktree you cannot act, only propose; anything else is refused.\n")
	} else {
		s.WriteString("\n" + BlockScalars + "\n\nYou cannot act, only propose; anything else is refused.\n")
	}
	system = s.String()
	var b strings.Builder
	task, err := os.ReadFile(filepath.Join(req.RunDir, "in", "task.md"))
	if err != nil {
		return "", "", err
	}
	b.WriteString("# Task\n\n" + string(task) + "\n\n")
	skills, err := Skills(req)
	if err != nil {
		return "", "", err
	}
	b.WriteString(skills)
	if fb, err := os.ReadFile(filepath.Join(req.RunDir, "out", "feedback.md")); err == nil {
		b.WriteString("# Your previous answer was refused\n\n" + string(fb) + "\n" +
			"Your new answer replaces it whole: give again, unchanged, every proposal and every change of it not refused above, and change only what is refused.\n\n")
	}
	if policy, ok := facet(req, "policy.md"); ok {
		b.WriteString("# Rules\n\n" + policy)
	}
	user = b.String()
	if budget := req.Role.Context.Budget; budget > 0 {
		chars := len(system) + len(user)
		if tokens := Tokens(chars); tokens > budget {
			return "", "", fmt.Errorf("%w: about %d tokens (%d characters at %s), over the role's budget of %d", ErrOverBudget, tokens, chars, TokensRatio, budget)
		}
	}
	return system, user, nil
}

// SkillsFile, in a run's in/, names the shared skills the run loads, one
// a line: the role's pre writes it when its task calls for one.
const SkillsFile = "skills"

var skillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Skills is the text of the shared skills the run's pre named, each after
// the task, under its own heading, its frontmatter left out: the role
// decides when a skill is due, so its body is given whole, not its
// description first (Agent Skills' progressive loading). A skill is looked
// up as a facet (skills/<name>/SKILL.md), then in the project's
// .workline/skills/, then as shipped; one the role does not declare in
// context.skills is refused.
func Skills(req Request) (string, error) {
	data, err := os.ReadFile(filepath.Join(req.RunDir, "in", SkillsFile))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, name := range strings.Fields(string(data)) {
		if !skillName.MatchString(name) || !slices.Contains(req.Role.Context.Skills, name) {
			return "", fmt.Errorf("skill %q: not one role %s declares in context.skills (%s)", name, req.Role.Name, strings.Join(req.Role.Context.Skills, ", "))
		}
		rel := filepath.Join("skills", name, "SKILL.md")
		text, ok := facet(req, rel)
		if !ok {
			if d, err := os.ReadFile(filepath.Join(req.Repo, ".workline", rel)); err == nil {
				text, ok = string(d), true
			} else if d, err := fs.ReadFile(workline.Skills, "skills/"+name+"/SKILL.md"); err == nil {
				text, ok = string(d), true
			}
		}
		if !ok {
			return "", fmt.Errorf("skill %q not found", name)
		}
		fmt.Fprintf(&b, "# Skill: %s\n\n%s\n\n", name, strings.TrimSpace(skillBody(text)))
	}
	return b.String(), nil
}

// skillBody is a SKILL.md without its frontmatter.
func skillBody(text string) string {
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		if _, body, ok := strings.Cut(rest, "\n---\n"); ok {
			return body
		}
	}
	return text
}

// Tokens estimates what a prompt of so many characters (bytes) costs, before
// the call, with no tokenizer: 711 + 0.82 a character. Fitted on ten real
// calls of the reviewer, Go code and diffs, Claude Sonnet and Opus, 87k to
// 117k characters (#147, roles/reviewer/docs/tried.md, 2026-10-06), within
// 2% of what Claude reported. On the documentalist's evaluation calls
// (Sonnet and Opus, 6k to 43k characters) it reads Markdown 5 to 15% high
// and C++ sources 10% low (#235): one ratio, code and prose alike, since
// prose here is far from four characters a token.
func Tokens(chars int) int { return 711 + chars*82/100 }

// TokensRatio says the estimate, in the refusal.
const TokensRatio = "711 + 0.82 a character"
