import { useMemo, useState } from 'react';
import { TrendingUp, UserCheck, UserPlus } from 'lucide-react';
import { ErrorState } from '../ui/ErrorState.tsx';
import { Select } from '../ui/Select.tsx';
import { StatsCard } from '../ui/StatsCard.tsx';
import { useMobileEventSummary } from '../../hooks/useAppMonitoring.ts';
import type {
  MobileEventSummaryItem,
  MobileEventType,
  MobilePlatform,
} from '../../types/api.ts';

const DAY_IN_MS = 24 * 60 * 60 * 1000;
const RECENT_PERIOD_DAY_COUNT = 30;

const PLATFORM_LABELS: Record<MobilePlatform, string> = {
  ios: 'iOS',
  android: 'Android',
};

const EVENT_TYPE_LABELS: Record<MobileEventType, string> = {
  signup_start: '가입 시작',
  signup_complete: '가입 완료',
  apply_complete: '지원 완료',
};

const KOREAN_DATE_FORMATTER = new Intl.DateTimeFormat('en-CA', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  timeZone: 'Asia/Seoul',
});

function getRecentDateRange(): { from: string; to: string } {
  const now = Date.now();
  return {
    from: KOREAN_DATE_FORMATTER.format(
      new Date(now - (RECENT_PERIOD_DAY_COUNT - 1) * DAY_IN_MS),
    ),
    to: KOREAN_DATE_FORMATTER.format(new Date(now)),
  };
}

function getEventCount(items: MobileEventSummaryItem[], eventType: MobileEventType): number {
  return items
    .filter((item) => item.eventType === eventType)
    .reduce((total, item) => total + item.count, 0);
}

export function BusinessEventsSection() {
  const dateRange = useMemo(() => getRecentDateRange(), []);
  const [platform, setPlatform] = useState<MobilePlatform | ''>('');
  const [eventType, setEventType] = useState<MobileEventType | ''>('');

  const funnelSummary = useMobileEventSummary({
    ...dateRange,
    platform: platform || undefined,
  });
  const filteredSummary = useMobileEventSummary({
    ...dateRange,
    platform: platform || undefined,
    eventType: eventType || undefined,
  });

  const filterLabel = platform ? PLATFORM_LABELS[platform] : '전체 플랫폼';

  return (
    <section className="space-y-4">
      <div>
        <h3 className="text-lg font-semibold text-dark-slate">비즈니스 이벤트</h3>
        <p className="mt-1 text-sm text-cool-gray">
          최근 30일({dateRange.from} ~ {dateRange.to}) 앱 이벤트 집계입니다.
        </p>
      </div>

      <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <h4 className="font-semibold text-dark-slate">가입 전환 퍼널</h4>
          <span className="text-xs text-cool-gray">{filterLabel}</span>
        </div>
        {funnelSummary.isError ? (
          <ErrorState onRetry={() => void funnelSummary.refetch()} />
        ) : funnelSummary.isLoading ? (
          <p className="py-8 text-center text-sm text-cool-gray">로딩 중...</p>
        ) : funnelSummary.data?.items.length ? (
          <SignupFunnel items={funnelSummary.data.items} />
        ) : (
          <p className="py-8 text-center text-sm text-cool-gray">
            선택한 기간에 가입 이벤트가 없습니다.
          </p>
        )}
      </div>

      <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
        <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h4 className="font-semibold text-dark-slate">이벤트 집계</h4>
            <p className="mt-1 text-xs text-cool-gray">플랫폼과 이벤트 종류별 발생 건수입니다.</p>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            <label className="text-xs text-cool-gray">
              플랫폼
              <Select
                className="mt-1 sm:w-40"
                value={platform}
                onChange={(event) => setPlatform(event.target.value as MobilePlatform | '')}
              >
                <option value="">전체 플랫폼</option>
                <option value="ios">iOS</option>
                <option value="android">Android</option>
              </Select>
            </label>
            <label className="text-xs text-cool-gray">
              이벤트 종류
              <Select
                className="mt-1 sm:w-40"
                value={eventType}
                onChange={(event) => setEventType(event.target.value as MobileEventType | '')}
              >
                <option value="">전체 이벤트</option>
                <option value="signup_start">가입 시작</option>
                <option value="signup_complete">가입 완료</option>
                <option value="apply_complete">지원 완료</option>
              </Select>
            </label>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border-light text-left text-cool-gray">
                <th className="px-3 py-2 font-medium">플랫폼</th>
                <th className="px-3 py-2 font-medium">이벤트</th>
                <th className="px-3 py-2 text-right font-medium">발생 건수</th>
              </tr>
            </thead>
            <tbody aria-live="polite">
              {filteredSummary.isError ? (
                <ErrorState colSpan={3} onRetry={() => void filteredSummary.refetch()} />
              ) : filteredSummary.isLoading ? (
                <tr>
                  <td colSpan={3} className="px-3 py-8 text-center text-cool-gray">로딩 중...</td>
                </tr>
              ) : filteredSummary.data?.items.length ? (
                filteredSummary.data.items.map((item) => (
                  <tr
                    key={`${item.platform}-${item.eventType}`}
                    className="border-b border-border-light last:border-b-0"
                  >
                    <td className="px-3 py-3 text-dark-slate">
                      {PLATFORM_LABELS[item.platform]}
                    </td>
                    <td className="px-3 py-3 text-cool-gray">
                      {EVENT_TYPE_LABELS[item.eventType]}
                    </td>
                    <td className="px-3 py-3 text-right text-dark-slate">
                      {item.count.toLocaleString()}
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={3} className="px-3 py-8 text-center text-cool-gray">
                    조건에 맞는 이벤트가 없습니다.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

function SignupFunnel({ items }: { items: MobileEventSummaryItem[] }) {
  const signupStartCount = getEventCount(items, 'signup_start');
  const signupCompleteCount = getEventCount(items, 'signup_complete');
  const conversionRate = signupStartCount > 0
    ? (signupCompleteCount / signupStartCount) * 100
    : null;

  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-3">
        <StatsCard
          label="1. 가입 시작"
          value={`${signupStartCount.toLocaleString()}건`}
          icon={<UserPlus className="h-5 w-5" />}
        />
        <StatsCard
          label="2. 가입 완료"
          value={`${signupCompleteCount.toLocaleString()}건`}
          icon={<UserCheck className="h-5 w-5" />}
        />
        <StatsCard
          label="가입 전환율"
          value={conversionRate == null ? '—' : `${conversionRate.toFixed(1)}%`}
          icon={<TrendingUp className="h-5 w-5" />}
          subtext={conversionRate == null ? '가입 시작 이벤트가 없습니다.' : '가입 완료 ÷ 가입 시작'}
        />
      </div>
      <div>
        <div className="mb-2 flex justify-between text-xs text-cool-gray">
          <span>가입 완료 전환</span>
          <span>{conversionRate == null ? '—' : `${conversionRate.toFixed(1)}%`}</span>
        </div>
        <div className="h-3 w-full overflow-hidden rounded-full bg-border-light">
          <div
            className="h-3 rounded-full bg-royal-indigo transition-all"
            style={{ width: `${Math.min(conversionRate ?? 0, 100)}%` }}
          />
        </div>
      </div>
    </div>
  );
}
