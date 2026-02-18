// auth_service.go — AuthService type definition and constructor
package service

import (
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// AuthService handles authentication, session management, and Kakao OAuth integration.
type AuthService struct {
	repo        *repository.AuthRepository
	sessionRepo *repository.SessionRepository
	cfg         *config.Config
	cache       *cache.Cache
	httpClient  *http.Client
	logger      zerolog.Logger
}

// NewAuthService creates an AuthService with all required dependencies.
func NewAuthService(
	repo *repository.AuthRepository,
	sessionRepo *repository.SessionRepository,
	cfg *config.Config,
	cacheStore *cache.Cache,
	logger zerolog.Logger,
) *AuthService {
	client := &http.Client{Timeout: 10 * time.Second}
	return &AuthService{
		repo:        repo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
		cache:       cacheStore,
		httpClient:  client,
		logger:      logger,
	}
}
