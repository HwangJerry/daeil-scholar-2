package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

var (
	ErrSocialLinkTokenInvalid    = errors.New("social link token is invalid or expired")
	ErrSocialLinkTokenInProgress = errors.New("social link token is already being processed")
	ErrSocialLinkTokenConsumed   = errors.New("social link token was already consumed")
	ErrSocialLinkLeaseInvalid    = errors.New("social link token lease is invalid")
)

const SocialLinkTokenTTL = 5 * time.Minute

type socialLinkTokenStatus string

const (
	socialLinkTokenReady      socialLinkTokenStatus = "ready"
	socialLinkTokenProcessing socialLinkTokenStatus = "processing"
	socialLinkTokenConsumed   socialLinkTokenStatus = "consumed"
)

type socialLinkTokenEntry struct {
	Data    model.SocialLinkData
	Status  socialLinkTokenStatus
	LeaseID string
}

type SocialLinkTokenSnapshot struct {
	Data      model.SocialLinkData
	ExpiresAt time.Time
}

type SocialLinkTokenLease struct {
	Token     string
	ID        string
	Data      model.SocialLinkData
	ExpiresAt time.Time
}

// SocialLinkTokenStore makes the in-memory link-token lifecycle atomic inside
// one backend process. Validation/reauthentication failures release a lease,
// while a committed account link permanently moves the token to consumed.
type SocialLinkTokenStore struct {
	cache         *cache.Cache
	mu            sync.Mutex
	leaseSequence uint64
}

func NewSocialLinkTokenStore(cacheStore *cache.Cache) *SocialLinkTokenStore {
	return &SocialLinkTokenStore{cache: cacheStore}
}

func (s *SocialLinkTokenStore) Put(token string, data model.SocialLinkData, ttl time.Duration) (time.Time, error) {
	if token == "" || ttl <= 0 {
		return time.Time{}, ErrSocialLinkTokenInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	expiresAt := time.Now().Add(ttl)
	s.cache.Set(token, socialLinkTokenEntry{
		Data:   data,
		Status: socialLinkTokenReady,
	}, ttl)
	return expiresAt, nil
}

func (s *SocialLinkTokenStore) Snapshot(token string) (SocialLinkTokenSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, expiresAt, err := s.entry(token)
	if err != nil {
		return SocialLinkTokenSnapshot{}, err
	}
	if entry.Status == socialLinkTokenConsumed {
		return SocialLinkTokenSnapshot{}, ErrSocialLinkTokenConsumed
	}
	return SocialLinkTokenSnapshot{Data: entry.Data, ExpiresAt: expiresAt}, nil
}

func (s *SocialLinkTokenStore) Update(
	token string,
	update func(model.SocialLinkData) model.SocialLinkData,
) (SocialLinkTokenSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, expiresAt, err := s.entry(token)
	if err != nil {
		return SocialLinkTokenSnapshot{}, err
	}
	if entry.Status == socialLinkTokenProcessing {
		return SocialLinkTokenSnapshot{}, ErrSocialLinkTokenInProgress
	}
	if entry.Status == socialLinkTokenConsumed {
		return SocialLinkTokenSnapshot{}, ErrSocialLinkTokenConsumed
	}
	entry.Data = update(entry.Data)
	s.cache.Set(token, entry, time.Until(expiresAt))
	return SocialLinkTokenSnapshot{Data: entry.Data, ExpiresAt: expiresAt}, nil
}

func (s *SocialLinkTokenStore) Begin(token string) (SocialLinkTokenLease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, expiresAt, err := s.entry(token)
	if err != nil {
		return SocialLinkTokenLease{}, err
	}
	switch entry.Status {
	case socialLinkTokenProcessing:
		return SocialLinkTokenLease{}, ErrSocialLinkTokenInProgress
	case socialLinkTokenConsumed:
		return SocialLinkTokenLease{}, ErrSocialLinkTokenConsumed
	}

	s.leaseSequence++
	entry.Status = socialLinkTokenProcessing
	entry.LeaseID = fmt.Sprintf("lease-%d", s.leaseSequence)
	s.cache.Set(token, entry, time.Until(expiresAt))
	return SocialLinkTokenLease{
		Token:     token,
		ID:        entry.LeaseID,
		Data:      entry.Data,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *SocialLinkTokenStore) Release(lease SocialLinkTokenLease) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, expiresAt, err := s.entry(lease.Token)
	if err != nil {
		return err
	}
	if entry.Status != socialLinkTokenProcessing || entry.LeaseID != lease.ID {
		return ErrSocialLinkLeaseInvalid
	}
	entry.Status = socialLinkTokenReady
	entry.LeaseID = ""
	s.cache.Set(lease.Token, entry, time.Until(expiresAt))
	return nil
}

func (s *SocialLinkTokenStore) Consume(lease SocialLinkTokenLease) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, expiresAt, err := s.entry(lease.Token)
	if err != nil {
		return err
	}
	if entry.Status != socialLinkTokenProcessing || entry.LeaseID != lease.ID {
		return ErrSocialLinkLeaseInvalid
	}
	entry.Data = model.SocialLinkData{}
	entry.Status = socialLinkTokenConsumed
	entry.LeaseID = ""
	s.cache.Set(lease.Token, entry, time.Until(expiresAt))
	return nil
}

func (s *SocialLinkTokenStore) entry(token string) (socialLinkTokenEntry, time.Time, error) {
	value, expiresAt, found := s.cache.GetWithExpiration(token)
	if !found || (!expiresAt.IsZero() && !expiresAt.After(time.Now())) {
		s.cache.Delete(token)
		return socialLinkTokenEntry{}, time.Time{}, ErrSocialLinkTokenInvalid
	}
	entry, ok := value.(socialLinkTokenEntry)
	if !ok {
		return socialLinkTokenEntry{}, time.Time{}, ErrSocialLinkTokenInvalid
	}
	return entry, expiresAt, nil
}
