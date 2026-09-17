// app_client_build_service.go — Records build numbers observed from mobile requests.
package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// appClientBuildObserveTTL throttles writes so a busy build touches the database
// at most once per window instead of once per request.
const appClientBuildObserveTTL = 10 * time.Minute

// maxAppClientVersionNameRunes matches the VERSION_NAME column width, so a
// spoofed or oversized header never fails the insert.
const maxAppClientVersionNameRunes = 64

type AppClientBuildStore interface {
	Upsert(platform string, build int64, versionName string, seenAt time.Time) error
	List(platform string) ([]model.AppClientBuild, error)
	LatestBuild(platform string) (int64, error)
}

type AppClientBuildService struct {
	store  AppClientBuildStore
	cache  *cache.Cache
	logger zerolog.Logger
	now    func() time.Time
}

func NewAppClientBuildService(store AppClientBuildStore, cacheStore *cache.Cache, logger zerolog.Logger) *AppClientBuildService {
	return &AppClientBuildService{store: store, cache: cacheStore, logger: logger, now: time.Now}
}

// Observe records that a platform/build pair is in use. It never fails a request:
// observation is bookkeeping for the admin build picker, not request handling.
func (s *AppClientBuildService) Observe(platform string, build int64, versionName string) {
	if !model.IsSupportedAppPlatform(platform) || build <= 0 {
		return
	}
	versionName = truncateVersionName(versionName)
	throttleKey := appClientBuildThrottleKey(platform, build)
	if s.cache != nil {
		if _, throttled := s.cache.Get(throttleKey); throttled {
			return
		}
	}
	if err := s.store.Upsert(platform, build, versionName, s.now()); err != nil {
		// The throttle is set only after a successful write, so a transient
		// database failure does not hide a new build for the whole window.
		s.logger.Warn().Err(err).
			Str("platform", platform).
			Int64("build", build).
			Msg("failed to record observed app build")
		return
	}
	if s.cache != nil {
		s.cache.Set(throttleKey, struct{}{}, appClientBuildObserveTTL)
	}
}

// List returns the builds seen for a platform, newest first.
func (s *AppClientBuildService) List(platform string) ([]model.AppClientBuild, error) {
	if !model.IsSupportedAppPlatform(platform) {
		return nil, ErrUnsupportedAppUpdatePlatform
	}
	return s.store.List(platform)
}

// LatestBuild reports the highest build seen for a platform, or 0 when none.
func (s *AppClientBuildService) LatestBuild(platform string) (int64, error) {
	if !model.IsSupportedAppPlatform(platform) {
		return 0, ErrUnsupportedAppUpdatePlatform
	}
	return s.store.LatestBuild(platform)
}

func truncateVersionName(versionName string) string {
	trimmed := strings.TrimSpace(versionName)
	runes := []rune(trimmed)
	if len(runes) > maxAppClientVersionNameRunes {
		return string(runes[:maxAppClientVersionNameRunes])
	}
	return trimmed
}

func appClientBuildThrottleKey(platform string, build int64) string {
	return "app_client_build:" + platform + ":" + strconv.FormatInt(build, 10)
}
