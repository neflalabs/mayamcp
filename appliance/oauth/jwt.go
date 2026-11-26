package oauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	tokenTTL       = 10 * time.Minute
	tokenType      = "at+jwt"
	tokenAlg       = "ES256"
	signingKeySize = 16
)

var (
	errInvalidToken = errors.New("oauth: invalid access token")
	errExpiredToken = errors.New("oauth: expired access token")
)

// SigningKey owns an ES256 (P-256) keypair and its stable key id.
type SigningKey struct {
	private *ecdsa.PrivateKey
	kid     string
}

// TokenClaims is the JWT payload. Now is a test hook and is never serialized.
type TokenClaims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	Scope     string `json:"scope"`
	ID        string `json:"jti"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`

	Now func() time.Time `json:"-"`
}

// GenerateSigningKey creates a new P-256 keypair.
func GenerateSigningKey() (*SigningKey, error) {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("oauth: generate signing key: %w", err)
	}
	return &SigningKey{private: private, kid: keyID(private)}, nil
}

// ParseSigningKey loads a PKCS#8 PEM key. An empty string is an error: unlike
// the original authorization server, this module never silently generates a
// key during parsing — the caller decides when to rotate.
func ParseSigningKey(pemBytes string) (*SigningKey, error) {
	block, _ := pem.Decode([]byte(pemBytes))
	if block == nil {
		return nil, errors.New("oauth: signing key is not valid PEM")
	}
	raw, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("oauth: parse signing key: %w", err)
	}
	private, ok := raw.(*ecdsa.PrivateKey)
	if !ok || private.Curve != elliptic.P256() {
		return nil, errors.New("oauth: signing key must be an ECDSA P-256 key")
	}
	return &SigningKey{private: private, kid: keyID(private)}, nil
}

// PEM serializes the private key as PKCS#8 PEM for caller-controlled storage.
func (k *SigningKey) PEM() (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(k.private)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}

// KeyID is a stable identifier derived from the public key.
func (k *SigningKey) KeyID() string { return k.kid }

// Public exposes the verification key.
func (k *SigningKey) Public() *ecdsa.PublicKey { return &k.private.PublicKey }

// JWK returns the public key as a JWK suitable for a /jwks endpoint.
func (k *SigningKey) JWK() map[string]string {
	pub := k.Public()
	raw, err := pub.Bytes()
	if err != nil || len(raw) != 65 || raw[0] != 4 {
		return nil
	}
	return map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(raw[1:33]),
		"y":   base64.RawURLEncoding.EncodeToString(raw[33:]),
		"use": "sig",
		"alg": tokenAlg,
		"kid": k.kid,
	}
}

func keyID(private *ecdsa.PrivateKey) string {
	raw, err := json.Marshal(map[string]string{"kty": "EC", "crv": "P-256"})
	if err != nil || private == nil {
		return "kid-unknown"
	}
	sum := sha256.Sum256(append(raw, private.PublicKey.X.Bytes()...))
	return "kid-" + base64.RawURLEncoding.EncodeToString(sum[:12])
}

// IssueToken signs an ES256 access token. Claims.ID is generated when empty;
// timestamps come from claims.Now (or time.Now) with a 10-minute lifetime.
func (k *SigningKey) IssueToken(claims TokenClaims) (string, error) {
	now := clock(claims.Now)()
	if claims.ID == "" {
		buf := make([]byte, signingKeySize)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("oauth: generate jti: %w", err)
		}
		claims.ID = base64.RawURLEncoding.EncodeToString(buf)
	}
	claims.IssuedAt = now.Unix()
	claims.ExpiresAt = now.Add(tokenTTL).Unix()

	header, err := json.Marshal(map[string]string{"typ": tokenType, "alg": tokenAlg, "kid": k.kid})
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(body)
	sum := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, k.private, sum[:])
	if err != nil {
		return "", err
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return input + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// VerifyToken checks the ES256 signature, expiry, issuer, and audience. The
// audience may be the bare issuer or the issuer + "/mcp" (RFC 8707).
func VerifyToken(token string, pub *ecdsa.PublicKey, issuer string, now func() time.Time) (*TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errInvalidToken
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errInvalidToken
	}
	var header struct {
		Typ string `json:"typ"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Alg != tokenAlg || header.Typ != tokenType {
		return nil, errInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errInvalidToken
	}
	var claims TokenClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, errInvalidToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != 64 {
		return nil, errInvalidToken
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(pub, sum[:],
		new(big.Int).SetBytes(signature[:32]),
		new(big.Int).SetBytes(signature[32:])) {
		return nil, errInvalidToken
	}
	if clock(now)().Unix() >= claims.ExpiresAt {
		return nil, errExpiredToken
	}
	if claims.Issuer != issuer || (claims.Audience != issuer && claims.Audience != issuer+"/mcp") {
		return nil, errInvalidToken
	}
	return &claims, nil
}
