// auth_service.go — AuthService type definition and constructor
package service

import (
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/social-auth/kakao"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// AuthService handles authentication, session management, and Kakao OAuth integration.
type AuthService struct {
	repo        *repository.AuthRepository
	sessionRepo *repository.SessionRepository
	cfg         *config.Config
	cache       *cache.Cache
	kakaoClient *kakao.Client
	httpClient  *http.Client
	logger      zerolog.Logger
}

// NewAuthService creates an AuthService with all required dependencies.
func NewAuthService(
	repo *repository.AuthRepository,
	sessionRepo *repository.SessionRepository,
	cfg *config.Config,
	cacheStore *cache.Cache,
	kakaoClient *kakao.Client,
	logger zerolog.Logger,
) *AuthService {
	client := &http.Client{Timeout: 10 * time.Second}
	if kakaoClient == nil {
		kakaoClient = kakao.NewClient(kakao.Config{
			ClientID:     cfg.Kakao.ClientID,
			ClientSecret: cfg.Kakao.ClientSecret,
			RedirectURI:  cfg.Kakao.RedirectURI,
			HTTPClient:   client,
		})
	}
	return &AuthService{
		repo:        repo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
		cache:       cacheStore,
		kakaoClient: kakaoClient,
		httpClient:  client,
		logger:      logger,
	}
}

// IsMobileSessionActive reports whether a mobile session (sid) is still active,
// i.e. has not been revoked (logout) or expired. Passthrough to the repository
// so callers outside this package (e.g. middleware) don't need direct DB access.
func (s *AuthService) IsMobileSessionActive(usrSeq int, sid string) (bool, error) {
	return s.repo.IsMobileSessionActive(usrSeq, sid)
}
