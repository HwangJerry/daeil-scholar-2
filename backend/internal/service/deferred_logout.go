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
	claims := &mobileClaims{}
	parsed, err := jwt.ParseWithClaims(proof, claims, func(token *jwt.Token) (any, error) { return []byte(s.cfg.JWT.Secret), nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithoutClaimsValidation())
	if err != nil || !parsed.Valid {
		return repository.ErrRefreshTokenInvalid
	}
	account, err := strconv.Atoi(claims.Subject)
	if err != nil || account <= 0 || strconv.Itoa(account) != claims.Subject {
		return repository.ErrRefreshTokenInvalid
	}
	expectedMetadata := claims.Type == mobileTokenTypeRefresh && claims.Version == mobileTokenVersion && claims.Issuer == mobileTokenIssuer
	expectedAudience := len(claims.Audience) == 1 && claims.Audience[0] == mobileTokenAudience
	originalIdentifiers := originalMobileIdentifier(claims.ID) && originalMobileIdentifier(claims.SessionID)
	requiredDates := claims.ExpiresAt != nil && claims.IssuedAt != nil && claims.NotBefore != nil
	if !expectedMetadata || !expectedAudience || !originalIdentifiers || !requiredDates {
		return repository.ErrRefreshTokenInvalid
	}
	now := time.Now()
	if claims.IssuedAt.After(now) || claims.NotBefore.After(now) || !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		return repository.ErrRefreshTokenInvalid
	}
	// Validate signature and all metadata before the idempotent expired outcome.
	// Original expiry bounds authority even if a successor extended the family.
	if !claims.ExpiresAt.After(now) {
		return nil
	}
	return s.repo.RevokeMobileSessionByProof(ctx, account, claims.SessionID, claims.ID, claims.ExpiresAt.Time)
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
