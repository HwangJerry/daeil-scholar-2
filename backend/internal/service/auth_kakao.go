// auth_kakao.go — Kakao OAuth token exchange, caching, and logout
package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
func (s *AuthService) ExchangeKakaoToken(code string) (KakaoUserInfo, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", s.cfg.Kakao.ClientID)
	form.Set("client_secret", s.cfg.Kakao.ClientSecret)
	form.Set("redirect_uri", s.cfg.Kakao.RedirectURI)
	form.Set("code", code)
	req, err := http.NewRequest(http.MethodPost, "https://kauth.kakao.com/oauth/token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return KakaoUserInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return KakaoUserInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return KakaoUserInfo{}, fmt.Errorf("kakao token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return KakaoUserInfo{}, err
	}
	userReq, err := http.NewRequest(http.MethodGet, "https://kapi.kakao.com/v2/user/me", nil)
	if err != nil {
		return KakaoUserInfo{}, err
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return KakaoUserInfo{}, err
	}
	defer userResp.Body.Close()
	if userResp.StatusCode >= 400 {
		body, _ := io.ReadAll(userResp.Body)
		return KakaoUserInfo{}, errors.New(string(body))
	}
	var userPayload struct {
		ID      int64 `json:"id"`
		Account struct {
			Email   string `json:"email"`
			Profile struct {
				Nickname        string `json:"nickname"`
				ProfileImageURL string `json:"profile_image_url"`
				IsDefaultImage  bool   `json:"is_default_image"`
			} `json:"profile"`
		} `json:"kakao_account"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&userPayload); err != nil {
		return KakaoUserInfo{}, err
	}
	profileImageURL := userPayload.Account.Profile.ProfileImageURL
	if userPayload.Account.Profile.IsDefaultImage {
		profileImageURL = ""
	}
	return KakaoUserInfo{
		KakaoID:         strconv.FormatInt(userPayload.ID, 10),
		Email:           userPayload.Account.Email,
		Nickname:        userPayload.Account.Profile.Nickname,
		ProfileImageURL: profileImageURL,
		AccessToken:     tokenResp.AccessToken,
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
	req, err := http.NewRequest(http.MethodPost, "https://kapi.kakao.com/v1/user/logout", nil)
	if err != nil {
		return fmt.Errorf("kakao logout request build: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kakao logout request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("kakao logout failed (status %d): %s", resp.StatusCode, string(body))
	}
	s.logger.Info().Int("usrSeq", usrSeq).Msg("kakao access token invalidated")
	return nil
}
