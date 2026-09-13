package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Only the server-resolved provider subject is accepted, never a client target ID.
func (s *AuthService) UnlinkKakaoAccount(ctx context.Context, user int) error {
	if s.cfg.Kakao.AdminKey == "" {
		return errors.New("KAKAO_ADMIN_UNLINK_CONFIGURATION_REQUIRED")
	}
	subject, err := s.repo.AccountDeletionKakaoSubject(user)
	if err != nil {
		return errors.New("KAKAO_SUBJECT_REVIEW_REQUIRED")
	}
	return s.unlinkKakaoSubject(ctx, subject)
}
func (s *AuthService) unlinkKakaoSubject(ctx context.Context, subject string) error {
	id, err := strconv.ParseInt(subject, 10, 64)
	if err != nil || id <= 0 {
		return errors.New("KAKAO_SUBJECT_REVIEW_REQUIRED")
	}
	form := url.Values{"target_id_type": {"user_id"}, "target_id": {subject}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://kapi.kakao.com/v1/user/unlink", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "KakaoAK "+s.cfg.Kakao.AdminKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	client := *s.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, err := client.Do(req)
	if err != nil {
		return errors.New("KAKAO_UNLINK_RETRY_REQUIRED")
	}
	defer res.Body.Close()
	var result struct {
		ID int64 `json:"id"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&result) != nil || result.ID != id {
		// An invalid/absent token or an ambiguous provider error is not erasure proof.
		return errors.New("KAKAO_UNLINK_REVIEW_REQUIRED")
	}
	return nil
}
