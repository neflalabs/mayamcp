package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidRequestHost(t *testing.T) {
	for _, host := range []string{
		"example.com", "localhost", "127.0.0.1:8080", "[::1]:8080", "[2001:db8::1]",
		"my_host.example:8443", "a-b.example",
	} {
		if !ValidRequestHost(host) {
			t.Errorf("ValidRequestHost(%q) = false, want true", host)
		}
	}
	for _, host := range []string{
		"", "bad host", "example.com:abc", "example.com:99999", "evil\nheader",
		"evil\\bad", "[::1", "a..b", string(make([]byte, 254)),
	} {
		if ValidRequestHost(host) {
			t.Errorf("ValidRequestHost(%q) = true, want false", host)
		}
	}
}

func TestGuardHostMiddleware(t *testing.T) {
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called++; w.WriteHeader(http.StatusOK) })
	handler := GuardHost(next)

	ok := httptest.NewRequest(http.MethodGet, "http://example.com/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, ok)
	if rec.Code != http.StatusOK || called != 1 {
		t.Errorf("valid host: code=%d called=%d", rec.Code, called)
	}

	bad := httptest.NewRequest(http.MethodGet, "http://example.com/mcp", nil)
	bad.Host = "evil\nheader"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, bad)
	if rec.Code != http.StatusBadRequest || called != 1 {
		t.Errorf("invalid host: code=%d called=%d", rec.Code, called)
	}
}
