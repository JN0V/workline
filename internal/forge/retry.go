package forge

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A forge call that fails for a moment is tried again: a read on no
// answer, a 5xx but 501 or a refusal for the forge's rate; a write only
// when the forge certainly did not take it (no connection made, or a
// refusal for the rate). A write answered late or with a 5xx is not sent
// twice: its failure says so. GitLab's calls (send) and GitHub's, through
// gh (cli.go), go by this one rule (again). Why, and what is borrowed:
// docs/gitlab/calls.md, docs/github/calls.md.

// retry is how a call is tried again: tries in all, the pause after the
// first (doubled after each), the longest wait asked that is waited for,
// how long an answer is waited for.
var retry = struct {
	tries     int
	pause     time.Duration
	waitMax   time.Duration
	answerMax time.Duration
}{tries: 3, pause: 2 * time.Second, waitMax: 60 * time.Second, answerMax: 60 * time.Second}

// httpClient gives each request a minute: gitlab.com answers in under one
// second (#489).
var httpClient = &http.Client{Timeout: retry.answerMax}

// answer is a call's answer, read whole.
type answer struct {
	status string
	code   int
	header http.Header
	data   []byte
	note   string // when it failed for a moment: why it was not tried again, or how often it was
}

// failure is how one try of a call failed, whatever the forge, as again
// reads it.
type failure struct {
	code    int           // the status answered; 0 when none came
	header  http.Header   // the answer's headers, when read
	cut     bool          // no answer came: no connection, one dropped, none in time
	lasting bool          // cut for good: a host that does not exist, a certificate refused
	unsent  bool          // it certainly never reached the forge: no connection was made
	limited bool          // refused for the forge's rate: a 429, GitHub's 403 saying so
	least   time.Duration // the wait the forge asks for its rate when no header says (GitHub: a minute)
	rate    string        // its rate headers' prefix, when it says when a rate spent is reset (GitHub: X-RateLimit-)
}

// send makes one call to GitLab, tried again by the rules above; read is
// whether it only reads, so may be sent again whatever happened to it. A
// call never answered is ErrUnreachable, the note said.
func (g *gitlab) send(method, target, contentType string, body []byte, read bool) (*answer, error) {
	for try := 1; ; try++ {
		var r io.Reader
		if body != nil {
			r = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, target, r)
		if err != nil {
			return nil, err
		}
		req.Header.Set(g.header, g.token)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		var a *answer
		resp, err := httpClient.Do(req)
		if err == nil {
			a = &answer{status: resp.Status, code: resp.StatusCode, header: resp.Header}
			a.data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
		}
		f := failure{cut: err != nil}
		if err != nil {
			f.lasting, f.unsent = lasting(err), unsent(err)
		} else {
			f.code, f.header, f.limited = a.code, a.header, a.code == http.StatusTooManyRequests
		}
		wait, note := again("GitLab", read, try, f)
		if wait < 0 || note != "" {
			if err != nil {
				return nil, fmt.Errorf("%w: %v%s", ErrUnreachable, err, aside(note))
			}
			a.note = note
			return a, nil
		}
		time.Sleep(wait)
	}
}

// again says how long to wait before a try of a call to forge is sent
// again, -1 when it is not because it did not fail for a moment; note,
// when it failed for a moment, is why it is not sent again, or how often
// it was.
func again(forge string, read bool, try int, f failure) (wait time.Duration, note string) {
	taken := "a write: not sent again, " + forge + " may have taken it"
	switch {
	case f.cut && f.lasting:
		return -1, ""
	case f.cut && !read && !f.unsent:
		return -1, taken
	case f.cut, f.limited:
	case f.code >= 500 && f.code != http.StatusNotImplemented && !read:
		return -1, taken
	case f.code >= 500 && f.code != http.StatusNotImplemented:
	default:
		return -1, ""
	}
	wait = retry.pause << (try - 1)
	if f.limited {
		wait = max(wait, f.least)
	}
	if after, said, ok := asked(f); ok {
		if after > retry.waitMax {
			return -1, fmt.Sprintf("%s asks to wait (%s), longer than workline waits (%d s): not tried again",
				forge, said, int(retry.waitMax.Seconds()))
		}
		wait = after
	}
	if try >= retry.tries {
		return -1, fmt.Sprintf("tried %d times", try)
	}
	return wait, ""
}

// asked is the wait the forge asks before a call is sent again, and the
// header that says it: a Retry-After; for a call refused for the rate,
// none left, the time it is reset at, on a forge that says it (GitHub's
// X-RateLimit-Reset).
func asked(f failure) (time.Duration, string, bool) {
	if f.header == nil {
		return 0, "", false
	}
	if v := strings.TrimSpace(f.header.Get("Retry-After")); v != "" {
		if after, ok := retryAfter(v); ok {
			return after, "Retry-After: " + v, true
		}
	}
	if !f.limited || f.rate == "" || strings.TrimSpace(f.header.Get(f.rate+"Remaining")) != "0" {
		return 0, "", false
	}
	reset, err := strconv.ParseInt(strings.TrimSpace(f.header.Get(f.rate+"Reset")), 10, 64)
	if err != nil {
		return 0, "", false
	}
	// Up to the second, not sooner than the reset; a second when it has
	// passed, which only a clock behind GitHub's sees.
	after := max((time.Until(time.Unix(min(reset, time.Now().Unix()+86400), 0)) + time.Second - 1).Truncate(time.Second), time.Second)
	return after, fmt.Sprintf("%sReset: in %d s", f.rate, int(after.Seconds())), true
}

// aside is a note in parentheses, after the failure it explains.
func aside(note string) string {
	if note == "" {
		return ""
	}
	return " (" + note + ")"
}

// lasting says whether a call failed for good: a host that does not
// exist, a certificate refused.
func lasting(err error) bool {
	var dns *net.DNSError
	var cert *tls.CertificateVerificationError
	return errors.As(err, &dns) && dns.IsNotFound || errors.As(err, &cert)
}

// unsent says whether a call certainly never reached GitLab: no connection
// was made to it (a refused connection, a name not resolved for a moment,
// a TLS handshake that did not end). A timeout once connected may follow a
// request GitLab took.
func unsent(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && op.Op == "dial" || strings.Contains(err.Error(), "TLS handshake timeout")
}

// retryAfter reads a Retry-After header: seconds, or an HTTP date. Seconds
// past a day are a day: never waited for, and no overflow.
func retryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if s, err := strconv.Atoi(v); err == nil {
		return time.Duration(min(max(s, 0), 86400)) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0).Round(time.Second), true
	}
	return 0, false
}
