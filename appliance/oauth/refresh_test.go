package oauth

import (
	"context"
	"testing"
	"time"
)

var refreshNow = time.Date(2026, 11, 26, 11, 0, 0, 0, time.UTC)

func TestRotateRefreshRejectsReplayAndRevokesFamily(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	first, err := IssueRefresh(ctx, st, "user-1", func() time.Time { return refreshNow }, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefresh: %v", err)
	}
	second, err := RotateRefresh(ctx, first, st, func() time.Time { return refreshNow.Add(time.Minute) }, time.Hour)
	if err != nil {
		t.Fatalf("RotateRefresh: %v", err)
	}
	if second == first {
		t.Fatal("rotated token equals the old one")
	}

	// Replaying the consumed token must fail and revoke the whole family.
	if _, err := RotateRefresh(ctx, first, st, func() time.Time { return refreshNow.Add(2 * time.Minute) }, time.Hour); err == nil {
		t.Fatal("replayed refresh token was accepted")
	}
	if _, err := RotateRefresh(ctx, second, st, func() time.Time { return refreshNow.Add(3 * time.Minute) }, time.Hour); err == nil {
		t.Fatal("family was not revoked after replay")
	}
}

func TestRotateRefreshRejectsUnknownAndExpired(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	now := func() time.Time { return refreshNow }
	if _, err := RotateRefresh(ctx, "unknown", st, now, time.Hour); err == nil {
		t.Error("unknown refresh token was accepted")
	}
	token, err := IssueRefresh(ctx, st, "u", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RotateRefresh(ctx, token, st, func() time.Time { return refreshNow.Add(2 * time.Minute) }, time.Minute); err == nil {
		t.Error("expired refresh token was accepted")
	}
}

func TestRefreshTokensAreUnique(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	now := func() time.Time { return refreshNow }
	a, _ := IssueRefresh(ctx, st, "u", now, time.Hour)
	b, _ := IssueRefresh(ctx, st, "u", now, time.Hour)
	if a == b {
		t.Error("two refresh tokens are identical")
	}
}
