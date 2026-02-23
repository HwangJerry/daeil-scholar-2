// DonationMonitorPage — renders KPI cards, achievement progress bar, and 30-day snapshot history table
import { Heart, Users, TrendingUp, Target } from 'lucide-react';
import { StatsCard } from '../components/ui/StatsCard.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { useDonationMonitor } from '../hooks/useDonationMonitor.ts';
import { formatAmount } from '../lib/formatAmount.ts';

export function DonationMonitorPage() {
  const { summary, history } = useDonationMonitor();

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold text-dark-slate">기부 현황</h2>

      {summary.isError ? (
        <ErrorState onRetry={() => void summary.refetch()} />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <StatsCard
              label="총 기부금"
              value={summary.isLoading ? '—' : `₩${formatAmount(summary.data?.displayAmount ?? 0)}`}
              icon={<Heart className="h-5 w-5" />}
            />
            <StatsCard
              label="기부자 수"
              value={summary.isLoading ? '—' : `${(summary.data?.donorCount ?? 0).toLocaleString()}명`}
              icon={<Users className="h-5 w-5" />}
            />
            <StatsCard
              label="달성률"
              value={summary.isLoading ? '—' : `${summary.data?.achievementRate.toFixed(1) ?? 0}%`}
              icon={<TrendingUp className="h-5 w-5" />}
            />
            <StatsCard
              label="목표금액"
              value={summary.isLoading ? '—' : `₩${formatAmount(summary.data?.goalAmount ?? 0)}`}
              icon={<Target className="h-5 w-5" />}
            />
          </div>

          {!summary.isLoading && summary.data && (
            <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
              <div className="mb-2 flex justify-between text-sm text-cool-gray">
                <span>달성률 진행 현황</span>
                <span>{summary.data.achievementRate.toFixed(1)}%</span>
              </div>
              <div className="h-3 w-full overflow-hidden rounded-full bg-border-light">
                <div
                  className="h-3 rounded-full bg-royal-indigo transition-all"
                  style={{ width: `${Math.min(summary.data.achievementRate, 100)}%` }}
                />
              </div>
              <p className="mt-2 text-xs text-cool-gray">
                스냅샷 기준일: {summary.data.snapshotDate?.slice(0, 10) ?? '—'}
              </p>
            </div>
          )}
        </>
      )}

      <div className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
        <h3 className="mb-4 font-semibold text-dark-slate">스냅샷 이력 (최근 30일)</h3>
        {history.isError ? (
          <ErrorState onRetry={() => void history.refetch()} />
        ) : history.isLoading ? (
          <p className="text-sm text-cool-gray">로딩 중...</p>
        ) : history.rows.length ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border-light text-left text-cool-gray">
                  <th className="px-3 py-2 font-medium">날짜</th>
                  <th className="px-3 py-2 font-medium text-right">기부금</th>
                  <th className="px-3 py-2 font-medium text-right">기부자수</th>
                  <th className="px-3 py-2 font-medium text-right">목표금액</th>
                  <th className="px-3 py-2 font-medium text-right">달성률</th>
                </tr>
              </thead>
              <tbody>
                {history.rows.map((row) => (
                  <tr key={row.date} className="border-b border-border-light">
                    <td className="px-3 py-2 text-dark-slate">{row.date}</td>
                    <td className="px-3 py-2 text-right text-dark-slate">₩{formatAmount(row.displayAmount)}</td>
                    <td className="px-3 py-2 text-right text-cool-gray">{row.donorCount.toLocaleString()}명</td>
                    <td className="px-3 py-2 text-right text-cool-gray">₩{formatAmount(row.goalAmount)}</td>
                    <td className="px-3 py-2 text-right text-cool-gray">{row.achievementRate}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-sm text-cool-gray">이력이 없습니다.</p>
        )}
      </div>
    </div>
  );
}
