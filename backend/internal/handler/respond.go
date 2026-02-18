package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, code string, message string) {
	respondJSON(w, status, model.APIError{Code: code, Message: message})
}
