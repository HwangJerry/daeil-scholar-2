package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

type sentryMonitoringServicerStub struct {
	limit       int
	crash       *model.SentryCrashSummaryResponse
	performance *model.SentryPerformanceSummaryResponse
	err         error
}

func (s *sentryMonitoringServicerStub) CrashSummary(_ context.Context, topN int) (*model.SentryCrashSummaryResponse, error) {
	s.limit = topN
	return s.crash, s.err
}

func (s *sentryMonitoringServicerStub) PerformanceSummary(_ context.Context) (*model.SentryPerformanceSummaryResponse, error) {
	return s.performance, s.err
}

func TestSentryMonitoringHandlerReturnsCrashSummary(t *testing.T) {
	stub := &sentryMonitoringServicerStub{crash: &model.SentryCrashSummaryResponse{
		StatsPeriod: "14d",
		Platforms: []model.SentryPlatformCrashSummary{{
			Platform:  model.MobilePlatformIOS,
			TopIssues: []model.SentryIssueSummary{{Title: "Fatal crash", OccurrenceCount: 9}},
		}},
	}}
	handler := &SentryMonitoringHandler{service: stub}
	response := httptest.NewRecorder()

	handler.CrashSummary(response, httptest.NewRequest(http.MethodGet, "/api/admin/monitoring/crash-summary?limit=8", nil))

	if response.Code != http.StatusOK || stub.limit != 8 {
		t.Fatalf("status = %d, limit = %d, body = %s", response.Code, stub.limit, response.Body.String())
	}
	for _, expected := range []string{`"statsPeriod":"14d"`, `"platform":"ios"`, `"title":"Fatal crash"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("body = %s, want %s", response.Body.String(), expected)
		}
	}
}

func TestSentryMonitoringHandlerUsesDefaultLimitAndRejectsInvalidLimit(t *testing.T) {
	stub := &sentryMonitoringServicerStub{crash: &model.SentryCrashSummaryResponse{}}
	handler := &SentryMonitoringHandler{service: stub}
	response := httptest.NewRecorder()
	handler.CrashSummary(response, httptest.NewRequest(http.MethodGet, "/api/admin/monitoring/crash-summary", nil))
	if stub.limit != defaultSentryTopIssueLimit {
		t.Fatalf("default limit = %d", stub.limit)
	}

	for _, value := range []string{"0", "21", "invalid"} {
		response = httptest.NewRecorder()
		handler.CrashSummary(response, httptest.NewRequest(http.MethodGet, "/api/admin/monitoring/crash-summary?limit="+value, nil))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_LIMIT"`) {
			t.Fatalf("limit = %q, status = %d, body = %s", value, response.Code, response.Body.String())
		}
	}
}

func TestSentryMonitoringHandlerMapsConfigurationAndUpstreamErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       string
	}{
		{name: "not configured", err: service.ErrSentryNotConfigured, statusCode: http.StatusServiceUnavailable, code: "SENTRY_NOT_CONFIGURED"},
		{name: "upstream", err: errors.New("upstream failed"), statusCode: http.StatusBadGateway, code: "SENTRY_UNAVAILABLE"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := &SentryMonitoringHandler{service: &sentryMonitoringServicerStub{err: test.err}}
			response := httptest.NewRecorder()
			handler.PerformanceSummary(response, httptest.NewRequest(http.MethodGet, "/api/admin/monitoring/performance-summary", nil))
			if response.Code != test.statusCode || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}
