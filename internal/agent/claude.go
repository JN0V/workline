package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
)

// claude runs Claude Code headless. It gets no tools, no MCP servers and no
// session, not even the Claude Code session it may be started from
// (claudeEnv), and runs outside the repository so the project's CLAUDE.md is
// not loaded: the role's facets are its whole context. Where the role gave
// it a worktree (WorktreeDir), it runs there instead, with the tools to
// read, edit, search and run commands (claudeTools), none of them asking a
// person, nor the repository's settings nor CLAUDE.md loaded: its edits
// are its code (ADR-0056). Where the role gave it a read tree instead
// (ReadTreeDir), it runs there with the tools to read and search alone
// (claudeReadTools), and what it read is logged (ADR-0063 §8). model and
// effort, when set,
// replace what the role's tier and effort ask: to compare models on the
// same cases, or to run a project on one model.
type claude struct{ model, effort string }

// Until the model grid is generated (docs/spec/model-grid.md), Claude's own
// aliases stand for the tiers.
var claudeTier = map[string]string{"light": "haiku", "standard": "sonnet", "frontier": "opus"}
var claudeEffort = map[string]string{"none": "low", "low": "low", "medium": "medium", "high": "high", "xhigh": "xhigh", "max": "max"}

func (c claude) Propose(req Request) (Call, error) {
	call := Call{Agent: "claude", Tier: req.tier(), Effort: req.Role.Model.Effort}
	if c.effort != "" {
		call.Effort = c.effort
	}
	bin, err := exec.LookPath("claude")
	if err != nil {
		return call, fmt.Errorf("%w: claude is not installed", ErrUnavailable)
	}
	if err := shellTools(Worktree(req.RunDir), runtime.GOOS); err != nil {
		return call, err
	}
	system, user, err := Prompt(req)
	if err != nil {
		return call, err
	}
	// The system prompt is what every call of the role shares (Prompt):
	// Claude Code caches it, the role's next call reads it back (#1128).
	// In a file, not on the command line: tens of kilobytes, past what
	// Windows passes to an npm claude.cmd (8,191 characters).
	systemFile, err := os.CreateTemp("", "workline-system-*.md")
	if err != nil {
		return call, err
	}
	defer func() { _ = os.Remove(systemFile.Name()) }()
	_, err = systemFile.WriteString(system)
	if cerr := systemFile.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return call, err
	}
	args := []string{"-p", "--output-format", "json", "--tools", "", "--strict-mcp-config",
		"--no-session-persistence", "--system-prompt-file", systemFile.Name()}
	tree := Worktree(req.RunDir)
	reads := ""
	if tree == "" {
		reads = ReadTree(req.RunDir)
	}
	if reads != "" {
		// An agent that reads (ADR-0063 §8): Read, Glob and Grep, no
		// shell, nothing it may write; none allowed by name, so that
		// dontAsk keeps them in the tree, where Claude Code reads without
		// asking: a file out of it (the clone's .git/config, a CI token
		// there) is denied, never read. Its prompt the role's alone, as
		// with no tools; each message on stdout, to log what it read. The
		// person's settings only, as in a worktree.
		args = []string{"-p", "--output-format", "stream-json", "--verbose", "--tools", claudeReadTools,
			"--permission-mode", "dontAsk", "--strict-mcp-config",
			"--setting-sources", "user", "--no-session-persistence", "--system-prompt-file", systemFile.Name()}
	}
	if tree != "" {
		// An agent that works (ADR-0056): its tools, each allowed so that
		// none waits on a person (dontAsk denies any other), Claude Code's
		// own prompt kept for them, the role's after it; each message on
		// stdout, to count the tools it used. The person's settings only:
		// the repository's (.claude/, CLAUDE.md) are what it works on, not
		// what runs it: a hook there would run, a permission widen it.
		args = []string{"-p", "--output-format", "stream-json", "--verbose", "--tools", claudeTools,
			"--allowedTools", claudeTools, "--permission-mode", "dontAsk", "--strict-mcp-config",
			"--setting-sources", "user", "--no-session-persistence", "--append-system-prompt-file", systemFile.Name()}
	}
	if call.Asked = claudeTier[call.Tier]; c.model != "" {
		call.Asked = c.model
	}
	if call.Asked != "" {
		args = append(args, "--model", call.Asked)
	}
	if e := claudeEffort[call.Effort]; e != "" {
		args = append(args, "--effort", e)
	}
	if req.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprint(req.MaxTurns))
	}
	limit, err := req.Role.Model.AnswerTimeout()
	if err != nil {
		return call, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir, cmd.WaitDelay = os.TempDir(), waitAfterTimeout
	if tree != "" {
		cmd.Dir = tree
	} else if reads != "" {
		cmd.Dir = reads
	}
	cmd.Env = claudeEnv(os.Environ())
	if tree != "" {
		// The commands it runs never see its login (ANTHROPIC_API_KEY,
		// CLAUDE_CODE_OAUTH_TOKEN…): Claude Code takes them out of its
		// tools' environment, as it does by default in GitHub Actions.
		cmd.Env = append(cmd.Env, "CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1")
	}
	cmd.Stdin = strings.NewReader(user)
	// In a group of its own, worktree or not: the commands its tools
	// start are killed with it, at its time and once it answered.
	inGroup(cmd)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	stubs := absentStubs(tree)
	runErr := settled(ctx, cmd.Run())
	_ = killGroup(cmd)
	removeStubs(tree, stubs)
	stdout := out.Bytes()
	if tree != "" || reads != "" {
		var read []string
		stdout, call.Tools, read = claudeStream(stdout)
		if reads != "" {
			call.Reads = readLog(reads, read)
		}
	}
	answer, parsed := claudeAnswer(stdout, &call)
	// Its turns spent is a cap reached, not a failure: in a worktree, what
	// it did so far is its answer (ADR-0056); out of one, its text is. Only its whole JSON says so: a stdout cut
	// short, or a run killed, is a failure.
	maxed := parsed && answer.Subtype == "error_max_turns" && ctx.Err() == nil && exitedItself(runErr)
	if !maxed && (runErr != nil || (parsed && answer.IsError)) {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return call, fmt.Errorf("%w: claude did not answer within %s (the role's model.timeout)", ErrUnavailable, limit)
		}
		said := out.String()
		if parsed {
			said = answer.Result
		} else if tree != "" || reads != "" {
			// A raw stream may hold the files it read and the output of
			// what it ran: never quoted, its stderr alone (#1245).
			said = ""
		}
		return call, fmt.Errorf("%w: claude failed: %s", ErrUnavailable, lastLine(errOut.String()+said))
	}
	if !parsed && tree != "" {
		// The raw stream may hold file contents and command output: never
		// kept nor posted as the agent's words (#750). Its edits are its code.
		_ = os.WriteFile(filepath.Join(req.RunDir, "out", ClosingUnread),
			[]byte("The agent's closing words could not be read: its output held no result message.\n"), 0o644)
		call.Mended = []string{"the agent's closing words could not be read: its output held no result message, and none of it is kept"}
		return call, os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), []byte("[]\n"), 0o644)
	}
	if !parsed && reads != "" {
		// Not the JSON asked for, from a read tree: what it read may be in
		// it, never kept nor read as its answer (#1245). Its answer
		// unread, as any answer the engine cannot read.
		return call, fmt.Errorf("%w: the agent's words could not be read: its output held no result message, and none of it is kept", ErrInvalidOutput)
	}
	if !parsed {
		// Not the JSON asked for, with no tools: nothing but its own words
		// can be in it, read as the answer itself (a judge's plain verdict).
		answer.Result = out.String()
	}
	// The answer as it came, kept for audit: what the engine reads of it
	// may drop what it holds (claims are judged, never applied).
	_ = os.WriteFile(filepath.Join(req.RunDir, "out", "agent-answer.txt"), []byte(answer.Result), 0o644)
	// In a worktree, its edits are its code: its closing words, cut short
	// by its turns or unreadable, lose nothing of it (closingWords, #746).
	var proposals []byte
	if tree != "" {
		proposals, call.Mended = closingWords(req.RunDir, answer.Result, req.Role.Intentions)
	} else {
		proposals, call.Mended, err = proposalsFrom(answer.Result, req.Role.Intentions...)
		if err != nil && maxed {
			// A role that answers in text and stopped at its turn cap
			// before writing it did no work: an answer unread, asked
			// again with its cap said, never "nothing to propose" (#1194).
			err = fmt.Errorf("it stopped at its turn cap (%d turns, ai-max-turns %d) before it wrote its answer: read fewer files, and answer within your turns", call.Turns, req.MaxTurns)
		}
	}
	if err != nil {
		return call, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return call, os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), proposals, 0o644)
}

