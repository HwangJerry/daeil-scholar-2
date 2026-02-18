// DonationSummaryCard — Fetches and displays cumulative donation progress with goal bar
import { useQuery } from '@tanstack/react-query';
import { api } from '../../api/client';
import { formatAmount } from '../../utils/formatAmount';
import type { DonationSummary } from '../../types/api';

const STALE_TIME_MS = 5 * 60_000;

export function DonationSummaryCard() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['donation', 'summary'],
    queryFn: () => api.get<DonationSummary>('/api/donation/summary'),
    staleTime: STALE_TIME_MS,
  });

  if (isLoading) {
    return <div className="h-40 rounded-xl skeleton-shimmer" />;
  }

  if (isError) {
    return (
      <div className="rounded-xl bg-red-50 border border-red-200 p-6 text-center space-y-2">
        <p className="text-sm text-red-600">기부 현황을 불러올 수 없습니다.</p>
        <p className="text-xs text-text-tertiary">잠시 후 다시 시도해 주세요.</p>
      </div>
    );
  }

  const displayAmount = data?.displayAmount ?? 0;
  const goalAmount = data?.goalAmount ?? 0;
  const donorCount = data?.donorCount ?? 0;
  const rate = data ? Math.min(data.achievementRate, 100) : 0;

  return (
    <div className="rounded-xl bg-gradient-to-br from-primary to-indigo-600 p-6 text-white shadow-xl">
      <p className="text-indigo-200 text-[13px] font-medium mb-1 tracking-wide uppercase">누적 기부액</p>
      <h2 className="text-3xl font-extrabold tracking-tight mb-4">
        {formatAmount(displayAmount)}원
      </h2>

      <div className="w-full bg-indigo-900/30 rounded-full h-2 mb-2">
        <div
          className="h-2 rounded-full bg-gradient-to-r from-white to-indigo-200 animate-progress"
          style={{ width: `${rate}%`, boxShadow: '0 0 8px rgba(255,255,255,0.3)' }}
        />
      </div>
      <div className="flex justify-between text-xs text-indigo-200">
        <span>{donorCount}명 참여</span>
        <span>목표: {goalAmount > 0 ? `${formatAmount(goalAmount)}원` : '-'}</span>
      </div>
    </div>
  );
}
