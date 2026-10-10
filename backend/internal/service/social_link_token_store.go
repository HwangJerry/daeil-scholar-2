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
	ErrSocialLinkTokenCancelled  = errors.New("social link token was cancelled")
	ErrSocialLinkLeaseInvalid    = errors.New("social link token lease is invalid")
)

const SocialLinkTokenTTL = 5 * time.Minute

type socialLinkTokenStatus string

const (
	socialLinkTokenReady      socialLinkTokenStatus = "ready"
	socialLinkTokenProcessing socialLinkTokenStatus = "processing"
	socialLinkTokenConsumed   socialLinkTokenStatus = "consumed"
	socialLinkTokenCancelled  socialLinkTokenStatus = "cancelled"
)

type socialLinkTokenEntry struct {
	Data      model.SocialLinkData
	Status    socialLinkTokenStatus
	LeaseID   string
	ExpiresAt time.Time
	Uploads   *socialLinkUploads
}
type socialLinkUploads struct {
	Results []UploadResult
	Timer   *time.Timer
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

// Begin/Cancel/Consume serialize within one backend process. Processing entries
// remain until Release/Consume even if TTL expires: expiry may not delete a photo
// while the member transaction is committing. The lease cannot be acquired again.
type SocialLinkTokenStore struct {
	cache         *cache.Cache
	mu            sync.Mutex
	leaseSequence uint64
	discardUpload func(*UploadResult) error
}

func NewSocialLinkTokenStore(c *cache.Cache) *SocialLinkTokenStore {
	return &SocialLinkTokenStore{cache: c}
}
func (s *SocialLinkTokenStore) SetUploadDiscarder(discard func(*UploadResult) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discardUpload = discard
}
func (s *SocialLinkTokenStore) Put(token string, data model.SocialLinkData, ttl time.Duration) (time.Time, error) {
	if token == "" || ttl <= 0 {
		return time.Time{}, ErrSocialLinkTokenInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expires := time.Now().Add(ttl)
	uploads := &socialLinkUploads{}
	// Only upload metadata is captured by the timer, never provider credentials.
	uploads.Timer = time.AfterFunc(ttl, func() { s.expire(token, uploads) })
	s.cache.Set(token, socialLinkTokenEntry{Data: data, Status: socialLinkTokenReady, ExpiresAt: expires, Uploads: uploads}, ttl)
	return expires, nil
}
func (s *SocialLinkTokenStore) Snapshot(token string) (SocialLinkTokenSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, err := s.entry(token)
	if err != nil {
		return SocialLinkTokenSnapshot{}, err
	}
	if err = available(entry); err != nil {
		return SocialLinkTokenSnapshot{}, err
	}
	return SocialLinkTokenSnapshot{entry.Data, entry.ExpiresAt}, nil
}
func available(entry socialLinkTokenEntry) error {
	switch entry.Status {
	case socialLinkTokenProcessing:
		return ErrSocialLinkTokenInProgress
	case socialLinkTokenConsumed:
		return ErrSocialLinkTokenConsumed
	case socialLinkTokenCancelled:
		return ErrSocialLinkTokenCancelled
	}
	return nil
}
func (s *SocialLinkTokenStore) Update(token string, update func(model.SocialLinkData) model.SocialLinkData) (SocialLinkTokenSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, err := s.entry(token)
	if err != nil {
		return SocialLinkTokenSnapshot{}, err
	}
	if err = available(entry); err != nil {
		return SocialLinkTokenSnapshot{}, err
	}
	entry.Data = update(entry.Data)
	s.save(token, entry)
	return SocialLinkTokenSnapshot{entry.Data, entry.ExpiresAt}, nil
}

// AttachUpload atomically installs and tracks an actual result. Caller discards
// a freshly uploaded result if cancellation/expiry/Begin won the race.
func (s *SocialLinkTokenStore) AttachUpload(token string, result *UploadResult) error {
	if result == nil || result.FSeq <= 0 || result.URL == "" {
		return ErrSocialLinkTokenInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, err := s.entry(token)
	if err != nil {
		return err
	}
	if err = available(entry); err != nil {
		return err
	}
	entry.Uploads.Results = append(entry.Uploads.Results, *result)
	entry.Data.ProfileImageURL = result.URL
	s.save(token, entry)
	return nil
}
func (s *SocialLinkTokenStore) Begin(token string) (SocialLinkTokenLease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, err := s.entry(token)
	if err != nil {
		return SocialLinkTokenLease{}, err
	}
	if err = available(entry); err != nil {
		return SocialLinkTokenLease{}, err
	}
	leaseData := entry.Data
	entry.Data = model.SocialLinkData{}
	s.leaseSequence++
	entry.Status = socialLinkTokenProcessing
	entry.LeaseID = fmt.Sprintf("lease-%d", s.leaseSequence)
	// Keep a processing lease past TTL solely to finalize commit/rollback safely.
	s.cache.Set(token, entry, cache.NoExpiration)
	return SocialLinkTokenLease{token, entry.LeaseID, leaseData, entry.ExpiresAt}, nil
}
func (s *SocialLinkTokenStore) Release(lease SocialLinkTokenLease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, err := s.leasedEntry(lease)
	if err != nil {
		return err
	}
	if !entry.ExpiresAt.After(time.Now()) {
		s.cache.Delete(lease.Token)
		s.cleanup(entry.Uploads, "")
		return ErrSocialLinkTokenInvalid
	}
	entry.Data = lease.Data
	entry.Status = socialLinkTokenReady
	entry.LeaseID = ""
	s.save(lease.Token, entry)
	return nil
}
func (s *SocialLinkTokenStore) Consume(lease SocialLinkTokenLease) error {
	return s.ConsumeWithPhoto(lease, lease.Data.ProfileImageURL)
}

// ConsumeWithPhoto is called immediately after member commit. Keep only the
// selected committed photo; orphaned previous uploads can be safely discarded.
func (s *SocialLinkTokenStore) ConsumeWithPhoto(lease SocialLinkTokenLease, committedPhoto string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, err := s.leasedEntry(lease)
	if err != nil {
		return err
	}
	s.cleanup(entry.Uploads, committedPhoto)
	entry.Data = model.SocialLinkData{}
	entry.Status = socialLinkTokenConsumed
	entry.LeaseID = ""
	entry.Uploads = nil
	if !entry.ExpiresAt.After(time.Now()) {
		s.cache.Delete(lease.Token)
	} else {
		s.save(lease.Token, entry)
	}
	return nil
}
func (s *SocialLinkTokenStore) Cancel(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found := s.cache.Get(token)
	entry, ok := value.(socialLinkTokenEntry)
	if !found || !ok {
		return nil
	}
	var err error
	if entry.Status != socialLinkTokenProcessing && !entry.ExpiresAt.After(time.Now()) {
		err = ErrSocialLinkTokenInvalid
	}
	if err != nil {
		return nil
	}
	switch entry.Status {
	case socialLinkTokenProcessing:
		return ErrSocialLinkTokenInProgress
	case socialLinkTokenConsumed:
		return ErrSocialLinkTokenConsumed
	case socialLinkTokenCancelled:
		return nil
	}
	s.cleanup(entry.Uploads, "")
	entry.Data = model.SocialLinkData{}
	entry.Status = socialLinkTokenCancelled
	entry.LeaseID = ""
	entry.Uploads = nil
	s.save(token, entry)
	return nil
}
func (s *SocialLinkTokenStore) save(token string, entry socialLinkTokenEntry) {
	remaining := time.Until(entry.ExpiresAt)
	if remaining <= 0 {
		s.cache.Delete(token)
		return
	}
	s.cache.Set(token, entry, remaining)
}
func (s *SocialLinkTokenStore) entry(token string) (socialLinkTokenEntry, error) {
	value, found := s.cache.Get(token)
	if !found {
		return socialLinkTokenEntry{}, ErrSocialLinkTokenInvalid
	}
	entry, ok := value.(socialLinkTokenEntry)
	if !ok {
		return socialLinkTokenEntry{}, ErrSocialLinkTokenInvalid
	}
	if !entry.ExpiresAt.After(time.Now()) {
		return socialLinkTokenEntry{}, ErrSocialLinkTokenInvalid
	}
	return entry, nil
}
func (s *SocialLinkTokenStore) leasedEntry(lease SocialLinkTokenLease) (socialLinkTokenEntry, error) {
	value, found := s.cache.Get(lease.Token)
	if !found {
		return socialLinkTokenEntry{}, ErrSocialLinkTokenInvalid
	}
	entry, ok := value.(socialLinkTokenEntry)
	if !ok || entry.Status != socialLinkTokenProcessing || entry.LeaseID != lease.ID {
		return socialLinkTokenEntry{}, ErrSocialLinkLeaseInvalid
	}
	return entry, nil
}
func (s *SocialLinkTokenStore) expire(token string, uploads *socialLinkUploads) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value, found := s.cache.Get(token); found {
		if entry, ok := value.(socialLinkTokenEntry); ok {
			if entry.Uploads != uploads {
				return
			}
			if entry.Status == socialLinkTokenProcessing {
				return
			}
			s.cache.Delete(token)
		}
	}
	s.cleanup(uploads, "")
}

// cleanup runs under mu, removes metadata immediately, and performs IO outside
// the lock. The discarder must check persistent ownership before each retry.
func (s *SocialLinkTokenStore) cleanup(uploads *socialLinkUploads, retainedURL string) {
	if uploads == nil {
		return
	}
	if uploads.Timer != nil {
		uploads.Timer.Stop()
	}
	results := uploads.Results
	uploads.Results = nil
	discard := s.discardUpload
	if discard == nil {
		return
	}
	for _, result := range results {
		if result.URL == retainedURL {
			continue
		}
		go func(result UploadResult) {
			for _, delay := range []time.Duration{0, time.Second, 5 * time.Second, 30 * time.Second} {
				if delay > 0 {
					time.Sleep(delay)
				}
				if discard(&result) == nil {
					return
				}
			}
		}(result)
	}
}
