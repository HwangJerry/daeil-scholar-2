import { ExternalLink, Gauge, Settings, ShieldCheck } from 'lucide-react';
import { Badge } from '../ui/Badge.tsx';
import { ErrorState } from '../ui/ErrorState.tsx';
import { StatsCard } from '../ui/StatsCard.tsx';
import {
  isSentryNotConfigured,
  useCrashSummary,
  usePerformanceSummary,
} from '../../hooks/useAppMonitoring.ts';
import type {
  MobilePlatform,
  SentryAppStartMetric,
  SentryPlatformPerformanceSummary,
} from '../../types/api.ts';

const PLATFORM_LABELS: Record<MobilePlatform, string> = {
  ios: 'iOS',
  android: 'Android',
};

const APP_START_LABELS: Record<string, string> = {
  'app.start.cold': 'Cold start',
  'app.start.warm': 'Warm start',
};

function formatStatsPeriod(statsPeriod: string): string {
  if (statsPeriod.endsWith('d')) return `최근 ${statsPeriod.slice(0, -1)}일`;
  return statsPeriod;
}

function formatCrashFreeRate(rate: number | null): string {
  return rate == null ? '—' : `${(rate * 100).toFixed(2)}%`;
}

function formatMilliseconds(value: number): string {
  return `${Math.round(value).toLocaleString()} ms`;
}

function SentryConfigurationRequired() {
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <Settings className="h-8 w-8 text-warning-text" />
      <div>
        <p className="text-sm font-medium text-dark-slate">Sentry 연동 설정이 필요합니다.</p>
        <p className="mt-1 text-xs text-cool-gray">
          서버에 SENTRY_AUTH_TOKEN과 프로젝트 설정을 등록하면 모니터링 데이터가 표시됩니다.
        </p>
      </div>
    </div>
  );
}

