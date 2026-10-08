package observability

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const loginRetentionDays = 90

type loginOutcomeKey struct{}
type loginOutcome struct{ value string }

func MarkLoginOutcome(ctx context.Context, value string) {
	if value != "success" && value != "link_required" && value != "rejected" {
		return
	}
	if result, ok := ctx.Value(loginOutcomeKey{}).(*loginOutcome); ok {
		result.value = value
	}
}

var installPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
var versionPattern = regexp.MustCompile(`^[0-9A-Za-z.+_-]{1,32}$`)

func loginProvider(route string) string {
	switch route {
	case "/api/auth/login", "/api/auth/mobile/login":
		return "password"
	case "/api/auth/kakao/mobile", "/api/auth/kakao/callback":
		return "kakao"
	case "/api/auth/apple/mobile":
		return "apple"
	default:
		return ""
	}
}

// Apache appends the connecting client to XFF. Read the rightmost address only
// when the immediate peer is loopback; direct clients cannot spoof XFF.
func ClientIP(r *http.Request) (string, string) {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil {
		return "", "unknown"
	}
	if peer.IsLoopback() {
		parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		if ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1])); ip != nil {
			return ip.String(), "trusted_proxy"
		}
	}
	return peer.String(), "socket"
}

type LoginEvent struct {
	ID        int64     `json:"id" db:"id"`
	EventID   string    `json:"event_id" db:"event_id"`
	At        time.Time `json:"occurred_at" db:"occurred_at"`
	Provider  string    `json:"provider" db:"provider"`
	Outcome   string    `json:"outcome" db:"outcome"`
	Status    int       `json:"response_status" db:"response_status"`
	Duration  float64   `json:"duration_ms" db:"duration_ms"`
	IP        string    `json:"client_ip" db:"client_ip"`
	IPSource  string    `json:"ip_source" db:"ip_source"`
	InstallID string    `json:"install_id" db:"install_id"`
	Platform  string    `json:"platform" db:"platform"`
	Version   string    `json:"app_version" db:"app_version"`
	Build     string    `json:"app_build" db:"app_build"`
	Trace     string    `json:"trace_id" db:"trace_id"`
	Schema    int       `json:"schema_version" db:"schema_version"`
}

