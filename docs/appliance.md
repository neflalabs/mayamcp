# Appliance Module (opt-in)

MayaMCP core stays small: it has no auth, terminal UI, web UI, or config store.
Those belong to the **appliance ring** — an application that imports the core.
`appliance/` is a separate Go module in this repository for the parts that are
genuinely hard to write correctly and safe to reuse.

Nothing here is imported by `api`, `internal/`, or `cmd/mayamcp`. If you only
want tools over stdio, you never see this module.

## Install

```sh
go get github.com/neflalabs/mayamcp/appliance/oauth@appliance/v0.1.0
```

The `appliance/v0.x.y` tag versions this module independently from the core
`v0.x.y` tag. Both can evolve at their own pace.

## What is in it today

| Package | Purpose | Depends on |
|---|---|---|
| `appliance/oauth` | State/PKCE, signed cookies, ES256 JWT/JWKS, host guard, handler, Store, refresh rotation | Standard library only |
| `appliance/oauth/server` | Discovery endpoints, JWKS, Bearer-token MCP guard | `appliance/oauth` |

### OAuth primitives

```go
import "github.com/neflalabs/mayamcp/appliance/oauth"

state, _ := oauth.NewState(nil)          // CSRF state with issue time
verifier, challenge, _ := oauth.NewPKCE() // RFC 7636 S256
key, _ := oauth.GenerateSigningKey()      // ES256 / P-256
```

- State is single-use via a `Store`; `VerifyState` enforces TTL in constant time.
- Cookies are HMAC-SHA256 signed and bound to the cookie name.
- JWTs are `at+jwt` / ES256 with issuer and audience checks.
- Refresh rotation revokes the whole token family on replay.
- Auth log records use the `auth: ` prefix — never `tool: `.

### Handler and server glue

```go
handler, _ := oauth.New(oauth.Config{
	ClientID:    "client-1",
	AuthURL:     "https://provider.example/authorize",
	RedirectURL: "https://app.example/oauth/callback",
	Cookie: oauth.CookieOptions{
		Name:   "app_session",
		Key:    keyFromConfig,      // supplied by the application
		Secure: true,
	},
	Exchange: exchangeWithProvider, // provider-specific, supplied by you
}, verifier, store, logger)

srv, _ := server.New(server.Config{
	Issuer:  "https://mcp.example",
	Key:     signingKey,
	Handler: handler,
})

mux := http.NewServeMux()
srv.RegisterRoutes(mux)
mux.Handle("/mcp", srv.MCP(mcpHandler))
```

The MCP guard accepts **Bearer tokens only**. A browser session cookie is
never accepted against `/mcp`, so a leaked cookie cannot be replayed as a
machine credential.

## What is deliberately NOT here

- Provider-specific client IDs, secrets, scopes, or consent screens.
- A default signing key, issuer, redirect URI, or storage path.
- Any change to the tool contract, golden file, token budget, or `ActivityPrefix`.
- Web UI and TUI packages — extracted only when a second consumer needs them.

## Development

```sh
make appliance-test   # tests for the appliance module
make check            # core unchanged
```

Tag `appliance/v0.x.y` to publish. `.github/workflows/release-appliance.yml`
runs the core CI, verifies the module, packages the source, and publishes a
separate GitHub Release.

See [`GLOSSARY.md`](GLOSSARY.md) for terms such as *appliance ring*, *PKCE*,
*state*, and *session cookie*.
