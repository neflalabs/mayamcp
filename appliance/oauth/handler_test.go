package oauth

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type funcVerifier func(context.Context, string) (Session, error)

func (f funcVerifier) Verify(ctx context.Context, subject string) (Session, error) {
	return f(ctx, subject)
}

func testHandlerConfig(t *testing.T, exchange func(context.Context, string, string) (Session, error)) (Config, Store, *slog.Logger, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	cfg := Config{
		ClientID:    "client-1",
		AuthURL:     "https://provider.example/authorize",
		RedirectURL: "https://app.example/callback",
		Scopes:      []string{"tools:read"},
		StateTTL:    time.Minute,
		Cookie: CookieOptions{
			Name:     "app_session",
			TTL:      time.Hour,
			Key:      bytes.Repeat([]byte("k"), 32),
			Now:      func() time.Time { return testNow },
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		},
		Exchange: exchange,
		Now:      func() time.Time { return testNow },
	}
	return cfg, NewMemoryStore(), logger, &logs
}

func TestNewRejectsIncompleteConfig(t *testing.T) {
	cfg, st, logger, _ := testHandlerConfig(t, nil)
	if _, err := New(cfg, nil, st, logger); err == nil {
		t.Fatal("nil verifier was accepted")
	}
	cfg.ClientID = ""
	if _, err := New(cfg, funcVerifier(func(context.Context, string) (Session, error) { return Session{}, nil }), st, logger); err == nil {
		t.Error("missing ClientID was accepted")
	}
}

func TestLoginRedirectAndStateCookie(t *testing.T) {
	cfg, st, logger, _ := testHandlerConfig(t, nil)
	h, err := New(cfg, funcVerifier(func(context.Context, string) (Session, error) { return Session{}, nil }), st, logger)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("code = %d, want 302", rec.Code)
	}
	target, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if target.Host != "provider.example" {
		t.Errorf("redirect host = %q", target.Host)
	}
	q := target.Query()
	if q.Get("client_id") != "client-1" || q.Get("code_challenge_method") != "S256" || q.Get("state") == "" || q.Get("code_challenge") == "" {
		t.Errorf("query = %v", q)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "app_session" || !cookies[0].HttpOnly {
		t.Errorf("cookies = %+v", cookies)
	}
}

func TestCallbackRejectsStateMismatch(t *testing.T) {
	cfg, st, logger, _ := testHandlerConfig(t, nil)
	h, _ := New(cfg, funcVerifier(func(context.Context, string) (Session, error) { return Session{}, nil }), st, logger)
	req := httptest.NewRequest(http.MethodGet, "/callback?code=abc&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: "app_session", Value: "wrong"})
	rec := httptest.NewRecorder()
	h.Callback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestCallbackSuccessAndSessionMiddleware(t *testing.T) {
	exchanged := false
	cfg, st, logger, logs := testHandlerConfig(t, func(_ context.Context, code, verifier string) (Session, error) {
		exchanged = true
		if code != "code-1" || verifier == "" {
			t.Errorf("exchange got code=%q verifier=%q", code, verifier)
		}
		return Session{Subject: "user-1", ExpiresAt: testNow.Add(time.Hour)}, nil
	})
	verifier := funcVerifier(func(context.Context, string) (Session, error) { return Session{}, nil })
	h, err := New(cfg, verifier, st, logger)
	if err != nil {
		t.Fatal(err)
	}

	loginRec := httptest.NewRecorder()
	h.Login(loginRec, httptest.NewRequest(http.MethodGet, "/login", nil))
	stateCookie := loginRec.Result().Cookies()[0]
	target, _ := url.Parse(loginRec.Header().Get("Location"))
	state := target.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/callback?code=code-1&state="+url.QueryEscape(state), nil)
	req.AddCookie(stateCookie)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)
	if rec.Code != http.StatusFound || !exchanged {
		t.Fatalf("callback code=%d exchanged=%v", rec.Code, exchanged)
	}

	var session *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "app_session" {
			session = c
		}
	}
	if session == nil {
		t.Fatal("no session cookie set")
	}
	authed := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	authed.AddCookie(session)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusOK) })
	mwRec := httptest.NewRecorder()
	h.Middleware(next).ServeHTTP(mwRec, authed)
	if !called || mwRec.Code != http.StatusOK {
		t.Errorf("authenticated request: called=%v code=%d", called, mwRec.Code)
	}

	anonRec := httptest.NewRecorder()
	h.Middleware(next).ServeHTTP(anonRec, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if anonRec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous code = %d, want 401", anonRec.Code)
	}
	if strings.Contains(logs.String(), "code-1") || strings.Contains(logs.String(), "Bearer") {
		t.Errorf("secret leaked into log: %q", logs.String())
	}
}
