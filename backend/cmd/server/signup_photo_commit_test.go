package main

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

func TestSocialSignupCommitConsumesProofAndCleansOnlyUnusedRealPhoto(t *testing.T) {
	s := newGoldenServer(t)
	store := service.NewSocialLinkTokenStore(s.deps.cacheStore)
	_, err := store.Put("photo-member", model.SocialLinkData{Provider: "KT", SocialID: "synthetic-photo-subject", Email: "photo@example.test"}, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	first := uploadRealSignupPhoto(t, s, "photo-member")
	selected := uploadRealSignupPhoto(t, s, "photo-member")
	root := os.Getenv("UPLOAD_BASE_PATH")
	disk := func(url string) string { return filepath.Join(root, strings.TrimPrefix(url, "/uploads/")) }
	for _, url := range []string{first, selected} {
		if _, err := os.Stat(disk(url)); err != nil {
			t.Fatal("real upload missing", err)
		}
	}
	requested := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "ios", "100", 200)
	confirmation := s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": decodeGolden[model.PhoneVerificationRequestResult](t, requested).VerificationID, "code": goldenCode}, "", "ios", "100", 200)
	grant := decodeGolden[model.PhoneVerificationConfirmResult](t, confirmation).VerificationToken
	body := s.request(t, http.MethodPost, "/api/auth/social/link", map[string]any{
		"token": "photo-member", "mode": "new", "client": "mobile", "name": "Synthetic Photo Member", "email": "photo@example.test", "phone": goldenPhone, "fn": "20", "fmDept": "영어", "phoneVerificationToken": grant,
		"privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true},
	}, "", "ios", "100", 200)
	result := decodeGolden[model.SocialAuthResult](t, body)
	if result.Session == nil {
		t.Fatal("signup did not mint session")
	}
	var photo string
	if err := s.db.Get(&photo, `SELECT USR_PHOTO FROM WEO_MEMBER WHERE USR_SEQ=?`, result.Session.User.USRSeq); err != nil || photo != selected {
		t.Fatalf("committed photo=%s error=%v", photo, err)
	}
	if _, err := store.Begin("photo-member"); err != service.ErrSocialLinkTokenConsumed {
		t.Fatalf("committed continuation=%v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var count int
		if err := s.db.Get(&count, `SELECT COUNT(*) FROM WEO_FILES WHERE CONCAT(FILE_PATH,'/',FILE_NAME)=?`, first); err != nil {
			t.Fatal(err)
		}
		_, statErr := os.Stat(disk(first))
		if count == 0 && os.IsNotExist(statErr) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("unused real photo not cleaned: DB=%d stat=%v", count, statErr)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(disk(selected)); err != nil {
		t.Fatal("committed real member photo erased", err)
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_FILES WHERE CONCAT(FILE_PATH,'/',FILE_NAME)=?`, selected)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE CONSUMED_USR_SEQ=? AND CONSUMED_YN='Y'`, result.Session.User.USRSeq)
}

func uploadRealSignupPhoto(t *testing.T, s *goldenServer, token string) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("token", token)
	part, err := writer.CreateFormFile("file", "synthetic.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(part, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	req, err := http.NewRequest(http.MethodPost, s.server.URL+"/api/auth/social/link/photo", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	setGoldenClientHeaders(req, "", "ios", "100")
	response, err := s.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("real photo upload=%d body=%s", response.StatusCode, raw)
	}
	var result map[string]string
	result = decodeGolden[map[string]string](t, raw)
	return result["url"]
}
