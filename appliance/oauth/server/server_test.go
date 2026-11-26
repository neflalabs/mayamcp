package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neflalabs/mayamcp/appliance/oauth"
)

const testIssuer = "https://mcp.example"

type testVerifier struct{}

func (testVerifier) Verify(context.Context, string) (oauth.Session, error) {
	return oauth.Session{}, nil
}

func testServer(t *testing.T) (*Server, *oauth.SigningKey) {
	t.Helper()
	key, err := oauth.GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	handler, err := oauth.New(oauth.Config{
		ClientID:    "client-1",
		AuthURL:     "https://provider.example/authorize",
		RedirectURL: testIssuer + "/oauth/callback",
		Cookie: oauth.CookieOptions{
			Name: "app_session",
			Key:  bytes.Repeat([]byte("k"), 32),
		},
	}, testVerifier{}, oauth.NewMemoryStore(), nil)
	if err != nil {
		t.Fatalf("oauth.New: %v", err)
	}
	cfg := Config{
		Issuer:  testIssuer,
		Key:     key,
		Handler: handler,
		Logger:  slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Now:     func() time.Time { return time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC) },
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s, key
}

func TestNewRejectsIncompleteConfig(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("empty config was accepted")
	}
	key, _ := oauth.GenerateSigningKey()
	if _, err := New(Config{Issuer: testIssuer, Key: key}); err == nil {
		t.Error("config without Handler was accepted")
	}
}

func TestWellKnownMetadataAndJWKS(t *testing.T) {
	s, key := testServer(t)
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metadata code = %d", rec.Code)
	}
	var meta map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["issuer"] != testIssuer || meta["jwks_uri"] == "" {
		t.Errorf("metadata = %+v", meta)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("jwks code = %d", rec.Code)
	}
	var jwks struct {
		Keys []struct {
			KID string `json:"kid"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0].KID != key.KeyID() {
		t.Errorf("jwks = %+v, want kid %q", jwks, key.KeyID())
	}
}

func TestMCPAcceptsBearerRejectsSessionCookie(t *testing.T) {
	s, key := testServer(t)
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called++; w.WriteHeader(http.StatusOK) })
	handler := s.MCP(next)

	token, err := key.IssueToken(oauth.TokenClaims{
		Issuer:   testIssuer,
		Subject:  "user-1",
		Audience: testIssuer + "/mcp",
		Now:      func() time.Time { return time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || called != 1 {
		t.Errorf("bearer request: code=%d called=%d", rec.Code, called)
	}

	// A browser session cookie must never authenticate /mcp.
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.AddCookie(&http.Cookie{Name: "app_session", Value: "anything"})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || called != 1 {
		t.Errorf("session cookie: code=%d called=%d", rec.Code, called)
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer bad.token.value")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || called != 1 {
		t.Errorf("bad bearer: code=%d called=%d", rec.Code, called)
	}
}