// claudeTools are the tools Claude Code gets in a worktree (ADR-0056):
// read, edit, search, and a shell to run the project's tests. Not the web
// tools, nor agents of its own: the network is the shell's, as the
// project's CI allows it.
const claudeTools = "Read,Edit,Write,Glob,Grep,Bash"

// claudeReadTools are the tools Claude Code gets in a read tree (ADR-0063
// §8): read and search, no shell, no edit.
const claudeReadTools = "Read,Glob,Grep"

// claudeShellTools are what Claude Code's shell needs on Linux once its
// commands' environment is scrubbed: it runs each command boxed by
// bubblewrap, its network through socat. Either missing, every command
// fails, and the agent writes code it never ran (#900).
var claudeShellTools = []string{"bwrap", "socat"}

// shellTools says, before the agent is asked, which of claudeShellTools
// a run in a worktree lacks on Linux: the agent unavailable, the role's
// finding naming what to install, rather than an agent with no shell. No
// worktree, no shell: nothing needed.
func shellTools(tree, goos string) error {
	if tree == "" || goos != "linux" {
		return nil
	}
	var missing []string
	for _, t := range claudeShellTools {
		if _, err := exec.LookPath(t); err != nil {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: claude's shell needs bubblewrap and socat, %s not installed: no command would run (apt-get install bubblewrap socat)",
			ErrUnavailable, strings.Join(missing, " and "))
	}
	// Installed is not enough: a container the runner does not let create
	// a namespace (Docker's default profile, not privileged) refuses
	// bubblewrap, and every command with it (#906). Its sandbox kept, not
	// dropped: without it, the commands would hold the agent's login.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "bwrap", "--ro-bind", "/", "/", "true").CombinedOutput()
	if err != nil {
		said := err.Error()
		if strings.TrimSpace(string(out)) != "" {
			said = lastLine(string(out))
		}
		return fmt.Errorf("%w: claude's shell cannot start its sandbox here, no command would run (%s): the runner must let a job's container create a namespace, as gitlab.com's shared runners do (a privileged runner)",
			ErrUnavailable, said)
	}
	return nil
}

