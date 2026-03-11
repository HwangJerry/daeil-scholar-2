package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type FeedHandler struct {
	service     *service.FeedService
	likeService *service.LikeService
	presenter   *presenter.FeedPresenter
}

func NewFeedHandler(svc *service.FeedService, likeSvc *service.LikeService, pres *presenter.FeedPresenter) *FeedHandler {
	return &FeedHandler{service: svc, likeService: likeSvc, presenter: pres}
}

func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userSeq := 0
	if user := middleware.GetAuthUser(r.Context()); user != nil {
		userSeq = user.USRSeq
	}
	feed, err := h.service.GetFeed(parseCursor(q.Get("cursor")), parseIntParam(q.Get("size")), parseExcludeAds(q.Get("exclude_ads")), parseIntParam(q.Get("exclude_seq")), userSeq)
	if err != nil {
		log.Error().Err(err).Msg("feed list failed")
		respondError(w, http.StatusInternalServerError, "FEED_FAILED", "Failed to load feed")
		return
	}
	respondJSON(w, http.StatusOK, feed)
}

func (h *FeedHandler) GetHero(w http.ResponseWriter, r *http.Request) {
	hero, err := h.service.GetHero()
	if err != nil {
		log.Error().Err(err).Msg("feed hero failed")
		respondError(w, http.StatusInternalServerError, "HERO_FAILED", "Failed to load hero")
		return
	}
	if hero == nil {
		respondError(w, http.StatusNotFound, "NO_HERO", "No hero notice found")
		return
	}
	respondJSON(w, http.StatusOK, hero)
}

func (h *FeedHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	detail, err := h.service.GetNoticeDetail(seq)
	if err != nil {
		log.Error().Err(err).Int("seq", seq).Msg("feed detail failed")
		respondError(w, http.StatusInternalServerError, "DETAIL_FAILED", "Failed to load detail")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Not found")
		return
	}

	if user := middleware.GetAuthUser(r.Context()); user != nil {
		if liked, err := h.likeService.HasUserLiked(seq, user.USRSeq); err == nil {
			detail.UserLiked = liked
		}
	}

	respondJSON(w, http.StatusOK, h.presenter.FormatNoticeDetail(detail))
}

func (h *FeedHandler) GetSiblings(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	siblings, err := h.service.GetPostSiblings(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SIBLINGS_FAILED", "Failed to load siblings")
		return
	}
	respondJSON(w, http.StatusOK, siblings)
}
