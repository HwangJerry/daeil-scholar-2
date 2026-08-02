// auth_social_link_error_test.go — Verifies canonical social-link domain error mapping.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/service"
)

func TestWriteSocialLinkErrorMapsUnsupportedMergeToConflict(t *testing.T) {
	recorder := httptest.NewRecorder()
	if handled := writeSocialLinkError(recorder, service.ErrAccountMergeNotSupported); !handled {
		t.Fatal("merge error was not handled")
	}

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", recorder.Code)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["code"] != "ACCOUNT_MERGE_NOT_SUPPORTED" {
		t.Fatalf("error envelope = %#v", response)
	}
}
