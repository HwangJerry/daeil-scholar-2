// auth_service.go — AuthService type definition and constructor
package service

import (
	"context"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/social-auth/kakao"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type KakaoAuthClient interface {
	AuthenticateByCode(context.Context, string, string) (kakao.AuthResult, error)
	AuthenticateByAccessToken(context.Context, string) (kakao.AuthResult, error)
	Logout(context.Context, string) error
}

// AuthService handles authentication, session management, and Kakao OAuth integration.
type AuthService struct {
	repo        *repository.AuthRepository
	sessionRepo *repository.SessionRepository
	cfg         *config.Config
	cache       *cache.Cache
	kakaoClient KakaoAuthClient
	httpClient  *http.Client
	logger      zerolog.Logger
}

// NewAuthService creates an AuthService with all required dependencies.
func NewAuthService(
	repo *repository.AuthRepository,
	sessionRepo *repository.SessionRepository,
	cfg *config.Config,
	cacheStore *cache.Cache,
	kakaoClient KakaoAuthClient,
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
