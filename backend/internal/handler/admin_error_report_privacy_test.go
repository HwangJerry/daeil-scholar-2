// admin_error_report_privacy_test.go — Browser diagnostics cannot leak into journal logs.
package handler

import (
	"bytes"
	"github.com/rs/zerolog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminErrorReportDropsRawBrowserInput(t *testing.T) {
	var log bytes.Buffer
	handler := NewAdminErrorReportHandler(zerolog.New(&log), nil)
	request := httptest.NewRequest("POST", "/api/admin/error-report", strings.NewReader(`{"message":"private@example.test","stack":"private@example.test","url":"https://example.test/?token=secret","component":"private@example.test"}`))
	response := httptest.NewRecorder()
	handler.Report(response, request)
	if response.Code != 204 {
		t.Fatal(response.Code)
	}
	if strings.Contains(log.String(), "private@example.test") || strings.Contains(log.String(), "secret") {
		t.Fatal("raw browser error persisted")
	}
}
