// sentry_client.go — Cached, read-only client for Sentry's REST API.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	cachelib "github.com/patrickmn/go-cache"
)

var ErrSentryNotConfigured = errors.New("sentry monitoring is not configured")

const (
	sentryAPIBaseURL            = "https://sentry.io/api/0/"
	sentryHTTPTimeout           = 8 * time.Second
	sentryCacheTTL              = 5 * time.Minute
	sentryMaxResponseBytes      = 2 << 20
	sentryRecentIssueFetchLimit = 100
)

type SentryAPIError struct {
	StatusCode int
}

func (e *SentryAPIError) Error() string {
	return fmt.Sprintf("sentry API returned status %d", e.StatusCode)
}

type SentryClient struct {
	config     config.SentryConfig
	baseURL    string
	httpClient *http.Client
	cache      *cachelib.Cache
}

func NewSentryClient(cfg config.SentryConfig, cacheStore *cachelib.Cache) *SentryClient {
	if cacheStore == nil {
		cacheStore = cachelib.New(sentryCacheTTL, 2*sentryCacheTTL)
	}
	return &SentryClient{
		config:     cfg,
		baseURL:    sentryAPIBaseURL,
		httpClient: &http.Client{Timeout: sentryHTTPTimeout},
		cache:      cacheStore,
	}
}

func (c *SentryClient) RecentIssues(ctx context.Context, project string) ([]model.SentryIssueSummary, error) {
	if err := c.checkConfigured(); err != nil {
		return nil, err
	}

	cacheKey := "sentry:issues:" + project
	if cached, found := c.cache.Get(cacheKey); found {
		if issues, ok := cached.([]model.SentryIssueSummary); ok {
			return issues, nil
		}
	}

	query := url.Values{}
	query.Set("project", project)
	query.Set("statsPeriod", model.SentryMonitoringStatsPeriod)
	query.Set("query", "is:unresolved lastSeen:-"+model.SentryMonitoringStatsPeriod)
	query.Set("sort", "freq")
	query.Set("limit", strconv.Itoa(sentryRecentIssueFetchLimit))

	var payload []sentryIssuePayload
	if err := c.getJSON(ctx, c.organizationPath("issues/"), query, &payload); err != nil {
		return nil, err
	}

	issues := make([]model.SentryIssueSummary, 0, len(payload))
	for _, issue := range payload {
		count, err := strconv.ParseUint(issue.Count, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("decode sentry issue count: %w", err)
		}
		title := issue.Title
		if title == "" {
			title = issue.Metadata.Title
		}
		issues = append(issues, model.SentryIssueSummary{
			Title:           title,
			OccurrenceCount: count,
			Link:            issue.Permalink,
		})
	}
	c.cache.Set(cacheKey, issues, sentryCacheTTL)
	return issues, nil
}

func (c *SentryClient) CrashFreeSessionRate(ctx context.Context, project string) (*float64, error) {
	if err := c.checkConfigured(); err != nil {
		return nil, err
	}

	cacheKey := "sentry:crash-free-session-rate:" + project
	if cached, found := c.cache.Get(cacheKey); found {
		if rate, ok := cached.(*float64); ok {
			return rate, nil
		}
	}

	const rateField = "crash_free_rate(session)"
	query := url.Values{}
	query.Set("project", project)
	query.Set("field", rateField)
	query.Set("statsPeriod", model.SentryMonitoringStatsPeriod)
	query.Set("includeSeries", "0")

	var payload sentrySessionsPayload
	if err := c.getJSON(ctx, c.organizationPath("sessions/"), query, &payload); err != nil {
		return nil, err
	}

	for _, group := range payload.Groups {
		if rate := group.Totals[rateField]; rate != nil {
			c.cache.Set(cacheKey, rate, sentryCacheTTL)
			return rate, nil
		}
	}

	var unavailableRate *float64
	c.cache.Set(cacheKey, unavailableRate, sentryCacheTTL)
	return nil, nil
}

func (c *SentryClient) AppStartMetrics(ctx context.Context, project string) ([]model.SentryAppStartMetric, error) {
	if err := c.checkConfigured(); err != nil {
		return nil, err
	}

	cacheKey := "sentry:app-start-metrics:" + project
	if cached, found := c.cache.Get(cacheKey); found {
		if metrics, ok := cached.([]model.SentryAppStartMetric); ok {
			return metrics, nil
		}
	}

	query := url.Values{}
	query.Set("dataset", "spans")
	query.Set("project", project)
	query.Set("statsPeriod", model.SentryMonitoringStatsPeriod)
	query.Set("query", "span.op:[app.start.cold,app.start.warm]")
	for _, field := range []string{"span.op", "count()", "avg(span.duration)", "p50(span.duration)", "p95(span.duration)"} {
		query.Add("field", field)
	}

	var payload sentryPerformancePayload
	if err := c.getJSON(ctx, c.organizationPath("events/"), query, &payload); err != nil {
		return nil, err
	}

	metrics := make([]model.SentryAppStartMetric, 0, len(payload.Data))
	for _, row := range payload.Data {
		metrics = append(metrics, model.SentryAppStartMetric{
			Operation:     row.Operation,
			Count:         uint64(row.Count),
			AverageTimeMS: float64(row.AverageTimeMS),
			P50TimeMS:     float64(row.P50TimeMS),
			P95TimeMS:     float64(row.P95TimeMS),
		})
	}
	c.cache.Set(cacheKey, metrics, sentryCacheTTL)
	return metrics, nil
}

func (c *SentryClient) checkConfigured() error {
	if !c.config.Configured() {
		return ErrSentryNotConfigured
	}
	return nil
}

func (c *SentryClient) organizationPath(resource string) string {
	return "organizations/" + url.PathEscape(c.config.Organization) + "/" + resource
}

func (c *SentryClient) getJSON(ctx context.Context, resource string, query url.Values, target interface{}) error {
	baseURL := strings.TrimRight(c.baseURL, "/") + "/"
	requestURL, err := url.Parse(baseURL + resource)
	if err != nil {
		return fmt.Errorf("build sentry request URL: %w", err)
	}
	requestURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("build sentry request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.config.AuthToken)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("request sentry API: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, sentryMaxResponseBytes))
		return &SentryAPIError{StatusCode: response.StatusCode}
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, sentryMaxResponseBytes)).Decode(target); err != nil {
		return fmt.Errorf("decode sentry API response: %w", err)
	}
	return nil
}

type sentryIssuePayload struct {
	Title     string `json:"title"`
	Count     string `json:"count"`
	Permalink string `json:"permalink"`
	Metadata  struct {
		Title string `json:"title"`
	} `json:"metadata"`
}

type sentrySessionsPayload struct {
	Groups []struct {
		Totals map[string]*float64 `json:"totals"`
	} `json:"groups"`
}

type sentryPerformancePayload struct {
	Data []struct {
		Operation     string       `json:"span.op"`
		Count         sentryNumber `json:"count()"`
		AverageTimeMS sentryNumber `json:"avg(span.duration)"`
		P50TimeMS     sentryNumber `json:"p50(span.duration)"`
		P95TimeMS     sentryNumber `json:"p95(span.duration)"`
	} `json:"data"`
}

type sentryNumber float64

func (n *sentryNumber) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), "\"")
	if raw == "" || raw == "null" {
		*n = 0
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return err
	}
	*n = sentryNumber(value)
	return nil
}