// claudeStubs are the files Claude Code creates empty in its folder, where
// absent, when its commands' environment is scrubbed: its sandbox binds
// over them, so that no command writes them. It lists them in
// .git/info/exclude, which a linked worktree's .git, a file, cannot hold:
// left there, they would be the agent's code (#745). Claude Code 2.1.314.
var claudeStubs = []string{
	"bunfig.toml", "package.json", ".npmrc", ".yarnrc", ".yarnrc.yml", ".gitmodules",
	"package-lock.json", "yarn.lock", "pnpm-lock.yaml",
	".env", ".env.local", ".env.development", ".env.development.local",
	".env.test", ".env.test.local", ".env.production", ".env.production.local",
}

// absentStubs are the stubs tree lacks before Claude Code runs there; no
// tree, none.
func absentStubs(tree string) []string {
	var absent []string
	for _, s := range claudeStubs {
		if _, err := os.Lstat(filepath.Join(tree, s)); tree != "" && errors.Is(err, fs.ErrNotExist) {
			absent = append(absent, s)
		}
	}
	return absent
}

// removeStubs removes each of the stubs tree lacked that is now an empty
// file: one its agent wrote holds something, and stays.
func removeStubs(tree string, absent []string) {
	for _, s := range absent {
		p := filepath.Join(tree, s)
		if st, err := os.Lstat(p); err == nil && st.Mode().IsRegular() && st.Size() == 0 {
			_ = os.Remove(p)
		}
	}
}

