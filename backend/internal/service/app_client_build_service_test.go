// app_client_build_service_test.go — Observation throttling and input hardening.
package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type appClientBuildStoreStub struct {
	upserts      []model.AppClientBuild
	latest       int64
	listPlatform string
	err          error
}

func (s *appClientBuildStoreStub) Upsert(platform string, build int64, versionName string, seenAt time.Time) error {
	if s.err != nil {
		return s.err
	}
	s.upserts = append(s.upserts, model.AppClientBuild{
		Platform: platform, Build: build, VersionName: versionName, LastSeenAt: seenAt,
	})
	return nil
}

func (s *appClientBuildStoreStub) List(platform string) ([]model.AppClientBuild, error) {
	s.listPlatform = platform
	return s.upserts, nil
}

func (s *appClientBuildStoreStub) LatestBuild(string) (int64, error) {
	return s.latest, nil
}

func newBuildServiceFixture() (*AppClientBuildService, *appClientBuildStoreStub) {
	store := &appClientBuildStoreStub{}
	return NewAppClientBuildService(store, cache.New(time.Minute, time.Minute), zerolog.Nop()), store
}

func TestAppClientBuildServiceRecordsBuildOncePerThrottleWindow(t *testing.T) {
	service, store := newBuildServiceFixture()

	service.Observe(model.AppPlatformIOS, 202609151230, "1.0.0")
	service.Observe(model.AppPlatformIOS, 202609151230, "1.0.0")

	if len(store.upserts) != 1 {
		t.Fatalf("upserts = %d, want 1", len(store.upserts))
	}
	if store.upserts[0].Platform != model.AppPlatformIOS || store.upserts[0].Build != 202609151230 {
		t.Fatalf("upsert = %#v", store.upserts[0])
	}
}

func TestAppClientBuildServiceRecordsEachPlatformSeparately(t *testing.T) {
	service, store := newBuildServiceFixture()

	service.Observe(model.AppPlatformIOS, 10, "1.0.0")
	service.Observe(model.AppPlatformAndroid, 10, "1.0.0")

	if len(store.upserts) != 2 {
		t.Fatalf("upserts = %#v, want one per platform", store.upserts)
	}
}

func TestAppClientBuildServiceIgnoresUnusableObservations(t *testing.T) {
	service, store := newBuildServiceFixture()

	service.Observe("windows", 10, "1.0.0")
	service.Observe(model.AppPlatformIOS, 0, "1.0.0")
	service.Observe(model.AppPlatformIOS, -5, "1.0.0")

	if len(store.upserts) != 0 {
		t.Fatalf("upserts = %#v, want none", store.upserts)
	}
}

// A client controls the version header, so an oversized value must not reach the
// fixed-width column.
func TestAppClientBuildServiceTruncatesVersionName(t *testing.T) {
	service, store := newBuildServiceFixture()

	service.Observe(model.AppPlatformIOS, 10, "  "+strings.Repeat("v", 200)+"  ")

	if got := len([]rune(store.upserts[0].VersionName)); got != maxAppClientVersionNameRunes {
		t.Fatalf("version name runes = %d, want %d", got, maxAppClientVersionNameRunes)
	}
}

// Observation is bookkeeping: a database failure must not surface to the caller,
// because the caller is a request in flight.
func TestAppClientBuildServiceSwallowsStoreFailures(t *testing.T) {
	store := &appClientBuildStoreStub{err: errors.New("db down")}
	service := NewAppClientBuildService(store, cache.New(time.Minute, time.Minute), zerolog.Nop())

	service.Observe(model.AppPlatformIOS, 10, "1.0.0")
}

// A transient write failure must not hide a brand-new build for the whole
// throttle window: during a rollout that build is what the admin needs to pick.
func TestAppClientBuildServiceRetriesAfterFailedUpsert(t *testing.T) {
	store := &appClientBuildStoreStub{err: errors.New("db down")}
	service := NewAppClientBuildService(store, cache.New(time.Minute, time.Minute), zerolog.Nop())

	service.Observe(model.AppPlatformIOS, 10, "1.0.0")
	store.err = nil
	service.Observe(model.AppPlatformIOS, 10, "1.0.0")

	if len(store.upserts) != 1 {
		t.Fatalf("upserts = %#v, want the retry to be written", store.upserts)
	}
}

func TestAppClientBuildServiceRejectsUnknownPlatformOnReads(t *testing.T) {
	service, _ := newBuildServiceFixture()

	if _, err := service.List("windows"); !errors.Is(err, ErrUnsupportedAppUpdatePlatform) {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := service.LatestBuild("windows"); !errors.Is(err, ErrUnsupportedAppUpdatePlatform) {
		t.Fatalf("LatestBuild() error = %v", err)
	}
}
