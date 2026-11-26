package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

func TestNewStateIsUniqueAndURLEncoded(t *testing.T) {
	a, err := NewState(func() time.Time { return testNow })
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	b, err := NewState(func() time.Time { return testNow })
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if a == b {
		t.Error("two states are identical")
	}
	if len(a) < 40 {
		t.Errorf("state too short: %q", a)
	}
	if strings.ContainsAny(a, "+/=") {
		t.Errorf("state is not URL-safe: %q", a)
	}
}

func TestVerifyStateAcceptsFreshState(t *testing.T) {
	s, err := NewState(func() time.Time { return testNow })
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyState(s, s, time.Minute, func() time.Time { return testNow.Add(30 * time.Second) }); err != nil {
		t.Errorf("fresh state rejected: %v", err)
	}
}

func TestVerifyStateRejectsExpiredState(t *testing.T) {
	s, err := NewState(func() time.Time { return testNow })
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyState(s, s, time.Minute, func() time.Time { return testNow.Add(time.Minute + time.Nanosecond) }); err == nil {
		t.Error("expired state was accepted")
	}
}

func TestVerifyStateRejectsMismatch(t *testing.T) {
	base := func() time.Time { return testNow }
	stored, _ := NewState(base)
	other, _ := NewState(base)
	if err := VerifyState(stored, other, time.Minute, base); err == nil {
		t.Error("mismatched state was accepted")
	}
}

func TestVerifyStateRejectsTamperedTimestamp(t *testing.T) {
	stored, err := NewState(func() time.Time { return testNow })
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(stored, ".")
	if len(parts) != 2 {
		t.Fatalf("unexpected state format: %q", stored)
	}
	parts[1] = "9999999999999999999"
	tampered := strings.Join(parts, ".")
	if err := VerifyState(tampered, tampered, time.Minute, func() time.Time { return testNow }); err == nil {
		t.Error("future-dated state was accepted")
	}
}

func TestVerifyStateRejectsMalformedInput(t *testing.T) {
	for _, raw := range []string{"", "no-dot", "abc.not-a-number"} {
		if err := VerifyState(raw, raw, time.Minute, func() time.Time { return testNow }); err == nil {
			t.Errorf("malformed state %q was accepted", raw)
		}
	}
}

func TestNewPKCEProducesS256Challenge(t *testing.T) {
	verifier, challenge, err := NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE: %v", err)
	}
	if len(verifier) < 43 {
		t.Errorf("verifier too short: %q", verifier)
	}
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Errorf("challenge = %q, want S256 %q", challenge, want)
	}
	if err := VerifyPKCE(verifier, challenge); err != nil {
		t.Errorf("VerifyPKCE rejected a matching pair: %v", err)
	}
}

func TestVerifyPKCERejectsWrongVerifier(t *testing.T) {
	_, challenge, _ := NewPKCE()
	other, _, _ := NewPKCE()
	if err := VerifyPKCE(other, challenge); err == nil {
		t.Error("wrong PKCE verifier was accepted")
	}
}

func TestNewPKCEIsRandom(t *testing.T) {
	a, _, _ := NewPKCE()
	b, _, _ := NewPKCE()
	if a == b {
		t.Error("two PKCE verifiers are identical")
	}
}

func TestStateAndPKCEAcceptDefaultClock(t *testing.T) {
	s, err := NewState(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyState(s, s, time.Minute, nil); err != nil {
		t.Errorf("default clock rejected fresh state: %v", err)
	}
	if _, _, err := NewPKCE(); err != nil {
		t.Errorf("NewPKCE: %v", err)
	}
}