// claudeStream reads `claude -p --output-format stream-json --verbose`:
// its result message, the shape claudeAnswer reads, how many times it
// used each tool, and the files its Read tool was given, in order, each
// once. No result message: stdout as it came, which claudeAnswer then
// does not read.
func claudeStream(stdout []byte) ([]byte, map[string]int, []string) {
	var result []byte
	var tools map[string]int
	var read []string
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		var m struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type  string `json:"type"`
					Name  string `json:"name"`
					Input struct {
						FilePath string `json:"file_path"`
					} `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &m) != nil {
			continue
		}
		switch m.Type {
		case "result":
			result = line
		case "assistant":
			for _, c := range m.Message.Content {
				if c.Type == "tool_use" && c.Name != "" {
					if tools == nil {
						tools = map[string]int{}
					}
					tools[c.Name]++
					if p := c.Input.FilePath; c.Name == "Read" && p != "" && !slices.Contains(read, p) {
						read = append(read, p)
					}
				}
			}
		}
	}
	if result == nil {
		return stdout, tools, read
	}
	return result, tools, read
}

// readLog is what an agent read in its read tree, each `<blob> <path>`:
// the path from the tree's root, the blob the tree's commit holds there
// (git ls-tree), "-" for none (a folder, a file not there); a path out of
// the tree as the agent gave it, after "-".
func readLog(tree string, read []string) []string {
	if len(read) == 0 {
		return nil
	}
	root, err := filepath.EvalSymlinks(tree)
	if err != nil {
		root = tree
	}
	rels := make([]string, len(read))
	args := []string{"-C", tree, "ls-tree", "-z", "HEAD", "--"}
	for i, p := range read {
		abs := filepath.Clean(p)
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(root, abs)
		} else if dir, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
			abs = filepath.Join(dir, filepath.Base(abs))
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		rels[i] = filepath.ToSlash(rel)
		args = append(args, ":(literal)"+rels[i])
	}
	blobs := map[string]string{}
	if out, err := exec.Command("git", args...).Output(); err == nil {
		for _, l := range strings.Split(string(out), "\x00") { // -z: paths unquoted
			meta, path, ok := strings.Cut(l, "\t")
			if f := strings.Fields(meta); ok && len(f) == 3 && f[1] == "blob" {
				blobs[path] = f[2]
			}
		}
	}
	log := make([]string, 0, len(read))
	for i, r := range rels {
		switch {
		case r == "":
			log = append(log, "- "+read[i])
		case blobs[r] != "":
			log = append(log, blobs[r]+" "+r)
		default:
			log = append(log, "- "+r)
		}
	}
	return log
}

// exitedItself says the agent's process ended by its own exit, whatever
// its code: not killed by a signal, nor failed to start.
func exitedItself(err error) bool {
	if err == nil {
		return true
	}
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.Exited()
}

// claudeSession are the variables a Claude Code session exports to every
// process it starts. Passed on, they made a role's call that session's
// child, at twice the tokens in (docs/research/claude-session-env.md).
// They are dropped, even one a person set: a call with no tools needs
// none, and its effort is the role's. The login (CLAUDE_CODE_OAUTH_TOKEN,
// ANTHROPIC_API_KEY, ANTHROPIC_BASE_URL…) and other settings pass.
var claudeSession = []string{
	"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_CHILD_SESSION",
	"CLAUDE_CODE_SESSION_ATTENDED", "CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_MESSAGING_TOKEN",
	"CLAUDE_CODE_EXECPATH", "CLAUDE_PID", "CLAUDE_EFFORT", "CLAUDE_AGENT_SDK_VERSION",
	"CLAUDE_CODE_ENABLE_TASKS", "CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING",
	"CLAUDE_CODE_EMIT_STARTUP_TIMING", "CLAUDE_CODE_QUESTION_PREVIEW_FORMAT",
	"MCP_CONNECTION_NONBLOCKING", "AI_AGENT",
}

// claudeEnv is environ without the session's variables.
func claudeEnv(environ []string) []string {
	return without(withoutTokens(environ), claudeSession)
}

// ForgeTokens are the variables holding what writes to the forge or signs
// for workline (forge.Tokens). No agent is handed one (principle 6): it
// proposes, the engine acts. The agent's own login passes.
var ForgeTokens = forge.Tokens

// withoutTokens is the environment an agent is called with: no forge
// token, nor the line's own settings for its forge calls.
func withoutTokens(environ []string) []string {
	return forge.ProjectEnv(without(environ, ForgeTokens))
}

func without(environ, names []string) []string {
	env := make([]string, 0, len(environ))
	for _, kv := range environ {
		if name, _, _ := strings.Cut(kv, "="); !slices.Contains(names, name) {
			env = append(env, kv)
		}
	}
	return env
}

// claudeResult is the part of `claude -p --output-format json` workline reads.
type claudeResult struct {
	Result     string  `json:"result"`
	IsError    bool    `json:"is_error"`
	Subtype    string  `json:"subtype"`   // error_max_turns: its --max-turns spent
	NumTurns   int     `json:"num_turns"` // the turns it took
	CostUSD    float64 `json:"total_cost_usd"`
	ModelUsage map[string]struct {
		InputTokens              int `json:"inputTokens"`
		OutputTokens             int `json:"outputTokens"`
		CacheReadInputTokens     int `json:"cacheReadInputTokens"`
		CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	} `json:"modelUsage"`
}

// claudeAnswer reads Claude Code's JSON answer, and records in call the model
// that wrote it and what the call used. Claude Code may call a small model on
// the side; the model that wrote the most is the one answering, the others
// count in the cost only. false: stdout was not that JSON.
func claudeAnswer(stdout []byte, call *Call) (claudeResult, bool) {
	var r claudeResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout), &r); err != nil {
		return r, false
	}
	// The model asked answers, when one of those used is it: a short answer
	// ("[]") can be outwritten by the small model on the side, which then
	// passed for the one answering (#126, tried.md).
	asked := func(m string) bool { return call.Asked != "" && strings.Contains(m, call.Asked) }
	most, mostAsked := -1, false
	for m, u := range r.ModelUsage {
		better := u.OutputTokens > most || (u.OutputTokens == most && m < call.Model)
		if a := asked(m); a != mostAsked {
			better = a
		}
		if better {
			call.Model, most, mostAsked = m, u.OutputTokens, asked(m)
		}
		call.TokensIn += u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
		call.TokensCached += u.CacheReadInputTokens
		call.TokensCacheW += u.CacheCreationInputTokens
		call.TokensOut += u.OutputTokens
	}
	call.CostUSD, call.Turns = r.CostUSD, r.NumTurns
	return r, true
}

// proposalsFrom extracts the YAML list from an answer, tolerating a code fence,
// and says what Mend mended of it; kinds, those the role writes, tell an
// item holding several keys where they go. The answer itself is tried first: a
// proposal may hold a code fence of its own (a doc's code block, moved by a
// patch). Only when it does not read is it taken from between the first and
// the last fence lines.
func proposalsFrom(answer string, kinds ...string) ([]byte, []string, error) {
	s := strings.TrimSpace(answer)
	if s == "[]" { // nothing to propose, said: a part whose share says nothing of the doc
		return []byte("[]\n"), nil, nil
	}
	s, mended := Mend(s)
	var list []map[string]any
	if err := yaml.Unmarshal([]byte(s), &list); err != nil || len(list) == 0 {
		// What the reader says is kept: the agent is asked again with it.
		why := "an empty list"
		if err != nil {
			why = err.Error()
		}
		lines := strings.Split(s, "\n")
		first, last := -1, -1
		for i, l := range lines {
			if strings.HasPrefix(l, "```") {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		read := false
		if first >= 0 && last > first {
			s, mended = Mend(strings.TrimSpace(strings.Join(lines[first+1:last], "\n")))
			list = nil
			if err := yaml.Unmarshal([]byte(s), &list); err != nil || len(list) == 0 {
				if err != nil {
					why = err.Error()
				}
			} else {
				read = true
			}
		}
		// Prose before the list, a line saying what comes (#1148): the
		// list read from its first item, the prose left out and said.
		if !read && first < 0 {
			if prose, rest, ok := proseFirst(lines); ok {
				s, mended = Mend(rest)
				list = nil
				if err := yaml.Unmarshal([]byte(s), &list); err == nil && len(list) > 0 {
					read = true
					mended = append([]string{fmt.Sprintf("prose before the list (%s): left out, the list read from its first item", Clip(prose, 120))}, mended...)
				}
			}
		}
		if !read {
			return nil, nil, fmt.Errorf("expected a YAML list of proposals (%s)", why)
		}
	}
	dropped := false
	for i, m := range list {
		if len(m) > 1 {
			said, err := besidePatch(m)
			if err != nil {
				return nil, nil, fmt.Errorf("proposal %d: %w", i+1, err)
			}
			if said != "" {
				dropped = true
				mended = append(mended, fmt.Sprintf("proposal %d held %s", i+1, said))
			}
		}
		switch len(m) {
		case 1:
		case 0:
			return nil, nil, fmt.Errorf("proposal %d holds no kind; each holds exactly one", i+1)
		default:
			return nil, nil, fmt.Errorf("proposal %d holds %d kinds (%s); each holds exactly one%s", i+1, len(m), strings.Join(slices.Sorted(maps.Keys(m)), ", "), whereKeysGo(m, kinds))
		}
	}
	if dropped {
		out, err := yaml.Marshal(list)
		if err != nil {
			return nil, nil, err
		}
		return out, mended, nil
	}
	return []byte(s + "\n"), mended, nil
}

