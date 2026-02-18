// DonationBanner — Displays cumulative donation amount, donor count, and goal progress bar
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api } from '../../api/client';
import { formatAmount } from '../../utils/formatAmount';
import type { DonationSummary } from '../../types/api';

const STALE_TIME_MS = 5 * 60_000;

export function DonationBanner() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['donation', 'summary'],
    queryFn: () => api.get<DonationSummary>('/api/donation/summary'),
    staleTime: STALE_TIME_MS,
  });

  if (isLoading) {
    return <div className="h-28 rounded-xl skeleton-shimmer" />;
  }

  if (isError) return null;

  if (!data) return null;

  const rate = Math.min(data.achievementRate, 100);
  const hasGoal = data.goalAmount > 0;

  return (
    <div className="rounded-xl bg-gradient-to-br from-primary to-indigo-600 p-5 text-white shadow-lg animate-fade-in-up">
      <h3 className="mb-3 text-[13px] font-semibold text-indigo-200 tracking-wide uppercase">누적 기부액</h3>
      <div className="mb-3 flex flex-wrap gap-2">
        <div className="rounded-lg bg-white/10 px-3 py-2 backdrop-blur-sm">
          <p className="text-xl font-bold">{formatAmount(data.displayAmount)}원</p>
        </div>
        <div className="rounded-lg bg-white/10 px-3 py-2 backdrop-blur-sm">
          <p className="text-sm font-medium">{data.donorCount}명 참여</p>
        </div>
        {hasGoal && (
          <div className="rounded-lg bg-white/10 px-3 py-2 backdrop-blur-sm">
            <p className="text-sm font-medium">달성률 {rate.toFixed(0)}%</p>
          </div>
        )}
      </div>
      {hasGoal && (
        <>
          <div className="h-2 overflow-hidden rounded-full bg-indigo-900/30">
            <div
              className="h-full rounded-full bg-gradient-to-r from-white to-indigo-200 animate-progress"
              style={{ width: `${rate}%`, boxShadow: '0 0 8px rgba(255,255,255,0.3)' }}
            />
          </div>
          <p className="mt-1.5 text-right text-xs text-indigo-200">
            목표 {formatAmount(data.goalAmount)}원
          </p>
        </>
      )}
      <Link
        to="/donation"
        className="mt-3 inline-block rounded-lg bg-white px-5 py-2 text-sm font-semibold text-primary shadow-sm transition-all duration-150 hover:shadow-md hover:bg-indigo-50"
      >
        기부하기
      </Link>
    </div>
  );
}
