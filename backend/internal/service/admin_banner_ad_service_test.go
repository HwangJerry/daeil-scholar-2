package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestValidateBannerAd(t *testing.T) {
	startDate := "2026-01-01 00:00:00"
	endDate := "2026-01-31 23:59:59"
	reversedStartDate := "2026-02-01 00:00:00"

	tests := []struct {
		name    string
		banner  model.AdminBannerAdInsert
		wantErr error
	}{
		{
			name: "valid active banner",
			banner: model.AdminBannerAdInsert{
				BNURL:       "https://example.com/banner",
				OpenYN:      "Y",
				BNStartDate: &startDate,
				BNEndDate:   &endDate,
				ImageURLs:   []string{"/uploads/bannerAd/banner.png"},
			},
		},
		{
			name:    "empty URL",
			banner:  model.AdminBannerAdInsert{BNURL: " ", OpenYN: "N"},
			wantErr: ErrInvalidBannerURL,
		},
		{
			name:    "unsupported URL scheme",
			banner:  model.AdminBannerAdInsert{BNURL: "ftp://example.com/banner", OpenYN: "N"},
			wantErr: ErrInvalidBannerURL,
		},
		{
			name:    "missing URL host",
			banner:  model.AdminBannerAdInsert{BNURL: "https:///banner", OpenYN: "N"},
			wantErr: ErrInvalidBannerURL,
		},
		{
			name:    "invalid open flag",
			banner:  model.AdminBannerAdInsert{BNURL: "https://example.com", OpenYN: "true"},
			wantErr: ErrInvalidOpenYN,
		},
		{
			name: "reversed period",
			banner: model.AdminBannerAdInsert{
				BNURL:       "https://example.com",
				OpenYN:      "N",
				BNStartDate: &reversedStartDate,
				BNEndDate:   &endDate,
			},
			wantErr: ErrInvalidBannerPeriod,
		},
		{
			name:    "active without images",
			banner:  model.AdminBannerAdInsert{BNURL: "https://example.com", OpenYN: "Y"},
			wantErr: ErrActiveWithoutImages,
		},
		{
			name:   "inactive without images",
			banner: model.AdminBannerAdInsert{BNURL: "https://example.com", OpenYN: "N"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateBannerAd(&test.banner)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("validateBannerAd() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestFileStorageDeleteUploadedURL(t *testing.T) {
	basePath := t.TempDir()
	bannerDir := filepath.Join(basePath, "bannerAd")
	if err := os.MkdirAll(bannerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(bannerDir, "banner.png")
	if err := os.WriteFile(targetPath, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}

	storage := NewFileStorageService(basePath)
	if err := storage.DeleteUploadedURL("/uploads/bannerAd/banner.png", "bannerAd"); err != nil {
		t.Fatalf("DeleteUploadedURL() error = %v", err)
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("deleted file still exists or stat failed unexpectedly: %v", err)
	}
}

func TestFileStorageDeleteUploadedURLIgnoresUnrelatedURL(t *testing.T) {
	basePath := t.TempDir()
	storage := NewFileStorageService(basePath)

	if err := storage.DeleteUploadedURL("/uploads/notice/notice.png", "bannerAd"); err != nil {
		t.Fatalf("unrelated URL returned error: %v", err)
	}
	if err := storage.DeleteUploadedURL("https://cdn.example.com/banner.png", "bannerAd"); err != nil {
		t.Fatalf("remote URL returned error: %v", err)
	}
}

func TestFileStorageDeleteUploadedURLRejectsTraversal(t *testing.T) {
	storage := NewFileStorageService(t.TempDir())
	if err := storage.DeleteUploadedURL("/uploads/bannerAd/../notice.png", "bannerAd"); err == nil {
		t.Fatal("DeleteUploadedURL() accepted a traversal path")
	}
}
