package observability

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var hex32 = regexp.MustCompile(`^[a-f0-9]{32}$`)
var hex16 = regexp.MustCompile(`^[a-f0-9]{16}$`)
var allowedScreens = map[string]bool{"launch": true, "login": true, "verification": true, "feed": true, "alumni": true, "donation": true, "messages": true, "profile": true, "signup": true}
var allowedRoutes = map[string]bool{"/api/auth": true, "/api/feed": true, "/api/alumni": true, "/api/profile": true, "/api/messages": true, "/api/settings": true, "/api/donation": true, "/api/notifications": true, "/api/push": true, "/api/account-deletion": true, "/api/other": true}
var mobileMu sync.Mutex
var mobileLimits = map[string]struct {
	minute int64
	count  int
}{}

type MobileEvent struct {
	Trace    string `json:"trace_id"`
	Span     string `json:"span_id"`
	Parent   string `json:"parent_span_id"`
	Screen   string `json:"screen"`
	Action   string `json:"action"`
	Route    string `json:"route"`
	Method   string `json:"method"`
	Status   int    `json:"status"`
	Start    int64  `json:"start_ms"`
	Duration int64  `json:"duration_ms"`
}

func validateMobile(e MobileEvent, now int64) bool {
	return hex32.MatchString(e.Trace) && hex16.MatchString(e.Span) && (e.Parent == "" || hex16.MatchString(e.Parent)) && e.Trace != "00000000000000000000000000000000" && e.Span != "0000000000000000" && allowedScreens[e.Screen] && (e.Action == "screen.view" || e.Action == "http.request") && (e.Action != "http.request" || (allowedRoutes[e.Route] && (e.Method == "GET" || e.Method == "POST" || e.Method == "PUT" || e.Method == "DELETE" || e.Method == "PATCH"))) && e.Status >= 0 && e.Status <= 599 && e.Duration >= 0 && e.Duration <= 120000 && e.Start >= now-300000 && e.Start <= now+60000 && e.Start+e.Duration <= now+60000
}
func (t *Telemetry) CollectMobile(w http.ResponseWriter, r *http.Request) {
	ip, _ := ClientIP(r)
	minute := time.Now().Unix() / 60
	mobileMu.Lock()
	entry := mobileLimits[ip]
	if entry.minute != minute {
		entry.minute = minute
		entry.count = 0
	}
	entry.count++
	mobileLimits[ip] = entry
	if len(mobileLimits) > 10000 {
		for k, v := range mobileLimits {
			if v.minute < minute {
				delete(mobileLimits, k)
			}
		}
	}
	limited := entry.count > 20 || len(mobileLimits) > 10000
	mobileMu.Unlock()
	if limited {
		http.Error(w, "Rate limited", 429)
		return
	}
	platform := r.Header.Get("X-Client-Platform")
	if platform != "ios" && platform != "android" {
		http.Error(w, "Invalid platform", 400)
		return
	}
	var body struct {
		Events []MobileEvent `json:"events"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if dec.Decode(&body) != nil || len(body.Events) == 0 || len(body.Events) > 20 {
		http.Error(w, "Invalid events", 400)
		return
	}
	var extra interface{}
	if dec.Decode(&extra) != io.EOF {
		http.Error(w, "Invalid body", 400)
		return
	}
	now := time.Now().UnixMilli()
	for _, e := range body.Events {
		if !validateMobile(e, now) {
			http.Error(w, "Invalid event", 400)
			return
		}
	}
	version := r.Header.Get("X-Client-Version")
	if !versionPattern.MatchString(version) {
		version = "unknown"
	}
	for _, e := range body.Events {
		code := 1
		if e.Action == "http.request" && (e.Status == 0 || e.Status >= 400) {
			code = 2
		}
		kind := 1
		name := "screen." + e.Screen
		if e.Action == "http.request" {
			kind = 3
			name = e.Method + " " + e.Route
		}
		a := attributes(map[string]string{"app.screen": e.Screen, "app.version": version, "breadcrumb.action": e.Action, "http.route": e.Route, "http.response.status_code": strconv.Itoa(e.Status)})
		span := object{"traceId": e.Trace, "spanId": e.Span, "name": name, "kind": kind, "startTimeUnixNano": strconv.FormatInt(e.Start*1000000, 10), "endTimeUnixNano": strconv.FormatInt((e.Start+e.Duration)*1000000, 10), "attributes": a, "status": object{"code": code}, "events": []object{{"name": e.Action, "timeUnixNano": strconv.FormatInt(e.Start*1000000, 10), "attributes": a}}}
		if e.Parent != "" {
			span["parentSpanId"] = e.Parent
		}
		t.enqueue(object{"service": "daeil-" + platform, "span": span}, nil)
	}
	w.WriteHeader(http.StatusNoContent)
}
