// Package oauth contains the reusable, transport-agnostic security primitives
// extracted from a full OAuth authorization server. It is an optional
// appliance module: it deliberately has no dependency on mayamcp or its api
// package, and can be used by any net/http application.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	stateBytes = 32
	pkceBytes  = 32
	defaultTTL = 5 * time.Minute
	minPKCELen = 43
	maxPKCELen = 128
)

// clock returns a usable clock, allowing tests to inject a fixed time.
func clock(now func() time.Time) func() time.Time {
	if now == nil {
		return time.Now
	}
	return now
}

// NewState creates a single-use CSRF state token. The token is a URL-safe
// random value followed by its issuance time; the caller must store the token
// and consume it through a Store (see S23). The timestamp lets VerifyState
// enforce a TTL without relying on global state.
func NewState(now func() time.Time) (string, error) {
	buf := make([]byte, stateBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("oauth: generate state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf) + "." +
		strconv.FormatInt(clock(now)().UnixNano(), 10), nil
}

// VerifyState checks that got matches stored exactly (constant-time) and that
// stored has not expired. A non-positive ttl selects a safe default.
func VerifyState(stored, got string, ttl time.Duration, now func() time.Time) error {
	issuedAt, err := parseStateTime(stored)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = defaultTTL
	}
	current := clock(now)()
	if issuedAt.After(current) {
		return errors.New("oauth: state was issued in the future")
	}
	if current.Sub(issuedAt) > ttl {
		return errors.New("oauth: state expired")
	}
	// Constant-time comparison; lengths differ is not a shortcut here because
	// subtle.ConstantTimeCompare handles it without leaking the prefix.
	if subtle.ConstantTimeCompare([]byte(stored), []byte(got)) != 1 {
		return errors.New("oauth: state mismatch")
	}
	return nil
}

// parseStateTime validates the random part and returns the embedded timestamp.
func parseStateTime(raw string) (time.Time, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return time.Time{}, errors.New("oauth: malformed state")
	}
	value, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(value) != stateBytes {
		return time.Time{}, errors.New("oauth: malformed state value")
	}
	nanos, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, errors.New("oauth: malformed state timestamp")
	}
	return time.Unix(0, nanos).UTC(), nil
}

// NewPKCE generates a verifier/challenge pair using RFC 7636 S256. The
// verifier is 43 URL-safe characters; the challenge is its SHA-256 digest.
func NewPKCE() (verifier, challenge string, err error) {
	buf := make([]byte, pkceBytes)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("oauth: generate PKCE verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// VerifyPKCE checks a verifier against an S256 challenge in constant time.
func VerifyPKCE(verifier, challenge string) error {
	if len(verifier) < minPKCELen || len(verifier) > maxPKCELen {
		return errors.New("oauth: invalid PKCE verifier length")
	}
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(want), []byte(challenge)) != 1 {
		return errors.New("oauth: PKCE challenge mismatch")
	}
	return nil
}
