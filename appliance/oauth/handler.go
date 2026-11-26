package oauth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Session is the authenticated identity produced by Exchange.
type Session struct {
	Subject   string
	ExpiresAt time.Time
	Extra     map[string]string
}

// Config configures the optional HTTP handler. Provider-specific work is
// supplied through Exchange; this module never hardcodes a provider or
// credentials.
type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RedirectURL  string
	Scopes       []string
	StateTTL     time.Duration
	Cookie       CookieOptions
	Exchange     func(ctx context.Context, code, verifier string) (Session, error)
	Now          func() time.Time
}

// Verifier confirms that a session subject is still valid. It is intentionally
// separate from Exchange so a consumer can revoke sessions centrally.
type Verifier interface {
	Verify(ctx context.Context, subject string) (Session, error)
}

// Handler owns the login/callback/logout flow and can wrap any http.Handler.
type Handler struct {
	cfg      Config
	verifier Verifier
	store    Store
	logger   *slog.Logger
}

// New validates configuration and returns a ready Handler.
func New(cfg Config, verifier Verifier, st Store, logger *slog.Logger) (*Handler, error) {
	if verifier == nil {
		return nil, errors.New("oauth: Verifier is required")
	}
	if st == nil {
		return nil, errors.New("oauth: Store is required")
	}
	if cfg.ClientID == "" || cfg.AuthURL == "" || cfg.RedirectURL == "" {
		return nil, errors.New("oauth: ClientID, AuthURL and RedirectURL are required")
	}
	if cfg.Cookie.Name == "" || len(cfg.Cookie.Key) < 16 {
		return nil, errors.New("oauth: Cookie.Name and a 16-byte Cookie.Key are required")
	}
	if cfg.StateTTL <= 0 {
		cfg.StateTTL = defaultTTL
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Handler{cfg: cfg, verifier: verifier, store: st, logger: logger}, nil
}

// Login stores a single-use state with its PKCE verifier and redirects to the
// configured authorization endpoint.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state, err := NewState(h.cfg.Now)
	if err != nil {
		http.Error(w, "oauth: state error", http.StatusInternalServerError)
		return
	}
	verifier, challenge, err := NewPKCE()
	if err != nil {
		http.Error(w, "oauth: PKCE error", http.StatusInternalServerError)
		return
	}
	if err := h.store.SaveState(ctx, "state:"+state, []byte(verifier), h.cfg.StateTTL); err != nil {
		http.Error(w, "oauth: storage error", http.StatusInternalServerError)
		return
	}
	raw, err := SignCookie(h.cfg.Cookie.Name, []byte(state), h.cfg.Cookie.Key, h.cfg.Cookie)
	if err != nil {
		http.Error(w, "oauth: cookie error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, NewSessionCookie(h.cfg.Cookie.Name, raw, h.cfg.Cookie))

	target, err := url.Parse(h.cfg.AuthURL)
	if err != nil {
		http.Error(w, "oauth: invalid AuthURL", http.StatusInternalServerError)
		return
	}
	q := target.Query()
	q.Set("response_type", "code")
	q.Set("client_id", h.cfg.ClientID)
	q.Set("redirect_uri", h.cfg.RedirectURL)
	q.Set("scope", strings.Join(h.cfg.Scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	target.RawQuery = q.Encode()
	h.logger.Info("auth: login ✓")
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// Callback validates state, exchanges the code through Config.Exchange, and
// sets a signed session cookie. Nothing sensitive is logged.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(h.cfg.Cookie.Name)
	if err != nil {
		http.Error(w, "oauth: missing state cookie", http.StatusBadRequest)
		return
	}
	cookieState, err := VerifyCookie(h.cfg.Cookie.Name, cookie.Value, h.cfg.Cookie.Key, h.cfg.Cookie.TTL, h.cfg.Now)
	if err != nil || string(cookieState) != state {
		http.Error(w, "oauth: invalid state cookie", http.StatusBadRequest)
		return
	}
	if err := VerifyState(state, state, h.cfg.StateTTL, h.cfg.Now); err != nil {
		http.Error(w, "oauth: expired state", http.StatusBadRequest)
		return
	}
	verifier, ok, err := h.store.ConsumeState(ctx, "state:"+state)
	if err != nil || !ok {
		http.Error(w, "oauth: state already used", http.StatusBadRequest)
		return
	}
	session, err := h.cfg.Exchange(ctx, code, string(verifier))
	if err != nil {
		http.Error(w, "oauth: token exchange failed", http.StatusBadGateway)
		return
	}

	value := session.Subject + "|" + strconv.FormatInt(session.ExpiresAt.Unix(), 10)
	raw, err := SignCookie(h.cfg.Cookie.Name, []byte(value), h.cfg.Cookie.Key, h.cfg.Cookie)
	if err != nil {
		http.Error(w, "oauth: cookie error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, NewSessionCookie(h.cfg.Cookie.Name, raw, h.cfg.Cookie))
	h.logger.Info("auth: callback ✓")
	http.Redirect(w, r, h.cfg.RedirectURL, http.StatusFound)
}

// Logout clears the session cookie and redirects back to the application.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.Cookie.Name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Cookie.Secure,
		SameSite: h.cfg.Cookie.SameSite,
		MaxAge:   -1,
	})
	h.logger.Info("auth: logout ✓")
	http.Redirect(w, r, h.cfg.RedirectURL, http.StatusFound)
}

// SessionFromRequest returns the signed session carried by the request, if
// any. It never accepts a token from query parameters or headers.
func (h *Handler) SessionFromRequest(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(h.cfg.Cookie.Name)
	if err != nil {
		return Session{}, false
	}
	value, err := VerifyCookie(h.cfg.Cookie.Name, cookie.Value, h.cfg.Cookie.Key, h.cfg.Cookie.TTL, h.cfg.Now)
	if err != nil {
		return Session{}, false
	}
	parts := strings.SplitN(string(value), "|", 2)
	if len(parts) != 2 {
		return Session{}, false
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || clock(h.cfg.Now)().Unix() >= expiry {
		return Session{}, false
	}
	return Session{Subject: parts[0], ExpiresAt: time.Unix(expiry, 0)}, true
}

// Middleware rejects requests without a valid session and delegates to the
// Verifier before calling next.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := h.SessionFromRequest(r)
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if _, err := h.verifier.Verify(r.Context(), session.Subject); err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