func outcome(status int) string {
	switch {
	case status == 200:
		return "success"
	case status == 202:
		return "link_required"
	case status == 429:
		return "rate_limited"
	case status == 401:
		return "invalid_credentials"
	case status == 403:
		return "rejected"
	case status >= 500:
		return "server_error"
	case status >= 400:
		return "invalid_request"
	default:
		return "redirect"
	}
}
func (t *Telemetry) recordLogin(r *http.Request, provider string, status int, elapsed float64, trace string) {
	ip, source := ClientIP(r)
	platform := r.Header.Get("X-App-Platform")
	if platform == "" {
		platform = r.Header.Get("X-Client-Platform")
	}
	if platform != "ios" && platform != "android" {
		platform = "web"
	}
	version := r.Header.Get("X-App-Version")
	if version == "" {
		version = r.Header.Get("X-Client-Version")
	}
	if !versionPattern.MatchString(version) {
		version = ""
	}
	build := r.Header.Get("X-App-Build")
	if len(build) > 20 {
		build = ""
	}
	if _, e := strconv.ParseUint(build, 10, 64); e != nil {
		build = ""
	}
	device := r.Header.Get("X-Dflh-Install-ID")
	if !installPattern.MatchString(device) {
		device = ""
	}
	ev := LoginEvent{EventID: ID(16), At: time.Now().UTC(), Provider: provider, Outcome: outcome(status), Status: status, Duration: elapsed * 1000, IP: ip, IPSource: source, InstallID: strings.ToLower(device), Platform: platform, Version: version, Build: build, Trace: trace, Schema: 1}
	if decision, ok := r.Context().Value(loginOutcomeKey{}).(*loginOutcome); ok && decision.value != "" {
		ev.Outcome = decision.value
	}
	t.mu.Lock()
	t.logins[provider+"|"+ev.Outcome]++
	t.mu.Unlock()
	if t.insert(ev) {
		return
	}
	t.mu.Lock()
	t.auditFailures++
	t.mu.Unlock()
	// Durable fallback is independent of application releases. No personal data
	// goes into journal, Loki or the failure message.
	files, _ := filepath.Glob(filepath.Join(t.spool, "*.json"))
	if len(files) >= 10000 {
		t.logger.Error().Msg("security login outbox full")
		return
	}
	b, _ := json.Marshal(ev)
	path := filepath.Join(t.spool, ev.EventID+".json")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e == nil {
		_, e = f.Write(b)
		if e == nil {
			e = f.Sync()
		}
		f.Close()
	}
	if e != nil {
		t.logger.Error().Msg("security login outbox write failed")
	}
}
func (t *Telemetry) insert(ev LoginEvent) bool {
	if t.db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, e := t.db.ExecContext(ctx, `INSERT IGNORE INTO LOGIN_SECURITY_EVENTS(event_id,occurred_at,provider,outcome,response_status,duration_ms,client_ip,ip_source,install_id,platform,app_version,app_build,trace_id,schema_version) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, ev.EventID, ev.At, ev.Provider, ev.Outcome, ev.Status, ev.Duration, ev.IP, ev.IPSource, ev.InstallID, ev.Platform, ev.Version, ev.Build, ev.Trace, ev.Schema)
	return e == nil
}
func (t *Telemetry) replayAudit() {
	files, _ := filepath.Glob(filepath.Join(t.spool, "*.json"))
	for i, p := range files {
		if i >= 100 {
			return
		}
		b, e := os.ReadFile(p)
		if e != nil {
			continue
		}
		var ev LoginEvent
		if json.Unmarshal(b, &ev) != nil {
			continue
		}
		if time.Since(ev.At) > loginRetentionDays*24*time.Hour {
			os.Remove(p)
			continue
		}
		if !t.insert(ev) {
			return
		}
		os.Remove(p)
	}
}
func (t *Telemetry) purgeAudit() {
	if t.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for batch := 0; batch < 20; batch++ {
		result, e := t.db.ExecContext(ctx, "DELETE FROM LOGIN_SECURITY_EVENTS WHERE occurred_at < ? LIMIT 10000", time.Now().UTC().AddDate(0, 0, -loginRetentionDays))
		if e != nil {
			t.logger.Warn().Msg("security login retention failed")
			return
		}
		count, _ := result.RowsAffected()
		if count < 10000 {
			return
		}
	}
}
func (t *Telemetry) pseudonym(kind, value string) string {
	if value == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(t.exportKey))
	h.Write([]byte(kind + ":" + value))
	return hex.EncodeToString(h.Sum(nil))
}

// Mount ONLY under AuthMiddleware + RootOnlyMiddleware. Keyset pagination
// supports reproducible daily training datasets without unbounded responses.
func (t *Telemetry) ExportLogins(w http.ResponseWriter, r *http.Request) {
	if t.db == nil {
		http.Error(w, "Unavailable", 503)
		return
	}
	from, e := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	if e != nil {
		http.Error(w, "from requires RFC3339", 400)
		return
	}
	to, e := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if e != nil || !to.After(from) || to.Sub(from) > 31*24*time.Hour || from.Before(time.Now().AddDate(0, 0, -loginRetentionDays)) {
		http.Error(w, "invalid export range (maximum 31 days, retention 90 days)", 400)
		return
	}
	cursor := int64(0)
	if raw := r.URL.Query().Get("after_id"); raw != "" {
		cursor, e = strconv.ParseInt(raw, 10, 64)
		if e != nil || cursor < 0 {
			http.Error(w, "invalid cursor", 400)
			return
		}
	}
	limit := 1000
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, e = strconv.Atoi(raw)
		if e != nil || limit < 1 || limit > 10000 {
			http.Error(w, "invalid limit", 400)
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	events := []LoginEvent{}
	e = t.db.SelectContext(ctx, &events, "SELECT * FROM LOGIN_SECURITY_EVENTS WHERE occurred_at >= ? AND occurred_at < ? AND id > ? ORDER BY id LIMIT ?", from.UTC(), to.UTC(), cursor, limit)
	if e != nil {
		http.Error(w, "export unavailable", 503)
		return
	}
	raw := r.URL.Query().Get("raw") == "true"
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if len(events) > 0 {
		w.Header().Set("X-Next-Cursor", strconv.FormatInt(events[len(events)-1].ID, 10))
	}
	enc := json.NewEncoder(w)
	for _, ev := range events {
		if !raw {
			ev.IP = t.pseudonym("ip", ev.IP)
			ev.InstallID = t.pseudonym("install", ev.InstallID)
		}
		row := object{"schema_version": ev.Schema, "id": ev.ID, "event_id": ev.EventID, "occurred_at": ev.At.UTC().Format(time.RFC3339Nano), "provider": ev.Provider, "outcome": ev.Outcome, "success": ev.Outcome == "success", "response_status": ev.Status, "duration_ms": ev.Duration, "ip": ev.IP, "ip_source": ev.IPSource, "install_id": ev.InstallID, "platform": ev.Platform, "app_version": ev.Version, "app_build": ev.Build, "trace_id": ev.Trace, "pseudonymized": !raw, "identity_claims": "client_reported"}
		if enc.Encode(row) != nil {
			break
		}
	}
	t.logger.Info().Int("rows", len(events)).Bool("raw", raw).Msg("security login dataset exported by root")
}
