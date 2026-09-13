package service

import (
	"context"
	"github.com/dflh-saf/backend/internal/config"
	"io"
	"net/http"
	"strings"
	"testing"
)

type erasureKakaoTransport func(*http.Request) (*http.Response, error)

func (f erasureKakaoTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestDelayedKakaoUnlinkUsesServerKeyAndValidatesSubject(t *testing.T) {
	for _, body := range []string{`{"id":123}`, `{"id":999}`, `{}`} {
		calls := 0
		svc := &AuthService{cfg: &config.Config{Kakao: config.KakaoConfig{AdminKey: "synthetic-key"}}, httpClient: &http.Client{Transport: erasureKakaoTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Path != "/v1/user/unlink" || r.Header.Get("Authorization") != "KakaoAK synthetic-key" {
				t.Fatal("wrong API/auth")
			}
			if err := r.ParseForm(); err != nil || r.Form.Get("target_id") != "123" || r.Form.Get("target_id_type") != "user_id" {
				t.Fatal("wrong subject")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}}
		err := svc.unlinkKakaoSubject(context.Background(), "123")
		if (err == nil) != (body == `{"id":123}`) || calls != 1 {
			t.Fatal("unverified response accepted", body, err)
		}
		if err = svc.unlinkKakaoSubject(context.Background(), "not-an-id"); err == nil || calls != 1 {
			t.Fatal("bad subject sent")
		}
	}
}
