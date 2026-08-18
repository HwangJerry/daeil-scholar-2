package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

func TestSocialLinkTokenStoreRetryAndConsumeLifecycle(t *testing.T) {
	store := NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := store.Put("token", model.SocialLinkData{SocialID: "subject"}, time.Minute); err != nil {
		t.Fatal(err)
	}

	firstLease, err := store.Begin("token")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Release(firstLease); err != nil {
		t.Fatal(err)
	}

	retryLease, err := store.Begin("token")
	if err != nil {
		t.Fatalf("retry after a retryable failure: %v", err)
	}
	if retryLease.Data.SocialID != "subject" {
		t.Fatalf("retry data = %#v", retryLease.Data)
	}
	if err := store.Consume(retryLease); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin("token"); !errors.Is(err, ErrSocialLinkTokenConsumed) {
		t.Fatalf("replay error = %v, want consumed", err)
	}
}

func TestSocialLinkTokenStoreAllowsOnlyOneConcurrentLease(t *testing.T) {
	store := NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := store.Put("token", model.SocialLinkData{}, time.Minute); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, err := store.Begin("token")
			results <- err
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var leased int
	var inProgress int
	for err := range results {
		switch {
		case err == nil:
			leased++
		case errors.Is(err, ErrSocialLinkTokenInProgress):
			inProgress++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if leased != 1 || inProgress != 1 {
		t.Fatalf("leased=%d inProgress=%d", leased, inProgress)
	}
}

func TestSocialLinkTokenStoreRejectsExpiredToken(t *testing.T) {
	store := NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := store.Put("token", model.SocialLinkData{}, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := store.Begin("token"); !errors.Is(err, ErrSocialLinkTokenInvalid) {
		t.Fatalf("expired token error = %v", err)
	}
}
