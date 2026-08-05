// auth_mobile_response.go — Canonical HTTP mapping for mobile authentication results.
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
)

func writeMobileAuthResult(w http.ResponseWriter, result model.SocialAuthResult) {
	switch result.Status {
	case model.SocialAuthAuthenticated:
		if result.Session == nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Invalid authentication result")
			return
		}
		respondJSON(w, http.StatusOK, result)
	case model.SocialAuthLinkRequired:
		if result.LinkRequired == nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Invalid authentication result")
			return
		}
		respondJSON(w, http.StatusAccepted, result)
	default:
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Invalid authentication result")
	}
}