// proseFirst cuts an answer at its first line opening a list item, `- `
// at the start of the line: the prose before it, and the rest. false
// when no prose comes first, or no item follows.
func proseFirst(lines []string) (string, string, bool) {
	for i, l := range lines {
		if strings.HasPrefix(l, "- ") || l == "-" {
			prose := strings.Join(strings.Fields(strings.Join(lines[:i], " ")), " ")
			return prose, strings.Join(lines[i:], "\n"), prose != ""
		}
	}
	return "", "", false
}

// Clip is a text cut to n runes, "…" said where it was cut.
func Clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// besidePatch drops from an item holding a diff under `patch` the keys an
// agent wrote beside it that the diff already says (#1148): a `file` or
// `path` naming the file it changes, a `content` that is the text the diff
// gives a new file. It says what it dropped; "" when none, the item then
// left to be refused as it is. A key that says otherwise than the diff is
// refused, named.
func besidePatch(m map[string]any) (string, error) {
	diff, ok := m["patch"].(string)
	if !ok {
		return "", nil
	}
	files := intent.PatchFiles(diff)
	var said []string
	for _, key := range []string{"file", "path"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		f, _ := v.(string)
		if f == "" || !slices.Contains(files, path.Clean(strings.TrimPrefix(f, "b/"))) {
			return "", fmt.Errorf("its `%s:` (%v) beside its patch is not a file the patch changes (%s): write the patch alone, its file named in its `+++` line", key, v, strings.Join(files, ", "))
		}
		delete(m, key)
		said = append(said, fmt.Sprintf("a `%s:` beside its patch, the patch's own file: left out", key))
	}
	if v, ok := m["content"]; ok {
		c, _ := v.(string)
		if len(files) != 1 || !strings.Contains(diff, "--- /dev/null") || strings.TrimRight(c, "\n") != strings.TrimRight(newFileText(diff), "\n") {
			return "", fmt.Errorf("its `content:` beside its patch is not the text the patch gives a new file: send either the patch or the file's content, not both")
		}
		delete(m, "content")
		said = append(said, "a `content:` beside its patch, the text the patch gives its new file: left out")
	}
	return strings.Join(said, "; "), nil
}

