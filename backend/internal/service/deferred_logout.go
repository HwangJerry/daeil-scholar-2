package service

import (
	"context"
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

// RevokeEndedMobileSession authenticates an existing refresh token solely as an
// ended-session proof. It does not call refresh, issue credentials or load profile.
func (s *AuthService) RevokeEndedMobileSession(ctx context.Context, proof string) error {
	return s.RevokeEndedMobileSessionWithDevice(ctx, proof, "")
}

func (s *AuthService) RevokeEndedMobileSessionWithDevice(ctx context.Context, proof, device string) error {
	_, err := s.RevokeEndedMobileSessionWithDeviceResult(ctx, proof, device)
	return err
}

// A signed expired/missing proof is idempotent, but is not revocation confirmation.
func (s *AuthService) RevokeEndedMobileSessionWithDeviceResult(ctx context.Context, proof, device string) (bool, error) {
	if device != "" && !validPushDeviceToken(device) {
		return false, ErrInvalidPushRequest
	}
	claims, err := s.endedSessionProof(proof)
	if err != nil {
		return false, err
	}
	if !claims.ExpiresAt.After(time.Now()) {
		return false, nil
	}
	account, _ := strconv.Atoi(claims.Subject)
	return s.repo.RevokeMobileSessionByProofWithDeviceResult(ctx, account, claims.SessionID, claims.ID, claims.ExpiresAt.Time, device)
}

// Global logout is one-shot: absent/revoked/expired original proof is not success.
func (s *AuthService) RevokeAllSessionsWithOriginalProof(ctx context.Context, proof string) error {
	claims, err := s.endedSessionProof(proof)
	if err != nil {
		return err
	}
	if !claims.ExpiresAt.After(time.Now()) {
		return repository.ErrRefreshTokenInvalid
	}
	account, _ := strconv.Atoi(claims.Subject)
	return s.repo.RevokeAllSessionsByProof(ctx, account, claims.SessionID, claims.ID, claims.ExpiresAt.Time)
}

func (s *AuthService) endedSessionProof(proof string) (*mobileClaims, error) {
	claims := &mobileClaims{}
	parsed, err := jwt.ParseWithClaims(proof, claims, func(token *jwt.Token) (any, error) { return []byte(s.cfg.JWT.Secret), nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithoutClaimsValidation())
	if err != nil || !parsed.Valid {
		return nil, repository.ErrRefreshTokenInvalid
	}
	account, err := strconv.Atoi(claims.Subject)
	if err != nil || account <= 0 || strconv.Itoa(account) != claims.Subject {
		return nil, repository.ErrRefreshTokenInvalid
	}
	expectedMetadata := claims.Type == mobileTokenTypeRefresh && claims.Version == mobileTokenVersion && claims.Issuer == mobileTokenIssuer
	expectedAudience := len(claims.Audience) == 1 && claims.Audience[0] == mobileTokenAudience
	originalIdentifiers := originalMobileIdentifier(claims.ID) && originalMobileIdentifier(claims.SessionID)
	requiredDates := claims.ExpiresAt != nil && claims.IssuedAt != nil && claims.NotBefore != nil
	if !expectedMetadata || !expectedAudience || !originalIdentifiers || !requiredDates {
		return nil, repository.ErrRefreshTokenInvalid
	}
	now := time.Now()
	if claims.IssuedAt.After(now) || claims.NotBefore.After(now) || !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		return nil, repository.ErrRefreshTokenInvalid
	}
	return claims, nil
}

const mobileSessionIdentifierHexLength = 32

func originalMobileIdentifier(value string) bool {
	if len(value) != mobileSessionIdentifierHexLength {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
