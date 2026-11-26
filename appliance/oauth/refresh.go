package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

const (
	refreshTokenBytes = 32
	familyBytes       = 16
)

// IssueRefresh creates a new refresh-token family for subject.
func IssueRefresh(ctx context.Context, st Store, subject string, now func() time.Time, ttl time.Duration) (string, error) {
	family, err := randomToken(familyBytes)
	if err != nil {
		return "", err
	}
	token, err := randomToken(refreshTokenBytes)
	if err != nil {
		return "", err
	}
	rec := RefreshRecord{Family: family, Subject: subject, ExpiresAt: clock(now)().Add(ttl)}
	if err := st.SaveRefresh(ctx, token, rec, ttl); err != nil {
		return "", err
	}
	return token, nil
}

// RotateRefresh consumes current and issues the next token in the same family.
// Replaying an already-consumed token revokes the entire family, matching the
// behaviour of the original authorization server.
func RotateRefresh(ctx context.Context, current string, st Store, now func() time.Time, ttl time.Duration) (string, error) {
	rec, ok, err := st.LoadRefresh(ctx, current)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("oauth: unknown refresh token")
	}
	currentTime := clock(now)()
	if currentTime.After(rec.ExpiresAt) {
		return "", errors.New("oauth: refresh token expired")
	}
	if st.IsRevoked(ctx, rec.Family) {
		return "", errors.New("oauth: refresh token family revoked")
	}
	consumed, ok, err := st.ConsumeRefresh(ctx, current)
	if err != nil {
		return "", err
	}
	if !ok {
		// Use the family from LoadRefresh: a consumed token returns no record
		// from ConsumeRefresh, so its zero value would revoke the wrong family.
		if err := st.RevokeFamily(ctx, rec.Family, ttl); err != nil {
			return "", err
		}
		return "", errors.New("oauth: refresh token replay detected")
	}

	next, err := randomToken(refreshTokenBytes)
	if err != nil {
		return "", err
	}
	nextRec := RefreshRecord{
		Family:    consumed.Family,
		Subject:   consumed.Subject,
		ExpiresAt: currentTime.Add(ttl),
	}
	if err := st.SaveRefresh(ctx, next, nextRec, ttl); err != nil {
		return "", err
	}
	return next, nil
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("oauth: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
