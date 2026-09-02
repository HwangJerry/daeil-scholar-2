package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/patrickmn/go-cache"
)

func TestSentryClientFetchesAndCachesMonitoringData(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("project") != "ios-project" {
			t.Errorf("project = %q", r.URL.Query().Get("project"))
		}
		if r.URL.Query().Get("statsPeriod") != "14d" {
			t.Errorf("statsPeriod = %q", r.URL.Query().Get("statsPeriod"))
		}

		switch r.URL.Path {
		case "/api/0/organizations/test-org/issues/":
			if r.URL.Query().Get("sort") != "freq" || r.URL.Query().Get("limit") != strconv.Itoa(sentryRecentIssueFetchLimit) {
				t.Errorf("issue query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[
				{"title":"Fatal crash","count":"42","permalink":"https://example.sentry.io/issues/1/"},
				{"metadata":{"title":"Fallback title"},"count":"7","permalink":"https://example.sentry.io/issues/2/"}
			]`))
		case "/api/0/organizations/test-org/sessions/":
			if r.URL.Query().Get("field") != "crash_free_rate(session)" || r.URL.Query().Get("includeSeries") != "0" {
				t.Errorf("session query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"groups":[{"by":{},"totals":{"crash_free_rate(session)":0.9987}}]}`))
		case "/api/0/organizations/test-org/events/":
			if r.URL.Query().Get("dataset") != "spans" || r.URL.Query().Get("query") != "span.op:[app.start.cold,app.start.warm]" {
				t.Errorf("performance query = %s", r.URL.RawQuery)
			}
			if len(r.URL.Query()["field"]) != 5 {
				t.Errorf("performance fields = %#v", r.URL.Query()["field"])
			}
			_, _ = w.Write([]byte(`{"data":[{"span.op":"app.start.cold","count()":12,"avg(span.duration)":850.5,"p50(span.duration)":"800","p95(span.duration)":1200}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestSentryClient(server)
	ctx := context.Background()
	for attempt := 0; attempt < 2; attempt++ {
		issues, err := client.RecentIssues(ctx, "ios-project")
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 2 || issues[0].OccurrenceCount != 42 || issues[1].Title != "Fallback title" {
			t.Fatalf("issues = %#v", issues)
		}

		rate, err := client.CrashFreeSessionRate(ctx, "ios-project")
		if err != nil {
			t.Fatal(err)
		}
		if rate == nil || *rate != 0.9987 {
			t.Fatalf("rate = %#v", rate)
		}

		metrics, err := client.AppStartMetrics(ctx, "ios-project")
		if err != nil {
			t.Fatal(err)
		}
		if len(metrics) != 1 || metrics[0].Count != 12 || metrics[0].P50TimeMS != 800 || metrics[0].P95TimeMS != 1200 {
			t.Fatalf("metrics = %#v", metrics)
		}
	}
	if requestCount.Load() != 3 {
		t.Fatalf("request count = %d, want 3 cached requests", requestCount.Load())
	}
}

func TestSentryClientRejectsMissingConfiguration(t *testing.T) {
	client := NewSentryClient(config.SentryConfig{}, cache.New(time.Minute, time.Minute))
	_, err := client.RecentIssues(context.Background(), "ios-project")
	if !errors.Is(err, ErrSentryNotConfigured) {
		t.Fatalf("error = %v, want ErrSentryNotConfigured", err)
	}
}

func TestSentryClientReturnsTypedUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestSentryClient(server)
	_, err := client.RecentIssues(context.Background(), "ios-project")
	var apiError *SentryAPIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusUnauthorized {
		t.Fatalf("error = %#v, want SentryAPIError 401", err)
	}
}

func newTestSentryClient(server *httptest.Server) *SentryClient {
	client := NewSentryClient(config.SentryConfig{
		AuthToken:      "test-token",
		Organization:   "test-org",
		IOSProject:     "ios-project",
		AndroidProject: "android-project",
	}, cache.New(time.Minute, time.Minute))
	client.baseURL = server.URL + "/api/0/"
	client.httpClient = server.Client()
	client.httpClient.Timeout = sentryHTTPTimeout
	return client
}
