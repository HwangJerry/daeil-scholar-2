// sentry_monitoring.go — Admin-facing mobile crash and performance summaries.
package model

const (
	SentryMonitoringStatsPeriod = "14d"
)

type SentryIssueSummary struct {
	Title           string `json:"title"`
	OccurrenceCount uint64 `json:"occurrenceCount"`
	Link            string `json:"link"`
}

type SentryPlatformCrashSummary struct {
	Platform                   string               `json:"platform"`
	Project                    string               `json:"project"`
	CrashFreeSessionRate       *float64             `json:"crashFreeSessionRate"`
	RecentIssueCount           int                  `json:"recentIssueCount"`
	RecentIssueCountIsCapped   bool                 `json:"recentIssueCountIsCapped"`
	RecentIssueOccurrenceCount uint64               `json:"recentIssueOccurrenceCount"`
	TopIssues                  []SentryIssueSummary `json:"topIssues"`
}

type SentryCrashSummaryResponse struct {
	StatsPeriod string                       `json:"statsPeriod"`
	Platforms   []SentryPlatformCrashSummary `json:"platforms"`
}

type SentryAppStartMetric struct {
	Operation     string  `json:"operation"`
	Count         uint64  `json:"count"`
	AverageTimeMS float64 `json:"averageTimeMs"`
	P50TimeMS     float64 `json:"p50TimeMs"`
	P95TimeMS     float64 `json:"p95TimeMs"`
}

type SentryPlatformPerformanceSummary struct {
	Platform string                 `json:"platform"`
	Project  string                 `json:"project"`
	AppStart []SentryAppStartMetric `json:"appStart"`
}

type SentryPerformanceSummaryResponse struct {
	StatsPeriod string                             `json:"statsPeriod"`
	Platforms   []SentryPlatformPerformanceSummary `json:"platforms"`
}
