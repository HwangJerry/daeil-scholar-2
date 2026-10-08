package observability

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testTelemetry(t *testing.T) *Telemetry {
	return &Telemetry{logger: zerolog.Nop(), started: time.Now(), requests: map[string]*bucket{}, logins: map[string]uint64{}, spool: t.TempDir(), exportKey: strings.Repeat("k", 64)}
}
func TestTrustedProxyIP(t *testing.T) {
	for _, c := range []struct{ peer, xff, want, source string }{{"127.0.0.1:123", "198.51.100.99, 192.0.2.1", "192.0.2.1", "trusted_proxy"}, {"192.0.2.2:123", "198.51.100.99", "192.0.2.2", "socket"}, {"[::1]:123", "2001:db8::1", "2001:db8::1", "trusted_proxy"}, {"invalid", "not an IP", "", "unknown"}} {
		r := httptest.NewRequest("POST", "/", nil)
		r.RemoteAddr = c.peer
		r.Header.Set("X-Forwarded-For", c.xff)
		ip, src := ClientIP(r)
		if ip != c.want || src != c.source {
			t.Fatalf("got %s %s", ip, src)
		}
	}
}
func TestSignalsNeverContainPrivatePathsOrBodies(t *testing.T) {
	tel := testTelemetry(t)
	r := chi.NewRouter()
	r.Use(tel.Middleware)
	r.Get("/api/profile/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	req := httptest.NewRequest("GET", "/api/profile/private-email@example.com?token=private-token", strings.NewReader("private-password"))
	req.Header.Set("Authorization", "Bearer private-secret")
	r.ServeHTTP(httptest.NewRecorder(), req)
	b, _ := json.Marshal([]interface{}{tel.metrics(), tel.spans, tel.logs})
	for _, s := range []string{"private-email", "private-token", "private-password", "private-secret"} {
		if bytes.Contains(b, []byte(s)) {
			t.Fatalf("private data leaked: %s", s)
		}
	}
	if !bytes.Contains(b, []byte("/api/profile/{id}")) {
		t.Fatal("registered route missing")
	}
}
func TestMobileRejectsPrivateFields(t *testing.T) {
	tel := testTelemetry(t)
	event := MobileEvent{Trace: ID(16), Span: ID(8), Screen: "login", Action: "screen.view", Start: time.Now().UnixMilli()}
	payload, _ := json.Marshal(map[string]interface{}{"events": []MobileEvent{event}, "password": "secret"})
	r := httptest.NewRequest("POST", "/", bytes.NewReader(payload))
	r.Header.Set("X-Client-Platform", "ios")
	w := httptest.NewRecorder()
	tel.CollectMobile(w, r)
	if w.Code != 400 || len(tel.spans) != 0 {
		t.Fatal("unknown private field accepted")
	}
	event.Route = "/api/profile/email@example.com"
	event.Action = "http.request"
	event.Method = "GET"
	if validateMobile(event, time.Now().UnixMilli()) {
		t.Fatal("raw path accepted")
	}
	event.Route = "/api/profile"
	event.Start = time.Now().Add(-10 * time.Minute).UnixMilli()
	if validateMobile(event, time.Now().UnixMilli()) {
		t.Fatal("old event accepted")
	}
}
func TestLoginLabels(t *testing.T) {
	if outcome(200) != "success" || outcome(202) == "success" || outcome(429) != "rate_limited" || outcome(500) != "server_error" {
		t.Fatal("invalid labels")
	}
	tel := testTelemetry(t)
	if tel.pseudonym("ip", "192.0.2.1") == tel.pseudonym("install", "192.0.2.1") {
		t.Fatal("domains not separated")
	}
	if tel.pseudonym("ip", "") != "" {
		t.Fatal("missing data became identity")
	}
}
func TestExportPseudonymizesByDefault(t *testing.T) {
	db, mock, e := sqlmock.New()
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	tel := testTelemetry(t)
	tel.db = sqlx.NewDb(db, "mysql")
	at := time.Now().UTC()
	columns := []string{"id", "event_id", "occurred_at", "provider", "outcome", "response_status", "duration_ms", "client_ip", "ip_source", "install_id", "platform", "app_version", "app_build", "trace_id", "schema_version"}
	mock.ExpectQuery("SELECT \\* FROM LOGIN_SECURITY_EVENTS").WillReturnRows(sqlmock.NewRows(columns).AddRow(1, ID(16), at, "password", "invalid_credentials", 401, 10, "192.0.2.1", "trusted_proxy", "abcdef00-1234-1234-1234-abcdef123456", "ios", "1.3.2", "2", ID(16), 1))
	r := httptest.NewRequest("GET", "/?from="+at.Add(-time.Hour).Format(time.RFC3339)+"&to="+at.Add(time.Hour).Format(time.RFC3339), nil)
	w := httptest.NewRecorder()
	tel.ExportLogins(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "192.0.2.1") || strings.Contains(w.Body.String(), "abcdef00-") {
		t.Fatal("raw identities exported")
	}
	if w.Header().Get("X-Next-Cursor") != "1" {
		t.Fatal("cursor missing")
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}

func TestMobileLimiterBoundsDistinctClients(t *testing.T) {
	mobileMu.Lock()
	mobileLimits = map[string]struct {
		minute int64
		count  int
	}{}
	mobileMu.Unlock()
	for i := 0; i < 10000; i++ {
		if !consumeMobile(strconv.Itoa(i), 100) {
			t.Fatal("premature limit")
		}
	}
	if consumeMobile("overflow", 100) || len(mobileLimits) != 10000 {
		t.Fatal("IP map exceeded its bound")
	}
	if !consumeMobile("new-minute", 101) {
		t.Fatal("expired clients not evicted")
	}
}

func TestLoginOutboxPersistsAndReplaysWithoutCredentialPayload(t *testing.T) {
	db, mock, e := sqlmock.New()
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	tel := testTelemetry(t)
	tel.db = sqlx.NewDb(db, "mysql")
	mock.ExpectExec("INSERT IGNORE INTO LOGIN_SECURITY_EVENTS").WillReturnError(errors.New("database unavailable"))
	req := httptest.NewRequest("POST", "/api/auth/mobile/login", strings.NewReader(`{"password":"never-persist-this"}`))
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-For", "192.0.2.1")
	req.Header.Set("X-Dflh-Install-ID", "abcdef00-1234-1234-1234-abcdef123456")
	req.Header.Set("X-Client-Platform", "ios")
	req.Header.Set("X-Client-Version", "1.3.2")
	tel.recordLogin(req, "password", 200, .01, ID(16))
	files, _ := filepath.Glob(filepath.Join(tel.spool, "*.json"))
	if len(files) != 1 {
		t.Fatal("audit outbox missing")
	}
	b, _ := os.ReadFile(files[0])
	if bytes.Contains(b, []byte("never-persist-this")) {
		t.Fatal("credential payload leaked")
	}
	var event LoginEvent
	if json.Unmarshal(b, &event) != nil || event.Outcome != "success" || event.IP != "192.0.2.1" || event.Version != "1.3.2" {
		t.Fatal("event metadata lost")
	}
	info, _ := os.Stat(files[0])
	if info.Mode().Perm() != 0600 {
		t.Fatal("outbox permissions")
	}
	mock.ExpectExec("INSERT IGNORE INTO LOGIN_SECURITY_EVENTS").WillReturnResult(sqlmock.NewResult(1, 1))
	tel.replayAudit()
	files, _ = filepath.Glob(filepath.Join(tel.spool, "*.json"))
	if len(files) != 0 {
		t.Fatal("replayed event retained")
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}
