package oauth

import (
	"strings"
	"testing"
	"time"
)

func TestSigningKeyPEMAndJWKRoundTrip(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	pemBytes, err := key.PEM()
	if err != nil {
		t.Fatalf("PEM: %v", err)
	}
	restored, err := ParseSigningKey(pemBytes)
	if err != nil {
		t.Fatalf("ParseSigningKey: %v", err)
	}
	if key.KeyID() != restored.KeyID() {
		t.Errorf("kid changed after round trip: %q vs %q", key.KeyID(), restored.KeyID())
	}
	jwk := key.JWK()
	for _, field := range []string{"kty", "crv", "x", "y", "use", "alg", "kid"} {
		if jwk[field] == "" {
			t.Errorf("JWK missing %q", field)
		}
	}
	if jwk["alg"] != "ES256" || jwk["crv"] != "P-256" {
		t.Errorf("unexpected JWK: %+v", jwk)
	}
}

func TestParseSigningKeyRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"not-pem", "-----BEGIN PRIVATE KEY-----\nQQ==\n-----END PRIVATE KEY-----\n"} {
		if _, err := ParseSigningKey(raw); err == nil {
			t.Errorf("accepted invalid key %q", raw)
		}
	}
}

func TestIssueAndVerifyToken(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	claims := TokenClaims{Issuer: "https://mcp.example", Subject: "user-1", Audience: "https://mcp.example", Scope: "tools:read", Now: func() time.Time { return testNow }}
	token, err := key.IssueToken(claims)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if len(strings.Split(token, ".")) != 3 {
		t.Fatalf("unexpected JWT shape: %q", token)
	}
	got, err := VerifyToken(token, key.Public(), "https://mcp.example", func() time.Time { return testNow.Add(5 * time.Minute) })
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if got.Subject != "user-1" || got.Scope != "tools:read" || got.ID == "" {
		t.Errorf("claims = %+v", got)
	}
}

func TestVerifyTokenRejectsTamperWrongIssuerAndExpiry(t *testing.T) {
	key, _ := GenerateSigningKey()
	claims := TokenClaims{Issuer: "https://mcp.example", Subject: "u", Audience: "https://mcp.example", Now: func() time.Time { return testNow }}
	token, _ := key.IssueToken(claims)
	now := func() time.Time { return testNow.Add(time.Minute) }

	parts := strings.Split(token, ".")
	parts[1] = parts[1][:len(parts[1])-2] + "xx"
	if _, err := VerifyToken(strings.Join(parts, "."), key.Public(), "https://mcp.example", now); err == nil {
		t.Error("tampered claims were accepted")
	}
	if _, err := VerifyToken(token, key.Public(), "https://other.example", now); err == nil {
		t.Error("wrong issuer was accepted")
	}
	if _, err := VerifyToken(token, key.Public(), "https://mcp.example", func() time.Time { return testNow.Add(24 * time.Hour) }); err == nil {
		t.Error("expired token was accepted")
	}
	if _, err := VerifyToken("a.b", key.Public(), "https://mcp.example", now); err == nil {
		t.Error("malformed token was accepted")
	}
}
