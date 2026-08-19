package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
)

func TestAdminDonationImportPreviewRequiresDonationDate(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "donations.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("not needed for request validation")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/admin/donation/import/preview", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	NewAdminDonationImportHandler(nil, 1).Preview(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "NO_DONATION_DATE") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestAdminDonationImportCommitRequiresDonationDate(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/admin/donation/import/commit", strings.NewReader(`{"rows":[]}`))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 7}))
	response := httptest.NewRecorder()

	NewAdminDonationImportHandler(nil, 1).Commit(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "NO_DONATION_DATE") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