// newFileText is the text a diff gives the file it creates: its added
// lines.
func newFileText(diff string) string {
	var b strings.Builder
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			b.WriteString(l[1:] + "\n")
		}
	}
	return b.String()
}

// whereKeysGo says, of an item holding several keys where one kind goes,
// where they go. Several kinds: an item each. A key no kind is named by is
// a key of a proposal, written under its kind, indented: the kind whose
// shape holds them all is named, among those the role writes (kinds; none
// given, any). A reviewer's lens wrote a finding's `decision` and `cause`
// as an item of their own (#490, #492) and, asked again only for valid
// YAML, wrote them so again.
func whereKeysGo(m map[string]any, kinds []string) string {
	var keys []string
	for k := range m {
		if _, kind := contracts[k]; !kind {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return ": write each as an item of its own"
	}
	slices.Sort(keys)
	are, them := "is no kind, but a key", "it"
	if len(keys) > 1 {
		are, them = "are no kind, but keys", "them"
	}
	said := strings.Join(keys[:len(keys)-1], ", ")
	if said != "" {
		said += " and "
	}
	said += keys[len(keys)-1]
	if len(kinds) == 0 {
		kinds = slices.Collect(maps.Keys(contracts))
	}
	var holders []string
	for _, kind := range kinds {
		shape, ok := contracts[kind]
		if !ok || engineOnly[kind] {
			continue
		}
		top := shapeKeys(shape)
		if !slices.ContainsFunc(keys, func(k string) bool { return !top[k] }) {
			holders = append(holders, kind)
		}
	}
	if len(holders) == 1 {
		return fmt.Sprintf(": %s %s of a `%s`, written under `- %s:`, indented, beside its other keys", said, are, holders[0], holders[0])
	}
	return fmt.Sprintf(": %s %s of a proposal, written under its kind, indented, as the answer's shape shows %s", said, are, them)
}

// shapeKeys are the keys a kind's shape names at its top: those of its
// map, and those its comment adds (a finding's `decision`); not those of a
// map inside it (a cause's `path`, which goes under the cause), nor words
// in a quoted text.
func shapeKeys(shape string) map[string]bool {
	keys := map[string]bool{}
	depth, quoted, from := 0, false, -1
	for i := 0; i < len(shape); i++ {
		c := shape[i]
		switch {
		case quoted:
			quoted = c != '"'
			continue
		case c == '"':
			quoted = true
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
		case c == ':' && from >= 0 && depth <= 1:
			keys[shape[from:i]] = true
		}
		word := c == '-' || c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
		switch {
		case !word:
			from = -1
		case from < 0:
			from = i
		}
	}
	return keys
}

// lastLine is a command's last line of output, as a finding quotes it:
// its secrets masked (forge.Mask, #1215).
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(forge.Mask(s)), "\n")
	return lines[len(lines)-1]
}
