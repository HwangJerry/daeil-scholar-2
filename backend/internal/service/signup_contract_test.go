package service

import (
	"encoding/json"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
	"os"
	"testing"
	"time"
)

func TestSignupWeakPasswordRejectedBeforeDependencies(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("weak password reached repositories/hash: %v", p)
		}
	}()
	_, err := (&RegistrationService{}).Register(model.RegisterRequest{Password: "x", Phone: "01012345678"})
	if err == nil {
		t.Fatal("weak password accepted")
	}
}

func TestSocialSignupCancelContract(t *testing.T) {
	raw := cache.New(time.Minute, time.Minute)
	store := NewSocialLinkTokenStore(raw)
	cancel, ok := any(store).(interface{ Cancel(string) error })
	if !ok {
		t.Fatal("missing atomic Cancel")
	}
	expires, _ := store.Put("cancel", model.SocialLinkData{Provider: "KT", SocialID: "subject", AccessToken: "synthetic-secret", Email: "synthetic@example.test"}, time.Minute)
	if err := cancel.Cancel("cancel"); err != nil {
		t.Fatal(err)
	}
	entry, after, found := raw.GetWithExpiration("cancel")
	if !found || after.After(expires.Add(time.Millisecond)) {
		t.Fatal("cancel removed tombstone or extended TTL")
	}
	if data := entry.(socialLinkTokenEntry).Data; data != (model.SocialLinkData{}) {
		t.Fatal("cancel retains provider data")
	}
	if _, err := store.Begin("cancel"); err == nil {
		t.Fatal("cancelled token can begin signup")
	}
	if _, err := store.Snapshot("cancel"); err == nil {
		t.Fatal("cancelled token exposes prefill")
	}
	if _, err := store.Update("cancel", func(data model.SocialLinkData) model.SocialLinkData { return data }); err == nil {
		t.Fatal("cancelled token accepts photo")
	}
	for _, token := range []string{"cancel", "missing", ""} {
		if err := cancel.Cancel(token); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = store.Put("processing", model.SocialLinkData{}, time.Minute)
	lease, _ := store.Begin("processing")
	if err := cancel.Cancel("processing"); !errors.Is(err, ErrSocialLinkTokenInProgress) {
		t.Fatalf("processing cancel: %v", err)
	}
	if err := store.Consume(lease); err != nil {
		t.Fatal(err)
	}
	if err := cancel.Cancel("processing"); !errors.Is(err, ErrSocialLinkTokenConsumed) {
		t.Fatalf("consumed cancel: %v", err)
	}
}

func TestSocialCancelAndBeginHaveOneWinner(t *testing.T) {
	for range 100 {
		store := NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
		_, _ = store.Put("race", model.SocialLinkData{AccessToken: "synthetic"}, time.Minute)
		start := make(chan struct{})
		began := make(chan error, 1)
		cancelled := make(chan error, 1)
		go func() { <-start; _, err := store.Begin("race"); began <- err }()
		go func() { <-start; cancelled <- store.Cancel("race") }()
		close(start)
		b, c := <-began, <-cancelled
		if c == nil && b == nil {
			t.Fatal("cancel success and member creation both allowed")
		}
		if b == nil && !errors.Is(c, ErrSocialLinkTokenInProgress) {
			t.Fatalf("begin won but cancel=%v", c)
		}
		if c == nil && !errors.Is(b, ErrSocialLinkTokenCancelled) {
			t.Fatalf("cancel won but begin=%v", b)
		}
	}
}

func TestSocialPhotoCleanupExpiryAndCommit(t *testing.T) {
	for _, which := range []string{"cancel", "expiry", "commit", "expired-commit", "expired-rollback"} {
		t.Run(which, func(t *testing.T) {
			store := NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
			discarded := make(chan int, 4)
			store.SetUploadDiscarder(func(result *UploadResult) error { discarded <- result.FSeq; return nil })
			ttl := time.Minute
			if which == "expiry" || which == "expired-commit" || which == "expired-rollback" {
				ttl = 35 * time.Millisecond
			}
			_, _ = store.Put("photo", model.SocialLinkData{ProfileImageURL: "https://provider.example/photo"}, ttl)
			if err := store.AttachUpload("photo", &UploadResult{URL: "/uploads/profile/one.jpg", FSeq: 1}); err != nil {
				t.Fatal(err)
			}
			if err := store.AttachUpload("photo", &UploadResult{URL: "/uploads/profile/two.jpg", FSeq: 2}); err != nil {
				t.Fatal(err)
			}
			switch which {
			case "cancel":
				if err := store.Cancel("photo"); err != nil {
					t.Fatal(err)
				}
			case "commit":
				lease, _ := store.Begin("photo")
				if err := store.ConsumeWithPhoto(lease, "/uploads/profile/two.jpg"); err != nil {
					t.Fatal(err)
				}
			case "expired-commit", "expired-rollback":
				lease, _ := store.Begin("photo")
				time.Sleep(60 * time.Millisecond)
				select {
				case id := <-discarded:
					t.Fatalf("processing lease deleted upload %d before result", id)
				default:
				}
				if which == "expired-commit" {
					if err := store.ConsumeWithPhoto(lease, "/uploads/profile/two.jpg"); err != nil {
						t.Fatal(err)
					}
				} else {
					_ = store.Release(lease)
				}
			}
			want := 2
			if which == "commit" || which == "expired-commit" {
				want = 1
			}
			got := map[int]bool{}
			for range want {
				select {
				case id := <-discarded:
					got[id] = true
				case <-time.After(time.Second):
					t.Fatal("orphan upload cleanup did not run")
				}
			}
			if !got[1] || (want == 1 && got[2]) {
				t.Fatalf("discarded=%v", got)
			}
			select {
			case id := <-discarded:
				t.Fatalf("unexpected discard %d", id)
			case <-time.After(10 * time.Millisecond):
			}
		})
	}
}

func TestSocialProcessingCredentialsOnlyInLease(t *testing.T) {
	raw := cache.New(time.Minute, time.Minute)
	store := NewSocialLinkTokenStore(raw)
	_, _ = store.Put("credential", model.SocialLinkData{AccessToken: "synthetic", Email: "synthetic@example.test"}, 20*time.Millisecond)
	lease, err := store.Begin("credential")
	if err != nil {
		t.Fatal(err)
	}
	if lease.Data.AccessToken != "synthetic" {
		t.Fatal("lease lost provider credential")
	}
	time.Sleep(40 * time.Millisecond)
	value, found := raw.Get("credential")
	if !found {
		t.Fatal("finalization metadata lost")
	}
	if value.(socialLinkTokenEntry).Data != (model.SocialLinkData{}) {
		t.Fatal("expired processing metadata retains provider data")
	}
	if err := store.Cancel("credential"); !errors.Is(err, ErrSocialLinkTokenInProgress) {
		t.Fatalf("expired in-flight cancel=%v", err)
	}
	if _, err := store.Begin("credential"); !errors.Is(err, ErrSocialLinkTokenInvalid) {
		t.Fatalf("expired processing begin=%v", err)
	}
	if err := store.Consume(lease); err != nil {
		t.Fatal(err)
	}
	if _, found := raw.Get("credential"); found {
		t.Fatal("expired committed metadata retained")
	}
}

func TestNewPasswordSharedGolden(t *testing.T) {
	data, err := os.ReadFile("testdata/password-policy-golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Name     string
			Password string
			Valid    bool
		}
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.Cases {
		t.Run(c.Name, func(t *testing.T) {
			if got := ValidateNewPassword(c.Password) == nil; got != c.Valid {
				t.Fatalf("valid=%v want %v", got, c.Valid)
			}
		})
	}
}
