package oauth

import (
	"context"
	"sync"
	"time"
)

// RefreshRecord is the server-side half of a refresh token. A family groups
// every token produced by rotating the same original grant.
type RefreshRecord struct {
	Family    string
	Subject   string
	ExpiresAt time.Time
}

// Store holds single-use state and refresh grants. A caller can implement it
// with a database; MemoryStore is the in-process default for tests and small
// deployments.
type Store interface {
	SaveState(ctx context.Context, key string, value []byte, ttl time.Duration) error
	ConsumeState(ctx context.Context, key string) ([]byte, bool, error)
	SaveRefresh(ctx context.Context, token string, rec RefreshRecord, ttl time.Duration) error
	LoadRefresh(ctx context.Context, token string) (RefreshRecord, bool, error)
	ConsumeRefresh(ctx context.Context, token string) (RefreshRecord, bool, error)
	RevokeFamily(ctx context.Context, family string, ttl time.Duration) error
	IsRevoked(ctx context.Context, family string) bool
}

type stateEntry struct {
	value []byte
	exp   time.Time
}

type refreshEntry struct {
	rec      RefreshRecord
	exp      time.Time
	consumed bool
}

// MemoryStore is a concurrency-safe in-memory Store. It is intentionally
// process-local; multi-instance deployments should supply their own Store.
type MemoryStore struct {
	mu        sync.Mutex
	states    map[string]stateEntry
	refreshes map[string]*refreshEntry
	revoked   map[string]time.Time
	now       func() time.Time
}

// NewMemoryStore creates an empty store using time.Now as its clock.
func NewMemoryStore() *MemoryStore {
	return NewMemoryStoreWithClock(time.Now)
}

// NewMemoryStoreWithClock allows a fixed clock in tests.
func NewMemoryStoreWithClock(now func() time.Time) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	return &MemoryStore{
		states:    make(map[string]stateEntry),
		refreshes: make(map[string]*refreshEntry),
		revoked:   make(map[string]time.Time),
		now:       now,
	}
}

func (s *MemoryStore) SaveState(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[key] = stateEntry{value: append([]byte(nil), value...), exp: s.now().Add(ttl)}
	return nil
}

func (s *MemoryStore) ConsumeState(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.states[key]
	if !ok {
		return nil, false, nil
	}
	delete(s.states, key)
	if s.now().After(entry.exp) {
		return nil, false, nil
	}
	return entry.value, true, nil
}

func (s *MemoryStore) SaveRefresh(_ context.Context, token string, rec RefreshRecord, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshes[token] = &refreshEntry{rec: rec, exp: s.now().Add(ttl)}
	return nil
}

func (s *MemoryStore) LoadRefresh(_ context.Context, token string) (RefreshRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.refreshes[token]
	if !ok || s.now().After(entry.exp) {
		return RefreshRecord{}, false, nil
	}
	return entry.rec, true, nil
}

func (s *MemoryStore) ConsumeRefresh(_ context.Context, token string) (RefreshRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.refreshes[token]
	if !ok || entry.consumed || s.now().After(entry.exp) {
		return RefreshRecord{}, false, nil
	}
	entry.consumed = true
	return entry.rec, true, nil
}

func (s *MemoryStore) RevokeFamily(_ context.Context, family string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[family] = s.now().Add(ttl)
	return nil
}

func (s *MemoryStore) IsRevoked(_ context.Context, family string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.revoked[family]
	return ok && !s.now().After(exp)
}
