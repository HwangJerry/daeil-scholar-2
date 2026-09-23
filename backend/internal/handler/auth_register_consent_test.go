package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

// A declined or missing consent must be rejected before any database work or phone
// verification runs, so the handler is built without a database or services.
func TestRegisterRejectsDeclinedConsentBeforeTouchingTheDatabase(t *testing.T) {
	handler := &AuthHandler{logger: zerolog.Nop()}
	handler.AttachPrivacyConsent(service.NewConsentService(nil, config.PrivacyConsentConfig{Version: "2026-09-30", Enforce: true}, zerolog.Nop()))

	cases := []struct {
		name     string
		consent  string
		wantCode string
	}{
		{"declined", `"privacyConsent":{"version":"2026-09-30","accepted":false}`, "CONSENT_REQUIRED"},
		{"missing", `"nick":""`, "CONSENT_REQUIRED"},
		{"outdated", `"privacyConsent":{"version":"2026-01-01","accepted":true}`, "CONSENT_VERSION_OUTDATED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"usrId":"member1","password":"pw","name":"홍길동","phone":"01012345678","email":"m@example.com","fn":"20","fmDept":"영어",` + tc.consent + `}`
			request := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
			recorder := httptest.NewRecorder()
			handler.Register(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tc.wantCode) {
				t.Fatalf("body %s does not contain %s", recorder.Body.String(), tc.wantCode)
			}
		})
	}
}
