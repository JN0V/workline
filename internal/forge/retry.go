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

// A GitLab call that fails for a moment is tried again: a read on no
// answer, a 5xx but 501 or a 429; a write only when GitLab certainly did
// not take it (no connection made, or a 429). A write answered late or
// with a 5xx is not sent twice: its failure says so. Why, and what is
// borrowed from GitLab's own client: docs/gitlab/calls.md.

// retry is how a call is tried again: tries in all, the pause after the
// first (doubled after each), the longest Retry-After waited for.
var retry = struct {
	tries   int
	pause   time.Duration
	waitMax time.Duration
}{tries: 3, pause: 2 * time.Second, waitMax: 60 * time.Second}

// httpClient gives each request a minute: gitlab.com answers in under one
// second (#489).
var httpClient = &http.Client{Timeout: 60 * time.Second}

// answer is a call's answer, read whole.
type answer struct {
	status string
	code   int
	header http.Header
	data   []byte
	note   string // when it failed for a moment: why it was not tried again, or how often it was
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
		wait, note := again(read, try, a, err)
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

// again says how long to wait before a try is sent again, -1 when it is
// not because it did not fail for a moment; note, when it failed for a
// moment, is why it is not sent again, or how often it was.
func again(read bool, try int, a *answer, err error) (wait time.Duration, note string) {
	switch {
	case err != nil && lasting(err):
		return -1, ""
	case err != nil && !read && !unsent(err):
		return -1, "a write: not sent again, GitLab may have taken it"
	case err != nil:
	case a.code == http.StatusTooManyRequests:
	case a.code >= 500 && a.code != http.StatusNotImplemented && !read:
		return -1, "a write: not sent again, GitLab may have taken it"
	case a.code >= 500 && a.code != http.StatusNotImplemented:
	default:
		return -1, ""
	}
	wait = retry.pause << (try - 1)
	if a != nil {
		if after, ok := retryAfter(a.header.Get("Retry-After")); ok {
			if after > retry.waitMax {
				return -1, fmt.Sprintf("GitLab asks to wait (Retry-After: %s), longer than workline waits (%d s): not tried again",
					strings.TrimSpace(a.header.Get("Retry-After")), int(retry.waitMax.Seconds()))
			}
			wait = after
		}
	}
	if try >= retry.tries {
		return -1, fmt.Sprintf("tried %d times", try)
	}
	return wait, ""
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
