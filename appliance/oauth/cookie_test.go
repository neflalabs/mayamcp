package oauth

import (
	"bytes"
	"net/http"
	"testing"
	"time"
)

var cookieKey = bytes.Repeat([]byte("k"), 32)

func TestSignCookieRoundTrip(t *testing.T) {
	opts := CookieOptions{TTL: time.Minute, Secure: true, SameSite: http.SameSiteLaxMode, Now: func() time.Time { return testNow }}
	raw, err := SignCookie("sid", []byte("session-1"), cookieKey, opts)
	if err != nil {
		t.Fatalf("SignCookie: %v", err)
	}
	got, err := VerifyCookie("sid", raw, cookieKey, time.Minute, func() time.Time { return testNow.Add(30 * time.Second) })
	if err != nil {
		t.Fatalf("VerifyCookie: %v", err)
	}
	if string(got) != "session-1" {
		t.Fatalf("value = %q, want session-1", got)
	}
}

func TestVerifyCookieRejectsTampering(t *testing.T) {
	opts := CookieOptions{TTL: time.Minute, Now: func() time.Time { return testNow }}
	raw, err := SignCookie("sid", []byte("session-1"), cookieKey, opts)
	if err != nil {
		t.Fatal(err)
	}
	tampered := raw[:len(raw)-2] + "xx"
	if _, err := VerifyCookie("sid", tampered, cookieKey, time.Minute, func() time.Time { return testNow }); err == nil {
		t.Error("tampered cookie was accepted")
	}
}

func TestVerifyCookieRejectsWrongNameAndKey(t *testing.T) {
	opts := CookieOptions{TTL: time.Minute, Now: func() time.Time { return testNow }}
	raw, _ := SignCookie("sid", []byte("v"), cookieKey, opts)
	now := func() time.Time { return testNow }
	if _, err := VerifyCookie("other", raw, cookieKey, time.Minute, now); err == nil {
		t.Error("cookie accepted under a different name")
	}
	if _, err := VerifyCookie("sid", raw, bytes.Repeat([]byte("j"), 32), time.Minute, now); err == nil {
		t.Error("cookie accepted with a different key")
	}
}

func TestVerifyCookieRejectsExpiredAndMalformed(t *testing.T) {
	opts := CookieOptions{TTL: time.Minute, Now: func() time.Time { return testNow }}
	raw, _ := SignCookie("sid", []byte("v"), cookieKey, opts)
	if _, err := VerifyCookie("sid", raw, cookieKey, time.Minute, func() time.Time { return testNow.Add(2 * time.Minute) }); err == nil {
		t.Error("expired cookie was accepted")
	}
	for _, bad := range []string{"", "a.b", "a.b.c.d"} {
		if _, err := VerifyCookie("sid", bad, cookieKey, time.Minute, func() time.Time { return testNow }); err == nil {
			t.Errorf("malformed cookie %q was accepted", bad)
		}
	}
}
