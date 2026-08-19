// auth_kakao.go — Kakao OAuth token exchange, caching, and logout
package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// KakaoUserInfo aggregates the fields fetched from Kakao OAuth token exchange + /v2/user/me.
type KakaoUserInfo struct {
	KakaoID         string
	Email           string
	Nickname        string
	ProfileImageURL string
	AccessToken     string
}

// ExchangeKakaoToken exchanges an OAuth authorization code for a Kakao access token
// and fetches the Kakao user profile. ProfileImageURL is empty when the user declined
// the optional profile_image consent or is using Kakao's default avatar.
func (s *AuthService) ExchangeKakaoTokenWithRedirect(code string, redirectURI string) (KakaoUserInfo, error) {
	result, err := s.kakaoClient.AuthenticateByCode(context.Background(), code, redirectURI)
	if err != nil {
		return KakaoUserInfo{}, err
	}
	return KakaoUserInfo{
		KakaoID:         result.Profile.KakaoID,
		Email:           result.Profile.Email,
		Nickname:        result.Profile.Nickname,
		ProfileImageURL: result.Profile.ProfileImageURL,
		AccessToken:     result.AccessToken,
	}, nil
}

// ExchangeKakaoToken exchanges using configured redirect URI.
func (s *AuthService) ExchangeKakaoToken(code string) (KakaoUserInfo, error) {
	return s.ExchangeKakaoTokenWithRedirect(code, s.cfg.Kakao.RedirectURI)
}

// GetKakaoProfileByAccessToken validates an existing Kakao access token and returns auth info.
func (s *AuthService) GetKakaoProfileByAccessToken(accessToken string) (KakaoUserInfo, error) {
	result, err := s.kakaoClient.AuthenticateByAccessToken(context.Background(), accessToken)
	if err != nil {
		return KakaoUserInfo{}, err
	}
	return KakaoUserInfo{
		KakaoID:         result.Profile.KakaoID,
		Email:           result.Profile.Email,
		Nickname:        result.Profile.Nickname,
		ProfileImageURL: result.Profile.ProfileImageURL,
		AccessToken:     result.AccessToken,
	}, nil
}

// CacheKakaoToken stores the Kakao access token in the in-memory cache, keyed by usrSeq.
// The TTL matches the JWT max age so the token expires alongside the app session.
func (s *AuthService) CacheKakaoToken(usrSeq int, token string) {
	key := fmt.Sprintf("kakao_token:%d", usrSeq)
	s.cache.Set(key, token, s.cfg.JWT.MaxAge)
}

// LogoutKakao calls the Kakao logout API to invalidate the cached access token.
// Returns nil if no token was cached (non-Kakao login or server restart).
func (s *AuthService) LogoutKakao(usrSeq int) error {
	key := fmt.Sprintf("kakao_token:%d", usrSeq)
	cached, found := s.cache.Get(key)
	if !found {
		return nil
	}
	s.cache.Delete(key)
	token, ok := cached.(string)
	if !ok {
		return nil
	}
	if err := s.kakaoClient.Logout(context.Background(), token); err != nil {
		return fmt.Errorf("kakao logout request failed: %w", err)
	}
	s.logger.Info().Int("usrSeq", usrSeq).Msg("kakao access token invalidated")
	return nil
}

// UnlinkKakaoToken calls Kakao's unlink API to fully disconnect the app from the
// user's Kakao account (distinct from Logout, which only invalidates the token).
// Used when a member disconnects their Kakao social login connection.
func (s *AuthService) UnlinkKakaoToken(ctx context.Context, accessToken string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://kapi.kakao.com/v1/user/unlink", nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := s.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("kakao unlink failed with status %d", response.StatusCode)
	}
	return nil
}
