package service

import (
	"context"
	"testing"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
)

type sentryMonitoringClientStub struct {
	issues  map[string][]model.SentryIssueSummary
	rates   map[string]*float64
	metrics map[string][]model.SentryAppStartMetric
}

func (s *sentryMonitoringClientStub) RecentIssues(_ context.Context, project string) ([]model.SentryIssueSummary, error) {
	return s.issues[project], nil
}

func (s *sentryMonitoringClientStub) CrashFreeSessionRate(_ context.Context, project string) (*float64, error) {
	return s.rates[project], nil
}

func (s *sentryMonitoringClientStub) AppStartMetrics(_ context.Context, project string) ([]model.SentryAppStartMetric, error) {
	return s.metrics[project], nil
}

func TestSentryMonitoringServiceBuildsPlatformCrashSummary(t *testing.T) {
	iosRate := 0.999
	androidRate := 0.995
	stub := &sentryMonitoringClientStub{
		issues: map[string][]model.SentryIssueSummary{
			"ios-project": {
				{Title: "first", OccurrenceCount: 12},
				{Title: "second", OccurrenceCount: 5},
			},
			"android-project": {},
		},
		rates: map[string]*float64{"ios-project": &iosRate, "android-project": &androidRate},
	}
	service := &SentryMonitoringService{client: stub, config: config.SentryConfig{
		IOSProject: "ios-project", AndroidProject: "android-project",
	}}

	summary, err := service.CrashSummary(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Platforms) != 2 || summary.Platforms[0].Platform != model.MobilePlatformIOS || summary.Platforms[1].Platform != model.MobilePlatformAndroid {
		t.Fatalf("platforms = %#v", summary.Platforms)
	}
	ios := summary.Platforms[0]
	if ios.RecentIssueCount != 2 || ios.RecentIssueOccurrenceCount != 17 || len(ios.TopIssues) != 1 || ios.TopIssues[0].Title != "first" {
		t.Fatalf("ios summary = %#v", ios)
	}
}

func TestSentryMonitoringServiceBuildsPerformanceSummary(t *testing.T) {
	stub := &sentryMonitoringClientStub{metrics: map[string][]model.SentryAppStartMetric{
		"ios-project":     {{Operation: "app.start.cold", Count: 4}},
		"android-project": {},
	}}
	service := &SentryMonitoringService{client: stub, config: config.SentryConfig{
		IOSProject: "ios-project", AndroidProject: "android-project",
	}}

	summary, err := service.PerformanceSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.StatsPeriod != "14d" || len(summary.Platforms) != 2 || len(summary.Platforms[0].AppStart) != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}
