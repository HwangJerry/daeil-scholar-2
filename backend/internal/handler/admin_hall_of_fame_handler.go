// admin_hall_of_fame_handler.go — Operator list and manual create/edit/delete of wall entries.
package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const hallOfFameNoTierFilter = "none"

type AdminHallOfFameHandler struct {
	service *service.HallOfFameService
}

func NewAdminHallOfFameHandler(hallOfFameService *service.HallOfFameService) *AdminHallOfFameHandler {
	return &AdminHallOfFameHandler{service: hallOfFameService}
}

func hallOfFameActor(w http.ResponseWriter, r *http.Request) (service.HallOfFameActor, bool) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return service.HallOfFameActor{}, false
	}
	return service.HallOfFameActor{UsrSeq: user.USRSeq, IP: requestIP(r)}, true
}

func parseHallOfFameFilter(r *http.Request) (model.HallOfFameAdminFilter, bool) {
	values := r.URL.Query()
	filter := model.HallOfFameAdminFilter{Kind: values.Get("kind"), Query: values.Get("q"), Page: parseIntParam(values.Get("page"))}
	if filter.Kind != "" && !model.HallOfFameKind(filter.Kind).Valid() {
		return filter, false
	}
	switch tier := strings.TrimSpace(values.Get("tier")); tier {
	case "":
	case hallOfFameNoTierFilter:
		filter.NoTier = true
	default:
		parsed, err := strconv.Atoi(tier)
		if err != nil {
			return filter, false
		}
		filter.Tier = &parsed
	}
	if raw := values.Get("visible"); raw != "" {
		visible, err := strconv.ParseBool(raw)
		if err != nil {
			return filter, false
		}
		filter.Visible = &visible
	}
	return filter, true
}

func (h *AdminHallOfFameHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseHallOfFameFilter(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "INVALID_FILTER", "필터 값이 올바르지 않습니다.")
		return
	}
	list, err := h.service.ListAdmin(filter)
	if err != nil {
		respondHallOfFameError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, list)
}

func (h *AdminHallOfFameHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := hallOfFameActor(w, r)
	if !ok {
		return
	}
	var input model.HallOfFameCreateInput
	if err := decodeClosedJSON(r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "요청 본문이 올바르지 않습니다.")
		return
	}
	entry, err := h.service.Create(input, actor)
	if err != nil {
		respondHallOfFameError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, entry)
}

func (h *AdminHallOfFameHandler) Patch(w http.ResponseWriter, r *http.Request) {
	actor, ok := hallOfFameActor(w, r)
	if !ok {
		return
	}
	id, ok := parseHallOfFameID(w, r)
	if !ok {
		return
	}
	var input model.HallOfFamePatchInput
	if err := decodeClosedJSON(r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "요청 본문이 올바르지 않습니다.")
		return
	}
	entry, err := h.service.Patch(id, input, actor)
	if err != nil {
		respondHallOfFameError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, entry)
}

func (h *AdminHallOfFameHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := hallOfFameActor(w, r)
	if !ok {
		return
	}
	id, ok := parseHallOfFameID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(id, actor); err != nil {
		respondHallOfFameError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseHallOfFameID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "잘못된 ID입니다.")
		return 0, false
	}
	return id, true
}

func respondHallOfFameError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrHallOfFameInvalidInput):
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", service.HallOfFameRejectionMessage(err))
	case errors.Is(err, service.ErrHallOfFameDuplicate), errors.Is(err, repository.ErrHallOfFameDuplicateKey):
		respondError(w, http.StatusConflict, "DUPLICATE_ENTRY", "같은 이름·기수·종류의 행이 이미 있습니다.")
	case errors.Is(err, repository.ErrHallOfFameEntryNotFound):
		respondError(w, http.StatusNotFound, "NOT_FOUND", "행을 찾을 수 없습니다.")
	default:
		log.Error().Err(err).Msg("hall of fame admin operation failed")
		respondError(w, http.StatusInternalServerError, "HALL_OF_FAME_FAILED", "처리 중 오류가 발생했습니다.")
	}
}
