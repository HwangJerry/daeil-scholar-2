// sms_ncp_sender_test.go — Pins the SENS request shape and signature algorithm so a
// refactor cannot silently break outbound verification codes.
package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

func testNCPConfig() config.SMSConfig {
	return config.SMSConfig{
		Provider:     "ncp",
		Sender:       "0212345678",
		NCPAccessKey: "test-access-key",
		NCPSecretKey: "test-secret-key",
		NCPServiceID: "ncp:sms:kr:123456789:dflh",
	}
}

func TestNCPSignatureMatchesDocumentedStringToSign(t *testing.T) {
	sender := NewNCPSENSSender(testNCPConfig(), zerolog.Nop())
	const uri = "/sms/v2/services/ncp:sms:kr:123456789:dflh/messages"
	const timestamp = "1764000000000"

	got := sender.signature(http.MethodPost, uri, timestamp)

	// NCP spec: base64(HmacSHA256(secretKey, "METHOD URI\ntimestamp\naccessKey"))
	mac := hmac.New(sha256.New, []byte("test-secret-key"))
	mac.Write([]byte("POST " + uri + "\n" + timestamp + "\n" + "test-access-key"))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Errorf("signature = %q, want %q", got, want)
	}
}

func TestNCPSendPostsDocumentedPayloadAndHeaders(t *testing.T) {
	var captured struct {
		path      string
		headers   http.Header
		body      ncpSENSRequest
		requested bool
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.requested = true
		captured.path = r.URL.Path
		captured.headers = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured.body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"requestId":"req-1","statusCode":"202","statusName":"success"}`))
	}))
	defer server.Close()

	sender := NewNCPSENSSender(testNCPConfig(), zerolog.Nop())
	sender.client = server.Client()
	sender.host = server.URL

	if err := sender.Send(model.SMSMessage{To: "01012345678", Body: "[대일외고장학회] 인증번호 123456 를 입력해 주세요."}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if !captured.requested {
		t.Fatal("SENS endpoint was never called")
	}
	if want := "/sms/v2/services/ncp:sms:kr:123456789:dflh/messages"; captured.path != want {
		t.Errorf("path = %q, want %q", captured.path, want)
	}
	for _, header := range []string{"X-Ncp-Apigw-Timestamp", "X-Ncp-Iam-Access-Key", "X-Ncp-Apigw-Signature-V2"} {
		if captured.headers.Get(header) == "" {
			t.Errorf("missing required header %s", header)
		}
	}
	if captured.body.Type != "SMS" {
		t.Errorf("type = %q, want SMS", captured.body.Type)
	}
	if captured.body.From != "0212345678" {
		t.Errorf("from = %q, want the registered sender", captured.body.From)
	}
	if len(captured.body.Messages) != 1 || captured.body.Messages[0].To != "01012345678" {
		t.Errorf("messages = %+v, want a single recipient 01012345678", captured.body.Messages)
	}
}

func TestNCPSendFailsWhenSENSRejectsTheMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"requestId":"req-2","statusCode":"401","statusName":"fail"}`))
	}))
	defer server.Close()

	sender := NewNCPSENSSender(testNCPConfig(), zerolog.Nop())
	sender.client = server.Client()
	sender.host = server.URL

	if err := sender.Send(model.SMSMessage{To: "01012345678", Body: "code"}); err == nil {
		t.Fatal("Send() error = nil, want a failure when SENS returns a non-202 statusCode")
	}
}

func TestSMSConfigRequiresProviderSpecificCredentials(t *testing.T) {
	cases := map[string]struct {
		cfg  config.SMSConfig
		want bool
	}{
		"complete ncp":        {testNCPConfig(), true},
		"ncp missing service": {config.SMSConfig{Provider: "ncp", Sender: "02", NCPAccessKey: "a", NCPSecretKey: "b"}, false},
		"ncp missing sender":  {config.SMSConfig{Provider: "ncp", NCPAccessKey: "a", NCPSecretKey: "b", NCPServiceID: "c"}, false},
		"complete aligo":      {config.SMSConfig{Provider: "aligo", Sender: "02", APIKey: "k", UserID: "u"}, true},
		"aligo keys only":     {config.SMSConfig{Provider: "aligo", APIKey: "k", UserID: "u"}, false},
		"unknown provider":    {config.SMSConfig{Provider: "twilio", Sender: "02"}, false},
		"empty":               {config.SMSConfig{}, false},
	}
	for name, testCase := range cases {
		if got := testCase.cfg.Configured(); got != testCase.want {
			t.Errorf("%s: Configured() = %v, want %v", name, got, testCase.want)
		}
	}
}
