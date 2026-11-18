package forge

// Conditional reads (#1762): a listing read again sends its last answer's
// ETag as If-None-Match; GitHub answers 304 Not Modified, at no cost to
// the token's hourly quota, when nothing changed, and that last answer is
// used. The answers are kept in the clone's git folder, from one command
// to the next: what a loop ticking every minute reads again.

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// conditional: the GitHub listings that can are read conditionally. Off
// by default: only a command run again and again, the tick, gains by it.
var conditional bool

// ReadConditionally turns conditional reads on (#1762): `workline tick`.
func ReadConditionally(on bool) { conditional = on }

// calls counts the GitHub calls this process sent: each page an answer,
// and those answered 304 Not Modified, which cost no quota.
var calls struct {
	sync.Mutex
	sent, free int
}

// Calls says how many GitHub calls this process sent, and how many of
// them cost no quota (a 304 Not Modified).
func Calls() (sent, free int) {
	calls.Lock()
	defer calls.Unlock()
	return calls.sent, calls.free
}

func counted(pages int, notModified bool) {
	calls.Lock()
	defer calls.Unlock()
	calls.sent += max(pages, 1)
	if notModified {
		calls.free++
	}
}

// keptAnswer is a listing's last answer kept: its ETag, what --jq made of it,
// whether a next page followed, and when it was last used.
type keptAnswer struct {
	ETag string    `json:"etag"`
	Body string    `json:"body"`
	Next bool      `json:"next,omitempty"`
	Used time.Time `json:"used"`
}

// answerKept: an answer unused this long is dropped.
const answerKept = 24 * time.Hour

var answers struct {
	sync.Mutex
	repo, path string
	m          map[string]keptAnswer
}

// keptAnswers is repo's answers kept, read once a process for each
// repository it reads: none when repo is no clone.
func keptAnswers(repo string) map[string]keptAnswer {
	if answers.m != nil && answers.repo == repo {
		return answers.m
	}
	answers.repo, answers.path, answers.m = repo, "", map[string]keptAnswer{}
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return answers.m
	}
	answers.path = filepath.Join(strings.TrimSpace(string(out)), "workline", "etags.json")
	if data, err := os.ReadFile(answers.path); err == nil {
		_ = json.Unmarshal(data, &answers.m)
	}
	return answers.m
}

// keep writes the answers, those unused for answerKept dropped; a write
// that fails loses nothing but the next read's saving.
func keep() {
	if answers.path == "" {
		return
	}
	now := time.Now()
	for k, a := range answers.m {
		if now.Sub(a.Used) > answerKept {
			delete(answers.m, k)
		}
	}
	data, err := json.Marshal(answers.m)
	if err != nil || os.MkdirAll(filepath.Dir(answers.path), 0o755) != nil {
		return
	}
	// A file of its own, renamed in place: two ticks at once each write
	// a whole file, the last one kept.
	f, err := os.CreateTemp(filepath.Dir(answers.path), "etags-*.json")
	if err != nil {
		return
	}
	_, werr := f.Write(data)
	if cerr := f.Close(); werr != nil || cerr != nil || os.Rename(f.Name(), answers.path) != nil {
		_ = os.Remove(f.Name())
	}
}

// listing reads one page of a GitHub listing, --jq applied: conditionally
// when on, its last answer used on a 304. next: a page follows (its Link).
func (g *github) listing(endpoint, jq string) (out []byte, next bool, err error) {
	if !conditional {
		body, header, err := runHeader(g.repo, "gh", "api", endpoint, "--jq", jq)
		return body, hasNext(header), err
	}
	answers.Lock()
	defer answers.Unlock()
	kept := keptAnswers(g.repo)
	key := endpoint + "\n" + jq
	args := []string{"api", endpoint, "--jq", jq}
	last, ok := kept[key]
	if ok && last.ETag != "" {
		args = append(args, "-H", "If-None-Match: "+last.ETag)
	}
	body, header, err := runHeader(g.repo, "gh", args...)
	if err != nil {
		return nil, false, err
	}
	if header.Get(statusHeader) == "304" && ok {
		last.Used = time.Now()
		kept[key] = last
		keep()
		return []byte(last.Body), last.Next, nil
	}
	a := keptAnswer{ETag: header.Get("Etag"), Body: string(body), Next: hasNext(header), Used: time.Now()}
	if a.ETag != "" {
		kept[key] = a
		keep()
	}
	return body, a.Next, nil
}

// statusHeader holds, in the headers runHeader returns, the answer's
// HTTP status.
const statusHeader = "X-Workline-Status"

func hasNext(h http.Header) bool { return strings.Contains(h.Get("Link"), `rel="next"`) }
