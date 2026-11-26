package oauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CookieOptions controls a signed session cookie. TTL and Now are used by
// both signing and verification; Secure/SameSite are applied by
// NewSessionCookie.
type CookieOptions struct {
	TTL      time.Duration
	Secure   bool
	SameSite http.SameSite
	Now      func() time.Time
}

// SignCookie returns an opaque, URL-safe value carrying value and its issue
// time, authenticated with HMAC-SHA256 over the cookie name. The name binding
// prevents replaying a cookie under a different cookie name.
func SignCookie(name string, value []byte, key []byte, o CookieOptions) (string, error) {
	if len(key) < 16 {
		return "", errors.New("oauth: cookie key must be at least 16 bytes")
	}
	issuedAt := clock(o.Now)().UnixNano()
	body := base64.RawURLEncoding.EncodeToString(value) + "." +
		strconv.FormatInt(issuedAt, 10)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(name + "." + body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// VerifyCookie checks the MAC before decoding anything, then enforces the TTL.
func VerifyCookie(name, raw string, key []byte, ttl time.Duration, now func() time.Time) ([]byte, error) {
	if len(key) < 16 {
		return nil, errors.New("oauth: cookie key must be at least 16 bytes")
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, errors.New("oauth: malformed cookie")
	}
	want, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("oauth: malformed cookie signature")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(name + "." + parts[0] + "." + parts[1]))
	if subtle.ConstantTimeCompare(want, mac.Sum(nil)) != 1 {
		return nil, errors.New("oauth: cookie signature mismatch")
	}
	value, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("oauth: malformed cookie value")
	}
	issuedAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, errors.New("oauth: malformed cookie timestamp")
	}
	if ttl <= 0 {
		ttl = defaultTTL
	}
	current := clock(now)()
	issued := time.Unix(0, issuedAt)
	if issued.After(current) {
		return nil, errors.New("oauth: cookie issued in the future")
	}
	if current.Sub(issued) > ttl {
		return nil, errors.New("oauth: cookie expired")
	}
	return value, nil
}

// NewSessionCookie wraps a signed value in an http.Cookie with safe defaults.
func NewSessionCookie(name, raw string, o CookieOptions) *http.Cookie {
	if o.TTL <= 0 {
		o.TTL = defaultTTL
	}
	return &http.Cookie{
		Name:     name,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   o.Secure,
		SameSite: o.SameSite,
		MaxAge:   int(o.TTL.Seconds()),
	}
}
