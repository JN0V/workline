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
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// claude runs Claude Code headless. It gets no tools, no MCP servers and no
// session, not even the Claude Code session it may be started from
// (claudeEnv), and runs outside the repository so the project's CLAUDE.md is
// not loaded: the role's facets are its whole context. Where the role gave
// it a worktree (WorktreeDir), it runs there instead, with the tools to
// read, edit, search and run commands (claudeTools), none of them asking a
// person, nor the repository's settings nor CLAUDE.md loaded: its edits
// are its code (ADR-0056). model and effort, when set,
// replace what the role's tier and effort ask: to compare models on the
// same cases, or to run a project on one model.
type claude struct{ model, effort string }

// Until the model grid is generated (docs/spec/model-grid.md), Claude's own
// aliases stand for the tiers.
var claudeTier = map[string]string{"light": "haiku", "standard": "sonnet", "frontier": "opus"}
var claudeEffort = map[string]string{"none": "low", "low": "low", "medium": "medium", "high": "high", "max": "max"}

func (c claude) Propose(req Request) (Call, error) {
	call := Call{Agent: "claude", Tier: req.tier(), Effort: req.Role.Model.Effort}
	if c.effort != "" {
		call.Effort = c.effort
	}
	bin, err := exec.LookPath("claude")
	if err != nil {
		return call, fmt.Errorf("%w: claude is not installed", ErrUnavailable)
	}
	system, user, err := Prompt(req)
	if err != nil {
		return call, err
	}
	args := []string{"-p", "--output-format", "json", "--tools", "", "--strict-mcp-config",
		"--no-session-persistence", "--system-prompt", system}
	tree := Worktree(req.RunDir)
	if tree != "" {
		// An agent that works (ADR-0056): its tools, each allowed so that
		// none waits on a person (dontAsk denies any other), Claude Code's
		// own prompt kept for them, the role's after it; each message on
		// stdout, to count the tools it used. The person's settings only:
		// the repository's (.claude/, CLAUDE.md) are what it works on, not
		// what runs it: a hook there would run, a permission widen it.
		args = []string{"-p", "--output-format", "stream-json", "--verbose", "--tools", claudeTools,
			"--allowedTools", claudeTools, "--permission-mode", "dontAsk", "--strict-mcp-config",
			"--setting-sources", "user", "--no-session-persistence", "--append-system-prompt", system}
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
	if tree != "" {
		stdout, call.Tools = claudeStream(stdout)
	}
	answer, parsed := claudeAnswer(stdout, &call)
	// Its turns spent is a cap reached, not a failure: what it did so far
	// is its answer (ADR-0056). Only its whole JSON says so: a stdout cut
	// short, or a run killed, is a failure.
	maxed := parsed && answer.Subtype == "error_max_turns" && ctx.Err() == nil && exitedItself(runErr)
	if !maxed && (runErr != nil || (parsed && answer.IsError)) {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return call, fmt.Errorf("%w: claude did not answer within %s (the role's model.timeout)", ErrUnavailable, limit)
		}
		said := out.String()
		if parsed {
			said = answer.Result
		}
		return call, fmt.Errorf("%w: claude failed: %s", ErrUnavailable, lastLine(errOut.String()+said))
	}
	if !parsed {
		answer.Result = out.String() // not the JSON asked for: read it as the answer itself
	}
	// The answer as it came, kept for audit: what the engine reads of it
	// may drop what it holds (claims are judged, never applied).
	_ = os.WriteFile(filepath.Join(req.RunDir, "out", "agent-answer.txt"), []byte(answer.Result), 0o644)
	proposals, mended, err := proposalsFrom(answer.Result, req.Role.Intentions...)
	call.Mended = mended
	switch {
	case err != nil && maxed:
		// Stopped mid-way, its answer is not written yet: what it edited
		// in its worktree is what it did.
		proposals, err = []byte("[]\n"), nil
	case err != nil && tree != "" && !strings.Contains(answer.Result, "objection"):
		// Its edits are its code: closing words that are no proposals
		// lose nothing of it, said, never asked again (a whole run of
		// turns again); its words stay in out/agent-answer.txt. Words
		// that name an objection are its agent failing: a Draft, never
		// an objection dropped in silence.
		call.Mended = append(call.Mended, "its closing words were no YAML list ("+err.Error()+
			"): read as proposing nothing beyond its edits (out/agent-answer.txt)")
		proposals, err = []byte("[]\n"), nil
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
// its result message, the shape claudeAnswer reads, and how many times it
// used each tool. No result message: stdout as it came, which claudeAnswer
// then does not read.
func claudeStream(stdout []byte) ([]byte, map[string]int) {
	var result []byte
	var tools map[string]int
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		var m struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type string `json:"type"`
					Name string `json:"name"`
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
				}
			}
		}
	}
	if result == nil {
		return stdout, tools
	}
	return result, tools
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
// for workline: the forge's tokens, the App's key, the signing key, and
// what a CI job holds that mints or carries one (GitHub's OIDC request
// token, GitLab's job token in its registry password and clone URL). No
// agent is handed one (principle 6): it proposes, the engine acts. The
// agent's own login passes.
var ForgeTokens = []string{
	"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN",
	"GITLAB_TOKEN", "GLAB_TOKEN", "WORKLINE_GITLAB_TOKEN", "WORKLINE_GITLAB_READ_TOKEN", "CI_JOB_TOKEN",
	"FORGEJO_TOKEN", "WORKLINE_APP_PRIVATE_KEY", "WORKLINE_SIGNING_KEY",
	"ACTIONS_ID_TOKEN_REQUEST_TOKEN", "ACTIONS_RUNTIME_TOKEN",
	"CI_REGISTRY_PASSWORD", "CI_DEPLOY_PASSWORD", "CI_JOB_JWT", "CI_JOB_JWT_V1", "CI_JOB_JWT_V2", "CI_REPOSITORY_URL",
	"CI_BUILD_TOKEN", "CI_DEPENDENCY_PROXY_PASSWORD",
}

// withoutTokens is the environment an agent is called with: no forge token.
func withoutTokens(environ []string) []string { return without(environ, ForgeTokens) }

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
		if first < 0 || last == first {
			return nil, nil, fmt.Errorf("expected a YAML list of proposals (%s)", why)
		}
		s, mended = Mend(strings.TrimSpace(strings.Join(lines[first+1:last], "\n")))
		list = nil
		if err := yaml.Unmarshal([]byte(s), &list); err != nil || len(list) == 0 {
			if err != nil {
				why = err.Error()
			}
			return nil, nil, fmt.Errorf("expected a YAML list of proposals (%s)", why)
		}
	}
	for i, m := range list {
		switch len(m) {
		case 1:
		case 0:
			return nil, nil, fmt.Errorf("proposal %d holds no kind; each holds exactly one", i+1)
		default:
			return nil, nil, fmt.Errorf("proposal %d holds %d kinds (%s); each holds exactly one%s", i+1, len(m), strings.Join(slices.Sorted(maps.Keys(m)), ", "), whereKeysGo(m, kinds))
		}
	}
	return []byte(s + "\n"), mended, nil
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

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
