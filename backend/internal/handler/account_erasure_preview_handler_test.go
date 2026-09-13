package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/repository"
)

func TestStaleErasurePlanReturnsConflict(t *testing.T) {
	w := httptest.NewRecorder()
	deletionRequestError(w, repository.ErrErasurePlanChanged)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "ERASURE_PLAN_CHANGED") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
