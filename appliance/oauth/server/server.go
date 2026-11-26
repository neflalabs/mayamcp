// Package server is the optional, opt-in server glue for the OAuth appliance.
// The primitives in the parent package stay dependency-free; this package
// exposes discovery endpoints and a Bearer-token guard for an MCP endpoint.
package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/neflalabs/mayamcp/appliance/oauth"
)

// Config assembles the discovery server from already-built primitives.
type Config struct {
	Issuer  string
	Key     *oauth.SigningKey
	Handler *oauth.Handler
	Logger  *slog.Logger
	Now     func() time.Time
}

// Server publishes metadata/JWKS and guards an MCP endpoint.
type Server struct {
	cfg Config
}

// New validates the glue configuration. A Handler is required so that the
// discovery document never advertises endpoints the application did not wire.
func New(cfg Config) (*Server, error) {
	if cfg.Issuer == "" || cfg.Key == nil || cfg.Handler == nil {
		return nil, errors.New("oauth/server: Issuer, Key and Handler are required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Server{cfg: cfg}, nil
}

// RegisterRoutes adds discovery, JWKS, and login/callback/logout endpoints to
// a caller-owned mux. It never replaces or closes the mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", s.metadata)
	mux.HandleFunc("GET /.well-known/jwks.json", s.jwks)
	mux.HandleFunc("GET /oauth/login", s.cfg.Handler.Login)
	mux.HandleFunc("GET /oauth/callback", s.cfg.Handler.Callback)
	mux.HandleFunc("POST /oauth/logout", s.cfg.Handler.Logout)
}

func (s *Server) metadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                           s.cfg.Issuer,
		"authorization_endpoint":           s.cfg.Issuer + "/oauth/authorize",
		"token_endpoint":                   s.cfg.Issuer + "/oauth/token",
		"jwks_uri":                         s.cfg.Issuer + "/.well-known/jwks.json",
		"response_types_supported":         []string{"code"},
		"code_challenge_methods_supported": []string{"S256"},
		"grant_types_supported":            []string{"authorization_code", "refresh_token"},
		"authorization_response_iss_parameter_supported": true,
	})
}

func (s *Server) jwks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"keys": []any{s.cfg.Key.JWK()},
	})
}

// MCP accepts only machine credentials: a browser session cookie is never
// accepted against the tool endpoint, so a leaked cookie cannot be replayed.
func (s *Server) MCP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			s.unauthorized(w)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if _, err := oauth.VerifyToken(token, s.cfg.Key.Public(), s.cfg.Issuer, s.cfg.Now); err != nil {
			s.unauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
