// admin_feed_category_handler.go — HTTP lifecycle for feed category admin endpoints
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type AdminFeedCategoryHandler struct {
	svc *service.AdminFeedCategoryService
}

func NewAdminFeedCategoryHandler(svc *service.AdminFeedCategoryService) *AdminFeedCategoryHandler {
	return &AdminFeedCategoryHandler{svc: svc}
}

// feedCategoryHasPostsResponse is the 409 body: the standard error envelope
// plus the post count the admin UI shows in its move-and-delete dialog.
type feedCategoryHasPostsResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	PostCount int    `json:"postCount"`
}

// List handles GET /api/admin/feed-categories — every category (incl. hidden) in order.
func (h *AdminFeedCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.List()
	if err != nil {
		log.Error().Err(err).Msg("feed category list failed")
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list feed categories")
		return
	}
	respondJSON(w, http.StatusOK, cats)
}

// Create handles POST /api/admin/feed-categories.
func (h *AdminFeedCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.AdminFeedCategoryUpsert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	cat, err := h.svc.Create(req)
	if err != nil {
		h.respondServiceError(w, err, "CREATE_FAILED")
		return
	}
	respondJSON(w, http.StatusCreated, cat)
}

// Update handles PUT /api/admin/feed-categories/{seq} — rename and app tab visibility.
func (h *AdminFeedCategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req model.AdminFeedCategoryUpsert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	cat, err := h.svc.Update(seq, req)
	if err != nil {
		h.respondServiceError(w, err, "UPDATE_FAILED")
		return
	}
	respondJSON(w, http.StatusOK, cat)
}

// Reorder handles PUT /api/admin/feed-categories/order — the full seq list in its new order.
func (h *AdminFeedCategoryHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req model.AdminFeedCategoryReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.svc.Reorder(req.Seqs); err != nil {
		h.respondServiceError(w, err, "REORDER_FAILED")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete handles DELETE /api/admin/feed-categories/{seq} with an optional
// {"moveToSeq": n} body; an empty body means "delete only if it has no posts".
func (h *AdminFeedCategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req model.AdminFeedCategoryDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.svc.Delete(seq, req.MoveToSeq); err != nil {
		h.respondServiceError(w, err, "DELETE_FAILED")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminFeedCategoryHandler) respondServiceError(w http.ResponseWriter, err error, failureCode string) {
	var rule *service.FeedCategoryRuleError
	var hasPosts *service.FeedCategoryHasPostsError
	switch {
	case errors.As(err, &rule):
		respondError(w, http.StatusBadRequest, rule.Code, rule.Message)
	case errors.As(err, &hasPosts):
		respondJSON(w, http.StatusConflict, feedCategoryHasPostsResponse{
			Code:      "CATEGORY_HAS_POSTS",
			Message:   "게시글이 있는 카테고리는 옮길 카테고리를 선택해야 삭제할 수 있습니다.",
			PostCount: hasPosts.PostCount,
		})
	case errors.Is(err, service.ErrFeedCategoryNotFound):
		respondError(w, http.StatusNotFound, "NOT_FOUND", "카테고리를 찾을 수 없습니다.")
	default:
		log.Error().Err(err).Str("code", failureCode).Msg("feed category change failed")
		respondError(w, http.StatusInternalServerError, failureCode, "Failed to change feed category")
	}
}
