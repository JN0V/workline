package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// run calls a forge CLI (gh) in the repository, which tells it which
// project to talk to. A CLI that is missing or cannot authenticate makes the
// forge unreachable, not the role wrong.
func run(repo, bin string, args ...string) ([]byte, error) {
	return runInput(repo, nil, bin, args...)
}

// runInput is run, given what the CLI reads on its standard input: a
// request's JSON body (`gh api --input -`). A write the forge refused for
// who the token is (HTTP 403) is errForbidden; one it answered and did not
// take (HTTP 422), errRefused, with what it said.
//
// A call that fails for a moment is tried again, by GitLab's rule
// (retry.go): gh has no option to, nor a timeout. How it failed is read
// from what gh prints, the answer's headers asked (--include): a status, a
// Retry-After; gh is stopped when it prints no answer for a minute.
func runInput(repo string, input []byte, bin string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("%w: %s is not installed", ErrUnreachable, bin)
	}
	read := ghRead(args)
	if asksHeads(args) {
		args = append(slices.Clone(args), "--include")
	}
	for try := 1; ; try++ {
		c := ghOnce(repo, input, bin, args)
		if !c.failed {
			return c.body, nil
		}
		wait, note := again("GitHub", read, try, c.failure)
		if wait < 0 || note != "" {
			return nil, c.err(bin, args[0], note)
		}
		time.Sleep(wait)
	}
}

// asksHeads says whether gh is asked for the answers' headers: not when
// it is already, nor for REST pages printed whole, whose arrays gh joins
// into one only without --include; their status is read from its words.
func asksHeads(args []string) bool {
	if slices.Contains(args, "--include") || slices.Contains(args, "-i") {
		return false
	}
	return !slices.Contains(args, "--paginate") || slices.Contains(args, "graphql") ||
		slices.Contains(args, "--jq") || slices.Contains(args, "-q")
}

// ghCall is one try of a gh call: what it printed, the answers' headers
// taken out, and how it failed.
type ghCall struct {
	body    []byte // its standard output, without the headers
	msg     string // what it said failing
	failed  bool
	failure failure
}

// ghOnce runs gh once, stopped when it prints nothing for retry.answerMax:
// gh prints each answer, a page's, as it comes.
func ghOnce(repo string, input []byte, bin string, args []string) ghCall {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = repo
	cmd.WaitDelay = time.Second
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var stalled atomic.Bool
	timer := time.AfterFunc(retry.answerMax, func() { stalled.Store(true); cancel() })
	defer timer.Stop()
	out := &awake{timer: timer}
	var errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = out, &errOut
	err := cmd.Run()
	code, header, body := heads(out.Bytes())
	c := ghCall{body: body, msg: strings.TrimSpace(errOut.String()), failed: err != nil}
	if !c.failed {
		return c
	}
	if stalled.Load() {
		c.msg = strings.TrimSpace(c.msg + " no answer within " + strconv.FormatFloat(retry.answerMax.Seconds(), 'f', -1, 64) + " s")
	}
	c.failure = ghFailure(stalled.Load(), c.msg, code, header)
	return c
}

// awake is gh's output as it comes: each write puts off the moment gh is
// stopped for giving no answer.
type awake struct {
	bytes.Buffer
	timer *time.Timer
}

func (w *awake) Write(p []byte) (int, error) {
	w.timer.Reset(retry.answerMax)
	return w.Buffer.Write(p)
}

// headBlock is what gh prints with --include before each answer: the
// status line, each header ended by CRLF, an empty line.
var headBlock = regexp.MustCompile(`(?m)^HTTP/[0-9.]+ ([0-9]{3})[^\r\n]*\r?\n((?:[^\r\n:]+:[^\r\n]*\r\n)*)\r\n`)

