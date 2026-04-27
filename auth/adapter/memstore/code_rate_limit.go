// Package memstore is an in-memory implementation of port.CodeRateLimitStore
// and port.ExchangeCodeStore.
//
// Use it for tests, single-instance dev environments, or as a fallback when
// Redis is unavailable. NOT suitable for multi-instance production deployments.
package memstore

import (
	"context"
	"sync"
	"time"

	"github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/port"
)

type codeEntry struct {
	code      string
	expiresAt time.Time
	lastSent  time.Time
	attempts  int
}

// CodeStore is an in-memory CodeRateLimitStore.
type CodeStore struct {
	mu    sync.Mutex
	codes map[string]*codeEntry // key: email
	daily map[string]int        // key: email|day_bucket
}

func NewCodeStore() *CodeStore {
	return &CodeStore{
		codes: make(map[string]*codeEntry),
		daily: make(map[string]int),
	}
}

func (s *CodeStore) SaveCode(_ context.Context, email, code string, expiresAt, lastSentAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[email] = &codeEntry{
		code:      code,
		expiresAt: expiresAt,
		lastSent:  lastSentAt,
	}
	return nil
}

func (s *CodeStore) LoadCode(_ context.Context, email string) (string, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.codes[email]
	if !ok {
		return "", time.Time{}, nil
	}
	if time.Now().After(e.expiresAt) {
		delete(s.codes, email)
		return "", time.Time{}, nil
	}
	return e.code, e.lastSent, nil
}

func (s *CodeStore) DeleteCode(_ context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.codes, email)
	return nil
}

func (s *CodeStore) IncrAttempts(_ context.Context, email string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.codes[email]
	if !ok {
		return 0, nil
	}
	e.attempts++
	return e.attempts, nil
}

func (s *CodeStore) IncrDailyCount(_ context.Context, email, dayBucket string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := email + "|" + dayBucket
	s.daily[key]++
	return s.daily[key], nil
}

var _ port.CodeRateLimitStore = (*CodeStore)(nil)

// ExchangeStore is an in-memory ExchangeCodeStore.
type ExchangeStore struct {
	mu    sync.Mutex
	codes map[string]domain.ExchangeCode
}

func NewExchangeStore() *ExchangeStore {
	return &ExchangeStore{codes: make(map[string]domain.ExchangeCode)}
}

func (s *ExchangeStore) Save(_ context.Context, code domain.ExchangeCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code.Code] = code
	return nil
}

// Consume returns the code and atomically removes it.
func (s *ExchangeStore) Consume(_ context.Context, code string) (*domain.ExchangeCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return nil, domain.ErrInvalidExchange
	}
	delete(s.codes, code)
	if time.Now().After(c.ExpiresAt) {
		return nil, domain.ErrInvalidExchange
	}
	return &c, nil
}

var _ port.ExchangeCodeStore = (*ExchangeStore)(nil)
