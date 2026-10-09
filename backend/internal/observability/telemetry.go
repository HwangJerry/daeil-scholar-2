// Package observability exports bounded, deliberately selected OTLP signals.
// Raw URLs, headers, bodies, account IDs and IPs never enter these signals.
package observability

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	cm "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type object = map[string]interface{}

const queueLimit = 2000

var bounds = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
var tracePattern = regexp.MustCompile(`^00-([a-f0-9]{32})-([a-f0-9]{16})-(00|01)$`)
var Default *Telemetry

type bucket struct {
	count   uint64
	sum     float64
	buckets []uint64
}
type Telemetry struct {
	mu                                sync.Mutex
	db                                *sqlx.DB
	logger                            zerolog.Logger
	started                           time.Time
	requests                          map[string]*bucket
	logins                            map[string]uint64
	spans, logs                       []object
	dropped, auditFailures            uint64
	endpoint, token, exportKey, spool string
	client                            *http.Client
	cancel                            context.CancelFunc
	done                              chan struct{}
}

func New(db *sqlx.DB, logger zerolog.Logger) (*Telemetry, error) {
	endpoint := strings.TrimRight(os.Getenv("TELEMETRY_ENDPOINT"), "/")
	if endpoint != "" {
		u, e := url.Parse(endpoint)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Path != "" {
			return nil, fmt.Errorf("invalid telemetry HTTPS origin")
		}
		if len(os.Getenv("TELEMETRY_TOKEN")) < 32 {
			return nil, fmt.Errorf("telemetry token missing")
		}
	}
	if len(os.Getenv("LOGIN_EXPORT_KEY")) < 32 {
		return nil, fmt.Errorf("login export key missing")
	}
	spool := os.Getenv("LOGIN_AUDIT_SPOOL")
	if spool == "" {
		spool = "/app/logs/security-login-outbox"
	}
	if err := os.MkdirAll(spool, 0700); err != nil {
		return nil, fmt.Errorf("login outbox unavailable")
	}
	t := &Telemetry{db: db, logger: logger, started: time.Now(), requests: map[string]*bucket{}, logins: map[string]uint64{}, endpoint: endpoint, token: os.Getenv("TELEMETRY_TOKEN"), exportKey: os.Getenv("LOGIN_EXPORT_KEY"), spool: spool, client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	go t.run(ctx)
	return t, nil
}
func (t *Telemetry) Close() { t.cancel(); <-t.done }
func ID(bytes int) string {
	b := make([]byte, bytes)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func attributes(m map[string]string) []object {
	out := make([]object, 0, len(m))
	for k, v := range m {
		out = append(out, object{"key": k, "value": object{"stringValue": v}})
	}
	return out
}
func resource(service string) object {
	return object{"attributes": attributes(map[string]string{"service.name": service, "service_id": "daeil-prod", "deployment.environment.name": "production", "location": "external"})}
}
func nano(t time.Time) string { return strconv.FormatInt(t.UnixNano(), 10) }
func (t *Telemetry) enqueue(span, log object) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if span != nil {
		if len(t.spans) < queueLimit {
			t.spans = append(t.spans, span)
		} else {
			t.dropped++
		}
	}
	if log != nil {
		if len(t.logs) < queueLimit {
			t.logs = append(t.logs, log)
		} else {
			t.dropped++
		}
	}
}
func (t *Telemetry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		r = r.WithContext(context.WithValue(r.Context(), loginOutcomeKey{}, &loginOutcome{}))
		trace, parent, sampled := ID(16), "", false
		if m := tracePattern.FindStringSubmatch(r.Header.Get("traceparent")); m != nil && m[1] != strings.Repeat("0", 32) && m[2] != strings.Repeat("0", 16) {
			trace, parent, sampled = m[1], m[2], m[3] == "01"
		}
		spanID := ID(8)
		w.Header().Set("X-Trace-ID", trace)
		ww := cm.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		route := "unmatched"
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			route = rc.RoutePattern()
		}
		if loginProvider(r.URL.Path) != "" {
			route = r.URL.Path
		} else if route == "unmatched" && strings.HasPrefix(r.URL.Path, "/api/") {
			route = "/api/unmatched"
		}
		if !strings.HasPrefix(route, "/api/") || route == "/api/mobile/telemetry" || route == "/api/health" {
			return
		}
		method := r.Method
		if !strings.Contains(" GET POST PUT PATCH DELETE HEAD OPTIONS ", " "+method+" ") {
			method = "OTHER"
		}
		status := ww.Status()
		if status == 0 {
			status = 200
		}
		elapsed := time.Since(start).Seconds()
		t.mu.Lock()
		key := method + "|" + route + "|" + strconv.Itoa(status)
		b := t.requests[key]
		if b == nil && len(t.requests) < 1024 {
			b = &bucket{buckets: make([]uint64, len(bounds)+1)}
			t.requests[key] = b
		}
		if b != nil {
			b.count++
			b.sum += elapsed
			i := 0
			for i < len(bounds) && elapsed > bounds[i] {
				i++
			}
			b.buckets[i]++
		}
		t.mu.Unlock()
		if provider := loginProvider(route); provider != "" {
			t.recordLogin(r, provider, status, elapsed, trace)
		}
		// All errors and auth attempts; 1/16 of other requests, plus sampled mobile journeys.
		if !sampled && status < 500 && loginProvider(route) == "" && trace[0] != '0' {
			return
		}
		code := 1
		level := "info"
		if status >= 500 {
			code = 2
			level = "error"
		} else if status >= 400 {
			level = "warn"
		}
		a := attributes(map[string]string{"http.route": route, "http.request.method": method, "http.response.status_code": strconv.Itoa(status)})
		span := object{"traceId": trace, "spanId": spanID, "name": method + " " + route, "kind": 2, "startTimeUnixNano": nano(start), "endTimeUnixNano": nano(time.Now()), "attributes": a, "status": object{"code": code}}
		if parent != "" {
			span["parentSpanId"] = parent
		}
		body, _ := json.Marshal(object{"event": "http.request", "route": route, "method": method, "status": status, "duration_ms": elapsed * 1000, "level": level, "trace_id": trace})
		t.enqueue(object{"service": "daeil-prod", "span": span}, object{"timeUnixNano": nano(time.Now()), "severityText": level, "body": object{"stringValue": string(body)}, "traceId": trace, "spanId": spanID})
	})
}
func point(v float64, now, started time.Time, labels map[string]string) object {
	return object{"attributes": attributes(labels), "startTimeUnixNano": nano(started), "timeUnixNano": nano(now), "asDouble": v}
}
func (t *Telemetry) metrics() object {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	counts, hists := []object{}, []object{}
	for key, b := range t.requests {
		p := strings.Split(key, "|")
		labels := map[string]string{"method": p[0], "route": p[1], "status": p[2]}
		counts = append(counts, point(float64(b.count), now, t.started, labels))
		bc := make([]string, len(b.buckets))
		for i, n := range b.buckets {
			bc[i] = strconv.FormatUint(n, 10)
		}
		hists = append(hists, object{"attributes": attributes(labels), "startTimeUnixNano": nano(t.started), "timeUnixNano": nano(now), "count": strconv.FormatUint(b.count, 10), "sum": b.sum, "explicitBounds": bounds, "bucketCounts": bc})
	}
	metrics := []object{
		{"name": "daeil_http_requests", "sum": object{"aggregationTemporality": 2, "isMonotonic": true, "dataPoints": counts}},
		{"name": "daeil_http_duration_seconds", "histogram": object{"aggregationTemporality": 2, "dataPoints": hists}},
		{"name": "daeil_api_heartbeat", "gauge": object{"dataPoints": []object{point(1, now, t.started, nil)}}},
		{"name": "daeil_telemetry_dropped", "sum": object{"aggregationTemporality": 2, "isMonotonic": true, "dataPoints": []object{point(float64(t.dropped), now, t.started, nil)}}},
		{"name": "daeil_login_audit_write_failures", "sum": object{"aggregationTemporality": 2, "isMonotonic": true, "dataPoints": []object{point(float64(t.auditFailures), now, t.started, nil)}}},
	}
	auth := []object{}
	for k, n := range t.logins {
		p := strings.Split(k, "|")
		auth = append(auth, point(float64(n), now, t.started, map[string]string{"provider": p[0], "outcome": p[1]}))
	}
	metrics = append(metrics, object{"name": "daeil_login_attempts", "sum": object{"aggregationTemporality": 2, "isMonotonic": true, "dataPoints": auth}})
	photoBlocks := []object{}
	for reason, count := range signupPhotoCleanupBlockCounts() {
		photoBlocks = append(photoBlocks, point(float64(count), now, t.started, map[string]string{"reason": reason}))
	}
	metrics = append(metrics, object{"name": "daeil_social_signup_photo_cleanup_blocked", "sum": object{"aggregationTemporality": 2, "isMonotonic": true, "dataPoints": photoBlocks}})
	if t.db != nil {
		stats := t.db.Stats()
		for k, v := range map[string]float64{"daeil_db_connections_open": float64(stats.OpenConnections), "daeil_db_connections_in_use": float64(stats.InUse), "daeil_db_connections_idle": float64(stats.Idle)} {
			metrics = append(metrics, object{"name": k, "gauge": object{"dataPoints": []object{point(v, now, t.started, nil)}}})
		}
	}
	return object{"resourceMetrics": []object{{"resource": resource("daeil-prod"), "scopeMetrics": []object{{"scope": object{"name": "daeil.observability.v1"}, "metrics": metrics}}}}}
}
func (t *Telemetry) post(ctx context.Context, signal string, payload object) bool {
	if t.endpoint == "" {
		return false
	}
	b, e := json.Marshal(payload)
	if e != nil {
		return false
	}
	req, e := http.NewRequestWithContext(ctx, "POST", t.endpoint+"/v1/"+signal, bytes.NewReader(b))
	if e != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")
	res, e := t.client.Do(req)
	if e != nil {
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return false
	}
	var result struct {
		PartialSuccess struct {
			RejectedSpans      int64 `json:"rejectedSpans,string"`
			RejectedLogRecords int64 `json:"rejectedLogRecords,string"`
			RejectedDataPoints int64 `json:"rejectedDataPoints,string"`
		} `json:"partialSuccess"`
	}
	_ = json.NewDecoder(res.Body).Decode(&result)
	return result.PartialSuccess.RejectedSpans == 0 && result.PartialSuccess.RejectedLogRecords == 0 && result.PartialSuccess.RejectedDataPoints == 0
}
func (t *Telemetry) flush(ctx context.Context) {
	t.post(ctx, "metrics", t.metrics())
	t.mu.Lock()
	spans := append([]object(nil), t.spans...)
	logs := append([]object(nil), t.logs...)
	t.mu.Unlock()
	groups := map[string][]object{}
	for _, s := range spans {
		service := s["service"].(string)
		groups[service] = append(groups[service], s["span"].(object))
	}
	rs := []object{}
	for service, s := range groups {
		rs = append(rs, object{"resource": resource(service), "scopeSpans": []object{{"scope": object{"name": "daeil.observability.v1"}, "spans": s}}})
	}
	if len(spans) > 0 && t.post(ctx, "traces", object{"resourceSpans": rs}) {
		t.mu.Lock()
		t.spans = t.spans[len(spans):]
		t.mu.Unlock()
	}
	if len(logs) > 0 && t.post(ctx, "logs", object{"resourceLogs": []object{{"resource": resource("daeil-prod"), "scopeLogs": []object{{"scope": object{"name": "daeil.observability.v1"}, "logRecords": logs}}}}}) {
		t.mu.Lock()
		t.logs = t.logs[len(logs):]
		t.mu.Unlock()
	}
}
func (t *Telemetry) run(ctx context.Context) {
	defer close(t.done)
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	retention := time.NewTicker(time.Hour)
	defer retention.Stop()
	t.replayAudit()
	t.purgeAudit()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			t.flush(ctx)
			t.replayAudit()
		case <-retention.C:
			t.purgeAudit()
		}
	}
}
