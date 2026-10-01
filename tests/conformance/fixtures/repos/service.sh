#!/bin/sh
# A small service whose technical doc describes four source files, one
# section each: where judging a doc whole and judging it in parts, a share of
# its sources at a time, can be compared on the same defects (ADR-0009).
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p src/service docs/tech
cat > src/service/config.go <<'GO'
package service

import "time"

// Defaults for a service started with no configuration.
const (
	DefaultPort           = 8080
	DefaultMaxConnections = 100
	DefaultReadTimeout    = 30 * time.Second
	DefaultSessionTTL     = 24 * time.Hour
	DefaultLogLevel       = "info"
)

// Config holds what the service reads when it starts.
type Config struct {
	Port           int
	MaxConnections int
	ReadTimeout    time.Duration
	SessionTTL     time.Duration
	LogLevel       string
}

// Load fills the fields left unset with their defaults.
func Load(c Config) Config {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.MaxConnections == 0 {
		c.MaxConnections = DefaultMaxConnections
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultReadTimeout
	}
	if c.SessionTTL == 0 {
		c.SessionTTL = DefaultSessionTTL
	}
	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}
	return c
}
GO
cat > src/service/server.go <<'GO'
package service

import (
	"fmt"
	"net/http"
)

// Serve listens on every interface, on the configured port. Past
// MaxConnections requests at once, it answers 503 Service Unavailable.
func Serve(c Config, h http.Handler) error {
	slots := make(chan struct{}, c.MaxConnections)
	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", c.Port),
		ReadTimeout: c.ReadTimeout,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
				h.ServeHTTP(w, r)
			default:
				http.Error(w, "busy", http.StatusServiceUnavailable)
			}
		}),
	}
	return srv.ListenAndServe()
}
GO
cat > src/service/store.go <<'GO'
package service

import (
	"sync"
	"time"
)

// Store keeps sessions in memory: they are lost when the service stops.
type Store struct {
	mu       sync.Mutex
	ttl      time.Duration
	sessions map[string]time.Time
}

// NewStore keeps each session for the configured session TTL.
func NewStore(c Config) *Store {
	s := &Store{ttl: c.SessionTTL, sessions: map[string]time.Time{}}
	go s.cleanup()
	return s
}

// Touch starts or renews a session.
func (s *Store) Touch(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = time.Now().Add(s.ttl)
}

// cleanup drops the expired sessions every ten minutes.
func (s *Store) cleanup() {
	for range time.Tick(10 * time.Minute) {
		s.mu.Lock()
		for id, until := range s.sessions {
			if time.Now().After(until) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}
GO
cat > src/service/auth.go <<'GO'
package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// ErrExpired is returned for a token past its expiry.
var ErrExpired = errors.New("token expired")

// VerifyToken checks a request's `Authorization: Bearer` token: its
// HMAC-SHA256 signature, then its expiry.
func VerifyToken(header string, key []byte, now time.Time) error {
	tok, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return errors.New("no bearer token")
	}
	body, sig, ok := strings.Cut(tok, ".")
	if !ok {
		return errors.New("malformed token")
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(sig)) {
		return errors.New("bad signature")
	}
	until, err := time.Parse(time.RFC3339, body)
	if err != nil {
		return err
	}
	if now.After(until) {
		return ErrExpired
	}
	return nil
}
GO
cat > docs/tech/service.md <<'DOC'
---
type: reference
sources: [src/service/config.go, src/service/server.go, src/service/store.go, src/service/auth.go]
checked: HEAD
---
# The service

## Starting

The service reads its configuration when it starts, and fills what is left
unset with defaults. It listens on every interface, on port 8080 unless the
configuration says otherwise. It logs at the `info` level by default.

## Connections

At most 100 requests are served at once; past that, the service answers
`503 Service Unavailable` rather than queueing them. A request whose body is
not read within 30 seconds is dropped.

## Sessions

Sessions are kept in memory, so they are lost when the service stops. Each
one lasts the configured session TTL, 24 hours by default, and is renewed
each time the user comes back. Expired sessions are dropped every ten
minutes.

## Authentication

Each request carries an `Authorization: Bearer` token, checked by
`VerifyToken`: first its HMAC-SHA256 signature, then its expiry. An expired
token is refused with `ErrExpired`.
DOC
git add . && git commit -q -m "feat(service): serve requests, keep sessions, check tokens"
# Replace the placeholder with the real commit, as a person confirming the doc would.
C=$(git rev-parse --short HEAD)
sed -i "s/checked: HEAD/checked: $C/" docs/tech/service.md
git add . && git commit -q -m "docs: confirm the service doc against the code"