// heads takes the headers gh printed out of its output: the last answer's
// status and headers, and what is left, the answers' bodies.
func heads(out []byte) (code int, header http.Header, body []byte) {
	blocks := headBlock.FindAllSubmatchIndex(out, -1)
	if len(blocks) == 0 {
		return 0, nil, out
	}
	last := blocks[len(blocks)-1]
	code, _ = strconv.Atoi(string(out[last[2]:last[3]]))
	header = http.Header{}
	for _, line := range strings.Split(string(out[last[4]:last[5]]), "\r\n") {
		if k, v, ok := strings.Cut(line, ":"); ok {
			header.Add(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
	from := 0
	for _, b := range blocks {
		body = append(body, out[from:b[0]]...)
		from = b[1]
	}
	return code, header, append(body, out[from:]...)
}

// transport is how gh says a request got no answer, Go's own words:
// `Get "https://…": EOF`.
var transport = regexp.MustCompile(`(?m)^[A-Z][a-z]+ "https?://[^"]*": (.+)$`)

var httpCode = regexp.MustCompile(`\bHTTP ([0-9]{3})\b`)

// ghFailure reads how a gh call failed. A host gh could not resolve, which
// it does not tell from one that does not exist ("error connecting to"),
// was sent nothing: tried again. A 403 is GitHub's rate when it says so,
// in its headers or its words; without a wait asked, GitHub asks a minute.
func ghFailure(stalled bool, msg string, code int, header http.Header) failure {
	if stalled {
		return failure{cut: true}
	}
	if m := transport.FindStringSubmatch(msg); m != nil {
		why := m[1]
		return failure{cut: true,
			lasting: strings.Contains(why, "x509:") || strings.Contains(why, "tls: failed to verify certificate"),
			unsent:  strings.Contains(why, "dial tcp") || strings.Contains(why, "TLS handshake timeout") || strings.Contains(why, "proxyconnect")}
	}
	if strings.HasPrefix(msg, "error connecting to ") {
		return failure{cut: true, unsent: true}
	}
	if code == 0 {
		if m := httpCode.FindStringSubmatch(msg); m != nil {
			code, _ = strconv.Atoi(m[1])
		}
	}
	if header == nil {
		header = http.Header{}
	}
	f := failure{code: code, header: header}
	f.limited = code == http.StatusTooManyRequests || code == http.StatusForbidden &&
		(header.Get("Retry-After") != "" || header.Get("X-RateLimit-Remaining") == "0" || strings.Contains(strings.ToLower(msg), "rate limit"))
	if f.limited {
		f.least, f.rate = time.Minute, "X-RateLimit-"
	}
	return f
}

// err is the call's failure, once not tried again.
func (c ghCall) err(bin, sub, note string) error {
	msg := c.msg
	switch {
	case c.failure.limited:
	case strings.Contains(msg, "404") || strings.Contains(strings.ToLower(msg), "not found"):
		return fmt.Errorf("%w: %s", errNotFound, msg)
	case strings.Contains(msg, "(HTTP 403)"):
		return fmt.Errorf("%w: %s %s: %s", errForbidden, bin, sub, msg)
	case strings.Contains(msg, "(HTTP 422)"):
		if said := strings.TrimSpace(string(c.body)); said != "" {
			msg += ": " + said
		}
		return fmt.Errorf("%w: %s %s: %s", errRefused, bin, sub, msg)
	}
	return fmt.Errorf("%w: %s %s: %s%s", ErrUnreachable, bin, sub, msg, aside(note))
}

// ghRead says whether a `gh api` call only reads, so may be sent again
// whatever happened to it: a GET, said or not, with no field nor input
// (gh sends a POST then); a GraphQL `query`, its shorthand `{…}` too. A
// mutation, or anything else, writes.
func ghRead(args []string) bool {
	method, query, fields, endpoint := "", "", false, ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() string {
			if i+1 < len(args) {
				i++
				return args[i]
			}
			return ""
		}
		switch {
		case a == "api":
		case a == "-X" || a == "--method":
			method = strings.ToUpper(next())
		case strings.HasPrefix(a, "--method="):
			method = strings.ToUpper(strings.TrimPrefix(a, "--method="))
		case strings.HasPrefix(a, "-X"):
			method = strings.ToUpper(strings.TrimPrefix(a, "-X"))
		case a == "-f" || a == "-F" || a == "--field" || a == "--raw-field" || a == "--input":
			fields = true
			if v, ok := strings.CutPrefix(next(), "query="); ok {
				query = v
			}
		case a == "--jq" || a == "-q" || a == "-H" || a == "--header" || a == "-t" || a == "--template" ||
			a == "-p" || a == "--preview" || a == "--hostname" || a == "--cache":
			next()
		case strings.HasPrefix(a, "-"):
		case endpoint == "":
			endpoint = a
		}
	}
	if endpoint == "graphql" {
		q := strings.TrimSpace(query)
		return (method == "" || method == "POST") && (strings.HasPrefix(q, "query") || strings.HasPrefix(q, "{"))
	}
	return method == "GET" || method == "" && !fields
}

var errNotFound = fmt.Errorf("not found")

// errForbidden marks a write the forge refused for who the token is: a
// note another user wrote, a role too low.
var errForbidden = fmt.Errorf("%w: forbidden", ErrUnreachable)

func decode(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
	}
	return nil
}