export function SentryMonitoringSection() {
  const crashSummary = useCrashSummary();
  const performanceSummary = usePerformanceSummary();

  return (
    <section className="space-y-4">
      <div>
        <h3 className="text-lg font-semibold text-dark-slate">크래시 및 성능</h3>
        <p className="mt-1 text-sm text-cool-gray">Sentry 기준 앱 안정성과 시작 성능입니다.</p>
      </div>

      <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <h4 className="font-semibold text-dark-slate">Crash-free 세션</h4>
          {crashSummary.data && (
            <Badge variant="muted">{formatStatsPeriod(crashSummary.data.statsPeriod)}</Badge>
          )}
        </div>

        {isSentryNotConfigured(crashSummary.error) ? (
          <SentryConfigurationRequired />
        ) : crashSummary.isError ? (
          <ErrorState onRetry={() => void crashSummary.refetch()} />
        ) : crashSummary.isLoading ? (
          <p className="py-8 text-center text-sm text-cool-gray">로딩 중...</p>
        ) : crashSummary.data?.platforms.length ? (
          <div className="space-y-5">
            <div className="grid gap-4 md:grid-cols-2">
              {crashSummary.data.platforms.map((platform) => {
                const issueCount = `${platform.recentIssueCount.toLocaleString()}${platform.recentIssueCountIsCapped ? '+' : ''}`;
                return (
                  <StatsCard
                    key={platform.platform}
                    label={`${PLATFORM_LABELS[platform.platform]} crash-free rate`}
                    value={formatCrashFreeRate(platform.crashFreeSessionRate)}
                    icon={<ShieldCheck className="h-5 w-5" />}
                    subtext={`최근 이슈 ${issueCount}건 · 발생 ${platform.recentIssueOccurrenceCount.toLocaleString()}회`}
                  />
                );
              })}
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              {crashSummary.data.platforms.map((platform) => (
                <div key={platform.platform} className="rounded-xl border border-border-light p-4">
                  <div className="mb-3 flex items-center justify-between">
                    <h5 className="text-sm font-semibold text-dark-slate">
                      {PLATFORM_LABELS[platform.platform]} 최근 이슈 Top 5
                    </h5>
                    <span className="text-xs text-cool-gray">{platform.project}</span>
                  </div>
                  {platform.topIssues.length ? (
                    <ul className="divide-y divide-border-light">
                      {platform.topIssues.map((issue) => (
                        <li key={`${issue.link}-${issue.title}`} className="flex items-center gap-3 py-3">
                          <a
                            href={issue.link}
                            target="_blank"
                            rel="noreferrer"
                            className="min-w-0 flex-1 truncate text-sm text-dark-slate hover:text-royal-indigo"
                            title={issue.title}
                          >
                            {issue.title}
                          </a>
                          <span className="whitespace-nowrap text-xs text-cool-gray">
                            {issue.occurrenceCount.toLocaleString()}회
                          </span>
                          <ExternalLink className="h-4 w-4 shrink-0 text-cool-gray" aria-hidden="true" />
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="py-6 text-center text-sm text-cool-gray">최근 이슈가 없습니다.</p>
                  )}
                </div>
              ))}
            </div>
          </div>
        ) : (
          <p className="py-8 text-center text-sm text-cool-gray">크래시 데이터가 없습니다.</p>
        )}
      </div>

      <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Gauge className="h-5 w-5 text-cool-gray" />
            <h4 className="font-semibold text-dark-slate">앱 시작 시간</h4>
          </div>
          {performanceSummary.data && (
            <Badge variant="muted">{formatStatsPeriod(performanceSummary.data.statsPeriod)}</Badge>
          )}
        </div>

        {isSentryNotConfigured(performanceSummary.error) ? (
          <SentryConfigurationRequired />
        ) : performanceSummary.isError ? (
          <ErrorState onRetry={() => void performanceSummary.refetch()} />
        ) : performanceSummary.isLoading ? (
          <p className="py-8 text-center text-sm text-cool-gray">로딩 중...</p>
        ) : performanceSummary.data?.platforms.length ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {performanceSummary.data.platforms.map((platform) => (
              <PerformancePlatformCard key={platform.platform} platform={platform} />
            ))}
          </div>
        ) : (
          <p className="py-8 text-center text-sm text-cool-gray">성능 데이터가 없습니다.</p>
        )}
      </div>
    </section>
  );
}

function PerformancePlatformCard({ platform }: { platform: SentryPlatformPerformanceSummary }) {
  return (
    <div className="overflow-hidden rounded-xl border border-border-light">
      <div className="flex items-center justify-between border-b border-border-light px-4 py-3">
        <h5 className="text-sm font-semibold text-dark-slate">
          {PLATFORM_LABELS[platform.platform]}
        </h5>
        <span className="text-xs text-cool-gray">{platform.project}</span>
      </div>
      {platform.appStart.length ? (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border-light text-left text-cool-gray">
                <th className="px-3 py-2 font-medium">구분</th>
                <th className="px-3 py-2 text-right font-medium">건수</th>
                <th className="px-3 py-2 text-right font-medium">평균</th>
                <th className="px-3 py-2 text-right font-medium">P50</th>
                <th className="px-3 py-2 text-right font-medium">P95</th>
              </tr>
            </thead>
            <tbody>
              {platform.appStart.map((metric) => (
                <PerformanceMetricRow key={metric.operation} metric={metric} />
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="py-8 text-center text-sm text-cool-gray">앱 시작 데이터가 없습니다.</p>
      )}
    </div>
  );
}

function PerformanceMetricRow({ metric }: { metric: SentryAppStartMetric }) {
  return (
    <tr className="border-b border-border-light last:border-b-0">
      <td className="whitespace-nowrap px-3 py-3 text-dark-slate">
        {APP_START_LABELS[metric.operation] ?? metric.operation}
      </td>
      <td className="whitespace-nowrap px-3 py-3 text-right text-cool-gray">
        {metric.count.toLocaleString()}
      </td>
      <td className="whitespace-nowrap px-3 py-3 text-right text-cool-gray">
        {formatMilliseconds(metric.averageTimeMs)}
      </td>
      <td className="whitespace-nowrap px-3 py-3 text-right text-cool-gray">
        {formatMilliseconds(metric.p50TimeMs)}
      </td>
      <td className="whitespace-nowrap px-3 py-3 text-right text-cool-gray">
        {formatMilliseconds(metric.p95TimeMs)}
      </td>
    </tr>
  );
}
