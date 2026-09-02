// sentry_monitoring_service.go — Platform orchestration for admin monitoring.
package service

import (
	"context"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
)

type SentryMonitoringClient interface {
	RecentIssues(ctx context.Context, project string) ([]model.SentryIssueSummary, error)
	CrashFreeSessionRate(ctx context.Context, project string) (*float64, error)
	AppStartMetrics(ctx context.Context, project string) ([]model.SentryAppStartMetric, error)
}

type SentryMonitoringService struct {
	client SentryMonitoringClient
	config config.SentryConfig
}

func NewSentryMonitoringService(client *SentryClient, cfg config.SentryConfig) *SentryMonitoringService {
	return &SentryMonitoringService{client: client, config: cfg}
}

func (s *SentryMonitoringService) CrashSummary(ctx context.Context, topN int) (*model.SentryCrashSummaryResponse, error) {
	platforms := []struct {
		name    string
		project string
	}{
		{name: model.MobilePlatformIOS, project: s.config.IOSProject},
		{name: model.MobilePlatformAndroid, project: s.config.AndroidProject},
	}

	response := &model.SentryCrashSummaryResponse{
		StatsPeriod: model.SentryMonitoringStatsPeriod,
		Platforms:   make([]model.SentryPlatformCrashSummary, 0, len(platforms)),
	}
	for _, platform := range platforms {
		issues, err := s.client.RecentIssues(ctx, platform.project)
		if err != nil {
			return nil, err
		}
		crashFreeRate, err := s.client.CrashFreeSessionRate(ctx, platform.project)
		if err != nil {
			return nil, err
		}

		occurrenceCount := uint64(0)
		for _, issue := range issues {
			occurrenceCount += issue.OccurrenceCount
		}
		topIssues := issues
		if len(topIssues) > topN {
			topIssues = topIssues[:topN]
		}
		response.Platforms = append(response.Platforms, model.SentryPlatformCrashSummary{
			Platform:                   platform.name,
			Project:                    platform.project,
			CrashFreeSessionRate:       crashFreeRate,
			RecentIssueCount:           len(issues),
			RecentIssueCountIsCapped:   len(issues) == sentryRecentIssueFetchLimit,
			RecentIssueOccurrenceCount: occurrenceCount,
			TopIssues:                  topIssues,
		})
	}
	return response, nil
}

func (s *SentryMonitoringService) PerformanceSummary(ctx context.Context) (*model.SentryPerformanceSummaryResponse, error) {
	platforms := []struct {
		name    string
		project string
	}{
		{name: model.MobilePlatformIOS, project: s.config.IOSProject},
		{name: model.MobilePlatformAndroid, project: s.config.AndroidProject},
	}

	response := &model.SentryPerformanceSummaryResponse{
		StatsPeriod: model.SentryMonitoringStatsPeriod,
		Platforms:   make([]model.SentryPlatformPerformanceSummary, 0, len(platforms)),
	}
	for _, platform := range platforms {
		metrics, err := s.client.AppStartMetrics(ctx, platform.project)
		if err != nil {
			return nil, err
		}
		response.Platforms = append(response.Platforms, model.SentryPlatformPerformanceSummary{
			Platform: platform.name,
			Project:  platform.project,
			AppStart: metrics,
		})
	}
	return response, nil
}
