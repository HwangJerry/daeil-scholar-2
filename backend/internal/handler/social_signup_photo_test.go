package handler

import (
	"bytes"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type racedPhotoUploader struct {
	DuringUpload func()
	Discarded    []*service.UploadResult
	Calls        int
}

func (u *racedPhotoUploader) Upload(_ multipart.File, _ *multipart.FileHeader, _ string) (*service.UploadResult, error) {
	u.Calls++
	u.DuringUpload()
	return &service.UploadResult{FSeq: 71, URL: "/uploads/profile/synthetic.jpg"}, nil
}
func (u *racedPhotoUploader) DiscardUnclaimedProfile(result *service.UploadResult) error {
	u.Discarded = append(u.Discarded, result)
	return nil
}
func TestSocialPhotoUploadLosingTokenRaceDiscardsExactResult(t *testing.T) {
	for _, mode := range []string{"cancel", "expiry", "begin"} {
		t.Run(mode, func(t *testing.T) {
			store := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
			ttl := time.Minute
			if mode == "expiry" {
				ttl = 20 * time.Millisecond
			}
			_, _ = store.Put("photo", model.SocialLinkData{ProfileImageURL: "https://provider.example/photo"}, ttl)
			uploader := &racedPhotoUploader{DuringUpload: func() {
				switch mode {
				case "cancel":
					_ = store.Cancel("photo")
				case "expiry":
					time.Sleep(40 * time.Millisecond)
				case "begin":
					_, _ = store.Begin("photo")
				}
			}}
			h := NewSocialLinkPhotoHandler(uploader, store, zerolog.Nop())
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			_ = form.WriteField("token", "photo")
			part, _ := form.CreateFormFile("file", "qa.jpg")
			_, _ = part.Write([]byte("synthetic"))
			_ = form.Close()
			req := httptest.NewRequest(http.MethodPost, "/api/auth/social/link/photo", &body)
			req.Header.Set("Content-Type", form.FormDataContentType())
			w := httptest.NewRecorder()
			h.Upload(w, req)
			if w.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if len(uploader.Discarded) != 1 || uploader.Discarded[0].FSeq != 71 || uploader.Discarded[0].URL != "/uploads/profile/synthetic.jpg" {
				t.Fatalf("discarded=%v", uploader.Discarded)
			}
		})
	}
}
func TestCancelledSocialPhotoRejectedBeforeUpload(t *testing.T) {
	store := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	_, _ = store.Put("photo", model.SocialLinkData{}, time.Minute)
	_ = store.Cancel("photo")
	uploader := &racedPhotoUploader{DuringUpload: func() { t.Fatal("cancelled token performed upload") }}
	h := NewSocialLinkPhotoHandler(uploader, store, zerolog.Nop())
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	_ = form.WriteField("token", "photo")
	_ = form.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/social/link/photo", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	w := httptest.NewRecorder()
	h.Upload(w, req)
	if w.Code != http.StatusNotFound || uploader.Calls != 0 {
		t.Fatalf("status=%d uploads=%d", w.Code, uploader.Calls)
	}
}

func TestProcessingSocialSignupPrefillAndPhotoReturnConflict(t *testing.T) {
	store := service.NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	_, _ = store.Put("processing", model.SocialLinkData{}, time.Minute)
	_, _ = store.Begin("processing")
	prefill := httptest.NewRecorder()
	(&AuthHandler{socialLinkTokens: store}).SocialLinkPrefill(prefill, httptest.NewRequest(http.MethodGet, "/api/auth/social/link/prefill?token=processing", nil))
	if prefill.Code != http.StatusConflict {
		t.Errorf("prefill=%d, want409", prefill.Code)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	_ = form.WriteField("token", "processing")
	_ = form.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/social/link/photo", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	w := httptest.NewRecorder()
	NewSocialLinkPhotoHandler(&racedPhotoUploader{}, store, zerolog.Nop()).Upload(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("photo=%d, want409", w.Code)
	}
}
