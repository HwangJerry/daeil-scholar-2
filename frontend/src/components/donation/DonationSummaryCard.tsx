// DonationSummaryCard — Cumulative donation progress with dark navy gradient editorial style
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
    return <div className="h-40 rounded-[20px] skeleton-shimmer" />;
  }

  if (isError) {
    return (
      <div className="rounded-[20px] bg-error-subtle border border-error-border p-6 text-center space-y-2">
        <p className="text-sm text-error-text">기부 현황을 불러올 수 없습니다.</p>
        <p className="text-xs text-text-tertiary">잠시 후 다시 시도해 주세요.</p>
      </div>
    );
  }

  const displayAmount = data?.displayAmount ?? 0;
  const goalAmount = data?.goalAmount ?? 0;
  const donorCount = data?.donorCount ?? 0;
  const rate = data ? Math.min(data.achievementRate, 100) : 0;

  return (
    <div className="relative overflow-hidden rounded-[20px] bg-gradient-to-br from-hero-from via-hero-via to-hero-to p-6 text-white shadow-xl">
      {/* Ambient glow */}
      <div className="absolute top-0 right-0 w-48 h-48 rounded-full bg-white/5 -translate-y-1/2 translate-x-1/2 blur-3xl pointer-events-none" />

      <p className="relative text-white/50 text-[13px] font-medium mb-1 tracking-wide uppercase">
        누적 기부액
      </p>
      <h2 className="relative text-3xl font-extrabold tracking-tight mb-4 font-serif text-white">
        {formatAmount(displayAmount)}원
      </h2>

      <div className="relative w-full bg-white/10 rounded-full h-2 mb-2 overflow-hidden">
        <div
          className="h-2 rounded-full bg-gradient-to-r from-white to-white/60 animate-fill-bar"
          style={{ width: `${rate}%` }}
        />
      </div>
      <div className="relative flex justify-between text-xs text-white/50">
        <span>{donorCount}명 참여</span>
        <span>목표: {goalAmount > 0 ? `${formatAmount(goalAmount)}원` : '-'}</span>
      </div>
    </div>
  );
}
