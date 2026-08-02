package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type memberBlockServiceStub struct {
	listState    *model.MemberBlockListResponse
	getState     *model.MemberBlockState
	blockState   *model.MemberBlockState
	unblockState *model.MemberBlockState
	blockErr     error
}

func (s *memberBlockServiceStub) List(int) (*model.MemberBlockListResponse, error) {
	return s.listState, nil
}
func (s *memberBlockServiceStub) Get(int, int) (*model.MemberBlockState, error) {
	return s.getState, nil
}
func (s *memberBlockServiceStub) Block(int, int) (*model.MemberBlockState, error) {
	return s.blockState, s.blockErr
}
func (s *memberBlockServiceStub) Unblock(int, int) (*model.MemberBlockState, error) {
	return s.unblockState, nil
}

func blockHandlerRouter(service MemberBlockServicer) chi.Router {
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 101})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	h := &MemberBlockHandler{service: service}
	router.Get("/api/blocks", h.List)
	router.Get("/api/blocks/{userSeq}", h.Get)
	router.Put("/api/blocks/{userSeq}", h.Put)
	router.Delete("/api/blocks/{userSeq}", h.Delete)
	return router
}

func TestMemberBlockHandlerWritesClosedActiveState(t *testing.T) {
	updatedAt := "2026-07-29T01:00:00Z"
	router := blockHandlerRouter(&memberBlockServiceStub{blockState: &model.MemberBlockState{UserSeq: 202, BlockedByMe: true, UpdatedAt: &updatedAt}})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/blocks/202", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 3 || body["userSeq"] != float64(202) || body["blockedByMe"] != true || body["updatedAt"] != updatedAt {
		t.Fatalf("body = %#v", body)
	}
}

func TestMemberBlockHandlerWritesNullTimestampWhenUnblocked(t *testing.T) {
	router := blockHandlerRouter(&memberBlockServiceStub{unblockState: &model.MemberBlockState{UserSeq: 202, BlockedByMe: false, UpdatedAt: nil}})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/api/blocks/202", nil))
	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 3 || body["updatedAt"] != nil || body["blockedByMe"] != false {
		t.Fatalf("body = %#v", body)
	}
}

func TestMemberBlockHandlerRejectsInvalidTarget(t *testing.T) {
	router := blockHandlerRouter(&memberBlockServiceStub{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/blocks/not-a-number", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "INVALID_USER_SEQ" {
		t.Fatalf("code = %q", response.Code)
	}
}

func TestMemberBlockHandlerHidesMissingOrUnapprovedPutTarget(t *testing.T) {
	router := blockHandlerRouter(&memberBlockServiceStub{blockErr: service.ErrMemberBlockTargetNotFound})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/blocks/202", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	var response model.APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "INVALID_USER_SEQ" {
		t.Fatalf("code = %q", response.Code)
	}
}
