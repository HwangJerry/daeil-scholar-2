package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

func (h *AuthHandler) AppleChallenge(w http.ResponseWriter, _ *http.Request) {
	challenge, err := h.appleVerifier.CreateChallenge()
	if err != nil {
		h.logger.Error().Err(err).Msg("apple challenge creation failed")
		respondError(w, http.StatusInternalServerError, "CHALLENGE_FAILED", "Apple 로그인을 시작할 수 없습니다.")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"challengeId": challenge.ID,
		"nonce":       challenge.Nonce,
		"expiresAt":   challenge.ExpiresAt.Unix(),
	})
}

func (h *AuthHandler) AppleMobileLogin(w http.ResponseWriter, r *http.Request) {
	var request model.AppleMobileLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if strings.TrimSpace(request.ChallengeID) == "" ||
		strings.TrimSpace(request.IdentityToken) == "" ||
		strings.TrimSpace(request.AuthorizationCode) == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Apple credential is incomplete")
		return
	}
	result, err := h.socialAuth.Authenticate(
		r.Context(),
		model.AppleAuthorization{
			ChallengeID:       request.ChallengeID,
			IdentityToken:     request.IdentityToken,
			AuthorizationCode: request.AuthorizationCode,
			GivenName:         strings.TrimSpace(request.GivenName),
			FamilyName:        strings.TrimSpace(request.FamilyName),
		},
	)
	if err != nil {
		h.logger.Warn().Err(err).Msg("apple mobile verification failed")
		respondError(w, http.StatusUnauthorized, "APPLE_VERIFICATION_FAILED", "Apple login failed")
		return
	}
	writeMobileAuthResult(w, result)
}
